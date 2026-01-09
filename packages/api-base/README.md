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
- ⚡ **Proxy Host Cache**: Built-in HTML caching for embedded scenarios (tunable via `cacheProxyHtml`)

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

// The browser requests: http://localhost:36221/api/v1/health
// The UI server proxies to: http://localhost:19750/api/v1/health (your API)
```

### Server-Side (Node.js/Express)

> ⚠️ **Important:** Your server file must use ESM imports (not CommonJS `require()`). Ensure your `package.json` includes `"type": "module"`.

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

> 🔄 **Fast proxying by default:** `createScenarioServer` now reuses HTTP connections between the UI and API, so proxied requests feel as fast as direct calls. Pass `proxyKeepAlive: false` (or provide a custom `proxyAgent`) to opt out when working with servers that don't support keep-alive.

> 🌀 **Streaming-first proxy:** `/api` is mounted before any body parsing so uploads and SSE stay streamable. UI-owned routes still get JSON parsing by default, and you can customize it with `bodyParser`:
>
> ```ts
> createScenarioServer({
>   bodyParser: (app) => {
>     app.use(express.json({ limit: '25mb' }))
>     app.use(express.urlencoded({ extended: true }))
>   }
> })
> ```
>
> Pass `bodyParser: false` if your UI server is entirely static.

> 🗃️ **Cached SPA fallback:** `dist/index.html` is cached in memory (and auto-invalidates when the file changes) so proxied page loads skip redundant disk reads. Set `cacheIndexHtml: false` if you need to regenerate HTML every request.

This automatically sets up:
- `/health` endpoint with API connectivity checks
- `/config` endpoint with runtime configuration
- `/api/*` proxy to your API server
- Static file serving from `./dist` (hashed assets served with immutable caching, SPA shell stays no-store)
- SPA fallback routing

## Propagating Changes to Scenarios

After editing `@vrooli/api-base`, run the refresh helper so scenarios rebuild with the updated bundle:

```bash
./scripts/scenarios/tools/refresh-shared-package.sh api-base <scenario|all> [--no-restart]
```

The script rebuilds this package, finds every targeted scenario that actually depends on `@vrooli/api-base`, runs `vrooli scenario setup`, and automatically restarts only the scenarios that were already running (unless you pass `--no-restart`). Stopped scenarios stay stopped after setup.

## API Server Requirements

Your API server (the backend that the UI proxies to) should expose all client-facing endpoints under `/api/v1/*`:

**Go Example:**
```go
// ✅ Correct - all endpoints under /api/v1
router.HandleFunc("/api/v1/health", handleHealth)
router.HandleFunc("/api/v1/users", handleUsers)
router.HandleFunc("/api/v1/data", handleData)

// ❌ Wrong - mixing root and /api/v1
router.HandleFunc("/health", handleHealth)        // Root level
router.HandleFunc("/api/v1/users", handleUsers)   // Versioned
```

**Node/Express Example:**
```javascript
// ✅ Correct - all endpoints under /api/v1
app.get('/api/v1/health', handleHealth)
app.get('/api/v1/users', handleUsers)
app.get('/api/v1/data', handleData)

// ❌ Wrong - mixing root and /api/v1
app.get('/health', handleHealth)        // Root level
app.get('/api/v1/users', handleUsers)   // Versioned
```

**Exception:** If your infrastructure needs direct API health checks (bypassing the UI proxy), you can expose `/health` at *both* root and `/api/v1`:

```go
// Health at both locations
router.HandleFunc("/health", handleHealth)        // For direct infrastructure checks
router.HandleFunc("/api/v1/health", handleHealth) // For client requests (proxied)
```

This ensures:
- Clients using `resolveApiBase({ appendSuffix: true })` can reach `/api/v1/health`
- Infrastructure tools can check API health directly at `http://API_HOST:API_PORT/health`

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

**API Server (`api/main.go`):**

```go
package main

import (
    "github.com/gorilla/mux"
    "net/http"
)

func main() {
    router := mux.NewRouter()

    // All endpoints under /api/v1
    router.HandleFunc("/api/v1/health", handleHealth).Methods("GET")
    router.HandleFunc("/api/v1/data", handleData).Methods("GET")

    http.ListenAndServe(":"+os.Getenv("API_PORT"), router)
}
```

**Or API Server (`api/server.js`):**

```javascript
import express from 'express'

const app = express()

// All endpoints under /api/v1
app.get('/api/v1/health', handleHealth)
app.get('/api/v1/data', handleData)

app.listen(process.env.API_PORT)
```

**Package Configuration (`ui/package.json`):**

```json
{
  "name": "my-scenario-ui",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "build": "vite build",
    "preview": "node server.js"
  },
  "dependencies": {
    "@vrooli/api-base": "workspace:*",
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^4.2.1",
    "express": "^4.18.2",
    "vite": "^5.0.8"
  }
}
```

> **Note:** The `"type": "module"` field is required for ESM imports in `server.js`.

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

### ERR_PACKAGE_PATH_NOT_EXPORTED

```
Error [ERR_PACKAGE_PATH_NOT_EXPORTED]: Package subpath './server' is not defined by "exports"
```

**Cause:** You're using CommonJS `require()` instead of ESM `import`, or your server file is named `.cjs`.

**Fix:**
1. Ensure your `package.json` has `"type": "module"`
2. Rename `server.cjs` to `server.js`
3. Change `const { createScenarioServer } = require('@vrooli/api-base/server')` to `import { createScenarioServer } from '@vrooli/api-base/server'`
4. Update your `.vrooli/service.json` develop step: `"run": "cd ui && node server.js"`
5. Update your `.vrooli/service.json` stop step: `"run": "pkill -f 'node server.js' || true"`

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

**Cause 1:** Server not proxying `/api/*` requests, or UI not built.

**Check:**
1. Are you using `createScenarioServer`? (You should be)
2. Is `ui/dist/` populated? Run `npm run build` in ui directory
3. Is server.js running? Check logs with `vrooli scenario logs <name> --step start-ui`

**Cause 2:** Your API has `/health` at root level instead of `/api/v1/health`.

**Check:**
```bash
curl http://localhost:${API_PORT}/health        # ✅ Works
curl http://localhost:${API_PORT}/api/v1/health # ❌ 404
```

**Fix:** Move your health endpoint under `/api/v1`:
- Go: `router.HandleFunc("/api/v1/health", handleHealth)`
- Express: `app.get('/api/v1/health', handleHealth)`

Or expose it at both locations if infrastructure checks need direct access (see [API Server Requirements](#api-server-requirements)).

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

### Migration Checklist

When migrating an existing scenario to `@vrooli/api-base`, follow these steps:

**Server Migration:**
- [ ] Rename `server.cjs` to `server.js` (if applicable)
- [ ] Add `"type": "module"` to `ui/package.json`
- [ ] Change `require()` statements to `import` statements in `server.js`
- [ ] Replace custom server code with `createScenarioServer()` or `startScenarioServer()`
- [ ] Update `.vrooli/service.json` develop step: `"run": "cd ui && node server.js"`
- [ ] Update `.vrooli/service.json` stop step: `"run": "pkill -f 'node server.js' || true"`

**Client Migration:**
- [ ] Replace custom `buildApiUrl()` with `import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'`
- [ ] Remove `explicitUrl`, `defaultPort`, or other custom API resolution logic
- [ ] Use simple pattern: `const API_BASE = resolveApiBase({ appendSuffix: true })`

**Vite Config Migration:**
- [ ] Simplify `vite.config.ts` to minimal config with `base: './'`
- [ ] Remove `VITE_API_BASE_URL` validation and environment loading
- [ ] Remove port checking and env-based config (api-base handles this)

**Build Process Migration:**
- [ ] Remove `VITE_API_BASE_URL` from build commands in `.vrooli/service.json`
- [ ] Change build step from `VITE_API_BASE_URL="..." pnpm run build` to just `pnpm run build`

**API Server Migration:**
- [ ] Audit API endpoints - ensure all client-facing endpoints are under `/api/v1/*`
- [ ] Move `/health` to `/api/v1/health` (or expose at both locations - see [API Server Requirements](#api-server-requirements))
- [ ] Test proxied endpoints: `curl http://localhost:${UI_PORT}/api/v1/health`

After migration, test all three deployment contexts: localhost, direct tunnel, and app-monitor proxy.

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

### From Complex Vite Config

**Before (with environment validation and port checking):**

```typescript
import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd());

  // Allow test mode to skip port validation
  if (mode === 'test') {
    return {
      plugins: [react()],
      base: './',
      test: { /* ... */ }
    };
  }

  const uiPort = process.env.UI_PORT;
  if (!uiPort) {
    throw new Error("UI_PORT environment variable is required");
  }

  const rawApiBase = env.VITE_API_BASE_URL || process.env.VITE_API_BASE_URL;
  if (!rawApiBase) {
    throw new Error("VITE_API_BASE_URL must be defined");
  }

  return {
    plugins: [react()],
    base: './',
    server: {
      host: true,
      port: Number(uiPort)
    },
    test: { /* ... */ }
  };
});
```

**After (simple and clean):**

```typescript
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  base: './',  // Required for tunnel/proxy contexts
  plugins: [react()],
  test: { /* ... */ }
});
```

**Why You Can Remove VITE_API_BASE_URL:**

The `@vrooli/api-base` library uses `window.location.origin` automatically in production bundles, so the API base URL is determined at runtime, not build time. This means:
- No need to pass `VITE_API_BASE_URL` during build
- No need for environment-specific builds
- Same bundle works in all deployment contexts (localhost, tunnel, proxy)

**Build Command Simplification:**

```bash
# Before
VITE_API_BASE_URL="http://localhost:${API_PORT}/api/v1" pnpm run build

# After
pnpm run build
```

Update your `.vrooli/service.json` build step accordingly:

```json
{
  "name": "build-ui",
  "run": "cd ui && pnpm run build",
  "description": "Build production UI bundle"
}
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
