# Migration Guide

Step-by-step instructions for migrating existing scenarios to `@vrooli/api-base`.

## Who Should Migrate?

You should migrate if your scenario:
- ✅ Has custom `config.ts` or API resolution logic
- ✅ Uses hardcoded `localhost` URLs
- ✅ Implements custom proxy detection
- ✅ Has a manual Express server setup
- ✅ Doesn't work in all three deployment contexts

## Migration Paths

Choose the path that matches your current implementation:

### Path A: Custom config.ts Implementation

**Example scenarios**: browser-automation-studio, some older scenarios

### Path B: Hardcoded URLs

**Example scenarios**: Simple scenarios with `const API_BASE = 'http://localhost:8080'`

### Path C: Old @vrooli/api-base (0.x)

**Example scenarios**: scenario-auditor, scenarios that already use the package

### Path D: Custom Server Setup

**Example scenarios**: Scenarios with manual Express configuration

---

## Path A: From Custom config.ts

### Before

Your scenario has a file like this:

```typescript
// src/config.ts
export async function getConfig() {
  try {
    const response = await fetch('/config')
    const config = await response.json()
    return config
  } catch (error) {
    return {
      apiUrl: 'http://localhost:8080/api/v1',
      wsUrl: 'ws://localhost:8080/ws'
    }
  }
}

// Usage
const config = await getConfig()
const API_BASE = config.apiUrl
```

### Migration Steps

#### 1. Install Package

```bash
cd ui
pnpm add @vrooli/api-base
```

#### 2. Replace config.ts

**Delete** `src/config.ts` (or keep for reference during migration).

**Update** your main app file:

```typescript
// src/App.tsx (or wherever you initialize)
import { resolveApiBase, resolveWsBase } from '@vrooli/api-base'

// Replace getConfig() call with:
const API_BASE = resolveApiBase({
  defaultPort: '8080',      // Your API port
  appendSuffix: true        // Adds /api/v1
})

const WS_BASE = resolveWsBase({
  defaultPort: '8080',
  appendSuffix: true        // Adds /ws
})

// Use API_BASE for all fetch calls
fetch(`${API_BASE}/data`)

// Use WS_BASE for WebSocket connections
const ws = new WebSocket(WS_BASE)
```

#### 3. Remove /config Endpoint (Optional)

The server template provides `/config` automatically, but if you want to keep your custom implementation:

```javascript
// server.js
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  configBuilder: (env) => ({
    // Your custom config here
    apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
    wsUrl: `ws://localhost:${env.WS_PORT}/ws`,
    customField: 'value'
  })
})
```

#### 4. Test

```bash
# Localhost
vrooli scenario start your-scenario
curl http://localhost:$UI_PORT/api/health

# Tunnel
curl https://your-scenario.itsagitime.com/api/health

# Proxied (via app-monitor)
curl https://app-monitor.itsagitime.com/apps/your-scenario/proxy/api/health
```

### Breaking Changes

- ⚠️ `getConfig()` is now synchronous (no async/await needed)
- ⚠️ Config is resolved at initialization, not fetched dynamically
- ✅ Works in all contexts automatically

---

## Path B: From Hardcoded URLs

### Before

```typescript
// ❌ Hardcoded localhost URL
const API_BASE = 'http://localhost:8080/api/v1'

fetch(`${API_BASE}/data`)
fetch(`${API_BASE}/users`)
```

### Migration Steps

#### 1. Install Package

```bash
cd ui
pnpm add @vrooli/api-base
```

#### 2. Replace Hardcoded URLs

```typescript
// ✅ Universal resolution
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

const API_BASE = resolveApiBase({
  defaultPort: '8080',
  appendSuffix: true
})

// Option 1: Template literals (same as before)
fetch(`${API_BASE}/data`)

// Option 2: buildApiUrl helper (recommended)
fetch(buildApiUrl('/data', { baseUrl: API_BASE }))
fetch(buildApiUrl('/users', { baseUrl: API_BASE }))
```

#### 3. Update Server

Replace manual setup with template:

```javascript
// server.js
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT || 3000,
  apiPort: process.env.API_PORT || 8080,
  distDir: './dist'
})

app.listen(process.env.UI_PORT || 3000)
```

### Breaking Changes

- ✅ None! Just more flexibility.

---

## Path C: From Old @vrooli/api-base (0.x)

### What Changed in 1.0?

| Feature | v0.x | v1.0 |
|---------|------|------|
| **Client exports** | ✅ Same | ✅ Same |
| **Global names** | `__APP_MONITOR_PROXY_INFO__` only | Both old and new supported |
| **WebSocket** | ❌ Not included | ✅ `resolveWsBase()` |
| **Runtime config** | ❌ Not included | ✅ `fetchRuntimeConfig()` |
| **Server utils** | ❌ Not included | ✅ Full template |
| **Types export** | ❌ Not available | ✅ `/types` export |

### Migration Steps

#### 1. Update Package

```bash
cd ui
pnpm update @vrooli/api-base
```

#### 2. Add Server Template (Optional)

If you want the new server features:

```javascript
// server.js
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'your-scenario',
  version: '1.0.0'
})

app.listen(process.env.UI_PORT)
```

#### 3. Add WebSocket Support (Optional)

```typescript
import { resolveWsBase } from '@vrooli/api-base'

const WS_BASE = resolveWsBase({
  defaultPort: '8080',
  appendSuffix: true
})

const ws = new WebSocket(WS_BASE)
```

### Breaking Changes

- ✅ None! Fully backwards compatible.
- ✅ Old global names still work (`__APP_MONITOR_PROXY_INFO__`)
- ✅ All existing code continues to work

---

## Path D: From Custom Server Setup

### Before

```javascript
// server.js - manual Express setup
const express = require('express')
const http = require('http')
const app = express()

// Static files
app.use(express.static('./dist'))

// API proxy
app.use('/api', (req, res) => {
  const options = {
    hostname: 'localhost',
    port: process.env.API_PORT || 8080,
    path: req.url,
    method: req.method,
    headers: req.headers
  }

  const proxyReq = http.request(options, (proxyRes) => {
    res.status(proxyRes.statusCode)
    proxyRes.pipe(res)
  })

  req.pipe(proxyReq)
})

// SPA fallback
app.get('*', (req, res) => {
  res.sendFile(__dirname + '/dist/index.html')
})

app.listen(3000)
```

### Migration Steps

#### 1. Install Package

```bash
cd ui
pnpm add @vrooli/api-base express
```

#### 2. Replace with Template

```javascript
// server.js
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT || 3000,
  apiPort: process.env.API_PORT || 8080,
  distDir: './dist',
  serviceName: 'your-scenario',
  version: '1.0.0',
  verbose: true  // Optional: log proxy requests
})

// Add custom routes if needed
app.get('/custom', (req, res) => {
  res.json({ custom: 'data' })
})

app.listen(process.env.UI_PORT || 3000)
```

#### 3. Update package.json (if using CommonJS)

If your `package.json` has `"type": "module"`, you're good.

If not, rename `server.js` to `server.cjs` and update service.json:

```json
{
  "develop": {
    "steps": [
      {
        "name": "start-ui",
        "run": "cd ui && node server.cjs",
        "background": true
      }
    ]
  }
}
```

### What You Get

The template automatically provides:
- ✅ Static file serving
- ✅ API proxy middleware
- ✅ `/health` endpoint
- ✅ `/config` endpoint
- ✅ CORS handling
- ✅ SPA fallback routing
- ✅ Error handling
- ✅ Request logging (when `verbose: true`)

### Custom Routes

Add custom routes after creating the server:

```javascript
const app = createScenarioServer({...})

// Custom routes
app.get('/special', (req, res) => {
  res.json({ special: true })
})

// Or use setupRoutes option
const app = createScenarioServer({
  // ... other options
  setupRoutes: (app) => {
    app.get('/special', (req, res) => {
      res.json({ special: true })
    })
  }
})
```

---

## Common Migration Issues

### Issue: TypeScript errors after migration

**Symptom**:
```
Cannot find module '@vrooli/api-base' or its corresponding type declarations
```

**Solution**:
```bash
# Rebuild the package
cd packages/api-base
pnpm build

# Rebuild your scenario
cd ../../scenarios/your-scenario/ui
pnpm install
```

### Issue: 502 API server unavailable

**Symptom**: API requests fail with 502 error

**Solution**: Check API_PORT environment variable:

```javascript
// server.js - add logging
console.log('API_PORT:', process.env.API_PORT)
console.log('UI_PORT:', process.env.UI_PORT)
```

Ensure service.json has correct ports:

```json
{
  "environment": {
    "API_PORT": "8080",
    "UI_PORT": "3000"
  }
}
```

### Issue: Works in localhost but fails in tunnel

**Symptom**: Scenario works locally but breaks when accessed via Cloudflare tunnel

**Solution**: Check proxy metadata in browser console:

```javascript
console.log('Proxy info:', window.__VROOLI_PROXY_INFO__)
console.log('Location:', window.location.href)
```

If proxy info is missing, the hosting scenario may need to be updated to inject metadata.

### Issue: Assets (CSS/JS) not loading when embedded

**Symptom**: Scenario loads but has no styles when embedded in app-monitor

**Solution**: Ensure your build uses relative paths:

```javascript
// vite.config.ts
export default defineConfig({
  base: './',  // Use relative paths
  // ...
})
```

---

## Migration Checklist

Use this checklist to ensure complete migration:

### Client-Side
- [ ] Installed `@vrooli/api-base`
- [ ] Replaced hardcoded URLs or config fetching
- [ ] Using `resolveApiBase()`
- [ ] Using `resolveWsBase()` for WebSockets (if applicable)
- [ ] Removed custom proxy detection logic
- [ ] Tested in localhost
- [ ] Tested in direct tunnel
- [ ] Tested when embedded in app-monitor

### Server-Side
- [ ] Using `createScenarioServer()` template
- [ ] Removed manual proxy setup
- [ ] `/health` endpoint works
- [ ] `/config` endpoint works
- [ ] `/api/*` proxy works
- [ ] Custom routes still work (if any)
- [ ] Static files serve correctly
- [ ] SPA fallback routing works

### Testing
- [ ] `vrooli scenario start` works
- [ ] Health checks pass in all contexts
- [ ] API requests work in all contexts
- [ ] WebSocket connections work (if applicable)
- [ ] No console errors in browser
- [ ] Build process works (`npm run build`)

---

## Need Help?

- Check the [Quick Start Guide](./quick-start.md) for basic setup
- See [Client Usage](./client-usage.md) for detailed client patterns
- See [Server Usage](./server-usage.md) for advanced server configuration
- Review [Examples](../examples/) for complete working code

## Summary

Migration typically takes **15-30 minutes** per scenario:
- ✅ Client code: Replace `config.ts` or hardcoded URLs (5 min)
- ✅ Server code: Use template (5 min)
- ✅ Testing: Verify all contexts (10-20 min)

The result: **Universal scenarios that work everywhere** with minimal code.
