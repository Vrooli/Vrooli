/**
 * Type definitions for @vrooli/api-base
 *
 * This module contains all shared TypeScript interfaces and types used across
 * client and server implementations.
 */

/**
 * Browser-like window environment
 *
 * Allows testing and usage in different JavaScript environments by providing
 * a minimal interface compatible with browser window objects.
 */
export interface WindowLike {
  /** Browser location object */
  location?: {
    hostname?: string
    origin?: string
    pathname?: string
    protocol?: string
    port?: string
    host?: string
  }
  /** Allow any additional properties for proxy globals and other extensions */
  [key: string]: unknown
}

/**
 * Port entry in proxy metadata
 *
 * Represents a single port/endpoint that can be proxied, with metadata
 * about how to route requests to it.
 */
export interface PortEntry {
  /** Application/scenario identifier */
  appId?: string
  /** Port number */
  port: number
  /** Human-readable label (e.g., "UI", "API", "WebSocket") */
  label: string | null
  /** Normalized lowercase label */
  normalizedLabel: string | null
  /** URL-safe slug derived from label */
  slug: string
  /** Source where this port was discovered (e.g., "port_mappings", "environment") */
  source: string
  /** Priority for selecting default/primary port (higher = more priority) */
  priority?: number
  /** Port kind/type if specified */
  kind?: string | null
  /** Whether this is the primary/default port */
  isPrimary: boolean
  /** Proxy path for this port (e.g., "/apps/scenario/proxy") */
  path: string | null
  /** Additional aliases for this port */
  aliases: string[]
  /** Asset namespace path for this port */
  assetNamespace?: string
}

/**
 * Proxy metadata injected by host scenarios
 *
 * This object is injected into the global scope (e.g., window.__VROOLI_PROXY_INFO__)
 * by parent/host scenarios when embedding child scenarios in iframes. It provides
 * routing information so the child can construct correct API URLs.
 */
export interface ProxyInfo {
  /** Scenario hosting the proxy (e.g., "app-monitor", "my-dashboard") */
  hostScenario?: string
  /** Target scenario being proxied (e.g., "scenario-auditor") */
  targetScenario?: string
  /** Application/scenario ID */
  appId?: string
  /** Timestamp when metadata was generated */
  generatedAt: number
  /** Hostnames that should be proxied (typically loopback addresses) */
  hosts: string[]
  /** Primary/default port entry */
  primary: PortEntry
  /** All available ports */
  ports: PortEntry[]
  /** Base path for proxy (e.g., "/apps/scenario-name/proxy") */
  basePath?: string
}

/**
 * Indexed proxy information for fast lookups
 *
 * Built from ProxyInfo, optimized for runtime resolution by port/alias.
 */
export interface ProxyIndex {
  /** Application/scenario ID */
  appId?: string
  /** Timestamp when index was generated */
  generatedAt: number
  /** Map of port aliases to port entries */
  aliasMap: Map<string, PortEntry>
  /** Primary/default port entry */
  primary: PortEntry
  /** Set of hostnames that should be proxied */
  hosts: Set<string>
}

/**
 * Runtime scenario configuration
 *
 * Provided by the scenario's server at runtime (e.g., via /config endpoint).
 * Contains actual URLs and ports for the running scenario.
 */
export interface ScenarioConfig {
  /** Full API URL (e.g., "http://localhost:8080/api/v1") */
  apiUrl: string
  /** Full WebSocket URL (e.g., "ws://localhost:8080/ws") */
  wsUrl: string
  /** API port as string */
  apiPort: string
  /** WebSocket port as string */
  wsPort?: string
  /** UI port as string */
  uiPort: string
  /** Optional: Version info */
  version?: string
  /** Optional: Service name */
  service?: string
  /** Allow additional custom config fields */
  [key: string]: unknown
}

/**
 * Options for resolving API base URL
 */
export interface ResolveOptions {
  /**
   * Explicit URL to use (bypasses all detection)
   * @example "https://example.com/api/v1"
   */
  explicitUrl?: string | null

  /**
   * Default port to use for localhost fallback
   * @default "15000"
   */
  defaultPort?: string

  /**
   * API path suffix to append
   * @default "/api/v1"
   */
  apiSuffix?: string

  /**
   * Whether to append the API suffix
   * @default false
   */
  appendSuffix?: boolean

  /**
   * Custom window object (for testing or non-browser environments)
   */
  windowObject?: WindowLike

  /**
   * Custom global variable names to check for proxy info
   * @default ["__VROOLI_PROXY_INFO__", "__VROOLI_PROXY_INDEX__", "__APP_MONITOR_PROXY_INFO__", "__APP_MONITOR_PROXY_INDEX__"]
   */
  proxyGlobalNames?: string[]

  /**
   * Endpoint to fetch runtime config from
   * @default "./config"
   */
  configEndpoint?: string

  /**
   * Global variable name for injected config
   * @default "__VROOLI_CONFIG__"
   */
  configGlobalName?: string
}

/**
 * Options for building API URLs
 */
export interface BuildUrlOptions extends ResolveOptions {
  /**
   * Pre-resolved base URL (skips resolution if provided)
   */
  baseUrl?: string
}

/**
 * Options for proxy metadata injection
 */
export interface ProxyMetadataOptions {
  /** Application/scenario ID */
  appId: string
  /** Host scenario name */
  hostScenario?: string
  /** Target scenario name */
  targetScenario?: string
  /** Port entries to include */
  ports: PortEntry[]
  /** Primary port entry */
  primaryPort: PortEntry
  /** Loopback hostnames to proxy */
  loopbackHosts?: string[]
}

/**
 * Options for creating proxy middleware
 */
export interface ProxyOptions {
  /** Target API port */
  apiPort: number | string
  /** Target API host */
  apiHost?: string
  /** Request timeout in milliseconds */
  timeout?: number
  /** Additional headers to add/override (can be static or a function that receives the request) */
  headers?: Record<string, string> | ((req: any) => Record<string, string>)
  /** Whether to log proxy requests */
  verbose?: boolean
}

/**
 * Options for config endpoint
 */
export interface ConfigEndpointOptions {
  /** API port */
  apiPort: number | string
  /** API host */
  apiHost?: string
  /** WebSocket port (defaults to API port) */
  wsPort?: number | string
  /** WebSocket host (defaults to API host) */
  wsHost?: string
  /** UI port */
  uiPort: number | string
  /** Service version */
  version?: string
  /** Service name */
  serviceName?: string
  /** Additional config fields */
  additionalConfig?: Record<string, unknown>
  /** Custom config builder that overrides default behavior */
  configBuilder?: () => ScenarioConfig
  /** Enable CORS headers */
  cors?: boolean
  /** Add timestamp to response */
  includeTimestamp?: boolean
  /** Add cache control headers */
  cacheControl?: boolean | string
}

/**
 * Options for health endpoint
 */
export interface HealthOptions {
  /** Service name */
  serviceName: string
  /** Service version */
  version?: string
  /** API port for connectivity check */
  apiPort?: number | string
  /** API host for connectivity check */
  apiHost?: string
  /** Health check timeout in milliseconds */
  timeout?: number
  /** Additional health check function */
  customHealthCheck?: () => Promise<Record<string, unknown>>
}

/**
 * Options for creating scenario server
 */
export interface ServerTemplateOptions {
  /** UI server port */
  uiPort: number | string
  /** API server port */
  apiPort: number | string
  /** API server host */
  apiHost?: string
  /** WebSocket port (defaults to API port) */
  wsPort?: number | string
  /** WebSocket host (defaults to API host) */
  wsHost?: string
  /** Directory containing built UI files */
  distDir?: string
  /** Service name */
  serviceName?: string
  /** Service version */
  version?: string
  /** CORS allowed origins */
  corsOrigins?: string | string[]
  /** Enable/disable logging */
  verbose?: boolean
  /** Custom config builder */
  configBuilder?: (env: Record<string, string | undefined>) => ScenarioConfig
  /** Custom route setup */
  setupRoutes?: (app: any) => void
  /** Inject proxy metadata into HTML */
  proxyMetadata?: ProxyInfo
  /** Inject scenario config into HTML */
  scenarioConfig?: ScenarioConfig
  /** WebSocket URL prefix to proxy (e.g., '/ws'). If set, automatically handles WebSocket upgrades */
  wsPathPrefix?: string
  /** URL transformation for WebSocket paths (e.g., '/ws' -> '/api/v1'). Defaults to replacing prefix with '/api/v1' */
  wsPathTransform?: (path: string) => string
  /** Additional headers to inject into proxied requests */
  proxyHeaders?: Record<string, string> | ((req: any) => Record<string, string>)
}

/**
 * Result of a health check
 */
export interface HealthCheckResult {
  /** Overall health status */
  status: 'healthy' | 'degraded' | 'unhealthy'
  /** Service identifier */
  service: string
  /** Timestamp of check */
  timestamp: string
  /** Service version */
  version?: string
  /** Whether service is ready to accept traffic */
  readiness: boolean
  /** API connectivity status */
  api_connectivity?: {
    connected: boolean
    api_url: string | null
    last_check: string
    error: {
      code: string
      message: string
      category: string
      retryable: boolean
    } | null
    latency_ms: number | null
    upstream?: unknown
  }
  /** Additional health details */
  [key: string]: unknown
}
