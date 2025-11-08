import {
  createScenarioProxyHost,
  createScenarioServer,
  createProxyMiddleware,
  injectBaseTag,
} from '@vrooli/api-base/server'
import axios from 'axios'
import http from 'http'
import path from 'path'
import { fileURLToPath } from 'url'
import { WebSocketServer } from 'ws'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const LOOPBACK_HOST = '127.0.0.1'
const LOOPBACK_HOSTS = ['127.0.0.1', 'localhost', '::1', '[::1]', '0.0.0.0']
const HOST_SCENARIO = 'app-monitor'
const CACHE_TTL_MS = 30_000
const DEFAULT_TIMEOUT_MS = 30_000
const CLIENT_DEBUG_EVENTS = []
const MAX_CLIENT_EVENTS = 250

if (!process.env.UI_PORT) {
  console.error('Error: UI_PORT environment variable is required')
  process.exit(1)
}
if (!process.env.API_PORT) {
  console.error('Error: API_PORT environment variable is required')
  process.exit(1)
}

const PORT = process.env.UI_PORT
const API_PORT = process.env.API_PORT
const API_BASE = `http://${LOOPBACK_HOST}:${API_PORT}`

const proxyHost = createScenarioProxyHost({
  hostScenario: HOST_SCENARIO,
  loopbackHosts: LOOPBACK_HOSTS,
  cacheTtlMs: CACHE_TTL_MS,
  timeoutMs: DEFAULT_TIMEOUT_MS,
  proxiedAppHeader: 'X-App-Monitor-App',
  childBaseTagAttribute: 'data-app-monitor',
  patchFetch: true,
  verbose: true,
  fetchAppMetadata: async (appId) => {
    const response = await axios.get(`${API_BASE}/api/v1/apps/${encodeURIComponent(appId)}`, {
      timeout: DEFAULT_TIMEOUT_MS,
    })
    return response.data?.data || response.data
  },
})

const additionalProxyRoutes = [
  '/scenarios',
  '/metadata',
  '/logs',
  '/health-aggregate',
  '/orchestrator',
  '/docker',
  '/timeline',
  '/resources',
]

const app = createScenarioServer({
  uiPort: PORT,
  apiPort: API_PORT,
  apiHost: LOOPBACK_HOST,
  distDir: path.join(__dirname, 'dist'),
  serviceName: 'app-monitor-ui',
  version: '1.0.0',
  corsOrigins: '*',
  verbose: true,
  setupRoutes: (expressApp) => {
    // Ensure host HTML always has base href="/" while leaving proxied apps untouched
    expressApp.use((req, res, next) => {
      if (req.path.startsWith('/apps/') && req.path.includes('/proxy')) {
        return next()
      }

      const originalSend = res.send
      res.send = function sendWithInjectedBase(body) {
        const contentType = res.getHeader('content-type')
        const isHtml = contentType && typeof contentType === 'string' && contentType.includes('text/html')

        if (isHtml && typeof body === 'string') {
          const modified = injectBaseTag(body, '/', {
            skipIfExists: true,
            dataAttribute: 'data-app-monitor-self',
          })
          return originalSend.call(this, modified)
        }

        return originalSend.call(this, body)
      }

      next()
    })

    additionalProxyRoutes.forEach((route) => {
      expressApp.use(route, createProxyMiddleware({
        apiPort: API_PORT,
        apiHost: LOOPBACK_HOST,
        timeout: DEFAULT_TIMEOUT_MS,
        verbose: true,
      }))
    })

    expressApp.post('/__debug/client-event', (req, res) => {
      const payload = req.body || {}
      const enriched = {
        event: typeof payload.event === 'string' ? payload.event : 'unknown',
        detail: payload.detail ?? null,
        timestamp: typeof payload.timestamp === 'number' ? payload.timestamp : Date.now(),
        userAgent: payload.userAgent || req.headers['user-agent'] || null,
        ip: req.ip,
      }

      CLIENT_DEBUG_EVENTS.push(enriched)
      if (CLIENT_DEBUG_EVENTS.length > MAX_CLIENT_EVENTS) {
        CLIENT_DEBUG_EVENTS.splice(0, CLIENT_DEBUG_EVENTS.length - MAX_CLIENT_EVENTS)
      }

      console.log(`[CLIENT_EVENT] ${JSON.stringify(enriched)}`)
      res.status(204).end()
    })

    expressApp.get('/__debug/client-event', (_req, res) => {
      res.json({ events: CLIENT_DEBUG_EVENTS })
    })

    expressApp.use(proxyHost.router)
  },
})

const server = http.createServer(app)
const wss = new WebSocketServer({ noServer: true })

wss.on('connection', (ws, req) => {
  const clientIp = req.socket.remoteAddress
  console.log(`[WS] Client connected from ${clientIp}`)

  ws.on('message', async (message) => {
    try {
      const data = JSON.parse(message.toString())
      console.log(`[WS] Received: ${JSON.stringify(data)}`)
      ws.send(JSON.stringify({ type: 'ack', payload: data }))
    } catch (error) {
      console.error('[WS] Error processing message:', error.message)
      ws.send(JSON.stringify({ type: 'error', message: error.message }))
    }
  })

  ws.on('close', () => {
    console.log(`[WS] Client disconnected from ${clientIp}`)
  })

  ws.on('error', (error) => {
    console.error('[WS] WebSocket error:', error.message)
  })
})

server.on('upgrade', (req, socket, head) => {
  ;(async () => {
    try {
      const url = new URL(req.url || '/', `http://${req.headers.host || 'localhost'}`)
      if (url.pathname === '/ws') {
        wss.handleUpgrade(req, socket, head, (ws) => {
          wss.emit('connection', ws, req)
        })
        return
      }

      if (await proxyHost.handleUpgrade(req, socket, head)) {
        return
      }

      socket.destroy()
    } catch (error) {
      console.error('[WS] Upgrade handling failed:', error.message)
      socket.destroy()
    }
  })()
})

server.listen(PORT, () => {
  console.log(`
╔════════════════════════════════════════════╗
║     VROOLI APP MONITOR - MATRIX UI         ║
║                                             ║
║  UI Server running on port ${PORT}            ║
║  WebSocket server active                    ║
║  API proxy to port ${API_PORT}                 ║
║                                             ║
║  Access dashboard at:                       ║
║  http://localhost:${PORT}                      ║
╚════════════════════════════════════════════╝
    `)
})

process.on('SIGTERM', () => {
  console.log('SIGTERM received, closing server...')
  server.close(() => {
    console.log('Server closed')
    process.exit(0)
  })
})
