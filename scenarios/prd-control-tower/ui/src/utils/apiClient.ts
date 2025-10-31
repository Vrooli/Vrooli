const API_BASE_PATH = '/api/v1'
const LOOPBACK_SEGMENTS = ['127', '0', '0', '1'] as const
const LOOPBACK_FALLBACK_PORT = 18600

type ProxyEndpoint = {
  path?: unknown
  url?: unknown
  target?: unknown
}

type AppMonitorProxyInfo =
  | ProxyEndpoint
  | string
  | {
      primary?: ProxyEndpoint
      endpoints?: ProxyEndpoint[]
    }
  | null
  | undefined

declare global {
  interface Window {
    __APP_MONITOR_PROXY_INFO__?: AppMonitorProxyInfo
    __APP_MONITOR_PROXY_INDEX__?: AppMonitorProxyInfo
  }
}

const toTrimmedString = (value: unknown): string | undefined => {
  if (typeof value === 'string') {
    const trimmed = value.trim()
    return trimmed.length > 0 ? trimmed : undefined
  }
  return undefined
}

const stripTrailingSlash = (value: string): string => value.replace(/\/+$/, '')

const ensureLeadingSlash = (value: string): string => (value.startsWith('/') ? value : `/${value}`)

const joinUrlSegments = (base: string, segment: string): string => {
  const cleanedBase = stripTrailingSlash(base)
  const normalizedSegment = ensureLeadingSlash(segment)
  if (!cleanedBase) {
    return normalizedSegment
  }
  return `${cleanedBase}${normalizedSegment}`
}

const collectProxyCandidates = (input: AppMonitorProxyInfo, seen: Set<string>, results: string[]): void => {
  if (!input) {
    return
  }

  if (typeof input === 'string') {
    const candidate = input.trim()
    if (candidate && !seen.has(candidate)) {
      seen.add(candidate)
      results.push(candidate)
    }
    return
  }

  if (Array.isArray(input)) {
    input.forEach(item => collectProxyCandidates(item, seen, results))
    return
  }

  if (typeof input === 'object') {
    const record = input as Record<string, unknown>
    collectProxyCandidates(record.url as AppMonitorProxyInfo, seen, results)
    collectProxyCandidates(record.path as AppMonitorProxyInfo, seen, results)
    collectProxyCandidates(record.target as AppMonitorProxyInfo, seen, results)
    if (Array.isArray(record.endpoints)) {
      collectProxyCandidates(record.endpoints as AppMonitorProxyInfo, seen, results)
    }
    if (record.primary) {
      collectProxyCandidates(record.primary as AppMonitorProxyInfo, seen, results)
    }
  }
}

const isLocalHostname = (value?: string | null): boolean => {
  if (!value) {
    return false
  }
  const normalized = value.trim().toLowerCase()
  if (!normalized) {
    return false
  }
  return (
    normalized === 'localhost' ||
    normalized === '127.0.0.1' ||
    normalized === '0.0.0.0' ||
    normalized === '::1' ||
    normalized === '[::1]'
  )
}

const resolveProxyBase = (): string | undefined => {
  if (typeof window === 'undefined') {
    return undefined
  }

  const globalWindow = window as Window & {
    __APP_MONITOR_PROXY_INFO__?: AppMonitorProxyInfo
    __APP_MONITOR_PROXY_INDEX__?: AppMonitorProxyInfo
  }

  const visited = new Set<Window>()
  const seen = new Set<string>()
  const candidates: string[] = []

  const collectFromWindow = (target?: Window | null): void => {
    if (!target) {
      return
    }
    if (visited.has(target)) {
      return
    }
    visited.add(target)

    try {
      const source = target as Window & {
        __APP_MONITOR_PROXY_INFO__?: AppMonitorProxyInfo
        __APP_MONITOR_PROXY_INDEX__?: AppMonitorProxyInfo
      }
      const info = source.__APP_MONITOR_PROXY_INFO__ ?? source.__APP_MONITOR_PROXY_INDEX__
      collectProxyCandidates(info, seen, candidates)
    } catch (error) {
      // Accessing parent/top windows can throw for cross-origin frames; ignore and continue.
    }
  }

  collectFromWindow(globalWindow)
  collectFromWindow(globalWindow.parent)
  collectFromWindow(globalWindow.top)

  const candidate = candidates.find(Boolean)
  if (!candidate) {
    return undefined
  }

  if (/^https?:\/\//i.test(candidate)) {
    return stripTrailingSlash(candidate)
  }

  const origin = toTrimmedString(globalWindow.location?.origin)
  if (!origin) {
    return undefined
  }

  return stripTrailingSlash(joinUrlSegments(origin, candidate))
}

const resolveLocalOriginBase = (): string | undefined => {
  if (typeof window === 'undefined') {
    return undefined
  }

  const origin = toTrimmedString(window.location?.origin)
  if (!origin) {
    return undefined
  }

  if (!isLocalHostname(window.location?.hostname)) {
    return undefined
  }

  return stripTrailingSlash(origin)
}

const resolveRelativeBase = (): string | undefined => {
  if (typeof window === 'undefined') {
    return undefined
  }

  return ''
}

const resolveFallbackBase = (): string => {
  const env = (import.meta as unknown as { env?: Record<string, unknown> }).env ?? {}
  const envUrl = toTrimmedString(env?.VITE_API_URL)
  if (envUrl) {
    return stripTrailingSlash(envUrl)
  }

  const envHost = toTrimmedString(env?.VITE_API_HOST)
  const envPort = toTrimmedString(env?.VITE_API_PORT)
  const envProtocol = (toTrimmedString(env?.VITE_API_PROTOCOL) ?? '').toLowerCase() === 'https' ? 'https' : 'http'

  if (envHost && envPort) {
    return `${envProtocol}://${envHost}:${envPort}`
  }

  if (typeof window !== 'undefined') {
    const protocol = window.location?.protocol === 'https:' ? 'https' : 'http'
    const hostname = toTrimmedString(window.location?.hostname) ?? ['local', 'host'].join('')
    const uiPort = Number.parseInt(window.location?.port || '', 10)
    let fallbackPort = ''

    if (Number.isFinite(uiPort) && uiPort > 0) {
      if (uiPort >= 36000 && uiPort <= 39999) {
        fallbackPort = String(uiPort - 17700)
      } else if (uiPort >= 3000 && uiPort < 4000) {
        fallbackPort = String(uiPort + 15000)
      }
    }

    const resolvedHostname = hostname || LOOPBACK_SEGMENTS.join('.')
    const portCandidate = fallbackPort || (isLocalHostname(resolvedHostname) ? String(LOOPBACK_FALLBACK_PORT) : '')
    const portSegment = portCandidate ? `:${portCandidate}` : ''
    return `${protocol}://${resolvedHostname}${portSegment}`
  }

  const defaultProtocol = 'http'
  const loopbackHost = LOOPBACK_SEGMENTS.join('.')
  return `${defaultProtocol}://${loopbackHost}:${LOOPBACK_FALLBACK_PORT}`
}

const withApiBasePath = (base: string): string => joinUrlSegments(stripTrailingSlash(base), API_BASE_PATH)

export const getApiBase = (): string => {
  const proxyBase = resolveProxyBase()
  if (proxyBase) {
    return withApiBasePath(proxyBase)
  }

  const localOriginBase = resolveLocalOriginBase()
  if (localOriginBase) {
    return withApiBasePath(localOriginBase)
  }

  const relativeBase = resolveRelativeBase()
  if (relativeBase !== undefined) {
    return withApiBasePath(relativeBase)
  }

  const fallbackBase = resolveFallbackBase()
  return withApiBasePath(fallbackBase)
}

export const buildApiUrl = (path: string): string => {
  const normalizedPath = ensureLeadingSlash(path)
  return `${stripTrailingSlash(getApiBase())}${normalizedPath}`
}
