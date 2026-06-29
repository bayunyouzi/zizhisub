package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func openAIVideoStatusSessionHash(requestID string) string {
	return service.HashUsageRequestPayload([]byte("openai-video-status|" + requestID))
}

// Videos handles OpenAI-compatible video generation and status APIs.
// POST /v1/videos/generations
// GET /v1/videos/:request_id
func (h *OpenAIGatewayHandler) Videos(c *gin.Context) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)

	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}

	reqLog := requestLogger(
		c,
		"handler.openai_gateway.videos",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	var body []byte
	var err error
	if c.Request != nil && c.Request.Method != http.MethodGet {
		body, err = pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
		if err != nil {
			if maxErr, ok := extractMaxBytesError(err); ok {
				h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
				return
			}
			h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
			return
		}
	}

	parsed, err := h.gatewayService.ParseOpenAIVideosRequest(c, body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	isStatusRequest := parsed.Endpoint == EndpointVideosStatus
	requestModel := parsed.Model
	setOpsRequestContext(c, requestModel, false)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(false, false)))

	if isStatusRequest {
		reqLog = reqLog.With(zap.String("request_id", parsed.RequestID))
	} else {
		reqLog = reqLog.With(
			zap.String("model", requestModel),
			zap.String("image_url", parsed.ImageURL),
			zap.Int("duration", parsed.Duration),
			zap.String("aspect_ratio", parsed.AspectRatio),
			zap.String("resolution", parsed.Resolution),
		)
	}

	if !isStatusRequest {
		if !service.GroupAllowsVideoGeneration(apiKey.Group) {
			h.errorResponse(c, http.StatusForbidden, "permission_error", service.VideoGenerationPermissionMessage())
			return
		}
		if decision := h.checkContentModeration(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIImages, requestModel, parsed.ModerationBody()); decision != nil && decision.Blocked {
			h.errorResponse(c, contentModerationStatus(decision), contentModerationErrorCode(decision), decision.Message)
			return
		}
		imageReleaseFunc, acquired := h.acquireImageGenerationSlot(c, streamStarted)
		if !acquired {
			return
		}
		if imageReleaseFunc != nil {
			defer imageReleaseFunc()
		}
	}

	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, requestModel)
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}
	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if !isStatusRequest {
		if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
			reqLog.Info("openai.videos.billing_eligibility_check_failed", zap.Error(err))
			status, code, message, retryAfter := billingErrorDetails(err)
			if retryAfter > 0 {
				c.Header("Retry-After", strconv.Itoa(retryAfter))
			}
			h.handleStreamingAwareError(c, status, code, message, streamStarted)
			return
		}
	}

	sessionHash := ""
	if isStatusRequest {
		sessionHash = openAIVideoStatusSessionHash(parsed.RequestID)
	} else {
		sessionHash = h.gatewayService.GenerateExplicitSessionHash(c, body)
		if sessionHash == "" {
			sessionHash = service.HashUsageRequestPayload([]byte(parsed.StickySessionSeed()))
		}
	}

	maxAccountSwitches := h.maxAccountSwitches
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		reqLog.Debug("openai.videos.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
		selection, scheduleDecision, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
			c.Request.Context(),
			apiKey.GroupID,
			"",
			sessionHash,
			requestModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportHTTPSSE,
			service.OpenAIEndpointCapabilityVideos,
			false,
		)
		if err != nil {
			reqLog.Warn("openai.videos.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			if len(failedAccountIDs) == 0 {
				markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
				h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available compatible accounts", streamStarted)
				return
			}
			if lastFailoverErr != nil {
				h.handleFailoverExhausted(c, lastFailoverErr, streamStarted)
			} else {
				h.handleFailoverExhaustedSimple(c, 502, streamStarted)
			}
			return
		}
		if selection == nil || selection.Account == nil {
			markOpsRoutingCapacityLimited(c)
			h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available compatible accounts", streamStarted)
			return
		}

		reqLog.Debug("openai.videos.account_schedule_decision",
			zap.String("layer", scheduleDecision.Layer),
			zap.Bool("sticky_session_hit", scheduleDecision.StickySessionHit),
			zap.Int("candidate_count", scheduleDecision.CandidateCount),
			zap.Int("top_k", scheduleDecision.TopK),
			zap.Int64("latency_ms", scheduleDecision.LatencyMs),
			zap.Float64("load_skew", scheduleDecision.LoadSkew),
		)

		account := selection.Account
		reqLog.Debug("openai.videos.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
		setOpsSelectedAccount(c, account.ID, account.Platform)

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, false, &streamStarted, reqLog)
		if !acquired {
			return
		}

		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		writerSizeBeforeForward := c.Writer.Size()
		result, err := func() (*service.OpenAIForwardResult, error) {
			defer func() {
				if accountReleaseFunc != nil {
					accountReleaseFunc()
				}
			}()
			return h.gatewayService.ForwardVideos(c.Request.Context(), c, account, parsed, channelMapping.MappedModel)
		}()
		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		upstreamLatencyMs, _ := getContextInt64(c, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, responseLatencyMs)

		if err != nil {
			var videoUpstreamErr *service.OpenAIVideosUpstreamError
			if errors.As(err, &videoUpstreamErr) {
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
				reqLog.Warn("openai.videos.upstream_user_error",
					zap.Int64("account_id", account.ID),
					zap.Int("status_code", videoUpstreamErr.StatusCode),
					zap.String("error_type", videoUpstreamErr.ErrorType),
					zap.String("error_code", videoUpstreamErr.Code),
					zap.Error(err),
				)
				return
			}
			var failoverErr *service.UpstreamFailoverError
			if errors.As(err, &failoverErr) {
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				h.gatewayService.RecordOpenAIAccountSwitch()
				failedAccountIDs[account.ID] = struct{}{}
				lastFailoverErr = failoverErr
				if switchCount >= maxAccountSwitches {
					h.handleFailoverExhausted(c, failoverErr, streamStarted)
					return
				}
				switchCount++
				if h.gatewayService.ShouldStopOpenAIOAuth429Failover(account, failoverErr.StatusCode, switchCount) {
					h.handleFailoverExhausted(c, failoverErr, streamStarted)
					return
				}
				reqLog.Warn("openai.videos.upstream_failover_switching",
					zap.Int64("account_id", account.ID),
					zap.Int("upstream_status", failoverErr.StatusCode),
					zap.Int("switch_count", switchCount),
					zap.Int("max_switches", maxAccountSwitches),
				)
				if failoverErr.RetryableOnSameAccount {
					select {
					case <-c.Request.Context().Done():
						return
					case <-time.After(sameAccountRetryDelay):
					}
				}
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			upstreamErrorAlreadyCommunicated := openAIForwardErrorAlreadyCommunicated(c, writerSizeBeforeForward, err)
			wroteFallback := false
			if !upstreamErrorAlreadyCommunicated {
				wroteFallback = h.ensureForwardErrorResponse(c, streamStarted)
			}
			fields := []zap.Field{
				zap.Int64("account_id", account.ID),
				zap.Bool("fallback_error_response_written", wroteFallback),
				zap.Bool("upstream_error_response_already_written", upstreamErrorAlreadyCommunicated),
				zap.Error(err),
			}
			if shouldLogOpenAIForwardFailureAsWarn(c, wroteFallback) {
				reqLog.Warn("openai.videos.forward_failed", fields...)
				return
			}
			reqLog.Error("openai.videos.forward_failed", fields...)
			return
		}

		if result != nil {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
		} else {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
		}

		if !isStatusRequest && result != nil && result.RequestID != "" {
			_ = h.gatewayService.BindStickySession(c.Request.Context(), apiKey.GroupID, openAIVideoStatusSessionHash(result.RequestID), account.ID)
		}

		if !isStatusRequest && result != nil && result.RequestID != "" {
			_ = h.gatewayService.BindOpenAIVideoPendingUsage(c.Request.Context(), apiKey.GroupID, result.RequestID, requestModel, parsed.Resolution)
		}

		if isStatusRequest && result != nil && result.VideoTerminal {
			deletePending := true
			if result.VideoSucceeded {
				pendingModel, pendingResolution, ok := h.gatewayService.GetOpenAIVideoPendingUsage(c.Request.Context(), apiKey.GroupID, parsed.RequestID)
				if ok {
					billingModel := pendingModel
					if billingModel == "" {
						billingModel = requestModel
					}
					billingResolution := pendingResolution
					if billingResolution == "" {
						billingResolution = parsed.Resolution
					}
					successChannelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, billingModel)
					upstreamBillingModel := billingModel
					if mapped := strings.TrimSpace(successChannelMapping.MappedModel); mapped != "" {
						upstreamBillingModel = mapped
					}
					billingResult := &service.OpenAIForwardResult{
						RequestID:       parsed.RequestID,
						Model:           billingModel,
						UpstreamModel:   upstreamBillingModel,
						ImageCount:      1,
						ImageSize:       service.NormalizeOpenAIVideoResolutionTierForUsage(billingResolution),
						ImageInputSize:  strings.TrimSpace(billingResolution),
						ImageOutputSize: strings.TrimSpace(billingResolution),
						VideoStatus:     result.VideoStatus,
						VideoTerminal:   true,
						VideoSucceeded:  true,
					}
					userAgent := c.GetHeader("User-Agent")
					clientIP := ip.GetClientIP(c)
					inboundEndpoint := GetInboundEndpoint(c)
					upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
					channelUsage := successChannelMapping.ToUsageFields(billingModel, upstreamBillingModel)
					if err := h.gatewayService.RecordUsage(context.Background(), &service.OpenAIRecordUsageInput{
						Result:             billingResult,
						APIKey:             apiKey,
						User:               apiKey.User,
						Account:            account,
						Subscription:       subscription,
						InboundEndpoint:    inboundEndpoint,
						UpstreamEndpoint:   upstreamEndpoint,
						UserAgent:          userAgent,
						IPAddress:          clientIP,
						RequestPayloadHash: "video-status-success:" + parsed.RequestID,
						APIKeyService:      h.apiKeyService,
						ChannelUsageFields: channelUsage,
					}); err != nil {
						deletePending = false
						logger.L().With(
							zap.String("component", "handler.openai_gateway.videos"),
							zap.Int64("user_id", subject.UserID),
							zap.Int64("api_key_id", apiKey.ID),
							zap.Any("group_id", apiKey.GroupID),
							zap.String("model", billingModel),
							zap.String("request_id", parsed.RequestID),
							zap.Int64("account_id", account.ID),
						).Error("openai.videos.record_usage_on_terminal_success_failed", zap.Error(err))
					}
				}
			}
			if deletePending {
				_ = h.gatewayService.DeleteOpenAIVideoPendingUsage(c.Request.Context(), apiKey.GroupID, parsed.RequestID)
			}
		}

		reqLog.Debug("openai.videos.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", switchCount),
		)
		return
	}
}
