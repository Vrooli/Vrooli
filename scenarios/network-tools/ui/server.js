const http = require('http')
const fs = require('fs')
const path = require('path')

// Validate required environment variables
function getRequiredEnv(name, defaultValue) {
  const value = process.env[name]
  if (!value && defaultValue === undefined) {
    console.error(`[network-tools][ui] FATAL: Required environment variable ${name} is not set`)
    process.exit(1)
  }
  return value || defaultValue
}

function validatePort(port, name) {
  const portNum = Number(port)
  if (isNaN(portNum) || portNum < 1 || portNum > 65535) {
    console.error(`[network-tools][ui] FATAL: Invalid port for ${name}: ${port}`)
    process.exit(1)
  }
  return portNum
}

// Use environment variable for host (default to 127.0.0.1 for security)
// For container deployments, explicitly set HOST='0.0.0.0' in environment
const HOST = process.env.HOST || '127.0.0.1'

// Require UI_PORT to be explicitly set (no defaults for ports)
// Prefer UI_PORT, but accept standard PORT env var for compatibility
const uiPortValue = process.env.UI_PORT || process.env.PORT
if (!uiPortValue) {
  console.error('[network-tools][ui] FATAL: UI_PORT or PORT environment variable must be set')
  process.exit(1)
}
const UI_PORT = validatePort(uiPortValue, 'UI_PORT')

// Require API_PORT to be explicitly set (no defaults)
if (!process.env.API_PORT) {
  console.error('[network-tools][ui] FATAL: API_PORT environment variable must be set')
  process.exit(1)
}
const API_PORT = validatePort(process.env.API_PORT, 'API_PORT')

const PUBLIC_DIR = path.join(__dirname, 'public')
const NODE_MODULES_DIR = path.join(__dirname, 'node_modules')

const MIME_TYPES = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.ico': 'image/x-icon',
  '.png': 'image/png',
  '.svg': 'image/svg+xml',
  '.woff': 'font/woff',
  '.woff2': 'font/woff2'
}

function sanitizePath(baseDir, requestPath) {
  const rawPath = requestPath === '/' ? '/index.html' : requestPath
  const normalized = path.normalize(path.join(baseDir, decodeURIComponent(rawPath)))
  if (!normalized.startsWith(baseDir)) {
    return null
  }
  return normalized
}

function respondNotFound(res) {
  res.writeHead(404, { 'Content-Type': 'application/json; charset=utf-8' })
  res.end(JSON.stringify({ error: 'Not Found' }))
}

function respondServerError(res, details) {
  res.writeHead(500, { 'Content-Type': 'application/json; charset=utf-8' })
  res.end(JSON.stringify({ error: 'Internal server error', details }))
}

function respondHealth(res) {
  // Check API connectivity (use 127.0.0.1 for local API communication)
  const apiHost = process.env.API_HOST || '127.0.0.1'
  const apiUrl = `http://${apiHost}:${API_PORT}/health`
  const checkStartTime = Date.now()

  const apiCheckReq = http.request(apiUrl, { method: 'GET', timeout: 5000 }, apiRes => {
    const latencyMs = Date.now() - checkStartTime
    let apiConnected = apiRes.statusCode === 200

    apiRes.on('data', () => {}) // drain response
    apiRes.on('end', () => {
      res.writeHead(200, { 'Content-Type': 'application/json; charset=utf-8' })
      res.end(
        JSON.stringify({
          status: apiConnected ? 'healthy' : 'degraded',
          service: 'network-tools-ui',
          timestamp: new Date().toISOString(),
          readiness: apiConnected,
          api_connectivity: {
            connected: apiConnected,
            api_url: apiUrl,
            last_check: new Date().toISOString(),
            error: apiConnected ? null : {
              code: 'HTTP_ERROR',
              message: `API returned status ${apiRes.statusCode}`,
              category: 'network',
              retryable: true
            },
            latency_ms: latencyMs
          }
        })
      )
    })
  })

  apiCheckReq.on('error', error => {
    const latencyMs = Date.now() - checkStartTime
    res.writeHead(200, { 'Content-Type': 'application/json; charset=utf-8' })
    res.end(
      JSON.stringify({
        status: 'degraded',
        service: 'network-tools-ui',
        timestamp: new Date().toISOString(),
        readiness: false,
        api_connectivity: {
          connected: false,
          api_url: apiUrl,
          last_check: new Date().toISOString(),
          error: {
            code: error.code || 'CONNECTION_ERROR',
            message: error.message || 'Failed to connect to API',
            category: 'network',
            retryable: true
          },
          latency_ms: latencyMs
        }
      })
    )
  })

  apiCheckReq.on('timeout', () => {
    apiCheckReq.destroy()
    const latencyMs = Date.now() - checkStartTime
    res.writeHead(200, { 'Content-Type': 'application/json; charset=utf-8' })
    res.end(
      JSON.stringify({
        status: 'degraded',
        service: 'network-tools-ui',
        timestamp: new Date().toISOString(),
        readiness: false,
        api_connectivity: {
          connected: false,
          api_url: apiUrl,
          last_check: new Date().toISOString(),
          error: {
            code: 'TIMEOUT',
            message: 'API health check timed out after 5000ms',
            category: 'network',
            retryable: true
          },
          latency_ms: latencyMs
        }
      })
    )
  })

  apiCheckReq.end()
}

function getContentType(filePath) {
  const ext = path.extname(filePath).toLowerCase()
  return MIME_TYPES[ext] || 'application/octet-stream'
}

function proxyToApi(req, res, apiPath) {
  const targetPath = apiPath || req.url
  const headers = {
    ...req.headers,
    host: `127.0.0.1:${API_PORT}`
  }

  const apiHost = process.env.API_HOST || '127.0.0.1'
  const options = {
    hostname: apiHost,
    port: API_PORT,
    path: targetPath,
    method: req.method,
    headers
  }

  const proxyReq = http.request(options, proxyRes => {
    // Forward CORS headers from the API server without adding wildcards
    const responseHeaders = { ...proxyRes.headers }
    if (proxyRes.headers['access-control-allow-origin']) {
      responseHeaders['access-control-allow-origin'] = proxyRes.headers['access-control-allow-origin']
    }
    if (proxyRes.headers['access-control-allow-credentials']) {
      responseHeaders['access-control-allow-credentials'] = proxyRes.headers['access-control-allow-credentials']
    }
    res.writeHead(proxyRes.statusCode || 502, responseHeaders)
    proxyRes.pipe(res, { end: true })
  })

  proxyReq.on('error', error => {
    console.error('[network-tools][ui] API proxy error', error)
    res.writeHead(502, { 'Content-Type': 'application/json; charset=utf-8' })
    res.end(JSON.stringify({ error: 'API server unavailable', details: error.message }))
  })

  if (req.method === 'GET' || req.method === 'HEAD' || req.method === 'OPTIONS') {
    proxyReq.end()
  } else {
    req.pipe(proxyReq, { end: true })
  }
}

const server = http.createServer((req, res) => {
  // Handle health check quickly
  if (req.url === '/health') {
    respondHealth(res)
    return
  }

  // Proxy API traffic through secure tunnel helper
  if (req.url.startsWith('/api/')) {
    proxyToApi(req, res, req.url)
    return
  }

  if (req.url.startsWith('/node_modules/')) {
    const modulePath = sanitizePath(NODE_MODULES_DIR, req.url.replace('/node_modules', ''))
    if (!modulePath) {
      respondNotFound(res)
      return
    }

    fs.stat(modulePath, (err, stats) => {
      if (err || !stats.isFile()) {
        respondNotFound(res)
        return
      }

      const stream = fs.createReadStream(modulePath)
      stream.on('open', () => {
        res.writeHead(200, { 'Content-Type': getContentType(modulePath) })
        stream.pipe(res)
      })
      stream.on('error', error => {
        console.error('[network-tools][ui] Failed to read module file', error)
        respondServerError(res, error.message)
      })
    })
    return
  }

  const filePath = sanitizePath(PUBLIC_DIR, req.url)
  if (!filePath) {
    res.writeHead(403, { 'Content-Type': 'application/json; charset=utf-8' })
    res.end(JSON.stringify({ error: 'Forbidden' }))
    return
  }

  fs.stat(filePath, (err, stats) => {
    if (err) {
      respondNotFound(res)
      return
    }

    let finalPath = filePath
    if (stats.isDirectory()) {
      finalPath = path.join(finalPath, 'index.html')
    }

    const stream = fs.createReadStream(finalPath)
    stream.on('open', () => {
      res.writeHead(200, { 'Content-Type': getContentType(finalPath) })
      stream.pipe(res)
    })
    stream.on('error', error => {
      console.error('[network-tools][ui] Failed to read file', error)
      respondServerError(res, error.message)
    })
  })
})

server.listen(UI_PORT, HOST, () => {
  const apiHost = process.env.API_HOST || '127.0.0.1'
  console.log(`[network-tools][ui] Web console listening on http://${HOST}:${UI_PORT}`)
  console.log(`[network-tools][ui] Proxying API traffic to http://${apiHost}:${API_PORT}`)
  console.log(`[network-tools][ui] Environment: UI_PORT=${UI_PORT}, API_PORT=${API_PORT}, HOST=${HOST}`)
})
