/**
 * Client-side API resolution
 *
 * Re-exports all client functionality for convenient importing.
 */

// Resolution
export { resolveApiBase, resolveWsBase } from './resolve.js'

// URL building
export { buildApiUrl, buildWsUrl } from './url.js'

// Proxy detection
export { isProxyContext, getProxyInfo, getProxyIndex } from './detect.js'

// Runtime configuration
export {
  fetchRuntimeConfig,
  getInjectedConfig,
  resolveWithConfig,
  createConfigCache,
} from './config.js'
