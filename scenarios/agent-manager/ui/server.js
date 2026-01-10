import { createScenarioServer, injectBaseTag, proxyWebSocketUpgrade } from '@vrooli/api-base/server'

const uiPort = process.env.UI_PORT || 3000
const apiPort = process.env.API_PORT || 8080

const app = createScenarioServer({
  uiPort,
  apiPort,
  distDir: './dist',
  serviceName: 'agent-manager',
  corsOrigins: '*',
  verbose: process.env.NODE_ENV === 'development',
  // Extended timeout for LLM-based operations (recommendation extraction)
  proxyTimeoutMs: 180000, // 3 minutes
  setupRoutes: (appInstance) => {
    appInstance.use((_req, res, next) => {
      const originalSend = res.send.bind(res)

      res.send = (body) => {
        const contentType = res.getHeader('content-type')
        const isHtml = typeof contentType === 'string' && contentType.includes('text/html')

        if (isHtml && typeof body === 'string') {
          const modifiedBody = injectBaseTag(body, '/', {
            skipIfExists: true,
            dataAttribute: 'data-agent-manager-base',
          })
          return originalSend(modifiedBody)
        }

        return originalSend(body)
      }

      next()
    })
  },
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
