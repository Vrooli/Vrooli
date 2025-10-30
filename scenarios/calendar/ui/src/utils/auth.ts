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

const resolveAppMonitorAuthBase = (): string | undefined => {
  if (typeof window === 'undefined') {
    return undefined
  }

  const pathname = window.location?.pathname || ''
  const match = pathname.match(/^\/apps\/[^/]+\/proxy/iu)
  if (!match) {
    return undefined
  }

  return stripTrailingSlash(`${window.location.origin}/apps/scenario-authenticator/proxy`)
}

const resolveLocalAuthBase = (): string => {
  if (typeof window === 'undefined') {
    return `http://localhost:${DEFAULT_AUTH_PORT}`
  }

  const envPort = readEnvCandidate('VITE_AUTH_PORT')
  const port = envPort || DEFAULT_AUTH_PORT

  try {
    const url = new URL(window.location.origin)
    url.port = port
    return stripTrailingSlash(url.toString())
  } catch (error) {
    console.warn('[Calendar] Unable to construct auth base from origin, falling back to localhost:', error)
    return `http://localhost:${port}`
  }
}

export const resolveAuthenticatorBaseUrl = (): string => {
  const explicit = resolveExplicitAuthBase()
  if (explicit) {
    return explicit
  }

  const monitorBase = resolveAppMonitorAuthBase()
  if (monitorBase) {
    return monitorBase
  }

  return resolveLocalAuthBase()
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
