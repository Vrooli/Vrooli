/**
 * @vrooli/api-base
 *
 * Universal API connectivity for Vrooli scenarios.
 * Handles API resolution, proxy detection, and runtime configuration
 * across all deployment contexts.
 *
 * @packageDocumentation
 */

// ============================================================================
// Client-side exports (default)
// ============================================================================

export {
  // Resolution
  resolveApiBase,
  resolveWsBase,
  // URL building
  buildApiUrl,
  buildWsUrl,
  // Proxy detection
  isProxyContext,
  getProxyInfo,
  getProxyIndex,
  // Runtime configuration
  fetchRuntimeConfig,
  getInjectedConfig,
  resolveWithConfig,
  createConfigCache,
} from './client/index.js'

// ============================================================================
// Type exports
// ============================================================================

export type {
  WindowLike,
  PortEntry,
  ProxyInfo,
  ProxyIndex,
  ScenarioConfig,
  ResolveOptions,
  BuildUrlOptions,
  ProxyMetadataOptions,
  ProxyOptions,
  ConfigEndpointOptions,
  HealthOptions,
  ServerTemplateOptions,
  HealthCheckResult,
} from './shared/types.js'

// ============================================================================
// Utility exports
// ============================================================================

/**
 * Utility functions for advanced use cases.
 * Most users won't need these - they're used internally.
 */
export const apiBaseUtils = {
  // From shared/utils.ts
  toTrimmedString: async () => (await import('./shared/utils.js')).toTrimmedString,
  normalizeBase: async () => (await import('./shared/utils.js')).normalizeBase,
  normalizeOrigin: async () => (await import('./shared/utils.js')).normalizeOrigin,
  appendSuffix: async () => (await import('./shared/utils.js')).appendSuffix,
  isLocalHostname: async () => (await import('./shared/utils.js')).isLocalHostname,
  convertHttpToWs: async () => (await import('./shared/utils.js')).convertHttpToWs,
  parsePort: async () => (await import('./shared/utils.js')).parsePort,
  isValidUrl: async () => (await import('./shared/utils.js')).isValidUrl,
}

// ============================================================================
// Constants export
// ============================================================================

export {
  DEFAULT_PROXY_GLOBALS,
  DEFAULT_CONFIG_GLOBAL,
  LOOPBACK_HOST,
  LOOPBACK_HOSTS,
  DEFAULT_API_PORT,
  DEFAULT_API_SUFFIX,
  DEFAULT_WS_SUFFIX,
  DEFAULT_CONFIG_ENDPOINT,
  PROXY_PATH_MARKER,
  APPS_PATH_MARKER,
  LOCAL_HOST_PATTERN,
  DEFAULT_HEALTH_CHECK_TIMEOUT,
  DEFAULT_PROXY_TIMEOUT,
  HOP_BY_HOP_HEADERS,
} from './shared/constants.js'
