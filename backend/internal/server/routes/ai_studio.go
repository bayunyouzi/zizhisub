package routes

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/tidwall/sjson"
)

// =============================================================================
// AI Studio 安全代理路由
//
// 设计目标（来自需求）：
//   - 提示词生成：默认模型 gpt-5.5，强制使用「系统提示词默认密钥」，所有人通用，
//     不开启思考模式，不限次数。
//   - 文生图 / 图生图：默认模型 gpt-image-2，默认使用「系统生图默认密钥」；
//     文生图+图生图「共享」每人每天 10 次（管理员豁免）；
//     若用户选择了自己的密钥，则不计数、不受限。
//   - 默认值与限额规则在服务端固化，前端点进去即生效，无需用户每次手动设置。
//
// 安全要点（务必遵守）：
//   - 系统默认密钥（sk-...）只允许通过环境变量注入，绝不写进代码或提交到仓库。
//   - 默认密钥永远停留在后端：浏览器拿不到，请求由后端注入后回环转发到本机 /v1 网关。
//   - 管理员豁免：role == admin 或邮箱命中白名单（默认含站长邮箱）。
//
// 环境变量：
//   AI_STUDIO_PROMPT_KEY    提示词生成用的系统默认密钥（sk-...）
//   AI_STUDIO_IMAGE_KEY     生图用的系统默认密钥（sk-...）
//   AI_STUDIO_PROMPT_MODEL  提示词默认模型（缺省 gpt-5.5）
//   AI_STUDIO_IMAGE_MODEL   生图默认模型（缺省 gpt-image-2）
//   AI_STUDIO_DAILY_LIMIT   默认密钥每日生图次数（缺省 10）
//   AI_STUDIO_ADMIN_EMAILS  逗号分隔的管理员邮箱白名单（无限使用）
//   AI_STUDIO_IMAGE_BASE_URL 生图直连地址（缺省=走本机网关）。
//     配置后生图请求绕过本机网关的 OAuth 分流，直连该地址的 /v1/images/* 端点。
//     用途：当网关 group 下绑的是 OAuth(ChatGPT Plus) 账号时，ChatGPT 内部会对
//     image_generation 输出做 web 缩放（返回非 16 倍数的小图），导致"请求 4K 但拿到
//     1672×941"。配置此项为 https://api.openai.com（或你的中转商地址）即可直连获取原图。
//     注意：直连模式不走网关计费/日志/failover，AI Studio 自有终身限额仍生效。
// =============================================================================

const (
	aiStudioDefaultPromptModel = "gpt-5.5"
	aiStudioDefaultImageModel  = "gpt-image-2"
	aiStudioDefaultFreeLimit   = 10
	// 站长邮箱：默认无限使用。也可通过 AI_STUDIO_ADMIN_EMAILS 覆盖/追加。
	aiStudioBuiltinAdminEmail = "1585062016@qq.com"
)

// aiStudioDeps 聚合 AI Studio 路由所需依赖。
type aiStudioDeps struct {
	redis        *redis.Client
	cfg          *config.Config
	promptKey    string
	imageKey     string
	promptModel  string
	imageModel   string
	freeLimit    int    // 每个账号终生免费次数（管理员无限）
	gatewayBase  string // 本机回环网关地址，如 http://127.0.0.1:8080/v1
	imageBaseURL string // 生图直连地址（AI_STUDIO_IMAGE_BASE_URL）。非空=直连 OpenAI 绕过网关 OAuth 分流，避免 ChatGPT 内部缩放导致非原图
	httpClient   *http.Client
}

// RegisterAIStudioRoutes 注册 /api/v1/ai-studio/* 路由（需要 JWT 登录态）。
func RegisterAIStudioRoutes(
	v1 *gin.RouterGroup,
	jwtAuth middleware.JWTAuthMiddleware,
	redisClient *redis.Client,
	cfg *config.Config,
) {
	deps := buildAIStudioDeps(redisClient, cfg)

	grp := v1.Group("/ai-studio")
	grp.Use(gin.HandlerFunc(jwtAuth))
	{
		// 返回前端所需的默认配置（默认模型、剩余次数等），不含任何密钥。
		grp.GET("/config", deps.handleConfig)
		// 提示词生成：固定默认模型，强制用系统提示词密钥，不限次数。
		grp.POST("/prompt", deps.handlePrompt)
		// 文生图：默认密钥计入每日共享限额；用户自带密钥不限。
		grp.POST("/image/generate", deps.handleImageGenerate)
		// 图生图：与文生图共享每日限额。
		grp.POST("/image/edit", deps.handleImageEdit)
	}
}

func buildAIStudioDeps(redisClient *redis.Client, cfg *config.Config) *aiStudioDeps {
	promptModel := strings.TrimSpace(os.Getenv("AI_STUDIO_PROMPT_MODEL"))
	if promptModel == "" {
		promptModel = aiStudioDefaultPromptModel
	}
	imageModel := strings.TrimSpace(os.Getenv("AI_STUDIO_IMAGE_MODEL"))
	if imageModel == "" {
		imageModel = aiStudioDefaultImageModel
	}
	freeLimit := aiStudioDefaultFreeLimit
	if v := strings.TrimSpace(os.Getenv("AI_STUDIO_FREE_LIMIT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			freeLimit = n
		}
	}

	admins := map[string]struct{}{
		strings.ToLower(aiStudioBuiltinAdminEmail): {},
	}
	for _, e := range strings.Split(os.Getenv("AI_STUDIO_ADMIN_EMAILS"), ",") {
		e = strings.ToLower(strings.TrimSpace(e))
		if e != "" {
			admins[e] = struct{}{}
		}
	}
	_ = admins // 站长账号 role 已是 admin，邮箱白名单暂保留为未来扩展点

	// 本机回环网关地址：直接打到自身 :port/v1，避免任何跨域/外网往返。
	port := cfg.Server.Port
	if port == 0 {
		port = 8080
	}
	gatewayBase := fmt.Sprintf("http://127.0.0.1:%d/v1", port)

	return &aiStudioDeps{
		redis:        redisClient,
		cfg:          cfg,
		promptKey:    strings.TrimSpace(os.Getenv("AI_STUDIO_PROMPT_KEY")),
		imageKey:     strings.TrimSpace(os.Getenv("AI_STUDIO_IMAGE_KEY")),
		promptModel:  promptModel,
		imageModel:   imageModel,
		freeLimit:    freeLimit,
		gatewayBase:  gatewayBase,
		imageBaseURL: strings.TrimSpace(os.Getenv("AI_STUDIO_IMAGE_BASE_URL")),
		httpClient: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
}

// ---------------------------------------------------------------------------
// GET /config —— 返回默认配置与剩余次数（绝不含密钥）
// ---------------------------------------------------------------------------

func (d *aiStudioDeps) handleConfig(c *gin.Context) {
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	isAdmin := d.isAdmin(c)

	used := d.peekImageUsage(c.Request.Context(), subject.UserID)
	remaining := d.freeLimit - used
	if remaining < 0 {
		remaining = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"prompt_model":          d.promptModel,
		"image_model":           d.imageModel,
		"has_prompt_key":        d.promptKey != "",
		"has_image_key":         d.imageKey != "",
		"free_image_limit":      d.freeLimit,
		"free_image_used":       used,
		"free_image_remaining":  remaining,
		"unlimited":             isAdmin,
	})
}

// ---------------------------------------------------------------------------
// POST /prompt —— 提示词生成（固定模型，系统密钥，不限次，不开思考）
// ---------------------------------------------------------------------------

type aiStudioPromptRequest struct {
	Theme  string   `json:"theme"`
	Styles []string `json:"styles"`
}

func (d *aiStudioDeps) handlePrompt(c *gin.Context) {
	if d.promptKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "提示词服务暂未配置（缺少系统密钥）"})
		return
	}

	var req aiStudioPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	theme := strings.TrimSpace(req.Theme)
	if theme == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主题不能为空"})
		return
	}

	styleHint := ""
	if len(req.Styles) > 0 {
		styleHint = "\n偏好风格：" + strings.Join(req.Styles, "、") + "。"
	}
	systemPrompt := "你是一位专业的 AI 绘图提示词工程师。请根据用户给的主题，生成一段高质量、" +
		"细节丰富的英文绘图提示词（prompt），包含主体、构图、光影、色彩、风格、画质等要素。" +
		"只输出提示词本身，不要解释、不要引号。"
	userPrompt := "主题：" + theme + styleHint

	// 不开启思考模式：显式非流式，并附常见"关闭思考"开关，上游支持则生效、不支持则忽略。
	payload := map[string]any{
		"model": d.promptModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature":       0.8,
		"stream":            false,
		"reasoning_effort":  "none",
		"enable_thinking":   false,
		"thinking":          map[string]any{"type": "disabled"},
	}

	status, body, err := d.forwardJSON(c.Request.Context(), "/chat/completions", d.promptKey, payload)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "提示词生成失败：" + err.Error()})
		return
	}
	if status < 200 || status >= 300 {
		c.Data(status, "application/json; charset=utf-8", body)
		return
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "上游返回无法解析"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"prompt": strings.TrimSpace(parsed.Choices[0].Message.Content)})
}

// ---------------------------------------------------------------------------
// POST /image/generate —— 文生图
// ---------------------------------------------------------------------------

type aiStudioImageGenRequest struct {
	Prompt   string `json:"prompt"`
	Model    string `json:"model"`
	Size     string `json:"size"`
	Quality  string `json:"quality"`
	N        int    `json:"n"`
	UserKey  string `json:"user_key"`  // 用户自带密钥（可空）。非空=用自己的，不限额。
}

func (d *aiStudioDeps) handleImageGenerate(c *gin.Context) {
	var req aiStudioImageGenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "提示词不能为空"})
		return
	}

	apiKey, usingDefault, errResp := d.resolveImageKey(c, req.UserKey)
	if errResp {
		return
	}

	// 仅在使用系统默认密钥且非管理员时，做每日共享限额（先占坑）。
	var commit func()
	if usingDefault {
		ok, used, release := d.consumeImageQuota(c)
		if !ok {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":     fmt.Sprintf("免费生图次数已用完（每账号 %d 次）。可填写你自己的密钥继续生成，不受限。", d.freeLimit),
				"used":      used,
				"limit":     d.freeLimit,
				"remaining": 0,
			})
			return
		}
		commit = release // 上游失败时回滚计数
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = d.imageModel
	}
	payload := map[string]any{
		"model":  model,
		"prompt": prompt,
	}
	if req.N > 0 {
		payload["n"] = req.N
	}
	if s := strings.TrimSpace(req.Size); s != "" {
		payload["size"] = s
	}
	if q := strings.TrimSpace(req.Quality); q != "" {
		payload["quality"] = q
	}

	status, body, err := d.forwardImageJSON(c.Request.Context(), "/images/generations", apiKey, payload)
	if err != nil {
		if commit != nil {
			commit() // 回滚
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "生图失败：" + err.Error()})
		return
	}
	if status < 200 || status >= 300 {
		if commit != nil {
			commit() // 上游业务失败也回滚，不浪费用户次数
		}
		c.Data(status, "application/json; charset=utf-8", body)
		return
	}

	// 解码真实像素并记录诊断日志，在响应中追加 image_actual_size 字段
	body = d.logAndAnnotateImageSize(body, strings.TrimSpace(req.Size))
	c.Data(status, "application/json; charset=utf-8", body)
}

// ---------------------------------------------------------------------------
// POST /image/edit —— 图生图（multipart）
// ---------------------------------------------------------------------------

func (d *aiStudioDeps) handleImageEdit(c *gin.Context) {
	prompt := strings.TrimSpace(c.PostForm("prompt"))
	if prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "提示词不能为空"})
		return
	}
	userKey := strings.TrimSpace(c.PostForm("user_key"))

	// 支持多张参考图（OpenAI images/edits 支持多个 image 字段）
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误，请使用 multipart/form-data"})
		return
	}
	fileHeaders := form.File["image"]
	if len(fileHeaders) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少参考图"})
		return
	}

	apiKey, usingDefault, errResp := d.resolveImageKey(c, userKey)
	if errResp {
		return
	}

	var commit func()
	if usingDefault {
		ok, used, release := d.consumeImageQuota(c)
		if !ok {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":     fmt.Sprintf("免费生图次数已用完（每账号 %d 次，文生图与图生图共享）。可填写你自己的密钥继续。", d.freeLimit),
				"used":      used,
				"limit":     d.freeLimit,
				"remaining": 0,
			})
			return
		}
		commit = release
	}

	model := strings.TrimSpace(c.PostForm("model"))
	if model == "" {
		model = d.imageModel
	}
	size := strings.TrimSpace(c.PostForm("size"))
	n := strings.TrimSpace(c.PostForm("n"))
	quality := strings.TrimSpace(c.PostForm("quality"))

	// 重新组装 multipart 转发到 /v1/images/edits
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("model", model)
	_ = mw.WriteField("prompt", prompt)
	if size != "" {
		_ = mw.WriteField("size", size)
	}
	if n != "" {
		_ = mw.WriteField("n", n)
	}
	if quality != "" {
		_ = mw.WriteField("quality", quality)
	}

	// 写入所有参考图
	for _, fh := range fileHeaders {
		src, err := fh.Open()
		if err != nil {
			if commit != nil {
				commit()
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "参考图读取失败: " + fh.Filename})
			return
		}
		part, err := mw.CreateFormFile("image", fh.Filename)
		if err == nil {
			_, err = io.Copy(part, src)
		}
		src.Close()
		if err != nil {
			if commit != nil {
				commit()
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "参考图处理失败"})
			return
		}
	}
	_ = mw.Close()

	status, body, err := d.forwardImageMultipart(c.Request.Context(), "/images/edits", apiKey, mw.FormDataContentType(), buf.Bytes())
	if err != nil {
		if commit != nil {
			commit()
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "图生图失败：" + err.Error()})
		return
	}
	if status < 200 || status >= 300 {
		if commit != nil {
			commit()
		}
		c.Data(status, "application/json; charset=utf-8", body)
		return
	}

	// 解码真实像素并记录诊断日志
	body = d.logAndAnnotateImageSize(body, size)
	c.Data(status, "application/json; charset=utf-8", body)
}

// ---------------------------------------------------------------------------
// 密钥解析 & 限额
// ---------------------------------------------------------------------------

// resolveImageKey 决定本次生图用哪把密钥。
// 返回 (apiKey, usingDefault, errorResponded)。
//   - 用户传了 user_key：用用户自己的，不限额。
//   - 否则用系统默认 imageKey（需已配置）。
func (d *aiStudioDeps) resolveImageKey(c *gin.Context, userKey string) (string, bool, bool) {
	userKey = strings.TrimSpace(userKey)
	if userKey != "" {
		return userKey, false, false
	}
	if d.imageKey == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "生图服务暂未配置默认密钥，请在下方填写你自己的密钥后再试。",
		})
		return "", false, true
	}
	return d.imageKey, true, false
}

// imageUsageKey 终生生图计数键（按用户隔离，文生图+图生图共享，永不过期）。
func (d *aiStudioDeps) imageUsageKey(userID int64) string {
	return fmt.Sprintf("ai_studio:img:%d:lifetime", userID)
}

func (d *aiStudioDeps) location() *time.Location {
	if d.cfg != nil && strings.TrimSpace(d.cfg.Timezone) != "" {
		if loc, err := time.LoadLocation(d.cfg.Timezone); err == nil {
			return loc
		}
	}
	return time.Local
}

// peekImageUsage 只读终生已用次数（不增加）。
func (d *aiStudioDeps) peekImageUsage(ctx context.Context, userID int64) int {
	if d.redis == nil || userID == 0 {
		return 0
	}
	val, err := d.redis.Get(ctx, d.imageUsageKey(userID)).Int()
	if err != nil {
		return 0
	}
	if val < 0 {
		return 0
	}
	return val
}

// consumeImageQuota 占用一次终生配额。
// 返回 (allowed, usedAfter, release)。release() 用于上游失败时回滚本次计数。
// 管理员豁免：直接放行且不计数。
func (d *aiStudioDeps) consumeImageQuota(c *gin.Context) (bool, int, func()) {
	if d.isAdmin(c) {
		return true, 0, nil
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	userID := subject.UserID
	if d.redis == nil || userID == 0 {
		return true, 0, nil
	}

	ctx := c.Request.Context()
	key := d.imageUsageKey(userID)

	count, err := d.redis.Incr(ctx, key).Result()
	if err != nil {
		return true, 0, nil
	}
	if int(count) > d.freeLimit {
		d.redis.Decr(ctx, key)
		return false, d.freeLimit, nil
	}

	release := func() {
		d.redis.Decr(context.Background(), key)
	}
	return true, int(count), release
}

func (d *aiStudioDeps) secondsUntilEndOfDay() time.Duration {
	now := time.Now().In(d.location())
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	dur := endOfDay.Sub(now)
	if dur < time.Minute {
		dur = time.Minute
	}
	return dur
}

// isAdmin 判定当前登录用户是否享有无限使用权。
// 站长账号（1585062016@qq.com）在系统里 role 即为 admin，JWT 中间件已将 role
// 写入 context，这里只需读 role 即可，无需回查数据库。
func (d *aiStudioDeps) isAdmin(c *gin.Context) bool {
	role, ok := middleware.GetUserRoleFromContext(c)
	return ok && role == service.RoleAdmin
}

// ---------------------------------------------------------------------------
// 生图直连 & 真实像素诊断
// ---------------------------------------------------------------------------

// resolveImageBase 返回生图请求应打的目标 base URL。
// 若配置了 AI_STUDIO_IMAGE_BASE_URL（如 https://api.openai.com），直连该地址，
// 绕过本机网关的 OAuth 分流——避免 ChatGPT 内部对 image_generation 输出做 web 缩放。
// 否则回环打本机网关（保留计费、日志、failover 等网关能力）。
func (d *aiStudioDeps) resolveImageBase() string {
	if d.imageBaseURL != "" {
		return d.imageBaseURL
	}
	return d.gatewayBase
}

// isDirectImageMode 是否走了直连模式（绕过本机网关）。
func (d *aiStudioDeps) isDirectImageMode() bool {
	return d.imageBaseURL != ""
}

// forwardImageJSON 向生图端点发送 JSON 请求（支持直连或回环网关）。
func (d *aiStudioDeps) forwardImageJSON(ctx context.Context, path, apiKey string, payload any) (int, []byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.resolveImageBase()+path, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	return d.do(req)
}

// forwardImageMultipart 向生图端点发送 multipart 请求（支持直连或回环网关）。
func (d *aiStudioDeps) forwardImageMultipart(ctx context.Context, path, apiKey, contentType string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.resolveImageBase()+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	return d.do(req)
}

// decodeActualImageSize 从 OpenAI Images 响应中解码第一张图片的真实像素尺寸。
// gpt-image-2 原生输出尺寸必须是 16 的倍数；若实际尺寸非 16 倍数，说明被上游缩放了。
func decodeActualImageSize(body []byte) (width, height int) {
	var resp struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil || len(resp.Data) == 0 {
		return 0, 0
	}
	b64 := strings.TrimSpace(resp.Data[0].B64JSON)
	if b64 == "" {
		return 0, 0
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return 0, 0
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

// logAndAnnotateImageSize 解码响应图片的真实像素，记录诊断日志，
// 并在响应 JSON 中追加 image_actual_size 字段供前端展示。
func (d *aiStudioDeps) logAndAnnotateImageSize(body []byte, requestedSize string) []byte {
	w, h := decodeActualImageSize(body)
	if w <= 0 || h <= 0 {
		return body
	}
	route := "本机网关"
	if d.isDirectImageMode() {
		route = "直连(" + d.imageBaseURL + ")"
	}
	actual := fmt.Sprintf("%dx%d", w, h)
	mismatch := requestedSize != "" && requestedSize != actual
	if mismatch {
		logger.LegacyPrintf("ai_studio.image",
			"[AI Studio] ⚠️ 生图尺寸不一致 路径=%s 请求size=%s 实际像素=%s — 图片被上游缩放！建议配置 AI_STUDIO_IMAGE_BASE_URL 直连 OpenAI 获取原图",
			route, requestedSize, actual)
	} else {
		logger.LegacyPrintf("ai_studio.image",
			"[AI Studio] ✓ 生图尺寸一致 路径=%s 请求size=%s 实际像素=%s",
			route, requestedSize, actual)
	}
	// 在响应 JSON 中追加真实尺寸字段，前端可读取展示
	if annotated, err := sjson.SetBytes(body, "image_actual_size", actual); err == nil {
		return annotated
	}
	return body
}

// ---------------------------------------------------------------------------
// 回环转发到本机 /v1 网关
// ---------------------------------------------------------------------------

func (d *aiStudioDeps) forwardJSON(ctx context.Context, path, apiKey string, payload any) (int, []byte, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.gatewayBase+path, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	return d.do(req)
}

func (d *aiStudioDeps) forwardMultipart(ctx context.Context, path, apiKey, contentType string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.gatewayBase+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	return d.do(req)
}

func (d *aiStudioDeps) do(req *http.Request) (int, []byte, error) {
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}
