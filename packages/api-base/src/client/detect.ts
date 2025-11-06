/**
 * Proxy detection utilities
 *
 * Functions to detect if code is running in a proxy context and extract proxy metadata.
 */

import { DEFAULT_PROXY_GLOBALS } from '../shared/constants.js'
import type { ProxyInfo, ProxyIndex, WindowLike } from '../shared/types.js'
import { getWindowLike, isLikelyProxyPath } from '../shared/utils.js'

/**
 * Check if currently running in a proxy context
 *
 * Determines whether the scenario is being accessed through a proxy (e.g., embedded
 * in app-monitor or another host scenario) by checking for proxy metadata globals
 * or proxy-like URL patterns.
 *
 * @param windowObjectOrOptions - Window object or options object
 * @param proxyGlobalNames - Custom global names (legacy parameter)
 * @returns True if in proxy context
 *
 * @example
 * ```typescript
 * if (isProxyContext()) {
 *   console.log('Running in embedded/proxied context')
 * } else {
 *   console.log('Running standalone')
 * }
 * ```
 */
export function isProxyContext(
  windowObjectOrOptions?: WindowLike | { windowObject?: WindowLike; proxyGlobalNames?: string[] },
  proxyGlobalNames?: string[]
): boolean {
  // Handle different call signatures for backwards compatibility
  let win: WindowLike | undefined
  let globalsToCheck: readonly string[]

  if (!windowObjectOrOptions) {
    win = getWindowLike()
    globalsToCheck = DEFAULT_PROXY_GLOBALS
  } else if (typeof windowObjectOrOptions === 'object' && ('windowObject' in windowObjectOrOptions || 'proxyGlobalNames' in windowObjectOrOptions)) {
    // Options object
    const opts = windowObjectOrOptions as { windowObject?: WindowLike; proxyGlobalNames?: string[] }
    win = opts.windowObject ?? getWindowLike()
    globalsToCheck = opts.proxyGlobalNames ?? DEFAULT_PROXY_GLOBALS
  } else {
    // Direct window object (legacy)
    win = windowObjectOrOptions as WindowLike
    globalsToCheck = proxyGlobalNames ?? DEFAULT_PROXY_GLOBALS
  }

  if (!win) {
    return false
  }

  // Check for proxy globals
  for (const globalName of globalsToCheck) {
    if (typeof win[globalName] !== 'undefined') {
      return true
    }
  }

  // Check pathname for proxy markers
  const pathname = win.location?.pathname
  return isLikelyProxyPath(pathname)
}

/**
 * Get proxy info from window globals
 *
 * Retrieves proxy metadata injected by the host scenario. Returns null if not
 * in a proxy context or if metadata is invalid.
 *
 * @param windowObjectOrOptions - Window object or options object
 * @param proxyGlobalNames - Custom global names (legacy parameter)
 * @returns Proxy metadata or null
 *
 * @example
 * ```typescript
 * const proxyInfo = getProxyInfo()
 * if (proxyInfo) {
 *   console.log(`Hosted by: ${proxyInfo.hostScenario}`)
 *   console.log(`Proxy path: ${proxyInfo.basePath}`)
 * }
 * ```
 */
export function getProxyInfo(
  windowObjectOrOptions?: WindowLike | { windowObject?: WindowLike; proxyGlobalNames?: string[] },
  proxyGlobalNames?: string[]
): ProxyInfo | null {
  // Handle different call signatures for backwards compatibility
  let win: WindowLike | undefined
  let globalsToCheck: readonly string[]

  if (!windowObjectOrOptions) {
    win = getWindowLike()
    globalsToCheck = DEFAULT_PROXY_GLOBALS
  } else if (typeof windowObjectOrOptions === 'object' && ('windowObject' in windowObjectOrOptions || 'proxyGlobalNames' in windowObjectOrOptions)) {
    // Options object
    const opts = windowObjectOrOptions as { windowObject?: WindowLike; proxyGlobalNames?: string[] }
    win = opts.windowObject ?? getWindowLike()
    globalsToCheck = opts.proxyGlobalNames ?? DEFAULT_PROXY_GLOBALS
  } else {
    // Direct window object (legacy)
    win = windowObjectOrOptions as WindowLike
    globalsToCheck = proxyGlobalNames ?? DEFAULT_PROXY_GLOBALS
  }

  if (!win) {
    return null
  }

  for (const globalName of globalsToCheck) {
    const value = win[globalName]
    if (value && typeof value === 'object') {
      // Basic validation
      const info = value as ProxyInfo
      if (info.generatedAt && (info.primary || info.ports)) {
        return info
      }
    }
  }

  return null
}

/**
 * Get proxy index from window globals
 *
 * Retrieves the indexed/optimized proxy metadata. This is faster for lookups
 * but contains the same information as ProxyInfo.
 *
 * @param windowObjectOrOptions - Window object or options object
 * @param proxyGlobalNames - Custom global names (legacy parameter)
 * @returns Proxy index or null
 *
 * @example
 * ```typescript
 * const proxyIndex = getProxyIndex()
 * if (proxyIndex) {
 *   const apiEntry = proxyIndex.aliasMap.get('api')
 *   console.log(`API path: ${apiEntry?.path}`)
 * }
 * ```
 */
export function getProxyIndex(
  windowObjectOrOptions?: WindowLike | { windowObject?: WindowLike; proxyGlobalNames?: string[] },
  proxyGlobalNames?: string[]
): ProxyIndex | null {
  // Handle different call signatures for backwards compatibility
  let win: WindowLike | undefined
  let globalsToCheck: readonly string[]

  if (!windowObjectOrOptions) {
    win = getWindowLike()
    globalsToCheck = DEFAULT_PROXY_GLOBALS
  } else if (typeof windowObjectOrOptions === 'object' && ('windowObject' in windowObjectOrOptions || 'proxyGlobalNames' in windowObjectOrOptions)) {
    // Options object
    const opts = windowObjectOrOptions as { windowObject?: WindowLike; proxyGlobalNames?: string[] }
    win = opts.windowObject ?? getWindowLike()
    globalsToCheck = opts.proxyGlobalNames ?? DEFAULT_PROXY_GLOBALS
  } else {
    // Direct window object (legacy)
    win = windowObjectOrOptions as WindowLike
    globalsToCheck = proxyGlobalNames ?? DEFAULT_PROXY_GLOBALS
  }

  if (!win) {
    return null
  }

  // Try INDEX globals first, otherwise build from INFO
  const indexGlobals = globalsToCheck.filter(name => name.includes('INDEX'))

  for (const globalName of indexGlobals) {
    const value = win[globalName]
    if (value && typeof value === 'object') {
      const index = value as ProxyIndex
      if (index.aliasMap && index.primary) {
        return index
      }
    }
  }

  // Fallback: Build index from proxy info
  const proxyInfo = getProxyInfo(win, proxyGlobalNames)
  if (proxyInfo) {
    return buildProxyIndexFromInfo(proxyInfo)
  }

  return null
}

/**
 * Build a ProxyIndex from ProxyInfo
 * @internal
 */
function buildProxyIndexFromInfo(info: ProxyInfo): ProxyIndex {
  const aliasMap = new Map<string, any>()

  // Index all ports by their aliases
  for (const port of info.ports || []) {
    const aliases = port.aliases || []
    for (const alias of aliases) {
      const key = String(alias).toLowerCase()
      if (!aliasMap.has(key)) {
        aliasMap.set(key, port)
      }
    }
    // Also index by port number
    aliasMap.set(String(port.port), port)
  }

  return {
    appId: info.appId,
    generatedAt: info.generatedAt,
    aliasMap,
    primary: info.primary,
    hosts: new Set(info.hosts.map(h => h.toLowerCase())),
  }
}
