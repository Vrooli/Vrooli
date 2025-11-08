/**
 * Shared utility functions for @vrooli/api-base
 *
 * Common functions used across both client and server implementations.
 */

import { LOCAL_HOST_PATTERN, LOOPBACK_HOSTS } from './constants.js'
import type { WindowLike } from './types.js'

/**
 * Convert unknown value to trimmed string
 *
 * @param value - Value to convert
 * @returns Trimmed string or undefined if not a valid string
 */
export function toTrimmedString(value: unknown): string | undefined {
  if (typeof value !== 'string') {
    return undefined
  }

  const trimmed = value.trim()
  return trimmed.length > 0 ? trimmed : undefined
}

/**
 * Check if a string looks like a URL or path
 *
 * @param value - String to check
 * @returns True if contains : or / characters
 */
export function stringLooksLikeUrlOrPath(value: string): boolean {
  return /[:/]/u.test(value)
}

/**
 * Parse hostname from URL string
 *
 * @param candidate - URL string
 * @returns Hostname or undefined if parsing fails
 */
export function tryParseHostname(candidate: string): string | undefined {
  try {
    const url = new URL(candidate)
    return url.hostname
  } catch {
    return undefined
  }
}

/**
 * Normalize origin by removing trailing slashes
 *
 * @param value - Origin string
 * @returns Normalized origin or undefined
 */
export function normalizeOrigin(value?: string): string | undefined {
  if (!value) {
    return undefined
  }

  const trimmed = value.trim()
  if (!trimmed) {
    return undefined
  }

  return trimmed.replace(/\/+$/, '')
}

/**
 * Normalize base URL by removing trailing slashes
 *
 * @param value - Base URL
 * @returns Normalized base URL
 */
export function normalizeBase(value: string): string {
  return value.replace(/\/+$/, '')
}

/**
 * Append suffix to base URL if not already present
 *
 * @param base - Base URL
 * @param suffix - Suffix to append
 * @returns Base URL with suffix
 */
export function appendSuffix(base: string, suffix: string): string {
  const normalizedBase = normalizeBase(base)
  const normalizedSuffix = suffix.startsWith('/') ? suffix : `/${suffix}`

  // Check if suffix already present (case-insensitive)
  if (normalizedBase.toLowerCase().endsWith(normalizedSuffix.toLowerCase())) {
    return normalizedBase
  }

  return `${normalizedBase}${normalizedSuffix}`
}

/**
 * Check if hostname is a loopback/localhost address
 *
 * @param hostname - Hostname to check
 * @returns True if hostname is loopback
 */
export function isLocalHostname(hostname?: string | null): boolean {
  if (!hostname) {
    return false
  }

  let candidate = hostname

  // Try to parse as URL first
  try {
    const parsed = new URL(hostname.includes('://') ? hostname : `http://${hostname}`)
    candidate = parsed.hostname
  } catch {
    // Not a valid URL, extract hostname manually
    candidate = hostname.replace(/^[a-zA-Z]+:\/\//, '').split('/')[0]
  }

  return LOCAL_HOST_PATTERN.test(candidate)
}

/**
 * Check if pathname likely indicates a proxy context
 *
 * @param pathname - URL pathname to check
 * @returns True if pathname suggests proxy
 */
export function isLikelyProxyPath(pathname?: string | null): boolean {
  if (!pathname) {
    return false
  }

  // Generic proxy detection - any path containing /proxy is considered proxied
  // This supports app-monitor's /apps/.../proxy/ pattern as well as custom patterns
  return pathname.includes('/proxy')
}

/**
 * Resolve protocol hint from origin or explicit protocol
 *
 * @param origin - Origin URL
 * @param explicit - Explicit protocol (e.g., "https:")
 * @returns Resolved protocol
 */
export function resolveProtocolHint(origin?: string, explicit?: string): string {
  // If explicit protocol provided and valid, use it
  if (explicit && explicit.endsWith(':')) {
    return explicit
  }

  if (explicit && /^[a-zA-Z][a-zA-Z0-9+-.]*:$/u.test(explicit)) {
    return explicit
  }

  // Try to extract from origin
  if (origin) {
    try {
      return new URL(origin).protocol || 'http:'
    } catch {
      // fall through
    }
  }

  return 'http:'
}

/**
 * Convert URL string to absolute URL
 *
 * @param candidate - URL or path string
 * @param origin - Base origin for relative URLs
 * @param protocolHint - Protocol to use if not specified
 * @returns Absolute URL or undefined
 */
export function toAbsoluteCandidate(
  candidate: string,
  origin?: string,
  protocolHint?: string
): string | undefined {
  const trimmed = candidate.trim()
  if (!trimmed) {
    return undefined
  }

  // Already absolute with protocol
  if (/^[a-zA-Z][a-zA-Z0-9+-.]*:\/\//u.test(trimmed)) {
    return normalizeBase(trimmed)
  }

  // Protocol-relative (//example.com)
  if (trimmed.startsWith('//')) {
    const protocol = protocolHint ?? resolveProtocolHint(origin)
    return normalizeBase(`${protocol}${trimmed}`)
  }

  // Has protocol but no slashes
  if (/^[a-zA-Z][a-zA-Z0-9+-.]*:/u.test(trimmed)) {
    return normalizeBase(trimmed)
  }

  // Relative path - needs origin
  const normalizedOrigin = normalizeOrigin(origin)
  if (!normalizedOrigin) {
    return undefined
  }

  const leadingSlash = trimmed.startsWith('/') ? '' : '/'
  return normalizeBase(`${normalizedOrigin}${leadingSlash}${trimmed}`)
}

/**
 * Get window-like object from options or global scope
 *
 * @param options - Options that may contain windowObject
 * @returns WindowLike object or undefined
 */
export function getWindowLike(options?: { windowObject?: WindowLike }): WindowLike | undefined {
  if (options?.windowObject) {
    return options.windowObject
  }

  // Try to access global window
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

/**
 * Ensure string has trailing slash
 *
 * @param value - String value
 * @returns String with trailing slash
 */
export function ensureTrailingSlash(value = '/'): string {
  if (typeof value !== 'string' || value.length === 0) {
    return '/'
  }
  return value.endsWith('/') ? value : `${value}/`
}

/**
 * Remove trailing slashes from string
 *
 * @param value - String value
 * @returns String without trailing slashes
 */
export function trimTrailingSlash(value = '/'): string {
  if (typeof value !== 'string') {
    return ''
  }
  return value.replace(/\/+$/, '')
}

/**
 * Convert HTTP(S) URL to WS(S) URL
 *
 * @param url - HTTP(S) URL
 * @returns WS(S) URL
 */
export function convertHttpToWs(url: string): string {
  try {
    const parsed = new URL(url)
    parsed.protocol = parsed.protocol === 'https:' ? 'wss:' : 'ws:'
    let result = parsed.toString()
    // Remove trailing slash if it's just the root path
    if (parsed.pathname === '/' && result.endsWith('/')) {
      result = result.slice(0, -1)
    }
    return result
  } catch {
    // Fallback: simple string replacement
    return url.replace(/^https?:/, (match) => (match === 'https:' ? 'wss:' : 'ws:'))
  }
}

/**
 * Parse port from various input formats
 *
 * @param value - Port value (string, number, or URL)
 * @param fallback - Fallback port if parsing fails
 * @returns Parsed port or fallback
 */
export function parsePort(value: unknown, fallback?: string): string | undefined {
  // If already a valid port number
  if (typeof value === 'number' && Number.isInteger(value) && value > 0 && value <= 65535) {
    return String(value)
  }

  // If string, try to parse
  if (typeof value === 'string') {
    const trimmed = value.trim()

    // Try as direct port number
    const parsed = Number.parseInt(trimmed, 10)
    if (!Number.isNaN(parsed) && parsed > 0 && parsed <= 65535) {
      return String(parsed)
    }

    // Try to extract from URL
    try {
      const url = new URL(trimmed)
      if (url.port) {
        return url.port
      }
    } catch {
      // Not a valid URL, continue
    }
  }

  return fallback
}

/**
 * Check if value is a valid URL
 *
 * @param value - Value to check
 * @returns True if valid URL
 */
export function isValidUrl(value: string): boolean {
  try {
    new URL(value)
    return true
  } catch {
    return false
  }
}

/**
 * Escape string for safe inclusion in inline script
 *
 * Prevents XSS by escaping closing script tags and other dangerous sequences.
 *
 * @param value - String to escape
 * @returns Escaped string
 */
export function escapeForInlineScript(value: string): string {
  return value
    .replace(/<\//g, '\\u003C/')
    .replace(/<!--/g, '\\u003C!--')
    .replace(/-->/g, '--\\u003E')
}

/**
 * Check if path has an asset file extension
 *
 * Checks if the path ends with a known asset extension like .js, .css, .png, etc.
 * Handles query strings and hash fragments correctly.
 *
 * @param path - URL path to check
 * @param extensions - Set of extensions to check (default: ASSET_EXTENSIONS)
 * @returns True if path has an asset extension
 *
 * @example
 * ```typescript
 * hasAssetExtension('/assets/main.js') // true
 * hasAssetExtension('/assets/main.js?v=123') // true
 * hasAssetExtension('/api/users') // false
 * hasAssetExtension('/apps/preview') // false
 * ```
 */
export function hasAssetExtension(path: string, extensions?: Set<string>): boolean {
  if (typeof path !== 'string') {
    return false
  }

  // Use provided extensions or import dynamically to avoid circular dependency
  const extensionsToCheck = extensions || new Set([
    '.js', '.mjs', '.ts', '.tsx', '.jsx', '.css', '.map', '.json',
    '.svg', '.png', '.jpg', '.jpeg', '.gif', '.ico', '.webp',
    '.woff', '.woff2', '.ttf', '.eot', '.wasm',
    '.mp3', '.mp4', '.webm', '.ogg', '.wav',
    '.pdf', '.zip', '.tar', '.gz',
  ])

  // Remove query string and hash
  let cleanPath = path
  const questionIndex = path.indexOf('?')
  if (questionIndex !== -1) {
    cleanPath = path.slice(0, questionIndex)
  }
  const hashIndex = cleanPath.indexOf('#')
  if (hashIndex !== -1) {
    cleanPath = cleanPath.slice(0, hashIndex)
  }

  // Extract filename from path
  const lastSlash = cleanPath.lastIndexOf('/')
  const filename = lastSlash === -1 ? cleanPath : cleanPath.slice(lastSlash + 1)

  // Check if filename has extension
  const dotIndex = filename.lastIndexOf('.')
  if (dotIndex === -1 || dotIndex === filename.length - 1) {
    return false
  }

  const ext = filename.slice(dotIndex).toLowerCase()
  return extensionsToCheck.has(ext)
}

/**
 * Check if path starts with a known asset prefix
 *
 * Detects paths that start with common asset prefixes like /@vite, /assets/, etc.
 *
 * @param path - URL path to check
 * @param prefixes - Array of prefixes to check (default: ASSET_PATH_PREFIXES)
 * @returns True if path starts with an asset prefix
 *
 * @example
 * ```typescript
 * startsWithAssetPrefix('/@vite/client') // true
 * startsWithAssetPrefix('/assets/logo.png') // true
 * startsWithAssetPrefix('/api/users') // false
 * ```
 */
export function startsWithAssetPrefix(path: string, prefixes?: string[]): boolean {
  if (typeof path !== 'string') {
    return false
  }

  const prefixesToCheck = prefixes || [
    '/@vite', '/@react-refresh', '/@fs/', '/src/', '/node_modules/.vite/',
    '/assets/', '/static/', '/public/', '/_next/', '/__webpack_hmr',
  ]

  return prefixesToCheck.some(prefix => path.startsWith(prefix))
}

/**
 * Check if path looks like an asset request
 *
 * Combines extension and prefix checks to determine if a path is likely
 * requesting an asset file (not an HTML page).
 *
 * @param path - URL path to check
 * @returns True if path looks like an asset request
 *
 * @example
 * ```typescript
 * isAssetRequest('/assets/main.js') // true
 * isAssetRequest('/logo.png') // true
 * isAssetRequest('/@vite/client') // true
 * isAssetRequest('/apps/preview') // false
 * isAssetRequest('/api/users') // false
 * ```
 */
export function isAssetRequest(path: string): boolean {
  return hasAssetExtension(path) || startsWithAssetPrefix(path)
}
