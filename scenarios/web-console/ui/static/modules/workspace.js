import { state } from './state.js'
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
import { proxyToApi, buildWebSocketUrl, textDecoder } from './utils.js'
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

export function configureWorkspace(options = {}) {
  workspaceCallbacks.queueSessionOverviewRefresh = typeof options.queueSessionOverviewRefresh === 'function'
    ? options.queueSessionOverviewRefresh
    : null
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
  try {
    const response = await proxyToApi('/api/v1/workspace')
    if (!response.ok) {
      throw new Error(`Failed to load workspace: ${response.status}`)
    }
    const workspace = await response.json()

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
          if (tabMeta.sessionId && (!existing.session || existing.session.id !== tabMeta.sessionId)) {
            reconnectSession(existing, tabMeta.sessionId).catch(() => {
              if (debugWorkspace) {
                console.log(`Session ${tabMeta.sessionId} no longer available for tab ${tabMeta.id}`)
              }
            })
          }
          return
        }

        const tab = createTerminalTab({
          focus: false,
          id: tabMeta.id,
          label: tabMeta.label,
          colorId: tabMeta.colorId
        })
        if (tab && tabMeta.sessionId) {
          reconnectSession(tab, tabMeta.sessionId).catch(() => {
            if (debugWorkspace) {
              console.log(`Session ${tabMeta.sessionId} no longer available for tab ${tabMeta.id}`)
            }
          })
        }
      })

      renderTabs()

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
    const initialTab = createTerminalTab({ focus: true })
    if (initialTab) {
      await syncTabToWorkspace(initialTab)
      startSession(initialTab, { reason: 'initial-tab' }).catch((startError) => {
        appendEvent(initialTab, 'session-error', startError)
        showError(initialTab, startError instanceof Error ? startError.message : 'Unable to start terminal session')
      })
    }
  }

  connectWorkspaceWebSocket()
}

export async function syncTabToWorkspace(tab) {
  try {
    await proxyToApi('/api/v1/workspace/tabs', {
      method: 'POST',
      json: {
        id: tab.id,
        label: tab.label,
        colorId: tab.colorId,
        order: state.tabs.indexOf(tab)
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
        sessionId: t.session ? t.session.id : null
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
