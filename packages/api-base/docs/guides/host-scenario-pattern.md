# Host Scenario Pattern

Guide for scenarios that host/embed other scenarios (like app-monitor).

## Overview

A **host scenario** is a scenario that displays other scenarios within it, typically using iframes or proxy routes. This creates a nested architecture where:

- The **host** (e.g., `app-monitor`) serves its own UI and manages routing
- The **embedded scenarios** (e.g., `scenario-auditor`) are proxied through the host

This pattern has unique challenges that `@vrooli/api-base` now handles automatically.

## The Core Challenge

Host scenarios face a dual routing problem:

1. **Their own assets** must load correctly from `/` regardless of what nested URL path they're accessed from
2. **Proxied scenario assets** must load from `/apps/{scenario}/proxy/` to route through the proxy

Without proper handling:
- Host's assets break when accessed via nested URLs like `/apps/scenario-x/preview`
- Proxied scenarios receive HTML instead of JS/CSS files (breaking the app)
- Base tags conflict between host and embedded scenarios

## New Features in api-base

### 1. Smart SPA Fallback

**Problem**: When a proxied scenario requests `/assets/main.js`, the host's SPA fallback was returning `index.html` instead of 404, breaking the embedded app.

**Solution**: `createScenarioServer()` now automatically detects asset requests:

```typescript
// Built into createScenarioServer - no configuration needed!
app.get('*', (req, res) => {
  if (isAssetRequest(req.path)) {
    // Return 404 for missing assets, not index.html
    res.status(404).send('Not found')
    return
  }

  // Only serve index.html for non-asset routes
  res.sendFile(indexPath)
})
```

**What counts as an asset**:
- Has extension: `.js`, `.css`, `.png`, `.json`, `.woff`, etc. (30+ extensions)
- Has prefix: `/@vite`, `/assets/`, `/static/`, `/_next/`, etc.

### 2. Base Tag Injection Helper

**Problem**: Manually injecting `<base>` tags is error-prone and requires HTML parsing logic.

**Solution**: New `injectBaseTag()` function:

```typescript
import { injectBaseTag } from '@vrooli/api-base/server'

// For host scenario's own HTML (load assets from root)
html = injectBaseTag(html, '/')

// For proxied scenario HTML (load assets through proxy)
html = injectBaseTag(html, '/apps/scenario-x/proxy/')
```

**Features**:
- Automatically finds `<head>` or appropriate location
- Skips injection if base tag already exists
- Ensures trailing slash for correct relative path resolution
- Adds data attributes for debugging

### 3. Asset Detection Utilities

**Problem**: Each scenario implements its own logic to detect asset requests.

**Solution**: Shared utilities in `@vrooli/api-base`:

```typescript
import { isAssetRequest, hasAssetExtension, startsWithAssetPrefix } from '@vrooli/api-base/server'

// High-level check
if (isAssetRequest('/logo.png')) { /* ... */ }

// Granular checks
if (hasAssetExtension('/main.js?v=123')) { /* true */ }
if (startsWithAssetPrefix('/@vite/client')) { /* true */ }
```

## Implementation Pattern

### Step 1: Create Base Server

Use `createScenarioServer()` for the host's own routes:

```javascript
import { createScenarioServer, injectBaseTag } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  distDir: './dist',
  serviceName: 'app-monitor',
  corsOrigins: '*',
  verbose: true,

  setupRoutes: (app) => {
    // Add your custom routes here (see steps below)
  }
})
```

### Step 2: Inject Base Tag for Host's Own Pages

Ensure the host's assets load from root, regardless of URL:

```javascript
setupRoutes: (app) => {
  // Middleware to inject base tag for host's own HTML
  app.use((req, res, next) => {
    // Skip proxied scenario requests
    if (req.path.startsWith('/apps/') && req.path.includes('/proxy')) {
      return next()
    }

    // Intercept HTML responses
    const originalSend = res.send
    res.send = function(body) {
      const contentType = res.getHeader('content-type')
      const isHtml = contentType && contentType.includes('text/html')

      if (isHtml && typeof body === 'string') {
        // Inject <base href="/"> for host's assets
        const modifiedBody = injectBaseTag(body, '/', {
          skipIfExists: true,
          dataAttribute: 'data-host-self'
        })
        return originalSend.call(this, modifiedBody)
      }

      return originalSend.call(this, body)
    }

    next()
  })
}
```

### Step 3: Add Proxy Routes for Embedded Scenarios

Handle requests to `/apps/{scenario}/proxy/*`:

```javascript
setupRoutes: (app) => {
  // ... base tag injection middleware from step 2 ...

  app.all('/apps/:appId/proxy/*', async (req, res) => {
    const { appId } = req.params
    const targetPath = req.params[0] || ''

    try {
      // 1. Fetch scenario metadata from your API
      const metadata = await fetchScenarioMetadata(appId)
      const targetPort = metadata.uiPort

      // 2. For HTML requests, inject proxy metadata
      const isHtmlRequest = req.headers.accept?.includes('text/html')

      if (isHtmlRequest && req.method === 'GET') {
        // Fetch HTML from target scenario
        const htmlResponse = await axios.get(
          `http://127.0.0.1:${targetPort}/${targetPath}`,
          { responseType: 'text' }
        )

        let html = htmlResponse.data

        // 3. Build and inject proxy metadata
        const proxyMetadata = buildProxyMetadata({
          appId,
          hostScenario: 'app-monitor',
          targetScenario: appId,
          ports: [/* your port entries */],
          primaryPort: /* UI port entry */
        })

        html = injectProxyMetadata(html, proxyMetadata, {
          patchFetch: true // For backwards compatibility
        })

        // 4. Inject base tag for proxied scenario's assets
        html = injectBaseTag(html, `/apps/${appId}/proxy/`, {
          skipIfExists: true,
          dataAttribute: 'data-proxied-scenario'
        })

        res.set('Content-Type', 'text/html')
        res.send(html)
      } else {
        // 5. For non-HTML requests, use direct proxy
        await proxyToApi(req, res, `/${targetPath}`, {
          apiPort: targetPort,
          apiHost: '127.0.0.1',
          timeout: 30000
        })
      }
    } catch (error) {
      console.error(`Proxy error for ${appId}:`, error.message)
      if (!res.headersSent) {
        res.status(502).json({ error: 'Proxy failed' })
      }
    }
  })
}
```

### Step 4: Always route API + WebSocket traffic through the host UI

- Use `proxyToApi` (or `createProxyMiddleware`) so every HTTP request flows through the host before touching the child scenario. This keeps Cloudflare/SSH/VPS tunnels happy and prevents "Bad Gateway" surprises when the child API isn't publicly reachable.
- When you need to forward to a specific child port, fetch the app metadata once, cache it, and resolve the correct UI/API port before proxying (exactly like the new `scenarios/app-monitor/ui/server.js`).
- Preserve query strings and HTTP verbs by rewriting the Express `req.url` (e.g., `modifiedReq.url = relativeUrl`) before handing it to `proxyToApi`.

```ts
await proxyToApi(modifiedReq, res, relativeUrl, {
  apiPort: targetPort,
  apiHost: '127.0.0.1',
  timeout: 30_000,
})
```

### Step 5: Bridge WebSocket upgrades

Host dashboards like app-monitor often need to show another scenario's WebSocket stream (issue trackers, consoles, etc.). Attach `proxyWebSocketUpgrade` to your HTTP server so `/apps/:id/proxy/...` sockets ride the same path as HTTP:

```ts
server.on('upgrade', async (req, socket, head) => {
  if (!req.url?.includes('/proxy')) {
    socket.destroy()
    return
  }

  const relativeUrl = extractProxyRelativeUrl(req.url)
  const targetPort = relativeUrl.startsWith('/api') ? apiPort : uiPort
  const proxyReq = Object.create(req)
  proxyReq.url = relativeUrl

  proxyWebSocketUpgrade(proxyReq, socket, head, {
    apiPort: targetPort,
    apiHost: '127.0.0.1',
  })
})
```

👉 **Important:** Clients should never hard-code `API_PORT`. The browser must call `resolveApiBase`/`resolveWsBase` so requests stay inside the host UI origin (even on localhost). This is mandatory when the host is exposed through Cloudflare tunnels, reverse proxies, or Kubernetes ingresses.

### Step 6: Restart via lifecycle commands

Whenever you tweak the host server (routing, caching, WS handling), run `vrooli scenario restart <scenario>` so both the API and UI are rebuilt and redeployed with the lifecycle hooks. This flushes stale processes and ensures your proxy code is the one actually running.

## Complete Example: app-monitor

See `/scenarios/app-monitor/ui/server.js` for the full implementation, which includes:

1. ✅ Base server with smart SPA fallback (automatic via `createScenarioServer`)
2. ✅ Base tag injection for app-monitor's own assets (`<base href="/">`)
3. ✅ Proxy routes for embedded scenarios (`/apps/:appId/proxy/*`)
4. ✅ Base tag injection for proxied scenarios (`<base href="/apps/x/proxy/">`)
5. ✅ WebSocket support for real-time updates
6. ✅ Additional API proxy routes for app-monitor-specific endpoints

## Key Benefits

### For Host Scenarios

- **Simplified code**: No manual asset detection or base tag parsing
- **Correct routing**: Assets load from correct paths automatically
- **Maintainable**: Logic centralized in api-base, not duplicated

### For Embedded Scenarios

- **Transparent proxying**: Scenarios don't need to know they're embedded
- **Asset resolution**: Base tags ensure relative paths work correctly
- **API access**: Proxy metadata enables correct API endpoint resolution

### For Developers

- **Standard pattern**: All host scenarios follow the same approach
- **Well-tested**: Logic tested once in api-base, not per-scenario
- **Debuggable**: Data attributes on injected tags aid troubleshooting

## Troubleshooting

### Host's Assets Not Loading

**Symptom**: When accessing `/apps/scenario-x/preview`, host's CSS/JS fails to load.

**Solution**: Ensure base tag injection middleware runs for host's pages:

```javascript
// Check browser devtools - should see: <base data-host-self="injected" href="/">
```

### Embedded Scenario's Assets Not Loading

**Symptom**: Proxied scenario shows blank page or console errors for missing JS/CSS.

**Solution**: Verify base tag injection for proxied HTML:

```javascript
// Check iframe HTML - should see: <base data-proxied-scenario="injected" href="/apps/x/proxy/">
```

### Asset Requests Return HTML

**Symptom**: Browser console shows "Uncaught SyntaxError: Unexpected token '<'" when loading JS.

**Solution**: This means the SPA fallback is serving HTML for asset requests. Ensure you're using the latest `createScenarioServer()` which has smart asset detection built-in. If using a custom SPA fallback, use `isAssetRequest()`:

```javascript
import { isAssetRequest } from '@vrooli/api-base/server'

app.get('*', (req, res) => {
  if (isAssetRequest(req.path)) {
    res.status(404).send('Not found')
    return
  }
  res.sendFile('dist/index.html')
})
```

## Migration from Old Pattern

### Before (Manual, Error-Prone)

```javascript
// 200+ lines of custom proxy logic
// Manual HTML parsing with parse5
// Custom asset detection regexes
// Manual base tag injection
// Duplicated across scenarios
```

### After (Standardized, Reliable)

```javascript
import { createScenarioServer, injectBaseTag, buildProxyMetadata, injectProxyMetadata } from '@vrooli/api-base/server'

const app = createScenarioServer({
  /* config */
  setupRoutes: (app) => {
    // ~30 lines of proxy route logic
    // Uses api-base helpers
    // Smart asset detection automatic
    // Standard pattern
  }
})
```

**Reduction**: ~170 lines → ~30 lines per host scenario

## Related Documentation

- [Deployment Contexts](../concepts/deployment-contexts.md) - Understanding localhost vs tunnel vs proxy
- [Proxy Resolution](../concepts/proxy-resolution.md) - How embedded scenarios resolve API URLs
- [Server Template](../api/server.md) - Full API reference for `createScenarioServer()`
- [Injection Utilities](../api/inject.md) - API reference for metadata and base tag injection

## Testing Strategy

Use `resource-browserless` to test the complete flow:

```bash
# Start app-monitor
vrooli scenario start app-monitor

# Start a scenario to proxy
vrooli scenario start scenario-auditor

# Test with browserless
resource-browserless execute \
  --url "http://localhost:${APP_MONITOR_PORT}/apps/scenario-auditor/proxy/" \
  --script "test-proxy-flow.js"
```

See `/scenarios/app-monitor/test/` for complete test suite.
