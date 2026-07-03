<script setup lang="ts">
import {
  AlertCircle,
  CheckCircle2,
  KeyRound,
  ListOrdered,
  LockKeyhole,
  Minus,
  Plus,
  RefreshCw,
  Search,
  Server,
  ShieldCheck,
  Trash2,
  Wifi,
  X
} from '@lucide/vue'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

interface SpamStatus {
  configured: boolean
  unlocked: boolean
}

interface SpamSection {
  token: string
  wid?: string
  user?: string
  contextid?: string
  verified: boolean
  status: string
  ready: boolean
  inSpam: boolean
  enabled: boolean
  priority?: number
  label?: string
}

const masterKey = ref(sessionStorage.getItem('quepasa.spam.masterkey') || '')
const status = ref<SpamStatus>({ configured: false, unlocked: false })
const sections = ref<SpamSection[]>([])
const searchResults = ref<SpamSection[]>([])
const search = ref('')
const searchOpen = ref(false)
const searchWrap = ref<HTMLElement | null>(null)
const loading = ref(true)
const validating = ref(false)
const searching = ref(false)
const savingToken = ref('')
const error = ref('')
const notice = ref('')
let noticeTimer: number | undefined

const unlocked = computed(() => status.value.configured && status.value.unlocked)
const activeSections = computed(() => sections.value.filter((item) => item.enabled).length)
const readySections = computed(() => sections.value.filter((item) => item.enabled && item.ready).length)
const priorityGroups = computed(() => new Set(sections.value.filter((item) => item.enabled).map(priorityOf)).size)
const queuedTokens = computed(() => new Set(sections.value.map((item) => item.token)))
const availableSearchResults = computed(() => searchResults.value.filter((item) => !item.inSpam && !queuedTokens.value.has(item.token)))
const ignoredSearchResults = computed(() => Math.max(0, searchResults.value.length - availableSearchResults.value.length))

onMounted(async () => {
  document.addEventListener('pointerdown', handleSearchOutside)
  await refreshStatus()
  if (status.value.configured && masterKey.value) {
    await unlock()
  }
  loading.value = false
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleSearchOutside)
  if (noticeTimer) window.clearTimeout(noticeTimer)
})

watch(notice, (value) => {
  if (noticeTimer) window.clearTimeout(noticeTimer)
  if (value) {
    noticeTimer = window.setTimeout(() => {
      notice.value = ''
    }, 3800)
  }
})

async function refreshStatus() {
  error.value = ''
  try {
    status.value = await fetchJson<SpamStatus>('/spam/status', { allowAnonymous: true })
  } catch (err) {
    error.value = errorMessage(err)
  }
}

async function unlock() {
  validating.value = true
  error.value = ''
  try {
    await loadSections()
    sessionStorage.setItem('quepasa.spam.masterkey', masterKey.value)
    status.value.unlocked = true
  } catch (err) {
    status.value.unlocked = false
    error.value = errorMessage(err)
  } finally {
    validating.value = false
  }
}

function lock() {
  masterKey.value = ''
  status.value.unlocked = false
  sections.value = []
  resetSearch()
  sessionStorage.removeItem('quepasa.spam.masterkey')
}

async function loadSections() {
  const response = await fetchJson<{ items: SpamSection[] }>('/spam/sections')
  sections.value = response.items ?? []
}

async function runSearch() {
  searching.value = true
  error.value = ''
  searchOpen.value = true
  try {
    const response = await fetchJson<{ items: SpamSection[] }>('/spam/sections/search', {
      method: 'POST',
      body: { search: search.value, limit: 100 }
    })
    searchResults.value = response.items ?? []
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    searching.value = false
  }
}

async function addSection(item: SpamSection) {
  savingToken.value = item.token
  error.value = ''
  notice.value = ''
  try {
    await fetchJson('/spam/sections', {
      method: 'POST',
      body: {
        token: item.token,
        enabled: true,
        priority: nextPriority(),
        label: item.label || item.wid || item.user || ''
      }
    })
    notice.value = 'Seção adicionada ao serviço de spam.'
    await loadSections()
    await runSearch()
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    savingToken.value = ''
  }
}

function resetSearch() {
  search.value = ''
  searchResults.value = []
  searchOpen.value = false
}

function handleSearchOutside(event: PointerEvent) {
  const target = event.target as Node | null
  if (!target || searchWrap.value?.contains(target)) return
  if (search.value || searchResults.value.length || searchOpen.value) resetSearch()
}

async function removeSection(item: SpamSection) {
  savingToken.value = item.token
  error.value = ''
  notice.value = ''
  try {
    await fetchJson(`/spam/sections?token=${encodeURIComponent(item.token)}`, { method: 'DELETE' })
    notice.value = 'Seção removida do serviço de spam.'
    await loadSections()
    if (searchResults.value.length) await runSearch()
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    savingToken.value = ''
  }
}

async function toggleSection(item: SpamSection) {
  savingToken.value = item.token
  error.value = ''
  notice.value = ''
  try {
    await fetchJson('/spam/sections', {
      method: 'PATCH',
      body: {
        token: item.token,
        enabled: !item.enabled,
        priority: priorityOf(item),
        label: item.label || ''
      }
    })
    notice.value = item.enabled ? 'Seção pausada.' : 'Seção ativada.'
    await loadSections()
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    savingToken.value = ''
  }
}

async function updatePriority(item: SpamSection, value: number) {
  const priority = Math.max(1, Math.trunc(value || 1))
  savingToken.value = item.token
  error.value = ''
  notice.value = ''
  try {
    await fetchJson('/spam/sections', {
      method: 'PATCH',
      body: {
        token: item.token,
        enabled: item.enabled,
        priority,
        label: item.label || ''
      }
    })
    notice.value = 'Prioridade atualizada.'
    await loadSections()
  } catch (err) {
    error.value = errorMessage(err)
  } finally {
    savingToken.value = ''
  }
}

function onPriorityChange(item: SpamSection, event: Event) {
  const target = event.target as HTMLInputElement | null
  updatePriority(item, Number(target?.value || priorityOf(item)))
}

async function refreshAll() {
  await refreshStatus()
  if (unlocked.value) {
    await loadSections()
    if (search.value || searchResults.value.length) await runSearch()
  }
}

function sectionTitle(item: SpamSection) {
  return item.wid || item.token
}

function sectionSubtitle(item: SpamSection) {
  return [item.user, item.contextid].filter(Boolean).join(' · ') || 'Sem proprietário registrado'
}

function queueSectionTitle(item: SpamSection) {
  const wid = item.wid?.replace(/@s\.whatsapp\.net$/i, '')
  if (wid) return wid
  if (item.label && item.label !== item.user) return item.label
  return 'Seção sem WhatsApp'
}

function ownerLabel(item: SpamSection) {
  return item.user || 'Sem usuário registrado'
}

function tokenPreview(token: string) {
  if (token.length <= 22) return token
  return `${token.slice(0, 8)}-${token.slice(9, 13)}...${token.slice(-6)}`
}

function statusLabel(item: SpamSection) {
  if (!item.enabled) return 'Pausada'
  if (item.ready) return 'Ready'

  const status = (item.status || '').trim().toLowerCase()
  if (!item.verified || status === 'unverified') return 'Não verificada'
  if (!status) return 'Sem estado'

  return item.status
}

function priorityOf(item: SpamSection) {
  return item.priority || 10
}

function nextPriority() {
  const highest = sections.value.reduce((value, item) => Math.max(value, priorityOf(item)), 0)
  return highest + 1
}

function apiPath(path: string) {
  const runtimeConfig = (window as unknown as { quepasa?: { apiBase?: string } }).quepasa
  const base = (runtimeConfig?.apiBase || '/api').replace(/\/+$/, '')
  return `${base}${path}`
}

async function fetchJson<T = unknown>(
  path: string,
  options: { method?: string; body?: unknown; allowAnonymous?: boolean } = {}
): Promise<T> {
  const headers: Record<string, string> = {}
  if (!options.allowAnonymous) {
    headers['X-QUEPASA-MASTERKEY'] = masterKey.value
  }
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }

  const response = await fetch(apiPath(path), {
    method: options.method || 'GET',
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  })

  const contentType = response.headers.get('content-type') || ''
  const payload = contentType.includes('application/json') ? await response.json() : await response.text()
  if (!response.ok) {
    throw new Error(typeof payload === 'string' ? payload : payload.result || payload.status || 'Falha na requisição')
  }
  return payload as T
}

function errorMessage(err: unknown) {
  return err instanceof Error ? err.message : 'Falha inesperada'
}
</script>

<template>
  <main class="spam-shell">
    <div class="toast-stack" aria-live="polite" aria-atomic="true">
      <div v-if="error" class="toast error" role="alert">
        <AlertCircle :size="18" />
        <span>{{ error }}</span>
        <button type="button" aria-label="Fechar aviso de erro" @click="error = ''">
          <X :size="16" />
        </button>
      </div>
      <div v-if="notice" class="toast ok" role="status">
        <CheckCircle2 :size="18" />
        <span>{{ notice }}</span>
        <button type="button" aria-label="Fechar aviso" @click="notice = ''">
          <X :size="16" />
        </button>
      </div>
    </div>

    <aside class="rail" aria-label="Estado do serviço">
      <div class="brand-lock">
        <div class="brand-mark"><ShieldCheck :size="26" /></div>
        <div>
          <strong>Spam Control</strong>
          <span>QuePasa master</span>
        </div>
      </div>

      <div class="rail-stat">
        <span>Seções na fila</span>
        <strong>{{ sections.length }}</strong>
      </div>
      <div class="rail-stat">
        <span>Ativas</span>
        <strong>{{ activeSections }}</strong>
      </div>
      <div class="rail-stat">
        <span>Prontas</span>
        <strong>{{ readySections }}</strong>
      </div>
      <div class="rail-stat">
        <span>Níveis</span>
        <strong>{{ priorityGroups }}</strong>
      </div>

      <button class="ghost-action" type="button" @click="refreshAll">
        <RefreshCw :size="18" />
        Atualizar
      </button>
      <button v-if="unlocked" class="ghost-action danger" type="button" @click="lock">
        <LockKeyhole :size="18" />
        Bloquear
      </button>
    </aside>

    <section class="workspace" aria-live="polite">
      <header class="topbar">
        <div>
          <p class="eyebrow">WHATSAPP</p>
          <h1>Serviço de spam</h1>
          <span>Defina quais seções o endpoint <code>/spam</code> pode usar e a prioridade de disparo.</span>
        </div>
        <div class="state-pill" :class="{ ok: unlocked, blocked: !status.configured }">
          <CheckCircle2 v-if="unlocked" :size="18" />
          <AlertCircle v-else :size="18" />
          {{ unlocked ? 'Master liberado' : status.configured ? 'Master key necessária' : 'Master key ausente' }}
        </div>
      </header>

      <div v-if="loading" class="empty-panel">
        <RefreshCw class="spin" :size="28" />
        <strong>Carregando serviço</strong>
      </div>

      <div v-else-if="!status.configured" class="empty-panel blocked-panel">
        <LockKeyhole :size="36" />
        <strong>Master key não configurada</strong>
        <span>Configure `MASTERKEY` no ambiente do QuePasa para habilitar este console.</span>
      </div>

      <form v-else-if="!unlocked" class="unlock-panel" @submit.prevent="unlock">
        <KeyRound :size="34" />
        <div>
          <h2>Acesso master</h2>
          <p>Informe a master key para gerenciar as seções autorizadas no `/spam`.</p>
        </div>
        <label>
          <span>Master key</span>
          <input v-model="masterKey" type="password" autocomplete="current-password" required />
        </label>
        <button class="primary-action" type="submit" :disabled="validating">
          <LockKeyhole :size="18" />
          {{ validating ? 'Validando' : 'Entrar' }}
        </button>
      </form>

      <template v-else>
          <section class="panel queue-panel">
            <div class="panel-heading">
              <div>
                <p class="eyebrow">PRIORIDADE</p>
                <h2>Fila do `/spam`</h2>
              </div>
              <ListOrdered :size="22" />
            </div>
            <p class="queue-help">
              Menor prioridade dispara primeiro. Se várias seções prontas tiverem a mesma prioridade, o `/spam` sorteia uma delas.
            </p>

            <div ref="searchWrap" class="queue-search-wrap">
              <form class="queue-search" @submit.prevent="runSearch">
                <label class="queue-search-field">
                  <Search :size="18" aria-hidden="true" />
                  <span class="sr-only">Pesquisar seção para adicionar ao spam</span>
                  <input
                    v-model="search"
                    type="search"
                    placeholder="Pesquisar token, telefone, usuário ou contexto"
                    autocomplete="off"
                    @focus="searchOpen = searchResults.length > 0"
                    @keydown.esc="resetSearch"
                  />
                </label>
                <button type="submit" :disabled="searching">
                  <Search :size="18" />
                  {{ searching ? 'Buscando' : 'Buscar' }}
                </button>
              </form>

              <div v-if="searchOpen" class="search-popover" role="listbox" aria-label="Resultados da pesquisa">
                <div v-if="ignoredSearchResults" class="search-meta">
                  {{ ignoredSearchResults }} resultado{{ ignoredSearchResults === 1 ? '' : 's' }} ignorado{{ ignoredSearchResults === 1 ? '' : 's' }} por já estar{{ ignoredSearchResults === 1 ? '' : 'em' }} na fila.
                </div>

                <article v-for="item in availableSearchResults" :key="item.token" class="section-card search-result" role="option">
                  <div class="status-dot" :class="{ ready: item.ready }" aria-hidden="true"></div>
                  <div class="section-copy">
                    <strong>{{ sectionTitle(item) }}</strong>
                    <span>{{ sectionSubtitle(item) }}</span>
                    <code>{{ item.token }}</code>
                  </div>
                  <button
                    class="icon-command add"
                    type="button"
                    :disabled="savingToken === item.token"
                    :aria-label="`Adicionar ${sectionTitle(item)}`"
                    @click="addSection(item)"
                  >
                    <Plus :size="20" />
                  </button>
                </article>

                <div v-if="!searching && !availableSearchResults.length" class="search-empty">
                  <Search :size="22" />
                  <strong>Nenhuma seção disponível para adicionar</strong>
                  <span v-if="ignoredSearchResults">Todos os resultados encontrados já estão na fila.</span>
                  <span v-else>Refine a busca por telefone, token, usuário ou contexto.</span>
                </div>
              </div>
            </div>

            <div class="queue-table" role="table" aria-label="Seções do serviço de spam">
              <div class="queue-row queue-head" role="row">
                <span>Prioridade</span>
                <span>Seção</span>
                <span>Status</span>
                <span>Ações</span>
              </div>

              <TransitionGroup name="queue-list" tag="div" class="queue-list" role="rowgroup">
                <article v-for="item in sections" :key="item.token" class="queue-row" role="row">
                  <div class="priority-cell">
                    <button
                      type="button"
                      :disabled="savingToken === item.token || priorityOf(item) <= 1"
                      :aria-label="`Reduzir prioridade de ${sectionTitle(item)}`"
                      @click.stop="updatePriority(item, priorityOf(item) - 1)"
                    >
                      <Minus :size="16" />
                    </button>
                    <label>
                      <span class="sr-only">Prioridade de {{ sectionTitle(item) }}</span>
                      <input
                        :value="priorityOf(item)"
                        type="number"
                        min="1"
                        inputmode="numeric"
                        @change="onPriorityChange(item, $event)"
                      />
                    </label>
                    <button
                      type="button"
                      :disabled="savingToken === item.token"
                      :aria-label="`Aumentar prioridade de ${sectionTitle(item)}`"
                      @click.stop="updatePriority(item, priorityOf(item) + 1)"
                    >
                      <Plus :size="16" />
                    </button>
                  </div>

                  <div class="section-copy queue-section">
                    <strong>{{ queueSectionTitle(item) }}</strong>
                    <span class="owner-line">{{ ownerLabel(item) }}</span>
                    <span v-if="item.contextid" class="context-line">
                      Contexto
                      <code>{{ item.contextid }}</code>
                    </span>
                    <code class="token-chip" :title="item.token">{{ tokenPreview(item.token) }}</code>
                  </div>

                  <div class="status-cell">
                    <span class="connection-state" :class="{ ready: item.ready, disabled: !item.enabled }">
                      <Wifi v-if="item.ready" :size="16" />
                      <Server v-else :size="16" />
                      {{ statusLabel(item) }}
                    </span>
                  </div>

                  <div class="row-actions">
                    <button class="soft-command" type="button" :disabled="savingToken === item.token" @click="toggleSection(item)">
                      {{ item.enabled ? 'Pausar' : 'Ativar' }}
                    </button>
                    <button
                      class="icon-command danger"
                      type="button"
                      :disabled="savingToken === item.token"
                      :aria-label="`Remover ${sectionTitle(item)}`"
                      @click="removeSection(item)"
                    >
                      <Trash2 :size="18" />
                    </button>
                  </div>
                </article>
              </TransitionGroup>

              <div v-if="!sections.length" class="hint-box wide">
                <ListOrdered :size="26" />
                <strong>Nenhuma seção configurada</strong>
                <span>Enquanto esta fila estiver vazia, o `/spam` usa o comportamento legado.</span>
              </div>
            </div>
          </section>
      </template>
    </section>
  </main>
</template>
