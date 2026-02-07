# Server API Reference

Complete reference for all server-side functions in `@vrooli/api-base/server`.

**Import:**
```typescript
import {
  createScenarioServer,
  startScenarioServer,
  createScenarioProxyHost,
  injectProxyMetadata,
  injectScenarioConfig,
  injectBaseTag,
  buildProxyMetadata,
  createProxyMiddleware,
  proxyToApi,
  proxyWebSocketUpgrade,
  createConfigEndpoint,
  createHealthEndpoint,
  createSimpleHealthEndpoint,
} from '@vrooli/api-base/server'
```

## Table of Contents

- [Server Template](#server-template)
  - [createScenarioServer](#createscenarioserver)
  - [startScenarioServer](#startscenarioserver)
- [Scenario Proxy Host](#scenario-proxy-host)
  - [createScenarioProxyHost](#createscenarioproxyhost)
- [Metadata Injection](#metadata-injection)
  - [injectProxyMetadata](#injectproxymetadata)
  - [injectScenarioConfig](#injectscenarioconfig)
  - [injectBaseTag](#injectbasetag)
  - [buildProxyMetadata](#buildproxymetadata)
- [API Proxying](#api-proxying)
  - [createProxyMiddleware](#createproxymiddleware)
  - [proxyToApi](#proxytoapi)
  - [proxyWebSocketUpgrade](#proxywebsocketupgrade)
- [Endpoints](#endpoints)
  - [createConfigEndpoint](#createconfigendpoint)
  - [createHealthEndpoint](#createhealthendpoint)
  - [createSimpleHealthEndpoint](#createsimplehealthendpoint)
- [Types](#types)

---

## Server Template

### createScenarioServer

Creates a fully-configured Express application with all standard Vrooli scenario features.

**Signature:**
```typescript
function createScenarioServer(options: ServerTemplateOptions): Express.Application
```

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `uiPort` | `string \| number` | ✅ | UI server port |
| `apiPort` | `string \| number` | ✅ | API server port |
| `apiHost` | `string` | ❌ | API host (default: `127.0.0.1`) |
| `wsPort` | `string \| number` | ❌ | WebSocket port (default: same as `apiPort`) |
| `wsHost` | `string` | ❌ | WebSocket host (default: same as `apiHost`) |
| `distDir` | `string` | ❌ | Static files directory (default: `./dist`) |
| `serviceName` | `string` | ❌ | Service name for logging |
| `version` | `string` | ❌ | Service version |
| `corsOrigins` | `string \| string[]` | ❌ | Allowed CORS origins. Use `'*'` for all origins, or specify patterns like `['https://*.example.com']`. Default: localhost + auto-detected tunnels |
| `verbose` | `boolean` | ❌ | Enable verbose logging (default: `false`) |
| `configBuilder` | `(env: NodeJS.ProcessEnv) => ScenarioConfig` | ❌ | Custom config builder |
| `setupRoutes` | `(app: Express.Application) => void` | ❌ | Custom route setup |
| `proxyMetadata` | `ProxyInfo` | ❌ | Proxy metadata to inject |
| `scenarioConfig` | `ScenarioConfig` | ❌ | Scenario config to inject |

**Returns:** `Express.Application` - Configured Express app

**Features:**
- ✅ JSON body parsing (10MB limit)
- ✅ CORS middleware
- ✅ API request proxying (`/api/*` → API server)
- ✅ `/config` endpoint (runtime configuration)
- ✅ `/health` endpoint (with API connectivity check)
- ✅ Static file serving from `distDir`
- ✅ SPA fallback routing (serves `index.html`)
- ✅ HTML metadata injection (proxy info, config)
- ✅ Graceful error handling

**Examples:**

**Basic Usage:**
```typescript
import express from 'express'
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT || 3000,
  apiPort: process.env.API_PORT || 8080,
  distDir: './dist',
})

app.listen(3000, () => {
  console.log('Server running on port 3000')
})
```

**With All Options:**
```typescript
const app = createScenarioServer({
  // Required
  uiPort: process.env.UI_PORT || 3000,
  apiPort: process.env.API_PORT || 8080,

  // Optional ports
  wsPort: process.env.WS_PORT || 8081,

  // Service info
  serviceName: 'my-scenario-ui',
  version: '1.0.0',

  // Static files
  distDir: path.join(__dirname, '../dist'),

  // CORS - allow all origins (recommended for scenarios accessible via tunnels)
  corsOrigins: '*',
  // Or specify patterns: ['http://localhost:*', 'https://*.example.com']

  // Debugging
  verbose: process.env.NODE_ENV === 'development',

  // Extend API proxy timeout (default 15s)
  proxyTimeoutMs: 60000,
  // Connection reuse controls (default keep-alive on)
  proxyKeepAlive: true,

  // Custom config
  configBuilder: (env) => ({
    apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
    wsUrl: `ws://localhost:${env.WS_PORT}/ws`,
    apiPort: String(env.API_PORT),
    wsPort: String(env.WS_PORT),
    uiPort: String(env.UI_PORT),
    customField: env.CUSTOM_VALUE,
  }),

  // Custom routes
  setupRoutes: (app) => {
    app.get('/custom', (req, res) => {
      res.json({ message: 'Custom endpoint' })
    })

    app.post('/webhook', async (req, res) => {
      // Handle webhook
      res.json({ received: true })
    })
  },
})
```

> Need full control over pooling? Pass `proxyAgent: new http.Agent({...})` to reuse an existing agent, or set `proxyKeepAlive: false` to fall back to one-request-per-connection behavior.

**With Proxy Metadata Injection:**
```typescript
import { buildProxyMetadata } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: 3000,
  apiPort: 8080,
  distDir: './dist',

  // Inject proxy metadata for embedded scenarios
  proxyMetadata: buildProxyMetadata({
    hostScenario: 'my-dashboard',
    targetScenario: 'embedded-app',
    basePath: '/embed/embedded-app/proxy',
    ports: [
      { port: 3001, label: 'ui', slug: 'ui' },
      { port: 8081, label: 'api', slug: 'api' },
    ],
    primaryPort: 3001,
  }),
})
```

**See Also:**
- [Quick Start Guide](../guides/quick-start.md#server-setup)
- [startScenarioServer](#startscenarioserver) (with auto-listen)

---

### startScenarioServer

Creates and starts a scenario server with automatic listening and lifecycle management.

**Signature:**
```typescript
function startScenarioServer(options: ServerTemplateOptions): Express.Application
```

**Parameters:**
Same as [createScenarioServer](#createscenarioserver)

**Returns:** `Express.Application` - Running Express app

**Features:**
- All features from `createScenarioServer`
- ✅ Automatically calls `app.listen()`
- ✅ Logs server URLs on startup
- ✅ Graceful shutdown on SIGTERM/SIGINT

**Example:**

```typescript
import { startScenarioServer } from '@vrooli/api-base/server'

// Single function call - server starts immediately
startScenarioServer({
  uiPort: process.env.UI_PORT || 3000,
  apiPort: process.env.API_PORT || 8080,
  distDir: './dist',
  serviceName: 'my-scenario',
})

// Console output:
// my-scenario UI server listening on port 3000
// Health: http://localhost:3000/health
// Config: http://localhost:3000/config
// UI: http://localhost:3000
```

**Graceful Shutdown:**
```typescript
// Server automatically handles SIGTERM/SIGINT
// Press Ctrl+C to trigger:
// ^C
// Shutting down gracefully...
```

**When to Use:**
- ✅ Simple scenarios that don't need custom server logic
- ✅ Development servers
- ✅ Production deployments with process managers (PM2, systemd)
- ❌ Complex applications that need custom server initialization
- ❌ Testing (use `createScenarioServer` for more control)

**See Also:**
- [createScenarioServer](#createscenarioserver) (for more control)

---

## Scenario Proxy Host

### createScenarioProxyHost

Creates a proxy host controller that routes requests from a host scenario (like app-monitor) to child scenarios. This is the core function powering the entire proxy system.

[CODE: packages/api-base/src/server/host.ts#createScenarioProxyHost]

**Signature:**
```typescript
function createScenarioProxyHost(options: ScenarioProxyHostOptions): ScenarioProxyHostController
```

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `hostScenario` | `string` | Yes | Host scenario identifier (e.g., `'app-monitor'`) |
| `fetchAppMetadata` | `(appId: string) => Promise<ScenarioProxyAppMetadata>` | Yes | Fetches port mappings from host API |
| `appsPathPrefix` | `string` | No | Base path prefix (default: `'/apps'`) |
| `proxyPathSegment` | `string` | No | Proxy marker segment (default: `'proxy'`) |
| `portsPathSegment` | `string` | No | Named port segment (default: `'ports'`) |
| `loopbackHosts` | `string[]` | No | Hosts for metadata injection (default: loopback addresses) |
| `cacheTtlMs` | `number` | No | Metadata cache TTL in ms (default: `30000`) |
| `upstreamHost` | `string` | No | Host running scenarios (default: `'127.0.0.1'`) |
| `timeoutMs` | `number` | No | Upstream request timeout (default: `30000`) |
| `verbose` | `boolean` | No | Enable logging (default: `false`) |
| `patchFetch` | `boolean` | No | Patch fetch/XHR in injected scripts (default: `false`) |
| `childBaseTagAttribute` | `string` | No | Data attribute on injected `<base>` tag (default: `'data-proxy-host'`) |
| `proxiedAppHeader` | `string` | No | Response header identifying proxied app (default: `'x-vrooli-proxied-app'`) |
| `hostEndpoints` | `HostEndpointDefinition[]` | No | Host paths that bypass fetch patching |
| `cacheProxyHtml` | `boolean` | No | Cache proxied HTML responses (default: `true`) |
| `proxyHtmlCacheTtlMs` | `number` | No | HTML cache TTL (defaults to `cacheTtlMs`) |
| `proxyHtmlCacheMaxEntries` | `number` | No | Max cached HTML entries (default: `200`) |
| `proxyKeepAlive` | `boolean` | No | Reuse upstream connections (default: `true`) |
| `proxyAgent` | `http.Agent` | No | Custom HTTP agent for upstream requests |
| `enableServerTiming` | `boolean` | No | Emit `Server-Timing` header (default: `true`) |
| `enableMetrics` | `boolean` | No | Collect aggregate metrics at `/__perf` (default: `false`) |
| `metricsSampleSize` | `number` | No | Ring-buffer size for percentile samples (default: `1000`) |

**Returns:** `ScenarioProxyHostController`

| Property | Type | Description |
|----------|------|-------------|
| `router` | `Router` | Express router with proxy routes - mount with `app.use(controller.router)` |
| `handleUpgrade` | `(req, socket, head) => Promise<boolean>` | WebSocket upgrade handler. Returns `true` if handled. |
| `invalidate` | `(appId?: string) => void` | Clear metadata + HTML cache for one or all apps |
| `clearCache` | `() => void` | Clear entire cache |
| `getMetrics` | `() => object \| null` | Latency/cache metrics snapshot (null if metrics disabled) |
| `resetMetrics` | `() => void` | Reset metric counters |

**Registered Routes:**

```
/apps/:appId/ports/:portKey/proxy/*   -> named port proxy
/apps/:appId/ports/:portKey/proxy     -> named port proxy (no trailing path)
/apps/:appId/proxy/*                  -> primary port proxy
/apps/:appId/proxy                    -> primary port proxy (no trailing path)
/__perf                               -> metrics JSON (if enableMetrics)
/__perf/reset                         -> reset metrics (if enableMetrics)
```

**Example (app-monitor):**

```javascript
import { createScenarioProxyHost, createScenarioServer } from '@vrooli/api-base/server'
import axios from 'axios'

const proxyHost = createScenarioProxyHost({
  hostScenario: 'app-monitor',
  cacheTtlMs: 300_000,  // 5 min (Go API invalidates on changes)
  patchFetch: true,
  verbose: true,
  fetchAppMetadata: async (appId) => {
    const res = await axios.get(`http://127.0.0.1:${API_PORT}/api/v1/apps/${appId}`)
    return res.data?.data || res.data
  },
})

const app = createScenarioServer({
  uiPort: PORT,
  apiPort: API_PORT,
  setupRoutes: (expressApp) => {
    // Cache invalidation endpoint (called by Go API)
    expressApp.post('/__cache/invalidate', (req, res) => {
      const { appId } = req.body || {}
      appId ? proxyHost.invalidate(appId) : proxyHost.clearCache()
      res.status(204).end()
    })

    // Mount proxy router last
    expressApp.use(proxyHost.router)
  },
})

// Handle WebSocket upgrades
const server = http.createServer(app)
server.on('upgrade', async (req, socket, head) => {
  if (await proxyHost.handleUpgrade(req, socket, head)) return
  socket.destroy()
})
```

**Request Flow:**

```
GET /apps/my-scenario/proxy/dashboard
    |
    +-> Path traversal check (appId + relative URL)
    +-> Metadata cache lookup (or fetchAppMetadata)
    +-> Route decision:
        |
        +-- /api path?      -> Forward to apiPort (no injection)
        +-- HTML request?    -> Stream from uiPort + inject metadata
        +-- Other (JS, CSS)  -> Forward to uiPort as-is
```

**See Also:**
- [Architecture Overview](../concepts/ARCHITECTURE.md) - Visual system diagrams
- [Host Scenario Pattern](../guides/host-scenario-pattern.md) - Step-by-step guide

---

## Metadata Injection

### injectProxyMetadata

Injects proxy metadata into HTML content.

**Signature:**
```typescript
function injectProxyMetadata(
  html: string,
  metadata: ProxyInfo,
  options?: {
    infoGlobalName?: string
    indexGlobalName?: string
    patchFetch?: boolean
  }
): string
```

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `html` | `string` | Yes | HTML content to modify |
| `metadata` | `ProxyInfo` | Yes | Proxy metadata to inject |
| `options.infoGlobalName` | `string` | No | Window global for metadata (default: `'__VROOLI_PROXY_INFO__'`) |
| `options.indexGlobalName` | `string` | No | Window global for index (default: `'__VROOLI_PROXY_INDEX__'`) |
| `options.patchFetch` | `boolean` | No | Patch fetch/XHR/WebSocket to rewrite localhost URLs (default: `false`) |

**Returns:** `string` - Modified HTML with injected `<script>` tag

**Injection Location:**
Inserts a `<script>` tag into the `<head>` section that sets:
- `window.__VROOLI_PROXY_INFO__` (or custom global)
- `window.__VROOLI_PROXY_INDEX__` (or custom global)
- When `patchFetch: true`: patches `fetch()`, `XMLHttpRequest.open()`, and `new WebSocket()` to rewrite localhost URLs through the proxy

**Example:**

```typescript
import { injectProxyMetadata, buildProxyMetadata } from '@vrooli/api-base/server'
import fs from 'fs'

// Read HTML
const html = fs.readFileSync('./dist/index.html', 'utf-8')

// Build metadata
const metadata = buildProxyMetadata({
  hostScenario: 'app-monitor',
  targetScenario: 'my-scenario',
  basePath: '/apps/my-scenario/proxy',
  ports: [
    { port: 3000, label: 'ui', slug: 'ui' },
    { port: 8080, label: 'api', slug: 'api' },
  ],
  primaryPort: 3000,
})

// Inject
const modifiedHtml = injectProxyMetadata(html, metadata)

// modifiedHtml now contains:
// <head>
//   ...
//   <script>
//     window.__VROOLI_PROXY_INFO__ = { ... };
//     window.__VROOLI_PROXY_INDEX__ = { ... };
//   </script>
// </head>
```

**Use Cases:**
- Scenario A embedding Scenario B in an iframe
- App-monitor displaying scenario previews
- Custom dashboards with embedded widgets

**See Also:**
- [buildProxyMetadata](#buildproxymetadata)
- [Proxy Resolution](../concepts/proxy-resolution.md)

---

### injectScenarioConfig

Injects scenario configuration into HTML content.

**Signature:**
```typescript
function injectScenarioConfig(html: string, config: ScenarioConfig): string
```

**Parameters:**

| Name | Type | Description |
|------|------|-------------|
| `html` | `string` | HTML content to modify |
| `config` | `ScenarioConfig` | Configuration to inject |

**Returns:** `string` - Modified HTML with injected `<script>` tag

**Injection Location:**
Inserts a `<script>` tag that sets `window.__VROOLI_CONFIG__`

**Example:**

```typescript
import { injectScenarioConfig } from '@vrooli/api-base/server'
import fs from 'fs'

const html = fs.readFileSync('./dist/index.html', 'utf-8')

const config = {
  apiUrl: `http://localhost:${process.env.API_PORT}/api/v1`,
  wsUrl: `ws://localhost:${process.env.WS_PORT}/ws`,
  apiPort: String(process.env.API_PORT),
  wsPort: String(process.env.WS_PORT),
  uiPort: String(process.env.UI_PORT),
  serviceName: 'my-scenario',
  version: '1.0.0',
}

const modifiedHtml = injectScenarioConfig(html, config)

// modifiedHtml now contains:
// <head>
//   ...
//   <script>
//     window.__VROOLI_CONFIG__ = {
//       apiUrl: "http://localhost:8080/api/v1",
//       wsUrl: "ws://localhost:8081/ws",
//       ...
//     };
//   </script>
// </head>
```

**Client-Side Usage:**
```typescript
// Client can access config synchronously
import { getScenarioConfig } from '@vrooli/api-base'

const config = getScenarioConfig()
if (config) {
  console.log(`API: ${config.apiUrl}`)
}
```

**See Also:**
- [Runtime Configuration](../concepts/runtime-config.md)
- [Client: getScenarioConfig](./client.md#getscenarioconfig)

---

### buildProxyMetadata

Builds proxy metadata object from configuration.

**Signature:**
```typescript
function buildProxyMetadata(options: ProxyMetadataOptions): ProxyInfo
```

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `hostScenario` | `string` | ✅ | Name of hosting scenario |
| `targetScenario` | `string` | ✅ | Name of embedded scenario |
| `basePath` | `string` | ✅ | Proxy base path |
| `ports` | `PortEntry[]` | ✅ | Port configurations |
| `primaryPort` | `number` | ✅ | Primary port number |
| `hosts` | `string[]` | ❌ | Loopback hostnames (default: `['localhost', '127.0.0.1']`) |

**Returns:** `ProxyInfo` - Proxy metadata object

**Example:**

```typescript
import { buildProxyMetadata } from '@vrooli/api-base/server'

const metadata = buildProxyMetadata({
  hostScenario: 'app-monitor',
  targetScenario: 'scenario-auditor',
  basePath: '/apps/scenario-auditor/proxy',
  ports: [
    { port: 36224, label: 'ui', slug: 'ui' },
    { port: 18508, label: 'api', slug: 'api' },
  ],
  primaryPort: 36224,
})

// Use with injectProxyMetadata
const html = injectProxyMetadata(htmlContent, metadata)
```

---

### injectBaseTag

Injects a `<base href="...">` tag into HTML to control relative URL resolution.

[CODE: packages/api-base/src/server/inject.ts#injectBaseTag]

**Signature:**
```typescript
function injectBaseTag(
  html: string,
  basePath: string,
  options?: {
    skipIfExists?: boolean
    dataAttribute?: string
  }
): string
```

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `html` | `string` | Yes | HTML content to modify |
| `basePath` | `string` | Yes | Base path (e.g., `'/'` or `'/apps/scenario/proxy/'`) |
| `options.skipIfExists` | `boolean` | No | Skip if `<base>` already present (default: `true`) |
| `options.dataAttribute` | `string` | No | Data attribute for identification (default: `'data-vrooli-base'`) |

**Returns:** `string` - Modified HTML with `<base>` tag injected into `<head>`

**Example:**

```typescript
import { injectBaseTag } from '@vrooli/api-base/server'

// For host scenario (load assets from root)
html = injectBaseTag(html, '/', {
  dataAttribute: 'data-app-monitor-self',
})
// Result: <head><base data-app-monitor-self="injected" href="/">

// For proxied scenario (load assets through proxy)
html = injectBaseTag(html, '/apps/scenario-a/proxy/', {
  dataAttribute: 'data-app-monitor',
})
// Result: <head><base data-app-monitor="injected" href="/apps/scenario-a/proxy/">
```

**Why it matters:** Without a `<base>` tag, a scenario at `/apps/x/proxy/` requesting `./assets/main.js` would resolve to `/apps/x/proxy/assets/main.js` (correct). But `fetch('/api/health')` would resolve to `/api/health` (incorrect - bypasses proxy). The `<base>` tag plus fetch patching together handle both cases.

**See Also:**
- [Host Scenario Pattern](../guides/host-scenario-pattern.md) - When to use each base path

---

## API Proxying

### createProxyMiddleware

Creates Express middleware that proxies API requests to the API server.

**Signature:**
```typescript
function createProxyMiddleware(options: ProxyOptions): RequestHandler
```

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `apiPort` | `number` | ✅ | API server port |
| `apiHost` | `string` | ❌ | API server host (default: `127.0.0.1`) |
| `timeout` | `number` | ❌ | Request timeout in ms (default: `30000`) |
| `verbose` | `boolean` | ❌ | Enable verbose logging (default: `false`) |

**Returns:** `RequestHandler` - Express middleware function

**Example:**

```typescript
import express from 'express'
import { createProxyMiddleware } from '@vrooli/api-base/server'

const app = express()

// Proxy all /api/* requests to API server
app.use('/api', createProxyMiddleware({
  apiPort: 8080,
  apiHost: '127.0.0.1',
  timeout: 60000,  // 60 second timeout
  verbose: true,   // Log all proxy requests
}))

// Client request:   GET /api/v1/health
// Proxied to:       GET http://127.0.0.1:8080/v1/health
```

**Features:**
- ✅ Streams request/response bodies (no buffering)
- ✅ Preserves headers (except hop-by-hop)
- ✅ Handles timeouts gracefully
- ✅ Proper error responses (502/504)
- ✅ Optional verbose logging

**Error Responses:**

| Status | Condition |
|--------|-----------|
| `502` | API connection failed |
| `504` | API request timeout |

**See Also:**
- [proxyToApi](#proxytoapi) (lower-level function)

---

### proxyToApi

Low-level function to proxy a single request to the API server.

[CODE: packages/api-base/src/server/proxy.ts#proxyToApi]

**Signature:**
```typescript
async function proxyToApi(
  req: Request,
  res: Response,
  targetPath: string,
  options: ProxyOptions
): Promise<void>
```

**Parameters:**

| Name | Type | Description |
|------|------|-------------|
| `req` | `Request` | Express request object |
| `res` | `Response` | Express response object |
| `targetPath` | `string` | API endpoint path to forward to |
| `options` | `ProxyOptions` | Proxy configuration |

**ProxyOptions:**

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `apiPort` | `number \| string` | **Required** | Target API server port |
| `apiHost` | `string` | `'127.0.0.1'` | Target API server host |
| `timeout` | `number` | `30000` | Request timeout in ms |
| `headers` | `Record<string, string> \| (req) => Record<string, string>` | `{}` | Additional headers (static or per-request function) |
| `verbose` | `boolean` | `false` | Enable logging |
| `keepAlive` | `boolean` | `true` | Reuse HTTP connections |
| `agent` | `http.Agent` | - | Custom HTTP agent |

**Example:**

```typescript
import { proxyToApi } from '@vrooli/api-base/server'

app.use('/custom-api', async (req, res) => {
  // Custom logic before proxying
  console.log(`Proxying: ${req.method} ${req.url}`)

  // Proxy to API with path rewrite
  await proxyToApi(req, res, `/v2${req.url}`, {
    apiPort: 8080,
    apiHost: '127.0.0.1',
    timeout: 30000,
    verbose: true,
  })

  // Custom logic after proxying
  console.log(`Proxy complete: ${res.statusCode}`)
})
```

**When to Use:**
- ✅ Need custom request/response handling
- ✅ Path rewriting or transformation
- ✅ Custom error handling
- ❌ Simple proxying (use `createProxyMiddleware` instead)

---

### proxyWebSocketUpgrade

Handles WebSocket upgrade requests by establishing a raw TCP tunnel between client and upstream server.

[CODE: packages/api-base/src/server/proxy.ts#proxyWebSocketUpgrade]

**Signature:**
```typescript
function proxyWebSocketUpgrade(
  req: http.IncomingMessage,
  clientSocket: net.Socket,
  head: Buffer,
  options: ProxyOptions
): void
```

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `req` | `http.IncomingMessage` | Yes | HTTP upgrade request |
| `clientSocket` | `net.Socket` | Yes | Client socket from upgrade event |
| `head` | `Buffer` | Yes | Initial data buffer |
| `options.apiPort` | `number \| string` | Yes | Target port for the WebSocket server |
| `options.apiHost` | `string` | No | Target host (default: `'127.0.0.1'`) |
| `options.verbose` | `boolean` | No | Enable logging (default: `false`) |

**How it works:**

```
Client socket  <--> net.connect(port, host) <--> Upstream socket
                    (bidirectional pipe)
```

1. Opens TCP connection to upstream via `net.connect()`
2. Forwards the HTTP upgrade request with **all** headers preserved (including `Sec-WebSocket-*`)
3. Pipes data bidirectionally between client and upstream sockets
4. Tears down both sockets on error or close

**Example:**

```typescript
import { proxyWebSocketUpgrade } from '@vrooli/api-base/server'

server.on('upgrade', (req, socket, head) => {
  if (req.url?.startsWith('/api')) {
    proxyWebSocketUpgrade(req, socket, head, {
      apiPort: process.env.API_PORT,
      apiHost: '127.0.0.1',
      verbose: true,
    })
  } else {
    socket.destroy()
  }
})
```

**Error Handling:**

| Condition | Response |
|-----------|----------|
| Invalid port | HTTP 502 written to socket, then destroyed |
| Upstream connect error | Both sockets torn down, error logged |
| Client disconnect | Upstream socket ended gracefully |
| Upstream disconnect | Client socket ended gracefully |

**See Also:**
- [WebSocket Support](../concepts/websocket-support.md) - Client-side WS resolution
- [Architecture: WebSocket Tunneling](../concepts/ARCHITECTURE.md#websocket-tunneling)

---

## Endpoints

### createConfigEndpoint

Creates `/config` endpoint that returns runtime configuration.

**Signature:**
```typescript
function createConfigEndpoint(options: ConfigEndpointOptions): RequestHandler
```

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `apiPort` | `number` | ✅ | API server port |
| `apiHost` | `string` | ❌ | API host (default: `127.0.0.1`) |
| `wsPort` | `number` | ❌ | WebSocket port (default: `apiPort`) |
| `wsHost` | `string` | ❌ | WebSocket host (default: `apiHost`) |
| `uiPort` | `number` | ✅ | UI server port |
| `serviceName` | `string` | ❌ | Service name |
| `version` | `string` | ❌ | Service version |

**Returns:** `RequestHandler` - Express middleware for `/config` endpoint

**Example:**

```typescript
import { createConfigEndpoint } from '@vrooli/api-base/server'

app.get('/config', createConfigEndpoint({
  apiPort: 8080,
  apiHost: '127.0.0.1',
  wsPort: 8081,
  uiPort: 3000,
  serviceName: 'my-scenario',
  version: '1.0.0',
}))

// GET /config
// Response:
// {
//   "apiUrl": "http://127.0.0.1:8080/api/v1",
//   "wsUrl": "ws://127.0.0.1:8081/ws",
//   "apiPort": "8080",
//   "wsPort": "8081",
//   "uiPort": "3000",
//   "serviceName": "my-scenario",
//   "version": "1.0.0"
// }
```

**Client Usage:**
```typescript
import { fetchRuntimeConfig } from '@vrooli/api-base'

const config = await fetchRuntimeConfig('./config')
console.log(`API: ${config.apiUrl}`)
```

**See Also:**
- [Runtime Configuration](../concepts/runtime-config.md)
- [Client: fetchRuntimeConfig](./client.md#fetchruntimeconfig)

---

### createHealthEndpoint

Creates `/health` endpoint with API connectivity checking.

**Signature:**
```typescript
function createHealthEndpoint(options: HealthOptions): RequestHandler
```

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `serviceName` | `string` | ✅ | Service name |
| `version` | `string` | ❌ | Service version |
| `apiPort` | `number` | ❌ | API port (enables connectivity check) |
| `apiHost` | `string` | ❌ | API host (default: `127.0.0.1`) |
| `timeout` | `number` | ❌ | Health check timeout in ms (default: `5000`) |
| `customHealthCheck` | `() => Promise<Record<string, unknown>>` | ❌ | Custom health checks |

**Returns:** `RequestHandler` - Express middleware for `/health` endpoint

**Response Status Codes:**

| Code | Status | Condition |
|------|--------|-----------|
| `200` | `healthy` | All checks pass |
| `503` | `degraded` | API unreachable but UI functional |
| `503` | `unhealthy` | Critical failure |

**Example:**

```typescript
import { createHealthEndpoint } from '@vrooli/api-base/server'

// Basic health check (no API check)
app.get('/health', createHealthEndpoint({
  serviceName: 'my-scenario-ui',
  version: '1.0.0',
}))

// GET /health
// Response (200):
// {
//   "status": "healthy",
//   "service": "my-scenario-ui",
//   "version": "1.0.0",
//   "timestamp": "2025-01-01T00:00:00.000Z",
//   "readiness": true
// }
```

**With API Connectivity Check:**
```typescript
app.get('/health', createHealthEndpoint({
  serviceName: 'my-scenario-ui',
  version: '1.0.0',
  apiPort: 8080,
  apiHost: '127.0.0.1',
  timeout: 3000,
}))

// GET /health
// Response (503 if API down):
// {
//   "status": "degraded",
//   "service": "my-scenario-ui",
//   "version": "1.0.0",
//   "timestamp": "2025-01-01T00:00:00.000Z",
//   "readiness": true,
//   "api_connectivity": {
//     "connected": false,
//     "api_url": "http://127.0.0.1:8080/health",
//     "last_check": "2025-01-01T00:00:00.000Z",
//     "error": {
//       "code": "CONNECTION_ERROR",
//       "message": "Failed to connect to API: ...",
//       "category": "network",
//       "retryable": true
//     },
//     "latency_ms": null
//   }
// }
```

**With Custom Health Checks:**
```typescript
app.get('/health', createHealthEndpoint({
  serviceName: 'my-scenario-ui',
  customHealthCheck: async () => {
    // Check database
    const dbHealthy = await checkDatabase()

    // Check cache
    const cacheHealthy = await checkCache()

    return {
      database: dbHealthy ? 'connected' : 'disconnected',
      cache: cacheHealthy ? 'available' : 'unavailable',
    }
  },
}))

// Response includes custom fields:
// {
//   "status": "healthy",
//   "service": "my-scenario-ui",
//   "timestamp": "2025-01-01T00:00:00.000Z",
//   "readiness": true,
//   "database": "connected",
//   "cache": "available"
// }
```

**See Also:**
- [createSimpleHealthEndpoint](#createsimplehealthendpoint) (minimal version)

---

### createSimpleHealthEndpoint

Creates minimal `/health` endpoint (no API connectivity check).

**Signature:**
```typescript
function createSimpleHealthEndpoint(
  serviceName: string,
  version?: string
): RequestHandler
```

**Parameters:**

| Name | Type | Description |
|------|------|-------------|
| `serviceName` | `string` | Service name |
| `version` | `string` | Optional service version |

**Returns:** `RequestHandler` - Express middleware for `/health` endpoint

**Example:**

```typescript
import { createSimpleHealthEndpoint } from '@vrooli/api-base/server'

app.get('/health', createSimpleHealthEndpoint('my-scenario-ui', '1.0.0'))

// GET /health
// Response (always 200):
// {
//   "status": "ok",
//   "service": "my-scenario-ui",
//   "version": "1.0.0",
//   "timestamp": "2025-01-01T00:00:00.000Z"
// }
```

**When to Use:**
- ✅ Simple UI-only scenarios
- ✅ Load balancer health checks (just need 200 OK)
- ✅ Minimal overhead
- ❌ Need API connectivity monitoring (use `createHealthEndpoint`)

---

## Types

All TypeScript types and interfaces are documented in the [Types Reference](./types.md).

**Core Types:**
- [`ServerTemplateOptions`](./types.md#servertemplateoptions)
- [`ScenarioProxyHostOptions`](./types.md#scenarioproxyhostoptions)
- [`ScenarioProxyHostController`](./types.md#scenarioproxyhostcontroller)
- [`ScenarioProxyAppMetadata`](./types.md#scenarioproxyappmetadata)
- [`ProxyInfo`](./types.md#proxyinfo)
- [`PortEntry`](./types.md#portentry)
- [`HostEndpointDefinition`](./types.md#hostendpointdefinition)
- [`ScenarioConfig`](./types.md#scenarioconfig)
- [`ProxyOptions`](./types.md#proxyoptions)
- [`HealthOptions`](./types.md#healthoptions)
- [`ConfigEndpointOptions`](./types.md#configendpointoptions)

---

## Complete Example

**Full server implementation:**

```typescript
import path from 'path'
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  // Ports from environment
  uiPort: process.env.UI_PORT || 3000,
  apiPort: process.env.API_PORT || 8080,
  wsPort: process.env.WS_PORT || 8081,

  // Static files
  distDir: path.join(__dirname, '../dist'),

  // Service info
  serviceName: 'my-scenario-ui',
  version: process.env.npm_package_version || '1.0.0',

  // CORS - allow all origins for simplicity
  corsOrigins: '*',

  // Verbose logging in development
  verbose: process.env.NODE_ENV === 'development',

  // Custom configuration
  configBuilder: (env) => ({
    apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
    wsUrl: `ws://localhost:${env.WS_PORT}/ws`,
    apiPort: String(env.API_PORT),
    wsPort: String(env.WS_PORT),
    uiPort: String(env.UI_PORT),
    environment: env.NODE_ENV || 'production',
    features: {
      analytics: env.ENABLE_ANALYTICS === 'true',
      beta: env.ENABLE_BETA === 'true',
    },
  }),

  // Custom routes
  setupRoutes: (app) => {
    // Analytics endpoint
    app.post('/analytics', (req, res) => {
      console.log('Analytics event:', req.body)
      res.status(204).end()
    })

    // Custom API endpoints
    app.get('/api/custom', (req, res) => {
      res.json({ custom: true })
    })
  },
})

// Start server
const port = Number(process.env.UI_PORT) || 3000
app.listen(port, '0.0.0.0', () => {
  console.log(`🚀 Server running on port ${port}`)
  console.log(`📊 Health: http://localhost:${port}/health`)
  console.log(`⚙️  Config: http://localhost:${port}/config`)
  console.log(`🌐 UI: http://localhost:${port}`)
})

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM received, shutting down gracefully')
  process.exit(0)
})
```

---

## See Also

- [Quick Start Guide](../guides/quick-start.md)
- [Client API Reference](./client.md)
- [Server Setup Guide](../examples/basic-scenario.md)
- [Proxy Setup Guide](../guides/host-scenario-pattern.md)
- [Runtime Configuration](../concepts/runtime-config.md)
