# Client API Reference

Complete reference for all client-side functions in `@vrooli/api-base`.

## Table of Contents

- [Core Resolution](#core-resolution)
  - [resolveApiBase](#resolveapibase)
  - [resolveWsBase](#resolvewsbase)
  - [resolveWithConfig](#resolvewithconfig)
- [URL Building](#url-building)
  - [buildApiUrl](#buildapiurl)
  - [buildWsUrl](#buildwsurl)
- [Context Detection](#context-detection)
  - [isProxyContext](#isproxycontext)
  - [getProxyInfo](#getproxyinfo)
- [Runtime Configuration](#runtime-configuration)
  - [fetchRuntimeConfig](#fetchruntimeconfig)
  - [getScenarioConfig](#getscenarioconfig)
- [Types](#types)

---

## Core Resolution

### resolveApiBase

Resolves the API base URL for the current deployment context.

**Signature:**
```typescript
function resolveApiBase(options?: ResolveOptions): string
```

**Parameters:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `options` | `ResolveOptions` | `{}` | Resolution configuration options |
| `options.explicitUrl` | `string` | `undefined` | Explicit URL to use (highest priority) |
| `options.defaultPort` | `string` | `'15000'` | Default API port for localhost |
| `options.apiSuffix` | `string` | `'/api/v1'` | API path suffix |
| `options.appendSuffix` | `boolean` | `false` | Whether to append the suffix |
| `options.windowObject` | `WindowLike` | `globalThis.window` | Window object for testing |
| `options.proxyGlobalNames` | `string[]` | See [constants](../concepts/proxy-resolution.md#global-names) | Custom proxy global names |

**Returns:** `string` - The resolved API base URL

**Resolution Priority:**
1. Explicit URL (if provided)
2. Proxy metadata (`window.__VROOLI_PROXY_INFO__`)
3. Path-based proxy detection (`/proxy` in pathname)
4. App shell pattern (`/apps/{slug}/proxy`)
5. Remote host origin (non-localhost domains)
6. Localhost fallback (`http://127.0.0.1:{defaultPort}`)

**Examples:**

```typescript
import { resolveApiBase } from '@vrooli/api-base'

// Basic usage - defaults to localhost
const apiBase = resolveApiBase({ defaultPort: '8080' })
// → "http://127.0.0.1:8080"

// With API suffix
const apiBase = resolveApiBase({
  defaultPort: '8080',
  appendSuffix: true
})
// → "http://127.0.0.1:8080/api/v1"

// Explicit URL (highest priority)
const apiBase = resolveApiBase({
  explicitUrl: 'https://api.example.com',
  appendSuffix: true
})
// → "https://api.example.com/api/v1"

// In proxy context (reads window.__VROOLI_PROXY_INFO__)
// URL: https://app-monitor.com/apps/my-scenario/proxy/
const apiBase = resolveApiBase({ appendSuffix: true })
// → "https://app-monitor.com/apps/my-scenario/proxy/api/v1"

// In direct tunnel
// URL: https://my-scenario.example.com
const apiBase = resolveApiBase({ appendSuffix: true })
// → "https://my-scenario.example.com/api/v1"
```

**Deployment Context Examples:**

| Context | URL | Result |
|---------|-----|--------|
| Localhost | `http://localhost:3000` | `http://127.0.0.1:8080` |
| Direct tunnel | `https://scenario.example.com` | `https://scenario.example.com` |
| App-monitor proxy | `https://host.com/apps/x/proxy/` | `https://host.com/apps/x/proxy` |
| Custom proxy | `https://host.com/embed/y/proxy/` | `https://host.com/embed/y/proxy` |
| VPS deployment | `https://app.company.com` | `https://app.company.com` |
| K8s deployment | `https://company.com/apps/x` | `https://company.com/apps/x` |

**See Also:**
- [Deployment Contexts](../concepts/deployment-contexts.md)
- [Proxy Resolution Algorithm](../concepts/proxy-resolution.md)

---

### resolveWsBase

Resolves the WebSocket base URL for the current deployment context.

**Signature:**
```typescript
function resolveWsBase(options?: ResolveOptions): string
```

**Parameters:**
Same as [resolveApiBase](#resolveapibase)

**Returns:** `string` - The resolved WebSocket base URL with `ws://` or `wss://` protocol

**Protocol Conversion:**
- `http://` → `ws://`
- `https://` → `wss://`

**Examples:**

```typescript
import { resolveWsBase } from '@vrooli/api-base'

// Localhost
const wsBase = resolveWsBase({ defaultPort: '8080' })
// → "ws://127.0.0.1:8080"

// HTTPS site (converts to wss)
// URL: https://my-scenario.example.com
const wsBase = resolveWsBase()
// → "wss://my-scenario.example.com"

// Proxy context
// URL: https://host.com/apps/scenario/proxy/
const wsBase = resolveWsBase()
// → "wss://host.com/apps/scenario/proxy"
```

**Common Pattern:**
```typescript
const API_BASE = resolveApiBase({ defaultPort: '8080', appendSuffix: true })
const WS_BASE = resolveWsBase({ defaultPort: '8080' })

// API requests
fetch(`${API_BASE}/health`)

// WebSocket connections
new WebSocket(`${WS_BASE}/ws`)
```

---

### resolveWithConfig

Resolves API base URL by first attempting to fetch runtime configuration.

**Signature:**
```typescript
async function resolveWithConfig(options?: ResolveOptions): Promise<string>
```

**Parameters:**
All [resolveApiBase](#resolveapibase) parameters plus:

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `options.configEndpoint` | `string` | `'./config'` | Config endpoint URL |

**Returns:** `Promise<string>` - The resolved API base URL

**Resolution Flow:**
1. Try to fetch runtime config from `configEndpoint`
2. If successful, use `config.apiUrl` as base
3. If fetch fails or not in proxy context, fall back to `resolveApiBase()`

**Examples:**

```typescript
import { resolveWithConfig } from '@vrooli/api-base'

// Async resolution with config
const apiBase = await resolveWithConfig({
  defaultPort: '8080',
  appendSuffix: true
})

// Using in component initialization
export async function initializeApp() {
  const API_BASE = await resolveWithConfig({
    defaultPort: process.env.API_PORT || '8080',
    configEndpoint: './config',
    appendSuffix: true
  })

  return API_BASE
}
```

**When to Use:**
- ✅ Production builds where config is injected at runtime
- ✅ Scenarios that need server-provided configuration
- ❌ Simple scenarios where ports are known at build time
- ❌ Performance-critical initialization (adds network request)

**See Also:**
- [Runtime Configuration](../concepts/runtime-config.md)

---

## URL Building

### buildApiUrl

Constructs a complete API URL from a base and path.

**Signature:**
```typescript
function buildApiUrl(path: string, options?: BuildApiUrlOptions): string
```

**Parameters:**

| Name | Type | Description |
|------|------|-------------|
| `path` | `string` | API endpoint path (e.g., `/health`, `/users`) |
| `options` | `BuildApiUrlOptions` | Same as `ResolveOptions` |
| `options.baseUrl` | `string` | Explicit base URL (skips resolution) |

**Returns:** `string` - Complete API URL

**Examples:**

```typescript
import { buildApiUrl, resolveApiBase } from '@vrooli/api-base'

// With explicit base
const url = buildApiUrl('/health', {
  baseUrl: 'http://localhost:8080/api/v1'
})
// → "http://localhost:8080/api/v1/health"

// Resolves base automatically
const url = buildApiUrl('/users', {
  defaultPort: '8080',
  appendSuffix: true
})
// → "http://127.0.0.1:8080/api/v1/users"

// Common pattern: resolve once, build many
const API_BASE = resolveApiBase({ defaultPort: '8080', appendSuffix: true })

fetch(buildApiUrl('/health', { baseUrl: API_BASE }))
fetch(buildApiUrl('/users', { baseUrl: API_BASE }))
fetch(buildApiUrl('/posts', { baseUrl: API_BASE }))
```

**Path Normalization:**
```typescript
buildApiUrl('/health', { baseUrl: 'http://localhost:8080' })
// → "http://localhost:8080/health"

buildApiUrl('health', { baseUrl: 'http://localhost:8080' })
// → "http://localhost:8080/health" (adds leading slash)

buildApiUrl('/health', { baseUrl: 'http://localhost:8080/' })
// → "http://localhost:8080/health" (removes trailing slash)
```

---

### buildWsUrl

Constructs a complete WebSocket URL from a base and path.

**Signature:**
```typescript
function buildWsUrl(path: string, options?: BuildApiUrlOptions): string
```

**Parameters:**
Same as [buildApiUrl](#buildapiurl)

**Returns:** `string` - Complete WebSocket URL with `ws://` or `wss://` protocol

**Examples:**

```typescript
import { buildWsUrl, resolveWsBase } from '@vrooli/api-base'

// With explicit base
const wsUrl = buildWsUrl('/ws', {
  baseUrl: 'ws://localhost:8080'
})
// → "ws://localhost:8080/ws"

// Resolves base automatically
const wsUrl = buildWsUrl('/events', {
  defaultPort: '8080'
})
// → "ws://127.0.0.1:8080/events"

// Common pattern
const WS_BASE = resolveWsBase({ defaultPort: '8080' })
const socket = new WebSocket(buildWsUrl('/ws', { baseUrl: WS_BASE }))
```

---

## Context Detection

### isProxyContext

Determines if the current environment is running in a proxy context.

**Signature:**
```typescript
function isProxyContext(options?: ContextDetectionOptions): boolean
```

**Parameters:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `options.windowObject` | `WindowLike` | `globalThis.window` | Window object for testing |
| `options.proxyGlobalNames` | `string[]` | See [constants](../concepts/proxy-resolution.md#global-names) | Custom proxy global names |

**Returns:** `boolean` - `true` if in proxy context

**Detection Logic:**
1. Check if any proxy global exists (`window.__VROOLI_PROXY_INFO__`, etc.)
2. Check if pathname contains `/proxy/` pattern
3. Return `true` if either condition is met

**Examples:**

```typescript
import { isProxyContext } from '@vrooli/api-base'

// Check if embedded
if (isProxyContext()) {
  console.log('Running in embedded/proxy context')
  // Use proxy-specific behavior
} else {
  console.log('Running standalone')
}

// Conditional configuration
const config = {
  apiBase: isProxyContext()
    ? resolveApiBase({ appendSuffix: true })
    : 'http://localhost:8080/api/v1'
}

// Custom proxy globals
const isCustomProxy = isProxyContext({
  proxyGlobalNames: ['__MY_APP_PROXY__']
})
```

---

### getProxyInfo

Retrieves proxy metadata from the window object.

**Signature:**
```typescript
function getProxyInfo(options?: ContextDetectionOptions): ProxyInfo | null
```

**Parameters:**
Same as [isProxyContext](#isproxycontext)

**Returns:** `ProxyInfo | null` - Proxy metadata or `null` if not in proxy context

**ProxyInfo Structure:**
```typescript
interface ProxyInfo {
  hostScenario: string        // Name of host app (e.g., 'app-monitor')
  targetScenario: string      // Name of embedded app
  generatedAt: number         // Timestamp when metadata was created
  hosts: string[]             // Loopback hostnames (localhost, 127.0.0.1)
  primary: PortEntry          // Primary port configuration
  ports: PortEntry[]          // All available ports
  basePath: string            // Proxy base path
}

interface PortEntry {
  port: number                // Port number
  label: string               // Port label (e.g., 'ui', 'api')
  slug: string                // URL-safe slug
  path: string                // Full proxy path
  aliases: string[]           // Alternative names
}
```

**Examples:**

```typescript
import { getProxyInfo } from '@vrooli/api-base'

const proxyInfo = getProxyInfo()

if (proxyInfo) {
  console.log(`Embedded in: ${proxyInfo.hostScenario}`)
  console.log(`My scenario: ${proxyInfo.targetScenario}`)
  console.log(`Proxy path: ${proxyInfo.basePath}`)
  console.log(`Primary port: ${proxyInfo.primary.port}`)

  // Find specific port by label
  const apiPort = proxyInfo.ports.find(p => p.label === 'api')
  if (apiPort) {
    console.log(`API is at: ${apiPort.path}`)
  }
}

// Use in conditional rendering
function AppHeader() {
  const proxyInfo = getProxyInfo()

  return (
    <header>
      {proxyInfo && (
        <div>Embedded in {proxyInfo.hostScenario}</div>
      )}
    </header>
  )
}
```

---

## Runtime Configuration

### fetchRuntimeConfig

Fetches runtime configuration from the server.

**Signature:**
```typescript
async function fetchRuntimeConfig(
  endpoint?: string,
  windowObject?: WindowLike
): Promise<ScenarioConfig | null>
```

**Parameters:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `endpoint` | `string` | `'./config'` | Config endpoint URL |
| `windowObject` | `WindowLike` | `globalThis.window` | Window object for testing |

**Returns:** `Promise<ScenarioConfig | null>` - Config object or `null` on failure

**ScenarioConfig Structure:**
```typescript
interface ScenarioConfig {
  apiUrl: string              // Full API URL
  wsUrl?: string              // Full WebSocket URL
  apiPort: string             // API port number
  wsPort?: string             // WebSocket port number
  uiPort: string              // UI port number
  serviceName?: string        // Service name
  version?: string            // Service version
  [key: string]: unknown      // Custom fields
}
```

**Examples:**

```typescript
import { fetchRuntimeConfig } from '@vrooli/api-base'

// Fetch default config
const config = await fetchRuntimeConfig()
if (config) {
  console.log(`API: ${config.apiUrl}`)
  console.log(`WS: ${config.wsUrl}`)
}

// Custom endpoint
const config = await fetchRuntimeConfig('/api/config')

// Use in initialization
async function initApp() {
  const config = await fetchRuntimeConfig()

  if (!config) {
    throw new Error('Failed to load configuration')
  }

  return {
    apiClient: createApiClient(config.apiUrl),
    wsClient: createWsClient(config.wsUrl)
  }
}
```

**Error Handling:**
```typescript
// Returns null on any error (doesn't throw)
const config = await fetchRuntimeConfig()

if (!config) {
  console.error('Config fetch failed, using fallback')
  // Use fallback configuration
}
```

---

### getScenarioConfig

Retrieves scenario configuration from the window object.

**Signature:**
```typescript
function getScenarioConfig(
  windowObject?: WindowLike
): ScenarioConfig | null
```

**Parameters:**

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `windowObject` | `WindowLike` | `globalThis.window` | Window object for testing |

**Returns:** `ScenarioConfig | null` - Config object or `null` if not set

**Examples:**

```typescript
import { getScenarioConfig } from '@vrooli/api-base'

// Get config from window.__VROOLI_CONFIG__
const config = getScenarioConfig()

if (config) {
  console.log(`Config loaded: ${config.serviceName}`)

  // Use config directly
  fetch(`${config.apiUrl}/health`)
}

// Fallback pattern
const config = getScenarioConfig() ?? {
  apiUrl: 'http://localhost:8080/api/v1',
  wsUrl: 'ws://localhost:8080/ws',
  apiPort: '8080',
  uiPort: '3000'
}
```

**When to Use:**
- Server has injected config via `injectScenarioConfig()`
- Need synchronous access to config (no async fetch)
- Config is static and doesn't change

**See Also:**
- [Runtime Configuration](../concepts/runtime-config.md)
- [Server: injectScenarioConfig](./server.md#injectscenarioconfig)

---

## Types

All TypeScript types and interfaces are documented in the [Types Reference](./types.md).

**Core Types:**
- [`ResolveOptions`](./types.md#resolveoptions)
- [`BuildApiUrlOptions`](./types.md#buildapiurloptions)
- [`ProxyInfo`](./types.md#proxyinfo)
- [`ScenarioConfig`](./types.md#scenarioconfig)
- [`WindowLike`](./types.md#windowlike)

---

## Complete Example

**Comprehensive client-side setup:**

```typescript
import {
  resolveApiBase,
  resolveWsBase,
  buildApiUrl,
  buildWsUrl,
  isProxyContext,
  getProxyInfo,
  fetchRuntimeConfig,
} from '@vrooli/api-base'

// App initialization
export async function initializeApp() {
  // Check deployment context
  const inProxy = isProxyContext()
  const proxyInfo = getProxyInfo()

  console.log(`Running in proxy: ${inProxy}`)
  if (proxyInfo) {
    console.log(`Hosted by: ${proxyInfo.hostScenario}`)
  }

  // Try fetching runtime config first
  const runtimeConfig = await fetchRuntimeConfig('./config')

  let API_BASE: string
  let WS_BASE: string

  if (runtimeConfig) {
    // Use server-provided config
    API_BASE = runtimeConfig.apiUrl
    WS_BASE = runtimeConfig.wsUrl || resolveWsBase()
  } else {
    // Fall back to resolution
    API_BASE = resolveApiBase({
      defaultPort: import.meta.env.VITE_API_PORT || '8080',
      appendSuffix: true
    })
    WS_BASE = resolveWsBase({
      defaultPort: import.meta.env.VITE_API_PORT || '8080'
    })
  }

  // Create API client
  const apiClient = {
    get: (path: string) => fetch(buildApiUrl(path, { baseUrl: API_BASE })),
    post: (path: string, body: unknown) =>
      fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      })
  }

  // Create WebSocket client
  const wsClient = new WebSocket(buildWsUrl('/ws', { baseUrl: WS_BASE }))

  return {
    API_BASE,
    WS_BASE,
    apiClient,
    wsClient,
    inProxy,
    proxyInfo,
  }
}

// Usage in React
import { useEffect, useState } from 'react'

function App() {
  const [app, setApp] = useState<Awaited<ReturnType<typeof initializeApp>> | null>(null)

  useEffect(() => {
    initializeApp().then(setApp)
  }, [])

  if (!app) return <div>Loading...</div>

  return (
    <div>
      <h1>My Scenario</h1>
      {app.inProxy && <p>Embedded in {app.proxyInfo?.hostScenario}</p>}
      <button onClick={() => app.apiClient.get('/health')}>
        Check Health
      </button>
    </div>
  )
}
```

---

## See Also

- [Quick Start Guide](../guides/quick-start.md)
- [Server API Reference](./server.md)
- [Deployment Contexts](../concepts/deployment-contexts.md)
- [Proxy Resolution](../concepts/proxy-resolution.md)
- [Runtime Configuration](../concepts/runtime-config.md)
- [WebSocket Support](../concepts/websocket-support.md)
