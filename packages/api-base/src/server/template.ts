/**
 * Full scenario server template
 *
 * Provides a complete Express server implementation that handles all common
 * scenario needs: static files, API proxy, config endpoint, health endpoint,
 * CORS, metadata injection, etc.
 */

import express, { type Express, type Request, type Response, type NextFunction } from 'express'
import * as path from 'node:path'
import * as fs from 'node:fs'
import * as http from 'node:http'
import type { ServerTemplateOptions } from '../shared/types.js'
import { createConfigEndpoint } from './config.js'
import { createHealthEndpoint } from './health.js'
import { createProxyMiddleware, proxyWebSocketUpgrade } from './proxy.js'
import { injectProxyMetadata, injectScenarioConfig } from './inject.js'
import { resolveProxyAgent } from './agent.js'
import { parsePort, isAssetRequest } from '../shared/utils.js'

const IMMUTABLE_ASSET_EXTENSIONS = new Set([
  '.js',
  '.mjs',
  '.cjs',
  '.css',
  '.map',
  '.json',
  '.png',
  '.jpg',
  '.jpeg',
  '.gif',
  '.svg',
  '.ico',
  '.webp',
  '.avif',
  '.woff',
  '.woff2',
  '.ttf',
  '.eot',
  '.otf',
  '.mp4',
  '.webm',
  '.mp3',
  '.wav',
])

const HASHED_FILENAME_PATTERN = /[-_][A-Za-z0-9_\-]{6,}\./
const ONE_YEAR_SECONDS = 60 * 60 * 24 * 365

function isImmutableBuildAsset(filePath: string): boolean {
  const ext = path.extname(filePath).toLowerCase()
  if (!IMMUTABLE_ASSET_EXTENSIONS.has(ext)) {
    return false
  }

  const normalizedPath = filePath.split(path.sep).join('/')
  if (!normalizedPath.includes('/assets/')) {
    return false
  }

  const fileName = path.basename(filePath)
  return HASHED_FILENAME_PATTERN.test(fileName)
}

function applyStaticCacheHeaders(res: Response, filePath: string): void {
  if (isImmutableBuildAsset(filePath)) {
    res.setHeader('Cache-Control', `public, max-age=${ONE_YEAR_SECONDS}, immutable`)
    return
  }

  if (path.basename(filePath).toLowerCase() === 'index.html') {
    res.setHeader('Cache-Control', 'no-store, max-age=0')
    return
  }

  if (!res.getHeader('Cache-Control')) {
    res.setHeader('Cache-Control', 'public, max-age=0')
  }
}

/**
 * Default allowed CORS origins
 *
 * Includes localhost and loopback addresses with any port.
 * Auto-detects tunnel environments and allows HTTPS origins.
 *
 * @internal
 */
function buildDefaultCorsOrigins(uiPort?: string): string[] {
  const origins = [
    'http://localhost',
    'http://127.0.0.1',
    'http://[::1]',
    'null', // For file:// protocol
  ]

  if (uiPort) {
    origins.push(`http://localhost:${uiPort}`)
    origins.push(`http://127.0.0.1:${uiPort}`)
  }

  // Auto-allow tunnel/proxy contexts
  // Detect common tunnel environment variables or presence of HTTPS connections
  const isTunnelEnv = Boolean(
    process.env.TUNNEL_URL ||
    process.env.CLOUDFLARE_TUNNEL ||
    process.env.CLOUDFLARED_TUNNEL_ID ||
    process.env.NGROK_URL ||
    process.env.CLOUDFLARE_TUNNEL_TOKEN
  )

  if (isTunnelEnv) {
    // When tunnel is detected, allow all HTTPS origins
    origins.push('https://*')
  }

  return origins
}

/**
 * Check if origin is allowed
 *
 * @param origin - Origin to check
 * @param allowedOrigins - List of allowed origins (supports wildcards)
 * @returns Whether origin is allowed
 *
 * @internal
 */
function isOriginAllowed(origin: string, allowedOrigins: string[]): boolean {
  // Check for exact match
  if (allowedOrigins.includes(origin)) {
    return true
  }

  // Check for wildcard match
  if (allowedOrigins.includes('*')) {
    return true
  }

  // Check for pattern match (e.g., "http://localhost:*")
  for (const pattern of allowedOrigins) {
    if (pattern.includes('*')) {
      const regex = new RegExp('^' + pattern.replace(/\*/g, '.*') + '$')
      if (regex.test(origin)) {
        return true
      }
    }
  }

  // Check if origin starts with any allowed origin (for port wildcards)
  const originLower = origin.toLowerCase()
  for (const allowed of allowedOrigins) {
    const allowedLower = allowed.toLowerCase()
    if (allowedLower.endsWith(':*')) {
      const baseAllowed = allowedLower.slice(0, -2)
      if (originLower.startsWith(baseAllowed)) {
        return true
      }
    }
  }

  return false
}

/**
 * Create CORS middleware
 *
 * @param corsOrigins - Allowed origins (supports wildcards, defaults to localhost)
 * @param verbose - Whether to log CORS decisions
 * @returns CORS middleware
 *
 * @internal
 */
function createCorsMiddleware(corsOrigins: string | string[] | undefined, verbose: boolean) {
  return (req: Request, res: Response, next: NextFunction) => {
    const origin = req.headers.origin

    // Normalize cors origins
    let allowedOrigins: string[]
    if (corsOrigins === '*') {
      allowedOrigins = ['*']
    } else if (Array.isArray(corsOrigins)) {
      allowedOrigins = corsOrigins
    } else if (typeof corsOrigins === 'string') {
      allowedOrigins = corsOrigins.split(',').map(o => o.trim())
    } else {
      // Default: build from UI port
      allowedOrigins = buildDefaultCorsOrigins()
    }

    // Handle wildcard
    if (allowedOrigins.includes('*')) {
      res.setHeader('Access-Control-Allow-Origin', '*')
      res.setHeader('Access-Control-Allow-Credentials', 'false')
    } else if (origin && isOriginAllowed(origin, allowedOrigins)) {
      res.setHeader('Access-Control-Allow-Origin', origin)
      res.setHeader('Access-Control-Allow-Credentials', 'true')
      res.setHeader('Vary', 'Origin')
    } else if (origin) {
      if (verbose) {
        console.log(`[cors] Blocked origin: ${origin}`)
      }
      res.status(403).json({ error: 'Origin not allowed' })
      return
    }

    // Set CORS headers
    res.setHeader('Access-Control-Allow-Methods', 'GET,POST,PUT,PATCH,DELETE,OPTIONS')
    res.setHeader(
      'Access-Control-Allow-Headers',
      req.headers['access-control-request-headers'] || 'Accept,Authorization,Content-Type,X-CSRF-Token,X-Requested-With'
    )
    res.setHeader('Access-Control-Expose-Headers', 'Link')
    res.setHeader('Access-Control-Max-Age', '300')

    // Handle preflight
    if (req.method === 'OPTIONS') {
      res.status(204).end()
      return
    }

    next()
  }
}

/**
 * Create HTML injection middleware
 *
 * Intercepts HTML responses and injects proxy metadata and/or config.
 *
 * @internal
 */
function createHtmlInjectionMiddleware(options: ServerTemplateOptions) {
  const { proxyMetadata, scenarioConfig } = options

  // If nothing to inject, return no-op middleware
  if (!proxyMetadata && !scenarioConfig) {
    return (_req: Request, _res: Response, next: NextFunction) => next()
  }

  return (req: Request, res: Response, next: NextFunction) => {
    // Only intercept HTML responses
    const originalSend = res.send

    res.send = function (body: any): Response {
      // Check if this is an HTML response
      const contentType = res.getHeader('content-type')
      const isHtml = contentType && typeof contentType === 'string' && contentType.includes('text/html')

      if (isHtml && typeof body === 'string') {
        let modifiedBody = body

        // Inject proxy metadata
        if (proxyMetadata) {
          modifiedBody = injectProxyMetadata(modifiedBody, proxyMetadata)
        }

        // Inject scenario config
        if (scenarioConfig) {
          modifiedBody = injectScenarioConfig(modifiedBody, scenarioConfig)
        }

        return originalSend.call(this, modifiedBody)
      }

      return originalSend.call(this, body)
    }

    next()
  }
}

interface IndexHtmlCacheEntry {
  content: string
  mtimeMs: number
  size: number
}

function createIndexHtmlLoader(indexPath: string, useCache: boolean, verbose: boolean) {
  let cached: IndexHtmlCacheEntry | null = null

  return () => {
    if (!useCache) {
      return fs.readFileSync(indexPath, 'utf-8')
    }

    const stats = fs.statSync(indexPath)
    const shouldRefresh = !cached ||
      cached.mtimeMs !== stats.mtimeMs ||
      cached.size !== stats.size

    if (shouldRefresh) {
      const content = fs.readFileSync(indexPath, 'utf-8')
      cached = {
        content,
        mtimeMs: stats.mtimeMs,
        size: stats.size,
      }

      if (verbose) {
        console.log(`[server] Cached index.html (${stats.size} bytes) for SPA fallback`)
      }

      return content
    }

    return cached!.content
  }
}

/**
 * Create a complete scenario server
 *
 * Returns a configured Express application with all standard middleware:
 * - CORS handling
 * - API proxying
 * - Config endpoint
 * - Health endpoint
 * - Static file serving
 * - SPA fallback routing
 * - Optional metadata injection
 *
 * @param options - Server configuration options
 * @returns Configured Express application
 *
 * @example
 * ```typescript
 * import { createScenarioServer } from '@vrooli/api-base/server'
 *
 * const app = createScenarioServer({
 *   uiPort: process.env.UI_PORT || 3000,
 *   apiPort: process.env.API_PORT || 8080,
 *   distDir: './dist',
 *   serviceName: 'my-scenario',
 *   version: '1.0.0'
 * })
 *
 * app.listen(process.env.UI_PORT || 3000, () => {
 *   console.log('Server running')
 * })
 * ```
 */
function applyBodyParser(app: Express, setting: ServerTemplateOptions['bodyParser']) {
  if (setting === false) {
    return
  }

  if (typeof setting === 'function') {
    setting(app)
    return
  }

  // Default: standard JSON parser for UI-owned routes
  app.use(express.json({ limit: '10mb' }))
}

export function createScenarioServer(options: ServerTemplateOptions): Express {
  const {
    uiPort,
    apiPort,
    apiHost,
    wsPort,
    wsHost,
    distDir = './dist',
    serviceName = 'scenario-ui',
    version,
    corsOrigins,
    verbose = false,
    configBuilder,
    setupRoutes,
    proxyMetadata,
    scenarioConfig,
    wsPathPrefix,
    wsPathTransform,
    proxyHeaders,
    proxyTimeoutMs,
    proxyKeepAlive,
    proxyAgent,
    bodyParser = 'json',
    cacheIndexHtml = true,
  } = options

  // Parse ports
  const parsedUiPort = parsePort(uiPort)
  const parsedApiPort = parsePort(apiPort)
  const parsedWsPort = parsePort(wsPort) || parsedApiPort

  if (!parsedUiPort) {
    throw new Error('Invalid UI_PORT configuration')
  }
  if (!parsedApiPort) {
    throw new Error('Invalid API_PORT configuration')
  }

  // Create Express app
  const app = express()

  // Disable X-Powered-By header for security
  app.disable('x-powered-by')

  // CORS middleware
  app.use(createCorsMiddleware(corsOrigins, verbose))

  // HTML injection middleware (if needed)
  if (proxyMetadata || scenarioConfig) {
    app.use(createHtmlInjectionMiddleware(options))
  }

  const resolvedProxyAgent = resolveProxyAgent({ agent: proxyAgent, keepAlive: proxyKeepAlive })

  // API proxy middleware
  app.use('/api', createProxyMiddleware({
    apiPort: parsedApiPort,
    apiHost,
    verbose,
    headers: proxyHeaders,
    timeout: proxyTimeoutMs,
    agent: resolvedProxyAgent,
  }))

  // Apply body parser AFTER proxy middleware so API requests stay streaming
  applyBodyParser(app, bodyParser)

  // Config endpoint
  if (configBuilder) {
    app.get('/config', (_req: Request, res: Response) => {
      try {
        const config = configBuilder(process.env)
        res.json(config)
      } catch (error) {
        res.status(500).json({
          error: 'Failed to build configuration',
          details: error instanceof Error ? error.message : 'Unknown error',
        })
      }
    })
  } else {
    app.get('/config', createConfigEndpoint({
      apiPort: parsedApiPort,
      apiHost,
      wsPort: parsedWsPort,
      wsHost,
      uiPort: parsedUiPort,
      serviceName,
      version,
    }))
  }

  // Health endpoint
  app.get('/health', createHealthEndpoint({
    serviceName,
    version,
    apiPort: parsedApiPort,
    apiHost,
  }))

  // Custom routes
  if (setupRoutes) {
    setupRoutes(app)
  }

  // Resolve dist directory
  const absoluteDistDir = path.isAbsolute(distDir) ? distDir : path.resolve(process.cwd(), distDir)
  const indexPath = path.join(absoluteDistDir, 'index.html')
  const loadIndexHtml = createIndexHtmlLoader(indexPath, cacheIndexHtml !== false, verbose)

  // Check if dist directory exists
  if (!fs.existsSync(absoluteDistDir)) {
    console.warn(`[server] Warning: dist directory does not exist: ${absoluteDistDir}`)
    console.warn('[server] Static files will not be served. Run build first.')
  }

  // Serve static files with production-friendly caching
  app.use(express.static(absoluteDistDir, {
    setHeaders: (res, servedPath) => {
      applyStaticCacheHeaders(res as Response, servedPath)
    },
  }))

  // SPA fallback - serve index.html for all other routes
  // IMPORTANT: This must be smart about assets to avoid returning HTML for .js/.css/etc requests
  app.use((req: Request, res: Response, next: NextFunction) => {
    if (req.method !== 'GET' && req.method !== 'HEAD') {
      return next()
    }

    const requestPath = req.path

    // CRITICAL: Skip proxy routes - they should be handled by custom route handlers
    // Proxy routes contain '/proxy' in the path (e.g., /apps/scenario/proxy/*)
    if (requestPath.includes('/proxy')) {
      // Don't serve index.html for proxy routes - let them 404 or be handled by custom routes
      if (verbose) {
        console.log(`[server] Skipping SPA fallback for proxy route: ${requestPath}`)
      }
      res.status(404).type('text/plain').send('Proxy route not handled')
      return
    }

    // If this looks like an asset request (has .js, .css, etc extension or asset prefix),
    // return 404 instead of index.html. This prevents the browser from receiving HTML
    // when it expects a JavaScript file, which would break the app.
    if (isAssetRequest(requestPath)) {
      if (verbose) {
        console.log(`[server] Asset not found (404): ${requestPath}`)
      }
      res.status(404).type('text/plain').send('Not found')
      return
    }

    // For non-asset routes, serve index.html (SPA fallback)
    if (!fs.existsSync(indexPath)) {
      res.status(404).send('Application not built. Run build command first.')
      return
    }

    if (verbose) {
      console.log(`[server] SPA fallback: ${requestPath} -> index.html`)
    }

    // CRITICAL: Use res.send() instead of res.sendFile() so that any middleware
    // that wraps res.send (like base tag injection) will be triggered.
    // sendFile() bypasses wrapped send() interceptors!
    try {
      const htmlContent = loadIndexHtml()
      res.setHeader('Cache-Control', 'no-store, max-age=0')
      res.type('text/html').send(htmlContent)
    } catch (error) {
      if (verbose) {
        console.error(`[server] Error reading index.html:`, error)
      }
      res.status(500).send('Failed to serve application')
    }
  })

  // Log configuration
  if (verbose) {
    console.log('[server] Configuration:')
    console.log(`  UI Port: ${parsedUiPort}`)
    console.log(`  API Port: ${parsedApiPort}`)
    console.log(`  WS Port: ${parsedWsPort}`)
    console.log(`  Dist Dir: ${absoluteDistDir}`)
    console.log(`  Service: ${serviceName}${version ? ` v${version}` : ''}`)
    if (wsPathPrefix) {
      console.log(`  WebSocket Proxy: ${wsPathPrefix} -> API server`)
    }
  }

  // Attach WebSocket upgrade handler if configured
  if (wsPathPrefix) {
    // Store upgrade handler on app for later attachment to HTTP server
    // This is a bit of a hack, but Express doesn't support upgrade events directly
    ;(app as any).__wsUpgradeHandler = (req: any, socket: any, head: any) => {
      if (!req.url || !req.url.startsWith(wsPathPrefix)) {
        socket.destroy()
        return
      }

      // Transform path
      let transformedPath: string
      if (wsPathTransform) {
        transformedPath = wsPathTransform(req.url)
      } else {
        // Default: replace prefix with /api/v1
        transformedPath = req.url.replace(new RegExp(`^${wsPathPrefix}`), '/api/v1')
      }

      // Build headers
      let headers = proxyHeaders || {}
      if (typeof headers === 'function') {
        headers = headers(req)
      }

      // Create modified request
      const modifiedReq = Object.create(req)
      modifiedReq.url = transformedPath
      if (Object.keys(headers).length > 0) {
        modifiedReq.headers = { ...req.headers, ...headers }
      }

      // Call proxyWebSocketUpgrade (imported at top)
      proxyWebSocketUpgrade(modifiedReq, socket, head, {
        apiPort: parsedWsPort || parsedApiPort,
        apiHost: wsHost || apiHost,
        verbose,
        headers,
      })
    }
  }

  return app
}

/**
 * Create and start scenario server
 *
 * Convenience function that creates the server and starts listening.
 *
 * @param options - Server configuration options
 * @returns Express application (already listening)
 *
 * @example
 * ```typescript
 * import { startScenarioServer } from '@vrooli/api-base/server'
 *
 * startScenarioServer({
 *   uiPort: process.env.UI_PORT || 3000,
 *   apiPort: process.env.API_PORT || 8080,
 *   distDir: './dist',
 *   serviceName: 'my-scenario'
 * })
 * ```
 */
export function startScenarioServer(options: ServerTemplateOptions): Express {
  const app = createScenarioServer(options)

  const port = parsePort(options.uiPort)
  if (!port) {
    throw new Error('Invalid UI_PORT for server startup')
  }

  // Create HTTP server (needed for WebSocket upgrade handling)
  const server = http.createServer(app)

  // Attach WebSocket upgrade handler if it was configured
  const wsUpgradeHandler = (app as any).__wsUpgradeHandler
  if (wsUpgradeHandler) {
    server.on('upgrade', wsUpgradeHandler)
  }

  server.listen(Number.parseInt(port, 10), '0.0.0.0', () => {
    console.log(`${options.serviceName || 'Scenario'} UI server listening on port ${port}`)
    console.log(`Health: http://localhost:${port}/health`)
    console.log(`Config: http://localhost:${port}/config`)
    console.log(`UI: http://localhost:${port}`)
  })

  // Graceful shutdown
  const shutdown = () => {
    console.log('\nShutting down gracefully...')
    server.close(() => {
      process.exit(0)
    })
  }

  process.on('SIGTERM', shutdown)
  process.on('SIGINT', shutdown)

  return app
}
