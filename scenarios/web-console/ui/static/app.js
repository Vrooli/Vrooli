import {
  elements,
  state
} from './modules/state.js'
import {
  configureTabManager,
  setTerminalEventHandlers,
  initializeTabCustomizationUI,
  initializeShortcutButtons,
  createTerminalTab,
  destroyTerminalTab,
  findTab,
  getActiveTab,
  renderTabs,
  closeTabCustomization,
  setActiveTab
} from './modules/tab-manager.js'
import {
  configureSessionService,
  startSession,
  stopSession,
  reconnectSession,
  handleTerminalData,
  transmitInputForTab,
  queueInputForTab,
  sendResizeForTab,
  handleSignOutAllSessions,
  handleCloseDetachedTabs,
  updateSessionActions,
  handleShortcutAction
} from './modules/session-service.js'
import {
  configureSessionOverview,
  renderSessionOverview,
  queueSessionOverviewRefresh,
  startSessionOverviewWatcher,
  stopSessionOverviewWatcher,
  refreshSessionOverview
} from './modules/session-overview.js'
import {
  configureWorkspace,
  initializeWorkspace,
  syncTabToWorkspace,
  deleteTabFromWorkspace,
  syncActiveTabState,
  connectWorkspaceWebSocket
} from './modules/workspace.js'
import {
  configureComposer,
  initializeComposerUI,
  openComposeDialog,
  closeComposeDialog,
  isComposerOpen,
  updateComposeFeedback
} from './modules/composer.js'
import {
  openDrawer,
  closeDrawer,
  toggleDrawer,
  applyDrawerState,
  updateDrawerIndicator,
  resetUnreadEvents
} from './modules/drawer.js'
import {
  renderEventMeta,
  renderEvents,
  renderError,
  configureEventFeed,
  appendEvent,
  showError
} from './modules/event-feed.js'
import { configureAIIntegration, initializeAIIntegration } from './modules/ai-integration.js'
import { initializeIframeBridge } from './modules/bridge.js'

configureSessionService({
  onActiveTabNeedsUpdate: updateUI,
  onSessionActionsChanged: updateSessionActions,
  queueSessionOverviewRefresh
})

configureTabManager({
  onActiveTabChanged: handleActiveTabChanged,
  onTabCloseRequested: (tab) => closeTab(tab.id),
  onTabMetadataChanged: handleTabMetadataChanged,
  onShortcut: handleShortcutAction
})

setTerminalEventHandlers({
  onResize: sendResizeForTab,
  onData: (tab, data) => handleTerminalData(tab, data)
})

configureEventFeed({
  onActiveTabMutation: updateUI
})

configureSessionOverview({
  updateSessionActions
})

configureWorkspace({
  queueSessionOverviewRefresh
})

configureComposer({
  onSubmit: handleComposerSubmit,
  getActiveTab
})

configureAIIntegration({
  getActiveTab,
  transmitInput: transmitInputForTab,
  queueInput: queueInputForTab,
  startSession
})

initializeTabCustomizationUI()
initializeShortcutButtons()
initializeComposerUI()
initializeAIIntegration()

applyDrawerState()
updateDrawerIndicator()
updateComposeFeedback()

initializeEventListeners()
initializeWorkspace()
startSessionOverviewWatcher()
updateSessionActions()
renderSessionOverview()
updateUI()
connectWorkspaceWebSocket()

function updateUI() {
  const tab = getActiveTab()

  if (!tab) {
    renderEventMeta(null)
    renderEvents(null)
    renderError(null)
    emitSessionUpdate(null)
    updateSessionActions()
    renderSessionOverview()
    updateDrawerIndicator()
    return
  }

  renderEventMeta(tab)
  renderEvents(tab)
  renderError(tab)
  emitSessionUpdate(tab)
  updateSessionActions()
  renderSessionOverview()
  updateDrawerIndicator()
}

function emitSessionUpdate(tab) {
  state.bridge?.emit('session-update', {
    tabId: tab?.id || null,
    phase: tab?.phase || 'idle',
    socketState: tab?.socketState || 'disconnected',
    session: tab?.session || null,
    transcriptSize: tab?.transcript?.length || 0,
    events: tab ? tab.events.slice(-20) : [],
    suppressed: tab ? { ...tab.suppressed } : {},
    tabs: state.tabs.map((entry) => ({
      id: entry.id,
      label: entry.label,
      phase: entry.phase,
      socketState: entry.socketState,
      hasSession: Boolean(entry.session)
    }))
  })
}

function handleActiveTabChanged(tab, previousId) {
  resetUnreadEvents()
  updateUI()
  syncActiveTabState()
}

function handleTabMetadataChanged(tab) {
  syncTabToWorkspace(tab)
  syncActiveTabState()
  renderTabs()
  updateUI()
}

async function closeTab(tabId) {
  const tab = findTab(tabId)
  if (!tab) return

  if (state.tabMenu.open && state.tabMenu.tabId === tabId) {
    closeTabCustomization()
  }

  await stopSession(tab)

  destroyTerminalTab(tab)
  state.tabs = state.tabs.filter((entry) => entry.id !== tabId)

  await deleteTabFromWorkspace(tabId)

  let fallback = null
  if (state.activeTabId === tabId) {
    fallback = state.tabs[state.tabs.length - 1] || state.tabs[0] || null
    state.activeTabId = fallback ? fallback.id : null
  }

  renderTabs()

  if (state.tabs.length === 0) {
    const replacement = createTerminalTab({ focus: true })
    if (replacement) {
      await syncTabToWorkspace(replacement)
      startSession(replacement, { reason: 'replacement-tab' }).catch((error) => {
        appendEvent(replacement, 'session-error', error)
        showError(replacement, error instanceof Error ? error.message : 'Unable to start terminal session')
      })
    }
  } else if (fallback) {
    setActiveTab(fallback.id)
  } else {
    updateUI()
  }

  syncActiveTabState()
  updateSessionActions()
  renderSessionOverview()
}

function handleComposerSubmit({ tab, value, appendNewline }) {
  if (!tab) {
    updateComposeFeedback()
    return false
  }

  const meta = {
    appendNewline,
    eventType: 'composer-send',
    source: 'composer'
  }

  let sent = false
  if (tab.socket && tab.socket.readyState === WebSocket.OPEN) {
    sent = transmitInputForTab(tab, value, meta)
  } else {
    queueInputForTab(tab, value, meta)
    if (tab.phase === 'idle' || tab.phase === 'closed') {
      startSession(tab, { reason: 'composer-input' }).catch((error) => {
        appendEvent(tab, 'session-error', error)
        showError(tab, error instanceof Error ? error.message : 'Unable to start terminal session')
      })
    }
    sent = true
  }

  if (!sent) {
    showError(tab, 'Failed to send message to terminal')
  }

  return sent
}

function initializeEventListeners() {
  if (elements.addTabBtn) {
    elements.addTabBtn.addEventListener('click', async () => {
      const tab = createTerminalTab({ focus: true })
      if (tab) {
        await syncTabToWorkspace(tab)
        startSession(tab, { reason: 'new-tab' }).catch((error) => {
          appendEvent(tab, 'session-error', error)
          showError(tab, error instanceof Error ? error.message : 'Unable to start terminal session')
        })
      }
    })
  }

  if (elements.signOutAllSessions) {
    elements.signOutAllSessions.addEventListener('click', handleSignOutAllSessions)
  }

  if (elements.closeDetachedTabs) {
    elements.closeDetachedTabs.addEventListener('click', handleCloseDetachedTabs)
  }

  if (elements.sessionOverviewRefresh) {
    elements.sessionOverviewRefresh.addEventListener('click', () => refreshSessionOverview({ silent: false }))
  }

  if (elements.drawerToggle) {
    elements.drawerToggle.addEventListener('click', () => toggleDrawer())
  }

  if (elements.drawerClose) {
    elements.drawerClose.addEventListener('click', () => closeDrawer())
  }

  if (elements.drawerBackdrop) {
    elements.drawerBackdrop.addEventListener('click', () => closeDrawer())
  }

  document.addEventListener('keydown', handleGlobalKeyDown)
  window.addEventListener('resize', handleWindowResize)
  document.addEventListener('visibilitychange', handleVisibilityChange)

  state.bridge = initializeIframeBridge({
    'init-session': () => {
      state.bridge?.emit('ready', { timestamp: Date.now() })
    },
    'end-session': async () => {
      const tab = getActiveTab()
      if (tab) {
        await stopSession(tab)
      }
      state.bridge?.emit('session-ended', { requestedByParent: true })
    },
    'request-screenshot': async () => {
      if (window.html2canvas) {
        const canvas = await window.html2canvas(document.body, { backgroundColor: '#0f172a' })
        state.bridge?.emit('screenshot', { image: canvas.toDataURL('image/png', 0.9), requestedAt: Date.now() })
      } else {
        state.bridge?.emit('error', { type: 'request-screenshot', message: 'html2canvas not available' })
      }
    },
    'request-transcript': () => {
      const tab = getActiveTab()
      state.bridge?.emit('transcript', { transcript: tab ? tab.transcript : [], requestedAt: Date.now(), tabId: tab?.id || null })
    },
    'request-logs': () => {
      const tab = getActiveTab()
      state.bridge?.emit('logs', { logs: tab ? tab.events.slice(-100) : [], requestedAt: Date.now(), tabId: tab?.id || null })
    }
  })
}

function handleGlobalKeyDown(event) {
  if (event.key !== 'Escape') return
  if (isComposerOpen()) {
    event.preventDefault()
    closeComposeDialog({ preserveValue: false })
    return
  }
  if (state.tabMenu.open) {
    event.preventDefault()
    closeTabCustomization()
    return
  }
  if (state.drawer.open) {
    event.preventDefault()
    closeDrawer()
  }
}

function handleWindowResize() {
  requestAnimationFrame(() => {
    const active = getActiveTab()
    if (active) {
      active.fitAddon?.fit()
    }
    if (state.tabMenu.open) {
      closeTabCustomization()
    }
  })
}

function handleVisibilityChange() {
  if (document.hidden) return

  const activeTab = getActiveTab()
  if (activeTab && activeTab.term) {
    const forceRender = () => {
      try {
        activeTab.term.focus()
        activeTab.fitAddon?.fit()
        activeTab.term.scrollToBottom()
        activeTab.term.write('')
        activeTab.term.refresh(0, activeTab.term.rows - 1)
      } catch (error) {
        console.warn('Failed to refresh terminal on visibility change:', error)
      }
    }

    requestAnimationFrame(forceRender)
    setTimeout(forceRender, 100)
    setTimeout(forceRender, 250)
  }

  state.tabs.forEach((tab) => {
    if (tab.session && tab.phase === 'running' && !tab.socket) {
      appendEvent(tab, 'visibility-reconnect', { sessionId: tab.session.id })
      reconnectSession(tab, tab.session.id).catch((error) => {
        console.warn('Failed to reconnect on visibility change:', error)
      })
    }
  })
}
