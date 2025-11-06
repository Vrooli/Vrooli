# Basic Scenario Example

Complete example of a simple scenario using `@vrooli/api-base`.

## Overview

This example shows how to set up a basic scenario with:
- React frontend (Vite)
- Go API backend
- Production bundle serving
- Proper API resolution in all contexts

---

## Project Structure

```
my-scenario/
├── api/
│   ├── main.go                 # Go API server
│   └── go.mod
├── ui/
│   ├── dist/                   # Built React app
│   ├── src/
│   │   ├── App.tsx            # React component
│   │   ├── api.ts              # API client
│   │   └── main.tsx            # Entry point
│   ├── index.html
│   ├── package.json
│   ├── server.js               # Production server
│   ├── tsconfig.json
│   └── vite.config.ts
└── .vrooli/
    └── service.json            # Vrooli configuration
```

---

## Step 1: API Server (Go)

**api/main.go**:
```go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type DataResponse struct {
	Message string   `json:"message"`
	Items   []string `json:"items"`
}

func main() {
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	// Health endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthResponse{
			Status:  "healthy",
			Service: "my-scenario-api",
		})
	})

	// API endpoints
	http.HandleFunc("/api/v1/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DataResponse{
			Message: "Hello from API!",
			Items:   []string{"Item 1", "Item 2", "Item 3"},
		})
	})

	log.Printf("API server listening on :%s", apiPort)
	log.Fatal(http.ListenAndServe(":"+apiPort, nil))
}
```

---

## Step 2: React Frontend

**ui/src/api.ts**:
```typescript
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

// Resolve API base URL (works in all contexts)
const API_BASE = resolveApiBase({
  defaultPort: '8080',
  appendSuffix: true,
  apiSuffix: '/api/v1',
})

export interface DataResponse {
  message: string
  items: string[]
}

export async function fetchData(): Promise<DataResponse> {
  const url = buildApiUrl('/data', { baseUrl: API_BASE })
  const response = await fetch(url)

  if (!response.ok) {
    throw new Error(`API error: ${response.statusText}`)
  }

  return response.json()
}

export async function checkHealth(): Promise<boolean> {
  try {
    // Health endpoint is at root, not /api/v1
    const url = buildApiUrl('/health', {
      baseUrl: resolveApiBase({ defaultPort: '8080' })
    })

    const response = await fetch(url)
    return response.ok
  } catch {
    return false
  }
}
```

**ui/src/App.tsx**:
```typescript
import { useEffect, useState } from 'react'
import { fetchData, checkHealth, type DataResponse } from './api'

function App() {
  const [data, setData] = useState<DataResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [healthy, setHealthy] = useState(false)

  useEffect(() => {
    // Check API health
    checkHealth().then(setHealthy)

    // Fetch data
    fetchData()
      .then(setData)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return <div>Loading...</div>
  }

  if (error) {
    return <div>Error: {error}</div>
  }

  return (
    <div>
      <h1>My Scenario</h1>

      <div style={{ marginBottom: '20px' }}>
        <strong>API Status:</strong>{' '}
        <span style={{ color: healthy ? 'green' : 'red' }}>
          {healthy ? 'Healthy ✓' : 'Unhealthy ✗'}
        </span>
      </div>

      {data && (
        <>
          <h2>{data.message}</h2>
          <ul>
            {data.items.map((item, i) => (
              <li key={i}>{item}</li>
            ))}
          </ul>
        </>
      )}
    </div>
  )
}

export default App
```

**ui/src/main.tsx**:
```typescript
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
```

---

## Step 3: Production Server

**ui/server.js**:
```javascript
import { startScenarioServer } from '@vrooli/api-base/server'

startScenarioServer({
  // Ports from environment
  uiPort: process.env.UI_PORT || 3000,
  apiPort: process.env.API_PORT || 8080,

  // Serve built files
  distDir: './dist',

  // Service info
  serviceName: 'my-scenario-ui',
  version: '1.0.0',

  // Enable logging in development
  verbose: process.env.NODE_ENV === 'development',

  // Custom configuration
  configBuilder: (env) => ({
    apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
    wsUrl: `ws://localhost:${env.API_PORT}/ws`,
    apiPort: String(env.API_PORT),
    uiPort: String(env.UI_PORT),
  }),
})
```

---

## Step 4: Package Configuration

**ui/package.json**:
```json
{
  "name": "my-scenario-ui",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "node server.js"
  },
  "dependencies": {
    "@vrooli/api-base": "workspace:*",
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "@types/node": "^20.10.0",
    "@types/react": "^18.2.43",
    "@types/react-dom": "^18.2.17",
    "@vitejs/plugin-react": "^4.2.1",
    "express": "^4.18.2",
    "typescript": "^5.3.3",
    "vite": "^5.0.8"
  }
}
```

**ui/vite.config.ts**:
```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

---

## Step 5: Vrooli Service Configuration

**.vrooli/service.json**:
```json
{
  "id": "my-scenario",
  "name": "My Scenario",
  "version": "1.0.0",
  "description": "Basic scenario example",

  "setup": {
    "steps": [
      {
        "name": "install-ui-deps",
        "run": "cd ui && pnpm install",
        "description": "Install UI dependencies"
      },
      {
        "name": "build-ui",
        "run": "cd ui && pnpm run build",
        "description": "Build production UI bundle"
      },
      {
        "name": "build-api",
        "run": "cd api && go build -o my-scenario-api",
        "description": "Build API binary"
      }
    ]
  },

  "develop": {
    "steps": [
      {
        "name": "api",
        "run": "cd api && go run main.go",
        "description": "Run API server",
        "env": {
          "API_PORT": "8080"
        }
      },
      {
        "name": "ui",
        "run": "cd ui && pnpm run dev",
        "description": "Run Vite dev server",
        "env": {
          "UI_PORT": "3000"
        }
      }
    ]
  },

  "start": {
    "steps": [
      {
        "name": "api",
        "run": "./api/my-scenario-api",
        "description": "Start API server",
        "env": {
          "API_PORT": "{{api_port}}"
        },
        "health": {
          "check": "http",
          "url": "http://localhost:{{api_port}}/health"
        }
      },
      {
        "name": "ui",
        "run": "cd ui && node server.js",
        "description": "Start UI server",
        "env": {
          "UI_PORT": "{{ui_port}}",
          "API_PORT": "{{api_port}}"
        },
        "health": {
          "check": "http",
          "url": "http://localhost:{{ui_port}}/health"
        }
      }
    ]
  },

  "environment": {
    "ui_port": {
      "type": "port",
      "label": "UI Port",
      "primary": true
    },
    "api_port": {
      "type": "port",
      "label": "API Port"
    }
  }
}
```

---

## Usage

### Development Mode

```bash
# Setup (build API, install deps, build UI)
vrooli scenario setup my-scenario

# Start development servers (Vite + Go API)
vrooli develop my-scenario

# Open browser
# → http://localhost:3000
```

### Production Mode

```bash
# Start production servers (built bundle + Go API)
vrooli scenario start my-scenario

# Check status
vrooli scenario status my-scenario

# View logs
vrooli scenario logs my-scenario

# Stop servers
vrooli scenario stop my-scenario
```

---

## Testing in All Contexts

### 1. Localhost

```bash
vrooli scenario start my-scenario
# → http://localhost:3000
# API requests: http://127.0.0.1:8080/api/v1/data
```

### 2. Direct Tunnel

```bash
# Setup Cloudflare tunnel
cloudflared tunnel --url http://localhost:3000

# → https://random-slug.trycloudflare.com
# API requests: https://random-slug.trycloudflare.com/api/v1/data
```

### 3. App-Monitor Proxy

```bash
# Start app-monitor
vrooli scenario start app-monitor

# Open app-monitor
# → http://localhost:8110

# Navigate to your scenario
# → http://localhost:8110/apps/my-scenario/proxy/
# API requests: http://localhost:8110/apps/my-scenario/proxy/api/v1/data
```

---

## Key Takeaways

### ✅ What Makes This Work

1. **Universal API Resolution**:
   ```typescript
   const API_BASE = resolveApiBase({
     defaultPort: '8080',
     appendSuffix: true,
   })
   ```
   - Works in localhost (uses defaultPort)
   - Works in tunnel (uses window.location.origin)
   - Works in proxy (uses proxy metadata)

2. **Server-Side Proxying**:
   ```javascript
   startScenarioServer({
     apiPort: process.env.API_PORT || 8080,
     // ... automatically proxies /api/* requests
   })
   ```
   - UI server proxies API requests
   - No CORS issues
   - Works in all contexts

3. **Config Injection**:
   ```javascript
   configBuilder: (env) => ({
     apiUrl: `http://localhost:${env.API_PORT}/api/v1`,
     // ... injected into HTML
   })
   ```
   - Available immediately (no fetch needed)
   - Works offline
   - Type-safe

---

## Next Steps

- **Add WebSockets**: See [WebSocket Support](../concepts/websocket-support.md)
- **Embed in Another Scenario**: See [Embedded Scenario Example](./embedded-scenario.md)
- **Custom Proxy Pattern**: See [Custom Proxy Example](./custom-proxy.md)
- **Advanced Patterns**: See [Advanced Patterns](./advanced-patterns.md)

---

## See Also

- [Quick Start Guide](../guides/quick-start.md)
- [Client API Reference](../api/client.md)
- [Server API Reference](../api/server.md)
- [Proxy Resolution](../concepts/proxy-resolution.md)
