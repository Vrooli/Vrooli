# Quick Start Guide

Get `@vrooli/api-base` running in your scenario in **5 minutes**.

## What You'll Build

A scenario with:
- ✅ Client-side API resolution that works everywhere
- ✅ Server that proxies API requests
- ✅ Health check endpoint
- ✅ Runtime configuration endpoint
- ✅ Works in localhost, tunnels, and when embedded

## Prerequisites

- Node.js 20+
- A Vrooli scenario with UI and API components
- Basic familiarity with Express and React/TypeScript

## Step 1: Install Package

```bash
cd your-scenario/ui
pnpm add @vrooli/api-base express
```

## Step 2: Update Client Code

**Before (hardcoded or custom logic):**

```typescript
// ❌ Old way - hardcoded
const API_BASE = 'http://localhost:8080/api/v1'
fetch(`${API_BASE}/data`)
```

**After (@vrooli/api-base):**

```typescript
// ✅ New way - universal
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

// Resolve once at app initialization
const API_BASE = resolveApiBase({
  appendSuffix: true        // Adds /api/v1
})

// Use it for all requests
fetch(buildApiUrl('/data', { baseUrl: API_BASE }))
```

**Complete React example:**

```typescript
// src/App.tsx
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'
import { useState, useEffect } from 'react'

// Resolve API base once
const API_BASE = resolveApiBase({
  appendSuffix: true
})

export function App() {
  const [data, setData] = useState(null)

  useEffect(() => {
    const fetchData = async () => {
      const url = buildApiUrl('/data', { baseUrl: API_BASE })
      const response = await fetch(url)
      setData(await response.json())
    }
    fetchData()
  }, [])

  return <div>{/* Your UI */}</div>
}
```

## Step 3: Configure Vite (Required)

**For Vite-based UIs**, add `base: './'` to ensure assets work through tunnels:

```typescript
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  base: './',  // ✅ Required for universal deployment
  plugins: [react()],
  // ... rest of config
})
```

**Why?** Without this, Vite generates absolute asset paths (`/assets/index.js`) which fail when accessed through tunnels or proxied contexts. Relative paths (`./assets/index.js`) work everywhere.

## Step 4: Create Server (Express)

Replace your custom `server.js` with the template:

**Before (custom server):**

```javascript
// ❌ Old way - manual setup
const express = require('express')
const app = express()

app.use(express.static('./dist'))
app.listen(3000)
```

**After (@vrooli/api-base/server):**

```javascript
// ✅ New way - full-featured template
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT || 3000,
  apiPort: process.env.API_PORT || 8080,
  distDir: './dist',
  serviceName: 'my-scenario',
  version: '1.0.0',
  corsOrigins: '*',  // Allow tunnel/proxy origins
  verbose: true       // Logs proxy requests
})

const PORT = process.env.UI_PORT || 3000
app.listen(PORT, () => {
  console.log(`UI server running on port ${PORT}`)
})
```

> 🌀 **Streaming-first proxy:** `/api` is now proxied before any body parsing so uploads stay streamable. UI-owned routes still get JSON parsing by default, and you can customize it with the new `bodyParser` option (pass `false` to disable or a function that mounts your own parsers).

This automatically sets up:
- Static file serving from `./dist`
- `/api/*` proxy to your API server
- `/health` endpoint with API connectivity check
- `/config` endpoint with runtime configuration
- CORS handling (set to `'*'` to allow all origins including tunnels)
- SPA fallback routing

## Step 5: Test All Contexts

### Localhost

```bash
# Start your scenario
vrooli scenario start my-scenario

# Test UI
curl http://localhost:3000/health

# Test API proxy
curl http://localhost:3000/api/health
```

### Direct Tunnel (via Cloudflare)

```bash
# Access via tunnel
https://my-scenario.itsagitime.com/health
https://my-scenario.itsagitime.com/api/health
```

### Proxied (via App-Monitor)

```bash
# Access via app-monitor
https://app-monitor.itsagitime.com/apps/my-scenario/proxy/
```

All three should work identically! ✨

## Step 6: Add WebSocket Support (Optional)

### Client-Side

If your scenario uses WebSockets:

```typescript
import { resolveWsBase } from '@vrooli/api-base'

const WS_BASE = resolveWsBase({
  appendSuffix: true,
  apiSuffix: '/ws'  // Custom WebSocket path
})

const ws = new WebSocket(WS_BASE)
ws.onopen = () => console.log('Connected!')
```

### Server-Side (Automatic)

The easiest way is to let `createScenarioServer` handle WebSocket proxying automatically:

```javascript
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'my-scenario',

  // Enable automatic WebSocket proxying
  wsPathPrefix: '/ws',  // Proxy all /ws/* requests

  // Optional: Transform WebSocket paths
  // Default transforms /ws/* to /api/v1/*
  wsPathTransform: (path) => path.replace(/^\/ws/, '/api/v1')
})

// IMPORTANT: Use startScenarioServer OR create HTTP server manually
// Option 1: Use startScenarioServer (handles everything)
import { startScenarioServer } from '@vrooli/api-base/server'
startScenarioServer({
  // ... options including wsPathPrefix
})

// Option 2: Create HTTP server manually
import http from 'http'
const server = http.createServer(app)

// Attach WebSocket upgrade handler if configured
if (app.__wsUpgradeHandler) {
  server.on('upgrade', app.__wsUpgradeHandler)
}

server.listen(process.env.UI_PORT)
```

### Server-Side (Manual)

If you need more control, use `proxyWebSocketUpgrade` directly:

```javascript
import http from 'http'
import { createScenarioServer, proxyWebSocketUpgrade } from '@vrooli/api-base/server'

const app = createScenarioServer({
  // ... options
})

const server = http.createServer(app)

server.on('upgrade', (req, socket, head) => {
  if (req.url?.startsWith('/ws/')) {
    // Transform path
    const apiPath = req.url.replace(/^\/ws/, '/api/v1')

    // Add custom headers
    const modifiedReq = Object.create(req)
    modifiedReq.url = apiPath
    modifiedReq.headers = {
      ...req.headers,
      'x-custom-header': 'value'
    }

    proxyWebSocketUpgrade(modifiedReq, socket, head, {
      apiPort: process.env.API_PORT,
      verbose: true
    })
  } else {
    socket.destroy()
  }
})

server.listen(process.env.UI_PORT)
```

## What Just Happened?

### Client-Side

`resolveApiBase()` automatically:
1. Checks for proxy metadata (`window.__VROOLI_PROXY_INFO__`)
2. Detects `/proxy` in the URL path
3. Uses `window.location.origin` if on remote host
4. Falls back to `window.location.origin` (production bundles)
5. SSR fallback: `http://127.0.0.1:{defaultPort}` (rarely used)

### Server-Side

`createScenarioServer()` automatically:
1. Serves static files from dist/
2. Proxies `/api/*` requests to your API server
3. Provides `/health` with API connectivity checks
4. Provides `/config` with runtime environment info
5. Handles CORS for localhost origins
6. Sets up SPA fallback routing

## Common Customizations

### Custom API Suffix

```typescript
const API_BASE = resolveApiBase({
  apiSuffix: '/api/v2',  // Custom suffix
  appendSuffix: true
})
```

### Custom Config Builder

```javascript
const app = createScenarioServer({
  uiPort: 3000,
  apiPort: 8080,
  distDir: './dist',
  configBuilder: (env) => ({
    apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
    wsUrl: `ws://localhost:${env.WS_PORT}/ws`,
    customField: 'my-value',
    features: {
      darkMode: true,
      analytics: false
    }
  })
})
```

### Custom Routes

```javascript
const app = createScenarioServer({
  uiPort: 3000,
  apiPort: 8080,
  distDir: './dist',
  setupRoutes: (app) => {
    app.get('/custom', (req, res) => {
      res.json({ custom: 'data' })
    })
  }
})
```

### Custom Proxy Headers

For APIs that require authentication or forwarding headers:

```javascript
const app = createScenarioServer({
  uiPort: 3000,
  apiPort: 8080,
  distDir: './dist',

  // Static headers
  proxyHeaders: {
    'x-custom-header': 'value'
  },

  // OR dynamic headers (built per-request)
  proxyHeaders: (req) => ({
    'x-forwarded-for': req.socket.remoteAddress || '127.0.0.1',
    'x-forwarded-proto': req.socket.encrypted ? 'https' : 'http',
    'x-forwarded-host': req.headers['host'] || ''
  })
})
```

These headers are automatically added to:
- All HTTP API proxy requests (`/api/*`)
- All WebSocket upgrade requests (if `wsPathPrefix` is configured)

### Disable Proxy Keep-Alive (rare)

The UI server now reuses HTTP connections to your API for faster proxied calls. If you're working with an upstream that can't handle keep-alive (or you're debugging socket churn), turn it off explicitly:

```javascript
const app = createScenarioServer({
  uiPort: 3000,
  apiPort: 8080,
  distDir: './dist',
  proxyKeepAlive: false,  // one request per connection
})
```

Prefer customizing the agent instead of disabling it altogether:

```javascript
import http from 'node:http'

const app = createScenarioServer({
  uiPort: 3000,
  apiPort: 8080,
  distDir: './dist',
  proxyAgent: new http.Agent({ keepAlive: true, maxSockets: 20 }),
})
```

You get the same knobs when hosting other scenarios via `createScenarioProxyHost` (`proxyKeepAlive` + `proxyAgent`).

### Verbose Logging

```javascript
const app = createScenarioServer({
  uiPort: 3000,
  apiPort: 8080,
  distDir: './dist',
  verbose: true  // Logs all proxy requests
})
```

## Troubleshooting

### Assets return 403 or 404 in tunnel

**Problem**: CSS/JS files fail to load when accessing through tunnels (e.g., `https://my-scenario.itsagitime.com`)

**Symptoms**:
```
GET https://my-scenario.itsagitime.com/assets/index-abc123.css 403 (Forbidden)
GET https://my-scenario.itsagitime.com/assets/index-xyz789.js 403 (Forbidden)
```

**Solution**: Add `base: './'` to your `vite.config.ts`:

```typescript
export default defineConfig({
  base: './',  // ✅ This fixes it
  plugins: [react()],
})
```

Then rebuild: `pnpm build`

### CORS errors in tunnel/proxy contexts

**Problem**: Requests blocked with "Origin not allowed" errors

**Symptoms**:
```
[cors] Blocked origin: https://my-scenario.itsagitime.com
```

**Solution**: Set `corsOrigins: '*'` in your server:

```javascript
const app = createScenarioServer({
  corsOrigins: '*',  // ✅ Allow all origins
  // ... other config
})
```

### API requests return 502

**Problem**: UI can't reach API

**Solution**: Check that `API_PORT` environment variable is set correctly:

```javascript
console.log('API_PORT:', process.env.API_PORT)
```

### Works locally but fails in tunnel

**Problem**: Proxy detection not working

**Solution**: Check browser console for proxy metadata:

```javascript
console.log(window.__VROOLI_PROXY_INFO__)
```

### TypeScript errors

**Problem**: Missing type definitions

**Solution**: Import types explicitly:

```typescript
import type { ResolveOptions } from '@vrooli/api-base/types'
```

## Next Steps

- **[Client Usage Guide](../api/client.md)** - Complete client-side patterns
- **[Server Usage Guide](../api/server.md)** - Advanced server configuration
- **[Deployment Contexts](../concepts/deployment-contexts.md)** - Understanding how resolution works
- **[Testing Guide](../../TESTING.md)** - Testing in all contexts

## Summary

You've successfully integrated `@vrooli/api-base`! Your scenario now:
- ✅ Works in localhost, direct tunnels, and proxied contexts
- ✅ Has health check and config endpoints
- ✅ Proxies API requests correctly
- ✅ Uses zero hardcoded URLs

**Your scenario is now universal!** 🎉
