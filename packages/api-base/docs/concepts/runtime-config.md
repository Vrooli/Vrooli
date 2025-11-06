# Runtime Configuration

Understanding how scenarios provide and consume runtime configuration.

## The Problem

Production JavaScript bundles are **static files** - they can't access `process.env` or read files. This creates a challenge:

**Build-time** (when running `npm run build`):
```typescript
// ❌ This works during build, but value is LOCKED IN
const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// If API_URL was undefined at build time, it stays undefined forever
// Even if environment variables change on the server
```

**Runtime** (when user loads the page):
```typescript
// ❌ Can't access process.env in browser
const API_URL = process.env.API_PORT  // undefined

// ❌ Can't read environment variables
fetch('/api/health')  // Works in dev (Vite proxy), fails in production
```

## The Solution: Runtime Configuration

`@vrooli/api-base` provides two complementary approaches:

### 1. **Config Injection** (Server → HTML)

Server injects configuration directly into HTML before serving:

```html
<head>
  <script>
    window.__VROOLI_CONFIG__ = {
      apiUrl: "http://localhost:8080/api/v1",
      wsUrl: "ws://localhost:8081/ws",
      apiPort: "8080",
      wsPort: "8081",
      uiPort: "3000"
    };
  </script>
  <!-- Rest of HTML -->
</head>
```

### 2. **Config Endpoint** (Server → JSON)

Server provides `/config` endpoint that returns runtime values:

```typescript
// Client fetches config
const response = await fetch('./config')
const config = await response.json()
// { apiUrl: "http://localhost:8080/api/v1", ... }
```

---

## Config Injection (Preferred)

### Server-Side Setup

**Using `createScenarioServer`** (automatic):
```typescript
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT || 3000,
  apiPort: process.env.API_PORT || 8080,
  wsPort: process.env.WS_PORT || 8081,
  distDir: './dist',

  // Custom config builder
  configBuilder: (env) => ({
    apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
    wsUrl: `ws://localhost:${env.WS_PORT}/ws`,
    apiPort: String(env.API_PORT),
    wsPort: String(env.WS_PORT),
    uiPort: String(env.UI_PORT),
    // Custom fields
    environment: env.NODE_ENV || 'production',
    features: {
      analytics: env.ENABLE_ANALYTICS === 'true',
    },
  }),
})

// Config automatically injected into HTML
```

**Manual injection**:
```typescript
import { injectScenarioConfig } from '@vrooli/api-base/server'
import fs from 'fs'

// Read HTML
const html = fs.readFileSync('./dist/index.html', 'utf-8')

// Build config
const config = {
  apiUrl: `http://localhost:${process.env.API_PORT}/api/v1`,
  wsUrl: `ws://localhost:${process.env.WS_PORT}/ws`,
  apiPort: String(process.env.API_PORT),
  wsPort: String(process.env.WS_PORT),
  uiPort: String(process.env.UI_PORT),
}

// Inject into HTML
const modifiedHtml = injectScenarioConfig(html, config)

// Serve modified HTML
res.send(modifiedHtml)
```

### Client-Side Access

**Synchronous access** (instant):
```typescript
import { getScenarioConfig } from '@vrooli/api-base'

// Available immediately on page load
const config = getScenarioConfig()

if (config) {
  console.log(`API: ${config.apiUrl}`)
  console.log(`WebSocket: ${config.wsUrl}`)
  console.log(`Environment: ${config.environment}`)
}
```

**Integration with resolution**:
```typescript
import { resolveApiBase, getScenarioConfig } from '@vrooli/api-base'

// Resolution automatically checks window.__VROOLI_CONFIG__
const apiBase = resolveApiBase({
  appendSuffix: true,
  configGlobalName: '__VROOLI_CONFIG__',
})

// Or use config directly
const config = getScenarioConfig()
const apiBase = config?.apiUrl || resolveApiBase({ appendSuffix: true })
```

**Advantages**:
- ✅ **Zero latency** - Available immediately
- ✅ **No network request** - Already in HTML
- ✅ **Works offline** - No `/config` endpoint needed
- ✅ **Type-safe** - Full TypeScript support

---

## Config Endpoint (Alternative)

### Server-Side Setup

**Using `createScenarioServer`** (automatic):
```typescript
import { createScenarioServer } from '@vrooli/api-base/server'

const app = createScenarioServer({
  uiPort: 3000,
  apiPort: 8080,
  wsPort: 8081,
  distDir: './dist',
})

// /config endpoint automatically created
```

**Manual setup**:
```typescript
import { createConfigEndpoint } from '@vrooli/api-base/server'

app.get('/config', createConfigEndpoint({
  apiPort: 8080,
  apiHost: '127.0.0.1',
  wsPort: 8081,
  uiPort: 3000,
  serviceName: 'my-scenario',
  version: '1.0.0',
  additionalConfig: {
    features: { analytics: true },
  },
}))
```

**Response format**:
```json
{
  "apiUrl": "http://127.0.0.1:8080/api/v1",
  "wsUrl": "ws://127.0.0.1:8081/ws",
  "apiPort": "8080",
  "wsPort": "8081",
  "uiPort": "3000",
  "serviceName": "my-scenario",
  "version": "1.0.0"
}
```

### Client-Side Fetching

**Async fetching**:
```typescript
import { fetchRuntimeConfig } from '@vrooli/api-base'

const config = await fetchRuntimeConfig()

if (config) {
  console.log(`API: ${config.apiUrl}`)
} else {
  console.error('Failed to fetch config')
}
```

**With options**:
```typescript
const config = await fetchRuntimeConfig('./config', {
  timeout: 5000,  // 5 second timeout
  retries: 3,     // Retry 3 times on failure
})
```

**Integration with resolution**:
```typescript
import { resolveWithConfig } from '@vrooli/api-base'

// Automatically fetches config and resolves
const apiBase = await resolveWithConfig({
  configEndpoint: './config',
  appendSuffix: true,
})
```

**Advantages**:
- ✅ **Dynamic** - Can change without rebuilding
- ✅ **Flexible** - Different configs per environment
- ✅ **Standard HTTP** - Works with caching, CDNs

**Disadvantages**:
- ⚠️ **Async** - Requires `await` and loading state
- ⚠️ **Network request** - Adds latency
- ⚠️ **Path issues** - Relative URLs in proxy contexts

---

## Comparison

| Feature | Config Injection | Config Endpoint |
|---------|------------------|-----------------|
| **Latency** | Zero (immediate) | ~10-50ms (network) |
| **Loading state** | Not needed | Required |
| **Offline support** | ✅ Yes | ❌ No |
| **Caching** | Automatic (in HTML) | Manual (HTTP cache) |
| **Setup complexity** | Low | Medium |
| **Dynamic updates** | Requires page reload | Can update without reload |
| **Proxy-safe** | ✅ Yes | ⚠️ Needs relative URL |

**Recommendation**: Use **Config Injection** for most scenarios. It's simpler, faster, and works in all contexts.

---

## Proxy Context Challenges

### The Problem

When embedded in a proxy, relative URLs can fail:

```typescript
// Scenario URL: https://app-monitor.com/apps/my-scenario/proxy/

// ❌ Wrong: Requests https://app-monitor.com/config (app-monitor's config)
fetch('/config')

// ✅ Correct: Requests https://app-monitor.com/apps/my-scenario/proxy/config
fetch('./config')
```

### The Solution

**1. Use relative URLs**:
```typescript
fetchRuntimeConfig('./config')  // ✅ Relative to current page
```

**2. Or use config injection** (no request needed):
```typescript
const config = getScenarioConfig()  // ✅ Always works
```

**3. Or construct absolute URL**:
```typescript
const configUrl = `${window.location.origin}${window.location.pathname}/config`
fetchRuntimeConfig(configUrl)
```

---

## Custom Configuration

### Adding Custom Fields

**Server-side**:
```typescript
createScenarioServer({
  uiPort: 3000,
  apiPort: 8080,

  configBuilder: (env) => ({
    apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
    wsUrl: `ws://localhost:${env.WS_PORT}/ws`,
    apiPort: String(env.API_PORT),
    wsPort: String(env.WS_PORT),
    uiPort: String(env.UI_PORT),

    // Custom fields
    environment: env.NODE_ENV || 'production',
    debugMode: env.DEBUG === 'true',
    features: {
      analytics: env.ENABLE_ANALYTICS === 'true',
      betaFeatures: env.ENABLE_BETA === 'true',
      darkMode: true,
    },
    apiKeys: {
      maps: env.MAPS_API_KEY,
      analytics: env.ANALYTICS_KEY,
    },
    thirdParty: {
      stripePublicKey: env.STRIPE_PUBLIC_KEY,
      sentryDsn: env.SENTRY_DSN,
    },
  }),
})
```

**Client-side** (TypeScript):
```typescript
// Extend the config type
declare module '@vrooli/api-base' {
  interface ScenarioConfig {
    environment?: string
    debugMode?: boolean
    features?: {
      analytics?: boolean
      betaFeatures?: boolean
      darkMode?: boolean
    }
    apiKeys?: {
      maps?: string
      analytics?: string
    }
    thirdParty?: {
      stripePublicKey?: string
      sentryDsn?: string
    }
  }
}

// Use with full type safety
const config = getScenarioConfig()
if (config?.features?.analytics) {
  initAnalytics(config.apiKeys?.analytics)
}
```

---

## Best Practices

### ✅ Do

**1. Use config injection for core settings**:
```typescript
// Server
createScenarioServer({
  configBuilder: (env) => ({
    apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
    wsUrl: `ws://localhost:${env.WS_PORT}/ws`,
    // ...
  }),
})

// Client
const config = getScenarioConfig()  // Instant access
```

**2. Provide fallbacks**:
```typescript
const config = getScenarioConfig()
const apiUrl = config?.apiUrl || resolveApiBase({
  appendSuffix: true,
})
```

**3. Type-safe custom fields**:
```typescript
// Extend the interface
declare module '@vrooli/api-base' {
  interface ScenarioConfig {
    customField?: string
  }
}
```

**4. Validate configuration**:
```typescript
const config = getScenarioConfig()
if (!config) {
  console.error('Configuration not found')
  return
}

if (!config.apiUrl) {
  console.error('API URL missing from config')
}
```

---

### ❌ Don't

**1. Use build-time env vars for runtime values**:
```typescript
// ❌ Bad: Locked in at build time
const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// ✅ Good: Runtime resolution
const config = getScenarioConfig()
const API_URL = config?.apiUrl || resolveApiBase({ appendSuffix: true })
```

**2. Hard-code environment-specific values**:
```typescript
// ❌ Bad
const API_URL = process.env.NODE_ENV === 'production'
  ? 'https://api.example.com'
  : 'http://localhost:8080'

// ✅ Good
const config = getScenarioConfig()
const API_URL = config?.apiUrl || resolveApiBase({ appendSuffix: true })
```

**3. Use absolute paths in proxy contexts**:
```typescript
// ❌ Bad: Fails in proxy
fetch('/config')

// ✅ Good: Relative path
fetch('./config')

// ✅ Better: No request needed
const config = getScenarioConfig()
```

**4. Store secrets in config**:
```typescript
// ❌ Bad: Client-visible!
configBuilder: (env) => ({
  apiUrl: '...',
  secretApiKey: env.SECRET_API_KEY,  // ❌ Exposed to browser!
})

// ✅ Good: Secrets stay on server
// Use secure cookies or session tokens instead
```

---

## Migration Guide

### From Hardcoded URLs

**Before**:
```typescript
const API_URL = 'http://localhost:8080/api/v1'

fetch(`${API_URL}/health`)
```

**After**:
```typescript
import { getScenarioConfig, resolveApiBase, buildApiUrl } from '@vrooli/api-base'

// Get injected config
const config = getScenarioConfig()
const API_BASE = config?.apiUrl || resolveApiBase({
  appendSuffix: true,
})

fetch(buildApiUrl('/health', { baseUrl: API_BASE }))
```

---

### From Vite Env Vars

**Before**:
```typescript
const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'
```

**After**:

1. **Server** (inject config):
```typescript
createScenarioServer({
  apiPort: process.env.API_PORT || 8080,
  configBuilder: (env) => ({
    apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
    // ...
  }),
})
```

2. **Client** (read config):
```typescript
const config = getScenarioConfig()
const API_URL = config?.apiUrl || resolveApiBase({
  appendSuffix: true,
})
```

---

### From Custom /config Endpoint

**Before**:
```typescript
// server.js
app.get('/config', (req, res) => {
  res.json({
    apiUrl: `http://localhost:${process.env.API_PORT}/api/v1`,
  })
})

// client
const response = await fetch('/config')
const config = await response.json()
```

**After**:
```typescript
// server.js
import { createConfigEndpoint } from '@vrooli/api-base/server'

app.get('/config', createConfigEndpoint({
  apiPort: process.env.API_PORT || 8080,
  uiPort: process.env.UI_PORT || 3000,
}))

// client
import { fetchRuntimeConfig } from '@vrooli/api-base'

const config = await fetchRuntimeConfig('./config')
```

---

## Troubleshooting

### Config is undefined

**Symptom**: `getScenarioConfig()` returns `null`

**Solutions**:

1. **Verify injection** (server):
```typescript
// Check that config is being injected
const app = createScenarioServer({
  configBuilder: (env) => ({
    apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
    // ...
  }),
  verbose: true,  // Enable logging
})
```

2. **Check HTML** (browser):
```typescript
// Open browser console
console.log(window.__VROOLI_CONFIG__)

// If undefined, config wasn't injected
// If present, getScenarioConfig should work
```

3. **Verify global name**:
```typescript
// Check custom global name
const config = getScenarioConfig('__CUSTOM_CONFIG__')
```

---

### Config fetch fails in proxy

**Symptom**: `/config` request goes to wrong server

**Solution**: Use relative URL

```typescript
// ❌ Bad
fetchRuntimeConfig('/config')  // Absolute

// ✅ Good
fetchRuntimeConfig('./config')  // Relative
```

---

### Environment variables not updating

**Symptom**: Config shows old values after env var changes

**Solution**: Config is injected at server start. Restart the server:

```bash
# Stop server
vrooli scenario stop my-scenario

# Update env vars
export API_PORT=9000

# Start server
vrooli scenario start my-scenario
```

---

## See Also

- [Client API: getScenarioConfig](../api/client.md#getscenarioconfig)
- [Client API: fetchRuntimeConfig](../api/client.md#fetchruntimeconfig)
- [Server API: injectScenarioConfig](../api/server.md#injectscenarioconfig)
- [Server API: createConfigEndpoint](../api/server.md#createconfigendpoint)
- [Proxy Resolution](./proxy-resolution.md)
- [Quick Start Guide](../guides/quick-start.md)
