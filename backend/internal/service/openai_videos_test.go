package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayServiceParseOpenAIVideosRequest_Generation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"grok-imagine-video-1.5-720p","prompt":"animate this scene","image":{"url":"https://example.com/source.png"},"duration":8,"aspect_ratio":"16:9","resolution":"1080p"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/videos/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIVideosRequest(c, body)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, openAIVideosGenerationsEndpoint, parsed.Endpoint)
	require.Equal(t, "grok-imagine-video-1.5-720p", parsed.Model)
	require.Equal(t, "animate this scene", parsed.Prompt)
	require.Equal(t, "https://example.com/source.png", parsed.ImageURL)
	require.Equal(t, 8, parsed.Duration)
	require.Equal(t, "16:9", parsed.AspectRatio)
	require.Equal(t, "1080p", parsed.Resolution)
	require.NotEmpty(t, parsed.StickySessionSeed())
}

func TestOpenAIGatewayServiceParseOpenAIVideosRequest_Status(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest(http.MethodGet, "/v1/videos/req_video_123", nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	c.Params = gin.Params{{Key: "request_id", Value: "req_video_123"}}

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIVideosRequest(c, nil)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, openAIVideosStatusEndpoint, parsed.Endpoint)
	require.Equal(t, "req_video_123", parsed.RequestID)
}

func TestOpenAIGatewayServiceParseOpenAIVideosRequest_RejectsNonVideoModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-image-2","prompt":"animate this scene","image":{"url":"https://example.com/source.png"}}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/videos/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIVideosRequest(c, body)
	require.Nil(t, parsed)
	require.ErrorContains(t, err, `videos endpoint requires a video model, got "gpt-image-2"`)
}

func TestBuildOpenAIVideosStatusURL_HandlesVersionedBaseURL(t *testing.T) {
	require.Equal(t,
		"https://video-upstream.example/v1/videos/req_video_123",
		buildOpenAIVideosStatusURL("https://video-upstream.example/v1", "req_video_123"),
	)
	require.Equal(t,
		"https://video-upstream.example/v1/videos/req_video_123",
		buildOpenAIVideosStatusURL("https://video-upstream.example/v1/videos", "req_video_123"),
	)
	require.Equal(t,
		"https://video-upstream.example/v1/videos/req_video_123",
		buildOpenAIVideosStatusURL("https://video-upstream.example", "req_video_123"),
	)
}

func TestNormalizeOpenAIVideoResolutionTier(t *testing.T) {
	require.Equal(t, ImageBillingSize1K, normalizeOpenAIVideoResolutionTier("480p"))
	require.Equal(t, ImageBillingSize2K, normalizeOpenAIVideoResolutionTier("720p"))
	require.Equal(t, ImageBillingSize4K, normalizeOpenAIVideoResolutionTier("1080p"))
	require.Equal(t, ImageBillingSize2K, normalizeOpenAIVideoResolutionTier("unknown"))
}

func TestAccountSupportsOpenAIEndpointCapabilityVideos(t *testing.T) {
	apiKeyAccount := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
	}
	require.True(t, apiKeyAccount.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityVideos))

	oauthAccount := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	}
	require.False(t, oauthAccount.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityVideos))

	limitedAccount := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"openai_capabilities": []any{"chat_completions"},
		},
	}
	require.False(t, limitedAccount.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityVideos))

	enabledAccount := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"openai_capabilities": []any{"chat_completions", "videos"},
		},
	}
	require.True(t, enabledAccount.SupportsOpenAIEndpointCapability(OpenAIEndpointCapabilityVideos))
}
