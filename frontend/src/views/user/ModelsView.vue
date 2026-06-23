<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl">
      <!-- Page Header -->
      <div class="mb-8">
        <div class="flex items-center gap-3">
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary-100 text-primary-600 dark:bg-primary-900/30 dark:text-primary-400">
            <Icon name="cube" size="lg" />
          </div>
          <div>
            <h1 class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">
              {{ t('aiStudio.models.title') }}
            </h1>
            <p class="mt-0.5 text-sm text-gray-500 dark:text-gray-400">
              {{ t('aiStudio.models.subtitle') }}
            </p>
          </div>
        </div>
      </div>

      <!-- Control Card -->
      <div class="card card-body mb-6">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-end">
          <div class="flex-1">
            <KeySelector v-model="selectedKey" :label="t('aiStudio.keySelector.label')" />
          </div>
          <div class="flex-shrink-0">
            <button
              class="btn btn-primary btn-lg w-full lg:w-auto"
              :disabled="!selectedKey || loading"
              @click="fetchModels"
            >
              <Icon name="search" size="md" :class="loading ? 'animate-spin' : ''" />
              {{ loading ? t('aiStudio.models.querying') : t('aiStudio.models.query') }}
            </button>
          </div>
        </div>
      </div>

      <!-- Results -->
      <div v-if="models.length > 0">
        <!-- Search + count bar -->
        <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="relative w-full sm:w-80">
            <Icon
              name="search"
              size="md"
              class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
            />
            <input
              v-model="filter"
              type="text"
              :placeholder="t('aiStudio.models.filterPlaceholder')"
              class="input pl-10"
            />
          </div>
          <span class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('aiStudio.models.count', { shown: filteredModels.length, total: models.length }) }}
          </span>
        </div>

        <!-- Model grid -->
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="m in filteredModels"
            :key="m.id"
            class="card card-hover group cursor-pointer p-5"
            @click="copyId(m.id)"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0 flex-1">
                <h3 class="truncate font-medium text-gray-900 dark:text-white" :title="m.id">
                  {{ m.id }}
                </h3>
                <p v-if="m.owned_by" class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ m.owned_by }}
                </p>
              </div>
              <Icon
                :name="copiedId === m.id ? 'check' : 'copy'"
                size="sm"
                :class="copiedId === m.id ? 'text-emerald-500 flex-shrink-0' : 'text-gray-300 group-hover:text-primary-500 dark:text-gray-600 flex-shrink-0 transition-colors'"
              />
            </div>
            <div v-if="m.created" class="mt-3 flex items-center gap-1.5 text-xs text-gray-400 dark:text-gray-500">
              <Icon name="clock" size="xs" />
              {{ formatCreated(m.created) }}
            </div>
          </div>
        </div>
      </div>

      <!-- Empty state -->
      <div
        v-else-if="!loading && hasQueried"
        class="card flex flex-col items-center justify-center py-16 text-center"
      >
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gray-100 text-gray-400 dark:bg-dark-700 dark:text-gray-500">
          <Icon name="cube" size="xl" />
        </div>
        <p class="text-gray-500 dark:text-gray-400">{{ t('aiStudio.models.noResults') }}</p>
      </div>

      <!-- Initial hint -->
      <div
        v-else-if="!loading && !hasQueried"
        class="card flex flex-col items-center justify-center py-16 text-center"
      >
        <div class="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary-50 text-primary-400 dark:bg-primary-900/20 dark:text-primary-500">
          <Icon name="search" size="xl" />
        </div>
        <p class="text-gray-500 dark:text-gray-400">{{ t('aiStudio.models.initialHint') }}</p>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import KeySelector from '@/components/ai/KeySelector.vue'
import { listModels, type GatewayModel } from '@/api/gateway'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const selectedKey = ref('')
const models = ref<GatewayModel[]>([])
const loading = ref(false)
const hasQueried = ref(false)
const filter = ref('')
const copiedId = ref('')

// 网关地址：留空 → 自动走当前站点同源 /v1（你自己的 sub2api 网关）。
// 你在本面板创建的密钥本就是给自己这个网关用的；走同源零跨域，
// 不会再出现浏览器直连第三方网关被 CORS 拦截的 Failed to fetch。
const gatewayBase = computed(() => '')

const filteredModels = computed(() => {
  const q = filter.value.trim().toLowerCase()
  if (!q) return models.value
  return models.value.filter(
    (m) =>
      m.id.toLowerCase().includes(q) ||
      (m.owned_by || '').toLowerCase().includes(q)
  )
})

async function fetchModels() {
  if (!selectedKey.value) return
  loading.value = true
  hasQueried.value = true
  try {
    const list = await listModels({ apiKey: selectedKey.value, baseUrl: gatewayBase.value })
    // 按 id 字母序排，便于查找
    models.value = list.slice().sort((a, b) => a.id.localeCompare(b.id))
  } catch (err: unknown) {
    models.value = []
    appStore.showError(err instanceof Error ? err.message : t('common.error'))
  } finally {
    loading.value = false
  }
}

async function copyId(id: string) {
  await copyToClipboard(id, t('aiStudio.models.copied'))
  copiedId.value = id
  setTimeout(() => {
    if (copiedId.value === id) copiedId.value = ''
  }, 1500)
}

function formatCreated(ts: number): string {
  try {
    return new Date(ts * 1000).toLocaleDateString()
  } catch {
    return ''
  }
}
</script>
