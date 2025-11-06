# Type Definitions Reference

Complete TypeScript type definitions for `@vrooli/api-base`.

**Import:**
```typescript
import type {
  // Core Types
  ProxyInfo,
  ScenarioConfig,
  PortEntry,
  WindowLike,

  // Options
  ResolveOptions,
  BuildUrlOptions,
  ServerTemplateOptions,
  ProxyOptions,
  ConfigEndpointOptions,
  HealthOptions,

  // Results
  HealthCheckResult,
  ProxyIndex,
} from '@vrooli/api-base/types'
```

## Table of Contents

- [Core Types](#core-types)
  - [ProxyInfo](#proxyinfo)
  - [ScenarioConfig](#scenarioconfig)
  - [PortEntry](#portentry)
  - [WindowLike](#windowlike)
  - [ProxyIndex](#proxyindex)
- [Client Options](#client-options)
  - [ResolveOptions](#resolveoptions)
  - [BuildUrlOptions](#buildurloptions)
- [Server Options](#server-options)
  - [ServerTemplateOptions](#servertemplateoptions)
  - [ProxyOptions](#proxyoptions)
  - [ConfigEndpointOptions](#configendpointoptions)
  - [HealthOptions](#healthoptions)
  - [ProxyMetadataOptions](#proxymetadataoptions)
- [Result Types](#result-types)
  - [HealthCheckResult](#healthcheckresult)

---

## Core Types

### ProxyInfo

Proxy metadata injected by host scenarios when embedding child scenarios.

```typescript
interface ProxyInfo {
  hostScenario?: string
  targetScenario?: string
  appId?: string
  generatedAt: number
  hosts: string[]
  primary: PortEntry
  ports: PortEntry[]
  basePath?: string
}
```

**Fields:**

| Name | Type | Description |
|------|------|-------------|
| `hostScenario` | `string` | Name of hosting scenario (e.g., "app-monitor") |
| `targetScenario` | `string` | Name of embedded scenario (e.g., "scenario-auditor") |
| `appId` | `string` | Application/scenario identifier |
| `generatedAt` | `number` | Unix timestamp when metadata was generated |
| `hosts` | `string[]` | Hostnames to proxy (typically `["localhost", "127.0.0.1"]`) |
| `primary` | [`PortEntry`](#portentry) | Primary/default port entry |
| `ports` | [`PortEntry[]`](#portentry) | All available ports |
| `basePath` | `string` | Proxy base path (e.g., "/apps/scenario/proxy") |

**Global Injection:**

This object is injected into the global scope by host scenarios:

```typescript
declare global {
  interface Window {
    __VROOLI_PROXY_INFO__?: ProxyInfo
    __VROOLI_PROXY_INDEX__?: ProxyIndex
    // Backwards compatibility
    __APP_MONITOR_PROXY_INFO__?: ProxyInfo
    __APP_MONITOR_PROXY_INDEX__?: ProxyIndex
  }
}
```

**Example:**

```typescript
const metadata: ProxyInfo = {
  hostScenario: 'app-monitor',
  targetScenario: 'scenario-auditor',
  appId: 'scenario-auditor',
  generatedAt: Date.now(),
  hosts: ['localhost', '127.0.0.1'],
  primary: {
    port: 36224,
    label: 'UI',
    slug: 'ui',
    source: 'port_mappings',
    isPrimary: true,
    path: '/apps/scenario-auditor/proxy',
    aliases: ['ui', 'primary', '36224'],
    normalizedLabel: 'ui',
  },
  ports: [/* ... */],
  basePath: '/apps/scenario-auditor/proxy',
}
```

**See Also:**
- [Proxy Resolution Concept](../concepts/proxy-resolution.md)
- [Client: getProxyInfo](./client.md#getproxyinfo)
- [Server: injectProxyMetadata](./server.md#injectproxymetadata)

---

### ScenarioConfig

Runtime configuration provided by the scenario's server.

```typescript
interface ScenarioConfig {
  apiUrl: string
  wsUrl: string
  apiPort: string
  wsPort?: string
  uiPort: string
  version?: string
  service?: string
  [key: string]: unknown
}
```

**Fields:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `apiUrl` | `string` | ✅ | Full API URL (e.g., "http://localhost:8080/api/v1") |
| `wsUrl` | `string` | ✅ | Full WebSocket URL (e.g., "ws://localhost:8081/ws") |
| `apiPort` | `string` | ✅ | API port as string |
| `wsPort` | `string` | ❌ | WebSocket port as string |
| `uiPort` | `string` | ✅ | UI port as string |
| `version` | `string` | ❌ | Service version |
| `service` | `string` | ❌ | Service name |
| `[key]` | `unknown` | ❌ | Additional custom fields |

**Global Injection:**

```typescript
declare global {
  interface Window {
    __VROOLI_CONFIG__?: ScenarioConfig
  }
}
```

**Example:**

```typescript
const config: ScenarioConfig = {
  apiUrl: 'http://localhost:8080/api/v1',
  wsUrl: 'ws://localhost:8081/ws',
  apiPort: '8080',
  wsPort: '8081',
  uiPort: '3000',
  version: '1.0.0',
  service: 'my-scenario',
  // Custom fields
  features: {
    analytics: true,
    beta: false,
  },
  environment: 'production',
}
```

**See Also:**
- [Runtime Configuration Concept](../concepts/runtime-config.md)
- [Client: getScenarioConfig](./client.md#getscenarioconfig)
- [Server: injectScenarioConfig](./server.md#injectscenarioconfig)

---

### PortEntry

Represents a single port/endpoint that can be proxied.

```typescript
interface PortEntry {
  appId?: string
  port: number
  label: string | null
  normalizedLabel: string | null
  slug: string
  source: string
  priority?: number
  kind?: string | null
  isPrimary: boolean
  path: string | null
  aliases: string[]
  assetNamespace?: string
}
```

**Fields:**

| Name | Type | Description |
|------|------|-------------|
| `appId` | `string` | Application/scenario identifier |
| `port` | `number` | Port number (1-65535) |
| `label` | `string \| null` | Human-readable label (e.g., "UI", "API") |
| `normalizedLabel` | `string \| null` | Lowercase normalized label |
| `slug` | `string` | URL-safe slug for routing |
| `source` | `string` | Where port was discovered (e.g., "port_mappings") |
| `priority` | `number` | Selection priority (higher = more priority) |
| `kind` | `string \| null` | Port type/kind if specified |
| `isPrimary` | `boolean` | Whether this is the primary/default port |
| `path` | `string \| null` | Proxy path for this port |
| `aliases` | `string[]` | Additional aliases (label, slug, port number) |
| `assetNamespace` | `string` | Asset namespace path |

**Example:**

```typescript
const port: PortEntry = {
  appId: 'my-scenario',
  port: 3000,
  label: 'UI',
  normalizedLabel: 'ui',
  slug: 'ui',
  source: 'port_mappings',
  priority: 80,
  kind: 'http',
  isPrimary: true,
  path: '/apps/my-scenario/proxy',
  aliases: ['ui', 'primary', 'default', '3000'],
  assetNamespace: '/apps/my-scenario/proxy',
}
```

---

### WindowLike

Minimal browser window interface for testing and non-browser environments.

```typescript
interface WindowLike {
  location?: {
    hostname?: string
    origin?: string
    pathname?: string
    protocol?: string
    port?: string
    host?: string
  }
  [key: string]: unknown
}
```

**Purpose:**
Allows `@vrooli/api-base` to work in:
- Browser environments (production)
- Test environments (jsdom, happy-dom)
- Node.js environments (with mock window)
- React Native (with polyfills)

**Example:**

```typescript
// Mock window for testing
const mockWindow: WindowLike = {
  location: {
    hostname: 'example.com',
    origin: 'https://example.com',
    pathname: '/apps/my-scenario/proxy/',
    protocol: 'https:',
    port: '443',
    host: 'example.com',
  },
  __VROOLI_PROXY_INFO__: { /* ... */ },
}

// Use in resolution
import { resolveApiBase } from '@vrooli/api-base'
const apiBase = resolveApiBase({ windowObject: mockWindow })
```

---

### ProxyIndex

Optimized index structure built from `ProxyInfo` for fast runtime lookups.

```typescript
interface ProxyIndex {
  appId?: string
  generatedAt: number
  aliasMap: Map<string, PortEntry>
  primary: PortEntry
  hosts: Set<string>
}
```

**Fields:**

| Name | Type | Description |
|------|------|-------------|
| `appId` | `string` | Application/scenario identifier |
| `generatedAt` | `number` | Unix timestamp when index was built |
| `aliasMap` | `Map<string, PortEntry>` | Fast lookup by port alias |
| `primary` | [`PortEntry`](#portentry) | Primary/default port |
| `hosts` | `Set<string>` | Set of hostnames to proxy |

**Usage:**

The index is built automatically and injected alongside `ProxyInfo`:

```typescript
// Client-side access
const index = window.__VROOLI_PROXY_INDEX__

// Fast alias lookup
const uiPort = index.aliasMap.get('ui')
const apiPort = index.aliasMap.get('api')
const port3000 = index.aliasMap.get('3000')
```

---

## Client Options

### ResolveOptions

Options for resolving API/WebSocket base URLs.

```typescript
interface ResolveOptions {
  explicitUrl?: string | null
  defaultPort?: string
  apiSuffix?: string
  appendSuffix?: boolean
  windowObject?: WindowLike
  proxyGlobalNames?: string[]
  configEndpoint?: string
  configGlobalName?: string
}
```

**Fields:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `explicitUrl` | `string` | - | Explicit URL (bypasses all detection) |
| `defaultPort` | `string` | `"15000"` | Default port for localhost fallback |
| `apiSuffix` | `string` | `"/api/v1"` | Path suffix to append |
| `appendSuffix` | `boolean` | `false` | Whether to append suffix |
| `windowObject` | [`WindowLike`](#windowlike) | `window` | Custom window object |
| `proxyGlobalNames` | `string[]` | See below | Custom proxy global names |
| `configEndpoint` | `string` | `"./config"` | Runtime config endpoint |
| `configGlobalName` | `string` | `"__VROOLI_CONFIG__"` | Config global variable name |

**Default Proxy Global Names:**
```typescript
[
  '__VROOLI_PROXY_INFO__',
  '__VROOLI_PROXY_INDEX__',
  '__APP_MONITOR_PROXY_INFO__',  // Backwards compatibility
  '__APP_MONITOR_PROXY_INDEX__',  // Backwards compatibility
]
```

**Examples:**

```typescript
// Minimal
resolveApiBase({ defaultPort: '8080' })

// With suffix
resolveApiBase({
  defaultPort: '8080',
  appendSuffix: true,
  apiSuffix: '/api/v2',
})

// Custom proxy globals
resolveApiBase({
  proxyGlobalNames: ['__CUSTOM_PROXY__'],
})

// Explicit URL (skips all detection)
resolveApiBase({
  explicitUrl: 'https://api.example.com/v1',
})
```

---

### BuildUrlOptions

Options for building complete API URLs (extends `ResolveOptions`).

```typescript
interface BuildUrlOptions extends ResolveOptions {
  baseUrl?: string
}
```

**Additional Fields:**

| Name | Type | Description |
|------|------|-------------|
| `baseUrl` | `string` | Pre-resolved base URL (skips resolution) |

**Example:**

```typescript
import { buildApiUrl } from '@vrooli/api-base'

// With auto-resolution
buildApiUrl('/health', {
  defaultPort: '8080',
  appendSuffix: true,
})
// → "http://127.0.0.1:8080/api/v1/health"

// With explicit base
buildApiUrl('/health', {
  baseUrl: 'https://example.com/api/v1',
})
// → "https://example.com/api/v1/health"
```

---

## Server Options

### ServerTemplateOptions

Options for creating a complete scenario server.

```typescript
interface ServerTemplateOptions {
  uiPort: number | string
  apiPort: number | string
  apiHost?: string
  wsPort?: number | string
  wsHost?: string
  distDir?: string
  serviceName?: string
  version?: string
  corsOrigins?: string | string[]
  verbose?: boolean
  configBuilder?: (env: Record<string, string | undefined>) => ScenarioConfig
  setupRoutes?: (app: any) => void
  proxyMetadata?: ProxyInfo
  scenarioConfig?: ScenarioConfig
}
```

**Fields:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `uiPort` | `number \| string` | **Required** | UI server port |
| `apiPort` | `number \| string` | **Required** | API server port |
| `apiHost` | `string` | `"127.0.0.1"` | API server host |
| `wsPort` | `number \| string` | `apiPort` | WebSocket port |
| `wsHost` | `string` | `apiHost` | WebSocket host |
| `distDir` | `string` | `"./dist"` | Static files directory |
| `serviceName` | `string` | - | Service name for logging |
| `version` | `string` | - | Service version |
| `corsOrigins` | `string \| string[]` | - | Allowed CORS origins |
| `verbose` | `boolean` | `false` | Enable verbose logging |
| `configBuilder` | `function` | - | Custom config builder |
| `setupRoutes` | `function` | - | Custom route setup function |
| `proxyMetadata` | [`ProxyInfo`](#proxyinfo) | - | Metadata to inject into HTML |
| `scenarioConfig` | [`ScenarioConfig`](#scenarioconfig) | - | Config to inject into HTML |

**Example:**

```typescript
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: 3000,
  apiPort: 8080,
  wsPort: 8081,
  distDir: './dist',
  serviceName: 'my-scenario',
  version: '1.0.0',
  corsOrigins: ['http://localhost:5173'],
  verbose: true,

  configBuilder: (env) => ({
    apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
    wsUrl: `ws://localhost:${env.WS_PORT}/ws`,
    apiPort: String(env.API_PORT),
    wsPort: String(env.WS_PORT),
    uiPort: String(env.UI_PORT),
    customField: env.CUSTOM_VALUE,
  }),

  setupRoutes: (app) => {
    app.get('/custom', (req, res) => {
      res.json({ message: 'Custom endpoint' })
    })
  },
})
```

**See Also:**
- [Server: createScenarioServer](./server.md#createscenarioserver)

---

### ProxyOptions

Options for creating API proxy middleware.

```typescript
interface ProxyOptions {
  apiPort: number | string
  apiHost?: string
  timeout?: number
  headers?: Record<string, string>
  verbose?: boolean
}
```

**Fields:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `apiPort` | `number \| string` | **Required** | Target API port |
| `apiHost` | `string` | `"127.0.0.1"` | Target API host |
| `timeout` | `number` | `30000` | Request timeout in ms |
| `headers` | `Record<string, string>` | `{}` | Additional headers |
| `verbose` | `boolean` | `false` | Enable request logging |

**Example:**

```typescript
import { createProxyMiddleware } from '@vrooli/api-base/server'

app.use('/api', createProxyMiddleware({
  apiPort: 8080,
  apiHost: '127.0.0.1',
  timeout: 60000,
  headers: {
    'X-Forwarded-For': 'proxy-server',
  },
  verbose: true,
}))
```

---

### ConfigEndpointOptions

Options for creating `/config` endpoint.

```typescript
interface ConfigEndpointOptions {
  apiPort: number | string
  apiHost?: string
  wsPort?: number | string
  wsHost?: string
  uiPort: number | string
  version?: string
  serviceName?: string
  additionalConfig?: Record<string, unknown>
  configBuilder?: () => ScenarioConfig
  cors?: boolean
  includeTimestamp?: boolean
  cacheControl?: boolean | string
}
```

**Fields:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `apiPort` | `number \| string` | **Required** | API port |
| `apiHost` | `string` | `"127.0.0.1"` | API host |
| `wsPort` | `number \| string` | `apiPort` | WebSocket port |
| `wsHost` | `string` | `apiHost` | WebSocket host |
| `uiPort` | `number \| string` | **Required** | UI port |
| `version` | `string` | - | Service version |
| `serviceName` | `string` | - | Service name |
| `additionalConfig` | `Record<string, unknown>` | `{}` | Extra config fields |
| `configBuilder` | `function` | - | Custom config builder |
| `cors` | `boolean` | `false` | Enable CORS headers |
| `includeTimestamp` | `boolean` | `false` | Add timestamp field |
| `cacheControl` | `boolean \| string` | `false` | Cache control header |

**Example:**

```typescript
import { createConfigEndpoint } from '@vrooli/api-base/server'

app.get('/config', createConfigEndpoint({
  apiPort: 8080,
  wsPort: 8081,
  uiPort: 3000,
  serviceName: 'my-scenario',
  version: '1.0.0',
  additionalConfig: {
    features: { analytics: true },
  },
  cors: true,
  cacheControl: 'no-cache',
}))
```

---

### HealthOptions

Options for creating `/health` endpoint.

```typescript
interface HealthOptions {
  serviceName: string
  version?: string
  apiPort?: number | string
  apiHost?: string
  timeout?: number
  customHealthCheck?: () => Promise<Record<string, unknown>>
}
```

**Fields:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `serviceName` | `string` | **Required** | Service name |
| `version` | `string` | - | Service version |
| `apiPort` | `number \| string` | - | API port (enables connectivity check) |
| `apiHost` | `string` | `"127.0.0.1"` | API host |
| `timeout` | `number` | `5000` | Health check timeout in ms |
| `customHealthCheck` | `function` | - | Additional health checks |

**Example:**

```typescript
import { createHealthEndpoint } from '@vrooli/api-base/server'

app.get('/health', createHealthEndpoint({
  serviceName: 'my-scenario-ui',
  version: '1.0.0',
  apiPort: 8080,
  timeout: 3000,
  customHealthCheck: async () => ({
    database: await checkDatabase(),
    cache: await checkCache(),
  }),
}))
```

---

### ProxyMetadataOptions

Options for building proxy metadata (used with `buildProxyMetadata`).

```typescript
interface ProxyMetadataOptions {
  appId: string
  hostScenario?: string
  targetScenario?: string
  ports: PortEntry[]
  primaryPort: PortEntry
  loopbackHosts?: string[]
}
```

**Fields:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `appId` | `string` | **Required** | Application/scenario ID |
| `hostScenario` | `string` | - | Host scenario name |
| `targetScenario` | `string` | - | Target scenario name |
| `ports` | [`PortEntry[]`](#portentry) | **Required** | Port configurations |
| `primaryPort` | [`PortEntry`](#portentry) | **Required** | Primary port |
| `loopbackHosts` | `string[]` | `["localhost", "127.0.0.1"]` | Loopback hostnames |

---

## Result Types

### HealthCheckResult

Result structure returned by health endpoints.

```typescript
interface HealthCheckResult {
  status: 'healthy' | 'degraded' | 'unhealthy'
  service: string
  timestamp: string
  version?: string
  readiness: boolean
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
  [key: string]: unknown
}
```

**Fields:**

| Name | Type | Description |
|------|------|-------------|
| `status` | `'healthy' \| 'degraded' \| 'unhealthy'` | Overall health status |
| `service` | `string` | Service identifier |
| `timestamp` | `string` | ISO 8601 timestamp |
| `version` | `string` | Service version (if provided) |
| `readiness` | `boolean` | Whether ready for traffic |
| `api_connectivity` | `object` | API connectivity check results |
| `[key]` | `unknown` | Custom health check fields |

**Status Codes:**

| Status | HTTP Code | Description |
|--------|-----------|-------------|
| `healthy` | `200` | All checks passed |
| `degraded` | `503` | API unreachable but UI functional |
| `unhealthy` | `503` | Critical failure |

**Example Response:**

```json
{
  "status": "healthy",
  "service": "my-scenario-ui",
  "version": "1.0.0",
  "timestamp": "2025-01-01T00:00:00.000Z",
  "readiness": true,
  "api_connectivity": {
    "connected": true,
    "api_url": "http://127.0.0.1:8080/health",
    "last_check": "2025-01-01T00:00:00.000Z",
    "error": null,
    "latency_ms": 15
  }
}
```

---

## See Also

- [Client API Reference](./client.md)
- [Server API Reference](./server.md)
- [Quick Start Guide](../guides/quick-start.md)
