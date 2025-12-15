import path from 'node:path'
import { fileURLToPath } from 'node:url'
import fs from 'node:fs'
import zlib from 'node:zlib'
import { startScenarioServer, injectBaseTag } from '@vrooli/api-base/server'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const SERVICE_NAME = 'browser-automation-studio'
const VERSION = process.env.npm_package_version || '1.0.0'
const API_HOST = process.env.API_HOST || 'localhost'
const WS_HOST = process.env.WS_HOST
const UI_PORT = process.env.UI_PORT || process.env.PORT || '3000'
const API_PORT = process.env.API_PORT

function ensureApiPort(port) {
  if (!port) {
    throw new Error('[server] API_PORT environment variable is required')
  }
  return port
}

const GZIP_CONTENT_TYPES = new Map([
  ['.js', 'application/javascript; charset=utf-8'],
  ['.css', 'text/css; charset=utf-8'],
  ['.html', 'text/html; charset=utf-8'],
  ['.json', 'application/json; charset=utf-8'],
  ['.svg', 'image/svg+xml'],
])

function ensureGzipAssetsSync(distDir) {
  if (!fs.existsSync(distDir)) {
    return
  }

  const shouldCompress = (fileName) => {
    const ext = path.extname(fileName)
    return GZIP_CONTENT_TYPES.has(ext) && !fileName.endsWith('.gz')
  }

  const walk = (dir) => {
    const entries = fs.readdirSync(dir, { withFileTypes: true })
    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name)
      if (entry.isDirectory()) {
        walk(fullPath)
        continue
      }
      if (!entry.isFile() || !shouldCompress(entry.name)) {
        continue
      }

      const gzPath = `${fullPath}.gz`
      const srcStat = fs.statSync(fullPath)
      const gzStat = fs.existsSync(gzPath) ? fs.statSync(gzPath) : null
      if (gzStat && gzStat.mtimeMs >= srcStat.mtimeMs) {
        continue
      }

      const src = fs.readFileSync(fullPath)
      const gz = zlib.gzipSync(src, { level: 6 })
      fs.writeFileSync(gzPath, gz)
    }
  }

  walk(distDir)
}

function enableGzipStatic(app, distDir) {
  app.use((req, res, next) => {
    if (req.method !== 'GET' && req.method !== 'HEAD') {
      return next()
    }

    const acceptEncoding = String(req.headers['accept-encoding'] || '')
    if (!acceptEncoding.includes('gzip')) {
      return next()
    }

    const pathname = req.path || ''
    const ext = path.extname(pathname)
    const contentType = GZIP_CONTENT_TYPES.get(ext)
    if (!contentType) {
      return next()
    }

    const relativePath = pathname.startsWith('/') ? pathname.slice(1) : pathname
    const gzPath = path.join(distDir, `${relativePath}.gz`)
    if (!fs.existsSync(gzPath)) {
      return next()
    }

    res.setHeader('Content-Encoding', 'gzip')
    res.setHeader('Content-Type', contentType)
    res.setHeader('Vary', 'Accept-Encoding')

    const urlValue = String(req.url || '')
    const queryIndex = urlValue.indexOf('?')
    if (queryIndex >= 0) {
      req.url = `${urlValue.slice(0, queryIndex)}.gz${urlValue.slice(queryIndex)}`
    } else {
      req.url = `${urlValue}.gz`
    }
    next()
  })
}

export function startServer() {
  const distDir = path.join(__dirname, 'dist')
  ensureGzipAssetsSync(distDir)

  return startScenarioServer({
    uiPort: UI_PORT,
    apiPort: ensureApiPort(API_PORT),
    apiHost: API_HOST,
    wsHost: WS_HOST || API_HOST,
    distDir: path.join(__dirname, 'dist'),
    serviceName: SERVICE_NAME,
    version: VERSION,
    corsOrigins: '*',
    wsPathPrefix: '/ws',
    wsPathTransform: (pathValue) => pathValue,
    proxyTimeoutMs: 60000,
    setupRoutes: (app) => {
      enableGzipStatic(app, distDir)

      // Inject a <base> tag so deep links resolve assets from the correct root.
      // Handles both direct routes (/) and proxied paths (/apps/<id>/proxy/*).
      app.use((req, res, next) => {
        const originalSend = res.send

        res.send = function (body) {
          const contentType = res.getHeader('content-type')
          const isHtml =
            contentType && typeof contentType === 'string' && contentType.includes('text/html')

          if (isHtml && typeof body === 'string') {
            const pathName = req.originalUrl?.split('?')[0] || '/'
            const proxyMarker = '/proxy/'
            const proxyIdx = pathName.indexOf(proxyMarker)
            const basePath =
              proxyIdx >= 0
                ? pathName.slice(0, proxyIdx + proxyMarker.length)
                : '/'

            const modified = injectBaseTag(body, basePath, {
              skipIfExists: true,
              dataAttribute: 'data-bas-base',
            })
            return originalSend.call(this, modified)
          }

          return originalSend.call(this, body)
        }

        next()
      })
    },
  })
}

if (import.meta.url === `file://${process.argv[1]}`) {
  startServer()
}
