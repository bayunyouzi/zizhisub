<template>
  <AppLayout>
    <div class="mx-auto max-w-[1600px]">
      <!-- Page Header -->
      <div class="mb-5 flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400"><Icon name="photo" size="lg" /></div>
          <div>
            <h1 class="text-xl font-semibold tracking-tight text-gray-900 dark:text-white">{{ t('aiStudio.image.title') }}</h1>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('aiStudio.image.subtitle') }}</p>
          </div>
        </div>
        <p v-if="keyMode === 'default'" class="text-xs" :class="quotaHintClass">
          <template v-if="cfg?.unlimited">{{ t('aiStudio.image.quotaUnlimited') }}</template>
          <template v-else>{{ t('aiStudio.image.quotaRemaining', { remaining, limit: dailyLimit }) }}</template>
        </p>
      </div>

      <!-- Control Bar -->
      <div class="card card-body mb-5 space-y-3">
        <!-- Row 1: Mode + Key + Model -->
        <div class="flex flex-wrap items-center gap-2">
          <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
            <button v-for="m in modes" :key="m.value" type="button" class="rounded-md px-3 py-1.5 text-xs font-medium transition-all"
              :class="mode === m.value ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500 hover:text-gray-700 dark:text-gray-400'" @click="mode = m.value">{{ m.label }}</button>
          </div>
          <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
            <button type="button" class="rounded-md px-3 py-1.5 text-xs font-medium transition-all"
              :class="keyMode === 'default' ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500'" @click="keyMode = 'default'">系统密钥</button>
            <button type="button" class="rounded-md px-3 py-1.5 text-xs font-medium transition-all"
              :class="keyMode === 'own' ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white' : 'text-gray-500'" @click="keyMode = 'own'">自备密钥</button>
          </div>
          <input v-model="model" type="text" :placeholder="'模型 (默认: '+imageModelDefault+')'" class="input w-40 text-sm" />
          <KeySelector v-if="keyMode === 'own'" v-model="selectedKey" class="w-40" />
        </div>

        <!-- Row 2: Prompt -->
        <div>
          <textarea v-model="prompt" rows="3" :placeholder="t('aiStudio.image.promptPlaceholder')" class="input resize-none text-sm"></textarea>
          <div class="mt-1 flex flex-wrap items-center gap-3">
            <button type="button" class="inline-flex items-center gap-1 text-xs text-primary-500 hover:text-primary-600" @click="showPromptHelper = !showPromptHelper">
              <Icon name="sparkles" size="sm" /> {{ showPromptHelper ? '收起提示词助手' : 'AI 帮你写提示词' }}
            </button>
            <label class="inline-flex cursor-pointer items-center gap-1 text-xs text-gray-500 hover:text-gray-700 dark:text-gray-400">
              <Icon name="photo" size="sm" /> 参考图
              <input ref="fileInputRef" type="file" accept="image/*" class="hidden" multiple @change="onFileChange" />
            </label>
          </div>
          <Transition enter-active-class="transition-all duration-200" enter-from-class="opacity-0 -translate-y-1" enter-to-class="opacity-100" leave-to-class="opacity-0">
            <div v-if="showPromptHelper" class="mt-2 rounded-xl border border-amber-100 bg-amber-50/60 p-3 dark:border-amber-900/30 dark:bg-amber-900/10">
              <div class="mb-2 flex flex-wrap gap-1.5">
                <button v-for="s in stylePresets" :key="s.value" type="button" class="rounded-full px-2 py-0.5 text-[11px] font-medium transition-all"
                  :class="selectedStyles.includes(s.value) ? 'bg-primary-500 text-white' : 'bg-white text-gray-600 hover:bg-gray-100 dark:bg-dark-700 dark:text-gray-300'" @click="toggleStyle(s.value)">{{ s.label }}</button>
              </div>
              <div class="flex gap-2">
                <input v-model="theme" :placeholder="t('aiStudio.prompt.themePlaceholder')" class="input flex-1 text-xs" />
                <button type="button" class="btn btn-secondary btn-sm flex-shrink-0" :disabled="!theme.trim() || promptLoading" @click="generatePrompt">
                  <Icon name="sparkles" size="sm" :class="promptLoading ? 'animate-pulse' : ''" />
                </button>
              </div>
            </div>
          </Transition>
        </div>

        <!-- Ref previews -->
        <div v-if="refPreviews.length > 0" class="flex flex-wrap items-center gap-2">
          <div v-for="(f, idx) in refPreviews" :key="idx" class="relative">
            <img :src="f" :alt="'ref-'+idx" class="h-16 w-16 rounded-lg object-cover" />
            <button class="absolute -top-1.5 -right-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-red-500 text-white text-[10px]" @click="removeRef(idx)">×</button>
          </div>
          <button v-if="refFiles.length < 2" class="btn btn-ghost btn-sm" @click="triggerFileInput"><Icon name="plus" size="sm" /></button>
          <button class="btn btn-ghost btn-sm text-red-500" @click="clearAllRefs">清空</button>
        </div>

        <!-- Row 3: Size + Quality + Count + Generate -->
        <div class="flex flex-wrap items-end gap-2">
          <div class="flex flex-wrap gap-1.5">
            <button v-for="opt in sizeOptions" :key="opt.value" type="button" class="rounded-lg px-3 py-1.5 text-xs font-medium transition-all"
              :class="size === opt.value ? 'bg-primary-500 text-white shadow-sm' : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'" @click="size = opt.value">{{ opt.label }}</button>
          </div>
          <select v-model="quality" class="input w-20 text-xs">
            <option v-for="q in qualityOptions" :key="q.value" :value="q.value">{{ q.label }}</option>
          </select>
          <select v-model.number="count" class="input w-16 text-xs">
            <option v-for="n in [1,2,3,4]" :key="n" :value="n">{{ n }}张</option>
          </select>
          <button class="btn btn-primary flex-1 sm:flex-none" :disabled="!canGenerate || loading" @click="generate">
            <Icon name="sparkles" size="sm" :class="loading ? 'animate-pulse' : ''" />
            {{ loading ? t('aiStudio.image.generating') : t('aiStudio.image.generate') }}
          </button>
        </div>
      </div>

      <!-- Results Grid -->
      <div class="card card-body min-h-[400px]">
        <div v-if="loading" class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div v-for="n in count" :key="n" class="aspect-[3/4] animate-pulse rounded-2xl bg-gray-100 dark:bg-dark-700"></div>
        </div>
        <div v-else-if="images.length > 0" class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          <div v-for="(img, i) in images" :key="i" class="group relative overflow-hidden rounded-2xl border border-gray-100 dark:border-dark-700">
            <img :src="img" :alt="'result-'+i" class="aspect-[3/4] w-full cursor-zoom-in object-cover transition-transform duration-300 group-hover:scale-[1.02]" @click="openLightbox(img)" />
            <div class="pointer-events-none absolute inset-0 flex items-end justify-end bg-gradient-to-t from-black/50 to-transparent p-3 opacity-0 transition-opacity group-hover:opacity-100">
              <div class="pointer-events-auto flex gap-2">
                <button class="rounded-xl bg-white/90 p-2.5 text-gray-800 shadow-md backdrop-blur transition-transform hover:scale-105" title="放大" @click.stop="openLightbox(img)"><Icon name="search" size="sm" /></button>
                <button class="rounded-xl bg-white/90 p-2.5 text-gray-800 shadow-md backdrop-blur transition-transform hover:scale-105" :title="t('common.download')" @click.stop="download(img, i)"><Icon name="download" size="sm" /></button>
              </div>
            </div>
          </div>
        </div>
        <div v-else class="flex h-full min-h-[300px] flex-col items-center justify-center text-center">
          <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-purple-50 text-purple-400 dark:bg-purple-900/20 dark:text-purple-500"><Icon name="photo" size="xl" /></div>
          <p class="text-gray-500 dark:text-gray-400">{{ t('aiStudio.image.emptyHint') }}</p>
        </div>
      </div>
    </div>

    <!-- Lightbox -->
    <Teleport to="body">
      <Transition name="lightbox-fade">
        <div v-if="lightboxSrc" class="fixed inset-0 z-[9999] flex items-center justify-center bg-black/80 backdrop-blur-md p-4" @click="closeLightbox">
          <button class="absolute right-5 top-5 rounded-full bg-white/10 p-3 text-white backdrop-blur transition-all hover:bg-white/20 hover:scale-110" title="关闭" @click.stop="closeLightbox"><Icon name="x" size="md" /></button>
          <img :src="lightboxSrc" alt="preview" class="max-h-[90vh] max-w-[90vw] rounded-2xl object-contain shadow-2xl" @click.stop />
        </div>
      </Transition>
    </Teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import KeySelector from '@/components/ai/KeySelector.vue'
import {
  getAIStudioConfig,
  generatePromptViaStudio,
  generateImageViaStudio,
  editImageViaStudio,
  type AIStudioConfig,
  type AIStudioImage
} from '@/api/aiStudio'
import { useAppStore } from '@/stores/app'

type Mode = 'generate' | 'edit'
type KeyMode = 'default' | 'own'

const { t } = useI18n()
const appStore = useAppStore()

const mode = ref<Mode>('generate')
// 密钥来源：default=系统默认密钥(有每日限额)，own=用户自己的密钥(不限)
const keyMode = ref<KeyMode>('default')
const selectedKey = ref('')
const model = ref('')
const prompt = ref('')
const size = ref('1024x1024')
const quality = ref('auto')
const count = ref(1)
const images = ref<string[]>([])
const loading = ref(false)

// Lightbox 放大查看
const lightboxSrc = ref<string | null>(null)
function openLightbox(src: string) {
  lightboxSrc.value = src
}
function closeLightbox() {
  lightboxSrc.value = null
}
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && lightboxSrc.value) closeLightbox()
}

// 后端下发的默认配置（默认模型、剩余次数等）
const cfg = ref<AIStudioConfig | null>(null)
const imageModelDefault = ref('gpt-image-2')
const promptModelDefault = ref('gpt-5.5')

// 参考图（文生图和图生图都支持，最多2张）
const refFiles = ref<File[]>([])
const refPreviews = ref<string[]>([])
const fileInputRef = ref<HTMLInputElement | null>(null)

// GPT Image 2 支持的尺寸选项（已验证文档）
// 规则: 长边≤3840px, 宽高为16倍数, 比例≤3:1, 总像素655360~8294400
const sizeOptions = [
  { value: '1024x1024', label: '1:1 正方形' },
  { value: '1536x1024', label: '3:2 横屏' },
  { value: '1024x1536', label: '2:3 竖屏' },
  { value: '2048x2048', label: '1:1 2K' },
  { value: '2048x1152', label: '16:9 2K横屏' },
  { value: '3840x2160', label: '16:9 4K横屏' },
  { value: '2160x3840', label: '9:16 4K竖屏' },
]

// 质量选项
const qualityOptions = [
  { value: 'auto', label: '自动' },
  { value: 'low', label: '低（快速草稿）' },
  { value: 'medium', label: '中（平衡）' },
  { value: 'high', label: '高（最佳质量）' },
]

// ===== 内联 AI 提示词生成器状态 =====
const showPromptHelper = ref(false)
const theme = ref('')
const selectedStyles = ref<string[]>([])
const promptLoading = ref(false)

const stylePresets = computed(() => [
  { value: 'photorealistic', label: t('aiStudio.prompt.styles.photorealistic') },
  { value: 'cinematic', label: t('aiStudio.prompt.styles.cinematic') },
  { value: 'anime', label: t('aiStudio.prompt.styles.anime') },
  { value: 'oil', label: t('aiStudio.prompt.styles.oil') },
  { value: 'watercolor', label: t('aiStudio.prompt.styles.watercolor') },
  { value: '3d', label: t('aiStudio.prompt.styles.threeD') },
  { value: 'minimal', label: t('aiStudio.prompt.styles.minimal') },
  { value: 'cyberpunk', label: t('aiStudio.prompt.styles.cyberpunk') }
])

// 每日限额展示
const dailyLimit = computed(() => cfg.value?.daily_image_limit ?? 10)
const remaining = computed(() => cfg.value?.daily_image_remaining ?? dailyLimit.value)
const quotaHintClass = computed(() =>
  remaining.value <= 0 && !cfg.value?.unlimited
    ? 'text-red-500'
    : 'text-gray-400 dark:text-gray-500'
)

function toggleStyle(value: string) {
  const idx = selectedStyles.value.indexOf(value)
  if (idx >= 0) selectedStyles.value.splice(idx, 1)
  else selectedStyles.value.push(value)
}

/**
 * 调用后端生成绘图提示词（系统密钥、固定默认模型、不限次、不开思考），
 * 成功后直接填入下方 prompt 文本框。
 */
async function generatePrompt() {
  if (!theme.value.trim()) return
  promptLoading.value = true
  try {
    const text = await generatePromptViaStudio({
      theme: theme.value.trim(),
      styles: selectedStyles.value
    })
    if (text.trim()) {
      prompt.value = text.trim()
      appStore.showSuccess(t('aiStudio.image.promptFilled'))
    } else {
      appStore.showError(t('aiStudio.prompt.emptyResult'))
    }
  } catch (err: unknown) {
    appStore.showError(resolveError(err))
  } finally {
    promptLoading.value = false
  }
}

const modes = computed(() => [
  { value: 'generate' as Mode, label: t('aiStudio.image.modeGenerate') },
  { value: 'edit' as Mode, label: t('aiStudio.image.modeEdit') }
])

const canGenerate = computed(() => {
  if (!prompt.value.trim()) return false
  // 用自己的密钥时必须先选一把
  if (keyMode.value === 'own' && !selectedKey.value) return false
  // 图生图模式必须有参考图
  if (mode.value === 'edit' && refFiles.value.length === 0) return false
  return true
})

// 进入页面即拉默认配置：默认模型、剩余次数等。设置一次记忆在后端，无需用户每次改。
onMounted(async () => {
  try {
    const data = await getAIStudioConfig()
    cfg.value = data
    imageModelDefault.value = data.image_model || 'gpt-image-2'
    promptModelDefault.value = data.prompt_model || 'gpt-5.5'
    if (!model.value) model.value = imageModelDefault.value
    // 没配默认生图密钥时，自动切到"用自己的密钥"
    if (!data.has_image_key) keyMode.value = 'own'
  } catch {
    if (!model.value) model.value = imageModelDefault.value
  }
  window.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
})

function triggerFileInput() {
  fileInputRef.value?.click()
}

function addRefFiles(files: FileList | File[]) {
  const newFiles = Array.from(files).filter(f => f.type.startsWith('image/'))
  const remaining = 2 - refFiles.value.length
  const toAdd = newFiles.slice(0, remaining)
  for (const f of toAdd) {
    refFiles.value.push(f)
    const reader = new FileReader()
    reader.onload = (e) => {
      refPreviews.value.push((e.target?.result as string) || '')
    }
    reader.readAsDataURL(f)
  }
}

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files) addRefFiles(input.files)
  input.value = '' // 重置以允许重复选择同一文件
}

function removeRef(idx: number) {
  refFiles.value.splice(idx, 1)
  refPreviews.value.splice(idx, 1)
}

function clearAllRefs() {
  refFiles.value = []
  refPreviews.value = []
}

/**
 * 把网关返回的 image 统一转成可显示的 src：
 * 优先 url，其次 b64_json 拼成 data URI。
 */
function toSrc(img: AIStudioImage): string {
  if (img.url) return img.url
  if (img.b64_json) return `data:image/png;base64,${img.b64_json}`
  return ''
}

// 从后端错误响应里取友好文案
function resolveError(err: unknown): string {
  const anyErr = err as { response?: { data?: { error?: string; message?: string } }; message?: string }
  return (
    anyErr?.response?.data?.error ||
    anyErr?.response?.data?.message ||
    anyErr?.message ||
    t('common.error')
  )
}

async function generate() {
  if (!canGenerate.value) return
  loading.value = true
  images.value = []
  // 用自己的密钥时把 key 传给后端；否则留空 → 后端用系统默认密钥并计入每日限额
  const userKey = keyMode.value === 'own' ? selectedKey.value : ''
  try {
    let result: AIStudioImage[]
    if (mode.value === 'edit' && refFiles.value.length > 0) {
      // 图生图模式：使用参考图
      result = await editImageViaStudio({
        prompt: prompt.value.trim(),
        images: refFiles.value,
        model: model.value,
        n: count.value,
        size: size.value,
        quality: quality.value,
        userKey
      })
    } else if (refFiles.value.length > 0) {
      // 文生图模式但有参考图：也走 edit 接口
      result = await editImageViaStudio({
        prompt: prompt.value.trim(),
        images: refFiles.value,
        model: model.value,
        n: count.value,
        size: size.value,
        quality: quality.value,
        userKey
      })
    } else {
      // 纯文生图
      result = await generateImageViaStudio({
        prompt: prompt.value.trim(),
        model: model.value,
        n: count.value,
        size: size.value,
        quality: quality.value,
        userKey
      })
    }
    images.value = result.map(toSrc).filter(Boolean)
    if (images.value.length === 0) {
      appStore.showError(t('aiStudio.image.emptyResult'))
    }
    // 成功后刷新剩余次数（仅默认密钥模式会变化）
    if (keyMode.value === 'default' && !cfg.value?.unlimited) {
      try {
        cfg.value = await getAIStudioConfig()
      } catch {
        /* 忽略 */
      }
    }
  } catch (err: unknown) {
    appStore.showError(resolveError(err))
  } finally {
    loading.value = false
  }
}

async function download(src: string, index: number) {
  try {
    const a = document.createElement('a')
    const filename = `ai-image-${Date.now()}-${index + 1}.png`
    if (src.startsWith('data:')) {
      a.href = src
    } else {
      // 远程 url：先 fetch 成 blob，避免跨域直接下载被当成导航
      const resp = await fetch(src)
      const blob = await resp.blob()
      a.href = URL.createObjectURL(blob)
    }
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    if (!src.startsWith('data:')) {
      setTimeout(() => URL.revokeObjectURL(a.href), 1000)
    }
  } catch {
    // 下载失败兜底：新标签打开
    window.open(src, '_blank')
  }
}
</script>

<style scoped>
.lightbox-fade-enter-active,
.lightbox-fade-leave-active {
  transition: opacity 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}
.lightbox-fade-enter-from,
.lightbox-fade-leave-to {
  opacity: 0;
}
.lightbox-fade-enter-active img,
.lightbox-fade-leave-active img {
  transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}
.lightbox-fade-enter-from img,
.lightbox-fade-leave-to img {
  transform: scale(0.92);
}
</style>
