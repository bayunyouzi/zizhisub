package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/responseheaders"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	openAIVideosGenerationsEndpoint = "/v1/videos/generations"
	openAIVideosStatusEndpoint      = "/v1/videos/:request_id"

	openAIVideosGenerationsURL = "https://api.openai.com/v1/videos/generations"
	openAIVideosURLPrefix      = "https://api.openai.com/v1/videos"
)

type OpenAIVideosRequest struct {
	Endpoint     string
	Model        string
	Prompt       string
	ImageURL     string
	Duration     int
	AspectRatio  string
	Resolution   string
	ResponseBody []byte
	RequestID    string
	Body         []byte
	bodyHash     string
}

func (r *OpenAIVideosRequest) ModerationBody() []byte {
	if r == nil {
		return nil
	}
	payload := map[string]any{}
	if prompt := strings.TrimSpace(r.Prompt); prompt != "" {
		payload["prompt"] = prompt
	}
	if imageURL := strings.TrimSpace(r.ImageURL); imageURL != "" {
		payload["images"] = []map[string]string{{"image_url": imageURL}}
	}
	if len(payload) == 0 {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return body
}

func (r *OpenAIVideosRequest) StickySessionSeed() string {
	if r == nil {
		return ""
	}
	parts := []string{
		"openai-videos",
		strings.TrimSpace(r.Endpoint),
		strings.TrimSpace(r.Model),
		strings.TrimSpace(r.Prompt),
		strings.TrimSpace(r.ImageURL),
		fmt.Sprintf("duration=%d", r.Duration),
		strings.TrimSpace(r.AspectRatio),
		strings.TrimSpace(r.Resolution),
	}
	seed := strings.Join(parts, "|")
	if strings.TrimSpace(r.Prompt) == "" && r.bodyHash != "" {
		seed += "|body=" + r.bodyHash
	}
	return seed
}

func (s *OpenAIGatewayService) ParseOpenAIVideosRequest(c *gin.Context, body []byte) (*OpenAIVideosRequest, error) {
	if c == nil || c.Request == nil {
		return nil, fmt.Errorf("missing request context")
	}
	endpoint := normalizeOpenAIVideosEndpointPath(c.Request.URL.Path)
	if endpoint == "" {
		return nil, fmt.Errorf("unsupported videos endpoint")
	}
	if endpoint == openAIVideosStatusEndpoint {
		requestID := strings.TrimSpace(c.Param("request_id"))
		if requestID == "" {
			requestID = strings.TrimSpace(lastPathSegment(c.Request.URL.Path))
		}
		if requestID == "" {
			return nil, fmt.Errorf("request_id is required")
		}
		return &OpenAIVideosRequest{Endpoint: endpoint, RequestID: requestID}, nil
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("request body is empty")
	}
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("failed to parse request body")
	}
	model := strings.TrimSpace(gjson.GetBytes(body, "model").String())
	if model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if !isOpenAIVideoGenerationModel(model) {
		return nil, fmt.Errorf("videos endpoint requires a video model, got %q", model)
	}
	prompt := strings.TrimSpace(gjson.GetBytes(body, "prompt").String())
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	imageURL := strings.TrimSpace(gjson.GetBytes(body, "image.url").String())
	if imageURL == "" {
		imageURL = strings.TrimSpace(gjson.GetBytes(body, "image_url").String())
	}
	if imageURL == "" {
		imageURL = strings.TrimSpace(gjson.GetBytes(body, "image").String())
	}
	if imageURL == "" {
		return nil, fmt.Errorf("image.url is required")
	}
	if !isModerationImageURLAllowed(imageURL) {
		return nil, fmt.Errorf("image.url must be a valid http(s) URL or data URI")
	}
	duration := 0
	if value := gjson.GetBytes(body, "duration"); value.Exists() {
		if value.Type != gjson.Number {
			return nil, fmt.Errorf("invalid duration field type")
		}
		duration = int(value.Int())
	}
	if duration <= 0 {
		duration = 5
	}
	aspectRatio := strings.TrimSpace(gjson.GetBytes(body, "aspect_ratio").String())
	if aspectRatio == "" {
		aspectRatio = strings.TrimSpace(gjson.GetBytes(body, "metadata.aspect_ratio").String())
	}
	resolution := strings.TrimSpace(gjson.GetBytes(body, "resolution").String())
	if resolution == "" {
		resolution = strings.TrimSpace(gjson.GetBytes(body, "metadata.resolution").String())
	}
	req := &OpenAIVideosRequest{
		Endpoint:     endpoint,
		Model:        model,
		Prompt:       prompt,
		ImageURL:     imageURL,
		Duration:     duration,
		AspectRatio:  aspectRatio,
		Resolution:   resolution,
		Body:         body,
		ResponseBody: body,
	}
	if len(body) > 0 {
		sum := sha256.Sum256(body)
		req.bodyHash = hex.EncodeToString(sum[:8])
	}
	return req, nil
}

func isOpenAIVideoGenerationModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(model, "grok-imagine-video")
}

func normalizeOpenAIVideosEndpointPath(path string) string {
	trimmed := strings.TrimSpace(path)
	switch {
	case strings.Contains(trimmed, "/videos/generations"):
		return openAIVideosGenerationsEndpoint
	case strings.Contains(trimmed, "/videos/"):
		return openAIVideosStatusEndpoint
	default:
		return ""
	}
}

func lastPathSegment(path string) string {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

func isModerationImageURLAllowed(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	lower := strings.ToLower(raw)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "data:image/")
}

func (s *OpenAIGatewayService) ForwardVideos(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	parsed *OpenAIVideosRequest,
	channelMappedModel string,
) (*OpenAIForwardResult, error) {
	if parsed == nil {
		return nil, fmt.Errorf("parsed videos request is required")
	}
	if account == nil {
		return nil, fmt.Errorf("account is required")
	}
	if account.Type != AccountTypeAPIKey {
		return nil, fmt.Errorf("videos endpoint requires api key account")
	}
	startTime := time.Now()
	if parsed.Endpoint == openAIVideosStatusEndpoint {
		return s.forwardOpenAIVideosStatus(ctx, c, account, parsed)
	}
	return s.forwardOpenAIVideosGeneration(ctx, c, account, parsed, channelMappedModel, startTime)
}

func (s *OpenAIGatewayService) forwardOpenAIVideosGeneration(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	parsed *OpenAIVideosRequest,
	channelMappedModel string,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	requestModel := strings.TrimSpace(parsed.Model)
	if mapped := strings.TrimSpace(channelMappedModel); mapped != "" {
		requestModel = mapped
	}
	if !isOpenAIVideoGenerationModel(requestModel) {
		return nil, fmt.Errorf("videos endpoint requires a video model, got %q", requestModel)
	}
	upstreamModel := strings.TrimSpace(account.GetMappedModel(requestModel))
	if upstreamModel == "" {
		upstreamModel = requestModel
	}
	if !isOpenAIVideoGenerationModel(upstreamModel) {
		return nil, fmt.Errorf("videos endpoint requires a video model, got %q", upstreamModel)
	}
	forwardBody, err := rewriteOpenAIVideosGenerationModel(parsed.Body, upstreamModel)
	if err != nil {
		return nil, err
	}
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIVideosGenerationRequest(ctx, c, account, forwardBody, token)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpUpstream.Do(upstreamReq, resolveProxyURL(account), account.ID, account.Concurrency)
	if err != nil {
		return nil, s.handleOpenAIUpstreamTransportError(ctx, c, account, err, false)
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := s.readOpenAIVideosBody(resp.Body, c)
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode >= 400 {
		return nil, s.handleOpenAIVideosErrorResponse(ctx, resp, body, c, account, upstreamModel)
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := "application/json"
	if s.cfg != nil && !s.cfg.Security.ResponseHeaders.Enabled {
		if upstreamType := resp.Header.Get("Content-Type"); upstreamType != "" {
			contentType = upstreamType
		}
	}
	c.Data(resp.StatusCode, contentType, body)
	requestID := strings.TrimSpace(gjson.GetBytes(body, "request_id").String())
	if requestID == "" {
		requestID = strings.TrimSpace(gjson.GetBytes(body, "id").String())
	}
	return &OpenAIForwardResult{
		RequestID:       requestID,
		Model:           strings.TrimSpace(parsed.Model),
		UpstreamModel:   upstreamModel,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
		VideoStatus:     "queued",
		VideoTerminal:   false,
		VideoSucceeded:  false,
	}, nil
}

func (s *OpenAIGatewayService) forwardOpenAIVideosStatus(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	parsed *OpenAIVideosRequest,
) (*OpenAIForwardResult, error) {
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	upstreamReq, err := s.buildOpenAIVideosStatusRequest(ctx, c, account, parsed.RequestID, token)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpUpstream.Do(upstreamReq, resolveProxyURL(account), account.ID, account.Concurrency)
	if err != nil {
		return nil, s.handleOpenAIUpstreamTransportError(ctx, c, account, err, false)
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := s.readOpenAIVideosBody(resp.Body, c)
	if readErr != nil {
		return nil, readErr
	}
	if resp.StatusCode >= 400 {
		return nil, s.handleOpenAIVideosErrorResponse(ctx, resp, body, c, account, "")
	}
	responseheaders.WriteFilteredHeaders(c.Writer.Header(), resp.Header, s.responseHeaderFilter)
	contentType := "application/json"
	if s.cfg != nil && !s.cfg.Security.ResponseHeaders.Enabled {
		if upstreamType := resp.Header.Get("Content-Type"); upstreamType != "" {
			contentType = upstreamType
		}
	}
	c.Data(resp.StatusCode, contentType, body)
	status := normalizeOpenAIVideoStatus(gjson.GetBytes(body, "status").String())
	videoSucceeded := isOpenAIVideoSuccessStatus(status, body)
	return &OpenAIForwardResult{
		RequestID:       parsed.RequestID,
		ResponseHeaders: resp.Header.Clone(),
		VideoStatus:     status,
		VideoTerminal:   determineOpenAIVideoTerminalStatus(status, body),
		VideoSucceeded:  videoSucceeded,
	}, nil
}

func (s *OpenAIGatewayService) buildOpenAIVideosGenerationRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	token string,
) (*http.Request, error) {
	targetURL := openAIVideosGenerationsURL
	baseURL := account.GetOpenAIBaseURL()
	if baseURL != "" {
		validatedURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		targetURL = buildOpenAIEndpointURL(validatedURL, openAIVideosGenerationsEndpoint)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Authorization", "Bearer "+token)
	for key, values := range c.Request.Header {
		if !openaiPassthroughAllowedHeaders[strings.ToLower(key)] {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Set("Content-Type", "application/json")
	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		req.Header.Set("User-Agent", customUA)
	}
	return req, nil
}

func (s *OpenAIGatewayService) buildOpenAIVideosStatusRequest(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	requestID string,
	token string,
) (*http.Request, error) {
	targetURL := buildOpenAIVideosStatusURL(openAIVideosURLPrefix, requestID)
	baseURL := account.GetOpenAIBaseURL()
	if baseURL != "" {
		validatedURL, err := s.validateUpstreamBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		targetURL = buildOpenAIVideosStatusURL(validatedURL, requestID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Header.Set("Authorization", "Bearer "+token)
	for key, values := range c.Request.Header {
		if !openaiPassthroughAllowedHeaders[strings.ToLower(key)] {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	if customUA := account.GetOpenAIUserAgent(); customUA != "" {
		req.Header.Set("User-Agent", customUA)
	}
	return req, nil
}

func buildOpenAIVideosStatusURL(base string, requestID string) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	requestID = strings.TrimLeft(strings.TrimSpace(requestID), "/")
	if normalized == "" {
		normalized = "https://api.openai.com"
	}
	if strings.HasSuffix(normalized, "/v1/videos") {
		return normalized + "/" + requestID
	}
	return buildOpenAIEndpointURL(normalized, "/v1/videos/"+requestID)
}

func rewriteOpenAIVideosGenerationModel(body []byte, model string) ([]byte, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return body, nil
	}
	rewritten, err := sjson.SetBytes(body, "model", model)
	if err != nil {
		return nil, fmt.Errorf("rewrite video request model: %w", err)
	}
	return rewritten, nil
}

func resolveProxyURL(account *Account) string {
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		return account.Proxy.URL()
	}
	return ""
}

func (s *OpenAIGatewayService) readOpenAIVideosBody(body io.Reader, c *gin.Context) ([]byte, error) {
	return ReadUpstreamResponseBody(body, s.cfg, c, openAITooLargeError)
}

func (s *OpenAIGatewayService) handleOpenAIVideosErrorResponse(
	ctx context.Context,
	resp *http.Response,
	body []byte,
	c *gin.Context,
	account *Account,
	requestedModel string,
) error {
	upstreamMsg := sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(body)))
	upstreamDetail := ""
	if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
		maxBytes := s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes
		if maxBytes <= 0 {
			maxBytes = 2048
		}
		upstreamDetail = truncateString(string(body), maxBytes)
	}
	setOpsUpstreamError(c, resp.StatusCode, upstreamMsg, upstreamDetail)
	if s.shouldFailoverOpenAIUpstreamResponse(resp.StatusCode, upstreamMsg, body) {
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: resp.StatusCode,
			UpstreamRequestID:  resp.Header.Get("x-request-id"),
			Kind:               "failover",
			Message:            upstreamMsg,
			Detail:             upstreamDetail,
		})
		s.handleFailoverSideEffects(ctx, resp, account, requestedModel)
		return &UpstreamFailoverError{
			StatusCode:   resp.StatusCode,
			ResponseBody: body,
		}
	}
	appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
		Platform:           account.Platform,
		AccountID:          account.ID,
		AccountName:        account.Name,
		UpstreamStatusCode: resp.StatusCode,
		UpstreamRequestID:  resp.Header.Get("x-request-id"),
		Kind:               "http_error",
		Message:            upstreamMsg,
		Detail:             upstreamDetail,
	})
	upErr := openAIVideosUpstreamErrorFromHTTP(resp.StatusCode, resp.Header, body)
	writeOpenAIVideosUpstreamErrorResponse(c, upErr)
	return upErr
}

type OpenAIVideosUpstreamError struct {
	StatusCode        int
	ErrorType         string
	Code              string
	Message           string
	Param             string
	UpstreamRequestID string
}

func (e *OpenAIVideosUpstreamError) Error() string {
	if e == nil {
		return ""
	}
	code := strings.TrimSpace(e.Code)
	if code == "" {
		code = strings.TrimSpace(e.ErrorType)
	}
	message := strings.TrimSpace(e.Message)
	if code != "" && message != "" {
		return fmt.Sprintf("openai videos upstream error: %s: %s", code, message)
	}
	if message != "" {
		return "openai videos upstream error: " + message
	}
	if code != "" {
		return "openai videos upstream error: " + code
	}
	return "openai videos upstream error"
}

func (e *OpenAIVideosUpstreamError) clientStatusCode() int {
	if e == nil {
		return http.StatusBadGateway
	}
	if e.StatusCode > 0 {
		return e.StatusCode
	}
	return http.StatusBadGateway
}

func (e *OpenAIVideosUpstreamError) clientErrorType() string {
	if e == nil {
		return "upstream_error"
	}
	if trimmed := strings.TrimSpace(e.ErrorType); trimmed != "" {
		return trimmed
	}
	return "upstream_error"
}

func (e *OpenAIVideosUpstreamError) clientMessage() string {
	if e == nil {
		return "Upstream request failed"
	}
	if trimmed := strings.TrimSpace(e.Message); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(e.Code); trimmed != "" {
		return trimmed
	}
	return "Upstream request failed"
}

func openAIVideosUpstreamErrorFromHTTP(statusCode int, header http.Header, body []byte) *OpenAIVideosUpstreamError {
	errType := strings.TrimSpace(gjson.GetBytes(body, "error.type").String())
	code := strings.TrimSpace(gjson.GetBytes(body, "error.code").String())
	message := strings.TrimSpace(gjson.GetBytes(body, "error.message").String())
	param := strings.TrimSpace(gjson.GetBytes(body, "error.param").String())
	if message == "" {
		message = sanitizeUpstreamErrorMessage(strings.TrimSpace(extractUpstreamErrorMessage(body)))
	}
	requestID := ""
	if header != nil {
		requestID = strings.TrimSpace(header.Get("x-request-id"))
	}
	return &OpenAIVideosUpstreamError{
		StatusCode:        statusCode,
		ErrorType:         errType,
		Code:              code,
		Message:           message,
		Param:             param,
		UpstreamRequestID: requestID,
	}
}

func writeOpenAIVideosUpstreamErrorResponse(c *gin.Context, err *OpenAIVideosUpstreamError) bool {
	if c == nil || c.Writer == nil || c.Writer.Written() || err == nil {
		return false
	}
	errorObj := gin.H{
		"type":    err.clientErrorType(),
		"message": err.clientMessage(),
	}
	if code := strings.TrimSpace(err.Code); code != "" {
		errorObj["code"] = code
	}
	if param := strings.TrimSpace(err.Param); param != "" {
		errorObj["param"] = param
	}
	c.JSON(err.clientStatusCode(), gin.H{"error": errorObj})
	return true
}

func normalizeOpenAIVideoResolutionTier(resolution string) string {
	switch strings.ToLower(strings.TrimSpace(resolution)) {
	case "480p":
		return ImageBillingSize1K
	case "1080p":
		return ImageBillingSize4K
	case "720p":
		fallthrough
	default:
		return ImageBillingSize2K
	}
}

func NormalizeOpenAIVideoResolutionTierForUsage(resolution string) string {
	return normalizeOpenAIVideoResolutionTier(resolution)
}

func normalizeOpenAIVideoStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func isOpenAIVideoTerminalStatus(status string) bool {
	switch normalizeOpenAIVideoStatus(status) {
	case "completed", "succeeded", "success", "failed", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func isOpenAIVideoSuccessStatus(status string, body []byte) bool {
	normalized := normalizeOpenAIVideoStatus(status)
	switch normalized {
	case "completed", "succeeded", "success":
		return true
	}
	if normalized != "" {
		return false
	}
	videoURL := strings.TrimSpace(gjson.GetBytes(body, "video.url").String())
	if videoURL == "" {
		videoURL = strings.TrimSpace(gjson.GetBytes(body, "data.0.url").String())
	}
	return videoURL != ""
}

func determineOpenAIVideoTerminalStatus(status string, body []byte) bool {
	if isOpenAIVideoTerminalStatus(status) {
		return true
	}
	return isOpenAIVideoSuccessStatus(status, body)
}

func openAIVideoPendingSessionHash(requestID string) string {
	return HashUsageRequestPayload([]byte("openai-video-pending|" + strings.TrimSpace(requestID)))
}

func encodeOpenAIVideoPendingUsage(model string, resolution string) (int64, bool) {
	trimmedModel := strings.ToLower(strings.TrimSpace(model))
	if trimmedModel == "" {
		return 0, false
	}
	trimmedResolution := strings.ToLower(strings.TrimSpace(resolution))
	if trimmedResolution == "" {
		trimmedResolution = inferOpenAIVideoResolutionFromModel(trimmedModel)
	}
	versionCode, ok := parseOpenAIVideoVersionCode(trimmedModel)
	if !ok {
		return 0, false
	}
	resolutionCode, ok := parseOpenAIVideoResolutionCode(trimmedResolution)
	if !ok {
		return 0, false
	}
	return int64(versionCode)*10000 + int64(resolutionCode), true
}

func decodeOpenAIVideoPendingUsage(encoded int64) (model string, resolution string, ok bool) {
	if encoded <= 0 {
		return "", "", false
	}
	versionCode := int(encoded / 10000)
	resolutionCode := int(encoded % 10000)
	resolution, ok = formatOpenAIVideoResolutionCode(resolutionCode)
	if !ok {
		return "", "", false
	}
	version, ok := formatOpenAIVideoVersionCode(versionCode)
	if !ok {
		return "", "", false
	}
	return "grok-imagine-video-" + version + "-" + resolution, resolution, true
}

func inferOpenAIVideoResolutionFromModel(model string) string {
	trimmed := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasSuffix(trimmed, "-480p"):
		return "480p"
	case strings.HasSuffix(trimmed, "-720p"):
		return "720p"
	case strings.HasSuffix(trimmed, "-1080p"):
		return "1080p"
	default:
		return ""
	}
}

func parseOpenAIVideoVersionCode(model string) (int, bool) {
	trimmed := strings.ToLower(strings.TrimSpace(model))
	const prefix = "grok-imagine-video-"
	if !strings.HasPrefix(trimmed, prefix) {
		return 0, false
	}
	remainder := strings.TrimPrefix(trimmed, prefix)
	idx := strings.LastIndex(remainder, "-")
	if idx <= 0 {
		return 0, false
	}
	version := remainder[:idx]
	parts := strings.SplitN(version, ".", 2)
	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 0 {
		return 0, false
	}
	minor := 0
	if len(parts) == 2 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil || minor < 0 || minor > 99 {
			return 0, false
		}
	}
	return major*100 + minor, true
}

func formatOpenAIVideoVersionCode(code int) (string, bool) {
	if code <= 0 {
		return "", false
	}
	major := code / 100
	minor := code % 100
	if minor == 0 {
		return strconv.Itoa(major), true
	}
	return fmt.Sprintf("%d.%d", major, minor), true
}

func parseOpenAIVideoResolutionCode(resolution string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(resolution)) {
	case "480p":
		return 480, true
	case "720p":
		return 720, true
	case "1080p":
		return 1080, true
	default:
		return 0, false
	}
}

func formatOpenAIVideoResolutionCode(code int) (string, bool) {
	switch code {
	case 480, 720, 1080:
		return fmt.Sprintf("%dp", code), true
	default:
		return "", false
	}
}

func (s *OpenAIGatewayService) BindOpenAIVideoPendingUsage(ctx context.Context, groupID *int64, requestID string, model string, resolution string) error {
	if s == nil || s.cache == nil {
		return nil
	}
	encoded, ok := encodeOpenAIVideoPendingUsage(model, resolution)
	if !ok {
		return nil
	}
	return s.cache.SetSessionAccountID(ctx, derefGroupID(groupID), openAIVideoPendingSessionHash(requestID), encoded, stickySessionTTL)
}

func (s *OpenAIGatewayService) GetOpenAIVideoPendingUsage(ctx context.Context, groupID *int64, requestID string) (model string, resolution string, ok bool) {
	if s == nil || s.cache == nil {
		return "", "", false
	}
	encoded, err := s.cache.GetSessionAccountID(ctx, derefGroupID(groupID), openAIVideoPendingSessionHash(requestID))
	if err != nil || encoded <= 0 {
		return "", "", false
	}
	return decodeOpenAIVideoPendingUsage(encoded)
}

func (s *OpenAIGatewayService) DeleteOpenAIVideoPendingUsage(ctx context.Context, groupID *int64, requestID string) error {
	if s == nil || s.cache == nil {
		return nil
	}
	return s.cache.DeleteSessionAccountID(ctx, derefGroupID(groupID), openAIVideoPendingSessionHash(requestID))
}
