# Server API Reference

Complete reference for all server-side functions in `@vrooli/api-base/server`.

**Import:**
```typescript
import {
  createScenarioServer,
  startScenarioServer,
  injectProxyMetadata,
  injectScenarioConfig,
  createProxyMiddleware,
  proxyToApi,
  createConfigEndpoint,
  createHealthEndpoint,
  createSimpleHealthEndpoint,
} from '@vrooli/api-base/server'
```

## Table of Contents

- [Server Template](#server-template)
  - [createScenarioServer](#createscenarioserver)
  - [startScenarioServer](#startscenarioserver)
- [Metadata Injection](#metadata-injection)
  - [injectProxyMetadata](#injectproxymetadata)
  - [injectScenarioConfig](#injectscenarioconfig)
  - [buildProxyMetadata](#buildproxymetadata)
- [API Proxying](#api-proxying)
  - [createProxyMiddleware](#createproxymiddleware)
  - [proxyToApi](#proxytoapi)
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

## Metadata Injection

### injectProxyMetadata

Injects proxy metadata into HTML content.

**Signature:**
```typescript
function injectProxyMetadata(html: string, metadata: ProxyInfo): string
```

**Parameters:**

| Name | Type | Description |
|------|------|-------------|
| `html` | `string` | HTML content to modify |
| `metadata` | `ProxyInfo` | Proxy metadata to inject |

**Returns:** `string` - Modified HTML with injected `<script>` tag

**Injection Location:**
Inserts a `<script>` tag into the `<head>` section that sets:
- `window.__VROOLI_PROXY_INFO__`
- `window.__VROOLI_PROXY_INDEX__`

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

**Signature:**
```typescript
async function proxyToApi(
  req: Request,
  res: Response,
  options: ProxyRequestOptions
): Promise<void>
```

**Parameters:**

| Name | Type | Description |
|------|------|-------------|
| `req` | `Request` | Express request object |
| `res` | `Response` | Express response object |
| `options` | `ProxyRequestOptions` | Proxy configuration |

**ProxyRequestOptions:**

| Field | Type | Description |
|-------|------|-------------|
| `targetPort` | `number` | API server port |
| `targetHost` | `string` | API server host (default: `127.0.0.1`) |
| `targetPath` | `string` | API endpoint path (default: `req.url`) |
| `timeout` | `number` | Request timeout in ms |
| `verbose` | `boolean` | Enable logging |

**Example:**

```typescript
import { proxyToApi } from '@vrooli/api-base/server'

app.use('/custom-api', async (req, res) => {
  // Custom logic before proxying
  console.log(`Proxying: ${req.method} ${req.url}`)

  // Proxy to API
  await proxyToApi(req, res, {
    targetPort: 8080,
    targetHost: '127.0.0.1',
    targetPath: `/v2${req.url}`,  // Rewrite path
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
- [`ProxyInfo`](./types.md#proxyinfo)
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
- [Server Setup Guide](../guides/server-usage.md)
- [Proxy Setup Guide](../guides/proxy-setup.md)
- [Runtime Configuration](../concepts/runtime-config.md)
