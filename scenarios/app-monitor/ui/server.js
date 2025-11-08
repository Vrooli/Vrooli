import {
  buildProxyMetadata,
  createProxyMiddleware,
  createScenarioServer,
  injectProxyMetadata,
  injectBaseTag,
  proxyToApi,
  proxyWebSocketUpgrade,
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
const SLUGIFY_REGEX = /[^a-z0-9]+/g

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

const appMetadataCache = new Map()
const proxyContextCache = new Map()
const CLIENT_DEBUG_EVENTS = []
const MAX_CLIENT_EVENTS = 250

function normalizePortKey(value) {
  if (value === undefined || value === null) {
    return ''
  }
  if (typeof value === 'number') {
    return String(value)
  }
  return value.trim().toLowerCase().replace(SLUGIFY_REGEX, '-')
}

function slugifyPortKey(value) {
  if (!value) {
    return ''
  }
  const normalized = value.trim().toLowerCase().replace(SLUGIFY_REGEX, '-')
  return normalized || `port-${Math.random().toString(36).slice(2, 8)}`
}

function parsePortNumber(value) {
  if (value === undefined || value === null) {
    return null
  }
  const port = Number(value)
  if (!Number.isFinite(port) || port <= 0) {
    return null
  }
  return port
}

function isLikelyUiKey(key) {
  return /(ui|front|preview|web|vite)/i.test(key)
}

function buildAliases(key, slug, port) {
  const aliases = new Set()
  aliases.add(key)
  aliases.add(key.toLowerCase())
  aliases.add(key.toUpperCase())
  aliases.add(slug)
  aliases.add(slug.replace(/-/g, '_'))
  aliases.add(String(port))
  if (key.toLowerCase().includes('api')) {
    aliases.add('api')
    aliases.add('api-port')
  }
  if (key.toLowerCase().includes('ui')) {
    aliases.add('ui')
    aliases.add('ui-port')
  }
  return Array.from(aliases).filter(Boolean)
}

function extractProxyRelativeUrl(originalUrl = '/') {
  const target = originalUrl || '/'
  const marker = '/proxy'
  const queryIndex = target.indexOf('?')
  const search = queryIndex >= 0 ? target.slice(queryIndex) : ''
  const withoutQuery = queryIndex >= 0 ? target.slice(0, queryIndex) : target
  const markerIndex = withoutQuery.lastIndexOf(marker)
  if (markerIndex === -1) {
    return `/${search || ''}`
  }
  const remainder = withoutQuery.slice(markerIndex + marker.length)
  const normalizedPath = remainder && remainder.length > 0 ? (remainder.startsWith('/') ? remainder : `/${remainder}`) : '/'
  return `${normalizedPath || '/'}${search}`
}

function isApiPath(relativeUrl) {
  const pathname = (relativeUrl || '/').split('?')[0].replace(/^\/+/, '').toLowerCase()
  return pathname === 'api' || pathname.startsWith('api/')
}

function isHtmlLikeRequest(req, relativeUrl) {
  if (req.method !== 'GET') {
    return false
  }
  const acceptHeader = req.headers.accept
  if (typeof acceptHeader === 'string' && acceptHeader.includes('text/html')) {
    return true
  }
  const pathOnly = (relativeUrl || '/').split('?')[0].toLowerCase()
  return pathOnly === '/' || pathOnly.endsWith('.html') || pathOnly.endsWith('.htm')
}

async function getAppMetadata(appId) {
  const now = Date.now()
  const cached = appMetadataCache.get(appId)
  if (cached && now - cached.timestamp < CACHE_TTL_MS) {
    return cached.data
  }

  const metadataUrl = `${API_BASE}/api/v1/apps/${encodeURIComponent(appId)}`
  const response = await axios.get(metadataUrl, { timeout: DEFAULT_TIMEOUT_MS })
  const appData = response.data?.data || response.data
  if (!appData) {
    throw new Error(`App ${appId} not found`)
  }

  appMetadataCache.set(appId, { data: appData, timestamp: now })
  return appData
}

function buildProxyContext(appId, appData) {
  const portMappings = appData?.port_mappings || {}
  const entries = []

  Object.entries(portMappings).forEach(([key, value]) => {
    const port = parsePortNumber(value)
    if (!port) {
      return
    }
    entries.push({ key, port, normalized: key.toLowerCase() })
  })

  if (entries.length === 0) {
    throw new Error(`App ${appId} has no port mappings`)
  }

  let uiPort = null
  let apiPort = null

  for (const entry of entries) {
    if (!uiPort && isLikelyUiKey(entry.normalized)) {
      uiPort = entry.port
    }
    if (!apiPort && entry.normalized.includes('api')) {
      apiPort = entry.port
    }
  }

  if (!uiPort) {
    uiPort = parsePortNumber(appData?.config?.primary_port) || entries[0].port
  }

  if (!apiPort) {
    apiPort = parsePortNumber(appData?.config?.api_port) || entries.find((entry) => entry.normalized.includes('api'))?.port || uiPort
  }

  const portLookup = new Map()
  const ports = entries.map(({ key, port, normalized }) => {
    const slug = slugifyPortKey(key)
    const isPrimary = port === uiPort
    const basePath = isPrimary ? `/apps/${appId}/proxy` : `/apps/${appId}/ports/${slug}/proxy`
    const aliases = buildAliases(key, slug, port)

    aliases.forEach((alias) => {
      portLookup.set(normalizePortKey(alias), port)
    })

    return {
      appId,
      port,
      label: key,
      normalizedLabel: normalized,
      slug,
      source: 'port_mappings',
      isPrimary,
      path: basePath,
      aliases,
      assetNamespace: `${basePath}/assets`,
    }
  })

  const primaryPort = ports.find((entry) => entry.isPrimary) || ports[0]
  if (!primaryPort) {
    throw new Error(`Unable to determine primary port for ${appId}`)
  }

  const metadata = buildProxyMetadata({
    appId,
    hostScenario: HOST_SCENARIO,
    targetScenario: appData?.scenario_name || appId,
    ports,
    primaryPort,
    loopbackHosts: LOOPBACK_HOSTS,
  })

  return {
    metadata,
    uiPort,
    apiPort,
    portLookup,
  }
}

async function getProxyContext(appId) {
  const now = Date.now()
  const cached = proxyContextCache.get(appId)
  if (cached && now - cached.timestamp < CACHE_TTL_MS) {
    return cached.context
  }

  const appData = await getAppMetadata(appId)
  const context = buildProxyContext(appId, appData)
  proxyContextCache.set(appId, { context, timestamp: now })
  return context
}

function resolvePortFromKey(context, portKey) {
  if (!portKey) {
    return null
  }
  const normalized = normalizePortKey(portKey)
  return context.portLookup.get(normalized) || null
}

async function forwardHttpRequest(req, res, relativeUrl, targetPort) {
  const normalizedUrl = relativeUrl.startsWith('/') ? relativeUrl : `/${relativeUrl}`
  const proxyReq = Object.create(req)
  proxyReq.url = normalizedUrl
  await proxyToApi(proxyReq, res, normalizedUrl, {
    apiPort: targetPort,
    apiHost: LOOPBACK_HOST,
    timeout: DEFAULT_TIMEOUT_MS,
    verbose: true,
  })
}

async function proxyHtmlFromUi({ appId, uiPort, relativeUrl, req, res, metadata }) {
  const targetUrl = `http://${LOOPBACK_HOST}:${uiPort}${relativeUrl}`
  const response = await axios.get(targetUrl, {
    timeout: DEFAULT_TIMEOUT_MS,
    headers: {
      ...req.headers,
      host: `${LOOPBACK_HOST}:${uiPort}`,
    },
    responseType: 'text',
    validateStatus: () => true,
  })

  let html = response.data
  const contentType = response.headers['content-type'] || 'text/html'

  if (typeof html === 'string' && contentType.includes('text/html')) {
    const metadataPayload = {
      ...metadata,
      generatedAt: Date.now(),
    }

    html = injectProxyMetadata(html, metadataPayload, {
      patchFetch: false,
    })

    html = injectBaseTag(html, `/apps/${appId}/proxy/`, {
      skipIfExists: true,
      dataAttribute: 'data-app-monitor',
    })
  }

  res.set('Content-Type', contentType)
  res.set('Cache-Control', 'no-store, no-cache, must-revalidate, max-age=0')
  res.set('X-App-Monitor-App', appId)
  res.status(response.status).send(html)
}

async function handlePortProxyRequest(req, res) {
  const { appId, portKey } = req.params
  const relativeUrl = extractProxyRelativeUrl(req.originalUrl || req.url || '/')
  console.log(`[PORT PROXY] ${appId}/${portKey} -> ${relativeUrl}`)

  try {
    const context = await getProxyContext(appId)
    const targetPort = resolvePortFromKey(context, portKey)
    if (!targetPort) {
      res.status(404).json({ error: `Port ${portKey} not found for ${appId}` })
      return
    }

    await forwardHttpRequest(req, res, relativeUrl, targetPort)
  } catch (error) {
    console.error(`[PORT PROXY] Failed for ${appId}/${portKey}:`, error.message)
    if (!res.headersSent) {
      res.status(502).json({ error: 'Failed to proxy port', details: error.message })
    }
  }
}

async function handleScenarioProxyRequest(req, res) {
  const { appId } = req.params
  const relativeUrl = extractProxyRelativeUrl(req.originalUrl || req.url || '/')
  console.log(`[APP PROXY] ${appId} -> ${relativeUrl}`)

  try {
    const context = await getProxyContext(appId)

    if (!context.uiPort) {
      res.status(502).json({ error: 'App has no UI port configured' })
      return
    }

    if (isApiPath(relativeUrl)) {
      const targetPort = context.apiPort || context.uiPort
      await forwardHttpRequest(req, res, relativeUrl, targetPort)
      return
    }

    if (isHtmlLikeRequest(req, relativeUrl)) {
      await proxyHtmlFromUi({
        appId,
        uiPort: context.uiPort,
        relativeUrl,
        req,
        res,
        metadata: context.metadata,
      })
      return
    }

    await forwardHttpRequest(req, res, relativeUrl, context.uiPort)
  } catch (error) {
    console.error(`[APP PROXY] Error proxying ${appId}:`, error.message)
    if (!res.headersSent) {
      res.status(502).json({ error: 'Failed to proxy application', details: error.message })
    }
  }
}

async function handleProxyWebSocket(req, socket, head) {
  if (!req.url) {
    return false
  }

  const url = new URL(req.url, `http://${req.headers.host || 'localhost'}`)
  const pathname = url.pathname
  const search = url.search || ''

  const portMatch = pathname.match(/^\/apps\/([^/]+)\/ports\/([^/]+)\/proxy(\/.*)?$/)
  if (portMatch) {
    const [, appId, portKey, remainder = '/'] = portMatch
    try {
      const context = await getProxyContext(appId)
      const targetPort = resolvePortFromKey(context, portKey)
      if (!targetPort) {
        throw new Error(`Port ${portKey} not found`)
      }
      const relativeUrl = (remainder || '/') + search
      const proxyReq = Object.create(req)
      proxyReq.url = relativeUrl
      proxyWebSocketUpgrade(proxyReq, socket, head, {
        apiPort: targetPort,
        apiHost: LOOPBACK_HOST,
        verbose: true,
      })
      return true
    } catch (error) {
      console.error(`[WS PORT PROXY] ${appId}/${portKey} failed:`, error.message)
      socket.destroy()
      return true
    }
  }

  const appMatch = pathname.match(/^\/apps\/([^/]+)\/proxy(\/.*)?$/)
  if (appMatch) {
    const [, appId, remainder = '/'] = appMatch
    try {
      const context = await getProxyContext(appId)
      const relativeUrl = (remainder || '/') + search
      const targetPort = isApiPath(relativeUrl) ? context.apiPort || context.uiPort : context.uiPort
      if (!targetPort) {
        throw new Error('No target port available')
      }
      const proxyReq = Object.create(req)
      proxyReq.url = relativeUrl
      proxyWebSocketUpgrade(proxyReq, socket, head, {
        apiPort: targetPort,
        apiHost: LOOPBACK_HOST,
        verbose: true,
      })
      return true
    } catch (error) {
      console.error(`[WS APP PROXY] ${appId} failed:`, error.message)
      socket.destroy()
      return true
    }
  }

  return false
}

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
    expressApp.use((req, res, next) => {
      if (req.path.startsWith('/apps/') && req.path.includes('/proxy')) {
        return next()
      }

      const originalSend = res.send
      res.send = function sendWithBase(body) {
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

    additionalProxyRoutes.forEach((route) => {
      expressApp.use(route, createProxyMiddleware({
        apiPort: API_PORT,
        apiHost: LOOPBACK_HOST,
        timeout: DEFAULT_TIMEOUT_MS,
        verbose: true,
      }))
    })

    expressApp.all('/apps/:appId/ports/:portKey/proxy/*', (req, res) => {
      handlePortProxyRequest(req, res)
    })

    expressApp.all('/apps/:appId/proxy/*', (req, res) => {
      handleScenarioProxyRequest(req, res)
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
  (async () => {
    try {
      const url = new URL(req.url || '/', `http://${req.headers.host || 'localhost'}`)
      if (url.pathname === '/ws') {
        wss.handleUpgrade(req, socket, head, (ws) => {
          wss.emit('connection', ws, req)
        })
        return
      }

      const handled = await handleProxyWebSocket(req, socket, head)
      if (!handled) {
        socket.destroy()
      }
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
