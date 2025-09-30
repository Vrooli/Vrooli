const elements = {
  startBtn: document.getElementById('startBtn'),
  stopBtn: document.getElementById('stopBtn'),
  sendBtn: document.getElementById('sendBtn'),
  operatorInput: document.getElementById('operatorInput'),
  sessionId: document.getElementById('sessionId'),
  sessionPhase: document.getElementById('sessionPhase'),
  socketState: document.getElementById('socketState'),
  sessionCommand: document.getElementById('sessionCommand'),
  transcriptSize: document.getElementById('transcriptSize'),
  eventFeed: document.getElementById('eventFeed'),
  eventMeta: document.getElementById('eventMeta'),
  statusBadge: document.getElementById('statusBadge'),
  inputBox: document.getElementById('inputBox'),
  inputForm: document.getElementById('inputForm'),
  errorBanner: document.getElementById('errorBanner'),
  terminalContainer: document.getElementById('terminal')
}

const shortcutButtons = Array.from(document.querySelectorAll('[data-shortcut-id]'))

const term = new window.Terminal({
  convertEol: true,
  cursorBlink: true,
  fontFamily: 'JetBrains Mono, SFMono-Regular, Menlo, monospace',
  fontSize: 14,
  theme: {
    background: '#0f172a',
    foreground: '#e2e8f0',
    cursor: '#38bdf8'
  }
})
const fitAddon = new window.FitAddon.FitAddon()
term.loadAddon(fitAddon)
term.open(elements.terminalContainer)
fitAddon.fit()
let lastSentSize = { cols: 0, rows: 0 }
window.addEventListener('resize', () => requestAnimationFrame(() => fitAddon.fit()))
term.onResize(({ cols, rows }) => {
  sendResize(cols, rows)
})

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

const state = {
  phase: 'idle',
  socketState: 'disconnected',
  session: null,
  socket: null,
  transcript: [],
  events: [],
  suppressed: initialSuppressedState(),
  bridge: null,
  pendingShortcuts: []
}

function handleShortcutButton(button) {
  if (!button) return
  const command = (button.dataset.command || '').trim()
  const id = button.dataset.shortcutId || 'unknown'
  if (!command) {
    appendEvent('shortcut-invalid', { id })
    return
  }

  if (state.socket && state.socket.readyState === WebSocket.OPEN) {
    const success = transmitInput(command, {
      eventType: 'shortcut',
      source: id,
      command,
      flushInputBox: false,
      clearError: true
    })
    if (!success) {
      showError('Terminal stream is not connected')
    }
    return
  }

  state.pendingShortcuts.push({ id, command })
  appendEvent('shortcut-queued', { id, command })

  if (state.phase === 'idle' || state.phase === 'closed') {
    startSession({ reason: `shortcut:${id}` }).catch((error) => {
      appendEvent('shortcut-start-error', error)
      showError('Unable to start terminal session for shortcut')
    })
  }
}

const textDecoder = new TextDecoder()

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
          if (message.payload && typeof message.payload === 'object' && typeof message.payload.operator === 'string') {
            elements.operatorInput.value = message.payload.operator
          }
          emit('ready', { timestamp: Date.now() })
          break
        case 'end-session':
          await stopSession()
          emit('session-ended', { requestedByParent: true })
          break
        case 'request-screenshot':
          if (window.html2canvas) {
            const canvas = await window.html2canvas(document.body, { backgroundColor: '#0f172a' })
            emit('screenshot', { image: canvas.toDataURL('image/png', 0.9), requestedAt: Date.now() })
          } else {
            emit('error', { type: 'request-screenshot', message: 'html2canvas not available' })
          }
          break
        case 'request-transcript':
          emit('transcript', { transcript: state.transcript, requestedAt: Date.now() })
          break
        case 'request-logs':
          emit('logs', { logs: state.events.slice(-100), requestedAt: Date.now() })
          break
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

state.bridge = initializeIframeBridge()

function setPhase(phase) {
  state.phase = phase
  updateUI()
}

function setSocketState(socket) {
  state.socketState = socket
  updateUI()
}

function updateUI() {
  elements.sessionPhase.textContent = state.phase
  elements.socketState.textContent = state.socketState
  elements.startBtn.disabled = state.phase === 'creating' || state.phase === 'running'
  elements.stopBtn.disabled = !state.session || state.phase === 'idle' || state.phase === 'closed'
  elements.sendBtn.disabled = state.socketState !== 'open'
  elements.inputBox.disabled = state.socketState !== 'open'
  elements.sessionId.textContent = state.session ? `${state.session.id.slice(0, 8)}…` : '—'
  if (elements.sessionCommand) {
    elements.sessionCommand.textContent = state.session ? formatCommandLabel(state.session.command, state.session.args) : '—'
  }
  elements.transcriptSize.textContent = `${state.transcript.length} records`

  shortcutButtons.forEach((button) => {
    if (!button) return
    button.disabled = state.phase === 'closing'
  })

  const badgeColor = state.phase === 'running' ? '#22c55e' : state.phase === 'closing' ? '#f97316' : '#38bdf8'
  elements.statusBadge.textContent = `${state.phase.toUpperCase()} · ${state.socketState.toUpperCase()}`
  elements.statusBadge.style.color = badgeColor

  renderEventMeta()
  renderEvents()

  state.bridge?.emit('session-update', {
    phase: state.phase,
    socketState: state.socketState,
    session: state.session,
    transcriptSize: state.transcript.length,
    events: state.events.slice(-20),
    suppressed: { ...state.suppressed }
  })
}

function renderEvents() {
  const recent = state.events.slice(-50).reverse()
  elements.eventFeed.innerHTML = ''
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

function renderEventMeta() {
  if (!elements.eventMeta) return
  const summaries = Object.entries(state.suppressed)
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

function appendEvent(type, payload) {
  const normalized = payload instanceof Error ? { message: payload.message, stack: payload.stack } : payload
  if (AGGREGATED_EVENT_TYPES.has(type)) {
    recordSuppressedEvent(type)
    return
  }
  state.events.push({ type, payload: normalized, timestamp: Date.now() })
  if (state.events.length > 500) {
    state.events.splice(0, state.events.length - 500)
  }
  updateUI()
}

function recordSuppressedEvent(type) {
  const current = (state.suppressed[type] || 0) + 1
  state.suppressed[type] = current
  if (current === 1 || current % 25 === 0) {
    renderEventMeta()
  }
}

function recordTranscript(entry) {
  state.transcript.push(entry)
  if (state.transcript.length > 5000) {
    state.transcript.splice(0, state.transcript.length - 5000)
  }
  updateUI()
}

function showError(message) {
  if (!message) {
    elements.errorBanner.classList.add('hidden')
    elements.errorBanner.textContent = ''
    return
  }
  elements.errorBanner.textContent = message
  elements.errorBanner.classList.remove('hidden')
}

async function startSession(options = {}) {
  if (state.phase === 'creating' || state.phase === 'running') return
  showError('')
  setPhase('creating')
  setSocketState('connecting')

  try {
    const payload = {}
    const operator = elements.operatorInput.value.trim()
    if (operator) payload.operator = operator
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
    state.session = {
      ...data,
      command: sessionCommand,
      args: sessionArgs,
      commandLine: formatCommandLabel(sessionCommand, sessionArgs)
    }
    state.transcript = []
    state.events = []
    state.suppressed = initialSuppressedState()
    term.reset()
    renderEventMeta()
    lastSentSize = { cols: 0, rows: 0 }
    appendEvent('session-created', {
      ...data,
      reason: options.reason || null,
      command: state.session.command,
      args: state.session.args
    })
    connectWebSocket(data.id)
    setPhase('running')
    state.bridge?.emit('session-started', data)
  } catch (error) {
    setPhase('idle')
    setSocketState('disconnected')
    showError(error instanceof Error ? error.message : 'Unknown error starting session')
    appendEvent('session-error', error)
  }
}

async function stopSession() {
  if (!state.session) return
  setPhase('closing')
  setSocketState('closing')
  appendEvent('session-stop-requested', { id: state.session.id })
  try {
    await proxyToApi(`/api/v1/sessions/${state.session.id}`, { method: 'DELETE' })
  } catch (error) {
    appendEvent('session-stop-error', error)
  } finally {
    state.socket?.close()
    state.socket = null
    setPhase('closed')
    setSocketState('disconnected')
    state.bridge?.emit('session-ended', { id: state.session?.id })
  }
}

function isArrayBufferView(value) {
  return value && typeof value === 'object' && ArrayBuffer.isView(value)
}

async function normalizeSocketData(data) {
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
    appendEvent('ws-decode-error', error)
  }
  return ''
}

function connectWebSocket(sessionId) {
  const url = buildWebSocketUrl(`/ws/sessions/${sessionId}/stream`)
  const socket = new WebSocket(url)
  state.socket = socket

  socket.addEventListener('open', () => {
    setSocketState('open')
    appendEvent('ws-open', { sessionId })
    if (term && typeof term.cols === 'number' && typeof term.rows === 'number') {
      sendResize(term.cols, term.rows)
    }
    flushPendingShortcuts()
  })

  socket.addEventListener('message', async (event) => {
    const raw = await normalizeSocketData(event.data)
    if (!raw || raw.trim().length === 0) {
      recordSuppressedEvent('ws-empty-frame')
      return
    }
    try {
      const envelope = JSON.parse(raw)
      handleStreamEnvelope(envelope)
    } catch (error) {
      appendEvent('ws-message-error', {
        message: error instanceof Error ? error.message : String(error),
        raw
      })
      showError('Failed to process stream message')
    }
  })

  socket.addEventListener('error', (error) => {
    setSocketState('error')
    showError('WebSocket error occurred')
    appendEvent('ws-error', error)
  })

  socket.addEventListener('close', (event) => {
    setSocketState('disconnected')
    appendEvent('ws-close', { code: event.code, reason: event.reason })
    if (state.phase === 'running' || state.phase === 'closing') {
      setPhase('closed')
    }
    state.bridge?.emit('session-ended', { id: sessionId, reason: 'ws-close' })
  })
}

function handleStreamEnvelope(envelope) {
  if (!envelope || typeof envelope.type !== 'string') return

  switch (envelope.type) {
    case 'output':
      handleOutputPayload(envelope.payload)
      break
    case 'status':
      handleStatusPayload(envelope.payload)
      break
    case 'heartbeat':
      appendEvent('session-heartbeat', envelope.payload)
      break
    default:
      appendEvent('session-envelope', envelope)
  }
}

function handleOutputPayload(payload) {
  if (!payload || typeof payload.data !== 'string') return

  let text = payload.data
  if (payload.encoding === 'base64') {
    try {
      const bytes = Uint8Array.from(atob(payload.data), (c) => c.charCodeAt(0))
      text = textDecoder.decode(bytes)
    } catch (error) {
      appendEvent('decode-error', error)
      return
    }
  }

  term.write(text)
  recordTranscript({
    timestamp: payload.timestamp ? Date.parse(payload.timestamp) : Date.now(),
    direction: payload.direction || 'stdout',
    encoding: payload.encoding,
    data: payload.data
  })
}

function handleStatusPayload(payload) {
  appendEvent('session-status', payload)

  if (payload && typeof payload === 'object') {
    recordTranscript({
      timestamp: payload.timestamp ? Date.parse(payload.timestamp) : Date.now(),
      direction: 'status',
      message: JSON.stringify(payload)
    })

    if (payload.status === 'started') {
      setPhase('running')
      setSocketState('open')
      showError('')
      return
    }

    if (payload.status === 'command_exit_error') {
      setPhase('closed')
      setSocketState('disconnected')
      showError(`Command exited: ${payload.reason || 'unknown error'}`)
      notifyPanic(payload)
      return
    }

    if (payload.status === 'closed') {
      setPhase('closed')
      setSocketState('disconnected')
      if (payload.reason && !payload.reason.includes('client_requested')) {
        showError(`Session closed: ${payload.reason}`)
      }
      return
    }

    return
  }

  recordTranscript({
    timestamp: Date.now(),
    direction: 'status',
    message: JSON.stringify(payload)
  })
}

function notifyPanic(payload) {
  const reason = payload && typeof payload === 'object' && payload.reason ? payload.reason : 'unknown error'
  term.write(`\r\n\u001b[31mCommand exited\u001b[0m: ${reason}\r\n`)
  term.write('Review the event feed or transcripts for details.\r\n')
}

function transmitInput(value, meta = {}) {
  if (!state.socket || state.socket.readyState !== WebSocket.OPEN) {
    return false
  }

  const normalized = value.endsWith('\n') ? value : `${value}\n`
  const payload = {
    type: 'input',
    payload: {
      data: normalized,
      encoding: 'utf-8'
    }
  }

  try {
    state.socket.send(JSON.stringify(payload))
  } catch (error) {
    appendEvent('stdin-send-error', error)
    return false
  }

  recordTranscript({
    timestamp: Date.now(),
    direction: 'stdin',
    encoding: 'utf-8',
    data: btoa(payload.payload.data)
  })

  appendEvent(meta.eventType || 'stdin', {
    length: payload.payload.data.length,
    source: meta.source || undefined,
    command: meta.command || value.trim()
  })

  if (meta.flushInputBox !== false) {
    elements.inputBox.value = ''
  }

  if (meta.clearError !== false) {
    showError('')
  }

  return true
}

function flushPendingShortcuts() {
  if (!state.socket || state.socket.readyState !== WebSocket.OPEN) {
    return
  }
  if (!Array.isArray(state.pendingShortcuts) || state.pendingShortcuts.length === 0) {
    return
  }
  const queue = state.pendingShortcuts.splice(0)
  queue.forEach((item) => {
    const success = transmitInput(item.command, {
      eventType: 'shortcut',
      source: item.id,
      command: item.command.trim(),
      flushInputBox: false,
      clearError: false
    })
    if (!success) {
      appendEvent('shortcut-dispatch-failed', { id: item.id })
    }
  })
}

elements.inputForm.addEventListener('submit', (event) => {
  event.preventDefault()
  sendInput(elements.inputBox.value)
})

elements.inputBox.addEventListener('keydown', (event) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendInput(elements.inputBox.value)
  }
})

elements.startBtn.addEventListener('click', () => startSession())
elements.stopBtn.addEventListener('click', () => stopSession())

elements.operatorInput.addEventListener('change', () => {
  state.bridge?.emit('operator-updated', { operator: elements.operatorInput.value })
})

shortcutButtons.forEach((button) => {
  button.addEventListener('click', () => handleShortcutButton(button))
})

function sendInput(value) {
  const trimmed = value.trim()
  if (!trimmed) return
  const success = transmitInput(value, { flushInputBox: true, clearError: true, eventType: 'stdin', command: trimmed })
  if (!success) {
    showError('Terminal stream is not connected')
  }
}

function sendResize(cols, rows) {
  if (!state.socket || state.socket.readyState !== WebSocket.OPEN) {
    return
  }
  if (!Number.isInteger(cols) || !Number.isInteger(rows) || cols <= 0 || rows <= 0) {
    return
  }
  const { cols: lastCols, rows: lastRows } = lastSentSize
  if (cols === lastCols && rows === lastRows) {
    return
  }
  const payload = {
    type: 'resize',
    payload: { cols, rows }
  }
  try {
    state.socket.send(JSON.stringify(payload))
    lastSentSize = { cols, rows }
    appendEvent('terminal-resize', { cols, rows })
  } catch (error) {
    appendEvent('terminal-resize-error', error)
  }
}

updateUI()

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
