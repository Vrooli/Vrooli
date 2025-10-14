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

const workspaceCallbacks = {
  queueSessionOverviewRefresh: null
}

export function configureWorkspace(options = {}) {
  workspaceCallbacks.queueSessionOverviewRefresh = typeof options.queueSessionOverviewRefresh === 'function'
    ? options.queueSessionOverviewRefresh
    : null
}

function queueSessionOverviewRefresh(delay = 0) {
  workspaceCallbacks.queueSessionOverviewRefresh?.(delay)
}

export async function initializeWorkspace() {
  try {
    const response = await proxyToApi('/api/v1/workspace')
    if (!response.ok) {
      throw new Error(`Failed to load workspace: ${response.status}`)
    }
    const workspace = await response.json()

    if (workspace.tabs && workspace.tabs.length > 0) {
      workspace.tabs.forEach((tabMeta) => {
        const tab = createTerminalTab({
          focus: false,
          id: tabMeta.id,
          label: tabMeta.label,
          colorId: tabMeta.colorId
        })
        if (tab && tabMeta.sessionId) {
          reconnectSession(tab, tabMeta.sessionId).catch(() => {
            console.log(`Session ${tabMeta.sessionId} no longer available for tab ${tabMeta.id}`)
          })
        }
      })

      if (workspace.activeTabId) {
        setActiveTab(workspace.activeTabId)
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
  proxyToApi('/api/v1/workspace', {
    method: 'PUT',
    json: {
      activeTabId: state.activeTabId,
      tabs: state.tabs.map((t, idx) => ({
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
  if (state.workspaceSocket && state.workspaceSocket.readyState === WebSocket.OPEN) {
    return
  }

  const url = buildWebSocketUrl('/ws/workspace/stream')
  const socket = new WebSocket(url)
  state.workspaceSocket = socket

  socket.addEventListener('open', () => {
    console.log('Workspace WebSocket connected')
    if (state.workspaceReconnectTimer) {
      clearTimeout(state.workspaceReconnectTimer)
      state.workspaceReconnectTimer = null
    }
  })

  socket.addEventListener('message', async (event) => {
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
        console.log('Received initial workspace state')
        return
      }

      handleWorkspaceEvent(data)
    } catch (error) {
      console.error('Failed to parse workspace event:', error)
    }
  })

  socket.addEventListener('close', () => {
    console.log('Workspace WebSocket closed, reconnecting...')
    state.workspaceSocket = null
    if (!state.workspaceReconnectTimer) {
      state.workspaceReconnectTimer = setTimeout(() => {
        connectWorkspaceWebSocket()
      }, 3000)
    }
  })

  socket.addEventListener('error', (error) => {
    console.error('Workspace WebSocket error:', error)
  })
}

function handleWorkspaceEvent(event) {
  if (!event || !event.type) return

  switch (event.type) {
    case 'workspace-full-update':
      console.log('Full workspace update:', event.payload)
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
      console.log('Keyboard toolbar mode changed event (deprecated):', event.payload)
      break
    default:
      console.log('Unknown workspace event:', event.type)
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
