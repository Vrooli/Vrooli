/**
 * Constants for @vrooli/api-base
 *
 * Centralized configuration values and patterns used throughout the package.
 * These can be overridden via options in most functions.
 */

/**
 * Default global variable names for proxy metadata
 *
 * Checked in order. Supports both new standard names and legacy names
 * for backwards compatibility.
 */
export const DEFAULT_PROXY_GLOBALS = [
  '__VROOLI_PROXY_INFO__',      // New standard
  '__VROOLI_PROXY_INDEX__',      // New standard (indexed)
  '__APP_MONITOR_PROXY_INFO__',  // Legacy - backwards compatibility
  '__APP_MONITOR_PROXY_INDEX__', // Legacy - backwards compatibility
] as const

/**
 * Default global variable name for scenario configuration
 */
export const DEFAULT_CONFIG_GLOBAL = '__VROOLI_CONFIG__' as const

/**
 * Loopback host used for localhost connections
 *
 * Uses 127.0.0.1 instead of "localhost" for consistency across platforms
 * and to avoid potential DNS resolution issues.
 */
export const LOOPBACK_HOST = '127.0.0.1' as const

/**
 * Set of all loopback/localhost hostnames
 *
 * Used to determine if a hostname refers to the local machine.
 */
export const LOOPBACK_HOSTS = new Set([
  LOOPBACK_HOST,
  'localhost',
  '0.0.0.0',
  '::1',
  '[::1]',
])

/**
 * Default API port when no port can be determined
 */
export const DEFAULT_API_PORT = '15000' as const

/**
 * Default API path suffix
 *
 * Most Vrooli scenarios use /api/v1 as their API base path.
 */
export const DEFAULT_API_SUFFIX = '/api/v1' as const

/**
 * Default WebSocket path suffix
 */
export const DEFAULT_WS_SUFFIX = '/ws' as const

/**
 * Default config endpoint path
 *
 * Relative path - resolves relative to current location.
 */
export const DEFAULT_CONFIG_ENDPOINT = './config' as const

/**
 * Generic proxy path marker
 *
 * Any URL path containing this marker is considered a proxy context.
 */
export const PROXY_PATH_MARKER = '/proxy' as const

/**
 * App shell proxy pattern (backwards compatibility)
 *
 * App-monitor and similar scenarios use this pattern:
 * https://host.com/apps/scenario-name/proxy/
 */
export const APPS_PATH_MARKER = '/apps/' as const

/**
 * Regex pattern for matching localhost/loopback addresses
 */
export const LOCAL_HOST_PATTERN = /^(localhost|127\.0\.0\.1|0\.0\.0\.0|\[?::1\]?)/i

/**
 * Default timeout for API health checks (milliseconds)
 */
export const DEFAULT_HEALTH_CHECK_TIMEOUT = 5000 as const

/**
 * Default timeout for proxy requests (milliseconds)
 */
export const DEFAULT_PROXY_TIMEOUT = 15000 as const

/**
 * HTTP headers that should not be forwarded in regular HTTP proxy requests
 *
 * These are hop-by-hop headers that are connection-specific.
 * Note: WebSocket headers MUST be forwarded during upgrade requests.
 */
export const HOP_BY_HOP_HEADERS = new Set([
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'te',
  'trailers',
  'transfer-encoding',
  'upgrade',
])

/**
 * WebSocket-specific headers that must be forwarded during upgrade
 *
 * These headers are required for WebSocket handshake and should NOT
 * be filtered out during WebSocket upgrade requests.
 */
export const WEBSOCKET_HEADERS = new Set([
  'sec-websocket-key',
  'sec-websocket-accept',
  'sec-websocket-version',
  'sec-websocket-protocol',
  'sec-websocket-extensions',
])

/**
 * Default CORS configuration
 */
export const DEFAULT_CORS_ORIGINS = [
  'http://localhost:*',
  'http://127.0.0.1:*',
  'null', // For file:// origins
] as const

/**
 * Script tag type for injected metadata
 */
export const INJECTION_SCRIPT_TYPE = 'application/json' as const

/**
 * ID for injected proxy metadata script tag
 */
export const PROXY_METADATA_SCRIPT_ID = 'vrooli-proxy-metadata' as const

/**
 * ID for injected scenario config script tag
 */
export const SCENARIO_CONFIG_SCRIPT_ID = 'vrooli-scenario-config' as const

/**
 * Common asset file extensions
 *
 * Used to detect asset requests and prevent SPA fallback from serving
 * index.html for these requests.
 */
export const ASSET_EXTENSIONS = new Set([
  '.js',
  '.mjs',
  '.ts',
  '.tsx',
  '.jsx',
  '.css',
  '.map',
  '.json',
  '.svg',
  '.png',
  '.jpg',
  '.jpeg',
  '.gif',
  '.ico',
  '.webp',
  '.woff',
  '.woff2',
  '.ttf',
  '.eot',
  '.wasm',
  '.mp3',
  '.mp4',
  '.webm',
  '.ogg',
  '.wav',
  '.pdf',
  '.zip',
  '.tar',
  '.gz',
])

/**
 * Common asset path prefixes
 *
 * Paths starting with these prefixes are typically assets from dev servers
 * or build tools.
 */
export const ASSET_PATH_PREFIXES = [
  '/@vite',
  '/@react-refresh',
  '/@fs/',
  '/src/',
  '/node_modules/.vite/',
  '/assets/',
  '/static/',
  '/public/',
  '/_next/',
  '/__webpack_hmr',
]
