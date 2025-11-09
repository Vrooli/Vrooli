# @vrooli/api-base

Universal API connectivity for Vrooli scenarios. Handles API resolution, WebSocket endpoints, runtime configuration, and server-side proxy injection across all deployment contexts (localhost, direct tunnels, proxied/embedded scenarios).

## Features

- 🌍 **Universal**: Works with any domain, any deployment
- 🔗 **Smart Resolution**: Automatically detects context (localhost, tunnel, proxy)
- 🔌 **WebSocket Support**: First-class WS/WSS endpoint resolution
- 🖥️ **Server Utilities**: Complete server with health, config, and proxy built-in
- 🧪 **Fully Tested**: 156 unit tests covering all edge cases
- 📦 **Zero Config**: Just `resolveApiBase({ appendSuffix: true })` and you're done
- 🔄 **Backwards Compatible**: Supports existing proxy patterns

## Installation

```bash
pnpm add @vrooli/api-base
```

For server-side features:

```bash
pnpm add @vrooli/api-base express
```

## Quick Start

### Client-Side (Browser)

```typescript
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

// Simple! Just specify if you want the /api/v1 suffix
const API_BASE = resolveApiBase({ appendSuffix: true })

// Build API URLs
const healthUrl = buildApiUrl('/health', { baseUrl: API_BASE })
// → http://localhost:36221/api/v1/health (localhost)
// → https://example.com/api/v1/health (remote)
// → https://host.com/apps/scenario/proxy/api/v1/health (proxied)

// Make requests
const response = await fetch(healthUrl)
```

### Server-Side (Node.js/Express)

```javascript
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'my-scenario',
  version: '1.0.0',
  corsOrigins: '*'  // Allow tunnel/proxy origins
})

app.listen(process.env.UI_PORT)
```

This automatically sets up:
- `/health` endpoint with API connectivity checks
- `/config` endpoint with runtime configuration
- `/api/*` proxy to your API server
- Static file serving from `./dist`
- SPA fallback routing

## The Vrooli Pattern

Vrooli scenarios always serve production bundles, even in local development. This means:

1. **UI is pre-built** (`npm run build` → `dist/` directory)
2. **Server serves static files** from `dist/`
3. **Server proxies API requests** from `/api/*` to API server
4. **No dev servers** like `vite dev` (too slow, cache issues)

This is why `@vrooli/api-base` defaults to using `window.location.origin`—every request should travel through the UI server's proxy. Even on localhost the resolver now returns the UI origin (for example `http://localhost:3000`) so that your browser, Cloudflare tunnel, or future VPS/Kubernetes ingress all behave the same. Never hard-code `API_PORT` into client code; let `resolveApiBase`/`resolveWsBase` decide so the single tunnel keeps working.

> **Note on WebSockets:** In proxied contexts `resolveWsBase` may return an origin-relative path such as `/apps/scenario/proxy`. Pass it directly to `new WebSocket(...)` or your WebSocket client—the browser will automatically prefix the current host and keep the socket inside the proxy path.

### Why Production Bundles?

- **Fast**: No dev server overhead, instant page loads
- **Reliable**: What you test is what users get
- **Cache-safe**: Vite hashes assets, no stale bundles
- **Universal**: Same pattern for local, tunnel, and deployed scenarios

### The Request Flow

```
Browser                  UI Server               API Server
   │                        │                       │
   │  GET /                 │                       │
   │ ──────────────────────>│                       │
   │  (serve dist/index.html)                       │
   │ <──────────────────────│                       │
   │                        │                       │
   │  GET /api/v1/issues    │                       │
   │ ──────────────────────>│  GET /api/v1/issues   │
   │                        │ ─────────────────────>│
   │                        │ <─────────────────────│
   │ <──────────────────────│                       │
```

Because the browser and API requests use the same origin (localhost:36221), there are no CORS issues.

## Deployment Contexts

`@vrooli/api-base` automatically handles three deployment contexts:

### 1. Localhost Development

```
http://localhost:36221
```

- UI serves from `localhost:36221`
- API at `localhost:19750`
- Resolution: `http://localhost:36221/api/v1` (proxied to API)

### 2. Direct Tunnel (Cloudflare/ngrok)

```
https://my-scenario.example.com
```

- Scenario has its own domain
- Resolution: `https://my-scenario.example.com/api/v1`

### 3. Proxied/Embedded (App-Monitor)

```
https://app-monitor.example.com/apps/my-scenario/proxy/
```

- Scenario embedded in another app
- Resolution: `https://app-monitor.example.com/apps/my-scenario/proxy/api/v1`
- Proxy metadata injected via `window.__VROOLI_PROXY_INFO__`

## Complete Example

This is the standard pattern for all Vrooli scenarios.

**Vite Config (`ui/vite.config.ts`):**

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  base: './',  // Required for tunnel/proxy contexts
  plugins: [react()],
})
```

**Frontend (`ui/src/App.tsx`):**

```typescript
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

// Simple! Works in all three deployment contexts automatically
const API_BASE = resolveApiBase({ appendSuffix: true })

async function fetchData() {
  const url = buildApiUrl('/data', { baseUrl: API_BASE })
  const response = await fetch(url)
  return response.json()
}
```

**Backend (`ui/server.js`):**

```javascript
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'my-scenario',
  corsOrigins: '*'  // Allow tunnel/proxy origins
})

app.listen(process.env.UI_PORT)
```

**That's it!** This works in all three deployment contexts automatically.

## API Reference

### Client Functions

#### `resolveApiBase(options?): string`

Resolves the API base URL for the current deployment context.

```typescript
import { resolveApiBase } from '@vrooli/api-base'

// Standard usage (recommended for all Vrooli scenarios)
const apiBase = resolveApiBase({ appendSuffix: true })
```

**Common Options:**
- `appendSuffix?: boolean` - Whether to append `/api/v1` (default: `false`)
- `apiSuffix?: string` - Custom suffix (default: `'/api/v1'`)

**Advanced Options (rarely needed):**
- `explicitUrl?: string` - Override URL (for special deployment scenarios)
- `windowObject?: WindowLike` - Custom window object (for testing only)
- `proxyGlobalNames?: string[]` - Custom proxy global names (advanced)

**Resolution Priority:**
1. Explicit URL (if provided)
2. Proxy metadata from `window.__VROOLI_PROXY_INFO__`
3. Path-based proxy detection (`/proxy` in pathname)
4. App shell pattern (`/apps/{slug}/proxy`)
5. Remote host origin (if not localhost)
6. Localhost origin (default for production bundles)

#### `resolveWsBase(options?): string`

Resolves WebSocket endpoint URL (ws:// or wss://).

```typescript
import { resolveWsBase } from '@vrooli/api-base'

const wsBase = resolveWsBase({ appendSuffix: true, apiSuffix: '/ws' })
// → ws://localhost:36221/ws (localhost)
// → wss://example.com/ws (remote/https)
```

#### `buildApiUrl(path, options?): string`

Builds full API URL from path and base URL.

```typescript
import { buildApiUrl } from '@vrooli/api-base'

const url = buildApiUrl('/users/123', { baseUrl: API_BASE })
```

#### `isProxyContext(options?): boolean`

Detects if running in a proxied context.

```typescript
import { isProxyContext } from '@vrooli/api-base'

if (isProxyContext()) {
  console.log('Running in proxied/embedded mode')
}
```

#### `getProxyInfo(options?): ProxyInfo | null`

Retrieves proxy metadata if available.

```typescript
import { getProxyInfo } from '@vrooli/api-base'

const proxyInfo = getProxyInfo()
if (proxyInfo) {
  console.log('Hosted by:', proxyInfo.hostScenario)
  console.log('Proxy path:', proxyInfo.primary.path)
}
```

### Server Functions

#### `createScenarioServer(options): Express.Application`

Creates a fully-configured Express server for a scenario.

```typescript
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  // Required
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,

  // Optional
  apiHost: '127.0.0.1',
  wsPort: process.env.WS_PORT,
  distDir: './dist',
  serviceName: 'my-scenario',
  version: '1.0.0',
  corsOrigins: '*',
  verbose: false,
  proxyTimeoutMs: 60000, // Optional: extend API proxy timeout for long-running requests

  // Custom config builder
  configBuilder: (env) => ({
    apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
    wsUrl: `ws://localhost:${env.WS_PORT}/ws`,
    customField: 'value'
  }),

  // Custom routes
  setupRoutes: (app) => {
    app.get('/custom', (req, res) => res.json({ ok: true }))
  },

  // Inject metadata (for hosting other scenarios)
  proxyMetadata: { /* ProxyInfo */ },
  scenarioConfig: { /* ScenarioConfig */ }
})
```

#### `createHealthEndpoint(options): RequestHandler`

Creates a health check endpoint.

```typescript
import { createHealthEndpoint } from '@vrooli/api-base/server'

app.get('/health', createHealthEndpoint({
  serviceName: 'my-scenario',
  version: '1.0.0',
  apiPort: process.env.API_PORT,
  timeout: 5000,
  customHealthCheck: async () => ({
    database: { connected: true }
  })
}))
```

#### `createConfigEndpoint(options): RequestHandler`

Creates a runtime config endpoint.

```typescript
import { createConfigEndpoint } from '@vrooli/api-base/server'

app.get('/config', createConfigEndpoint({
  apiPort: process.env.API_PORT,
  uiPort: process.env.UI_PORT,
  serviceName: 'my-scenario',
  version: '1.0.0'
}))
```

#### `createProxyMiddleware(options): RequestHandler`

Creates a proxy middleware for API requests.

```typescript
import { createProxyMiddleware } from '@vrooli/api-base/server'

app.use('/api', createProxyMiddleware({
  targetPort: process.env.API_PORT,
  targetHost: 'localhost',
  timeout: 30000,
  verbose: false
}))
```

#### `createScenarioProxyHost(options): ScenarioProxyHostController`

Complete proxy stack for host scenarios like **app-monitor** that need to embed
other scenarios via `/apps/:id/proxy/*`.

```typescript
import { createScenarioProxyHost } from '@vrooli/api-base/server'
import axios from 'axios'

const proxyHost = createScenarioProxyHost({
  hostScenario: 'app-monitor',
  cacheTtlMs: 30_000,
  fetchAppMetadata: async (appId) => {
    const response = await axios.get(`${API_BASE}/api/v1/apps/${encodeURIComponent(appId)}`)
    return response.data?.data
  },
})

app.use(proxyHost.router)

server.on('upgrade', async (req, socket, head) => {
  if (await proxyHost.handleUpgrade(req, socket, head)) {
    return
  }
  // Host-specific WebSocket handling here
})
```

Features:

- App metadata loading + caching
- Automatic UI/API port detection
- `/apps/:id/proxy/*` and `/apps/:id/ports/:portKey/proxy/*` routes
- HTML injection (proxy metadata + `<base>` tag) with optional `patchFetch`
- Asset + API proxying, including custom port aliases
- Built-in WebSocket upgrade handling for proxied apps

Common options:

- `appsPathPrefix` (default: `/apps`)
- `proxyPathSegment` (default: `proxy`)
- `portsPathSegment` (default: `ports`)
- `cacheTtlMs` (default: 30000)
- `loopbackHosts` (defaults to standard localhost values)
- `patchFetch` (default: `false`)
- `childBaseTagAttribute` (default: `data-proxy-host`)
- `proxiedAppHeader` (default: `x-vrooli-proxied-app`)

The returned controller exposes `router`, `handleUpgrade`, `invalidate(appId?)`,
and `clearCache()` so hosts can manage state explicitly.

#### `injectProxyMetadata(html, metadata, options?): string`

Injects proxy metadata into HTML for hosting embedded scenarios.

```typescript
import { injectProxyMetadata, buildProxyMetadata } from '@vrooli/api-base/server'

const metadata = buildProxyMetadata({
  appId: 'embedded-scenario',
  hostScenario: 'app-monitor',
  targetScenario: 'embedded-scenario',
  ports: [{
    port: 3000,
    label: 'ui',
    slug: 'ui',
    path: '/apps/embedded/proxy',
    aliases: [],
    isPrimary: true,
    source: 'manual',
    priority: 100
  }],
  primaryPort: /* PortEntry */
})

const modifiedHtml = injectProxyMetadata(html, metadata, {
  patchFetch: true,
  infoGlobalName: '__VROOLI_PROXY_INFO__',
  indexGlobalName: '__VROOLI_PROXY_INDEX__'
})
```

## TypeScript

Full TypeScript support with comprehensive type definitions.

```typescript
import type {
  ProxyInfo,
  PortEntry,
  ScenarioConfig,
  ResolveOptions,
  ServerTemplateOptions,
  HealthOptions,
  ConfigEndpointOptions
} from '@vrooli/api-base/types'
```

## Troubleshooting

### CORS Errors on Localhost

```
Cross-Origin Request Blocked: The Same Origin Policy disallows reading
the remote resource at http://127.0.0.1:8080/api/v1/...
```

**Cause:** You're trying to connect directly to the API server instead of going through the proxy.

**Fix:** Make sure you're using the simple pattern without `defaultPort`:

```typescript
// ✅ Correct
const API_BASE = resolveApiBase({ appendSuffix: true })

// ❌ Wrong - causes CORS errors
const API_BASE = resolveApiBase({
  defaultPort: '8080',  // Don't use this!
  appendSuffix: true
})
```

The library automatically uses `window.location.origin` on localhost, which routes requests through your UI server's proxy.

### API Requests Return 404

**Cause:** Server not proxying `/api/*` requests, or UI not built.

**Check:**
1. Are you using `createScenarioServer`? (You should be)
2. Is `ui/dist/` populated? Run `npm run build` in ui directory
3. Is server.js running? Check logs with `vrooli scenario logs <name> --step start-ui`

### Production Bundle Not Updating

**Cause:** Stale build artifacts.

**Fix:**
```bash
cd ui
rm -rf dist
npm run build
vrooli scenario stop <name>
vrooli scenario start <name>
```

## Migration Guide

### From Custom server.js

**Before (190 lines of custom proxy code):**

```javascript
import express from 'express'
import http from 'http'
import httpProxy from 'http-proxy'

const app = express()
const apiProxy = httpProxy.createProxyServer({ /* ... */ })

function proxyToApi(req, res, upstreamPath) {
  // ... lots of custom proxy logic
}

app.use('/api', (req, res) => proxyToApi(req, res))
app.use(express.static('dist'))
app.get('*', (req, res) => res.sendFile('dist/index.html'))
// ... more setup
```

**After (18 lines):**

```javascript
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'my-scenario'
})

app.listen(process.env.UI_PORT)
```

### From Custom API Resolution

**Before:**

```typescript
// Custom buildApiUrl implementation
function buildApiUrl(baseUrl: string, path: string): string {
  const normalizedBase = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl
  return `${normalizedBase}${path}`
}

const API_BASE = "http://localhost:8080/api/v1"  // Hardcoded!
```

**After:**

```typescript
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

const API_BASE = resolveApiBase({ appendSuffix: true })
// That's it! Works everywhere automatically
```

### From Custom Config Fetching

**Before:**

```typescript
async function getConfig() {
  const response = await fetch('/config')
  return response.json()
}

const config = await getConfig()
const API_BASE = config.apiUrl
```

**After:**

```typescript
import { resolveApiBase } from '@vrooli/api-base'

const API_BASE = resolveApiBase({ appendSuffix: true })
// No async needed - works synchronously
```

## Advanced Usage

### Custom Proxy Globals

If you're hosting scenarios and want to use custom global variable names:

```typescript
import { resolveApiBase } from '@vrooli/api-base'

const API_BASE = resolveApiBase({
  proxyGlobalNames: ['__MY_CUSTOM_PROXY__'],
  appendSuffix: true
})
```

### Hosting Embedded Scenarios

If your scenario hosts other scenarios in iframes:

```typescript
import express from 'express'
import { injectProxyMetadata, buildProxyMetadata } from '@vrooli/api-base/server'

app.get('/embed/:scenario', async (req, res) => {
  const scenarioHtml = await fetchScenarioHtml(req.params.scenario)

  const metadata = buildProxyMetadata({
    appId: req.params.scenario,
    hostScenario: 'my-host',
    targetScenario: req.params.scenario,
    ports: [/* ... */],
    primaryPort: /* ... */
  })

  const modifiedHtml = injectProxyMetadata(scenarioHtml, metadata, {
    patchFetch: true
  })

  res.send(modifiedHtml)
})
```

### WebSocket Support

```typescript
import { resolveWsBase } from '@vrooli/api-base'

const WS_BASE = resolveWsBase({ appendSuffix: true, apiSuffix: '/ws' })

const ws = new WebSocket(WS_BASE)
ws.onopen = () => console.log('Connected')
ws.onmessage = (event) => console.log('Message:', event.data)
```

## Testing

Run tests:

```bash
pnpm test              # Run all tests
pnpm test:watch        # Watch mode
pnpm test:coverage     # With coverage report
```

## Contributing

See [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for development roadmap and architecture details.

## License

MIT
