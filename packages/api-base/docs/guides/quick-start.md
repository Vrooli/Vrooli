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
  defaultPort: '8080',      // Your API port
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
  defaultPort: process.env.VITE_API_PORT || '8080',
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

## Step 3: Create Server (Express)

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
  verbose: true  // Logs proxy requests
})

const PORT = process.env.UI_PORT || 3000
app.listen(PORT, () => {
  console.log(`UI server running on port ${PORT}`)
})
```

This automatically sets up:
- Static file serving from `./dist`
- `/api/*` proxy to your API server
- `/health` endpoint with API connectivity check
- `/config` endpoint with runtime configuration
- CORS handling
- SPA fallback routing

## Step 4: Test All Contexts

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

## Step 5: Add WebSocket Support (Optional)

If your scenario uses WebSockets:

```typescript
import { resolveWsBase } from '@vrooli/api-base'

const WS_BASE = resolveWsBase({
  defaultPort: '8080',
  appendSuffix: true  // Adds /ws
})

const ws = new WebSocket(WS_BASE)
ws.onopen = () => console.log('Connected!')
```

## What Just Happened?

### Client-Side

`resolveApiBase()` automatically:
1. Checks for proxy metadata (`window.__VROOLI_PROXY_INFO__`)
2. Detects `/proxy` in the URL path
3. Uses `window.location.origin` if on remote host
4. Falls back to `http://127.0.0.1:{defaultPort}`

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
  defaultPort: '8080',
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

- **[Client Usage Guide](./client-usage.md)** - Complete client-side patterns
- **[Server Usage Guide](./server-usage.md)** - Advanced server configuration
- **[Deployment Contexts](../concepts/deployment-contexts.md)** - Understanding how resolution works
- **[Testing Guide](./testing-guide.md)** - Testing in all contexts

## Summary

You've successfully integrated `@vrooli/api-base`! Your scenario now:
- ✅ Works in localhost, direct tunnels, and proxied contexts
- ✅ Has health check and config endpoints
- ✅ Proxies API requests correctly
- ✅ Uses zero hardcoded URLs

**Your scenario is now universal!** 🎉
