/**
 * AI Studio API client
 *
 * 走后端 /api/v1/ai-studio/* 端点（带登录态，由 apiClient 自动附 JWT）。
 *
 * 与 gateway.ts 的区别：
 * - gateway.ts：前端直接拿「用户自己的明文密钥」打 OpenAI 兼容网关。
 * - 这里：默认使用「系统默认密钥」，密钥只存在后端、由后端注入转发，
 *   浏览器永远拿不到。用于「所有人通用默认密钥 + 每日限额」的场景。
 *
 * 行为：
 * - 提示词生成：固定默认模型、系统密钥、不限次、不开思考。
 * - 文生图 / 图生图：默认用系统密钥并计入每日共享限额（管理员豁免）；
 *   若传入 userKey（用户自己的密钥），后端改用该密钥且不限额。
 */

import { apiClient } from './client'

export interface AIStudioConfig {
  prompt_model: string
  image_model: string
  has_prompt_key: boolean
  has_image_key: boolean
  free_image_limit: number
  free_image_used: number
  free_image_remaining: number
  unlimited: boolean
}

export interface AIStudioImage {
  url?: string
  b64_json?: string
  [key: string]: unknown
}

/** 拉取默认配置 + 今日剩余生图次数（不含密钥）。 */
export async function getAIStudioConfig(): Promise<AIStudioConfig> {
  const { data } = await apiClient.get<AIStudioConfig>('/ai-studio/config')
  return data
}

/** 生成绘图提示词（系统密钥、固定默认模型、不限次）。返回提示词文本。 */
export async function generatePromptViaStudio(params: {
  theme: string
  styles?: string[]
}): Promise<string> {
  const { data } = await apiClient.post<{ prompt: string }>('/ai-studio/prompt', {
    theme: params.theme,
    styles: params.styles || []
  })
  return data.prompt || ''
}

/** 文生图。userKey 非空时用用户自己的密钥（不限额）。 */
export async function generateImageViaStudio(params: {
  prompt: string
  model?: string
  size?: string
  quality?: string
  n?: number
  userKey?: string
}): Promise<AIStudioImage[]> {
  const { data } = await apiClient.post<{ data?: AIStudioImage[] }>('/ai-studio/image/generate', {
    prompt: params.prompt,
    model: params.model || '',
    size: params.size || '',
    quality: params.quality || '',
    n: params.n || 1,
    user_key: params.userKey || ''
  })
  return data?.data || []
}

/** 图生图 / 参考图生图（multipart）。当前仅支持单张参考图。userKey 非空时用用户自己的密钥（不限额）。 */
export async function editImageViaStudio(params: {
  prompt: string
  images: File[]
  model?: string
  size?: string
  quality?: string
  n?: number
  userKey?: string
}): Promise<AIStudioImage[]> {
  const form = new FormData()
  form.append('prompt', params.prompt)
  if (params.images.length > 0) {
    form.append('image', params.images[0])
  }
  if (params.model) form.append('model', params.model)
  if (params.size) form.append('size', params.size)
  if (params.quality) form.append('quality', params.quality)
  if (params.n) form.append('n', String(params.n))
  if (params.userKey) form.append('user_key', params.userKey)

  // 注意：不能手动设置 Content-Type: multipart/form-data，否则会缺少 boundary，
  // 后端无法解析 multipart body。让 axios 检测到 FormData 后自动生成带 boundary 的 header。
  const { data } = await apiClient.post<{ data?: AIStudioImage[] }>(
    '/ai-studio/image/edit',
    form
  )
  return data?.data || []
}
