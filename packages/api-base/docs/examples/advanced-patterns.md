# Advanced Patterns

Advanced usage patterns and real-world scenarios.

## Pattern 1: Multi-Environment Configuration

```typescript
import { resolveApiBase, getScenarioConfig } from '@vrooli/api-base'

function getApiBase(): string {
  // 1. Check for injected config (production)
  const config = getScenarioConfig()
  if (config?.apiUrl) {
    return config.apiUrl
  }

  // 2. Check for explicit override (testing)
  const explicitUrl = localStorage.getItem('API_URL_OVERRIDE')
  if (explicitUrl) {
    return explicitUrl
  }

  // 3. Environment-based resolution
  return resolveApiBase({
    defaultPort: import.meta.env.VITE_API_PORT || '8080',
    appendSuffix: true,
  })
}

export const API_BASE = getApiBase()
```

---

## Pattern 2: Retry with Fallback

```typescript
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

async function fetchWithFallback<T>(path: string): Promise<T> {
  const bases = [
    resolveApiBase({ defaultPort: '8080' }),
    resolveApiBase({ explicitUrl: 'https://api-backup.example.com' }),
  ]

  for (const base of bases) {
    try {
      const url = buildApiUrl(path, { baseUrl: base })
      const response = await fetch(url)
      if (response.ok) {
        return response.json()
      }
    } catch (error) {
      console.warn(`Failed to fetch from ${base}:`, error)
    }
  }

  throw new Error('All API endpoints failed')
}
```

---

## Pattern 3: Dynamic API Client

```typescript
import { resolveApiBase, buildApiUrl, resolveWsBase } from '@vrooli/api-base'

class APIClient {
  private httpBase: string
  private wsBase: string

  constructor(options?: { defaultPort?: string }) {
    this.httpBase = resolveApiBase({
      defaultPort: options?.defaultPort || '8080',
      appendSuffix: true,
    })

    this.wsBase = resolveWsBase({
      defaultPort: options?.defaultPort || '8080',
      appendSuffix: true,
      apiSuffix: '/ws',
    })
  }

  async get<T>(path: string): Promise<T> {
    const url = buildApiUrl(path, { baseUrl: this.httpBase })
    const response = await fetch(url)
    if (!response.ok) throw new Error(`GET ${path}: ${response.statusText}`)
    return response.json()
  }

  async post<T>(path: string, body: unknown): Promise<T> {
    const url = buildApiUrl(path, { baseUrl: this.httpBase })
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
    if (!response.ok) throw new Error(`POST ${path}: ${response.statusText}`)
    return response.json()
  }

  connect(channel: string): WebSocket {
    const url = `${this.wsBase}/${channel}`
    return new WebSocket(url)
  }
}

// Usage
const api = new APIClient({ defaultPort: '8080' })
const data = await api.get('/data')
const ws = api.connect('events')
```

---

## Pattern 4: Context-Aware Components

```typescript
import { isProxyContext, getProxyInfo } from '@vrooli/api-base'

function ContextBanner() {
  const inProxy = isProxyContext()
  const proxyInfo = getProxyInfo()

  if (!inProxy) {
    return <div className="banner">Running standalone</div>
  }

  return (
    <div className="banner banner-embedded">
      Embedded in {proxyInfo?.hostScenario || 'unknown'}
      <button onClick={() => window.parent.postMessage({ type: 'close' }, '*')}>
        Close
      </button>
    </div>
  )
}
```

---

## See Also

- [Basic Scenario Example](./basic-scenario.md)
- [Client API Reference](../api/client.md)
- [Server API Reference](../api/server.md)
