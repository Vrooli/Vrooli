# @vrooli/api-base

Universal API connectivity for Vrooli scenarios. Handles API resolution, WebSocket endpoints, runtime configuration, and server-side proxy injection across all deployment contexts (localhost, direct tunnels, proxied/embedded scenarios).

## Features

- 🌍 **Universal**: Works with any domain, any proxy pattern, any deployment
- 🔗 **Smart Resolution**: Automatically detects deployment context (localhost, tunnel, proxy)
- 🔌 **WebSocket Support**: First-class WS/WSS endpoint resolution
- ⚙️ **Runtime Config**: Fetch configuration dynamically in production bundles
- 🖥️ **Server Utilities**: Middleware for proxying, health checks, config endpoints
- 🧪 **Fully Tested**: 156 unit tests with 79% coverage
- 📦 **Zero Dependencies**: No runtime dependencies (Express is peer/optional)
- 🔄 **Backwards Compatible**: Supports existing `__APP_MONITOR_PROXY_INFO__` globals

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

// Automatically resolves API base URL for current context
const API_BASE = resolveApiBase({
  defaultPort: '8080',
  appendSuffix: true  // Adds /api/v1
})

// Build API URLs
const healthUrl = buildApiUrl('/health', { baseUrl: API_BASE })
// → http://127.0.0.1:8080/api/v1/health (localhost)
// → https://example.com/api/v1/health (remote)
// → https://host.com/apps/scenario/proxy/api/v1/health (proxied)

// Make requests
const response = await fetch(healthUrl)
```

### Server-Side (Node.js/Express)

```typescript
import express from 'express'
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT || 3000,
  apiPort: process.env.API_PORT || 8080,
  distDir: './dist',
  serviceName: 'my-scenario',
  version: '1.0.0'
})

app.listen(3000, () => {
  console.log('Scenario UI server running on port 3000')
})
```

This automatically sets up:
- `/health` endpoint with API connectivity checks
- `/config` endpoint with runtime configuration
- `/api/*` proxy to your API server
- Static file serving from `./dist`
- SPA fallback routing

## Deployment Contexts

`@vrooli/api-base` automatically handles three deployment contexts:

### 1. Localhost Development

```
http://localhost:3000
```

- UI serves from `localhost:3000`
- API at `localhost:8080`
- Resolution: `http://127.0.0.1:8080/api/v1`

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

## API Reference

### Client Functions

#### `resolveApiBase(options?): string`

Resolves the API base URL for the current deployment context.

```typescript
import { resolveApiBase } from '@vrooli/api-base'

const apiBase = resolveApiBase({
  explicitUrl: 'https://custom.com/api',  // Override (optional)
  defaultPort: '8080',                     // Localhost fallback port
  apiSuffix: '/api/v1',                    // Path suffix
  appendSuffix: true,                      // Whether to append suffix
  windowObject: customWindow,              // Custom window (for testing)
  proxyGlobalNames: ['__CUSTOM__'],        // Custom proxy globals
  configEndpoint: './config'               // Runtime config endpoint
})
```

**Resolution Priority:**
1. Explicit URL (if provided)
2. Proxy metadata from `window.__VROOLI_PROXY_INFO__`
3. Path-based proxy detection (`/proxy` in pathname)
4. App shell pattern (`/apps/{slug}/proxy`)
5. Remote host origin (if not localhost)
6. Localhost fallback (`http://127.0.0.1:{defaultPort}`)

#### `resolveWsBase(options?): string`

Resolves WebSocket endpoint URL (ws:// or wss://).

```typescript
import { resolveWsBase } from '@vrooli/api-base'

const wsBase = resolveWsBase({
  defaultPort: '8080',
  appendSuffix: true,  // Adds /ws
  apiSuffix: '/ws'
})
// → ws://127.0.0.1:8080/ws (localhost)
// → wss://example.com/ws (remote/https)
```

#### `buildApiUrl(path, options?): string`

Builds full API URL from path and options.

```typescript
import { buildApiUrl } from '@vrooli/api-base'

const url = buildApiUrl('/users/123', {
  baseUrl: 'https://api.example.com',
  appendSuffix: false
})
// → https://api.example.com/users/123
```

#### `isProxyContext(windowObject?, proxyGlobalNames?): boolean`

Detects if running in a proxied context.

```typescript
import { isProxyContext } from '@vrooli/api-base'

if (isProxyContext()) {
  console.log('Running in proxied/embedded mode')
}
```

#### `getProxyInfo(windowObject?, proxyGlobalNames?): ProxyInfo | null`

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
  uiPort: 3000,
  apiPort: 8080,

  // Optional
  apiHost: '127.0.0.1',
  wsPort: 8081,
  distDir: './dist',
  serviceName: 'my-scenario',
  version: '1.0.0',
  corsOrigins: '*',
  verbose: true,

  // Custom config builder
  configBuilder: (env) => ({
    apiUrl: `http://custom:${env.API_PORT}`,
    wsUrl: `ws://custom:${env.WS_PORT}`,
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
  apiPort: 8080,
  timeout: 5000,
  customHealthCheck: async () => ({
    database: { connected: true },
    redis: { connected: false }
  })
}))
```

#### `createConfigEndpoint(options): RequestHandler`

Creates a runtime config endpoint.

```typescript
import { createConfigEndpoint } from '@vrooli/api-base/server'

app.get('/config', createConfigEndpoint({
  apiPort: 8080,
  uiPort: 3000,
  serviceName: 'my-scenario',
  version: '1.0.0',
  cors: true,
  includeTimestamp: true,
  cacheControl: 'no-cache, must-revalidate',

  // Or use custom builder
  configBuilder: () => ({
    apiUrl: 'http://custom.com/api',
    wsUrl: 'ws://custom.com/ws',
    customField: 'value'
  })
}))
```

#### `createProxyMiddleware(options): RequestHandler`

Creates a proxy middleware for API requests.

```typescript
import { createProxyMiddleware } from '@vrooli/api-base/server'

app.use('/api', createProxyMiddleware({
  targetPort: 8080,
  targetHost: 'localhost',
  timeout: 30000,
  verbose: true
}))
```

#### `injectProxyMetadata(html, metadata, options?): string`

Injects proxy metadata into HTML for hosting embedded scenarios.

```typescript
import { injectProxyMetadata, buildProxyMetadata } from '@vrooli/api-base/server'

const metadata = buildProxyMetadata({
  appId: 'embedded-scenario',
  hostScenario: 'app-monitor',
  targetScenario: 'embedded-scenario',
  ports: [
    {
      port: 3000,
      label: 'ui',
      slug: 'ui',
      path: '/apps/embedded/proxy',
      aliases: [],
      isPrimary: true,
      source: 'manual',
      priority: 100
    }
  ],
  primaryPort: /* PortEntry */
})

const modifiedHtml = injectProxyMetadata(html, metadata, {
  patchFetch: true,  // Automatically rewrite fetch() URLs
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

## Testing

Run tests:

```bash
pnpm test              # Run all tests
pnpm test:watch        # Watch mode
pnpm test:coverage     # With coverage report
```

## Examples

### Basic Scenario

**Client (`ui/src/App.tsx`):**

```typescript
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

const API_BASE = resolveApiBase({
  defaultPort: '8080',
  appendSuffix: true
})

async function fetchData() {
  const url = buildApiUrl('/data', { baseUrl: API_BASE })
  const response = await fetch(url)
  return response.json()
}
```

**Server (`ui/server.js`):**

```javascript
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT || 3000,
  apiPort: process.env.API_PORT || 8080,
  distDir: './dist',
  serviceName: 'my-scenario'
})

app.listen(process.env.UI_PORT || 3000)
```

### With WebSockets

```typescript
import { resolveWsBase } from '@vrooli/api-base'

const WS_BASE = resolveWsBase({
  defaultPort: '8080',
  appendSuffix: true
})

const ws = new WebSocket(WS_BASE)
ws.onopen = () => console.log('Connected')
```

### Custom Proxy Globals

```typescript
import { resolveApiBase } from '@vrooli/api-base'

const API_BASE = resolveApiBase({
  proxyGlobalNames: ['__MY_CUSTOM_PROXY__'],
  defaultPort: '8080'
})
```

### Hosting Embedded Scenarios

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

## Migration Guide

### From Custom `config.ts` Implementations

**Before:**

```typescript
// Custom config.ts
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

const API_BASE = resolveApiBase({
  defaultPort: '8080',
  appendSuffix: true
})
```

### From Hardcoded URLs

**Before:**

```typescript
const API_BASE = 'http://localhost:8080/api/v1'
fetch(`${API_BASE}/data`)
```

**After:**

```typescript
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

const API_BASE = resolveApiBase({ defaultPort: '8080', appendSuffix: true })
fetch(buildApiUrl('/data', { baseUrl: API_BASE }))
```

## Contributing

See [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) for development roadmap and architecture details.

## License

MIT
