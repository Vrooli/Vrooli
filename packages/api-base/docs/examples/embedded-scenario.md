# Embedded Scenario Example

How to embed one scenario inside another using `@vrooli/api-base`.

## Overview

**Parent Scenario** (Dashboard) embeds **Child Scenario** (Widget) in an iframe with proper proxy metadata injection.

---

## Parent: Dashboard Scenario

### Inject Proxy Metadata

**dashboard/ui/server.js**:
```javascript
import { createScenarioServer, buildProxyMetadata, injectProxyMetadata } from '@vrooli/api-base/server'
import fs from 'fs'
import path from 'path'

const app = createScenarioServer({
  uiPort: process.env.UI_PORT || 3000,
  apiPort: process.env.API_PORT || 8080,
  distDir: './dist',
})

// Serve embedded widget with injected metadata
app.get('/embed/widget', async (req, res) => {
  // Read widget HTML
  const widgetHtml = fs.readFileSync(
    path.join(__dirname, '../widget/dist/index.html'),
    'utf-8'
  )

  // Build proxy metadata
  const metadata = buildProxyMetadata({
    appId: 'widget',
    hostScenario: 'dashboard',
    targetScenario: 'widget',
    ports: [
      { port: 3001, label: 'ui', slug: 'ui' },
      { port: 8081, label: 'api', slug: 'api' },
    ],
    primaryPort: { port: 3001, label: 'ui', slug: 'ui' },
    basePath: '/embed/widget/proxy',
  })

  // Inject metadata into HTML
  const modifiedHtml = injectProxyMetadata(widgetHtml, metadata)

  res.send(modifiedHtml)
})

// Proxy widget API requests
app.use('/embed/widget/proxy/api', createProxyMiddleware({
  apiPort: 8081,  // Widget's API port
  apiHost: '127.0.0.1',
}))

app.listen(3000)
```

### Embed in React Component

**dashboard/ui/src/components/WidgetEmbed.tsx**:
```typescript
import { useEffect, useRef } from 'react'

export function WidgetEmbed() {
  const iframeRef = useRef<HTMLIFrameElement>(null)

  useEffect(() => {
    // Optional: Listen for messages from widget
    const handleMessage = (event: MessageEvent) => {
      if (event.data.type === 'widget-ready') {
        console.log('Widget loaded successfully')
      }
    }

    window.addEventListener('message', handleMessage)
    return () => window.removeEventListener('message', handleMessage)
  }, [])

  return (
    <div style={{ width: '100%', height: '600px' }}>
      <iframe
        ref={iframeRef}
        src="/embed/widget"
        style={{
          width: '100%',
          height: '100%',
          border: 'none',
        }}
        title="Widget"
      />
    </div>
  )
}
```

---

## Child: Widget Scenario

Widget uses standard API resolution - **no changes needed**!

**widget/ui/src/api.ts**:
```typescript
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

// This automatically detects proxy context
const API_BASE = resolveApiBase({
  appendSuffix: true,
})

export async function fetchWidgetData() {
  const url = buildApiUrl('/data', { baseUrl: API_BASE })
  const response = await fetch(url)
  return response.json()
}
```

**How it works**:
1. Widget loaded at: `http://localhost:3000/embed/widget`
2. Proxy metadata injected: `window.__VROOLI_PROXY_INFO__`
3. `resolveApiBase()` detects metadata
4. API requests go to: `http://localhost:3000/embed/widget/proxy/api/v1/data`
5. Dashboard proxies to widget API: `http://127.0.0.1:8081/api/v1/data`

---

## See Also

- [Basic Scenario Example](./basic-scenario.md)
- [Proxy Resolution](../concepts/proxy-resolution.md)
- [Server API: injectProxyMetadata](../api/server.md#injectproxymetadata)
