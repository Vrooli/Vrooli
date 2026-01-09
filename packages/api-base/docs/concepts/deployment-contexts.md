# Deployment Contexts

Understanding the three environments where Vrooli scenarios run - and why universal API resolution matters.

## The Challenge

A Vrooli scenario must work seamlessly in **three fundamentally different deployment contexts**, each with unique networking requirements:

| Context | User Access | API Location | Challenge |
|---------|-------------|--------------|-----------|
| **Localhost** | `http://localhost:3000` | `http://localhost:8080` | Simple - both on loopback |
| **Direct Tunnel** | `https://scenario.example.com` | Same server | API requests need same origin |
| **Proxied/Embedded** | `https://host.com/apps/scenario/proxy/` | Behind proxy | Requests need proxy path prefix |

## Context 1: Localhost Development

### Environment

```
Developer's Machine
├── UI Server (localhost:3000)
└── API Server (localhost:8080)
```

### User Experience

Developer opens `http://localhost:3000` in their browser.

### API Resolution

The UI needs to make requests to `http://localhost:8080/api/v1`.

**Challenges:**
- ✅ Simple - both services on same machine
- ✅ Direct network access
- ⚠️ Port must be known at build time OR fetched at runtime

### Example Request Flow

```
Browser: http://localhost:3000
  ↓
  Fetches: http://localhost:3000/api/health
  ↓
  UI Server proxies to: http://localhost:8080/api/health
  ↓
  API Server responds
  ↓
  Response returned to browser
```

### Key Insight

The UI server **must proxy** API requests to handle CORS and maintain flexibility.

## Context 2: Direct Tunnel

### Environment

```
Local Server + Cloudflare Tunnel
├── UI Server (localhost:3000) ←─┐
├── API Server (localhost:8080) ←─┤
└── Cloudflare Tunnel ────────────┘
         ↓
    Internet: https://my-scenario.example.com
```

### User Experience

User accesses `https://my-scenario.example.com` from anywhere in the world.

### API Resolution

The UI needs to make requests to `https://my-scenario.example.com/api/v1`.

**Challenges:**
- ⚠️ Must use HTTPS (not localhost URLs)
- ⚠️ Must use same origin (no cross-origin requests)
- ✅ Tunnel handles routing to local ports

### Example Request Flow

```
Browser: https://my-scenario.example.com
  ↓
  Fetches: https://my-scenario.example.com/api/health
  ↓
  Cloudflare Tunnel routes to: localhost:3000/api/health
  ↓
  UI Server proxies to: localhost:8080/api/health
  ↓
  API Server responds
  ↓
  Response returns through tunnel
  ↓
  Browser receives response
```

### Key Insight

UI must detect it's being accessed via a **remote origin** and use `window.location.origin` instead of localhost.

## Context 3: Proxied/Embedded

### Environment

```
Host Scenario (App-Monitor)
├── UI Server (localhost:4000)
│   └── Embeds: Scenario X in iframe
├── Proxy Routes:
│   └── /apps/scenario-x/proxy/* → Scenario X (localhost:3000)
└── Cloudflare Tunnel
         ↓
    Internet: https://app-monitor.example.com
```

### User Experience

User accesses `https://app-monitor.example.com/apps/scenario-x/proxy/`.

Scenario X is loaded in an iframe, with **all requests proxied through** app-monitor.

### API Resolution

The embedded scenario needs to make requests to:
```
https://app-monitor.example.com/apps/scenario-x/proxy/api/v1
```

**Challenges:**
- ⚠️ Must include proxy path prefix (`/apps/scenario-x/proxy/`)
- ⚠️ Must use host origin (app-monitor.example.com, not scenario-x.com)
- ⚠️ Must work without knowing host URL ahead of time
- ⚠️ Assets (JS, CSS) must also use proxied paths

### Example Request Flow

```
Browser: https://app-monitor.example.com/apps/scenario-x/proxy/
  ↓
  Loads iframe with scenario-x UI
  ↓
  UI Fetches: https://app-monitor.example.com/apps/scenario-x/proxy/api/health
  ↓
  App-Monitor proxies to: http://localhost:3000/api/health
  ↓
  Scenario-X UI Server proxies to: http://localhost:8080/api/health
  ↓
  Scenario-X API Server responds
  ↓
  Response returns through double-proxy
  ↓
  iframe receives response
```

### Key Insight

The host scenario **injects metadata** into the embedded scenario's HTML:

```javascript
window.__VROOLI_PROXY_INFO__ = {
  hostScenario: 'app-monitor',
  targetScenario: 'scenario-x',
  primary: {
    path: '/apps/scenario-x/proxy'
  },
  // ... more metadata
}
```

The embedded scenario reads this metadata to construct correct URLs.

## Resolution Strategy

`@vrooli/api-base` resolves API URLs using this **priority chain**:

### 1. Explicit URL (if provided)

```typescript
resolveApiBase({
  explicitUrl: 'https://custom-api.example.com/v1'
})
// → https://custom-api.example.com/v1
```

**Use case**: Override for testing or custom deployments.

### 2. Proxy Metadata (Context 3)

```typescript
// Reads window.__VROOLI_PROXY_INFO__
resolveApiBase()
// → https://app-monitor.example.com/apps/scenario-x/proxy
```

**Use case**: When embedded in another scenario.

### 3. Path Detection (Context 3 fallback)

```typescript
// Detects /proxy in window.location.pathname
resolveApiBase()
// → https://current-origin.com/path/to/proxy
```

**Use case**: Proxy metadata missing but URL pattern present.

### 4. Remote Origin (Context 2)

```typescript
// Detects non-localhost hostname
resolveApiBase()
// → https://my-scenario.example.com
```

**Use case**: Direct tunnel or deployed to VPS/Kubernetes.

### 5. Localhost Fallback (Context 1)

```typescript
resolveApiBase({ appendSuffix: true })
// → http://127.0.0.1:8080
```

**Use case**: Local development.

## Practical Examples

### Localhost

```typescript
// window.location: http://localhost:3000
const API_BASE = resolveApiBase({ appendSuffix: true })
console.log(API_BASE)
// → http://127.0.0.1:8080/api/v1
```

### Direct Tunnel

```typescript
// window.location: https://my-scenario.example.com
const API_BASE = resolveApiBase({ appendSuffix: true })
console.log(API_BASE)
// → https://my-scenario.example.com/api/v1
```

### Proxied

```typescript
// window.location: https://host.com/apps/my-scenario/proxy/
// window.__VROOLI_PROXY_INFO__.primary.path = '/apps/my-scenario/proxy'
const API_BASE = resolveApiBase({ appendSuffix: true })
console.log(API_BASE)
// → https://host.com/apps/my-scenario/proxy/api/v1
```

## Why This Matters

### Without Universal Resolution

Each scenario implements custom logic:

```typescript
// ❌ Fragile custom logic
let apiBase
if (window.__APP_MONITOR_PROXY_INFO__) {
  apiBase = window.__APP_MONITOR_PROXY_INFO__.primary.path
} else if (window.location.hostname !== 'localhost') {
  apiBase = window.location.origin
} else {
  apiBase = 'http://localhost:8080'
}
```

**Problems:**
- 🚫 Duplicated across scenarios
- 🚫 Easy to get wrong
- 🚫 Breaks when conventions change
- 🚫 Hard to test all paths
- 🚫 No WebSocket support

### With @vrooli/api-base

```typescript
// ✅ Universal, tested, maintained
import { resolveApiBase } from '@vrooli/api-base'

const API_BASE = resolveApiBase({ appendSuffix: true })
```

**Benefits:**
- ✅ One line of code
- ✅ Works in all contexts
- ✅ Centrally tested
- ✅ Handles edge cases
- ✅ WebSocket support included
- ✅ Backwards compatible

## Advanced: Custom Deployment Contexts

`@vrooli/api-base` supports **any** deployment pattern:

### VPS with Reverse Proxy

```
nginx (vps.example.com)
├── /scenario-a → localhost:3000
└── /scenario-b → localhost:4000
```

```typescript
// Works automatically - detects remote host
resolveApiBase()
// → https://vps.example.com
```

### Kubernetes with Ingress

```yaml
ingress:
  rules:
    - host: scenarios.k8s.example.com
      paths:
        - path: /my-scenario
          backend: my-scenario-service:3000
```

```typescript
// Detects /proxy if present, otherwise uses origin
resolveApiBase()
// → https://scenarios.k8s.example.com
```

### Custom Proxy Pattern

```typescript
// window.location: https://custom.com/embed/my-app/v1/
resolveApiBase()
// → https://custom.com/embed/my-app/v1
// (detects /v1/ as proxy pattern)
```

## Testing Across Contexts

See the [Testing Guide](../../TESTING.md) for strategies to validate your scenario in all three contexts.

## Summary

| Context | Detection Method | Example URL |
|---------|------------------|-------------|
| **Localhost** | `hostname === 'localhost'` | `http://127.0.0.1:8080/api/v1` |
| **Direct Tunnel** | Non-localhost hostname | `https://scenario.com/api/v1` |
| **Proxied** | Proxy metadata or `/proxy` in path | `https://host.com/apps/x/proxy/api/v1` |

`@vrooli/api-base` handles all three automatically, allowing you to write code once and deploy anywhere.

## Next Steps

- **[Proxy Resolution](./proxy-resolution.md)** - Deep dive into the resolution algorithm
- **[Runtime Configuration](./runtime-config.md)** - How production bundles get config
- **[WebSocket Support](./websocket-support.md)** - WS/WSS resolution patterns
