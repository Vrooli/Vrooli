/**
 * Runtime configuration utilities
 *
 * Functions for fetching and using runtime configuration from the scenario server.
 * This solves the build-time vs runtime configuration problem for production bundles.
 */

import { DEFAULT_CONFIG_ENDPOINT, DEFAULT_CONFIG_GLOBAL } from '../shared/constants.js'
import type { ScenarioConfig, ResolveOptions } from '../shared/types.js'
import { getWindowLike } from '../shared/utils.js'
import { resolveApiBase } from './resolve.js'

/**
 * Fetch runtime configuration from server
 *
 * Fetches the /config endpoint (or custom endpoint) to get runtime configuration
 * values like API URLs and ports. This is essential for production bundles that
 * can't access process.env at build time.
 *
 * @param options - Fetch options
 * @param options.endpoint - Config endpoint path (default: "./config")
 * @param options.windowObject - Custom window object
 * @param options.timeout - Request timeout in milliseconds
 * @returns Scenario configuration or null if fetch fails
 *
 * @example
 * ```typescript
 * const config = await fetchRuntimeConfig()
 * if (config) {
 *   console.log(`API URL: ${config.apiUrl}`)
 *   console.log(`WS URL: ${config.wsUrl}`)
 * }
 * ```
 */
export async function fetchRuntimeConfig(options: {
  endpoint?: string
  windowObject?: ReturnType<typeof getWindowLike>
  timeout?: number
} = {}): Promise<ScenarioConfig | null> {
  const {
    endpoint = DEFAULT_CONFIG_ENDPOINT,
    windowObject,
    timeout = 5000,
  } = options

  // Only works in browser environment
  if (typeof fetch !== 'function') {
    return null
  }

  try {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), timeout)

    const response = await fetch(endpoint, {
      headers: { accept: 'application/json' },
      signal: controller.signal,
    })

    clearTimeout(timeoutId)

    if (!response.ok) {
      console.warn(`[api-base] Config fetch failed: ${response.status}`)
      return null
    }

    const config = await response.json() as ScenarioConfig
    return config
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unknown error'
    console.warn(`[api-base] Failed to fetch runtime config: ${message}`)
    return null
  }
}

/**
 * Get runtime configuration from injected global
 *
 * Checks for configuration injected directly into the HTML by the server.
 * This is faster than fetching but requires the server to inject it.
 *
 * @param options - Retrieval options
 * @param options.windowObject - Custom window object
 * @param options.configGlobalName - Global variable name (default: "__VROOLI_CONFIG__")
 * @returns Scenario configuration or null
 *
 * @example
 * ```typescript
 * const config = getInjectedConfig()
 * if (config) {
 *   // Use injected config (no fetch needed)
 * } else {
 *   // Fallback to fetch
 *   const config = await fetchRuntimeConfig()
 * }
 * ```
 */
export function getInjectedConfig(options: {
  windowObject?: ReturnType<typeof getWindowLike>
  configGlobalName?: string
} = {}): ScenarioConfig | null {
  const {
    configGlobalName = DEFAULT_CONFIG_GLOBAL,
  } = options

  const win = options.windowObject ?? getWindowLike()
  if (!win) {
    return null
  }

  const value = win[configGlobalName]
  if (value && typeof value === 'object') {
    return value as ScenarioConfig
  }

  return null
}

/**
 * Resolve API base with runtime configuration
 *
 * Combines automatic resolution with runtime configuration fetching.
 * This is the recommended way to resolve API URLs in production bundles.
 *
 * Resolution strategy:
 * 1. Check for injected config global
 * 2. Try standard resolution (proxy metadata, path detection)
 * 3. If standard resolution returns localhost, try fetching runtime config
 * 4. Use config.apiUrl if available, otherwise fall back to resolution
 *
 * @param options - Resolution options
 * @returns Resolved API base URL
 *
 * @example
 * ```typescript
 * // Automatically tries config fetch if needed
 * const apiBase = await resolveWithConfig({
 *   defaultPort: '8080',
 *   appendSuffix: true
 * })
 *
 * // Now use it for requests
 * const response = await fetch(`${apiBase}/health`)
 * ```
 */
export async function resolveWithConfig(options: ResolveOptions = {}): Promise<string> {
  // 1. Check for injected config
  const injectedConfig = getInjectedConfig({
    windowObject: options.windowObject,
    configGlobalName: options.configGlobalName,
  })

  if (injectedConfig?.apiUrl) {
    return injectedConfig.apiUrl
  }

  // 2. Try standard resolution
  const resolved = resolveApiBase(options)

  // 3. If resolved to localhost, try fetching runtime config
  const win = getWindowLike(options)
  const hostname = win?.location?.hostname
  const isRemote = hostname && !hostname.match(/^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)$/i)

  if (isRemote && resolved.includes('127.0.0.1')) {
    // We're on a remote host but resolved to localhost - try config fetch
    const fetchedConfig = await fetchRuntimeConfig({
      endpoint: options.configEndpoint,
      windowObject: options.windowObject,
    })

    if (fetchedConfig?.apiUrl) {
      return fetchedConfig.apiUrl
    }
  }

  // 4. Fall back to standard resolution
  return resolved
}

/**
 * Create a configuration cache
 *
 * Caches configuration to avoid repeated fetches. Useful for applications
 * that need to resolve URLs multiple times.
 *
 * @returns Configuration cache object
 *
 * @example
 * ```typescript
 * const configCache = createConfigCache()
 *
 * // First call fetches
 * const apiBase1 = await configCache.resolve({ defaultPort: '8080' })
 *
 * // Second call uses cache
 * const apiBase2 = await configCache.resolve({ defaultPort: '8080' })
 *
 * // Force refresh
 * configCache.clear()
 * const apiBase3 = await configCache.resolve({ defaultPort: '8080' })
 * ```
 */
export function createConfigCache() {
  let cachedConfig: ScenarioConfig | null = null
  let cacheTimestamp: number = 0
  const cacheDuration = 60000 // 1 minute

  return {
    /**
     * Resolve API base with caching
     */
    resolve: async (options: ResolveOptions = {}): Promise<string> => {
      // Check cache validity
      const now = Date.now()
      if (cachedConfig && (now - cacheTimestamp) < cacheDuration) {
        return cachedConfig.apiUrl
      }

      // Try to get fresh config
      const injected = getInjectedConfig(options)
      if (injected) {
        cachedConfig = injected
        cacheTimestamp = now
        return injected.apiUrl
      }

      const fetched = await fetchRuntimeConfig(options)
      if (fetched) {
        cachedConfig = fetched
        cacheTimestamp = now
        return fetched.apiUrl
      }

      // Fall back to standard resolution
      return resolveApiBase(options)
    },

    /**
     * Clear cached configuration
     */
    clear: () => {
      cachedConfig = null
      cacheTimestamp = 0
    },

    /**
     * Get cached configuration without fetching
     */
    get: () => cachedConfig,
  }
}
