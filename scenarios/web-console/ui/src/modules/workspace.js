import { elements, state } from './state.js'
import {
  createTerminalTab,
  findTab,
  setActiveTab,
  destroyTerminalTab,
  renderTabs,
  applyTabAppearance,
  getActiveTab,
  refreshTabButton
} from './tab-manager.js'
import { appendEvent, showError } from './event-feed.js'
import { proxyToApi, buildWebSocketUrl, textDecoder, scheduleMicrotask, parseApiError } from './utils.js'
import {
  reconnectSession,
  startSession,
  updateSessionActions,
  setTabPhase,
  setTabSocketState
} from './session-service.js'
import { showToast } from './notifications.js'

const workspaceCallbacks = {
  queueSessionOverviewRefresh: null
}

const debugWorkspace = typeof window !== 'undefined' &&
  window.__WEB_CONSOLE_DEBUG__ &&
  window.__WEB_CONSOLE_DEBUG__.workspace === true

const workspaceAnomalyState = {
  toastShown: false
}

const IDLE_TIMEOUT_DEFAULT_MINUTES = 15
const IDLE_TIMEOUT_MAX_MINUTES = 1440

function setWorkspaceLoading(value) {
  if (!state.workspace) {
    state.workspace = {
      loading: Boolean(value),
      idleTimeoutSeconds: 0,
      updatingIdleTimeout: false,
      idleControlsInitialized: false
    }
    return
  }
  state.workspace.loading = Boolean(value)
  if (typeof state.workspace.idleTimeoutSeconds !== 'number') {
    state.workspace.idleTimeoutSeconds = 0
  }
  if (typeof state.workspace.updatingIdleTimeout !== 'boolean') {
    state.workspace.updatingIdleTimeout = false
  }
  if (typeof state.workspace.idleControlsInitialized !== 'boolean') {
    state.workspace.idleControlsInitialized = false
  }
}

function isWorkspaceLoading() {
  return Boolean(state.workspace && state.workspace.loading)
}

function ensureSessionPlaceholder(tab, sessionId) {
  if (!tab || !sessionId) return
  if (!tab.session || tab.session.id !== sessionId) {
    tab.session = { id: sessionId }
  }
  tab.wasDetached = true
  refreshTabButton(tab)
}

function updateIdleTimeoutControls() {
  const toggle = elements.idleTimeoutToggle
  const minutesInput = elements.idleTimeoutMinutes
  if (!toggle || !minutesInput) {
    return
  }
  const seconds = Number(state.workspace?.idleTimeoutSeconds || 0)
  const enabled = seconds > 0
  toggle.checked = enabled
  minutesInput.disabled = !enabled || state.workspace?.updatingIdleTimeout === true
  if (enabled) {
    const minutes = Math.max(1, Math.round(seconds / 60))
    minutesInput.value = String(minutes)
  } else {
    minutesInput.value = ''
  }
  toggle.disabled = state.workspace?.updatingIdleTimeout === true
}

function applyIdleTimeoutFromServer(seconds) {
  const normalized = Number.isFinite(seconds) && seconds > 0
    ? Math.min(Math.max(Math.round(seconds), 60), IDLE_TIMEOUT_MAX_MINUTES * 60)
    : 0
  state.workspace.idleTimeoutSeconds = normalized
  updateIdleTimeoutControls()
}

async function persistIdleTimeoutSeconds(seconds) {
  if (state.workspace.updatingIdleTimeout) {
    return
  }
  const previous = state.workspace.idleTimeoutSeconds || 0
  if (previous === seconds) {
    updateIdleTimeoutControls()
    return
  }
  state.workspace.updatingIdleTimeout = true
  state.workspace.idleTimeoutSeconds = seconds
  updateIdleTimeoutControls()
  try {
    const response = await proxyToApi('/api/v1/workspace', {
      method: 'PATCH',
      json: { idleTimeoutSeconds: seconds }
    })
    if (!response.ok) {
      const text = await response.text().catch(() => '')
      const { message } = parseApiError(text, 'Unable to update idle timeout')
      throw new Error(message)
    }
  } catch (error) {
    console.error('Failed to update idle timeout:', error)
    state.workspace.idleTimeoutSeconds = previous
    await showToast(
      error instanceof Error ? error.message : 'Unable to update idle timeout',
      'error',
      3600
    )
  } finally {
    state.workspace.updatingIdleTimeout = false
    updateIdleTimeoutControls()
  }
}

function coerceMinutesValue(value) {
  const parsed = Number.parseInt(String(value), 10)
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return null
  }
  return Math.min(Math.max(parsed, 1), IDLE_TIMEOUT_MAX_MINUTES)
}

function handleIdleToggleChange(event) {
  const checked = event.currentTarget?.checked === true
  const minutesInput = elements.idleTimeoutMinutes
  if (!checked) {
    persistIdleTimeoutSeconds(0)
    return
  }
  const currentMinutes = minutesInput?.value ? coerceMinutesValue(minutesInput.value) : null
  const minutes = currentMinutes ?? IDLE_TIMEOUT_DEFAULT_MINUTES
  if (minutesInput) {
    minutesInput.value = String(minutes)
  }
  persistIdleTimeoutSeconds(minutes * 60)
}

function handleIdleMinutesChange(event) {
  const toggle = elements.idleTimeoutToggle
  if (!toggle || !toggle.checked) {
    return
  }
  const minutes = coerceMinutesValue(event.currentTarget?.value)
  if (!minutes) {
    event.currentTarget.value = String(Math.max(1, Math.round((state.workspace.idleTimeoutSeconds || 60) / 60)))
    return
  }
  persistIdleTimeoutSeconds(minutes * 60)
}

function initializeIdleTimeoutControls() {
  if (state.workspace.idleControlsInitialized) {
    return
  }
  const toggle = elements.idleTimeoutToggle
  const minutesInput = elements.idleTimeoutMinutes
  if (toggle) {
    toggle.addEventListener('change', handleIdleToggleChange)
  }
  if (minutesInput) {
    minutesInput.addEventListener('change', handleIdleMinutesChange)
    minutesInput.addEventListener('blur', handleIdleMinutesChange)
  }
  state.workspace.idleControlsInitialized = true
  updateIdleTimeoutControls()
}

export function configureWorkspace(options = {}) {
  workspaceCallbacks.queueSessionOverviewRefresh = typeof options.queueSessionOverviewRefresh === 'function'
    ? options.queueSessionOverviewRefresh
    : null
}

export function initializeWorkspaceSettingsUI() {
  initializeIdleTimeoutControls()
  updateIdleTimeoutControls()
}

function queueSessionOverviewRefresh(delay = 0) {
  workspaceCallbacks.queueSessionOverviewRefresh?.(delay)
}

function sanitizeWorkspaceTabsFromServer(rawTabs) {
  const duplicates = []
  const invalid = []
  const seen = new Set()
  const sanitized = []

  rawTabs.forEach((entry) => {
    if (!entry || typeof entry !== 'object') {
      return
    }
    const id = typeof entry.id === 'string' ? entry.id.trim() : ''
    if (!id) {
      invalid.push(entry.id || '(missing)')
      return
    }
    if (seen.has(id)) {
      duplicates.push(id)
      return
    }
    seen.add(id)
    sanitized.push({
      id,
      label: entry.label || `Terminal ${sanitized.length + 1}`,
      colorId: entry.colorId || 'sky',
      order: sanitized.length,
      sessionId: entry.sessionId || null
    })
  })

  return {
    tabs: sanitized,
    duplicateIds: duplicates,
    invalidIds: invalid
  }
}

function pruneDuplicateLocalTabs() {
  const seen = new Set()
  const filtered = []
  const removedIds = []

  state.tabs.slice().forEach((tab) => {
    if (!tab || typeof tab !== 'object' || typeof tab.id !== 'string') {
      return
    }
    if (seen.has(tab.id)) {
      removedIds.push(tab.id)
      destroyTerminalTab(tab)
      return
    }
    seen.add(tab.id)
    filtered.push(tab)
  })

  state.tabs = filtered
  if (removedIds.length > 0) {
    renderTabs()
  }
  return { tabs: filtered, removedIds }
}

function reportWorkspaceAnomaly({ duplicateIds = [], invalidIds = [], localDuplicates = [] } = {}) {
  if (!duplicateIds.length && !invalidIds.length && !localDuplicates.length) {
    return
  }

  const parts = []
  if (duplicateIds.length) {
    parts.push(`duplicate tab IDs: ${duplicateIds.join(', ')}`)
  }
  if (invalidIds.length) {
    parts.push(`invalid entries: ${invalidIds.join(', ')}`)
  }
  if (localDuplicates.length) {
    parts.push(`pruned local duplicates: ${localDuplicates.join(', ')}`)
  }

  const summary = parts.join('; ')

  if (debugWorkspace) {
    console.warn('Workspace anomalies detected and repaired:', summary)
  }

  const activeTab = getActiveTab()
  if (activeTab) {
    appendEvent(activeTab, 'workspace-sanitized', {
      summary,
      duplicateIds,
      invalidIds,
      localDuplicates
    })
  }

  if (!workspaceAnomalyState.toastShown) {
    workspaceAnomalyState.toastShown = true
    const maybePromise = showToast('Workspace layout was repaired to remove duplicate tabs.', 'warning', 4200)
    if (maybePromise && typeof maybePromise.catch === 'function') {
      maybePromise.catch(() => {})
    }
  }
}

export async function initializeWorkspace() {
  setWorkspaceLoading(true)
  const pendingReconnections = []
  try {
    const response = await proxyToApi('/api/v1/workspace')
    if (!response.ok) {
      throw new Error(`Failed to load workspace: ${response.status}`)
    }
    const workspace = await response.json()

    applyIdleTimeoutFromServer(
      typeof workspace.idleTimeoutSeconds === 'number' ? workspace.idleTimeoutSeconds : 0
    )

    const rawTabs = Array.isArray(workspace.tabs) ? workspace.tabs : []
    const { tabs: sanitizedTabs, duplicateIds, invalidIds } = sanitizeWorkspaceTabsFromServer(rawTabs)
    const { removedIds: localDuplicates } = pruneDuplicateLocalTabs()

    if (duplicateIds.length > 0 || invalidIds.length > 0 || localDuplicates.length > 0) {
      reportWorkspaceAnomaly({ duplicateIds, invalidIds, localDuplicates })
    }

    if (sanitizedTabs.length > 0) {
      sanitizedTabs.forEach((tabMeta) => {
        const existing = findTab(tabMeta.id)
        if (existing) {
          existing.label = tabMeta.label || existing.label
          existing.colorId = tabMeta.colorId || existing.colorId
          applyTabAppearance(existing)
          refreshTabButton(existing)
          const sessionId = typeof tabMeta.sessionId === 'string' ? tabMeta.sessionId.trim() : ''
          if (sessionId) {
            ensureSessionPlaceholder(existing, sessionId)
            const reconnectPromise = reconnectSession(existing, sessionId).catch(() => {
              if (debugWorkspace) {
                console.log(`Session ${sessionId} no longer available for tab ${tabMeta.id}`)
              }
              existing.session = null
              refreshTabButton(existing)
            })
            pendingReconnections.push(reconnectPromise)
          }
          return
        }

        const tab = createTerminalTab({
          focus: false,
          id: tabMeta.id,
          label: tabMeta.label,
          colorId: tabMeta.colorId
        })
        const sessionId = typeof tabMeta.sessionId === 'string' ? tabMeta.sessionId.trim() : ''
        if (tab && sessionId) {
          ensureSessionPlaceholder(tab, sessionId)
          const reconnectPromise = reconnectSession(tab, sessionId).catch(() => {
            if (debugWorkspace) {
              console.log(`Session ${sessionId} no longer available for tab ${tabMeta.id}`)
            }
            tab.session = null
            refreshTabButton(tab)
          })
          pendingReconnections.push(reconnectPromise)
        }
      })

      renderTabs()

      if (pendingReconnections.length > 0) {
        await Promise.allSettled(pendingReconnections)
      }

      const requestedActiveId = typeof workspace.activeTabId === 'string' ? workspace.activeTabId.trim() : ''
      if (requestedActiveId && findTab(requestedActiveId)) {
        setActiveTab(requestedActiveId)
      } else if (sanitizedTabs[0]) {
        setActiveTab(sanitizedTabs[0].id)
      }
    } else {
      const initialTab = createTerminalTab({ focus: true })
      if (initialTab) {
        await syncTabToWorkspace(initialTab)
        startSession(initialTab, { reason: 'initial-tab' }).catch((error) => {
          appendEvent(initialTab, 'session-error', error)
          showError(initialTab, error instanceof Error ? error.message : 'Unable to start terminal session')
        })
      }
    }
  } catch (error) {
    console.error('Failed to initialize workspace:', error)
    applyIdleTimeoutFromServer(0)
    const initialTab = createTerminalTab({ focus: true })
    if (initialTab) {
      await syncTabToWorkspace(initialTab)
      startSession(initialTab, { reason: 'initial-tab' }).catch((startError) => {
        appendEvent(initialTab, 'session-error', startError)
        showError(initialTab, startError instanceof Error ? startError.message : 'Unable to start terminal session')
      })
    }
  } finally {
    setWorkspaceLoading(false)
    if (state.tabs.some((tab) => tab && tab.session && tab.session.id)) {
      scheduleMicrotask(() => {
        if (!isWorkspaceLoading()) {
          syncActiveTabState()
        }
      })
    }
    connectWorkspaceWebSocket()
  }
}

export async function syncTabToWorkspace(tab) {
  try {
    await proxyToApi('/api/v1/workspace/tabs', {
      method: 'POST',
      json: {
        id: tab.id,
        label: tab.label,
        colorId: tab.colorId,
        order: state.tabs.indexOf(tab),
        sessionId: tab.session && tab.session.id ? tab.session.id : null
      }
    })
  } catch (error) {
    console.error('Failed to sync tab to workspace:', error)
  }
}

export async function deleteTabFromWorkspace(tabId) {
  try {
    await proxyToApi(`/api/v1/workspace/tabs/${tabId}`, {
      method: 'DELETE'
    })
  } catch (error) {
    console.error('Failed to delete tab from server:', error)
  }
}

export function syncActiveTabState() {
  if (isWorkspaceLoading()) {
    return
  }
  const { tabs: uniqueTabs, removedIds } = pruneDuplicateLocalTabs()
  if (removedIds.length > 0) {
    reportWorkspaceAnomaly({ localDuplicates: removedIds })
  }

  let activeId = state.activeTabId
  if (activeId && !uniqueTabs.some((tab) => tab.id === activeId)) {
    activeId = uniqueTabs.length > 0 ? uniqueTabs[0].id : null
    state.activeTabId = activeId
  }

  proxyToApi('/api/v1/workspace', {
    method: 'PUT',
    json: {
      activeTabId: activeId,
      tabs: uniqueTabs.map((t, idx) => ({
        id: t.id,
        label: t.label,
        colorId: t.colorId,
        order: idx,
        sessionId: t.session && t.session.id ? t.session.id : null
      }))
    }
  }).catch((error) => {
    console.error('Failed to sync active tab to server:', error)
  })
}

export function connectWorkspaceWebSocket() {
  if (state.workspaceSocket) {
    const { readyState } = state.workspaceSocket
    if (readyState === WebSocket.OPEN || readyState === WebSocket.CONNECTING) {
      return
    }
    try {
      state.workspaceSocket.close()
    } catch (_error) {
      // ignore close failures on stale sockets
    }
  }

  const url = buildWebSocketUrl('/ws/workspace/stream')
  const socket = new WebSocket(url)
  state.workspaceSocket = socket

  socket.addEventListener('open', () => {
    if (state.workspaceSocket !== socket) {
      return
    }
    if (debugWorkspace) {
      console.log('Workspace WebSocket connected')
    }
    if (state.workspaceReconnectTimer) {
      clearTimeout(state.workspaceReconnectTimer)
      state.workspaceReconnectTimer = null
    }
  })

  socket.addEventListener('message', async (event) => {
    if (state.workspaceSocket !== socket) {
      return
    }
    try {
      let rawData = event.data
      if (rawData instanceof Blob) {
        rawData = await rawData.text()
      } else if (rawData instanceof ArrayBuffer) {
        rawData = textDecoder.decode(rawData)
      }

      const data = JSON.parse(rawData)

      if (data.type === 'status' && data.payload?.status === 'upstream_connected') {
        return
      }

      if (data.activeTabId !== undefined && data.tabs !== undefined) {
        if (debugWorkspace) {
          console.log('Received initial workspace state')
        }
        return
      }

      handleWorkspaceEvent(data)
    } catch (error) {
      console.error('Failed to parse workspace event:', error)
    }
  })

  socket.addEventListener('close', () => {
    if (state.workspaceSocket !== socket) {
      return
    }
    if (debugWorkspace) {
      console.log('Workspace WebSocket closed, reconnecting...')
    }
    state.workspaceSocket = null
    if (!state.workspaceReconnectTimer) {
      state.workspaceReconnectTimer = setTimeout(() => {
        connectWorkspaceWebSocket()
      }, 3000)
    }
  })

  socket.addEventListener('error', (error) => {
    if (state.workspaceSocket !== socket) {
      return
    }
    console.error('Workspace WebSocket error:', error)
  })
}

function handleWorkspaceEvent(event) {
  if (!event || !event.type) return

  switch (event.type) {
    case 'workspace-full-update':
      if (debugWorkspace) {
        console.log('Full workspace update:', event.payload)
      }
      break
    case 'tab-added':
      handleTabAdded(event.payload)
      break
    case 'tab-updated':
      handleTabUpdated(event.payload)
      break
    case 'tab-removed':
      handleTabRemoved(event.payload)
      break
    case 'active-tab-changed':
      handleActiveTabChanged(event.payload)
      break
    case 'session-attached':
      handleSessionAttached(event.payload)
      break
    case 'session-detached':
      handleSessionDetached(event.payload)
      break
    case 'idle-timeout-changed':
      if (event.payload && typeof event.payload.idleTimeoutSeconds === 'number') {
        applyIdleTimeoutFromServer(event.payload.idleTimeoutSeconds)
      }
      break
    case 'keyboard-toolbar-mode-changed':
      if (debugWorkspace) {
        console.log('Keyboard toolbar mode changed event (deprecated):', event.payload)
      }
      break
    default:
      if (debugWorkspace) {
        console.log('Unknown workspace event:', event.type)
      }
  }
}

function handleTabAdded(payload) {
  const existing = findTab(payload.id)
  if (existing) return

  createTerminalTab({
    focus: false,
    id: payload.id,
    label: payload.label,
    colorId: payload.colorId
  })
}

function handleTabUpdated(payload) {
  const tab = findTab(payload.id)
  if (!tab) return

  tab.label = payload.label
  tab.colorId = payload.colorId
  applyTabAppearance(tab)
  renderTabs()
  if (tab.id === state.activeTabId) {
    updateSessionActions()
  }
}

function handleTabRemoved(payload) {
  const tab = findTab(payload.id)
  if (!tab) return

  destroyTerminalTab(tab)
  state.tabs = state.tabs.filter((entry) => entry.id !== payload.id)

  if (state.activeTabId === payload.id) {
    const fallback = state.tabs[state.tabs.length - 1] || state.tabs[0] || null
    state.activeTabId = fallback ? fallback.id : null
  }

  renderTabs()
  if (state.activeTabId) {
    const active = getActiveTab()
    if (active) {
      setActiveTab(active.id)
    }
  }
}

function handleActiveTabChanged(payload) {
  if (payload.id && payload.id !== state.activeTabId) {
    setActiveTab(payload.id)
  }
}

function handleSessionAttached(payload) {
  const tab = findTab(payload.tabId)
  if (!tab || !payload.sessionId) return

  if (!tab.session || tab.session.id !== payload.sessionId) {
    reconnectSession(tab, payload.sessionId)
  }
}

function handleSessionDetached(payload) {
  const tab = findTab(payload.tabId)
  if (!tab) return

  if (tab.session) {
    tab.session = null
    setTabPhase(tab, 'idle')
    setTabSocketState(tab, 'disconnected')
  }
  tab.wasDetached = true
  refreshTabButton(tab)
  queueSessionOverviewRefresh(250)
  updateSessionActions()
}

export async function getWorkspaceState() {
  try {
    const response = await proxyToApi('/api/v1/workspace')
    if (!response.ok) return null
    return await response.json()
  } catch (error) {
    console.error('Failed to get workspace state:', error)
    return null
  }
}
