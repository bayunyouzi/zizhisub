<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl">
      <!-- Page Header -->
      <div class="mb-8">
        <div class="flex items-center gap-3">
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400">
            <Icon name="photo" size="lg" />
          </div>
          <div>
            <h1 class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">
              {{ t('aiStudio.image.title') }}
            </h1>
            <p class="mt-0.5 text-sm text-gray-500 dark:text-gray-400">
              {{ t('aiStudio.image.subtitle') }}
            </p>
          </div>
        </div>
      </div>

      <!-- Mode Tabs -->
      <div class="mb-6 inline-flex rounded-2xl bg-gray-100 p-1 dark:bg-dark-800">
        <button
          v-for="m in modes"
          :key="m.value"
          type="button"
          class="rounded-xl px-5 py-2 text-sm font-medium transition-all"
          :class="mode === m.value
            ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
            : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
          @click="mode = m.value"
        >
          {{ m.label }}
        </button>
      </div>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-5">
        <!-- Control Panel -->
        <div class="card card-body space-y-5 lg:col-span-2">
          <!-- 密钥来源：默认密钥(有每日限额) / 自己的密钥(不限) -->
          <div>
            <label class="input-label">{{ t('aiStudio.image.keyModeLabel') }}</label>
            <div class="inline-flex w-full rounded-2xl bg-gray-100 p-1 dark:bg-dark-800">
              <button
                type="button"
                class="flex-1 rounded-xl px-3 py-2 text-sm font-medium transition-all"
                :class="keyMode === 'default'
                  ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
                  : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
                @click="keyMode = 'default'"
              >
                {{ t('aiStudio.image.keyModeDefault') }}
              </button>
              <button
                type="button"
                class="flex-1 rounded-xl px-3 py-2 text-sm font-medium transition-all"
                :class="keyMode === 'own'
                  ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
                  : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'"
                @click="keyMode = 'own'"
              >
                {{ t('aiStudio.image.keyModeOwn') }}
              </button>
            </div>

            <!-- 默认密钥：剩余次数提示 -->
            <p v-if="keyMode === 'default'" class="mt-2 text-xs" :class="quotaHintClass">
              <template v-if="cfg?.unlimited">{{ t('aiStudio.image.quotaUnlimited') }}</template>
              <template v-else>
                {{ t('aiStudio.image.quotaRemaining', { remaining: remaining, limit: dailyLimit }) }}
              </template>
            </p>

            <!-- 自己的密钥：选择已创建的 key -->
            <div v-else class="mt-3">
              <KeySelector v-model="selectedKey" :label="t('aiStudio.keySelector.label')" />
            </div>
          </div>

          <div>
            <label class="input-label">{{ t('aiStudio.image.modelLabel') }}</label>
            <input
              v-model="model"
              type="text"
              :placeholder="t('aiStudio.image.modelPlaceholder')"
              class="input"
            />
            <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">{{ t('aiStudio.image.modelHint', { model: imageModelDefault }) }}</p>
          </div>

          <!-- 参考图（文生图和图生图都支持，最多2张） -->
          <div>
            <label class="input-label">
              {{ mode === 'edit' ? t('aiStudio.image.refImageLabel') : '参考图（可选，最多2张）' }}
            </label>
            <div
              class="flex flex-col items-center justify-center rounded-2xl border-2 border-dashed border-gray-200 px-4 py-6 transition-colors hover:border-primary-400 dark:border-dark-600 dark:hover:border-primary-500"
              :class="{ 'border-primary-400 dark:border-primary-500': dragging }"
              @dragover.prevent="dragging = true"
              @dragleave.prevent="dragging = false"
              @drop.prevent="onDrop"
            >
              <!-- 已选图片预览 -->
              <div v-if="refFiles.length > 0" class="w-full">
                <div class="flex flex-wrap gap-3 justify-center mb-3">
                  <div v-for="(f, idx) in refPreviews" :key="idx" class="relative">
                    <img :src="f" :alt="`ref-${idx}`" class="h-28 rounded-xl object-contain" />
                    <button
                      class="absolute -top-2 -right-2 rounded-full bg-red-500 text-white p-0.5 shadow-md hover:bg-red-600 transition-colors"
                      @click="removeRef(idx)"
                    >
                      <Icon name="x" size="sm" />
                    </button>
                  </div>
                </div>
                <div class="flex gap-2 justify-center">
                  <button v-if="refFiles.length < 2" class="btn btn-ghost btn-sm" @click="triggerFileInput">
                    <Icon name="plus" size="sm" /> 添加更多
                  </button>
                  <button class="btn btn-ghost btn-sm text-red-500" @click="clearAllRefs">
                    <Icon name="x" size="sm" /> 清空全部
                  </button>
                </div>
              </div>
              <!-- 空状态 -->
              <template v-else>
                <Icon name="photo" size="xl" class="mb-2 text-gray-300 dark:text-gray-600" />
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('aiStudio.image.dropHint') }}</p>
                <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">支持 JPG/PNG，最多 2 张</p>
                <label class="btn btn-secondary btn-sm mt-3 cursor-pointer">
                  {{ t('aiStudio.image.selectFile') }}
                  <input ref="fileInputRef" type="file" accept="image/*" class="hidden" multiple @change="onFileChange" />
                </label>
              </template>
            </div>
          </div>

          <!-- Prompt -->
          <div>
            <label class="input-label flex items-center justify-between">
              <span>{{ t('aiStudio.image.promptLabel') }}</span>
              <button
                type="button"
                class="inline-flex items-center gap-1 text-xs font-medium text-primary-500 transition-colors hover:text-primary-600"
                @click="showPromptHelper = !showPromptHelper"
              >
                <Icon name="sparkles" size="sm" />
                {{ showPromptHelper ? t('aiStudio.image.hidePromptHelper') : t('aiStudio.image.showPromptHelper') }}
              </button>
            </label>

            <!-- 内联 AI 提示词生成器 -->
            <Transition
              enter-active-class="transition-all duration-300 ease-out"
              enter-from-class="opacity-0 -translate-y-2"
              enter-to-class="opacity-100 translate-y-0"
              leave-active-class="transition-all duration-200 ease-in"
              leave-from-class="opacity-100 translate-y-0"
              leave-to-class="opacity-0 -translate-y-2"
            >
              <div
                v-if="showPromptHelper"
                class="mb-3 space-y-3 rounded-2xl border border-amber-100 bg-amber-50/60 p-4 dark:border-amber-900/30 dark:bg-amber-900/10"
              >
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('aiStudio.image.promptHelperHint', { model: promptModelDefault }) }}
                </p>

                <div>
                  <label class="input-label text-xs">{{ t('aiStudio.prompt.themeLabel') }}</label>
                  <textarea
                    v-model="theme"
                    rows="2"
                    :placeholder="t('aiStudio.prompt.themePlaceholder')"
                    class="input resize-none text-sm"
                  ></textarea>
                </div>

                <div>
                  <label class="input-label text-xs">{{ t('aiStudio.prompt.styleLabel') }}</label>
                  <div class="flex flex-wrap gap-1.5">
                    <button
                      v-for="s in stylePresets"
                      :key="s.value"
                      type="button"
                      class="rounded-full px-2.5 py-1 text-xs font-medium transition-all"
                      :class="selectedStyles.includes(s.value)
                        ? 'bg-primary-500 text-white shadow-sm shadow-primary-500/30'
                        : 'bg-white text-gray-600 hover:bg-gray-100 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'"
                      @click="toggleStyle(s.value)"
                    >
                      {{ s.label }}
                    </button>
                  </div>
                </div>

                <button
                  type="button"
                  class="btn btn-secondary btn-sm w-full"
                  :disabled="!theme.trim() || promptLoading"
                  @click="generatePrompt"
                >
                  <Icon name="sparkles" size="sm" :class="promptLoading ? 'animate-pulse' : ''" />
                  {{ promptLoading ? t('aiStudio.prompt.generating') : t('aiStudio.image.generateAndFill') }}
                </button>
              </div>
            </Transition>

            <textarea
              v-model="prompt"
              rows="5"
              :placeholder="t('aiStudio.image.promptPlaceholder')"
              class="input resize-none"
            ></textarea>
          </div>

          <!-- Size + Quality + Count -->
          <div class="space-y-3">
            <div>
              <label class="input-label">{{ t('aiStudio.image.sizeLabel') }}</label>
              <div class="grid grid-cols-2 gap-2">
                <button
                  v-for="opt in sizeOptions"
                  :key="opt.value"
                  type="button"
                  class="rounded-xl px-3 py-2 text-xs font-medium transition-all text-center"
                  :class="size === opt.value
                    ? 'bg-primary-500 text-white shadow-sm shadow-primary-500/30'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600'"
                  @click="size = opt.value"
                >
                  <div>{{ opt.label }}</div>
                  <div class="text-[10px] opacity-70">{{ opt.value }}</div>
                </button>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="input-label">质量</label>
                <div class="relative">
                  <select v-model="quality" class="input appearance-none pr-9">
                    <option v-for="q in qualityOptions" :key="q.value" :value="q.value">{{ q.label }}</option>
                  </select>
                  <Icon name="chevronDown" size="sm" class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-gray-400" />
                </div>
              </div>
              <div>
                <label class="input-label">{{ t('aiStudio.image.countLabel') }}</label>
                <div class="relative">
                  <select v-model.number="count" class="input appearance-none pr-9">
                    <option v-for="n in [1, 2, 3, 4]" :key="n" :value="n">{{ n }}</option>
                  </select>
                  <Icon name="chevronDown" size="sm" class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-gray-400" />
                </div>
              </div>
            </div>
          </div>

          <button
            class="btn btn-primary btn-lg w-full"
            :disabled="!canGenerate || loading"
            @click="generate"
          >
            <Icon name="sparkles" size="md" :class="loading ? 'animate-pulse' : ''" />
            {{ loading ? t('aiStudio.image.generating') : t('aiStudio.image.generate') }}
          </button>
        </div>

        <!-- Result Panel -->
        <div class="lg:col-span-3">
          <div class="card card-body min-h-[400px]">
            <!-- Loading skeleton -->
            <div v-if="loading" class="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div
                v-for="n in count"
                :key="n"
                class="aspect-square animate-pulse rounded-2xl bg-gray-100 dark:bg-dark-700"
              ></div>
            </div>

            <!-- Results -->
            <div v-else-if="images.length > 0" class="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div
                v-for="(img, i) in images"
                :key="i"
                class="group relative overflow-hidden rounded-2xl border border-gray-100 dark:border-dark-700"
              >
                <img
                  :src="img"
                  :alt="`result-${i}`"
                  class="aspect-square w-full cursor-zoom-in object-cover transition-transform duration-300 group-hover:scale-[1.02]"
                  @click="openLightbox(img)"
                />
                <div class="pointer-events-none absolute inset-0 flex items-end justify-end bg-gradient-to-t from-black/50 to-transparent p-3 opacity-0 transition-opacity group-hover:opacity-100">
                  <div class="pointer-events-auto flex gap-2">
                    <button
                      class="rounded-xl bg-white/90 p-2.5 text-gray-800 shadow-md backdrop-blur transition-transform hover:scale-105"
                      title="放大查看"
                      @click.stop="openLightbox(img)"
                    >
                      <Icon name="search" size="sm" />
                    </button>
                    <button
                      class="rounded-xl bg-white/90 p-2.5 text-gray-800 shadow-md backdrop-blur transition-transform hover:scale-105"
                      :title="t('common.download')"
                      @click.stop="download(img, i)"
                    >
                      <Icon name="download" size="sm" />
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Empty -->
            <div v-else class="flex h-full min-h-[360px] flex-col items-center justify-center text-center">
              <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-purple-50 text-purple-400 dark:bg-purple-900/20 dark:text-purple-500">
                <Icon name="photo" size="xl" />
              </div>
              <p class="text-gray-500 dark:text-gray-400">{{ t('aiStudio.image.emptyHint') }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Lightbox 图片放大查看 -->
    <Teleport to="body">
      <Transition name="lightbox-fade">
        <div
          v-if="lightboxSrc"
          class="fixed inset-0 z-[9999] flex items-center justify-center bg-black/80 backdrop-blur-md p-4"
          @click="closeLightbox"
        >
          <button
            class="absolute right-5 top-5 rounded-full bg-white/10 p-3 text-white backdrop-blur transition-all hover:bg-white/20 hover:scale-110"
            title="关闭"
            @click.stop="closeLightbox"
          >
            <Icon name="x" size="md" />
          </button>
          <img
            :src="lightboxSrc"
            alt="preview"
            class="max-h-[90vh] max-w-[90vw] rounded-2xl object-contain shadow-2xl"
            @click.stop
          />
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
const dragging = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

// gpt-image-2 实际支持的尺寸（DALL-E 3 兼容）
// 上游 API 不支持 4K/自定义分辨率，只接受标准尺寸
const sizeOptions = [
  { value: '1024x1024', label: '1:1 正方形' },
  { value: '1792x1024', label: '16:9 横屏' },
  { value: '1024x1792', label: '9:16 竖屏' },
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

function onDrop(e: DragEvent) {
  dragging.value = false
  if (e.dataTransfer?.files) addRefFiles(e.dataTransfer.files)
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
