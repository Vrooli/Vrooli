const elements = {
  sessionId: document.getElementById('sessionId'),
  sessionPhase: document.getElementById('sessionPhase'),
  socketState: document.getElementById('socketState'),
  sessionCommand: document.getElementById('sessionCommand'),
  transcriptSize: document.getElementById('transcriptSize'),
  eventFeed: document.getElementById('eventFeed'),
  eventMeta: document.getElementById('eventMeta'),
  errorBanner: document.getElementById('errorBanner'),
  tabList: document.getElementById('tabList'),
  addTabBtn: document.getElementById('addTabBtn'),
  tabAddSlot: document.getElementById('tabAddSlot'),
  terminalHost: document.getElementById('terminalHost'),
  layout: document.getElementById('mainLayout'),
  drawerToggle: document.getElementById('drawerToggle'),
  drawerClose: document.getElementById('drawerClose'),
  drawerIndicator: document.getElementById('drawerIndicator'),
  detailsDrawer: document.getElementById('detailsDrawer'),
  drawerBackdrop: document.getElementById('drawerBackdrop'),
  tabContextMenu: document.getElementById('tabContextMenu'),
  tabContextForm: document.getElementById('tabContextForm'),
  tabContextName: document.getElementById('tabContextName'),
  tabContextColors: document.getElementById('tabContextColors'),
  tabContextCancel: document.getElementById('tabContextCancel'),
  tabContextReset: document.getElementById('tabContextReset'),
  tabContextBackdrop: document.getElementById('tabContextBackdrop'),
  signOutAllSessions: document.getElementById('signOutAllSessions'),
  aiCommandInput: document.getElementById('aiCommandInput'),
  aiGenerateBtn: document.getElementById('aiGenerateBtn')
}

const shortcutButtons = Array.from(document.querySelectorAll('[data-shortcut-id]'))

const TAB_COLOR_OPTIONS = [
  {
    id: 'sky',
    label: 'Sky',
    swatch: '#38bdf8',
    styles: {
      border: 'rgba(56, 189, 248, 0.28)',
      hover: 'rgba(56, 189, 248, 0.45)',
      active: 'rgba(56, 189, 248, 0.65)',
      selectedStart: 'rgba(56, 189, 248, 0.35)',
      selectedEnd: 'rgba(14, 116, 144, 0.38)',
      glow: 'rgba(56, 189, 248, 0.35)',
      label: '#f8fafc'
    }
  },
  {
    id: 'emerald',
    label: 'Emerald',
    swatch: '#34d399',
    styles: {
      border: 'rgba(52, 211, 153, 0.26)',
      hover: 'rgba(16, 185, 129, 0.42)',
      active: 'rgba(16, 185, 129, 0.62)',
      selectedStart: 'rgba(16, 185, 129, 0.32)',
      selectedEnd: 'rgba(5, 150, 105, 0.38)',
      glow: 'rgba(16, 185, 129, 0.32)',
      label: '#ecfeff'
    }
  },
  {
    id: 'amber',
    label: 'Amber',
    swatch: '#f59e0b',
    styles: {
      border: 'rgba(245, 158, 11, 0.32)',
      hover: 'rgba(245, 158, 11, 0.48)',
      active: 'rgba(245, 158, 11, 0.68)',
      selectedStart: 'rgba(251, 191, 36, 0.35)',
      selectedEnd: 'rgba(217, 119, 6, 0.42)',
      glow: 'rgba(245, 158, 11, 0.35)',
      label: '#fffbeb'
    }
  },
  {
    id: 'sunset',
    label: 'Sunset',
    swatch: '#f97316',
    styles: {
      border: 'rgba(249, 115, 22, 0.32)',
      hover: 'rgba(249, 115, 22, 0.48)',
      active: 'rgba(249, 115, 22, 0.7)',
      selectedStart: 'rgba(251, 146, 60, 0.36)',
      selectedEnd: 'rgba(194, 65, 12, 0.38)',
      glow: 'rgba(249, 115, 22, 0.36)',
      label: '#fff7ed'
    }
  },
  {
    id: 'violet',
    label: 'Violet',
    swatch: '#a855f7',
    styles: {
      border: 'rgba(168, 85, 247, 0.32)',
      hover: 'rgba(139, 92, 246, 0.5)',
      active: 'rgba(139, 92, 246, 0.7)',
      selectedStart: 'rgba(196, 181, 253, 0.36)',
      selectedEnd: 'rgba(126, 58, 242, 0.42)',
      glow: 'rgba(168, 85, 247, 0.38)',
      label: '#f5f3ff'
    }
  },
  {
    id: 'slate',
    label: 'Slate',
    swatch: '#64748b',
    styles: {
      border: 'rgba(100, 116, 139, 0.32)',
      hover: 'rgba(100, 116, 139, 0.5)',
      active: 'rgba(100, 116, 139, 0.7)',
      selectedStart: 'rgba(148, 163, 184, 0.32)',
      selectedEnd: 'rgba(71, 85, 105, 0.48)',
      glow: 'rgba(100, 116, 139, 0.35)',
      label: '#f8fafc'
    }
  }
]

const TAB_COLOR_DEFAULT = TAB_COLOR_OPTIONS[0]?.id || 'sky'
const TAB_COLOR_MAP = TAB_COLOR_OPTIONS.reduce((map, option) => {
  map[option.id] = option
  return map
}, {})

const TAB_LONG_PRESS_DELAY = 550
const tabColorButtons = new Map()

const terminalDefaults = {
  convertEol: true,
  cursorBlink: true,
  fontFamily: 'JetBrains Mono, SFMono-Regular, Menlo, monospace',
  fontSize: 14,
  theme: {
    background: '#0f172a',
    foreground: '#e2e8f0',
    cursor: '#38bdf8'
  }
}

const AGGREGATED_EVENT_TYPES = new Set(['ws-empty-frame'])
const SUPPRESSED_EVENT_LABELS = {
  'ws-empty-frame': 'Empty frames suppressed'
}

function initialSuppressedState() {
  return {
    'ws-empty-frame': 0
  }
}

function formatCommandLabel(command, args) {
  if (!command) return '—'
  if (Array.isArray(args) && args.length > 0) {
    return `${command} ${args.join(' ')}`
  }
  return command
}

const textDecoder = new TextDecoder()

const state = {
  tabs: [],
  activeTabId: null,
  bridge: null,
  workspaceSocket: null,
  workspaceReconnectTimer: null,
  drawer: {
    open: false,
    unreadCount: 0,
    previousFocus: null
  },
  tabMenu: {
    open: false,
    tabId: null,
    selectedColor: TAB_COLOR_DEFAULT,
    anchor: { x: 0, y: 0 }
  }
}

let tabSequence = 0

state.bridge = initializeIframeBridge()

// Load workspace from server
async function initializeWorkspace() {
  try {
    const response = await proxyToApi('/api/v1/workspace')
    if (!response.ok) {
      throw new Error(`Failed to load workspace: ${response.status}`)
    }
    const workspace = await response.json()

    // Restore tabs from workspace
    if (workspace.tabs && workspace.tabs.length > 0) {
      workspace.tabs.forEach((tabMeta) => {
        const tab = createTerminalTab({
          focus: false,
          id: tabMeta.id,
          label: tabMeta.label,
          colorId: tabMeta.colorId
        })
        if (tab && tabMeta.sessionId) {
          // Try to reconnect to existing session (may fail if session expired)
          reconnectSession(tab, tabMeta.sessionId).catch(() => {
            // Session no longer exists, that's okay
            console.log(`Session ${tabMeta.sessionId} no longer available for tab ${tabMeta.id}`)
          })
        }
      })

      // Set active tab
      if (workspace.activeTabId) {
        setActiveTab(workspace.activeTabId)
      }
    } else {
      // No tabs in workspace, create initial tab
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
    // Fallback: create initial tab
    const initialTab = createTerminalTab({ focus: true })
    if (initialTab) {
      await syncTabToWorkspace(initialTab)
      startSession(initialTab, { reason: 'initial-tab' }).catch((error) => {
        appendEvent(initialTab, 'session-error', error)
        showError(initialTab, error instanceof Error ? error.message : 'Unable to start terminal session')
      })
    }
  }

  // Connect workspace WebSocket
  connectWorkspaceWebSocket()
}

// Sync a local tab to the server workspace
async function syncTabToWorkspace(tab) {
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

// Initialize workspace on load
initializeWorkspace()

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

shortcutButtons.forEach((button) => {
  button.addEventListener('click', () => handleShortcutButton(button))
})

if (elements.signOutAllSessions) {
  elements.signOutAllSessions.addEventListener('click', handleSignOutAllSessions)
}

initializeTabCustomizationUI()

if (elements.drawerToggle) {
  elements.drawerToggle.addEventListener('click', () => toggleDrawer())
}

if (elements.drawerClose) {
  elements.drawerClose.addEventListener('click', () => closeDrawer())
}

if (elements.drawerBackdrop) {
  elements.drawerBackdrop.addEventListener('click', () => closeDrawer())
}

document.addEventListener('keydown', (event) => {
  if (event.key !== 'Escape') return
  if (state.tabMenu.open) {
    event.preventDefault()
    closeTabCustomization()
    return
  }
  if (state.drawer.open) {
    event.preventDefault()
    closeDrawer()
  }
})

window.addEventListener('resize', () => {
  requestAnimationFrame(() => {
    const active = getActiveTab()
    if (active) {
      active.fitAddon?.fit()
    }
    if (state.tabMenu.open) {
      positionTabContextMenu(state.tabMenu.anchor)
    }
  })
})

updateUI()
updateDrawerIndicator()
applyDrawerState()

function toggleDrawer(forceState) {
  if (typeof forceState === 'boolean') {
    return forceState ? openDrawer() : closeDrawer()
  }
  if (state.drawer.open) {
    closeDrawer()
  } else {
    openDrawer()
  }
}

function openDrawer() {
  if (state.drawer.open) return
  state.drawer.open = true
  state.drawer.previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
  applyDrawerState()
  resetUnreadEvents()
  requestAnimationFrame(() => {
    if (!elements.detailsDrawer || typeof elements.detailsDrawer.focus !== 'function') return
    try {
      elements.detailsDrawer.focus({ preventScroll: true })
    } catch (_error) {
      elements.detailsDrawer.focus()
    }
  })
}

function closeDrawer() {
  if (!state.drawer.open) return
  state.drawer.open = false
  applyDrawerState()
  updateDrawerIndicator()
  const fallback = state.drawer.previousFocus && state.drawer.previousFocus.isConnected ? state.drawer.previousFocus : elements.drawerToggle
  requestAnimationFrame(() => {
    fallback?.focus?.()
    state.drawer.previousFocus = null
  })
}

function applyDrawerState() {
  if (!elements.layout || !elements.detailsDrawer || !elements.drawerToggle) return
  elements.layout.classList.toggle('drawer-open', state.drawer.open)
  elements.detailsDrawer.setAttribute('aria-hidden', state.drawer.open ? 'false' : 'true')
  if (state.drawer.open) {
    elements.detailsDrawer.removeAttribute('inert')
  } else {
    elements.detailsDrawer.setAttribute('inert', '')
  }
  elements.drawerToggle.setAttribute('aria-expanded', state.drawer.open ? 'true' : 'false')
  if (elements.drawerBackdrop) {
    elements.drawerBackdrop.setAttribute('aria-hidden', state.drawer.open ? 'false' : 'true')
  }
}

function resetUnreadEvents() {
  if (!state.drawer) return
  if (state.drawer.unreadCount !== 0) {
    state.drawer.unreadCount = 0
  }
  updateDrawerIndicator()
}

function updateDrawerIndicator() {
  if (!elements.drawerIndicator) return
  const count = state.drawer?.unreadCount || 0
  const hasUnread = count > 0
  if (elements.drawerToggle) {
    elements.drawerToggle.setAttribute('aria-label', hasUnread ? 'Open session details (new activity)' : 'Open session details')
    elements.drawerToggle.classList.toggle('has-unread', hasUnread)
  }
  if (hasUnread) {
    elements.drawerIndicator.classList.remove('hidden')
    elements.drawerIndicator.setAttribute('aria-hidden', 'false')
  } else {
    elements.drawerIndicator.classList.add('hidden')
    elements.drawerIndicator.setAttribute('aria-hidden', 'true')
  }
}

function createTerminalTab({ focus = false, id = null, label = null, colorId = TAB_COLOR_DEFAULT } = {}) {
  if (!elements.terminalHost) return null
  if (!id) {
    id = `tab-${Date.now()}-${++tabSequence}`
  }
  if (!label) {
    tabSequence++
    label = `Terminal ${tabSequence}`
  }

  const container = document.createElement('div')
  container.className = 'terminal-screen'
  container.dataset.tabId = id

  elements.terminalHost.appendChild(container)

  const term = new window.Terminal({ ...terminalDefaults })
  const fitAddon = new window.FitAddon.FitAddon()
  term.loadAddon(fitAddon)
  term.open(container)

  // Write initial prompt to ensure terminal renders content immediately
  term.write('\r')

  fitAddon.fit()

  const tab = {
    id,
    label,
    defaultLabel: label,
    colorId: colorId || TAB_COLOR_DEFAULT,
    term,
    fitAddon,
    container,
    phase: 'idle',
    socketState: 'disconnected',
    session: null,
    socket: null,
    transcript: [],
    events: [],
    suppressed: initialSuppressedState(),
    pendingWrites: [],
    errorMessage: '',
    lastSentSize: { cols: 0, rows: 0 },
    domItem: null,
    domButton: null,
    domClose: null,
    domLabel: null
  }

  term.onResize(({ cols, rows }) => {
    sendResize(tab, cols, rows)
  })

  // Track last input time to prevent duplicate visual input bug
  let lastInputTime = 0
  let lastInputData = ''

  term.onData((data) => {
    const now = Date.now()

    // Deduplicate rapid identical inputs (visual bug workaround)
    // If same data arrives within 50ms, it's likely a duplicate event
    if (data === lastInputData && (now - lastInputTime) < 50) {
      console.log('Duplicate input event detected and ignored:', data)
      return
    }

    lastInputTime = now
    lastInputData = data
    handleTerminalData(tab, data)
  })

  container.addEventListener('mousedown', () => {
    setActiveTab(tab.id)
  })

  state.tabs.push(tab)
  renderTabs()

  if (focus || state.activeTabId === null) {
    setActiveTab(tab.id)
  } else {
    tab.container.classList.remove('active')
  }

  return tab
}

function destroyTerminalTab(tab) {
  if (!tab) return
  if (state.tabMenu.open && state.tabMenu.tabId === tab.id) {
    closeTabCustomization()
  }
  try {
    tab.term?.dispose()
  } catch (_error) {
    // ignore
  }
  try {
    tab.socket?.close()
  } catch (_error) {
    // ignore
  }
  if (tab.container?.parentElement) {
    tab.container.parentElement.removeChild(tab.container)
  }
  if (tab.domItem?.parentElement) {
    tab.domItem.parentElement.removeChild(tab.domItem)
  }
  tab.domItem = null
  tab.domButton = null
  tab.domClose = null
  tab.domLabel = null
}

function getActiveTab() {
  if (!state.activeTabId) return null
  return state.tabs.find((tab) => tab.id === state.activeTabId) || null
}

function findTab(tabId) {
  return state.tabs.find((tab) => tab.id === tabId) || null
}

function setActiveTab(tabId) {
  const tab = findTab(tabId)
  if (!tab) {
    return
  }
  const previousActiveId = state.activeTabId
  state.activeTabId = tabId
  state.tabs.forEach((entry) => {
    if (!entry.container) return
    if (entry.id === tabId) {
      entry.container.classList.add('active')
    } else {
      entry.container.classList.remove('active')
    }
    updateTabButtonState(entry)
  })
  resetUnreadEvents()
  focusTerminal(tab)
  updateUI()

  // Sync active tab change to server (only if it actually changed)
  if (previousActiveId !== tabId) {
    proxyToApi('/api/v1/workspace', {
      method: 'PUT',
      json: {
        activeTabId: tabId,
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
}

function updateTabButtonState(tab) {
  if (!tab.domButton) return
  tab.domButton.setAttribute('aria-selected', tab.id === state.activeTabId ? 'true' : 'false')
  tab.domButton.dataset.phase = tab.phase
  tab.domButton.dataset.socketState = tab.socketState
  tab.domButton.classList.toggle('tab-running', tab.phase === 'running')
  tab.domButton.classList.toggle('tab-error', tab.socketState === 'error')
  if (tab.domClose) {
    tab.domClose.disabled = state.tabs.length === 1
  }
  applyTabAppearance(tab)
}

function focusTerminal(tab) {
  if (!tab) return
  requestAnimationFrame(() => {
    tab.term.focus()
    tab.fitAddon?.fit()
  })
}

function renderTabs() {
  if (!elements.tabList) return

  const knownIds = new Set(state.tabs.map((tab) => tab.id))

  Array.from(elements.tabList.children).forEach((child) => {
    const tabId = child instanceof HTMLElement ? child.dataset.tabId : undefined
    if (tabId && !knownIds.has(tabId)) {
      elements.tabList.removeChild(child)
    }
  })

  state.tabs.forEach((tab) => {
    if (!tab.domItem) {
      const item = document.createElement('div')
      item.className = 'tab-item'
      item.dataset.tabId = tab.id

      const selectBtn = document.createElement('button')
      selectBtn.type = 'button'
      selectBtn.className = 'tab-button'
      selectBtn.dataset.tabId = tab.id
      selectBtn.setAttribute('role', 'tab')
      selectBtn.title = tab.label
      const labelSpan = document.createElement('span')
      labelSpan.className = 'tab-label'
      labelSpan.textContent = tab.label
      selectBtn.appendChild(labelSpan)
      selectBtn.addEventListener('click', () => setActiveTab(tab.id))
      registerTabCustomizationHandlers(selectBtn, tab)

      const closeBtn = document.createElement('button')
      closeBtn.type = 'button'
      closeBtn.className = 'tab-close'
      closeBtn.dataset.tabId = tab.id
      closeBtn.setAttribute('aria-label', `Close ${tab.label}`)
      closeBtn.textContent = '×'
      closeBtn.addEventListener('click', (event) => {
        event.stopPropagation()
        closeTab(tab.id)
      })

      item.appendChild(selectBtn)
      item.appendChild(closeBtn)

      const insertionPoint = elements.tabAddSlot || elements.addTabBtn || null
      if (insertionPoint && elements.tabList.contains(insertionPoint)) {
        elements.tabList.insertBefore(item, insertionPoint)
      } else {
        elements.tabList.appendChild(item)
      }

      tab.domItem = item
      tab.domButton = selectBtn
      tab.domClose = closeBtn
      tab.domLabel = labelSpan
    } else {
      const insertionPoint = elements.tabAddSlot || elements.addTabBtn || null
      if (insertionPoint && elements.tabList.contains(insertionPoint)) {
        elements.tabList.insertBefore(tab.domItem, insertionPoint)
      } else {
        elements.tabList.appendChild(tab.domItem)
      }
    }

    if (tab.domLabel) {
      tab.domLabel.textContent = tab.label
    } else {
      tab.domButton.textContent = tab.label
    }
    tab.domButton.title = tab.label
    if (tab.domClose) {
      tab.domClose.setAttribute('aria-label', `Close ${tab.label}`)
    }
    applyTabAppearance(tab)
    updateTabButtonState(tab)
  })
}

function registerTabCustomizationHandlers(button, tab) {
  if (!button) return
  button.addEventListener('contextmenu', (event) => {
    event.preventDefault()
    handleTabCustomizationRequest(tab, event)
  })

  button.addEventListener('pointerdown', (event) => {
    if (event.pointerType === 'mouse') return
    let longPressTriggered = false
    const timer = window.setTimeout(() => {
      longPressTriggered = true
      handleTabCustomizationRequest(tab, event)
    }, TAB_LONG_PRESS_DELAY)

    const cancel = () => {
      window.clearTimeout(timer)
      button.removeEventListener('pointerup', cancel)
      button.removeEventListener('pointerleave', cancel)
      button.removeEventListener('pointercancel', cancel)
      if (longPressTriggered) {
        button.addEventListener('click', suppressNextClick, { once: true, capture: true })
      }
    }

    button.addEventListener('pointerup', cancel)
    button.addEventListener('pointerleave', cancel)
    button.addEventListener('pointercancel', cancel)
  })
}

function handleTabCustomizationRequest(tab, triggerEvent) {
  if (!tab) return
  let anchor
  if (triggerEvent && typeof triggerEvent.clientX === 'number' && typeof triggerEvent.clientY === 'number') {
    anchor = { x: triggerEvent.clientX, y: triggerEvent.clientY }
  } else if (tab.domButton) {
    const rect = tab.domButton.getBoundingClientRect()
    anchor = { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
  } else {
    anchor = { x: window.innerWidth / 2, y: window.innerHeight / 2 }
  }
  setActiveTab(tab.id)
  openTabCustomization(tab, anchor)
}

let tabMenuCloseTimer = null

function initializeTabCustomizationUI() {
  const container = elements.tabContextColors
  if (!container) return
  container.innerHTML = ''
  tabColorButtons.clear()

  TAB_COLOR_OPTIONS.forEach((option) => {
    const button = document.createElement('button')
    button.type = 'button'
    button.className = 'tab-color-option'
    button.dataset.colorId = option.id
    button.style.setProperty('--swatch-color', option.swatch)
    button.setAttribute('role', 'option')
    button.setAttribute('aria-label', `${option.label} tab color`)
    button.addEventListener('click', () => {
      selectTabColor(option.id)
    })
    tabColorButtons.set(option.id, button)
    container.appendChild(button)
  })

  if (elements.tabContextForm) {
    elements.tabContextForm.addEventListener('submit', handleTabContextSubmit)
  }
  if (elements.tabContextCancel) {
    elements.tabContextCancel.addEventListener('click', () => closeTabCustomization())
  }
  if (elements.tabContextBackdrop) {
    elements.tabContextBackdrop.addEventListener('click', () => closeTabCustomization())
  }
  if (elements.tabContextReset) {
    elements.tabContextReset.addEventListener('click', handleTabContextReset)
  }

  selectTabColor(TAB_COLOR_DEFAULT)
}

function openTabCustomization(tab, anchor) {
  if (!elements.tabContextMenu || !elements.tabContextBackdrop || !elements.tabContextName) return

  if (tabMenuCloseTimer) {
    window.clearTimeout(tabMenuCloseTimer)
    tabMenuCloseTimer = null
  }

  const option = TAB_COLOR_MAP[tab.colorId] ? tab.colorId : TAB_COLOR_DEFAULT

  state.tabMenu.open = true
  state.tabMenu.tabId = tab.id
  state.tabMenu.selectedColor = option
  state.tabMenu.anchor = anchor || { x: window.innerWidth / 2, y: window.innerHeight / 2 }

  elements.tabContextName.value = tab.label
  selectTabColor(state.tabMenu.selectedColor)

  elements.tabContextBackdrop.classList.remove('hidden')
  elements.tabContextMenu.classList.remove('hidden')
  elements.tabContextMenu.style.left = '0px'
  elements.tabContextMenu.style.top = '0px'

  requestAnimationFrame(() => {
    elements.tabContextBackdrop.classList.add('active')
    elements.tabContextBackdrop.setAttribute('aria-hidden', 'false')
    positionTabContextMenu(state.tabMenu.anchor)
    elements.tabContextMenu.classList.add('active')
    elements.tabContextMenu.setAttribute('aria-hidden', 'false')
    try {
      elements.tabContextName.focus({ preventScroll: true })
    } catch (_error) {
      elements.tabContextName.focus()
    }
    elements.tabContextName.select()
  })
}

function closeTabCustomization() {
  if (!state.tabMenu.open) return
  state.tabMenu.open = false
  state.tabMenu.tabId = null
  state.tabMenu.anchor = { x: 0, y: 0 }
  state.tabMenu.selectedColor = TAB_COLOR_DEFAULT

  if (!elements.tabContextMenu || !elements.tabContextBackdrop) return

  elements.tabContextMenu.classList.remove('active')
  elements.tabContextMenu.setAttribute('aria-hidden', 'true')
  elements.tabContextBackdrop.classList.remove('active')
  elements.tabContextBackdrop.setAttribute('aria-hidden', 'true')

  tabMenuCloseTimer = window.setTimeout(() => {
    elements.tabContextMenu?.classList.add('hidden')
    elements.tabContextBackdrop?.classList.add('hidden')
    tabMenuCloseTimer = null
  }, 180)
}

function positionTabContextMenu(anchor) {
  if (!elements.tabContextMenu) return
  const menu = elements.tabContextMenu
  const { x, y } = anchor || { x: window.innerWidth / 2, y: window.innerHeight / 2 }

  const padding = 16
  const rect = menu.getBoundingClientRect()

  let left = x
  let top = y

  if (left + rect.width + padding > window.innerWidth) {
    left = window.innerWidth - rect.width - padding
  }
  if (left < padding) {
    left = padding
  }
  if (top + rect.height + padding > window.innerHeight) {
    top = window.innerHeight - rect.height - padding
  }
  if (top < padding) {
    top = padding
  }

  menu.style.left = `${left}px`
  menu.style.top = `${top}px`
}

function selectTabColor(colorId) {
  const option = TAB_COLOR_MAP[colorId] ? colorId : TAB_COLOR_DEFAULT
  state.tabMenu.selectedColor = option
  tabColorButtons.forEach((button, id) => {
    const isSelected = id === option
    button.setAttribute('aria-selected', isSelected ? 'true' : 'false')
    button.classList.toggle('selected', isSelected)
  })
}

function handleTabContextSubmit(event) {
  event.preventDefault()
  const tabId = state.tabMenu.tabId
  const tab = tabId ? findTab(tabId) : null
  if (!tab) {
    closeTabCustomization()
    return
  }

  const inputValue = elements.tabContextName?.value?.trim() || ''
  const colorId = state.tabMenu.selectedColor && TAB_COLOR_MAP[state.tabMenu.selectedColor] ? state.tabMenu.selectedColor : TAB_COLOR_DEFAULT

  tab.label = inputValue || tab.defaultLabel || tab.label
  tab.colorId = colorId

  if (tab.domLabel) {
    tab.domLabel.textContent = tab.label
  }
  if (tab.domButton) {
    tab.domButton.title = tab.label
  }
  if (tab.domClose) {
    tab.domClose.setAttribute('aria-label', `Close ${tab.label}`)
  }

  applyTabAppearance(tab)
  updateTabButtonState(tab)
  renderTabs()
  if (tab.id === state.activeTabId) {
    updateUI()
  }

  // Sync to server
  proxyToApi(`/api/v1/workspace/tabs/${tabId}`, {
    method: 'PATCH',
    json: {
      label: tab.label,
      colorId: tab.colorId
    }
  }).catch((error) => {
    console.error('Failed to update tab on server:', error)
  })

  closeTabCustomization()
}

function handleTabContextReset() {
  if (!state.tabMenu.open) return
  const tabId = state.tabMenu.tabId
  const tab = tabId ? findTab(tabId) : null
  if (!tab) {
    closeTabCustomization()
    return
  }
  const fallbackLabel = tab.defaultLabel || tab.label
  if (elements.tabContextName) {
    elements.tabContextName.value = fallbackLabel
  }
  state.tabMenu.selectedColor = TAB_COLOR_DEFAULT
  selectTabColor(TAB_COLOR_DEFAULT)
}

function suppressNextClick(event) {
  event.preventDefault()
  event.stopImmediatePropagation()
}

function applyTabAppearance(tab) {
  if (!tab?.domButton) return
  if (!TAB_COLOR_MAP[tab.colorId]) {
    tab.colorId = TAB_COLOR_DEFAULT
  }
  const option = TAB_COLOR_MAP[tab.colorId] || TAB_COLOR_MAP[TAB_COLOR_DEFAULT]
  const styles = option?.styles || TAB_COLOR_MAP[TAB_COLOR_DEFAULT].styles
  if (!styles) return
  const style = tab.domButton.style
  tab.domButton.dataset.tabColor = option?.id || TAB_COLOR_DEFAULT
  style.setProperty('--tab-color-border', styles.border)
  style.setProperty('--tab-color-border-hover', styles.hover)
  style.setProperty('--tab-color-border-active', styles.active)
  style.setProperty('--tab-color-selected-start', styles.selectedStart)
  style.setProperty('--tab-color-selected-end', styles.selectedEnd)
  style.setProperty('--tab-color-glow', styles.glow)
  style.setProperty('--tab-color-label', styles.label)
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

  // Sync deletion to server
  proxyToApi(`/api/v1/workspace/tabs/${tabId}`, {
    method: 'DELETE'
  }).catch((error) => {
    console.error('Failed to delete tab from server:', error)
  })

  if (state.activeTabId === tabId) {
    const fallback = state.tabs[state.tabs.length - 1] || state.tabs[0] || null
    state.activeTabId = fallback ? fallback.id : null
  }

  renderTabs()

  if (state.tabs.length === 0) {
    const replacement = createTerminalTab({ focus: true })
    if (replacement) {
      startSession(replacement, { reason: 'replacement-tab' }).catch((error) => {
        appendEvent(replacement, 'session-error', error)
        showError(replacement, error instanceof Error ? error.message : 'Unable to start terminal session')
      })
    }
  } else if (state.activeTabId) {
    const active = getActiveTab()
    if (active) {
      setActiveTab(active.id)
    }
  } else {
    updateUI()
  }
}

function handleShortcutButton(button) {
  if (!button) return
  const command = (button.dataset.command || '').trim()
  const id = button.dataset.shortcutId || 'unknown'
  if (!command) {
    const tab = getActiveTab()
    if (tab) appendEvent(tab, 'shortcut-invalid', { id })
    return
  }

  let tab = getActiveTab()
  if (!tab) {
    tab = createTerminalTab({ focus: true })
    if (tab) {
      startSession(tab, { reason: `shortcut:${id}` }).catch((error) => {
        appendEvent(tab, 'session-error', error)
        showError(tab, error instanceof Error ? error.message : 'Unable to start terminal session')
      })
    }
  }
  if (!tab) return

  const meta = {
    eventType: 'shortcut',
    source: id,
    command,
    clearError: true,
    appendNewline: true
  }

  if (tab.socket && tab.socket.readyState === WebSocket.OPEN) {
    const success = transmitInput(tab, command, meta)
    if (!success) {
      showError(tab, 'Terminal stream is not connected')
    }
    return
  }

  queueInput(tab, command, meta)
  appendEvent(tab, 'shortcut-queued', { id, command })

  if (tab.phase === 'idle' || tab.phase === 'closed') {
    startSession(tab, { reason: `shortcut:${id}` }).catch((error) => {
      appendEvent(tab, 'shortcut-start-error', error)
      showError(tab, 'Unable to start terminal session for shortcut')
    })
  }
}

function handleTerminalData(tab, data) {
  if (!tab || typeof data !== 'string' || data.length === 0) {
    return
  }

  if (tab.socket && tab.socket.readyState === WebSocket.OPEN) {
    const success = transmitInput(tab, data, { appendNewline: false, clearError: true })
    if (!success) {
      showError(tab, 'Terminal stream is not connected')
    }
    return
  }

  queueInput(tab, data, { appendNewline: false })
  if (tab.phase === 'idle' || tab.phase === 'closed') {
    startSession(tab, { reason: 'auto-start:input' }).catch((error) => {
      appendEvent(tab, 'session-error', error)
      showError(tab, error instanceof Error ? error.message : 'Unable to start terminal session')
    })
  }
  showError(tab, 'Terminal session is starting…')
}

function queueInput(tab, value, meta = {}) {
  if (!tab || typeof value !== 'string' || value.length === 0) {
    return
  }
  if (!Array.isArray(tab.pendingWrites)) {
    tab.pendingWrites = []
  }
  tab.pendingWrites.push({ value, meta })
}

function flushPendingWrites(tab) {
  if (!tab || !tab.socket || tab.socket.readyState !== WebSocket.OPEN) {
    return
  }
  if (!Array.isArray(tab.pendingWrites) || tab.pendingWrites.length === 0) {
    return
  }
  const queue = tab.pendingWrites.splice(0)
  queue.forEach((item) => {
    const success = transmitInput(tab, item.value, item.meta)
    if (!success) {
      tab.pendingWrites.unshift(item)
    }
  })
}

function setTabPhase(tab, phase) {
  if (!tab) return
  tab.phase = phase
  updateTabButtonState(tab)
  if (tab.id === state.activeTabId) {
    updateUI()
  }
}

function setTabSocketState(tab, socketState) {
  if (!tab) return
  tab.socketState = socketState
  updateTabButtonState(tab)
  if (tab.id === state.activeTabId) {
    updateUI()
  }
}

function updateUI() {
  const tab = getActiveTab()

  if (!tab) {
    if (elements.sessionId) elements.sessionId.textContent = '—'
    if (elements.sessionPhase) elements.sessionPhase.textContent = 'idle'
    if (elements.socketState) elements.socketState.textContent = 'disconnected'
    if (elements.sessionCommand) elements.sessionCommand.textContent = '—'
    if (elements.transcriptSize) elements.transcriptSize.textContent = '0 records'
    renderEventMeta(null)
    renderEvents(null)
    renderError(null)
    emitSessionUpdate(null)
    updateSessionActions()
    resetUnreadEvents()
    return
  }

  const phase = tab.phase || 'idle'
  const socketState = tab.socketState || 'disconnected'

  if (elements.sessionId) {
    elements.sessionId.textContent = tab.session ? `${tab.session.id.slice(0, 8)}…` : '—'
  }
  if (elements.sessionPhase) {
    elements.sessionPhase.textContent = phase
  }
  if (elements.socketState) {
    elements.socketState.textContent = socketState
  }
  if (elements.sessionCommand) {
    elements.sessionCommand.textContent = tab.session ? formatCommandLabel(tab.session.command, tab.session.args) : '—'
  }
  if (elements.transcriptSize) {
    elements.transcriptSize.textContent = `${tab.transcript.length} records`
  }

  renderEventMeta(tab)
  renderEvents(tab)
  renderError(tab)
  emitSessionUpdate(tab)
  updateSessionActions()
  updateDrawerIndicator()
}

function updateSessionActions() {
  const button = elements.signOutAllSessions
  if (!button) return
  if (button.dataset.busy === 'true') {
    button.disabled = true
    return
  }
  const hasActiveSessions = state.tabs.some((entry) => entry.session && entry.phase !== 'closed')
  button.disabled = !hasActiveSessions
}

function renderError(tab) {
  if (!elements.errorBanner) return
  const message = tab ? tab.errorMessage : ''
  if (message) {
    elements.errorBanner.textContent = message
    elements.errorBanner.classList.remove('hidden')
  } else {
    elements.errorBanner.textContent = ''
    elements.errorBanner.classList.add('hidden')
  }
}

function renderEvents(tab) {
  if (!elements.eventFeed) return
  elements.eventFeed.innerHTML = ''
  if (!tab) return
  const recent = tab.events.slice(-50).reverse()
  recent.forEach((event) => {
    const li = document.createElement('li')
    const time = document.createElement('time')
    time.textContent = new Date(event.timestamp).toLocaleTimeString()
    const label = document.createElement('div')
    label.textContent = event.type
    label.style.fontWeight = '600'
    li.appendChild(time)
    li.appendChild(label)
    if (event.payload !== undefined) {
      const detail = document.createElement('pre')
      detail.className = 'event-detail'
      detail.textContent = formatEventPayload(event.payload)
      li.appendChild(detail)
    }
    elements.eventFeed.appendChild(li)
  })
}

function renderEventMeta(tab) {
  if (!elements.eventMeta) return
  if (!tab) {
    elements.eventMeta.textContent = ''
    elements.eventMeta.classList.add('hidden')
    return
  }
  const summaries = Object.entries(tab.suppressed)
    .filter(([, count]) => count > 0)
    .map(([type, count]) => `${SUPPRESSED_EVENT_LABELS[type] || type}: ${count}`)

  if (summaries.length === 0) {
    elements.eventMeta.textContent = ''
    elements.eventMeta.classList.add('hidden')
  } else {
    elements.eventMeta.textContent = summaries.join(' • ')
    elements.eventMeta.classList.remove('hidden')
  }
}

function appendEvent(tab, type, payload) {
  if (!tab) return
  const normalized = payload instanceof Error ? { message: payload.message, stack: payload.stack } : payload
  if (AGGREGATED_EVENT_TYPES.has(type)) {
    recordSuppressedEvent(tab, type)
    return
  }
  tab.events.push({ type, payload: normalized, timestamp: Date.now() })
  if (tab.events.length > 500) {
    tab.events.splice(0, tab.events.length - 500)
  }
  if (!state.drawer.open && tab.id === state.activeTabId) {
    const nextCount = (state.drawer.unreadCount || 0) + 1
    state.drawer.unreadCount = nextCount > 999 ? 999 : nextCount
  }
  if (tab.id === state.activeTabId) {
    updateUI()
  }
}

function recordSuppressedEvent(tab, type) {
  if (!tab) return
  const current = (tab.suppressed[type] || 0) + 1
  tab.suppressed[type] = current
  if (tab.id === state.activeTabId && (current === 1 || current % 25 === 0)) {
    renderEventMeta(tab)
  }
}

function recordTranscript(tab, entry) {
  if (!tab) return
  tab.transcript.push(entry)
  if (tab.transcript.length > 5000) {
    tab.transcript.splice(0, tab.transcript.length - 5000)
  }
  if (tab.id === state.activeTabId) {
    updateUI()
  }
}

function showError(tab, message) {
  if (!tab) return
  tab.errorMessage = message || ''
  if (tab.id === state.activeTabId) {
    renderError(tab)
  }
}

async function startSession(tab, options = {}) {
  if (!tab || tab.phase === 'creating' || tab.phase === 'running') return
  showError(tab, '')
  setTabPhase(tab, 'creating')
  setTabSocketState(tab, 'connecting')

  try {
    const payload = {}
    if (options.reason) payload.reason = options.reason
    if (options.command) payload.command = options.command
    if (Array.isArray(options.args) && options.args.length > 0) payload.args = options.args
    if (options.metadata) payload.metadata = options.metadata
    if (tab.id) payload.tabId = tab.id

    const response = await proxyToApi('/api/v1/sessions', {
      method: 'POST',
      json: payload
    })
    if (!response.ok) {
      const text = await response.text()
      throw new Error(text || `API error (${response.status})`)
    }
    const data = await response.json()
    const sessionArgs = Array.isArray(data.args) ? [...data.args] : Array.isArray(options.args) ? [...options.args] : []
    const sessionCommand = data.command || options.command || ''
    tab.session = {
      ...data,
      command: sessionCommand,
      args: sessionArgs,
      commandLine: formatCommandLabel(sessionCommand, sessionArgs)
    }
    tab.transcript = []
    tab.events = []
    tab.suppressed = initialSuppressedState()
    tab.pendingWrites = Array.isArray(tab.pendingWrites) ? tab.pendingWrites : []
    tab.errorMessage = ''
    tab.lastSentSize = { cols: 0, rows: 0 }
    tab.term.reset()
    renderEventMeta(tab)
    appendEvent(tab, 'session-created', {
      ...data,
      reason: options.reason || null,
      command: tab.session.command,
      args: tab.session.args
    })
    connectWebSocket(tab, data.id)
    setTabPhase(tab, 'running')
    state.bridge?.emit('session-started', { ...data, tabId: tab.id })
  } catch (error) {
    setTabPhase(tab, 'idle')
    setTabSocketState(tab, 'disconnected')
    showError(tab, error instanceof Error ? error.message : 'Unknown error starting session')
    appendEvent(tab, 'session-error', error)
  }
}

async function stopSession(tab) {
  if (!tab || !tab.session) return
  if (tab.phase === 'closing') return
  setTabPhase(tab, 'closing')
  setTabSocketState(tab, 'closing')
  appendEvent(tab, 'session-stop-requested', { id: tab.session.id })
  try {
    await proxyToApi(`/api/v1/sessions/${tab.session.id}`, { method: 'DELETE' })
  } catch (error) {
    appendEvent(tab, 'session-stop-error', error)
  } finally {
    try {
      tab.socket?.close()
    } catch (_error) {
      // ignore
    }
    tab.socket = null
    setTabPhase(tab, 'closed')
    setTabSocketState(tab, 'disconnected')
    state.bridge?.emit('session-ended', { id: tab.session.id, tabId: tab.id })
  }
}

async function handleSignOutAllSessions() {
  const button = elements.signOutAllSessions
  if (!button || button.dataset.busy === 'true') return

  const sessions = state.tabs.filter((tab) => tab.session && tab.phase !== 'closed')
  if (sessions.length === 0) {
    button.blur()
    return
  }

  const confirmMessage =
    sessions.length === 1
      ? 'Sign out the active session? This will close the terminal.'
      : `Sign out all ${sessions.length} sessions? This will close every terminal.`
  if (!window.confirm(confirmMessage)) {
    return
  }

  const originalLabel = button.textContent || ''
  button.dataset.busy = 'true'
  button.dataset.label = originalLabel
  button.disabled = true
  button.textContent = 'Signing out…'

  sessions.forEach((tab) => {
    appendEvent(tab, 'session-stop-all-requested', { id: tab.session.id })
    setTabPhase(tab, 'closing')
    const socketOpen = tab.socket && tab.socket.readyState === WebSocket.OPEN
    setTabSocketState(tab, socketOpen ? 'closing' : 'disconnected')
  })

  try {
    const response = await proxyToApi('/api/v1/sessions', { method: 'DELETE' })
    if (!response.ok) {
      const text = await response.text()
      throw new Error(text || `API error (${response.status})`)
    }

    let payload
    const contentType = response.headers.get('Content-Type') || ''
    if (contentType.includes('application/json')) {
      try {
        payload = await response.json()
      } catch (_error) {
        payload = null
      }
    }

    const eventPayload =
      payload && typeof payload === 'object'
        ? payload
        : { status: 'terminating_all', terminated: sessions.length }

    sessions.forEach((tab) => {
      appendEvent(tab, 'session-stop-all', eventPayload)
      if (tab.socket) {
        try {
          tab.socket.close()
        } catch (_error) {
          // ignore close failures
        }
      }
    })
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unable to sign out sessions'
    const activeTab = getActiveTab()
    if (activeTab) {
      showError(activeTab, message)
    }
    sessions.forEach((tab) => {
      appendEvent(tab, 'session-stop-all-error', error)
      if (tab.phase === 'closing') {
        setTabPhase(tab, tab.session ? 'running' : 'idle')
      }
      if (tab.socketState === 'closing') {
        const socketOpen = tab.socket && tab.socket.readyState === WebSocket.OPEN
        setTabSocketState(tab, socketOpen ? 'open' : 'disconnected')
      }
    })
  } finally {
    const label = button.dataset.label || 'Sign out all sessions'
    button.textContent = label
    delete button.dataset.label
    delete button.dataset.busy
    button.disabled = false
    updateSessionActions()
  }
}

function connectWebSocket(tab, sessionId) {
  const url = buildWebSocketUrl(`/ws/sessions/${sessionId}/stream`)
  const socket = new WebSocket(url)
  tab.socket = socket

  socket.addEventListener('open', () => {
    setTabSocketState(tab, 'open')
    appendEvent(tab, 'ws-open', { sessionId })
    if (typeof tab.term.cols === 'number' && typeof tab.term.rows === 'number') {
      sendResize(tab, tab.term.cols, tab.term.rows)
    }
    flushPendingWrites(tab)
  })

  socket.addEventListener('message', async (event) => {
    const raw = await normalizeSocketData(event.data, tab)
    if (!raw || raw.trim().length === 0) {
      recordSuppressedEvent(tab, 'ws-empty-frame')
      return
    }
    try {
      const envelope = JSON.parse(raw)
      handleStreamEnvelope(tab, envelope)
    } catch (error) {
      appendEvent(tab, 'ws-message-error', {
        message: error instanceof Error ? error.message : String(error),
        raw
      })
      showError(tab, 'Failed to process stream message')
    }
  })

  socket.addEventListener('error', (error) => {
    setTabSocketState(tab, 'error')
    showError(tab, 'WebSocket error occurred')
    appendEvent(tab, 'ws-error', error)
  })

  socket.addEventListener('close', (event) => {
    setTabSocketState(tab, 'disconnected')
    appendEvent(tab, 'ws-close', { code: event.code, reason: event.reason })
    if (tab.phase === 'running' || tab.phase === 'closing') {
      setTabPhase(tab, 'closed')
    }
    tab.socket = null
    state.bridge?.emit('session-ended', { id: sessionId, reason: 'ws-close', tabId: tab.id })
  })
}

function handleStreamEnvelope(tab, envelope) {
  if (!tab || !envelope || typeof envelope.type !== 'string') return

  switch (envelope.type) {
    case 'output':
      handleOutputPayload(tab, envelope.payload)
      break
    case 'status':
      handleStatusPayload(tab, envelope.payload)
      break
    case 'heartbeat':
      appendEvent(tab, 'session-heartbeat', envelope.payload)
      break
    default:
      appendEvent(tab, 'session-envelope', envelope)
  }
}

function handleOutputPayload(tab, payload) {
  if (!tab || !payload || typeof payload.data !== 'string') return

  let text = payload.data
  if (payload.encoding === 'base64') {
    try {
      const bytes = Uint8Array.from(atob(payload.data), (c) => c.charCodeAt(0))
      text = textDecoder.decode(bytes)
    } catch (error) {
      appendEvent(tab, 'decode-error', error)
      return
    }
  }

  tab.term.write(text)
  recordTranscript(tab, {
    timestamp: payload.timestamp ? Date.parse(payload.timestamp) : Date.now(),
    direction: payload.direction || 'stdout',
    encoding: payload.encoding,
    data: payload.data
  })
}

function handleStatusPayload(tab, payload) {
  if (!tab) return
  appendEvent(tab, 'session-status', payload)

  if (payload && typeof payload === 'object') {
    recordTranscript(tab, {
      timestamp: payload.timestamp ? Date.parse(payload.timestamp) : Date.now(),
      direction: 'status',
      message: JSON.stringify(payload)
    })

    if (payload.status === 'started') {
      setTabPhase(tab, 'running')
      setTabSocketState(tab, 'open')
      showError(tab, '')
      return
    }

    if (payload.status === 'command_exit_error') {
      setTabPhase(tab, 'closed')
      setTabSocketState(tab, 'disconnected')
      showError(tab, `Command exited: ${payload.reason || 'unknown error'}`)
      notifyPanic(tab, payload)
      return
    }

    if (payload.status === 'closed') {
      setTabPhase(tab, 'closed')
      setTabSocketState(tab, 'disconnected')
      if (payload.reason && !payload.reason.includes('client_requested')) {
        showError(tab, `Session closed: ${payload.reason}`)
      }
      return
    }

    return
  }

  recordTranscript(tab, {
    timestamp: Date.now(),
    direction: 'status',
    message: JSON.stringify(payload)
  })
}

function notifyPanic(tab, payload) {
  if (!tab) return
  const reason = payload && typeof payload === 'object' && payload.reason ? payload.reason : 'unknown error'
  tab.term.write(`\r\n\u001b[31mCommand exited\u001b[0m: ${reason}\r\n`)
  tab.term.write('Review the event feed or transcripts for details.\r\n')
}

function transmitInput(tab, value, meta = {}) {
  if (!tab || !tab.socket || tab.socket.readyState !== WebSocket.OPEN) {
    return false
  }
  if (typeof value !== 'string' || value.length === 0) {
    return false
  }

  const shouldAppendNewline = meta.appendNewline === true
  const normalized = shouldAppendNewline ? (value.endsWith('\n') ? value : `${value}\n`) : value
  if (!normalized) return true

  const payload = {
    type: 'input',
    payload: {
      data: normalized,
      encoding: 'utf-8'
    }
  }

  try {
    tab.socket.send(JSON.stringify(payload))
  } catch (error) {
    appendEvent(tab, 'stdin-send-error', error)
    return false
  }

  recordTranscript(tab, {
    timestamp: Date.now(),
    direction: 'stdin',
    encoding: 'utf-8',
    data: btoa(payload.payload.data)
  })

  if (meta.eventType) {
    appendEvent(tab, meta.eventType, {
      length: payload.payload.data.length,
      source: meta.source || undefined,
      command: meta.command || undefined
    })
  }

  if (meta.clearError !== false) {
    showError(tab, '')
  }

  return true
}

function sendResize(tab, cols, rows) {
  if (!tab || !tab.socket || tab.socket.readyState !== WebSocket.OPEN) {
    return
  }
  if (!Number.isInteger(cols) || !Number.isInteger(rows) || cols <= 0 || rows <= 0) {
    return
  }
  const { cols: lastCols, rows: lastRows } = tab.lastSentSize
  if (cols === lastCols && rows === lastRows) {
    return
  }
  const payload = {
    type: 'resize',
    payload: { cols, rows }
  }
  try {
    tab.socket.send(JSON.stringify(payload))
    tab.lastSentSize = { cols, rows }
    appendEvent(tab, 'terminal-resize', { cols, rows })
  } catch (error) {
    appendEvent(tab, 'terminal-resize-error', error)
  }
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

function proxyToApi(path, { method = 'GET', json, headers } = {}) {
  const targetPath = path.startsWith('/api') ? path : `/api${path.startsWith('/') ? path : `/${path}`}`
  const requestInit = { method, headers: headers ? new Headers(headers) : new Headers() }
  if (json !== undefined) {
    requestInit.headers.set('Content-Type', 'application/json')
    requestInit.body = JSON.stringify(json)
  }
  return fetch(targetPath, requestInit)
}

function buildWebSocketUrl(path) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const normalized = path.startsWith('/') ? path : `/${path}`
  return `${protocol}//${host}${normalized}`
}

function isArrayBufferView(value) {
  return value && typeof value === 'object' && ArrayBuffer.isView(value)
}

async function normalizeSocketData(data, tab) {
  if (typeof data === 'string') {
    return data
  }
  try {
    if (data instanceof Blob && typeof data.text === 'function') {
      return await data.text()
    }
    if (data instanceof ArrayBuffer) {
      return textDecoder.decode(data)
    }
    if (isArrayBufferView(data)) {
      const view = data
      const buffer = view.byteOffset === 0 && view.byteLength === view.buffer.byteLength
        ? view.buffer
        : view.buffer.slice(view.byteOffset, view.byteOffset + view.byteLength)
      return textDecoder.decode(buffer)
    }
  } catch (error) {
    if (tab) {
      appendEvent(tab, 'ws-decode-error', error)
    }
  }
  return ''
}

function initializeIframeBridge() {
  const CHANNEL = 'web-console'
  const emit = (type, payload) => {
    if (window.parent) {
      window.parent.postMessage({ channel: CHANNEL, type, payload }, '*')
    }
  }

  window.addEventListener('message', async (event) => {
    const message = event.data
    if (!message || message.channel !== CHANNEL) {
      return
    }
    try {
      switch (message.type) {
        case 'init-session':
          emit('ready', { timestamp: Date.now() })
          break
        case 'end-session': {
          const tab = getActiveTab()
          if (tab) {
            await stopSession(tab)
          }
          emit('session-ended', { requestedByParent: true })
          break
        }
        case 'request-screenshot':
          if (window.html2canvas) {
            const canvas = await window.html2canvas(document.body, { backgroundColor: '#0f172a' })
            emit('screenshot', { image: canvas.toDataURL('image/png', 0.9), requestedAt: Date.now() })
          } else {
            emit('error', { type: 'request-screenshot', message: 'html2canvas not available' })
          }
          break
        case 'request-transcript': {
          const tab = getActiveTab()
          emit('transcript', { transcript: tab ? tab.transcript : [], requestedAt: Date.now(), tabId: tab?.id || null })
          break
        }
        case 'request-logs': {
          const tab = getActiveTab()
          emit('logs', { logs: tab ? tab.events.slice(-100) : [], requestedAt: Date.now(), tabId: tab?.id || null })
          break
        }
        default:
          emit('error', { type: 'unknown-command', payload: message })
      }
    } catch (error) {
      emit('error', { type: message.type, message: error instanceof Error ? error.message : 'unknown error' })
    }
  })

  emit('bridge-initialized', { timestamp: Date.now() })
  return { emit }
}

function formatEventPayload(payload) {
  if (payload === undefined || payload === null) {
    return ''
  }
  if (typeof payload === 'string') {
    return payload
  }
  try {
    return JSON.stringify(payload, null, 2)
  } catch (_error) {
    return '[unserializable payload]'
  }
}

// Workspace WebSocket connection
function connectWorkspaceWebSocket() {
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
      // Handle different data types (string, Blob, ArrayBuffer)
      let rawData = event.data
      if (rawData instanceof Blob) {
        rawData = await rawData.text()
      } else if (rawData instanceof ArrayBuffer) {
        rawData = new TextDecoder().decode(rawData)
      }

      const data = JSON.parse(rawData)

      // Ignore status messages from proxy
      if (data.type === 'status' && data.payload?.status === 'upstream_connected') {
        return
      }

      // First message might be the initial workspace state (not wrapped in event)
      if (data.activeTabId !== undefined && data.tabs !== undefined) {
        console.log('Received initial workspace state')
        return // We already loaded workspace via REST, ignore this
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

// Handle workspace events from server
function handleWorkspaceEvent(event) {
  if (!event || !event.type) return

  switch (event.type) {
    case 'workspace-full-update':
      handleWorkspaceFullUpdate(event.payload)
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
      handleKeyboardToolbarModeChanged(event.payload)
      break
    default:
      console.log('Unknown workspace event:', event.type)
  }
}

function handleWorkspaceFullUpdate(payload) {
  // TODO: Implement full workspace sync
  console.log('Full workspace update:', payload)
}

function handleTabAdded(payload) {
  // Check if tab already exists locally
  const existing = findTab(payload.id)
  if (existing) return

  // Create tab locally
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
    updateUI()
  }
}

function handleTabRemoved(payload) {
  const tab = findTab(payload.id)
  if (!tab) return

  // Remove tab locally (but don't sync back to server)
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
  } else {
    updateUI()
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

  // Reconnect to session if needed
  if (!tab.session || tab.session.id !== payload.sessionId) {
    reconnectSession(tab, payload.sessionId)
  }
}

function handleSessionDetached(payload) {
  const tab = findTab(payload.tabId)
  if (!tab) return

  // Mark session as detached
  if (tab.session) {
    tab.session = null
    setTabPhase(tab, 'idle')
    setTabSocketState(tab, 'disconnected')
  }
}

function handleKeyboardToolbarModeChanged(payload) {
  // Deprecated - keyboard toolbar mode has been removed
  console.log('Keyboard toolbar mode changed event (deprecated):', payload)
}

async function reconnectSession(tab, sessionId) {
  try {
    const response = await proxyToApi(`/api/v1/sessions/${sessionId}`)
    if (!response.ok) {
      console.error('Failed to fetch session:', sessionId)
      return
    }
    const data = await response.json()
    tab.session = {
      ...data,
      commandLine: formatCommandLabel(data.command, data.args)
    }
    connectWebSocket(tab, sessionId)
    setTabPhase(tab, 'running')
  } catch (error) {
    console.error('Failed to reconnect session:', error)
  }
}


// Get current workspace state
async function getWorkspaceState() {
  try {
    const response = await proxyToApi('/api/v1/workspace')
    if (!response.ok) return null
    return await response.json()
  } catch (error) {
    console.error('Failed to get workspace state:', error)
    return null
  }
}

// AI Command Generation initialization
if (typeof window !== 'undefined' && document.readyState === 'complete' || document.readyState === 'interactive') {
  initAICommandAsync()
} else {
  window.addEventListener('DOMContentLoaded', initAICommandAsync)
}

async function initAICommandAsync() {
  try {
    const { generateAICommand, showSnackbar, initializeMobileToolbar } = await import('./modules/mobile-toolbar.js')

    // Helper function to get active tab
    const getActiveTabFn = () => getActiveTab()

    // Helper function to send keys to active terminal
    const sendKeyToTerminalFn = (key) => {
      const tab = getActiveTab()
      if (!tab) {
        console.warn('No active terminal to send key to')
        return
      }

      // Send key directly to terminal via WebSocket if connected
      if (tab.socket && tab.socket.readyState === WebSocket.OPEN) {
        transmitInput(tab, key, { appendNewline: false, clearError: true })
      } else {
        // Queue for when session starts
        queueInput(tab, key, { appendNewline: false })
        if (tab.phase === 'idle' || tab.phase === 'closed') {
          startSession(tab, { reason: 'ai-command-input' }).catch((error) => {
            appendEvent(tab, 'session-error', error)
            showError(tab, error instanceof Error ? error.message : 'Unable to start terminal session')
          })
        }
      }
    }

    // Initialize mobile toolbar
    const mobileToolbar = initializeMobileToolbar(getActiveTabFn, sendKeyToTerminalFn)

    // Set up AI command input handlers
    if (elements.aiGenerateBtn && elements.aiCommandInput) {
      elements.aiGenerateBtn.addEventListener('click', async () => {
        const prompt = elements.aiCommandInput.value.trim()
        if (!prompt) return

        // Show loading state
        const iconElement = elements.aiGenerateBtn.querySelector('.ai-icon')
        const originalIcon = iconElement ? iconElement.outerHTML : ''
        elements.aiGenerateBtn.disabled = true
        elements.aiGenerateBtn.classList.add('loading')
        elements.aiGenerateBtn.innerHTML = '<span class="loading-spinner"></span>'

        try {
          const result = await generateAICommand(prompt, getActiveTabFn, sendKeyToTerminalFn)

          if (result.success) {
            // Clear input on success
            elements.aiCommandInput.value = ''
            showSnackbar('Command generated successfully', 'success', 2000)
          } else {
            showSnackbar(result.error || 'Failed to generate command', 'error', 4000)
          }
        } finally {
          elements.aiGenerateBtn.disabled = false
          elements.aiGenerateBtn.classList.remove('loading')
          elements.aiGenerateBtn.innerHTML = originalIcon
          // Re-initialize lucide icons after replacing HTML
          if (window.lucide) {
            lucide.createIcons()
          }
        }
      })

      // Allow Enter key to trigger generation
      elements.aiCommandInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          e.preventDefault()
          elements.aiGenerateBtn.click()
        }
      })

      console.log('AI command generation initialized successfully')
    }

    // Set up debug toolbar toggle
    const debugToolbarToggle = document.getElementById('debugToolbarToggle')
    if (debugToolbarToggle && mobileToolbar) {
      debugToolbarToggle.addEventListener('change', (e) => {
        const isEnabled = e.target.checked
        // Persist state to localStorage
        localStorage.setItem('debugToolbarEnabled', isEnabled.toString())

        if (isEnabled) {
          // Enable floating mode with debug flag to override desktop behavior
          mobileToolbar.setMode('floating', true)
          mobileToolbar.show()
        } else {
          mobileToolbar.setMode('disabled', false)
          mobileToolbar.hide()
        }
      })
    }
  } catch (error) {
    console.error('Failed to initialize AI command generation:', error)
  }
}

