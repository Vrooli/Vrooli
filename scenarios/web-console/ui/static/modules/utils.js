/**
 * Shared utility functions for web-console
 */

export const textDecoder = new TextDecoder()

/**
 * Proxy API requests through the UI server
 */
export function proxyToApi(path, { method = 'GET', json, headers } = {}) {
  const targetPath = path.startsWith('/api') ? path : `/api${path.startsWith('/') ? path : `/${path}`}`
  const requestInit = { method, headers: headers ? new Headers(headers) : new Headers() }
  if (json !== undefined) {
    requestInit.headers.set('Content-Type', 'application/json')
    requestInit.body = JSON.stringify(json)
  }
  return fetch(targetPath, requestInit)
}

/**
 * Build WebSocket URL from path
 */
export function buildWebSocketUrl(path) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const normalized = path.startsWith('/') ? path : `/${path}`
  return `${protocol}//${host}${normalized}`
}

/**
 * Check if value is ArrayBufferView
 */
export function isArrayBufferView(value) {
  return value && typeof value === 'object' && ArrayBuffer.isView(value)
}

/**
 * Normalize WebSocket data to string
 */
export async function normalizeSocketData(data, tab = null) {
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
    console.error('Failed to normalize socket data:', error)
  }
  return ''
}

/**
 * Format command label from command and args
 */
export function formatCommandLabel(command, args) {
  if (!command) return '—'
  if (Array.isArray(args) && args.length > 0) {
    return `${command} ${args.join(' ')}`
  }
  return command
}

/**
 * Format event payload for display
 */
export function formatEventPayload(payload) {
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

/**
 * Initialize iframe bridge for parent-child communication
 */
export function initializeIframeBridge(handlers) {
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
      const handler = handlers[message.type]
      if (handler) {
        await handler(message, emit)
      } else {
        emit('error', { type: 'unknown-command', payload: message })
      }
    } catch (error) {
      emit('error', { type: message.type, message: error instanceof Error ? error.message : 'unknown error' })
    }
  })

  emit('bridge-initialized', { timestamp: Date.now() })
  return { emit }
}
