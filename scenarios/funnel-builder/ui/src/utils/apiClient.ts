const API_BASE_PATH = '/api/v1'

type ProxyEndpoint = {
  path?: string
  url?: string
  target?: string
  basePath?: string
}

export interface AppMonitorProxyInfo {
  path?: string
  basePath?: string
  primary?: ProxyEndpoint
  endpoints?: ProxyEndpoint[]
}

declare global {
  interface Window {
    __APP_MONITOR_PROXY_INFO__?: AppMonitorProxyInfo
    __APP_MONITOR_PROXY_INDEX__?: AppMonitorProxyInfo
  }
}

declare const __API_URL__: string | undefined

type EnvRecord = Record<string, string | undefined>

const stripTrailingSlash = (value: string): string => {
  let next = value
  while (next.length > 1 && next.endsWith('/')) {
    next = next.slice(0, -1)
  }
  return next
}

const ensureLeadingSlash = (value: string): string => (value.startsWith('/') ? value : `/${value}`)

const joinUrlSegments = (base: string, segment: string): string => {
  const normalizedBase = stripTrailingSlash(base)
  const normalizedSegment = ensureLeadingSlash(segment)
  if (!normalizedBase) {
    return normalizedSegment
  }
  return `${normalizedBase}${normalizedSegment}`
}

const ensureApiSuffix = (base: string): string => {
  const normalized = stripTrailingSlash(base)
  if (normalized.toLowerCase().endsWith(API_BASE_PATH)) {
    return normalized
  }
  return stripTrailingSlash(joinUrlSegments(normalized, API_BASE_PATH))
}

const selectProxyCandidate = (info?: AppMonitorProxyInfo): string | undefined => {
  if (!info) {
    return undefined
  }

  const inspectEndpoint = (endpoint?: ProxyEndpoint): string | undefined => {
    if (!endpoint) {
      return undefined
    }
    return endpoint.url ?? endpoint.path ?? endpoint.basePath ?? endpoint.target
  }

  const primaryCandidate = inspectEndpoint(info.primary)
  if (primaryCandidate) {
    return primaryCandidate
  }

  if (Array.isArray(info.endpoints)) {
    for (const endpoint of info.endpoints) {
      const candidate = inspectEndpoint(endpoint)
      if (candidate) {
        return candidate
      }
    }
  }

  if (info.path) {
    return info.path
  }

  if (info.basePath) {
    return info.basePath
  }

  return undefined
}

const resolveProxyApiBase = (): string | undefined => {
  if (typeof window === 'undefined') {
    return undefined
  }

  const proxyInfo = window.__APP_MONITOR_PROXY_INFO__ ?? window.__APP_MONITOR_PROXY_INDEX__
  const candidate = selectProxyCandidate(proxyInfo)
  if (!candidate) {
    return undefined
  }

  if (/^https?:\/\//iu.test(candidate)) {
    return ensureApiSuffix(candidate)
  }

  if (candidate.startsWith('//')) {
    const protocol = typeof window.location?.protocol === 'string' ? window.location.protocol : 'http:'
    return ensureApiSuffix(`${protocol}${candidate}`)
  }

  const origin = typeof window.location?.origin === 'string' ? window.location.origin : ''
  return ensureApiSuffix(joinUrlSegments(origin, candidate))
}

const resolveExplicitApiBase = (): string | undefined => {
  const env = ((import.meta as unknown as { env?: EnvRecord }).env ?? {}) as EnvRecord
  const explicit = env.VITE_API_URL ?? env.API_URL ?? env.VITE_PROXY_API_ORIGIN ?? env.API_ORIGIN
  if (explicit && explicit.trim().length > 0) {
    return ensureApiSuffix(explicit.trim())
  }

  if (typeof __API_URL__ === 'string' && __API_URL__) {
    return ensureApiSuffix(__API_URL__)
  }

  return undefined
}

const resolveFallbackProtocol = (): 'http' | 'https' => {
  if (typeof window !== 'undefined' && window.location?.protocol === 'https:') {
    return 'https'
  }

  const env = ((import.meta as unknown as { env?: EnvRecord }).env ?? {}) as EnvRecord
  const protocol = env.VITE_API_PROTOCOL ?? env.API_PROTOCOL
  if (protocol && protocol.toLowerCase() === 'https') {
    return 'https'
  }

  return 'http'
}

const resolveFallbackHost = (): string => {
  const env = ((import.meta as unknown as { env?: EnvRecord }).env ?? {}) as EnvRecord
  const host = env.VITE_API_HOST ?? env.API_HOST
  if (host && host.trim().length > 0) {
    return host.trim()
  }

  if (typeof window !== 'undefined' && window.location?.hostname) {
    return window.location.hostname
  }

  return '127.0.0.1'
}

const resolveFallbackPort = (): string => {
  const env = ((import.meta as unknown as { env?: EnvRecord }).env ?? {}) as EnvRecord
  const port = env.VITE_API_PORT ?? env.API_PORT
  if (port && port.trim().length > 0) {
    return port.trim()
  }

  if (typeof window !== 'undefined' && window.location?.port) {
    const parsed = Number.parseInt(window.location.port, 10)
    if (Number.isFinite(parsed) && parsed > 0) {
      if (parsed >= 36000 && parsed <= 39999) {
        return String(parsed - 17700)
      }
      if (parsed >= 3000 && parsed < 4000) {
        return String(parsed + 15000)
      }
    }
  }

  return '15000'
}

const resolveFallbackApiBase = (): string => {
  const protocol = resolveFallbackProtocol()
  const host = resolveFallbackHost()
  const port = resolveFallbackPort()
  const authority = port ? `${host}:${port}` : host
  return ensureApiSuffix(`${protocol}://${authority}`)
}

const runtimeApiBase = (() => {
  const proxyBase = resolveProxyApiBase()
  if (proxyBase) {
    return proxyBase
  }

  const explicitBase = resolveExplicitApiBase()
  if (explicitBase) {
    return explicitBase
  }

  return resolveFallbackApiBase()
})()

export const API_BASE_URL = runtimeApiBase

const buildUrl = (path: string): string => {
  const segment = ensureLeadingSlash(path)
  return `${API_BASE_URL}${segment}`
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(buildUrl(path), {
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json',
      ...init?.headers,
    },
    ...init,
  })

  if (!response.ok) {
    const message = await response.text().catch(() => response.statusText)
    throw new Error(`API request failed: ${response.status} ${message}`)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}
