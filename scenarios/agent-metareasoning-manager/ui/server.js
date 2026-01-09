const http = require('http')
const fs = require('fs')
const path = require('path')

const HOST = process.env.UI_HOST || '0.0.0.0'
const UI_PORT = Number(process.env.UI_PORT || process.env.PORT || 36000)
const API_PORT = Number(process.env.API_PORT || 8090)
const PUBLIC_DIR = path.join(__dirname, 'public')
const BRIDGE_SOURCE = path.join(__dirname, '../../../packages/iframe-bridge/dist/iframeBridgeChild.js')

const MIME_TYPES = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.ico': 'image/x-icon',
  '.png': 'image/png',
  '.svg': 'image/svg+xml',
  '.webp': 'image/webp',
  '.woff': 'font/woff',
  '.woff2': 'font/woff2'
}

function log(message, extra = '') {
  const suffix = extra ? ` ${extra}` : ''
  console.log(`[agent-metareasoning-manager][ui] ${message}${suffix}`)
}

function sanitizePath(baseDir, requestPath) {
  const rawPath = requestPath === '/' ? '/index.html' : requestPath
  const decoded = decodeURIComponent(rawPath)
  const normalised = path.normalize(path.join(baseDir, decoded))
  if (!normalised.startsWith(baseDir)) {
    return null
  }
  return normalised
}

function sendJson(res, statusCode, payload) {
  res.writeHead(statusCode, { 'Content-Type': 'application/json; charset=utf-8' })
  res.end(JSON.stringify(payload))
}

function streamFile(res, filePath) {
  fs.stat(filePath, (err, stats) => {
    if (err || !stats.isFile()) {
      sendJson(res, 404, { error: 'Not Found' })
      return
    }

    const ext = path.extname(filePath).toLowerCase()
    const contentType = MIME_TYPES[ext] || 'application/octet-stream'
    res.writeHead(200, { 'Content-Type': contentType })
    const stream = fs.createReadStream(filePath)
    stream.pipe(res)
    stream.on('error', error => {
      console.error('[agent-metareasoning-manager][ui] Failed to read file', error)
      if (!res.headersSent) {
        sendJson(res, 500, { error: 'Internal server error', details: error.message })
      } else {
        res.destroy(error)
      }
    })
  })
}

function proxyToApi(req, res, apiPath) {
  const targetPath = apiPath.startsWith('/') ? apiPath : `/${apiPath}`
  const headers = {
    ...req.headers,
    host: `127.0.0.1:${API_PORT}`
  }

  const options = {
    hostname: '127.0.0.1',
    port: API_PORT,
    path: targetPath,
    method: req.method,
    headers
  }

  const proxyReq = http.request(options, proxyRes => {
    res.writeHead(proxyRes.statusCode || 502, {
      ...proxyRes.headers,
      'access-control-allow-origin': proxyRes.headers['access-control-allow-origin'] || req.headers.origin || '*',
      'access-control-allow-credentials': proxyRes.headers['access-control-allow-credentials'] || 'true'
    })
    proxyRes.pipe(res, { end: true })
  })

  proxyReq.on('error', error => {
    console.error('[agent-metareasoning-manager][ui] API proxy error', error)
    if (!res.headersSent) {
      sendJson(res, 502, { error: 'API server unavailable', details: error.message })
    } else {
      res.destroy(error)
    }
  })

  if (req.method === 'GET' || req.method === 'HEAD' || req.method === 'OPTIONS') {
    proxyReq.end()
  } else {
    req.pipe(proxyReq, { end: true })
  }
}

const server = http.createServer((req, res) => {
  const method = req.method || 'GET'
  const url = new URL(req.url, `http://${req.headers.host || 'localhost'}`)

  if (method === 'OPTIONS') {
    res.writeHead(200, {
      'Access-Control-Allow-Origin': req.headers.origin || '*',
      'Access-Control-Allow-Methods': 'GET,POST,PUT,PATCH,DELETE,OPTIONS',
      'Access-Control-Allow-Headers': req.headers['access-control-request-headers'] || 'Content-Type, Authorization',
      'Access-Control-Allow-Credentials': 'true',
      'Access-Control-Max-Age': '86400'
    })
    res.end()
    return
  }

  if (url.pathname === '/health') {
    sendJson(res, 200, {
      status: 'ok',
      uiPort: UI_PORT,
      apiPort: API_PORT,
      timestamp: new Date().toISOString()
    })
    return
  }

  if (url.pathname.startsWith('/api')) {
    const search = url.search || ''
    const targetPath = url.pathname.replace(/^\/api/, '') || '/'
    proxyToApi(req, res, `${targetPath}${search}`)
    return
  }

  if (url.pathname === '/bridge/iframeBridgeChild.js') {
    streamFile(res, BRIDGE_SOURCE)
    return
  }

  const filePath = sanitizePath(PUBLIC_DIR, url.pathname)
  if (!filePath) {
    sendJson(res, 403, { error: 'Forbidden' })
    return
  }

  fs.stat(filePath, (err, stats) => {
    if (err) {
      sendJson(res, 404, { error: 'Not Found' })
      return
    }

    if (stats.isDirectory()) {
      const indexPath = path.join(filePath, 'index.html')
      streamFile(res, indexPath)
      return
    }

    streamFile(res, filePath)
  })
})

server.listen(UI_PORT, HOST, () => {
  log('UI server listening', `http://${HOST}:${UI_PORT}`)
  log('Proxying API traffic to', `http://127.0.0.1:${API_PORT}`)
})

process.on('SIGINT', () => {
  log('Shutting down UI server')
  server.close(() => process.exit(0))
})

process.on('SIGTERM', () => {
  log('Shutting down UI server')
  server.close(() => process.exit(0))
})
