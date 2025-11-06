# Complete WebSocket Example

End-to-end example of a scenario with WebSocket support using `@vrooli/api-base`.

## Overview

This example shows how to build a real-time chat application with:
- React frontend with WebSocket connection
- Go API backend with WebSocket endpoint
- Proper proxy configuration for all deployment contexts
- Reconnection logic and error handling

---

## Project Structure

```
chat-scenario/
├── api/
│   ├── main.go                 # Go WebSocket server
│   └── go.mod
├── ui/
│   ├── dist/                   # Built React app
│   ├── src/
│   │   ├── App.tsx            # Main component
│   │   ├── useWebSocket.ts    # WebSocket hook
│   │   ├── api.ts             # API client
│   │   └── main.tsx           # Entry point
│   ├── index.html
│   ├── package.json
│   ├── server.js              # Production server with WebSocket proxy
│   └── vite.config.ts
└── .vrooli/
    └── service.json           # Vrooli configuration
```

---

## Step 1: API Server with WebSocket (Go)

**api/main.go**:
```go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message represents a chat message
type Message struct {
	Type      string    `json:"type"`
	User      string    `json:"user"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// In production, validate origin properly
		return true
	},
}

// Hub manages WebSocket connections
type Hub struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan Message
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			count := len(h.clients)
			h.mu.Unlock()
			log.Printf("Client connected. Total: %d", count)

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			count := len(h.clients)
			h.mu.Unlock()
			log.Printf("Client disconnected. Total: %d", count)

		case message := <-h.broadcast:
			h.mu.RLock()
			for conn := range h.clients {
				err := conn.WriteJSON(message)
				if err != nil {
					log.Printf("Write error: %v", err)
					conn.Close()
					delete(h.clients, conn)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func main() {
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}

	hub := newHub()
	go hub.run()

	// Health endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"service": "chat-api",
		})
	})

	// WebSocket endpoint
	http.HandleFunc("/api/v1/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})

	log.Printf("Chat API listening on :%s", apiPort)
	log.Printf("WebSocket endpoint: /api/v1/ws")
	log.Fatal(http.ListenAndServe(":"+apiPort, nil))
}

func handleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}

	hub.register <- conn

	// Send welcome message
	welcomeMsg := Message{
		Type:      "system",
		User:      "System",
		Text:      "Welcome to the chat!",
		Timestamp: time.Now(),
	}
	conn.WriteJSON(welcomeMsg)

	// Handle incoming messages
	go func() {
		defer func() {
			hub.unregister <- conn
		}()

		for {
			var msg Message
			err := conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Read error: %v", err)
				}
				break
			}

			// Add timestamp
			msg.Timestamp = time.Now()

			// Handle ping messages
			if msg.Type == "ping" {
				pong := Message{
					Type:      "pong",
					Timestamp: time.Now(),
				}
				conn.WriteJSON(pong)
				continue
			}

			// Broadcast message to all clients
			log.Printf("Broadcasting: %s: %s", msg.User, msg.Text)
			hub.broadcast <- msg
		}
	}()
}
```

**api/go.mod**:
```go
module chat-scenario

go 1.21

require github.com/gorilla/websocket v1.5.1
```

---

## Step 2: React Frontend

**ui/src/useWebSocket.ts**:
```typescript
import { useEffect, useRef, useState, useCallback } from 'react'

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

interface UseWebSocketOptions {
  url: string
  onMessage: (data: any) => void
  enabled?: boolean
  reconnectInterval?: number
  maxReconnectAttempts?: number
}

interface UseWebSocketReturn {
  status: ConnectionStatus
  error: Error | null
  send: (data: any) => void
  reconnect: () => void
}

export function useWebSocket({
  url,
  onMessage,
  enabled = true,
  reconnectInterval = 5000,
  maxReconnectAttempts = Infinity,
}: UseWebSocketOptions): UseWebSocketReturn {
  const [status, setStatus] = useState<ConnectionStatus>('disconnected')
  const [error, setError] = useState<Error | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const onMessageRef = useRef(onMessage)

  // Keep onMessage ref fresh
  useEffect(() => {
    onMessageRef.current = onMessage
  }, [onMessage])

  const connect = useCallback(() => {
    if (!enabled || wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    try {
      setStatus('connecting')
      setError(null)

      console.log(`[WebSocket] Connecting to ${url}`)
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        console.log('[WebSocket] Connected')
        setStatus('connected')
        setError(null)
        reconnectAttemptsRef.current = 0
      }

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          onMessageRef.current(data)
        } catch (err) {
          console.error('[WebSocket] Failed to parse message:', err)
        }
      }

      ws.onerror = () => {
        const errorObj = new Error('WebSocket connection error')
        console.error('[WebSocket]', errorObj)
        setError(errorObj)
        setStatus('error')
      }

      ws.onclose = () => {
        console.log('[WebSocket] Connection closed')
        setStatus('disconnected')
        wsRef.current = null

        // Attempt to reconnect if enabled and within retry limits
        if (enabled && reconnectAttemptsRef.current < maxReconnectAttempts) {
          reconnectAttemptsRef.current += 1
          const attempt = reconnectAttemptsRef.current

          console.log(`[WebSocket] Reconnecting in ${reconnectInterval}ms (attempt ${attempt})`)

          // Exponential backoff with jitter
          const backoff = Math.min(
            reconnectInterval * Math.pow(1.5, attempt - 1),
            30000 // Max 30 seconds
          )
          const jitter = Math.random() * 1000
          const delay = backoff + jitter

          reconnectTimeoutRef.current = window.setTimeout(() => {
            if (enabled) {
              connect()
            }
          }, delay)
        }
      }
    } catch (err) {
      const errorObj = err instanceof Error ? err : new Error(String(err))
      console.error('[WebSocket] Connection failed:', errorObj)
      setError(errorObj)
      setStatus('error')
    }
  }, [url, enabled, reconnectInterval, maxReconnectAttempts])

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current !== null) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }

    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }

    reconnectAttemptsRef.current = 0
    setStatus('disconnected')
  }, [])

  const send = useCallback((data: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(data))
    } else {
      console.warn('[WebSocket] Cannot send, not connected')
    }
  }, [])

  const reconnect = useCallback(() => {
    disconnect()
    connect()
  }, [disconnect, connect])

  useEffect(() => {
    if (enabled) {
      connect()
    }

    return () => {
      disconnect()
    }
  }, [enabled, url, connect, disconnect])

  return {
    status,
    error,
    send,
    reconnect,
  }
}
```

**ui/src/App.tsx**:
```typescript
import { useState, useCallback } from 'react'
import { resolveWsBase } from '@vrooli/api-base'
import { useWebSocket } from './useWebSocket'

interface Message {
  type: string
  user: string
  text: string
  timestamp: string
}

function App() {
  const [messages, setMessages] = useState<Message[]>([])
  const [username, setUsername] = useState('User' + Math.floor(Math.random() * 1000))
  const [messageText, setMessageText] = useState('')

  // Resolve WebSocket URL using api-base
  const wsUrl = resolveWsBase({
    appendSuffix: true,
    apiSuffix: '/api/v1/ws',
  })

  const handleMessage = useCallback((data: Message) => {
    // Skip pong messages
    if (data.type === 'pong') {
      return
    }

    setMessages((prev) => [...prev, data])
  }, [])

  const { status, send, reconnect } = useWebSocket({
    url: wsUrl,
    onMessage: handleMessage,
    enabled: true,
  })

  const sendMessage = () => {
    if (!messageText.trim()) {
      return
    }

    send({
      type: 'message',
      user: username,
      text: messageText,
    })

    setMessageText('')
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage()
    }
  }

  return (
    <div style={{ maxWidth: '800px', margin: '20px auto', padding: '20px' }}>
      <h1>Real-Time Chat</h1>

      {/* Connection Status */}
      <div style={{
        padding: '10px',
        marginBottom: '20px',
        borderRadius: '5px',
        backgroundColor: status === 'connected' ? '#d4edda' : '#f8d7da',
        border: `1px solid ${status === 'connected' ? '#c3e6cb' : '#f5c6cb'}`,
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
      }}>
        <div>
          <strong>Status:</strong> {status}
        </div>
        {status !== 'connected' && (
          <button onClick={reconnect}>Reconnect</button>
        )}
      </div>

      {/* Username */}
      <div style={{ marginBottom: '20px' }}>
        <label>
          <strong>Username:</strong>{' '}
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            style={{ marginLeft: '10px', padding: '5px' }}
          />
        </label>
      </div>

      {/* Messages */}
      <div style={{
        height: '400px',
        overflowY: 'auto',
        border: '1px solid #ccc',
        borderRadius: '5px',
        padding: '10px',
        marginBottom: '20px',
        backgroundColor: '#f9f9f9',
      }}>
        {messages.length === 0 && (
          <div style={{ color: '#999', textAlign: 'center', padding: '20px' }}>
            No messages yet. Send a message to get started!
          </div>
        )}
        {messages.map((msg, idx) => (
          <div key={idx} style={{
            marginBottom: '10px',
            padding: '10px',
            borderRadius: '5px',
            backgroundColor: msg.type === 'system' ? '#e7f3ff' : '#fff',
            border: '1px solid #ddd',
          }}>
            <div style={{ fontSize: '12px', color: '#666', marginBottom: '5px' }}>
              <strong>{msg.user}</strong> · {new Date(msg.timestamp).toLocaleTimeString()}
            </div>
            <div>{msg.text}</div>
          </div>
        ))}
      </div>

      {/* Message Input */}
      <div style={{ display: 'flex', gap: '10px' }}>
        <input
          type="text"
          value={messageText}
          onChange={(e) => setMessageText(e.target.value)}
          onKeyPress={handleKeyPress}
          placeholder="Type a message..."
          disabled={status !== 'connected'}
          style={{
            flex: 1,
            padding: '10px',
            borderRadius: '5px',
            border: '1px solid #ccc',
          }}
        />
        <button
          onClick={sendMessage}
          disabled={status !== 'connected' || !messageText.trim()}
          style={{
            padding: '10px 20px',
            borderRadius: '5px',
            border: 'none',
            backgroundColor: status === 'connected' ? '#007bff' : '#ccc',
            color: 'white',
            cursor: status === 'connected' ? 'pointer' : 'not-allowed',
          }}
        >
          Send
        </button>
      </div>

      {/* Debug Info */}
      <div style={{ marginTop: '20px', fontSize: '12px', color: '#666' }}>
        <strong>WebSocket URL:</strong> {wsUrl}
      </div>
    </div>
  )
}

export default App
```

---

## Step 3: Production Server with WebSocket Proxy

**ui/server.js**:
```javascript
import { createScenarioServer, proxyWebSocketUpgrade } from '@vrooli/api-base/server'

const uiPort = process.env.UI_PORT || 3000
const apiPort = process.env.API_PORT || 8080

const app = createScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'chat-scenario',
  version: '1.0.0',
  verbose: true, // Enable to debug WebSocket connections
})

const server = app.listen(uiPort, () => {
  console.log(`\n🚀 Chat Scenario UI: http://localhost:${uiPort}`)
  console.log(`📡 API Server: http://localhost:${apiPort}`)
  console.log(`🔌 WebSocket: ws://localhost:${uiPort}/api/v1/ws\n`)
})

// CRITICAL: Handle WebSocket upgrade requests
server.on('upgrade', (req, socket, head) => {
  console.log(`[upgrade] Request for: ${req.url}`)

  // Proxy WebSocket connections that start with /api
  if (req.url?.startsWith('/api')) {
    proxyWebSocketUpgrade(req, socket, head, {
      apiPort,
      apiHost: '127.0.0.1',
      verbose: true, // See all WebSocket proxy activity
    })
  } else {
    console.log(`[upgrade] Rejecting non-API upgrade request: ${req.url}`)
    socket.destroy()
  }
})

// Graceful shutdown
process.on('SIGTERM', () => {
  console.log('SIGTERM received, closing server...')
  server.close(() => {
    console.log('Server closed')
    process.exit(0)
  })
})
```

---

## Step 4: Package Configuration

**ui/package.json**:
```json
{
  "name": "chat-scenario-ui",
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
      // Proxy WebSocket and HTTP requests to API
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true, // Enable WebSocket proxying
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
  "id": "chat-scenario",
  "name": "Real-Time Chat",
  "version": "1.0.0",
  "description": "WebSocket-based chat application",

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
        "name": "install-go-deps",
        "run": "cd api && go mod download",
        "description": "Install Go dependencies"
      },
      {
        "name": "build-api",
        "run": "cd api && go build -o chat-api",
        "description": "Build API binary"
      }
    ]
  },

  "develop": {
    "steps": [
      {
        "name": "api",
        "run": "cd api && go run main.go",
        "description": "Run API server with WebSocket support",
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
        "run": "./api/chat-api",
        "description": "Start WebSocket API server",
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
        "description": "Start UI server with WebSocket proxy",
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
# Setup (install deps, build)
vrooli scenario setup chat-scenario

# Start development servers
vrooli develop chat-scenario

# Open browser
# → http://localhost:3000
```

### Production Mode

```bash
# Start production servers
vrooli scenario start chat-scenario

# Check status
vrooli scenario status chat-scenario

# View logs (API)
vrooli scenario logs chat-scenario --step api

# View logs (UI with WebSocket proxy)
vrooli scenario logs chat-scenario --step ui

# Stop servers
vrooli scenario stop chat-scenario
```

---

## Testing WebSocket Connection

### Manual Testing

1. **Open multiple browser tabs** to `http://localhost:3000`
2. **Send messages** from one tab
3. **Verify messages appear in all tabs** (real-time broadcast)

### Using Browser Console

```javascript
// Check WebSocket URL resolution
import { resolveWsBase } from '@vrooli/api-base'
const wsUrl = resolveWsBase({ appendSuffix: true, apiSuffix: '/api/v1/ws' })
console.log('WebSocket URL:', wsUrl)

// Manual WebSocket connection test
const ws = new WebSocket(wsUrl)

ws.onopen = () => console.log('Connected!')
ws.onmessage = (event) => console.log('Received:', event.data)
ws.onerror = (error) => console.error('Error:', error)

// Send test message
ws.send(JSON.stringify({
  type: 'message',
  user: 'TestUser',
  text: 'Hello from console!'
}))

// Close connection
ws.close()
```

### Using curl

```bash
# Test API health
curl http://localhost:8080/health

# Test WebSocket upgrade (requires wscat)
npm install -g wscat
wscat -c ws://localhost:3000/api/v1/ws

# Send message (after connection established)
> {"type":"message","user":"CLI","text":"Hello from terminal"}
```

---

## Troubleshooting

### Connection Fails

1. **Check server logs** with verbose enabled
2. **Verify upgrade handler** in server.js
3. **Test API directly**: `curl http://localhost:8080/api/v1/ws`
4. **Check browser console** for exact error

### Messages Not Appearing

1. **Check message handler** in useWebSocket hook
2. **Verify broadcast logic** in Go server
3. **Test with multiple browser tabs**

### Reconnection Issues

1. **Adjust reconnectInterval** in useWebSocket options
2. **Check maxReconnectAttempts** setting
3. **Implement ping/pong** for keep-alive

---

## Key Takeaways

### ✅ What Makes This Work

1. **Universal WebSocket Resolution**:
   ```typescript
   const wsUrl = resolveWsBase({ appendSuffix: true, apiSuffix: '/api/v1/ws' })
   ```
   - Works in localhost, tunnel, and proxy contexts
   - Automatic protocol (ws:// or wss://)
   - No hardcoded URLs

2. **Server Upgrade Handler**:
   ```javascript
   server.on('upgrade', (req, socket, head) => {
     if (req.url?.startsWith('/api')) {
       proxyWebSocketUpgrade(req, socket, head, { apiPort })
     }
   })
   ```
   - Required for WebSocket proxy
   - Forwards upgrade requests to API
   - Preserves WebSocket headers

3. **Reconnection Logic**:
   ```typescript
   // Exponential backoff with jitter
   const backoff = Math.min(
     reconnectInterval * Math.pow(1.5, attempt - 1),
     30000
   )
   ```
   - Automatic reconnection on disconnect
   - Prevents thundering herd
   - Configurable retry limits

4. **Proper Cleanup**:
   ```typescript
   return () => {
     console.log('Cleaning up WebSocket')
     ws.close()
   }
   ```
   - Prevents connection leaks
   - React useEffect cleanup
   - Graceful shutdown

---

## Deployment Contexts

This example works in all three contexts without any code changes:

### 1. Localhost
```
http://localhost:3000
ws://localhost:3000/api/v1/ws
```

### 2. Direct Tunnel
```
https://chat.example.com
wss://chat.example.com/api/v1/ws
```

### 3. Proxied/Embedded
```
https://app-monitor.com/apps/chat/proxy/
wss://app-monitor.com/apps/chat/proxy/api/v1/ws
```

---

## Next Steps

- **Add Authentication**: Implement token-based auth for WebSocket connections
- **Add Persistence**: Store messages in database
- **Add Rooms**: Support multiple chat rooms
- **Add Typing Indicators**: Show when users are typing
- **Add File Sharing**: Support image/file uploads
- **Add Rate Limiting**: Prevent spam/abuse

---

## See Also

- [WebSocket Support](../concepts/websocket-support.md)
- [Basic Scenario Example](./basic-scenario.md)
- [Troubleshooting Guide](../concepts/websocket-support.md#troubleshooting)
- [Integration Tests](../../src/__tests__/integration/websocket-proxy.test.ts)
