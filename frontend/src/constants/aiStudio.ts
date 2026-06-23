/**
 * AI Studio 相关常量
 */

// 提示词页 → 绘图页 的跨页传递键（使用 sessionStorage，刷新即清，符合"纯前端临时"约束）
export const PROMPT_HANDOFF_KEY = 'ai_studio_prompt_handoff'

// 默认网关 base url（当后台未配置 api_base_url 时兜底）
export const DEFAULT_GATEWAY_BASE = 'https://ai.bayunzi.shop/v1'
