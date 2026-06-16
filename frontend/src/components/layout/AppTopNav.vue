<template>
  <header class="top-nav">
    <div class="top-nav-inner">
      <!-- Logo -->
      <div class="top-nav-brand">
        <div class="top-nav-logo">
          <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
        </div>
        <span class="top-nav-title">{{ siteName }}</span>
      </div>

      <!-- Nav items -->
      <nav class="top-nav-items" ref="navContainer">
        <template v-for="item in navItems" :key="item.path || item.label">
          <!-- Dropdown group -->
          <div v-if="item.children?.length" class="top-nav-dropdown" :class="{ 'top-nav-dropdown-open': openDropdown === item }">
              <button
              class="top-nav-link"
              :class="{ 'top-nav-link-active': isGroupActive(item) }"
              @click.stop="toggleDropdown(item)"
            >
              <span>{{ item.label }}</span>
              <svg class="top-nav-chevron" :class="{ 'rotate-180': openDropdown === item }" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5" /></svg>
            </button>
            <div v-if="openDropdown === item" class="top-nav-dropdown-menu glass-card" @mouseleave="closeDropdown">
              <router-link
                v-for="child in item.children"
                :key="child.path"
                :to="child.path"
                class="top-nav-dropdown-item"
                :class="{ 'top-nav-dropdown-item-active': route.path === child.path }"
                @click="closeDropdown"
              >
                <component :is="child.icon" class="h-4 w-4" />
                <span>{{ child.label }}</span>
              </router-link>
            </div>
          </div>
          <!-- Single link -->
          <router-link
            v-else
            :to="item.path"
            class="top-nav-link"
            :class="{ 'top-nav-link-active': isActive(item.path) }"
          >
            <span>{{ item.label }}</span>
          </router-link>
        </template>
      </nav>

      <!-- Right actions -->
      <div class="top-nav-actions">
        <LocaleSwitcher />
        <button @click="toggleTheme" class="top-nav-icon-btn" :title="isDark ? '浅色模式' : '深色模式'">
          <svg v-if="isDark" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" class="h-5 w-5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 3v2.25m6.364.386l-1.591 1.591M21 12h-2.25m-.386 6.364l-1.591-1.591M12 18.75V21m-4.773-4.227l-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0z" /></svg>
          <svg v-else fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" class="h-5 w-5"><path stroke-linecap="round" stroke-linejoin="round" d="M21.752 15.002A9.718 9.718 0 0118 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 003 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 009.002-5.998z" /></svg>
        </button>

        <!-- User menu -->
        <div v-if="isAuthenticated" class="top-nav-dropdown" :class="{ 'top-nav-dropdown-open': openUserMenu }">
          <button class="top-nav-user-btn" @click="openUserMenu = !openUserMenu">
            <span class="top-nav-avatar">{{ userInitial }}</span>
          </button>
          <div v-if="openUserMenu" class="top-nav-dropdown-menu glass-card top-nav-dropdown-right" @mouseleave="openUserMenu = false">
            <div class="top-nav-dropdown-header">
              <span class="top-nav-dropdown-name">{{ authStore.user?.email }}</span>
              <span class="top-nav-dropdown-role">{{ isAdmin ? '管理员' : '用户' }}</span>
            </div>
            <template v-for="item in userMenuItems" :key="item.path">
              <router-link :to="item.path" class="top-nav-dropdown-item" @click="openUserMenu = false">
                <component :is="item.icon" class="h-4 w-4" />
                <span>{{ item.label }}</span>
              </router-link>
            </template>
            <button class="top-nav-dropdown-item top-nav-dropdown-item-danger" @click="handleLogout">
              <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5" class="h-4 w-4"><path stroke-linecap="round" stroke-linejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15m3 0l3-3m0 0l-3-3m3 3H9" /></svg>
              <span>退出登录</span>
            </button>
          </div>
        </div>
        <router-link v-else to="/login" class="top-nav-login-btn">登录</router-link>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, h, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()

const isDark = ref(document.documentElement.classList.contains('dark'))

// SVG Icons (minimal set for top nav)
const CogIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M9.594 3.94c.09-.542.56-.94 1.11-.94h2.593c.55 0 1.02.398 1.11.94l.213 1.281c.063.374.313.686.645.87.074.04.147.083.22.127.324.196.72.257 1.075.124l1.217-.456a1.125 1.125 0 011.37.49l1.296 2.247a1.125 1.125 0 01-.26 1.431l-1.003.827c-.293.24-.438.613-.431.992a6.759 6.759 0 010 .255c-.007.378.138.75.43.99l1.005.828c.424.35.534.954.26 1.43l-1.298 2.247a1.125 1.125 0 01-1.369.491l-1.217-.456c-.355-.133-.75-.072-1.076.124a6.57 6.57 0 01-.22.128c-.331.183-.581.495-.644.869l-.213 1.28c-.09.543-.56.941-1.11.941h-2.594c-.55 0-1.02-.398-1.11-.94l-.213-1.281c-.062-.374-.312-.686-.644-.87a6.52 6.52 0 01-.22-.127c-.325-.196-.72-.257-1.076-.124l-1.217.456a1.125 1.125 0 01-1.369-.49l-1.297-2.247a1.125 1.125 0 01.26-1.431l1.004-.827c.292-.24.437-.613.43-.992a6.932 6.932 0 010-.255c.007-.378-.138-.75-.43-.99l-1.004-.828a1.125 1.125 0 01-.26-1.43l1.297-2.247a1.125 1.125 0 011.37-.491l1.216.456c.356.133.751.072 1.076-.124.072-.044.146-.087.22-.128.332-.183.582-.495.644-.869l.214-1.281z' }), h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M15 12a3 3 0 11-6 0 3 3 0 016 0z' })]) }
const UserIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M15.75 6a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0zM4.501 20.118a7.5 7.5 0 0114.998 0A17.933 17.933 0 0112 21.75c-2.676 0-5.216-.584-7.499-1.632z' })]) }
const KeyIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', d: 'M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.563.43L10.5 17.25H8.25v2.25H6v2.25H2.25v-2.818c0-.597.237-1.17.659-1.591l6.499-6.499c.404-.404.527-1 .43-1.563A6 6 0 1121.75 8.25z' })]) }

const isAdmin = computed(() => authStore.isAdmin)
const isAuthenticated = computed(() => authStore.isAuthenticated)
const siteName = computed(() => appStore.siteName)
const siteLogo = computed(() => appStore.siteLogo)
const userInitial = computed(() => (authStore.user?.email || '?')[0].toUpperCase())

const openDropdown = ref<any>(null)
const openUserMenu = ref(false)
const navContainer = ref<HTMLElement | null>(null)

interface NavItem {
  path: string
  label: string
  icon?: any
  children?: { path: string; label: string; icon: any }[]
}

const navItems = computed((): NavItem[] => {
  if (isAdmin.value) {
    return [
      { path: '/admin/dashboard', label: '仪表盘' },
      {
        path: '/admin/users',
        label: '用户',
        children: [
          { path: '/admin/users', label: '用户管理', icon: UserIcon },
          { path: '/admin/groups', label: '用户分组', icon: UserIcon },
          { path: '/admin/subscriptions', label: '订阅管理', icon: KeyIcon },
        ],
      },
      {
        path: '/admin/channels',
        label: '渠道',
        children: [
          { path: '/admin/accounts', label: '账号管理', icon: CogIcon },
          { path: '/admin/channels', label: '渠道管理', icon: CogIcon },
          { path: '/admin/proxies', label: '代理管理', icon: CogIcon },
          { path: '/admin/usage', label: '用量统计', icon: UserIcon },
        ],
      },
      {
        path: '/admin/orders',
        label: '财务',
        children: [
          { path: '/admin/orders', label: '订单管理', icon: CogIcon },
          { path: '/admin/redeem', label: '兑换码', icon: KeyIcon },
          { path: '/admin/promo-codes', label: '优惠码', icon: KeyIcon },
        ],
      },
      {
        path: '/admin/settings',
        label: '系统',
        children: [
          { path: '/admin/settings', label: '系统设置', icon: CogIcon },
          { path: '/admin/announcements', label: '公告管理', icon: UserIcon },
          { path: '/admin/risk-control', label: '风控管理', icon: UserIcon },
        ],
      },
    ]
  }
  return [
    { path: '/dashboard', label: '仪表盘' },
    { path: '/keys', label: 'API 密钥' },
    { path: '/usage', label: '用量' },
    { path: '/profile', label: '个人中心' },
  ]
})

const userMenuItems = computed(() => {
  const items = [
    { path: '/profile', label: '个人资料', icon: UserIcon },
    { path: '/keys', label: 'API 密钥', icon: KeyIcon },
  ]
  if (isAdmin.value) {
    items.push({ path: '/admin/settings', label: '系统设置', icon: CogIcon })
  }
  return items
})

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function toggleDropdown(item: any) {
  openDropdown.value = openDropdown.value === item ? null : item
}

function closeDropdown() {
  openDropdown.value = null
}

function isActive(path: string): boolean {
  return route.path === path || route.path.startsWith(path + '/')
}

function isGroupActive(item: NavItem): boolean {
  return item.children?.some(c => isActive(c.path)) ?? false
}

async function handleLogout() {
  openUserMenu.value = false
  await authStore.logout()
  router.push('/login')
}
</script>

<style scoped>
.top-nav {
  @apply fixed left-0 right-0 top-0 z-30;
  @apply glass;
  @apply border-b border-gray-200/40 dark:border-white/5;
  @apply shadow-glass-sm;
  height: 3.5rem;
}

.top-nav-inner {
  @apply mx-auto flex h-14 max-w-screen-2xl items-center gap-4 px-4 lg:px-8;
}

.top-nav-brand {
  @apply flex items-center gap-2.5 flex-shrink-0;
  min-width: 0;
}

.top-nav-logo {
  @apply h-8 w-8 overflow-hidden rounded-lg flex-shrink-0;
}

.top-nav-title {
  @apply hidden text-sm font-semibold text-gray-900 dark:text-white sm:block;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.top-nav-items {
  @apply flex items-center gap-1 flex-1 overflow-x-auto;
  -webkit-overflow-scrolling: touch;
}

.top-nav-link {
  @apply flex items-center gap-1 rounded-lg px-3 py-1.5;
  @apply text-sm font-medium text-gray-600 dark:text-gray-400;
  @apply transition-all duration-200 cursor-pointer;
  @apply hover:bg-gray-100/70 dark:hover:bg-white/10;
  @apply hover:text-gray-900 dark:hover:text-white;
  white-space: nowrap;
  flex-shrink: 0;
  user-select: none;
}

.top-nav-link-active {
  @apply bg-gray-100/80 dark:bg-white/10;
  @apply text-gray-900 dark:text-white;
}

.top-nav-chevron {
  @apply h-3.5 w-3.5 opacity-50 transition-transform duration-200;
  flex-shrink: 0;
}

.top-nav-chevron.rotate-180 {
  transform: rotate(180deg);
}

.top-nav-dropdown {
  @apply relative;
}

.top-nav-dropdown-menu {
  @apply absolute left-0 top-full mt-1 min-w-[180px] py-1.5;
  @apply z-50;
  animation: slideDown 0.15s ease-out;
}

.top-nav-dropdown-right {
  @apply left-auto right-0;
}

.top-nav-dropdown-item {
  @apply flex items-center gap-2.5 px-4 py-2 text-sm;
  @apply text-gray-600 dark:text-gray-400;
  @apply transition-colors duration-150;
  @apply hover:bg-gray-100/70 dark:hover:bg-white/10;
  @apply hover:text-gray-900 dark:hover:text-white;
  width: 100%;
}

.top-nav-dropdown-item-active {
  @apply bg-indigo-50 dark:bg-indigo-900/20;
  @apply text-indigo-600 dark:text-indigo-400;
  @apply font-medium;
}

.top-nav-dropdown-item-danger {
  @apply text-red-500 hover:text-red-600 dark:text-red-400;
  @apply border-t border-gray-200/50 dark:border-white/10 pt-2 mt-1;
}

.top-nav-dropdown-header {
  @apply px-4 py-2 border-b border-gray-200/50 dark:border-white/10 mb-1;
}

.top-nav-dropdown-name {
  @apply block text-sm font-medium text-gray-900 dark:text-white truncate max-w-[200px];
}

.top-nav-dropdown-role {
  @apply text-xs text-gray-400 dark:text-gray-500;
}

.top-nav-actions {
  @apply flex items-center gap-2 flex-shrink-0;
}

.top-nav-icon-btn {
  @apply flex h-9 w-9 items-center justify-center rounded-lg;
  @apply text-gray-500 dark:text-gray-400;
  @apply transition-colors hover:bg-gray-100 dark:hover:bg-white/10;
}

.top-nav-user-btn {
  @apply flex items-center;
}

.top-nav-avatar {
  @apply flex h-8 w-8 items-center justify-center rounded-full;
  @apply bg-indigo-500 text-xs font-bold text-white;
  @apply transition-transform hover:scale-105;
}

.top-nav-login-btn {
  @apply rounded-lg bg-indigo-500 px-3 py-1.5 text-xs font-medium text-white;
  @apply transition-colors hover:bg-indigo-600;
}

@keyframes slideDown {
  from { opacity: 0; transform: translateY(-4px); }
  to { opacity: 1; transform: translateY(0); }
}

/* Mobile - hide nav items, show only brand + actions */
@media (max-width: 768px) {
  .top-nav-items {
    display: none;
  }
}
</style>
