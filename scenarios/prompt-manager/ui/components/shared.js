// Shared utilities for Prompt Manager UI
// API configuration and HTTP utilities with proxy-aware discovery

const PROXY_INFO_KEYS = ['__APP_MONITOR_PROXY_INFO__', '__APP_MONITOR_PROXY_INDEX__']
const DEFAULT_API_PORT = 15000

const trimTrailingSlashes = (value = '') => value.replace(/\/+$/, '')
const ensureProtocol = () => {
    if (typeof window !== 'undefined' && window.location?.protocol === 'https:') {
        return 'https:'
    }
    return 'http:'
}

const isLocalHostname = (hostname = '') => {
    const normalized = hostname.toLowerCase()
    return [
        'localhost',
        '127.0.0.1',
        '0.0.0.0',
        '::1',
        '[::1]'
    ].includes(normalized)
}

const isLikelyProxiedPath = (pathname = '') => pathname.includes('/apps/') && pathname.includes('/proxy/')

const fallbackLoopbackOrigin = () => `${ensureProtocol()}//127.0.0.1:${DEFAULT_API_PORT}`

const pushProxyCandidate = (list, candidate) => {
    if (!candidate) {
        return
    }

    if (typeof candidate === 'string') {
        list.push(candidate)
        return
    }

    if (typeof candidate === 'object') {
        pushProxyCandidate(list, candidate.url || candidate.href || candidate.path || candidate.target)
    }
}

const resolveProxyCandidate = (info) => {
    if (!info) {
        return undefined
    }

    const candidates = []

    if (Array.isArray(info.endpoints)) {
        info.endpoints.forEach((endpoint) => pushProxyCandidate(candidates, endpoint))
    }

    pushProxyCandidate(candidates, info.api)
    pushProxyCandidate(candidates, info.apiUrl || info.apiURL)
    pushProxyCandidate(candidates, info.baseUrl || info.baseURL)
    pushProxyCandidate(candidates, info.url)
    pushProxyCandidate(candidates, info.target)

    if (typeof info === 'string') {
        candidates.push(info)
    }

    for (const rawCandidate of candidates) {
        if (typeof rawCandidate !== 'string') {
            continue
        }

        const candidate = rawCandidate.trim()
        if (!candidate) {
            continue
        }

        if (/^https?:\/\//i.test(candidate)) {
            return trimTrailingSlashes(candidate)
        }

        if (candidate.startsWith('/') && typeof window !== 'undefined' && window.location?.origin) {
            return trimTrailingSlashes(`${window.location.origin}${candidate}`)
        }
    }

    return undefined
}

const resolveProxyAwareBaseUrl = (explicit) => {
    const explicitValue = typeof explicit === 'string' ? explicit.trim() : ''
    const normalizedExplicit = explicitValue ? trimTrailingSlashes(explicitValue) : ''

    if (typeof window !== 'undefined') {
        for (const key of PROXY_INFO_KEYS) {
            const proxyCandidate = resolveProxyCandidate(window[key])
            if (proxyCandidate) {
                return proxyCandidate
            }
        }

        const { origin, hostname, pathname } = window.location || {}
        const hasProxyBootstrap = PROXY_INFO_KEYS.some((key) => typeof window[key] !== 'undefined')
        const proxiedPath = isLikelyProxiedPath(pathname || '')

        if (normalizedExplicit) {
            if (origin && hostname && !isLocalHostname(hostname) && !hasProxyBootstrap && !proxiedPath) {
                if (/localhost|127\.0\.0\.1|0\.0\.0\.0/i.test(normalizedExplicit)) {
                    return trimTrailingSlashes(origin)
                }
            }
            return normalizedExplicit
        }

        if (origin && hostname && !isLocalHostname(hostname) && !hasProxyBootstrap && !proxiedPath) {
            return trimTrailingSlashes(origin)
        }

        if (origin) {
            return trimTrailingSlashes(origin)
        }
    }

    if (normalizedExplicit) {
        return normalizedExplicit
    }

    return trimTrailingSlashes(fallbackLoopbackOrigin())
}

let API_URL = resolveProxyAwareBaseUrl(null)

const hydrateApiConfig = async () => {
    try {
        const response = await fetch('/api/config')
        if (!response.ok) {
            throw new Error(`Config request failed with status ${response.status}`)
        }
        const config = await response.json()
        API_URL = resolveProxyAwareBaseUrl(config?.apiUrl)
    } catch (error) {
        console.error('Failed to fetch config:', error)
        API_URL = resolveProxyAwareBaseUrl(API_URL)
    }

    if (typeof window !== 'undefined') {
        window.__PROMPT_MANAGER_API_URL__ = API_URL
        window.__PROMPT_MANAGER_RESOLVE_API__ = resolveProxyAwareBaseUrl
    }
}

hydrateApiConfig()

const api = {
    get: async (endpoint) => {
        const response = await fetch(`${API_URL}${endpoint}`)
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`)
        return response.json()
    },
    post: async (endpoint, data) => {
        const response = await fetch(`${API_URL}${endpoint}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        })
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`)
        return response.json()
    },
    put: async (endpoint, data) => {
        const response = await fetch(`${API_URL}${endpoint}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        })
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`)
        return response.json()
    },
    delete: async (endpoint) => {
        const response = await fetch(`${API_URL}${endpoint}`, {
            method: 'DELETE'
        })
        if (!response.ok) throw new Error(`HTTP error! status: ${response.status}`)
        return response.status === 204 ? {} : response.json()
    }
}

// Make API available globally for components
if (typeof window !== 'undefined') {
    window.PromptManagerAPI = api
}
