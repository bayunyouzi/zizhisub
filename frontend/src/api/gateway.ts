/**
 * Gateway API client (OpenAI-compatible endpoints)
 *
 * 这些请求面向 OpenAI 兼容网关。默认走**当前站点同源** /v1
 * （也就是你自己部署的 sub2api 网关），用本面板创建的密钥作为 Bearer Token。
 *
 * 为什么默认同源？
 *   你在本面板创建的密钥，本来就是给「你自己这个 sub2api 网关」用的。
 *   而浏览器直连第三方跨域网关（如 ai.bayunzi.shop）会被 CORS 拦截
 *   （报 Failed to fetch），因为对方没有放开跨域。同源调用零跨域问题。
 *
 * 与管理后台 API（/api/v1，走 apiClient）完全不同：
 * - 使用用户自行选择的"已创建密钥"作为 Bearer Token
 * - 不携带 cookie / 后台登录态
 * - 端点：/models、/chat/completions、/images/generations、/images/edits
 *
 * 网关 base url 优先级：
 *   1. 显式传入的 baseUrl 参数（用户在页面手动填写时）
 *   2. 当前站点同源 origin + /v1（默认，零跨域）
 */

/**
 * 同源网关 base：取当前页面 origin 拼 /v1。
 * SSR / 无 window 环境兜底为相对路径 /v1。
 */
function sameOriginBase(): string {
  if (typeof window !== 'undefined' && window.location?.origin) {
    return `${window.location.origin}/v1`
  }
  return '/v1'
}

/**
 * 归一化网关 base url：确保以 /v1 结尾、无多余斜杠。
 * 接受形如 https://x.com、https://x.com/、https://x.com/v1 的输入。
 * 空值时回落到「当前站点同源 /v1」。
 */
export function normalizeGatewayBase(raw?: string | null): string {
  let base = (raw || '').trim()
  if (!base) return sameOriginBase()
  base = base.replace(/\/+$/, '') // 去掉末尾斜杠
  if (/\/v\d+$/.test(base)) return base // 已带 /v1、/v2 等版本段
  return `${base}/v1`
}

export interface GatewayModel {
  id: string
  object?: string
  created?: number
  owned_by?: string
  [key: string]: unknown
}

export interface ChatMessage {
  role: 'system' | 'user' | 'assistant'
  content:
    | string
    | Array<
        | { type: 'text'; text: string }
        | { type: 'image_url'; image_url: { url: string } }
      >
}

export interface GeneratedImage {
  url?: string
  b64_json?: string
  revised_prompt?: string
}

/**
 * 统一的网关错误归一化：把后端各种错误结构抽成一句可读信息。
 */
async function parseGatewayError(resp: Response): Promise<string> {
  let detail = ''
  try {
    const data = await resp.json()
    detail =
      data?.error?.message ||
      data?.error ||
      data?.message ||
      data?.detail ||
      ''
  } catch {
    try {
      detail = await resp.text()
    } catch {
      detail = ''
    }
  }
  const prefix = `HTTP ${resp.status}`
  return detail ? `${prefix}: ${detail}` : prefix
}

interface RequestOpts {
  apiKey: string
  baseUrl?: string | null
  signal?: AbortSignal
}

function authHeaders(apiKey: string): Record<string, string> {
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${apiKey}`
  }
}

/**
 * 查询模型列表 GET /models
 */
export async function listModels(opts: RequestOpts): Promise<GatewayModel[]> {
  const base = normalizeGatewayBase(opts.baseUrl)
  const resp = await fetch(`${base}/models`, {
    method: 'GET',
    headers: authHeaders(opts.apiKey),
    signal: opts.signal
  })
  if (!resp.ok) throw new Error(await parseGatewayError(resp))
  const data = await resp.json()
  // 兼容 { data: [...] } 与直接数组两种返回
  const list: GatewayModel[] = Array.isArray(data) ? data : data?.data ?? []
  return list
}

export interface ChatCompletionOpts extends RequestOpts {
  model: string
  messages: ChatMessage[]
  temperature?: number
  max_tokens?: number
}

/**
 * 对话补全 POST /chat/completions（非流式），返回首条回复的文本。
 */
export async function chatCompletion(opts: ChatCompletionOpts): Promise<string> {
  const base = normalizeGatewayBase(opts.baseUrl)
  const body: Record<string, unknown> = {
    model: opts.model,
    messages: opts.messages,
    stream: false
  }
  if (opts.temperature !== undefined) body.temperature = opts.temperature
  if (opts.max_tokens !== undefined) body.max_tokens = opts.max_tokens

  const resp = await fetch(`${base}/chat/completions`, {
    method: 'POST',
    headers: authHeaders(opts.apiKey),
    body: JSON.stringify(body),
    signal: opts.signal
  })
  if (!resp.ok) throw new Error(await parseGatewayError(resp))
  const data = await resp.json()
  const content: string = data?.choices?.[0]?.message?.content ?? ''
  return content
}

export interface ImageGenerationOpts extends RequestOpts {
  model: string
  prompt: string
  n?: number
  size?: string
}

/**
 * 文生图 POST /images/generations
 */
export async function generateImages(opts: ImageGenerationOpts): Promise<GeneratedImage[]> {
  const base = normalizeGatewayBase(opts.baseUrl)
  const body: Record<string, unknown> = {
    model: opts.model,
    prompt: opts.prompt
  }
  if (opts.n !== undefined) body.n = opts.n
  if (opts.size) body.size = opts.size

  const resp = await fetch(`${base}/images/generations`, {
    method: 'POST',
    headers: authHeaders(opts.apiKey),
    body: JSON.stringify(body),
    signal: opts.signal
  })
  if (!resp.ok) throw new Error(await parseGatewayError(resp))
  const data = await resp.json()
  return (data?.data ?? []) as GeneratedImage[]
}

export interface ImageEditOpts extends RequestOpts {
  model: string
  prompt: string
  image: File
  n?: number
  size?: string
}

/**
 * 图生图 POST /images/edits（multipart/form-data）。
 * 注意：此处不能带 Content-Type: application/json，需让浏览器自动设置 boundary。
 */
export async function editImage(opts: ImageEditOpts): Promise<GeneratedImage[]> {
  const base = normalizeGatewayBase(opts.baseUrl)
  const form = new FormData()
  form.append('model', opts.model)
  form.append('prompt', opts.prompt)
  form.append('image', opts.image)
  if (opts.n !== undefined) form.append('n', String(opts.n))
  if (opts.size) form.append('size', opts.size)

  const resp = await fetch(`${base}/images/edits`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${opts.apiKey}` },
    body: form,
    signal: opts.signal
  })
  if (!resp.ok) throw new Error(await parseGatewayError(resp))
  const data = await resp.json()
  return (data?.data ?? []) as GeneratedImage[]
}

export const gatewayAPI = {
  normalizeGatewayBase,
  listModels,
  chatCompletion,
  generateImages,
  editImage
}

export default gatewayAPI
