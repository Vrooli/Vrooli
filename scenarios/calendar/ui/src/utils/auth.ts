const DEFAULT_AUTH_PORT = '15785'

const stripTrailingSlash = (value?: string | null): string => {
  if (!value) {
    return ''
  }
  return value.replace(/\/+$/u, '')
}

const readEnvCandidate = (key: string): string | undefined => {
  const raw = (import.meta as unknown as { env?: Record<string, unknown> }).env?.[key]
  if (typeof raw === 'string' && raw.trim()) {
    return stripTrailingSlash(raw.trim())
  }
  return undefined
}

const ENV_CANDIDATE_KEYS = [
  'VITE_AUTH_UI_URL',
  'VITE_AUTH_URL',
  'VITE_AUTH_BASE_URL',
  'VITE_AUTHENTICATOR_URL',
  'VITE_SCENARIO_AUTHENTICATOR_URL',
  'VITE_SCENARIO_AUTH_URL'
]

const resolveExplicitAuthBase = (): string | undefined => {
  for (const key of ENV_CANDIDATE_KEYS) {
    const candidate = readEnvCandidate(key)
    if (candidate) {
      return candidate
    }
  }
  return undefined
}

const LOCAL_HOST_PATTERN = /^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)/iu

const isLocalHostname = (value?: string | null): boolean => {
  if (!value) {
    return false
  }

  const trimmed = value.trim().toLowerCase()
  if (!trimmed) {
    return false
  }

  if (LOCAL_HOST_PATTERN.test(trimmed)) {
    return true
  }

  try {
    const parsed = new URL(trimmed)
    return LOCAL_HOST_PATTERN.test(parsed.hostname)
  } catch {
    const fromString = trimmed.replace(/^[a-zA-Z]+:\/\//u, '').split(/[/?#:]/u)[0]
    return LOCAL_HOST_PATTERN.test(fromString)
  }
}

const normalizeCandidate = (candidate?: string | null): string | undefined => {
  if (!candidate) {
    return undefined
  }

  const trimmed = candidate.trim()
  if (!trimmed) {
    return undefined
  }

  if (/^[a-zA-Z][a-zA-Z0-9+-.]*:\/\//u.test(trimmed)) {
    return stripTrailingSlash(trimmed)
  }

  if (trimmed.startsWith('//')) {
    const protocol =
      (typeof window !== 'undefined' && window.location?.protocol) || 'https:'
    return stripTrailingSlash(`${protocol}${trimmed}`)
  }

  if (typeof window !== 'undefined' && window.location?.origin) {
    const origin = stripTrailingSlash(window.location.origin)
    const prefix = trimmed.startsWith('/') ? '' : '/'
    return stripTrailingSlash(`${origin}${prefix}${trimmed}`)
  }

  return stripTrailingSlash(trimmed)
}

const extractHostname = (value: string): string | undefined => {
  if (!value) {
    return undefined
  }

  try {
    const parsed = new URL(value)
    return parsed.hostname
  } catch {
    if (value.startsWith('//')) {
      try {
        const parsed = new URL(`https:${value}`)
        return parsed.hostname
      } catch {
        return undefined
      }
    }

    if (/^[a-zA-Z][a-zA-Z0-9+-.]*:/u.test(value)) {
      try {
        const parsed = new URL(value)
        return parsed.hostname
      } catch {
        return undefined
      }
    }

    const normalized = value.replace(/^[a-zA-Z]+:\/\//u, '').split(/[/?#:]/u)[0]
    return normalized || undefined
  }
}

type ProxyAwareWindow = Window & {
  __APP_MONITOR_PROXY_INFO__?: unknown
  __APP_MONITOR_PROXY_INDEX__?: unknown
}

const collectAuthenticatorCandidates = (
  source: unknown,
  results: Set<string>,
  visited: Set<unknown>
): void => {
  if (!source || visited.has(source)) {
    return
  }

  visited.add(source)

  if (typeof source === 'string') {
    const trimmed = source.trim()
    if (trimmed && /scenario-authenticator/iu.test(trimmed)) {
      results.add(trimmed)
    }
    return
  }

  if (Array.isArray(source)) {
    source.forEach(entry => collectAuthenticatorCandidates(entry, results, visited))
    return
  }

  if (source instanceof Map) {
    source.forEach(entry => collectAuthenticatorCandidates(entry, results, visited))
    return
  }

  if (typeof source === 'object') {
    const record = source as Record<string, unknown>
    const prioritizedKeys = ['url', 'path', 'target', 'href', 'value', 'endpoint']
    prioritizedKeys.forEach(key => {
      if (key in record) {
        collectAuthenticatorCandidates(record[key], results, visited)
      }
    })

    Object.keys(record).forEach(key => {
      if (prioritizedKeys.includes(key)) {
        return
      }
      collectAuthenticatorCandidates(record[key], results, visited)
    })
  }
}

const scoreProxyCandidate = (value: string): number => {
  const normalized = value.trim().toLowerCase()
  let score = 0

  if (normalized.startsWith('http://') || normalized.startsWith('https://')) {
    score += 4
  }
  if (normalized.includes('/proxy')) {
    score += 3
  }
  if (normalized.includes('/apps/')) {
    score += 2
  }
  if (normalized.includes('auth')) {
    score += 1
  }

  return score
}

const resolveProxyAuthBase = (): string | undefined => {
  if (typeof window === 'undefined') {
    return undefined
  }

  const results = new Set<string>()
  const visited = new Set<unknown>()

  const collectFromWindow = (candidate?: Window | null): void => {
    if (!candidate) {
      return
    }

    try {
      const scoped = candidate as ProxyAwareWindow
      collectAuthenticatorCandidates(scoped.__APP_MONITOR_PROXY_INFO__, results, visited)
      collectAuthenticatorCandidates(scoped.__APP_MONITOR_PROXY_INDEX__, results, visited)
    } catch {
      // Accessing parent/top windows can throw for cross-origin frames.
    }
  }

  collectFromWindow(window)
  collectFromWindow(window.parent)
  collectFromWindow(window.top)

  const candidates = Array.from(results).sort((a, b) => scoreProxyCandidate(b) - scoreProxyCandidate(a))
  for (const raw of candidates) {
    const normalized = normalizeCandidate(raw)
    if (normalized) {
      return normalized
    }
  }

  const origin = window.location?.origin
  const pathname = window.location?.pathname || ''
  if (origin && pathname.includes('/apps/')) {
    return stripTrailingSlash(`${stripTrailingSlash(origin)}/apps/scenario-authenticator/proxy`)
  }

  return undefined
}

const normalizePort = (value?: string | null): string | undefined => {
  if (!value) {
    return undefined
  }

  const trimmed = value.trim()
  return /^\d+$/u.test(trimmed) ? trimmed : undefined
}

const resolveAuthPort = (): string => {
  const envPort = normalizePort(readEnvCandidate('VITE_AUTH_PORT'))
  return envPort ?? DEFAULT_AUTH_PORT
}

const resolveLoopbackAuthBase = (): string => {
  const port = resolveAuthPort()

  if (typeof window === 'undefined') {
    return `http://127.0.0.1:${port}`
  }

  const { origin, protocol, hostname } = window.location ?? {}
  const normalizedOrigin = origin ? stripTrailingSlash(origin) : undefined

  if (normalizedOrigin && hostname && isLocalHostname(hostname)) {
    try {
      const url = new URL(normalizedOrigin)
      url.port = port
      return stripTrailingSlash(url.toString())
    } catch (error) {
      console.warn('[Calendar] Failed to reuse window origin for auth base:', error)
    }
  }

  const scheme = protocol === 'https:' ? 'https' : 'http'
  return `${scheme}://127.0.0.1:${port}`
}

export const resolveAuthenticatorBaseUrl = (): string => {
  const explicit = resolveExplicitAuthBase()
  const proxyBase = resolveProxyAuthBase()
  const normalizedExplicit = normalizeCandidate(explicit)
  const remoteHost = typeof window !== 'undefined' && !isLocalHostname(window.location?.hostname)
  const explicitHostname = normalizedExplicit ? extractHostname(normalizedExplicit) : undefined

  if (
    normalizedExplicit &&
    (!remoteHost || (explicitHostname && !isLocalHostname(explicitHostname)))
  ) {
    return normalizedExplicit
  }

  if (proxyBase) {
    return proxyBase
  }

  if (normalizedExplicit) {
    return normalizedExplicit
  }

  return resolveLoopbackAuthBase()
}

export const resolveAuthenticatorLoginUrl = (options?: { redirectTo?: string }): string => {
  const base = resolveAuthenticatorBaseUrl()
  const loginPath = '/auth/login'
  const redirectTo = options?.redirectTo || (typeof window !== 'undefined' ? window.location.href : '')
  const redirectSuffix = redirectTo ? `?redirect=${encodeURIComponent(redirectTo)}` : ''
  return `${base}${loginPath}${redirectSuffix}`
}

const AUTH_TOKEN_KEYS = ['auth_token', 'authToken']
const REFRESH_TOKEN_KEYS = ['refresh_token', 'refreshToken']
const USER_KEYS = ['user']

const getStorageChain = (): Storage[] => {
  if (typeof window === 'undefined') {
    return []
  }
  const storages: Storage[] = []
  if (window.localStorage) {
    storages.push(window.localStorage)
  }
  if (window.sessionStorage) {
    storages.push(window.sessionStorage)
  }
  return storages
}

const readFromStorages = (keys: string[]): string | null => {
  for (const storage of getStorageChain()) {
    for (const key of keys) {
      try {
        const value = storage.getItem(key)
        if (value && value.trim()) {
          return value
        }
      } catch (error) {
        console.warn('[Calendar] Unable to access auth storage key:', key, error)
      }
    }
  }
  return null
}

export const getStoredAuthToken = (): string | null => readFromStorages(AUTH_TOKEN_KEYS)

export const getStoredRefreshToken = (): string | null => readFromStorages(REFRESH_TOKEN_KEYS)

export const getStoredAuthUser = (): Record<string, unknown> | null => {
  const raw = readFromStorages(USER_KEYS)
  if (!raw) {
    return null
  }
  try {
    return JSON.parse(raw) as Record<string, unknown>
  } catch (error) {
    console.warn('[Calendar] Failed to parse stored auth user', error)
    return null
  }
}

export const clearStoredAuth = (): void => {
  for (const storage of getStorageChain()) {
    try {
      for (const key of [...AUTH_TOKEN_KEYS, ...REFRESH_TOKEN_KEYS, ...USER_KEYS, 'authData']) {
        storage.removeItem(key)
      }
    } catch (error) {
      console.warn('[Calendar] Failed to clear auth storage', error)
    }
  }
}

export const extractUserIdentifiers = (
  user: Record<string, unknown> | null
): { userId?: string; email?: string } => {
  if (!user) {
    return {}
  }

  const maybeString = (value: unknown): string | undefined => {
    if (typeof value === 'string' && value.trim()) {
      return value.trim()
    }
    return undefined
  }

  const userIdCandidates = [
    maybeString(user.user_id),
    maybeString(user.userId),
    maybeString(user.id),
    maybeString(user.auth_user_id),
    maybeString(user.authUserId),
    maybeString(user.uuid)
  ]
    .filter(Boolean) as string[]

  const emailCandidates = [
    maybeString(user.email),
    maybeString(user.user_email),
    maybeString(user.contact_email)
  ].filter(Boolean) as string[]

  return {
    userId: userIdCandidates[0],
    email: emailCandidates[0]
  }
}

export const dispatchAuthRequiredEvent = (loginUrl?: string): void => {
  if (typeof window === 'undefined' || typeof window.dispatchEvent !== 'function') {
    return
  }

  const detail = loginUrl ? { loginUrl } : undefined
  const event = new CustomEvent('calendar:auth-required', { detail })
  window.dispatchEvent(event)
}
