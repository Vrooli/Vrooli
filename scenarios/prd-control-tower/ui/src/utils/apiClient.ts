const API_BASE_PATH = '/api/v1'

type ProxyEndpoint = {
  path?: string
  url?: string
  target?: string
}

type AppMonitorProxyInfo = {
  primary?: ProxyEndpoint
  endpoints?: ProxyEndpoint[]
}

declare global {
  interface Window {
    __APP_MONITOR_PROXY_INFO__?: AppMonitorProxyInfo
  }
}

const stripTrailingSlash = (value: string): string => {
  if (value.endsWith('/') && !/https?:\/\/$/i.test(value)) {
    return value.replace(/\/+$/, '')
  }
  return value
}

const ensureLeadingSlash = (value: string): string => (value.startsWith('/') ? value : `/${value}`)

const joinUrlSegments = (base: string, segment: string): string => {
  const cleanedBase = stripTrailingSlash(base)
  const normalizedSegment = ensureLeadingSlash(segment)
  if (!cleanedBase) {
    return normalizedSegment
  }
  return `${cleanedBase}${normalizedSegment}`
}

const pickProxyCandidate = (info?: AppMonitorProxyInfo): string | undefined => {
  if (!info) {
    return undefined
  }

  if (info.primary) {
    const candidate = info.primary.url ?? info.primary.path ?? info.primary.target
    if (candidate) {
      return candidate
    }
  }

  if (Array.isArray(info.endpoints)) {
    for (const endpoint of info.endpoints) {
      if (!endpoint) {
        continue
      }
      const candidate = endpoint.url ?? endpoint.path ?? endpoint.target
      if (candidate) {
        return candidate
      }
    }
  }

  return undefined
}

const resolveProxyBase = (): string | undefined => {
  const proxyInfo = (window as Window & { __APP_MONITOR_PROXY_INFO__?: AppMonitorProxyInfo }).__APP_MONITOR_PROXY_INFO__
  const candidate = pickProxyCandidate(proxyInfo)
  if (!candidate) {
    return undefined
  }

  if (/^https?:\/\//i.test(candidate)) {
    return stripTrailingSlash(candidate)
  }

  return stripTrailingSlash(joinUrlSegments(window.location.origin, candidate))
}

const resolveFallbackHost = (): string => {
  const envHost = (import.meta as unknown as { env?: Record<string, unknown> }).env?.VITE_API_HOST
  if (typeof envHost === 'string' && envHost.trim().length > 0) {
    return envHost.trim()
  }

  if (window.location.hostname) {
    return window.location.hostname
  }

  return ['local', 'host'].join('')
}

const resolveFallbackPort = (): string => {
  const envPort = (import.meta as unknown as { env?: Record<string, unknown> }).env?.VITE_API_PORT
  if (typeof envPort === 'string' && envPort.trim().length > 0) {
    return envPort.trim()
  }

  const uiPortValue = window.location.port
  const parsed = Number(uiPortValue)
  if (Number.isFinite(parsed) && parsed > 0) {
    if (parsed >= 36000 && parsed <= 39999) {
      return String(parsed - 17700)
    }
    if (parsed >= 3000 && parsed < 4000) {
      return String(parsed + 15000)
    }
  }

  return '18600'
}

const resolveFallbackProtocol = (): 'http' | 'https' => (window.location.protocol === 'https:' ? 'https' : 'http')

const withApiBasePath = (base: string): string => joinUrlSegments(stripTrailingSlash(base), API_BASE_PATH)

export const getApiBase = (): string => {
  const proxyBase = resolveProxyBase()
  if (proxyBase) {
    return withApiBasePath(proxyBase)
  }

  const protocol = resolveFallbackProtocol()
  const host = resolveFallbackHost()
  const port = resolveFallbackPort()

  return `${protocol}://${host}:${port}${API_BASE_PATH}`
}

export const buildApiUrl = (path: string): string => {
  const normalizedPath = ensureLeadingSlash(path)
  return `${stripTrailingSlash(getApiBase())}${normalizedPath}`
}
