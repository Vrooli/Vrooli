/**
 * API and WebSocket base URL resolution
 *
 * Core resolution logic for determining the correct base URL for API requests
 * across different deployment contexts (localhost, remote, proxied).
 */

import {
  DEFAULT_API_PORT,
  DEFAULT_API_SUFFIX,
  DEFAULT_WS_SUFFIX,
  LOOPBACK_HOST,
  PROXY_PATH_MARKER,
  APPS_PATH_MARKER,
} from '../shared/constants.js'
import type { ResolveOptions, PortEntry } from '../shared/types.js'
import {
  getWindowLike,
  normalizeBase,
  normalizeOrigin,
  appendSuffix,
  isLocalHostname,
  tryParseHostname,
  toAbsoluteCandidate,
  resolveProtocolHint,
  toTrimmedString,
  stringLooksLikeUrlOrPath,
  convertHttpToWs,
} from '../shared/utils.js'
import { getProxyInfo, getProxyIndex, isProxyContext } from './detect.js'

/**
 * Collect proxy path candidates from metadata
 *
 * Recursively searches through proxy metadata to find potential proxy paths.
 *
 * @internal
 */
function collectProxyCandidates(
  source: unknown,
  addCandidate: (value: string) => void,
  visited: Set<unknown>
): void {
  if (!source || visited.has(source)) {
    return
  }

  if (typeof source === 'string') {
    visited.add(source)
    const candidate = toTrimmedString(source)
    if (candidate && stringLooksLikeUrlOrPath(candidate)) {
      addCandidate(candidate)
    }
    return
  }

  if (Array.isArray(source)) {
    visited.add(source)
    source.forEach(item => collectProxyCandidates(item, addCandidate, visited))
    return
  }

  if (source instanceof Map) {
    visited.add(source)
    source.forEach(value => collectProxyCandidates(value, addCandidate, visited))
    return
  }

  if (typeof source === 'object') {
    visited.add(source)
    const record = source as Record<string, unknown>

    // Prioritize specific keys
    const prioritizedKeys: Array<keyof typeof record> = ['url', 'href', 'path', 'target']
    prioritizedKeys.forEach(key => {
      if (key in record) {
        const value = record[key]
        const direct = toTrimmedString(value)
        if (direct && stringLooksLikeUrlOrPath(direct)) {
          addCandidate(direct)
        }
        collectProxyCandidates(value, addCandidate, visited)
      }
    })

    // Check nested structures
    const nestedKeys: Array<keyof typeof record> = ['endpoints', 'ports', 'primary', 'value', 'entries']
    nestedKeys.forEach(key => {
      if (key in record) {
        collectProxyCandidates(record[key], addCandidate, visited)
      }
    })

    // Check alias map
    const aliasMap = (record as { aliasMap?: unknown }).aliasMap
    if (aliasMap) {
      collectProxyCandidates(aliasMap, addCandidate, visited)
    }

    // Check remaining keys
    Object.keys(record).forEach(key => {
      if (prioritizedKeys.includes(key as keyof typeof record)) {
        return
      }
      if (nestedKeys.includes(key as keyof typeof record)) {
        return
      }
      if (key === 'aliasMap' || key === 'aliases') {
        return
      }
      collectProxyCandidates(record[key], addCandidate, visited)
    })
  }
}

/**
 * Resolve proxy metadata base URL
 *
 * Attempts to extract a valid proxy base URL from injected proxy metadata.
 *
 * @internal
 */
function resolveProxyMetadataBase(
  win: ReturnType<typeof getWindowLike>,
  remoteHost: boolean,
  proxyGlobalNames?: string[]
): string | undefined {
  if (!win) {
    return undefined
  }

  const origin = normalizeOrigin(win.location?.origin)
  const protocolHint = resolveProtocolHint(origin, win.location?.protocol)

  const seenValues = new Set<string>()
  const visited = new Set<unknown>()
  const candidates: string[] = []

  const pushCandidate = (value: string) => {
    if (seenValues.has(value)) {
      return
    }
    seenValues.add(value)
    candidates.push(value)
  }

  // Collect from both INFO and INDEX globals
  const proxyInfo = getProxyInfo({ windowObject: win, proxyGlobalNames })
  const proxyIndex = getProxyIndex({ windowObject: win, proxyGlobalNames })

  collectProxyCandidates(proxyInfo, pushCandidate, visited)
  collectProxyCandidates(proxyIndex, pushCandidate, visited)

  // Try each candidate
  for (const candidate of candidates) {
    const absolute = toAbsoluteCandidate(candidate, origin, protocolHint)
    if (!absolute) {
      continue
    }

    // Skip localhost URLs when on remote host
    if (remoteHost) {
      const host = tryParseHostname(absolute)
      if (host && isLocalHostname(host)) {
        continue
      }
    }

    return absolute
  }

  return undefined
}

/**
 * Derive proxy base from pathname
 *
 * Extracts proxy base URL by looking for /proxy marker in the pathname.
 * Works with any proxy pattern, not just /apps/.
 *
 * @internal
 */
function deriveProxyBaseFromPath(origin?: string, pathname?: string): string | undefined {
  if (!origin || !pathname) {
    return undefined
  }

  const proxyMarker = PROXY_PATH_MARKER
  const index = pathname.indexOf(proxyMarker)
  if (index === -1) {
    return undefined
  }

  // Find end of proxy segment (next / or end of string)
  let sliceEnd = pathname.indexOf('/', index + proxyMarker.length)
  if (sliceEnd === -1) {
    sliceEnd = pathname.length
  }

  const basePath = pathname.slice(0, sliceEnd)
  const normalizedOrigin = normalizeOrigin(origin)
  if (!normalizedOrigin) {
    return undefined
  }

  return normalizeBase(`${normalizedOrigin}${basePath}`)
}

/**
 * Derive proxy base from app shell pattern
 *
 * Handles the legacy /apps/{scenario}/proxy pattern used by app-monitor.
 * This is a convenience helper for backwards compatibility.
 *
 * @internal
 */
function deriveProxyBaseFromAppShell(
  origin?: string,
  pathname?: string,
  remoteHost?: boolean
): string | undefined {
  // Only apply this pattern on remote hosts
  if (!remoteHost) {
    return undefined
  }

  if (!origin || !pathname) {
    return undefined
  }

  // Check for /apps/ pattern
  if (!pathname.includes(APPS_PATH_MARKER)) {
    return undefined
  }

  const normalizedOrigin = normalizeOrigin(origin)
  if (!normalizedOrigin) {
    return undefined
  }

  const marker = APPS_PATH_MARKER
  const index = pathname.indexOf(marker)
  if (index === -1) {
    return undefined
  }

  // Extract scenario slug: /apps/{slug}/...
  const remainder = pathname.slice(index + marker.length)
  const segments = remainder.split('/').filter(Boolean)
  if (segments.length === 0) {
    return undefined
  }

  const appSlug = segments[0]
  if (!appSlug) {
    return undefined
  }

  // Construct: {origin}/apps/{slug}/proxy
  return normalizeBase(`${normalizedOrigin}${marker}${appSlug}/proxy`)
}

/**
 * Resolve API base URL
 *
 * Determines the correct base URL for API requests based on the current context.
 * Works in localhost, remote domains, proxied contexts, and secure tunnels.
 *
 * Resolution priority:
 * 1. Explicit URL (if provided)
 * 2. Proxy metadata globals (window.__VROOLI_PROXY_INFO__)
 * 3. Path-based proxy detection (/proxy in pathname)
 * 4. App shell pattern (/apps/{scenario}/proxy)
 * 5. Remote host origin
 * 6. Localhost fallback
 *
 * @param options - Resolution options
 * @returns Resolved API base URL
 *
 * @example
 * ```typescript
 * // Basic usage
 * const apiBase = resolveApiBase({ defaultPort: '8080' })
 * // → http://127.0.0.1:8080
 *
 * // With suffix
 * const apiBase = resolveApiBase({
 *   defaultPort: '8080',
 *   appendSuffix: true
 * })
 * // → http://127.0.0.1:8080/api/v1
 *
 * // Explicit URL
 * const apiBase = resolveApiBase({
 *   explicitUrl: 'https://api.example.com'
 * })
 * // → https://api.example.com
 * ```
 */
export function resolveApiBase(options: ResolveOptions = {}): string {
  const {
    explicitUrl,
    defaultPort = DEFAULT_API_PORT,
    apiSuffix = DEFAULT_API_SUFFIX,
    appendSuffix: shouldAppendSuffix = false,
    proxyGlobalNames,
  } = options

  const win = getWindowLike(options)
  const location = win?.location
  const hostname = location?.hostname
  const origin = location?.origin
  const pathname = location?.pathname
  const hasProxyBootstrap = isProxyContext({ windowObject: win, proxyGlobalNames })
  const proxiedPath = pathname?.includes(PROXY_PATH_MARKER) ?? false
  const remoteHost = hostname ? !isLocalHostname(hostname) : false

  // Helper to ensure suffix if requested
  const ensureSuffixIfNeeded = (base?: string) => {
    const candidate = base && base.trim().length > 0 ? base : `http://${LOOPBACK_HOST}:${defaultPort}`
    return shouldAppendSuffix ? appendSuffix(candidate, apiSuffix) : normalizeBase(candidate)
  }

  // 1. Explicit URL takes highest priority
  if (explicitUrl && explicitUrl.trim().length > 0) {
    const normalizedExplicit = normalizeBase(explicitUrl)

    // If on remote host without proxy context, but explicit URL is localhost,
    // use current origin instead (prevents CORS issues)
    if (remoteHost && !hasProxyBootstrap && !proxiedPath && isLocalHostname(normalizedExplicit)) {
      return ensureSuffixIfNeeded(origin ?? normalizedExplicit)
    }

    return ensureSuffixIfNeeded(normalizedExplicit)
  }

  // 2. Try proxy metadata
  const proxyMetadataBase = resolveProxyMetadataBase(win, remoteHost, proxyGlobalNames)
  if (proxyMetadataBase) {
    return ensureSuffixIfNeeded(proxyMetadataBase)
  }

  // 3. Try path-based proxy detection
  const derivedProxyBase = proxiedPath ? deriveProxyBaseFromPath(origin, pathname) : undefined
  if (derivedProxyBase) {
    return ensureSuffixIfNeeded(derivedProxyBase)
  }

  // 4. Try app shell pattern (backwards compat)
  const derivedAppShellProxyBase = deriveProxyBaseFromAppShell(origin, pathname, remoteHost)
  if (derivedAppShellProxyBase) {
    return ensureSuffixIfNeeded(derivedAppShellProxyBase)
  }

  // 5. If on remote host without proxy, use current origin
  if (remoteHost && !hasProxyBootstrap && !proxiedPath) {
    return ensureSuffixIfNeeded(origin)
  }

  // 6. Localhost fallback - use current origin to go through UI server's proxy
  // Production bundles should ALWAYS proxy through the UI server, never connect directly to API
  if (origin) {
    return ensureSuffixIfNeeded(origin)
  }

  // 7. SSR/non-browser fallback only (should rarely happen)
  return ensureSuffixIfNeeded(`http://${LOOPBACK_HOST}:${defaultPort}`)
}

/**
 * Resolve WebSocket base URL
 *
 * Similar to resolveApiBase but returns a WebSocket URL (ws:// or wss://).
 * Follows the same resolution logic but converts the protocol.
 *
 * @param options - Resolution options
 * @returns Resolved WebSocket base URL
 *
 * @example
 * ```typescript
 * const wsBase = resolveWsBase({ defaultPort: '8080' })
 * // → ws://127.0.0.1:8080
 *
 * // On HTTPS site
 * const wsBase = resolveWsBase({
 *   windowObject: mockWindow({ protocol: 'https:' })
 * })
 * // → wss://...
 * ```
 */
export function resolveWsBase(options: ResolveOptions = {}): string {
  // Use custom suffix for WebSocket if not specified
  const wsOptions: ResolveOptions = {
    ...options,
    apiSuffix: options.apiSuffix ?? DEFAULT_WS_SUFFIX,
  }

  const apiBase = resolveApiBase(wsOptions)
  return convertHttpToWs(apiBase)
}
