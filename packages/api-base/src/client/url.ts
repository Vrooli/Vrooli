/**
 * URL building utilities
 *
 * Functions for constructing full URLs from base URLs and paths.
 */

import type { BuildUrlOptions } from '../shared/types.js'
import { normalizeBase } from '../shared/utils.js'
import { resolveApiBase, resolveWsBase } from './resolve.js'

/**
 * Build full API URL from path
 *
 * Combines a base URL with a path to create a complete API endpoint URL.
 * If no base URL is provided, resolves it automatically.
 *
 * @param path - API path (e.g., "/health", "/users")
 * @param options - URL building options
 * @returns Complete API URL
 *
 * @example
 * ```typescript
 * // With explicit base
 * const url = buildApiUrl('/health', {
 *   baseUrl: 'https://api.example.com/v1'
 * })
 * // → https://api.example.com/v1/health
 *
 * // Auto-resolve base
 * const url = buildApiUrl('/health', {
 *   defaultPort: '8080'
 * })
 * // → http://127.0.0.1:8080/health
 *
 * // With suffix
 * const url = buildApiUrl('/health', {
 *   defaultPort: '8080',
 *   appendSuffix: true
 * })
 * // → http://127.0.0.1:8080/api/v1/health
 * ```
 */
export function buildApiUrl(path: string, options: BuildUrlOptions = {}): string {
  const baseUrl = options.baseUrl ?? resolveApiBase({ ...options, appendSuffix: options.appendSuffix ?? false })
  const normalizedBase = normalizeBase(baseUrl)

  // Handle empty path
  if (!path || path.length === 0) {
    return normalizedBase
  }

  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${normalizedBase}${normalizedPath}`
}

/**
 * Build full WebSocket URL from path
 *
 * Combines a base WebSocket URL with a path to create a complete WebSocket endpoint URL.
 * If no base URL is provided, resolves it automatically using WebSocket protocol.
 *
 * @param path - WebSocket path (e.g., "/ws", "/socket")
 * @param options - URL building options
 * @returns Complete WebSocket URL
 *
 * @example
 * ```typescript
 * // With explicit base
 * const url = buildWsUrl('/ws', {
 *   baseUrl: 'wss://api.example.com'
 * })
 * // → wss://api.example.com/ws
 *
 * // Auto-resolve base
 * const url = buildWsUrl('/ws', {
 *   defaultPort: '8080'
 * })
 * // → ws://127.0.0.1:8080/ws
 * ```
 */
export function buildWsUrl(path: string, options: BuildUrlOptions = {}): string {
  const baseUrl = options.baseUrl ?? resolveWsBase({ ...options, appendSuffix: options.appendSuffix ?? false })
  const normalizedBase = normalizeBase(baseUrl)

  // Handle empty path
  if (!path || path.length === 0) {
    return normalizedBase
  }

  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  return `${normalizedBase}${normalizedPath}`
}
