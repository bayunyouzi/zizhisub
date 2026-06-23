<template>
  <div class="key-selector">
    <label v-if="label" class="input-label">{{ label }}</label>
    <div class="relative">
      <select
        :value="modelValue"
        :disabled="loading || keys.length === 0"
        class="input appearance-none pr-10"
        @change="onChange(($event.target as HTMLSelectElement).value)"
      >
        <option value="" disabled>
          {{ loading ? t('aiStudio.keySelector.loading') : keys.length === 0 ? t('aiStudio.keySelector.empty') : t('aiStudio.keySelector.placeholder') }}
        </option>
        <option v-for="k in keys" :key="k.id" :value="k.key">
          {{ k.name }} · {{ maskApiKey(k.key) }}
        </option>
      </select>
      <Icon
        name="chevronDown"
        size="sm"
        class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
      />
    </div>
    <p v-if="keys.length === 0 && !loading" class="input-hint">
      {{ t('aiStudio.keySelector.emptyHint') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import keysAPI from '@/api/keys'
import { maskApiKey } from '@/utils/maskApiKey'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { ApiKey } from '@/types'

const props = defineProps<{
  modelValue: string
  label?: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const keys = ref<ApiKey[]>([])
const loading = ref(false)

function onChange(value: string) {
  emit('update:modelValue', value)
}

/**
 * 拉取用户已创建的、可用（active）的密钥。
 * 一次性取较大 pageSize，绘图/查询场景密钥数量通常不多。
 */
async function loadKeys() {
  loading.value = true
  try {
    const resp = await keysAPI.list(1, 100, { status: 'active' })
    keys.value = resp.items ?? []
    // 自动选中第一个，省去用户一次点击
    if (!props.modelValue && keys.value.length > 0) {
      emit('update:modelValue', keys.value[0].key)
    }
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(loadKeys)

defineExpose({ reload: loadKeys })
</script>
