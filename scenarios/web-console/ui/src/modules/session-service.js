import { elements, state } from './state.js'
import {
  appendEvent,
  recordTranscript,
  renderEventMeta,
  renderEvents,
  renderError,
  showError
} from './event-feed.js'
import {
  queueInput,
  flushPendingWrites,
  transmitInput,
  sendResize,
  handleTerminalData as bufferHandleTerminalData,
  filterLocalEcho
} from './input-buffer.js'
import {
  proxyToApi,
  buildWebSocketUrl,
  normalizeSocketData,
  formatCommandLabel,
  parseApiError,
  textDecoder
} from './utils.js'
import { debugLog } from './debug.js'
import { showToast } from './notifications.js'
import {
  getActiveTab,
  renderTabs,
  applyTabAppearance,
  getDetachedTabs,
  createTerminalTab,
  refreshTabButton
} from './tab-manager.js'

const callbacks = {
  onActiveTabNeedsUpdate: null,
  onSessionActionsChanged: null,
  queueSessionOverviewRefresh: null
}

export function configureSessionService(options = {}) {
  callbacks.onActiveTabNeedsUpdate = typeof options.onActiveTabNeedsUpdate === 'function'
    ? options.onActiveTabNeedsUpdate
    : null
  callbacks.onSessionActionsChanged = typeof options.onSessionActionsChanged === 'function'
    ? options.onSessionActionsChanged
    : null
  callbacks.queueSessionOverviewRefresh = typeof options.queueSessionOverviewRefresh === 'function'
    ? options.queueSessionOverviewRefresh
    : null
}

function notifyActiveTabUpdate() {
  callbacks.onActiveTabNeedsUpdate?.()
}

function notifySessionActionsChanged() {
  callbacks.onSessionActionsChanged?.()
}

function queueSessionOverviewRefresh(delay = 0) {
  callbacks.queueSessionOverviewRefresh?.(delay)
}

export function setTabPhase(tab, phase) {
  if (!tab) return
  tab.phase = phase
  refreshTabButton(tab)
  if (tab.id === state.activeTabId) {
    notifyActiveTabUpdate()
  }
}

export function setTabSocketState(tab, socketState) {
  if (!tab) return
  tab.socketState = socketState
  refreshTabButton(tab)
  if (tab.id === state.activeTabId) {
    notifyActiveTabUpdate()
  }
}

export async function startSession(tab, options = {}) {
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
      let text = ''
      try {
        text = await response.text()
      } catch (_error) {
        text = ''
      }
      const { message, raw } = parseApiError(text, `API error (${response.status})`)
      const error = new Error(message)
      error.status = response.status
      if (raw) {
        error.payload = raw
      }
      throw error
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
    tab.wasDetached = false
    refreshTabButton(tab)
    tab.transcript = []
    tab.transcriptByteSize = 0
    tab.events = []
    tab.suppressed = tab.suppressed || {}
    Object.keys(tab.suppressed).forEach((key) => {
      tab.suppressed[key] = 0
    })
    tab.pendingWrites = Array.isArray(tab.pendingWrites) ? tab.pendingWrites : []
    tab.errorMessage = ''
    tab.lastSentSize = { cols: 0, rows: 0 }
    tab.inputSeq = 0
    tab.replayPending = false
    tab.replayComplete = true
    tab.lastReplayCount = 0
    tab.lastReplayTruncated = false
    tab.hasReceivedLiveOutput = false
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
    queueSessionOverviewRefresh(250)
    notifySessionActionsChanged()
  } catch (error) {
    setTabPhase(tab, 'idle')
    setTabSocketState(tab, 'disconnected')
    const message = error instanceof Error ? error.message : 'Unknown error starting session'
    showError(tab, message)
    appendEvent(tab, 'session-error', error)
    tab.wasDetached = true
    refreshTabButton(tab)
    notifySessionActionsChanged()
    if (typeof message === 'string' && message.toLowerCase().includes('capacity reached')) {
      queueSessionOverviewRefresh(100)
      await showToast('Terminal pool is full. Sign out stale sessions or wait a few seconds.', 'error', 4000)
    }
  }
}

export async function stopSession(tab) {
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
    tab.wasDetached = true
    refreshTabButton(tab)
    queueSessionOverviewRefresh(250)
    notifySessionActionsChanged()
  }
}

export async function reconnectSession(tab, sessionId) {
  if (!tab || !sessionId) {
    return
  }
  if (tab.reconnecting) {
    return
  }

  tab.reconnecting = true
  try {
    setTabSocketState(tab, 'connecting')
    const response = await proxyToApi(`/api/v1/sessions/${sessionId}`)
    if (!response.ok) {
      const status = response.status
      appendEvent(tab, 'session-reconnect-miss', { sessionId, status })
      if (status === 404) {
        tab.session = null
        setTabPhase(tab, 'idle')
        setTabSocketState(tab, 'disconnected')
        showError(tab, 'Session expired. Type to launch a new shell.')
        tab.wasDetached = true
        refreshTabButton(tab)
        queueSessionOverviewRefresh(250)
      } else {
        setTabSocketState(tab, 'error')
        showError(tab, `Unable to reconnect (status ${status})`)
        tab.wasDetached = true
        refreshTabButton(tab)
      }
      notifySessionActionsChanged()
      return
    }
    const data = await response.json()
    tab.session = {
      ...data,
      commandLine: formatCommandLabel(data.command, data.args)
    }
    tab.wasDetached = false
    refreshTabButton(tab)
    connectWebSocket(tab, sessionId)
    setTabPhase(tab, 'running')
    showError(tab, '')
    notifySessionActionsChanged()
  } catch (error) {
    appendEvent(tab, 'session-reconnect-error', {
      sessionId,
      message: error instanceof Error ? error.message : String(error)
    })
    setTabSocketState(tab, 'error')
    showError(tab, 'Unable to reconnect to session')
    tab.wasDetached = true
    refreshTabButton(tab)
    notifySessionActionsChanged()
  } finally {
    tab.reconnecting = false
  }
}

export function handleTerminalData(tab, data) {
  bufferHandleTerminalData(tab, data, ensureSessionForPendingInput)
}

export function ensureSessionForPendingInput(tab, reason) {
  if (!tab) return

  if (tab.session && tab.session.id && !tab.socket) {
    if (tab.reconnecting) {
      showError(tab, 'Reconnecting to session…')
      return
    }
    showError(tab, 'Reconnecting to session…')
    reconnectSession(tab, tab.session.id).catch((error) => {
      appendEvent(tab, 'reconnect-on-input-failed', error)
      if (tab.phase === 'idle' || tab.phase === 'closed') {
        showError(tab, 'Starting new session…')
        startSession(tab, { reason: reason || 'auto-start:input-after-reconnect-fail' }).catch((startError) => {
          appendEvent(tab, 'session-error', startError)
          showError(tab, startError instanceof Error ? startError.message : 'Unable to start terminal session')
        })
      }
    })
    return
  }

  if (tab.phase === 'idle' || tab.phase === 'closed') {
    showError(tab, 'Starting new session…')
    startSession(tab, { reason: reason || 'auto-start:input' }).catch((error) => {
      appendEvent(tab, 'session-error', error)
      showError(tab, error instanceof Error ? error.message : 'Unable to start terminal session')
    })
    return
  }

  if (!tab.socket || tab.socket.readyState !== WebSocket.OPEN) {
    showError(tab, 'Connecting…')
  }
}

export function queueInputForTab(tab, value, meta) {
  queueInput(tab, value, meta)
}

export function flushPendingWritesForTab(tab) {
  flushPendingWrites(tab)
}

export function transmitInputForTab(tab, value, meta) {
  return transmitInput(tab, value, meta)
}

export function sendResizeForTab(tab, cols, rows) {
  sendResize(tab, cols, rows)
}

export function connectWebSocket(tab, sessionId) {
  if (!tab || !sessionId) {
    return
  }

  const url = buildWebSocketUrl(`/ws/sessions/${sessionId}/stream`)
  const previousSocket = tab.socket
  const socket = new WebSocket(url)
  tab.inputSeq = 0
  tab.socket = socket
  tab.replayPending = true
  tab.replayComplete = false
  tab.lastReplayCount = 0
  tab.lastReplayTruncated = false

  if (previousSocket && previousSocket !== socket) {
    try {
      previousSocket.close()
    } catch (error) {
      appendEvent(tab, 'ws-close-error', {
        message: error instanceof Error ? error.message : String(error)
      })
    }
  }

  socket.addEventListener('open', () => {
    if (tab.socket !== socket) {
      return
    }
    setTabSocketState(tab, 'open')
    appendEvent(tab, 'ws-open', { sessionId })
    if (typeof tab.term.cols === 'number' && typeof tab.term.rows === 'number') {
      sendResize(tab, tab.term.cols, tab.term.rows)
    }
    flushPendingWrites(tab)

    if (tab.heartbeatInterval) {
      clearInterval(tab.heartbeatInterval)
    }
    tab.heartbeatInterval = setInterval(() => {
      if (tab.socket === socket && socket.readyState === WebSocket.OPEN) {
        try {
          socket.send(JSON.stringify({ type: 'heartbeat', payload: {} }))
        } catch (error) {
          console.warn('Heartbeat send failed:', error)
        }
      }
    }, 30000)
  })

  socket.addEventListener('message', async (event) => {
    if (tab.socket !== socket) {
      return
    }
    const raw = await normalizeSocketData(event.data, (error) => {
      appendEvent(tab, 'ws-decode-error', error)
    })
    if (!raw || raw.trim().length === 0) {
      appendEvent(tab, 'ws-empty-frame', null)
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
    if (tab.socket !== socket) {
      return
    }
    setTabSocketState(tab, 'error')
    showError(tab, 'WebSocket error occurred')
    appendEvent(tab, 'ws-error', error)
  })

  socket.addEventListener('close', (event) => {
    if (tab.socket !== socket) {
      return
    }

    if (tab.heartbeatInterval) {
      clearInterval(tab.heartbeatInterval)
      tab.heartbeatInterval = null
    }

    setTabSocketState(tab, 'disconnected')
    appendEvent(tab, 'ws-close', { code: event.code, reason: event.reason, sessionId })

    tab.socket = null

    const isExplicitClose = tab.phase === 'closing'

    if (isExplicitClose) {
      setTabPhase(tab, 'closed')
    } else if (tab.session && tab.phase === 'running') {
      appendEvent(tab, 'ws-reconnecting', { sessionId, attempt: 1, code: event.code })
      setTimeout(() => {
        if (tab.session && tab.session.id === sessionId && !tab.socket) {
          reconnectSession(tab, sessionId).catch((error) => {
            appendEvent(tab, 'ws-reconnect-failed', error)
            setTabPhase(tab, 'closed')
            showError(tab, 'Session disconnected. Type to start a new session.')
          })
        }
      }, 1000)
    } else {
      if (tab.phase !== 'closed' && tab.phase !== 'idle') {
        setTabPhase(tab, 'closed')
      }
    }

    notifySessionActionsChanged()
  })
}

function handleStreamEnvelope(tab, envelope) {
  if (!tab || !envelope || typeof envelope.type !== 'string') return

  switch (envelope.type) {
    case 'output':
      handleOutputPayload(tab, envelope.payload)
      break
    case 'output_replay':
      handleReplayPayload(tab, envelope.payload)
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

function handleOutputPayload(tab, payload, options = {}) {
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

  const filteredText = filterLocalEcho(tab, text)

  debugLog(tab, 'output-received', {
    length: typeof text === 'string' ? text.length : 0,
    renderedLength: typeof filteredText === 'string' ? filteredText.length : 0,
    replay: options.replay === true,
    timestamp: payload.timestamp || null
  })

  if (filteredText.length > 0) {
    tab.term.write(filteredText)
  }

  if (options.replay !== true && !tab.hasReceivedLiveOutput) {
    tab.hasReceivedLiveOutput = true
    const forceRender = () => {
      try {
        if (tab.id === state.activeTabId && document.hasFocus()) {
          tab.term.focus()
        }
        tab.fitAddon?.fit()
        tab.term.scrollToBottom()
        tab.term.write('')
        tab.term.refresh(0, tab.term.rows - 1)
      } catch (error) {
        console.warn('Failed to refresh terminal on first output:', error)
      }
    }
    forceRender()
    requestAnimationFrame(forceRender)
    setTimeout(forceRender, 50)
  }

  if (options.record !== false) {
    const entry = {
      timestamp: payload.timestamp ? Date.parse(payload.timestamp) : Date.now(),
      direction: payload.direction || 'stdout',
      encoding: payload.encoding,
      data: payload.data
    }
    if (options.replay === true) {
      entry.replay = true
    }
    recordTranscript(tab, entry)
  }

  if (tab.id === state.activeTabId) {
    notifyActiveTabUpdate()
  }
}

function handleReplayPayload(tab, payload) {
  if (!tab || !payload || typeof payload !== 'object') return

  const chunks = Array.isArray(payload.chunks) ? payload.chunks : []
  const wasPending = tab.replayPending === true

  if (wasPending) {
    tab.replayPending = false
    tab.replayComplete = false
    tab.lastReplayCount = 0
    tab.lastReplayTruncated = false

    if (!tab.hasEverConnected && chunks.length > 0) {
      try {
        tab.term.reset()
      } catch (_error) {
        // ignore
      }
      tab.transcript = []
    }
  }

  chunks.forEach((chunk) => {
    if (!chunk || typeof chunk !== 'object') return
    handleOutputPayload(tab, { ...chunk, direction: chunk.direction || 'stdout' }, { replay: true, record: chunk.record !== false })
    tab.lastReplayCount += 1
  })

  if (payload.truncated !== undefined) {
    tab.lastReplayTruncated = Boolean(payload.truncated)
  }

  if (payload.complete) {
    tab.replayComplete = true
    tab.hasEverConnected = true
    const eventPayload = {
      count: tab.lastReplayCount,
      truncated: tab.lastReplayTruncated
    }
    if (payload.generated) {
      const generatedTs = Date.parse(payload.generated)
      if (!Number.isNaN(generatedTs)) {
        eventPayload.generatedAt = generatedTs
      }
    }
    appendEvent(tab, 'output-replay-complete', eventPayload)

    if (tab.id === state.activeTabId) {
      const forceRender = () => {
        try {
          tab.term.focus()
          tab.fitAddon?.fit()
          tab.term.scrollToBottom()
          tab.term.write('')
          tab.term.refresh(0, tab.term.rows - 1)
        } catch (error) {
          console.warn('Failed to refresh terminal after replay:', error)
        }
      }
      requestAnimationFrame(forceRender)
      setTimeout(forceRender, 50)
    }
  } else {
    tab.replayPending = true
  }

  if (tab.id === state.activeTabId) {
    notifyActiveTabUpdate()
  }
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

export async function handleSignOutAllSessions() {
  const button = elements.signOutAllSessions
  if (!button || button.dataset.busy === 'true') return

  const attachedTabs = state.tabs.filter((tab) => tab.session && tab.phase !== 'closed')
  const attachedSessionIds = new Set(attachedTabs.map((tab) => tab.session.id))
  const apiSessions = Array.isArray(state.sessions.items) ? state.sessions.items : []
  const detachedSessions = apiSessions.filter(
    (session) => session && typeof session.id === 'string' && !attachedSessionIds.has(session.id)
  )

  const totalSessions = attachedTabs.length + detachedSessions.length
  if (totalSessions === 0) {
    button.blur()
    await showToast('No active sessions to sign out.', 'info', 2200)
    return
  }

  let confirmMessage
  if (totalSessions === 1) {
    confirmMessage = 'Sign out the active session? This will close the terminal.'
  } else if (detachedSessions.length > 0 && attachedTabs.length > 0) {
    confirmMessage = `Sign out all ${totalSessions} sessions? (${detachedSessions.length} detached)`
  } else if (detachedSessions.length > 0) {
    confirmMessage = detachedSessions.length === 1
      ? 'Sign out the detached session? This will terminate the remote shell.'
      : `Sign out all ${detachedSessions.length} detached sessions?`
  } else {
    confirmMessage = `Sign out all ${totalSessions} sessions? This will close every terminal.`
  }
  if (!window.confirm(confirmMessage)) {
    return
  }

  const originalLabel = button.textContent || ''
  button.dataset.busy = 'true'
  button.dataset.label = originalLabel
  button.disabled = true
  button.textContent = 'Signing out…'

  attachedTabs.forEach((tab) => {
    appendEvent(tab, 'session-stop-all-requested', { id: tab.session.id })
    setTabPhase(tab, 'closing')
    const socketOpen = tab.socket && tab.socket.readyState === WebSocket.OPEN
    setTabSocketState(tab, socketOpen ? 'closing' : 'disconnected')
    tab.wasDetached = true
    refreshTabButton(tab)
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
        : { status: 'terminating_all', terminated: totalSessions }

    attachedTabs.forEach((tab) => {
      appendEvent(tab, 'session-stop-all', eventPayload)
      if (tab.socket) {
        try {
          tab.socket.close()
        } catch (_error) {
          // ignore close failures
        }
      }
    })
    queueSessionOverviewRefresh(250)

    if (detachedSessions.length > 0 && totalSessions > attachedTabs.length) {
      const message = detachedSessions.length === 1
        ? 'Requested sign out for the detached session.'
        : `Requested sign out for ${detachedSessions.length} detached sessions.`
      await showToast(message, 'info', 2400)
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unable to sign out sessions'
    const activeTab = getActiveTab()
    if (activeTab) {
      showError(activeTab, message)
    }
    attachedTabs.forEach((tab) => {
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
    notifySessionActionsChanged()
    queueSessionOverviewRefresh(500)
  }
}

export async function handleCloseDetachedTabs() {
  const button = elements.closeDetachedTabs
  if (!button || button.dataset.busy === 'true') return

  const tabsToClose = getDetachedTabs().slice()
  if (tabsToClose.length === 0) {
    button.classList.add('hidden')
    return
  }

  const originalLabel = button.textContent || 'Close detached tabs'
  button.dataset.busy = 'true'
  button.disabled = true
  button.textContent = `Closing ${tabsToClose.length}…`
  notifySessionActionsChanged()

  let closedCount = 0
  let hadErrors = false
  for (const tab of tabsToClose) {
    try {
      await stopSession(tab)
      closedCount += 1
    } catch (error) {
      console.error('Failed to close detached tab', error)
      hadErrors = true
    }
  }

  button.textContent = originalLabel
  delete button.dataset.busy
  button.disabled = false

  renderTabs()
  notifySessionActionsChanged()

  if (closedCount > 0) {
    queueSessionOverviewRefresh(200)
  }
  if (hadErrors) {
    await showToast('Some detached tabs could not be closed. Check the console for details.', 'error', 3500)
  } else if (closedCount > 0) {
    await showToast(
      closedCount === 1 ? 'Closed detached tab.' : `Closed ${closedCount} detached tabs.`,
      'success',
      2500
    )
  } else {
    await showToast('No detached tabs to close.', 'info', 2500)
  }
}

export function updateSessionActions() {
  const signOutButton = elements.signOutAllSessions
  if (signOutButton) {
    if (signOutButton.dataset.busy === 'true') {
      signOutButton.disabled = true
    } else {
      const hasLocalSessions = state.tabs.some((entry) => entry.session && entry.phase !== 'closed')
      const remoteSessionCount = Array.isArray(state.sessions.items)
        ? state.sessions.items.filter((session) => session && typeof session.id === 'string').length
        : 0
      signOutButton.disabled = !hasLocalSessions && remoteSessionCount === 0
    }
  }

  const closeDetachedButton = elements.closeDetachedTabs
  if (closeDetachedButton) {
    const detachedTabs = getDetachedTabs()
    const detachedCount = detachedTabs.length
    if (closeDetachedButton.dataset.busy === 'true') {
      closeDetachedButton.disabled = true
      closeDetachedButton.classList.remove('hidden')
    } else {
      const label = detachedCount === 1 ? 'Close 1 detached tab' : `Close ${detachedCount} detached tabs`
      closeDetachedButton.textContent = label
      closeDetachedButton.disabled = detachedCount === 0
      closeDetachedButton.classList.toggle('hidden', detachedCount === 0)
    }
  }
}

export async function handleShortcutAction({ command, id }) {
  const sourceId = id || 'unknown'
  const trimmedCommand = typeof command === 'string' ? command.trim() : ''

  if (!trimmedCommand) {
    const tab = getActiveTab()
    if (tab) {
      appendEvent(tab, 'shortcut-invalid', { id: sourceId })
    }
    return
  }

  let tab = getActiveTab()
  if (!tab) {
    tab = createTerminalTab({ focus: true })
    if (tab) {
      startSession(tab, { reason: `shortcut:${sourceId}` }).catch((error) => {
        appendEvent(tab, 'session-error', error)
        showError(tab, error instanceof Error ? error.message : 'Unable to start terminal session')
      })
    }
  }
  if (!tab) return

  const meta = {
    eventType: 'shortcut',
    source: sourceId,
    command: trimmedCommand,
    clearError: true,
    appendNewline: true
  }

  if (tab.socket && tab.socket.readyState === WebSocket.OPEN) {
    const success = transmitInput(tab, trimmedCommand, meta)
    if (!success) {
      showError(tab, 'Terminal stream is not connected')
    }
    return
  }

  queueInput(tab, trimmedCommand, meta)
  appendEvent(tab, 'shortcut-queued', { id: sourceId, command: trimmedCommand })

  if (tab.phase === 'idle' || tab.phase === 'closed') {
    startSession(tab, { reason: `shortcut:${sourceId}` }).catch((error) => {
      appendEvent(tab, 'shortcut-start-error', error)
      showError(tab, 'Unable to start terminal session for shortcut')
    })
  }
}
