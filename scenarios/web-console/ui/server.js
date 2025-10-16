import { createServer, request as httpRequest, STATUS_CODES } from 'http'
import { createReadStream, existsSync } from 'fs'
import { promises as fs } from 'fs'
import path from 'path'
import { WebSocketServer, WebSocket } from 'ws'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const { UI_PORT, API_PORT } = process.env

if (!UI_PORT) {
  console.error('Web Console UI requires UI_PORT environment variable')
  process.exit(1)
}

if (!API_PORT) {
  console.error('Web Console UI requires API_PORT environment variable')
  process.exit(1)
}

const distRoot = path.join(__dirname, 'dist')
const publicRoot = path.join(__dirname, 'public')

const MIME_TYPES = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.ico': 'image/x-icon',
  '.jpeg': 'image/jpeg',
  '.jpg': 'image/jpeg',
  '.js': 'application/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.png': 'image/png',
  '.svg': 'image/svg+xml'
}

function contentTypeFor(filePath) {
  const ext = path.extname(filePath).toLowerCase()
  return MIME_TYPES[ext] || 'application/octet-stream'
}

function cacheControlFor(filePath) {
  const ext = path.extname(filePath).toLowerCase()
  if (ext === '.html') {
    return 'no-store'
  }
  if (ext === '.js' || ext === '.css') {
    return 'public, max-age=31536000, immutable'
  }
  return 'public, max-age=86400'
}

function sanitizeCloseCode(code, fallback = 1000) {
  return Number.isInteger(code) && code >= 1000 && code <= 4999 ? code : fallback
}

function sanitizeCloseReason(reason) {
  if (typeof reason !== 'string' || reason.length === 0) {
    return undefined
  }
  const encoder = new TextEncoder()
  const bytes = encoder.encode(reason)
  if (bytes.length <= 123) {
    return reason
  }
  const decoder = new TextDecoder()
  return decoder.decode(bytes.slice(0, 123))
}

function safeClose(socket, code, reason) {
  if (!socket || socket.readyState === WebSocket.CLOSED) {
    return
  }
  try {
    if (code === undefined && reason === undefined) {
      socket.close()
    } else {
      socket.close(code, reason)
    }
  } catch (error) {
    console.error('[ws-proxy] close error', error.message)
    if (typeof socket.terminate === 'function') {
      socket.terminate()
    }
  }
}

function buildForwardedFor(req) {
  const prior = req.headers['x-forwarded-for']
  const remote = req.socket.remoteAddress
  return prior ? `${prior}, ${remote}` : remote
}

function proxyToApi(req, res, apiPath) {
  const proxyReq = httpRequest(
    {
      hostname: 'localhost',
      port: API_PORT,
      path: apiPath,
      method: req.method,
      headers: {
        ...req.headers,
        host: `localhost:${API_PORT}`,
        'x-forwarded-for': buildForwardedFor(req),
        'x-forwarded-proto': req.headers['x-forwarded-proto'] || (req.socket.encrypted ? 'https' : 'http'),
        'x-forwarded-host': req.headers['host'] || ''
      }
    },
    (proxyRes) => {
      res.statusCode = proxyRes.statusCode || 500
      for (const [key, value] of Object.entries(proxyRes.headers)) {
        if (value !== undefined) {
          res.setHeader(key, value)
        }
      }
      proxyRes.pipe(res)
    }
  )

  proxyReq.on('error', (error) => {
    console.error('[proxy] API request failed', error.message)
    if (!res.headersSent) {
      res.statusCode = 502
      res.setHeader('Content-Type', 'application/json')
      res.end(
        JSON.stringify({
          error: 'API_UNAVAILABLE',
          message: error.message,
          target: `http://localhost:${API_PORT}${apiPath}`
        })
      )
    } else {
      res.end()
    }
  })

  if (req.method === 'GET' || req.method === 'HEAD') {
    proxyReq.end()
  } else {
    req.pipe(proxyReq)
  }
}

async function serveStatic(req, res) {
  const urlPath = (req.url || '/').split('?')[0]
  const safePath = urlPath === '/' ? '/index.html' : urlPath
  const normalized = path.normalize(safePath).replace(/^\.\.+/, '')
  const candidatePaths = [
    path.join(distRoot, normalized),
    path.join(publicRoot, normalized)
  ]

  for (const filePath of candidatePaths) {
    try {
      const stat = await fs.stat(filePath)
      if (stat.isDirectory()) {
        continue
      }

      if (path.basename(filePath) === 'index.html') {
        const html = await fs.readFile(filePath, 'utf-8')
        res.setHeader('Content-Type', contentTypeFor(filePath))
        res.setHeader('Cache-Control', cacheControlFor(filePath))
        res.end(html)
        return
      }

      res.setHeader('Content-Type', contentTypeFor(filePath))
      res.setHeader('Cache-Control', cacheControlFor(filePath))
      const stream = createReadStream(filePath)
      stream.on('error', (err) => {
        console.error('[static] stream error', err.message)
        serveFallback(res).catch((fallbackError) => {
          console.error('[static] fallback error', fallbackError.message)
          if (!res.headersSent) {
            res.statusCode = 500
            res.setHeader('Content-Type', 'text/plain; charset=utf-8')
            res.end(STATUS_CODES[500])
          }
        })
      })
      stream.pipe(res)
      return
    } catch (error) {
      if (error.code !== 'ENOENT') {
        console.error('[static] error', error.message)
        res.statusCode = 500
        res.setHeader('Content-Type', 'text/plain; charset=utf-8')
        res.end(STATUS_CODES[500])
        return
      }
    }
  }

  await serveFallback(res)
}

async function serveFallback(res) {
  const fallbackPaths = [
    path.join(distRoot, 'index.html'),
    path.join(publicRoot, 'index.html')
  ]

  for (const fallback of fallbackPaths) {
    if (!existsSync(fallback)) {
      continue
    }
    try {
      const html = await fs.readFile(fallback, 'utf-8')
      res.setHeader('Content-Type', contentTypeFor(fallback))
      res.setHeader('Cache-Control', cacheControlFor(fallback))
      res.end(html)
      return
    } catch (error) {
      console.error('[static] fallback read error', error.message)
      res.statusCode = 500
      res.setHeader('Content-Type', 'text/plain; charset=utf-8')
      res.end(STATUS_CODES[500])
      return
    }
  }

  res.statusCode = 404
  res.setHeader('Content-Type', 'text/plain; charset=utf-8')
  res.end('Web Console UI assets not built yet. Run the Vite build before starting the server.')
}

const httpServer = createServer((req, res) => {
  if (!req.url) {
    res.statusCode = 400
    res.end('Bad Request')
    return
  }

  if (req.url.startsWith('/api/')) {
    proxyToApi(req, res, req.url)
    return
  }

  if (req.url === '/api' || req.url === '/api/') {
    proxyToApi(req, res, '/api/')
    return
  }

  if (req.url === '/health') {
    proxyToApi(req, res, '/healthz')
    return
  }

  if (req.method === 'GET' || req.method === 'HEAD') {
    serveStatic(req, res)
    return
  }

  res.statusCode = 404
  res.end('Not found')
})

const wsBridge = new WebSocketServer({ noServer: true })

wsBridge.on('connection', (client, req) => {
  const upstreamPath = req.url.replace(/^\/ws/, '/api/v1')
  const targetUrl = `ws://localhost:${API_PORT}${upstreamPath}`
  const upstream = new WebSocket(targetUrl, {
    headers: {
      'x-forwarded-for': buildForwardedFor(req),
      'x-forwarded-proto': 'ws',
      'x-forwarded-host': req.headers.host || ''
    }
  })

  upstream.on('open', () => {
    if (client.readyState === WebSocket.OPEN) {
      client.send(JSON.stringify({ type: 'status', payload: { status: 'upstream_connected' } }))
    }
  })

  upstream.on('message', (data) => {
    if (client.readyState === WebSocket.OPEN) {
      client.send(data)
    }
  })

  upstream.on('close', (code, reason) => {
    if (client.readyState === WebSocket.OPEN) {
      safeClose(client, sanitizeCloseCode(code), sanitizeCloseReason(reason))
    }
  })

  upstream.on('error', (error) => {
    console.error('[ws-proxy] upstream error', error.message)
    if (client.readyState === WebSocket.OPEN) {
      safeClose(client, 1011, sanitizeCloseReason(error.message))
    }
  })

  client.on('message', (message) => {
    if (upstream.readyState === WebSocket.OPEN) {
      upstream.send(message)
    }
  })

  client.on('close', () => {
    if (upstream.readyState === WebSocket.OPEN || upstream.readyState === WebSocket.CONNECTING) {
      safeClose(upstream)
    }
  })

  client.on('error', (error) => {
    console.error('[ws-proxy] client error', error.message)
    safeClose(upstream, 1011, sanitizeCloseReason(error.message))
  })
})

httpServer.on('upgrade', (req, socket, head) => {
  if (!req.url.startsWith('/ws/')) {
    socket.destroy()
    return
  }

  wsBridge.handleUpgrade(req, socket, head, (ws) => {
    wsBridge.emit('connection', ws, req)
  })
})

httpServer.listen(Number(UI_PORT), '0.0.0.0', () => {
  console.log(`Web Console UI listening on http://localhost:${UI_PORT}`)
})

export { proxyToApi }
