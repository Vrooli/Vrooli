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
  - [HostEndpointDefinition](#hostendpointdefinition)
  - [WindowLike](#windowlike)
  - [ProxyIndex](#proxyindex)
- [Client Options](#client-options)
  - [ResolveOptions](#resolveoptions)
  - [BuildUrlOptions](#buildurloptions)
- [Server Options](#server-options)
  - [ServerTemplateOptions](#servertemplateoptions)
  - [ScenarioProxyHostOptions](#scenarioproxyhostoptions)
  - [ScenarioProxyHostController](#scenarioproxyhostcontroller)
  - [ScenarioProxyAppMetadata](#scenarioproxyappmetadata)
  - [ProxyOptions](#proxyoptions)
  - [ProxyMetadataOptions](#proxymetadataoptions)
  - [ConfigEndpointOptions](#configendpointoptions)
  - [HealthOptions](#healthoptions)
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
  hostEndpoints?: HostEndpointDefinition[]
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
| `hostEndpoints` | [`HostEndpointDefinition[]`](#hostendpointdefinition) | Host-owned endpoints that bypass proxy fetch rewriting |

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

### HostEndpointDefinition

Defines a host-owned endpoint path that should bypass proxy fetch rewriting.

[CODE: packages/api-base/src/shared/types.ts#HostEndpointDefinition]

```typescript
interface HostEndpointDefinition {
  path: string
  method?: string
}
```

**Fields:**

| Name | Type | Description |
|------|------|-------------|
| `path` | `string` | Path pattern beginning with `/` (supports `:param` and `*` wildcards) |
| `method` | `string` | Optional HTTP method (e.g., `'GET'`, `'POST'`). If omitted, matches all methods. |

**Purpose:**

When the host scenario (e.g., app-monitor) has its own endpoints like `/scenarios` or `/health-aggregate`, the client-side fetch patching must **not** rewrite requests to those paths through the child scenario's proxy. Host endpoint definitions tell the patched `fetch()` which paths belong to the host.

**Example:**

```typescript
const hostEndpoints: HostEndpointDefinition[] = [
  { path: '/scenarios' },
  { path: '/health-aggregate' },
  { path: '/api/v1/apps/:appId', method: 'GET' },
  { path: '/ws', method: 'WS' },
]
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
| `defaultPort` | `string` | `"15000"` | **⚠️ Avoid in production!** Only for SSR/testing. Omit for production bundles to use `window.location.origin` |
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
// ✅ RECOMMENDED: Production bundles (Vrooli scenarios)
resolveApiBase({ appendSuffix: true })

// ✅ Custom suffix
resolveApiBase({
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
  wsPathPrefix?: string
  wsPathTransform?: (path: string) => string
  proxyHeaders?: Record<string, string> | ((req: any) => Record<string, string>)
  proxyTimeoutMs?: number
  proxyKeepAlive?: boolean
  proxyAgent?: import('node:http').Agent
  bodyParser?: 'json' | false | ((app: any) => void)
  cacheIndexHtml?: boolean
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
| `wsPathPrefix` | `string` | - | Mount path that should be proxied for WebSockets |
| `wsPathTransform` | `function` | replace prefix with `/api/v1` | Override how WS paths are rewritten before hitting the API |
| `proxyHeaders` | `Record<string, string> \| (req) => Record<string, string>` | - | Extra headers appended to every proxied request |
| `proxyTimeoutMs` | `number` | `15000` | Override HTTP proxy timeout |
| `proxyKeepAlive` | `boolean` | `true` | Control whether UI->API connections are pooled |
| `proxyAgent` | `Agent` | shared keep-alive agent | Provide your own Node HTTP agent |
| `bodyParser` | `'json' \| false \| (app) => void` | `'json'` | Configure body parsing for UI-owned routes (runs after the `/api` proxy) |
| `cacheIndexHtml` | `boolean` | `true` | Cache `dist/index.html` between SPA fallback requests (auto-invalidates on rebuilds) |

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

### ScenarioProxyHostOptions

Options for configuring the scenario proxy host (e.g., app-monitor).

[CODE: packages/api-base/src/shared/types.ts#ScenarioProxyHostOptions]

```typescript
interface ScenarioProxyHostOptions {
  hostScenario: string
  fetchAppMetadata: (appId: string) => Promise<ScenarioProxyAppMetadata | null | undefined>
  appsPathPrefix?: string
  proxyPathSegment?: string
  portsPathSegment?: string
  loopbackHosts?: string[]
  cacheTtlMs?: number
  upstreamHost?: string
  timeoutMs?: number
  verbose?: boolean
  patchFetch?: boolean
  childBaseTagAttribute?: string
  proxiedAppHeader?: string
  hostEndpoints?: HostEndpointDefinition[]
  proxyKeepAlive?: boolean
  proxyAgent?: import('node:http').Agent
  cacheProxyHtml?: boolean
  proxyHtmlCacheTtlMs?: number
  proxyHtmlCacheMaxEntries?: number
  healthCheckIntervalMs?: number
  healthCheckTimeoutMs?: number
  enableServerTiming?: boolean
  enableMetrics?: boolean
  metricsSampleSize?: number
}
```

**Fields:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `hostScenario` | `string` | **Required** | Host scenario identifier (e.g., `'app-monitor'`) |
| `fetchAppMetadata` | `(appId) => Promise<...>` | **Required** | Fetches port mappings from host API |
| `appsPathPrefix` | `string` | `'/apps'` | Base path prefix for app routes |
| `proxyPathSegment` | `string` | `'proxy'` | Proxy marker in URLs |
| `portsPathSegment` | `string` | `'ports'` | Named port segment in URLs |
| `loopbackHosts` | `string[]` | `['127.0.0.1', 'localhost', '::1']` | Hosts for metadata injection |
| `cacheTtlMs` | `number` | `300000` | Metadata cache TTL in ms |
| `upstreamHost` | `string` | `'127.0.0.1'` | Host running scenarios |
| `timeoutMs` | `number` | `30000` | Upstream request timeout in ms |
| `verbose` | `boolean` | `false` | Enable logging |
| `patchFetch` | `boolean` | `false` | Patch fetch/XHR/WebSocket in injected scripts |
| `childBaseTagAttribute` | `string` | `'data-proxy-host'` | Data attribute on injected `<base>` tag |
| `proxiedAppHeader` | `string` | `'x-vrooli-proxied-app'` | Response header identifying proxied app |
| `hostEndpoints` | `HostEndpointDefinition[]` | `[]` | Host paths that bypass fetch patching |
| `proxyKeepAlive` | `boolean` | `true` | Reuse upstream connections |
| `proxyAgent` | `http.Agent` | - | Custom HTTP agent for upstream requests |
| `cacheProxyHtml` | `boolean` | `true` | Cache proxied HTML responses |
| `proxyHtmlCacheTtlMs` | `number` | `cacheTtlMs` | HTML cache TTL in ms |
| `proxyHtmlCacheMaxEntries` | `number` | `200` | Max cached HTML entries (FIFO eviction) |
| `healthCheckIntervalMs` | `number` | `5000` | Interval between background TCP health probes (ms) |
| `healthCheckTimeoutMs` | `number` | `500` | Timeout for each background health probe (ms) |
| `enableServerTiming` | `boolean` | `true` | Emit `Server-Timing` header |
| `enableMetrics` | `boolean` | `false` | Collect aggregate metrics at `/__perf` |
| `metricsSampleSize` | `number` | `1000` | Ring-buffer size for percentile samples |

**See Also:**
- [Server: createScenarioProxyHost](./server.md#createscenarioproxyhost)

---

### ScenarioProxyHostController

Controller returned by `createScenarioProxyHost()`.

[CODE: packages/api-base/src/shared/types.ts#ScenarioProxyHostController]

```typescript
interface ScenarioProxyHostController {
  router: Router
  handleUpgrade: (req: IncomingMessage, socket: any, head: Buffer) => Promise<boolean>
  invalidate: (appId?: string) => void
  clearCache: () => void
  getMetrics: () => object | null
  resetMetrics: () => void
  destroy: () => void
}
```

**Fields:**

| Name | Type | Description |
|------|------|-------------|
| `router` | `Router` | Express router with proxy routes - mount with `app.use(controller.router)` |
| `handleUpgrade` | `(req, socket, head) => Promise<boolean>` | WebSocket upgrade handler. Returns `true` if the request was handled. |
| `invalidate` | `(appId?) => void` | Clear metadata + HTML cache for one app (by ID) or all apps (no argument) |
| `clearCache` | `() => void` | Clear entire cache (metadata + HTML) |
| `getMetrics` | `() => object \| null` | Latency/cache metrics snapshot (`null` if metrics disabled) |
| `resetMetrics` | `() => void` | Reset metric counters |
| `destroy` | `() => void` | Stop background health checks and release resources |

---

### ScenarioProxyAppMetadata

Raw app metadata returned by the host API. Used by `fetchAppMetadata` to provide port information for proxy routing.

[CODE: packages/api-base/src/shared/types.ts#ScenarioProxyAppMetadata]

```typescript
interface ScenarioProxyAppMetadata {
  id?: string
  appId?: string
  scenario?: string
  scenario_name?: string
  scenarioName?: string
  name?: string
  port_mappings?: Record<string, unknown>
  portMappings?: Record<string, unknown>
  config?: Record<string, unknown>
  [key: string]: unknown
}
```

**Fields:**

| Name | Type | Description |
|------|------|-------------|
| `id` | `string` | Application identifier |
| `appId` | `string` | Alternate application identifier |
| `scenario` | `string` | Scenario name |
| `scenario_name` | `string` | Scenario name (snake_case) |
| `scenarioName` | `string` | Scenario name (camelCase) |
| `name` | `string` | Human-readable name |
| `port_mappings` | `Record<string, unknown>` | Port mappings (e.g., `{ ui: 3001, api: 8080 }`) |
| `portMappings` | `Record<string, unknown>` | Alternate port mappings (camelCase) |
| `config` | `Record<string, unknown>` | Additional configuration (e.g., `{ primary_port: 3001 }`) |

**Note:** Multiple naming conventions are supported for flexibility with different API response formats. The proxy system normalizes these internally.

---

### ProxyOptions

Options for creating API proxy middleware.

```typescript
interface ProxyOptions {
  apiPort: number | string
  apiHost?: string
  timeout?: number
  headers?: Record<string, string> | ((req: any) => Record<string, string>)
  verbose?: boolean
  keepAlive?: boolean
  agent?: import('node:http').Agent
}
```

**Fields:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `apiPort` | `number \| string` | **Required** | Target API port |
| `apiHost` | `string` | `"127.0.0.1"` | Target API host |
| `timeout` | `number` | `30000` | Request timeout in ms |
| `headers` | `Record<string, string> \| (req) => Record<string, string>` | `{}` | Additional headers (static object or per-request function) |
| `verbose` | `boolean` | `false` | Enable request logging |
| `keepAlive` | `boolean` | `true` | Reuse HTTP connections via keep-alive agent |
| `agent` | `http.Agent` | - | Custom HTTP agent for connection pooling |

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
  hostEndpoints?: HostEndpointDefinition[]
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
| `hostEndpoints` | [`HostEndpointDefinition[]`](#hostendpointdefinition) | - | Host-owned endpoints that bypass fetch patching |

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
