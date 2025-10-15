/**
 * Shared utility helpers for the web-console UI.
 */

const sharedTextDecoder = new TextDecoder()
const sharedTextEncoder = new TextEncoder()

export const textDecoder = sharedTextDecoder
export const textEncoder = sharedTextEncoder

let resourceTimingCleanupTimer = null

function scheduleResourceTimingCleanup() {
  if (typeof window === 'undefined' || typeof performance === 'undefined') {
    return
  }
  if (typeof performance.clearResourceTimings !== 'function') {
    return
  }
  if (resourceTimingCleanupTimer) {
    return
  }
  resourceTimingCleanupTimer = window.setTimeout(() => {
    resourceTimingCleanupTimer = null
    try {
      performance.clearResourceTimings()
    } catch (error) {
      console.warn('Failed to clear resource timings:', error)
    }
  }, 15000)
}

export function proxyToApi(path, { method = 'GET', json, headers } = {}) {
  const targetPath = path.startsWith('/api') ? path : `/api${path.startsWith('/') ? path : `/${path}`}`
  const requestInit = { method, headers: headers ? new Headers(headers) : new Headers() }
  if (json !== undefined) {
    requestInit.headers.set('Content-Type', 'application/json')
    requestInit.body = JSON.stringify(json)
  }
  const request = fetch(targetPath, requestInit)
  scheduleResourceTimingCleanup()
  return request
}

export function buildWebSocketUrl(path) {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.host
  const normalized = path.startsWith('/') ? path : `/${path}`
  return `${protocol}//${host}${normalized}`
}

export function isArrayBufferView(value) {
  return value && typeof value === 'object' && ArrayBuffer.isView(value)
}

export async function normalizeSocketData(data, onError) {
  if (typeof data === 'string') {
    return data
  }
  try {
    if (data instanceof Blob && typeof data.text === 'function') {
      return await data.text()
    }
    if (data instanceof ArrayBuffer) {
      return sharedTextDecoder.decode(data)
    }
    if (isArrayBufferView(data)) {
      const view = data
      const buffer = view.byteOffset === 0 && view.byteLength === view.buffer.byteLength
        ? view.buffer
        : view.buffer.slice(view.byteOffset, view.byteOffset + view.byteLength)
      return sharedTextDecoder.decode(buffer)
    }
  } catch (error) {
    if (typeof onError === 'function') {
      onError(error)
    } else {
      console.error('Failed to normalize socket data:', error)
    }
  }
  return ''
}

export function formatCommandLabel(command, args) {
  if (!command) return '—'
  if (Array.isArray(args) && args.length > 0) {
    return `${command} ${args.join(' ')}`
  }
  return command
}

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

export function formatRelativeTimestamp(value, formatter) {
  if (!value) return ''
  try {
    const timestamp = typeof value === 'number' ? value : Date.parse(value)
    if (Number.isNaN(timestamp)) {
      return ''
    }
    if (!formatter) {
      return new Date(timestamp).toLocaleTimeString()
    }
    const diffSeconds = Math.round((timestamp - Date.now()) / 1000)
    const absSeconds = Math.abs(diffSeconds)
    if (absSeconds < 60) {
      return formatter.format(diffSeconds, 'second')
    }
    if (absSeconds < 3600) {
      return formatter.format(Math.round(diffSeconds / 60), 'minute')
    }
    if (absSeconds < 86400) {
      return formatter.format(Math.round(diffSeconds / 3600), 'hour')
    }
    return formatter.format(Math.round(diffSeconds / 86400), 'day')
  } catch (_error) {
    return ''
  }
}

export function formatAbsoluteTime(value) {
  if (!value) return ''
  try {
    const timestamp = typeof value === 'number' ? value : Date.parse(value)
    if (Number.isNaN(timestamp)) {
      return ''
    }
    const date = new Date(timestamp)
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  } catch (_error) {
    return ''
  }
}

export function parseApiError(raw, fallback) {
  if (!raw) {
    return { message: fallback }
  }
  const trimmed = typeof raw === 'string' ? raw.trim() : ''
  if (!trimmed) {
    return { message: fallback }
  }
  try {
    const parsed = JSON.parse(trimmed)
    if (parsed && typeof parsed === 'object' && typeof parsed.message === 'string') {
      return { message: parsed.message.trim() || fallback, raw: parsed }
    }
  } catch (_error) {
    // Ignore JSON parse failures
  }
  return { message: trimmed || fallback }
}

export const scheduleMicrotask = typeof queueMicrotask === 'function'
  ? queueMicrotask
  : (cb) => Promise.resolve().then(cb)
