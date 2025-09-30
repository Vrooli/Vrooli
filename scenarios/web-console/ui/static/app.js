const elements = {
  statusBadge: document.getElementById('statusBadge'),
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
  drawerIndicator: document.getElementById('drawerIndicator'),
  detailsDrawer: document.getElementById('detailsDrawer'),
  drawerBackdrop: document.getElementById('drawerBackdrop')
}

const shortcutButtons = Array.from(document.querySelectorAll('[data-shortcut-id]'))

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
  drawer: {
    open: false,
    unreadCount: 0,
    previousFocus: null
  }
}

let tabSequence = 0

state.bridge = initializeIframeBridge()

const initialTab = createTerminalTab({ focus: true })
if (initialTab) {
  startSession(initialTab, { reason: 'initial-tab' }).catch((error) => {
    appendEvent(initialTab, 'session-error', error)
    showError(initialTab, error instanceof Error ? error.message : 'Unable to start terminal session')
  })
}

elements.addTabBtn.addEventListener('click', () => {
  const tab = createTerminalTab({ focus: true })
  if (tab) {
    startSession(tab, { reason: 'new-tab' }).catch((error) => {
      appendEvent(tab, 'session-error', error)
      showError(tab, error instanceof Error ? error.message : 'Unable to start terminal session')
    })
  }
})

shortcutButtons.forEach((button) => {
  button.addEventListener('click', () => handleShortcutButton(button))
})

if (elements.drawerToggle) {
  elements.drawerToggle.addEventListener('click', () => toggleDrawer())
}

if (elements.drawerBackdrop) {
  elements.drawerBackdrop.addEventListener('click', () => closeDrawer())
}

document.addEventListener('keydown', (event) => {
  if (event.key === 'Escape' && state.drawer.open) {
    event.preventDefault()
    closeDrawer()
  }
})

window.addEventListener('resize', () => {
  const active = getActiveTab()
  if (!active) return
  requestAnimationFrame(() => {
    active.fitAddon?.fit()
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
    elements.detailsDrawer?.focus()
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

function createTerminalTab({ focus = false } = {}) {
  if (!elements.terminalHost) return null
  const id = `tab-${Date.now()}-${++tabSequence}`
  const label = `Terminal ${tabSequence}`

  const container = document.createElement('div')
  container.className = 'terminal-screen'
  container.dataset.tabId = id

  elements.terminalHost.appendChild(container)

  const term = new window.Terminal({ ...terminalDefaults })
  const fitAddon = new window.FitAddon.FitAddon()
  term.loadAddon(fitAddon)
  term.open(container)
  fitAddon.fit()

  const tab = {
    id,
    label,
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
    domLabel: null
  }

  term.onResize(({ cols, rows }) => {
    sendResize(tab, cols, rows)
  })

  term.onData((data) => {
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
    updateTabButtonState(tab)
  })
}

async function closeTab(tabId) {
  const tab = findTab(tabId)
  if (!tab) return

  await stopSession(tab)

  destroyTerminalTab(tab)

  state.tabs = state.tabs.filter((entry) => entry.id !== tabId)

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
    if (elements.statusBadge) {
      elements.statusBadge.textContent = 'NO ACTIVE TAB'
      elements.statusBadge.style.color = '#38bdf8'
    }
    if (elements.sessionId) elements.sessionId.textContent = '—'
    if (elements.sessionPhase) elements.sessionPhase.textContent = 'idle'
    if (elements.socketState) elements.socketState.textContent = 'disconnected'
    if (elements.sessionCommand) elements.sessionCommand.textContent = '—'
    if (elements.transcriptSize) elements.transcriptSize.textContent = '0 records'
    renderEventMeta(null)
    renderEvents(null)
    renderError(null)
    emitSessionUpdate(null)
    resetUnreadEvents()
    return
  }

  const phase = tab.phase || 'idle'
  const socketState = tab.socketState || 'disconnected'
  const badgeColor = phase === 'running' ? '#22c55e' : phase === 'closing' ? '#f97316' : '#38bdf8'

  if (elements.statusBadge) {
    elements.statusBadge.textContent = `${phase.toUpperCase()} · ${socketState.toUpperCase()}`
    elements.statusBadge.style.color = badgeColor
  }

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
  updateDrawerIndicator()
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
