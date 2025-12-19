import { createScenarioServer, proxyWebSocketUpgrade } from '@vrooli/api-base/server'

const uiPort = process.env.UI_PORT || 3000
const apiPort = process.env.API_PORT || 8080

const app = createScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'agent-manager',
  corsOrigins: '*',
  verbose: process.env.NODE_ENV === 'development',
})

const server = app.listen(uiPort, () => {
  console.log(`Agent Manager UI: http://localhost:${uiPort}`)
  console.log(`API Server: http://localhost:${apiPort}`)
  console.log(`WebSocket: ws://localhost:${uiPort}/api/v1/ws`)
})

// Handle WebSocket upgrade requests - proxy to API server
server.on('upgrade', (req, socket, head) => {
  // Proxy WebSocket connections that start with /api
  if (req.url?.startsWith('/api')) {
    proxyWebSocketUpgrade(req, socket, head, {
      apiPort,
      apiHost: '127.0.0.1',
      verbose: process.env.NODE_ENV === 'development',
    })
  } else {
    // Reject non-API WebSocket connections
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
