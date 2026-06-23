<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <iframe v-if="isHomeContentUrl" :src="homeContent.trim()" class="h-screen w-full border-0" allowfullscreen></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Default Home Page - Apple Style -->
  <div v-else class="relative flex min-h-screen flex-col overflow-hidden">
    <!-- Ambient Background -->
    <div class="pointer-events-none fixed inset-0">
      <div class="absolute -right-1/4 -top-1/4 h-[600px] w-[600px] rounded-full bg-primary-400/10 blur-[120px]"></div>
      <div class="absolute -bottom-1/3 -left-1/4 h-[500px] w-[500px] rounded-full bg-violet-400/8 blur-[100px]"></div>
      <div class="absolute left-1/2 top-1/3 h-96 w-96 -translate-x-1/2 rounded-full bg-indigo-300/5 blur-[80px]"></div>
    </div>

    <!-- Header -->
    <header class="glass fixed left-0 right-0 top-0 z-20">
      <nav class="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 lg:px-8">
        <div class="flex items-center gap-3">
          <div class="h-8 w-8 overflow-hidden rounded-lg shadow-sm">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ siteName }}</span>
        </div>
        <div class="flex items-center gap-2">
          <a href="https://huabu.bayunzi.shop/" target="_blank" rel="noopener noreferrer" class="rounded-lg bg-gradient-to-r from-violet-500 to-purple-500 px-3 py-1.5 text-xs font-medium text-white transition-all hover:from-violet-600 hover:to-purple-600 hover:shadow-md">
            {{ t('home.infiniteCanvas') }}
          </a>
          <LocaleSwitcher />
          <button @click="toggleTheme" class="flex h-9 w-9 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-white/10" :title="isDark ? '浅色模式' : '深色模式'">
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link v-if="isAuthenticated" :to="dashboardPath" class="flex h-8 w-8 items-center justify-center rounded-full bg-indigo-500 text-xs font-bold text-white shadow-sm transition-transform hover:scale-105">
            {{ userInitial }}
          </router-link>
          <router-link v-else to="/login" class="rounded-lg bg-indigo-500 px-4 py-1.5 text-xs font-medium text-white transition-colors hover:bg-indigo-600">登录</router-link>
        </div>
      </nav>
    </header>

    <!-- Hero -->
    <section class="relative z-10 flex flex-col items-center justify-center px-4 pt-32 pb-20 text-center">
      <h1 class="mb-6 max-w-3xl text-4xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-5xl lg:text-6xl">
        {{ siteName }}
      </h1>
      <p class="mb-4 max-w-xl text-lg text-gray-500 dark:text-gray-400">
        {{ siteSubtitle }}
      </p>

      <!-- Feature pills -->
      <div class="mb-10 flex flex-wrap items-center justify-center gap-2">
        <span class="glass-card inline-flex items-center gap-1.5 rounded-full px-4 py-1.5 text-xs font-medium text-gray-600 dark:text-gray-300">
          <Icon name="swap" size="sm" class="text-primary-500" />
          {{ t('home.tags.subscriptionToApi') }}
        </span>
        <span class="glass-card inline-flex items-center gap-1.5 rounded-full px-4 py-1.5 text-xs font-medium text-gray-600 dark:text-gray-300">
          <Icon name="shield" size="sm" class="text-primary-500" />
          {{ t('home.tags.stickySession') }}
        </span>
        <span class="glass-card inline-flex items-center gap-1.5 rounded-full px-4 py-1.5 text-xs font-medium text-gray-600 dark:text-gray-300">
          <Icon name="chart" size="sm" class="text-primary-500" />
          {{ t('home.tags.realtimeBilling') }}
        </span>
      </div>

      <!-- CTA -->
      <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="btn btn-primary px-8 py-3 text-base shadow-lg shadow-primary-500/25">
        {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
        <Icon name="arrowRight" size="md" class="ml-2" :stroke-width="2" />
      </router-link>
    </section>

    <!-- Features Section -->
    <section class="relative z-10 mx-auto w-full max-w-6xl px-4 pb-16">
      <div class="grid gap-5 md:grid-cols-3">
        <div class="glass-card magnetic group p-8">
          <div class="mb-5 flex h-12 w-12 items-center justify-center rounded-2xl bg-indigo-500/10 text-indigo-500 transition-transform group-hover:scale-110">
            <Icon name="server" size="lg" />
          </div>
          <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">{{ t('home.features.unifiedGateway') }}</h3>
          <p class="text-sm leading-relaxed text-gray-500 dark:text-gray-400">{{ t('home.features.unifiedGatewayDesc') }}</p>
        </div>
        <div class="glass-card magnetic group p-8">
          <div class="mb-5 flex h-12 w-12 items-center justify-center rounded-2xl bg-violet-500/10 text-violet-500 transition-transform group-hover:scale-110">
            <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M18 18.72a9.094 9.094 0 003.741-.479 3 3 0 00-4.682-2.72m.94 3.198l.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0112 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 016 18.719m12 0a5.971 5.971 0 00-.941-3.197m0 0A5.995 5.995 0 0012 12.75a5.995 5.995 0 00-5.058 2.772m0 0a3 3 0 00-4.681 2.72 8.986 8.986 0 003.74.477m.94-3.197a5.971 5.971 0 00-.94 3.197M15 6.75a3 3 0 11-6 0 3 3 0 016 0zm6 3a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0zm-13.5 0a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z" /></svg>
          </div>
          <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">{{ t('home.features.multiAccount') }}</h3>
          <p class="text-sm leading-relaxed text-gray-500 dark:text-gray-400">{{ t('home.features.multiAccountDesc') }}</p>
        </div>
        <div class="glass-card magnetic group p-8">
          <div class="mb-5 flex h-12 w-12 items-center justify-center rounded-2xl bg-emerald-500/10 text-emerald-500 transition-transform group-hover:scale-110">
            <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z" /></svg>
          </div>
          <h3 class="mb-2 text-lg font-semibold text-gray-900 dark:text-white">{{ t('home.features.balanceQuota') }}</h3>
          <p class="text-sm leading-relaxed text-gray-500 dark:text-gray-400">{{ t('home.features.balanceQuotaDesc') }}</p>
        </div>
      </div>
    </section>

    <!-- Providers -->
    <section class="relative z-10 mx-auto w-full max-w-4xl px-4 pb-24">
      <h2 class="mb-3 text-center text-lg font-semibold text-gray-900 dark:text-white">{{ t('home.providers.title') }}</h2>
      <p class="mb-8 text-center text-sm text-gray-400">{{ t('home.providers.description') }}</p>
      <div class="flex flex-wrap items-center justify-center gap-3">
        <div class="glass-card flex items-center gap-2 rounded-xl px-4 py-2.5">
          <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-orange-400 to-orange-500 text-[10px] font-bold text-white">C</div>
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('home.providers.claude') }}</span>
          <span class="rounded bg-indigo-100 px-1.5 py-0.5 text-[10px] font-medium text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400">{{ t('home.providers.supported') }}</span>
        </div>
        <div class="glass-card flex items-center gap-2 rounded-xl px-4 py-2.5">
          <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-green-500 to-green-600 text-[10px] font-bold text-white">G</div>
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">GPT</span>
          <span class="rounded bg-indigo-100 px-1.5 py-0.5 text-[10px] font-medium text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400">{{ t('home.providers.supported') }}</span>
        </div>
        <div class="glass-card flex items-center gap-2 rounded-xl px-4 py-2.5">
          <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 to-blue-600 text-[10px] font-bold text-white">G</div>
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('home.providers.gemini') }}</span>
          <span class="rounded bg-indigo-100 px-1.5 py-0.5 text-[10px] font-medium text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400">{{ t('home.providers.supported') }}</span>
        </div>
        <div class="glass-card flex items-center gap-2 rounded-xl px-4 py-2.5">
          <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gradient-to-br from-rose-500 to-pink-600 text-[10px] font-bold text-white">A</div>
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('home.providers.antigravity') }}</span>
          <span class="rounded bg-indigo-100 px-1.5 py-0.5 text-[10px] font-medium text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-400">{{ t('home.providers.supported') }}</span>
        </div>
        <div class="glass-card flex items-center gap-2 rounded-xl px-4 py-2.5 opacity-50">
          <div class="flex h-7 w-7 items-center justify-center rounded-lg bg-gray-400 text-[10px] font-bold text-white">+</div>
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('home.providers.more') }}</span>
          <span class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-500 dark:bg-gray-800 dark:text-gray-400">{{ t('home.providers.soon') }}</span>
        </div>
      </div>
    </section>

    <!-- Footer -->
    <footer class="glass relative z-10 border-t border-gray-200/40 px-6 py-6 dark:border-white/5">
      <div class="mx-auto flex max-w-6xl flex-col items-center justify-center gap-3 text-center sm:flex-row sm:text-left">
        <p class="text-sm text-gray-400">&copy; {{ currentYear }} {{ siteName }}</p>
        <div class="flex items-center gap-4">
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="text-sm text-gray-400 transition-colors hover:text-gray-700 dark:hover:text-white">文档</a>
          <a :href="githubUrl" target="_blank" rel="noopener noreferrer" class="text-sm text-gray-400 transition-colors hover:text-gray-700 dark:hover:text-white">GitHub</a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = ref(document.documentElement.classList.contains('dark'))

// GitHub URL
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Toggle theme
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

// Initialize theme
function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>
