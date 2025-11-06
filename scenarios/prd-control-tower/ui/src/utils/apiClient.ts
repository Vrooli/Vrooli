const API_BASE_PATH = '/api/v1'
const LOOPBACK_HOST = ['127', '0', '0', '1'].join('.')
const DEFAULT_API_PORT = 18600

type AppMonitorProxyEndpoint =
  | string
  | {
      url?: unknown
      path?: unknown
      target?: unknown
    }

type AppMonitorProxyInfo =
  | AppMonitorProxyEndpoint
  | {
      primary?: AppMonitorProxyEndpoint
      endpoints?: AppMonitorProxyEndpoint[]
    }
  | null
  | undefined

declare global {
  interface Window {
    __APP_MONITOR_PROXY_INFO__?: AppMonitorProxyInfo
    __APP_MONITOR_PROXY_INDEX__?: AppMonitorProxyInfo
  }
}

function coerceString(value: unknown): string | undefined {
  if (typeof value === 'string') {
    const trimmed = value.trim()
    return trimmed.length > 0 ? trimmed : undefined
  }
  return undefined
}

function collectProxyCandidates(value: unknown, seen: Set<string>, list: string[]): void {
  if (!value) {
    return
  }

  if (typeof value === 'string') {
    const candidate = value.trim()
    if (candidate && !seen.has(candidate)) {
      seen.add(candidate)
      list.push(candidate)
    }
    return
  }

  if (typeof value === 'object') {
    const record = value as Record<string, unknown>
    collectProxyCandidates(record.url, seen, list)
    collectProxyCandidates(record.path, seen, list)
    collectProxyCandidates(record.target, seen, list)
  }
}

function pickProxyBase(info: AppMonitorProxyInfo): string | undefined {
  const seen = new Set<string>()
  const candidates: string[] = []

  collectProxyCandidates(info, seen, candidates)

  if (info && typeof info === 'object') {
    const descriptor = info as { primary?: AppMonitorProxyEndpoint; endpoints?: AppMonitorProxyEndpoint[] }
    collectProxyCandidates(descriptor.primary, seen, candidates)
    if (Array.isArray(descriptor.endpoints)) {
      descriptor.endpoints.forEach((endpoint) => collectProxyCandidates(endpoint, seen, candidates))
    }
  }

  return candidates.find(Boolean)
}

function ensureLeadingSlash(value: string): string {
  if (!value) {
    return '/'
  }
  return value.startsWith('/') ? value : `/${value}`
}

function stripTrailingSlash(value: string): string {
  return value.replace(/\/+$/, '')
}

function sanitizePort(port: unknown): string {
  if (typeof port === 'number') {
    return port > 0 ? String(port) : ''
  }
  if (typeof port === 'string') {
    const trimmed = port.trim()
    return trimmed ? trimmed : ''
  }
  return ''
}

function resolveProxyBase(): string | undefined {
  if (typeof window === 'undefined') {
    return undefined
  }

  const info = window.__APP_MONITOR_PROXY_INFO__ ?? window.__APP_MONITOR_PROXY_INDEX__
  const candidate = pickProxyBase(info)
  if (!candidate) {
    return undefined
  }

  if (/^https?:\/\//i.test(candidate)) {
    return stripTrailingSlash(candidate)
  }

  const origin = coerceString(window.location?.origin)
  if (!origin) {
    return undefined
  }

  return stripTrailingSlash(origin + ensureLeadingSlash(candidate))
}

/**
 * Resolves the base URL for API requests.
 *
 * Strategy:
 * 1. Check VITE_API_URL environment variable
 * 2. Derive from App Monitor proxy metadata when available
 * 3. Use current origin (covers local proxy server)
 * 4. Fallback to localhost with calculated port
 */
function getApiBaseUrl(): string {
  // Strategy 1: Explicit environment variable
  const env = (import.meta as { env?: Record<string, unknown> }).env
  const envApiUrl = env?.VITE_API_URL
  if (typeof envApiUrl === 'string' && envApiUrl.trim()) {
    return envApiUrl.trim().replace(/\/+$/, '')
  }

  // Strategy 2: App Monitor proxy metadata
  if (typeof window !== 'undefined') {
    const proxyBase = resolveProxyBase()
    if (proxyBase) {
      return proxyBase
    }
  }

  // Strategy 3: Use current origin (works with UI proxy server)
  if (typeof window !== 'undefined') {
    const origin = coerceString(window.location?.origin)
    if (origin) {
      return stripTrailingSlash(origin)
    }
  }

  // Strategy 4: Fallback for direct file access or edge cases
  if (typeof window !== 'undefined') {
    const protocol = window.location?.protocol === 'https:' ? 'https' : 'http'
    const hostname = coerceString(window.location?.hostname) || LOOPBACK_HOST
    const portSegment = sanitizePort(window.location?.port)

    let apiPort = DEFAULT_API_PORT
    if (portSegment) {
      const uiPort = Number.parseInt(portSegment, 10)
      if (Number.isFinite(uiPort) && uiPort >= 36000 && uiPort <= 39999) {
        apiPort = uiPort - 17700
      }
    }

    return `${protocol}://${hostname}:${apiPort}`
  }

  // Final fallback when window is unavailable
  return `http://localhost:${DEFAULT_API_PORT}`
}

/**
 * Get the full API base URL including the /api/v1 path
 */
export function getApiBase(): string {
  const baseUrl = getApiBaseUrl()
  return `${baseUrl}${API_BASE_PATH}`
}

/**
 * Build a complete API URL for a given path
 */
export function buildApiUrl(path: string): string {
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${getApiBase()}${normalizedPath}`
}
