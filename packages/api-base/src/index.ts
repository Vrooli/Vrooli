export interface WindowLike {
  location?: {
    hostname?: string
    origin?: string
    pathname?: string
  }
  __APP_MONITOR_PROXY_INFO__?: unknown
}

export interface ResolveApiBaseOptions {
  explicitUrl?: string | null
  defaultPort?: string
  apiSuffix?: string
  appendSuffix?: boolean
  windowObject?: WindowLike
}

export interface BuildApiUrlOptions extends ResolveApiBaseOptions {
  baseUrl?: string
}

const LOOPBACK_HOST = '127.0.0.1'
const DEFAULT_PORT = '15000'
const DEFAULT_SUFFIX = '/api/v1'

const LOCAL_HOST_PATTERN = /^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)/i

function getWindowLike(options?: ResolveApiBaseOptions | BuildApiUrlOptions): WindowLike | undefined {
  if (options?.windowObject) {
    return options.windowObject
  }

  const global = typeof globalThis === 'object' && globalThis ? (globalThis as any) : undefined
  if (!global) {
    return undefined
  }

  const win = global.window as WindowLike | undefined
  if (win && typeof win === 'object') {
    return win
  }

  return undefined
}

function normalizeBase(value: string): string {
  return value.replace(/\/+$/, '')
}

function appendSuffix(base: string, suffix: string): string {
  const normalizedBase = normalizeBase(base)
  const normalizedSuffix = suffix.startsWith('/') ? suffix : `/${suffix}`

  if (normalizedBase.toLowerCase().endsWith(normalizedSuffix.toLowerCase())) {
    return normalizedBase
  }

  return `${normalizedBase}${normalizedSuffix}`
}

function isLocalHostname(hostname?: string | null): boolean {
  if (!hostname) {
    return false
  }

  return LOCAL_HOST_PATTERN.test(hostname)
}

function isLikelyProxyPath(pathname?: string | null): boolean {
  if (!pathname) {
    return false
  }

  return pathname.includes('/apps/') && pathname.includes('/proxy/')
}

export function isProxyContext(windowObject?: WindowLike): boolean {
  const win = windowObject ?? getWindowLike()
  if (!win) {
    return false
  }

  if (typeof win.__APP_MONITOR_PROXY_INFO__ !== 'undefined') {
    return true
  }

  const pathname = win.location?.pathname
  return isLikelyProxyPath(pathname)
}

export function resolveApiBase(options: ResolveApiBaseOptions = {}): string {
  const {
    explicitUrl,
    defaultPort = DEFAULT_PORT,
    apiSuffix = DEFAULT_SUFFIX,
    appendSuffix: shouldAppendSuffix = false,
  } = options

  const win = getWindowLike(options)
  const location = win?.location
  const hostname = location?.hostname
  const origin = location?.origin
  const pathname = location?.pathname
  const hasProxyBootstrap = typeof win?.__APP_MONITOR_PROXY_INFO__ !== 'undefined'
  const proxiedPath = isLikelyProxyPath(pathname)
  const remoteHost = hostname ? !isLocalHostname(hostname) : false

  const ensureSuffixIfNeeded = (base?: string) => {
    const candidate = base && base.trim().length > 0 ? base : `http://${LOOPBACK_HOST}:${defaultPort}`
    return shouldAppendSuffix ? appendSuffix(candidate, apiSuffix) : normalizeBase(candidate)
  }

  if (explicitUrl && explicitUrl.trim().length > 0) {
    const normalizedExplicit = normalizeBase(explicitUrl)

    if (remoteHost && !hasProxyBootstrap && !proxiedPath && LOCAL_HOST_PATTERN.test(normalizedExplicit)) {
      return ensureSuffixIfNeeded(origin ?? normalizedExplicit)
    }

    return ensureSuffixIfNeeded(normalizedExplicit)
  }

  if (remoteHost && !hasProxyBootstrap && !proxiedPath) {
    return ensureSuffixIfNeeded(origin)
  }

  if (hasProxyBootstrap || proxiedPath) {
    return ensureSuffixIfNeeded(`http://${LOOPBACK_HOST}:${defaultPort}`)
  }

  return ensureSuffixIfNeeded(`http://${LOOPBACK_HOST}:${defaultPort}`)
}

export function buildApiUrl(path: string, options: BuildApiUrlOptions = {}): string {
  const baseUrl = options.baseUrl ?? resolveApiBase({ ...options, appendSuffix: options.appendSuffix ?? false })
  const normalizedBase = normalizeBase(baseUrl)
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${normalizedBase}${normalizedPath}`
}

export const apiBaseUtils = {
  resolveApiBase,
  buildApiUrl,
  isProxyContext,
}

export type { WindowLike as ApiBaseWindowLike }
