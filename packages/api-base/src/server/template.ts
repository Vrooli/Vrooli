/**
 * Full scenario server template
 *
 * Provides a complete Express server implementation that handles all common
 * scenario needs: static files, API proxy, config endpoint, health endpoint,
 * browser origin policy, metadata injection, etc.
 */

import express, { type Express, type Request, type Response, type NextFunction } from 'express'
import * as path from 'node:path'
import * as fs from 'node:fs'
import * as http from 'node:http'
import * as zlib from 'node:zlib'
import type { ServerTemplateOptions } from '../shared/types.js'
import { createConfigEndpoint } from './config.js'
import { createHealthEndpoint } from './health.js'
import { createProxyMiddleware, proxyToApi, proxyWebSocketUpgrade } from './proxy.js'
import { injectProxyMetadata, injectScenarioConfig, injectBaseTag } from './inject.js'
import { resolveProxyAgent } from './agent.js'
import { createEmbeddedProxyRouter } from './embedded.js'
import { createLocalPortProxy } from './local-port-proxy.js'
import { createOriginPolicy } from './origin-policy.js'
import { APP_NOTIFICATIONS_PREFIX, createAppNotificationsRouter, resolveAppNotifications } from './app-notifications.js'
import { assertLifecycleManagedUI } from './lifecycle.js'
import { parsePort, isAssetRequest } from '../shared/utils.js'

// Connect-RPC URL grammar: /<dotted.package.Service>/<UpperCamelMethod>.
// The package must contain at least one '.' so we never match plain SPA
// routes like /login or static assets like /index.html. Method names start
// with an uppercase letter, distinguishing /foo.bar/Baz from /assets/img.png
// or any other lowercase-leading second segment.
const CONNECT_PATH_PATTERN = /^\/[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)+\/[A-Z][A-Za-z0-9_]*$/

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
const MIN_GZIP_BYTES = 1024
const COMPRESSIBLE_CONTENT_TYPE = /^(?:text\/|application\/(?:javascript|json|manifest\+json|xml|wasm)|image\/svg\+xml)/i

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

function appendVary(res: Response, value: string): void {
  const current = res.getHeader('Vary')
  if (!current) {
    res.setHeader('Vary', value)
    return
  }

  const values = Array.isArray(current) ? current.join(',') : String(current)
  if (values.toLowerCase().split(',').map(item => item.trim()).includes(value.toLowerCase())) {
    return
  }
  res.setHeader('Vary', `${values}, ${value}`)
}

function acceptsGzip(req: Request): boolean {
  return String(req.headers['accept-encoding'] || '').split(',').some(part => part.trim().toLowerCase().startsWith('gzip'))
}

function shouldCompressResponse(res: Response, bodyLength: number): boolean {
  if (bodyLength < MIN_GZIP_BYTES) {
    return false
  }
  if (res.statusCode < 200 || res.statusCode >= 300) {
    return false
  }
  if (res.getHeader('Content-Encoding')) {
    return false
  }
  const contentType = String(res.getHeader('Content-Type') || '')
  return COMPRESSIBLE_CONTENT_TYPE.test(contentType)
}

function gzipStaticResponseMiddleware(req: Request, res: Response, next: NextFunction): void {
  if (req.method !== 'GET' || !acceptsGzip(req)) {
    next()
    return
  }

  const chunks: Buffer[] = []
  const originalWrite = res.write.bind(res) as (...args: any[]) => boolean
  const originalEnd = res.end.bind(res) as (...args: any[]) => Response

  const bufferFrom = (chunk: any, encoding?: any): Buffer => {
    if (Buffer.isBuffer(chunk)) {
      return chunk
    }
    return Buffer.from(chunk, typeof encoding === 'string' ? encoding as BufferEncoding : undefined)
  }

  ;(res as unknown as { write: (...args: any[]) => boolean }).write = (chunk: any, encoding?: any, callback?: any): boolean => {
    if (chunk) {
      chunks.push(bufferFrom(chunk, encoding))
    }
    if (typeof encoding === 'function') {
      encoding()
    } else if (typeof callback === 'function') {
      callback()
    }
    return true
  }

  ;(res as unknown as { end: (...args: any[]) => Response }).end = (chunk?: any, encoding?: any, callback?: any): Response => {
    if (chunk) {
      chunks.push(bufferFrom(chunk, encoding))
    }

    const body = Buffer.concat(chunks)
    appendVary(res, 'Accept-Encoding')
    if (!shouldCompressResponse(res, body.length)) {
      for (const buffered of chunks) {
        originalWrite(buffered)
      }
      return originalEnd(typeof callback === 'function' ? callback : undefined)
    }

    const compressed = zlib.gzipSync(body)
    res.setHeader('Content-Encoding', 'gzip')
    res.removeHeader('Content-Length')
    return originalEnd(compressed, typeof callback === 'function' ? callback : undefined)
  }

  next()
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
 * - Browser origin policy (same-origin by default; see origin-policy.ts)
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
  assertLifecycleManagedUI({
    serviceName: options.serviceName,
    disableLifecycleGuard: options.disableLifecycleGuard,
  })

  if ('corsOrigins' in options) {
    throw new Error(
      `${options.serviceName ?? 'scenario-ui'}: the corsOrigins option was removed. ` +
      'Scenario UI servers are same-origin by default, including through the tunnel and Web Console; ' +
      'delete the option, or list exact cross-origin callers in allowedOrigins. See packages/api-base/SECURITY.md.',
    )
  }
  const originPolicy = createOriginPolicy({
    allowedOrigins: options.allowedOrigins,
    publicPaths: options.publicPaths,
  })

  const {
    uiPort,
    apiPort,
    apiHost,
    wsPort,
    wsHost,
    distDir = './dist',
    serviceName = 'scenario-ui',
    version,
    verbose = false,
    configBuilder,
    setupRoutes,
    setupBeforeProxy,
    proxyMetadata,
    scenarioConfig,
    wsPathPrefix,
    wsPathTransform,
    localPortProxy: localPortProxyOption,
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

  const localPortProxy = localPortProxyOption
    ? createLocalPortProxy({
      ...(localPortProxyOption === true ? {} : localPortProxyOption),
      excludedPorts: [parsedUiPort, parsedApiPort, parsedWsPort ?? parsedApiPort, ...(localPortProxyOption === true ? [] : localPortProxyOption.excludedPorts ?? [])],
    })
    : null
  const configuredLocalPortPathPrefix = localPortProxyOption && typeof localPortProxyOption === 'object' ? localPortProxyOption.pathPrefix : undefined
  const localPortPathPrefix = (configuredLocalPortPathPrefix || '/_local').trim().replace(/\/+$/, '') || '/_local'

  // Create Express app
  const app = express()

  // Disable X-Powered-By header for security
  app.disable('x-powered-by')

  // The loopback proxy has its own opaque-origin capability policy: its
  // sandboxed documents send `Origin: null` by design. Mount it before the
  // scenario origin policy, which would refuse them, and keep the two
  // policies independent.
  if (localPortProxy) {
    app.use(localPortPathPrefix, localPortProxy.router)
    ;(app as any).__localPortProxyDestroy = localPortProxy.destroy
  }

  // Browser origin policy: refuse cross-origin browser requests before any
  // route, proxy or static handler runs (see origin-policy.ts).
  app.use(originPolicy.middleware({ verbose }))

  // Scenario ingress policy must run before any generic proxy.
  if (setupBeforeProxy) setupBeforeProxy(app)

  // HTML injection middleware (if needed)
  if (proxyMetadata || scenarioConfig) {
    app.use(createHtmlInjectionMiddleware(options))
  }

  const resolvedProxyAgent = resolveProxyAgent({ agent: proxyAgent, keepAlive: proxyKeepAlive })

  // Connect-RPC proxy.
  //
  // Connect procedures live at /<fully.qualified.Service>/<MethodName> with
  // no shared prefix, so the /api mount below can't reach them. The pattern
  // requires a dotted package (at least one '.' before the first slash) and
  // an UpperCamelCase method name, which matches Connect's URL grammar
  // exactly and can't shadow any SPA route or static asset.
  //
  // proxyToApi is invoked directly here (rather than createProxyMiddleware)
  // because the middleware coerces every path under /api; Connect paths must
  // be forwarded unchanged.
  app.use(CONNECT_PATH_PATTERN, async (req: Request, res: Response) => {
    const headers = typeof proxyHeaders === 'function' ? proxyHeaders(req) : (proxyHeaders ?? {})
    await proxyToApi(req, res, req.originalUrl, {
      apiPort: parsedApiPort,
      apiHost,
      verbose,
      headers,
      timeout: proxyTimeoutMs,
      agent: resolvedProxyAgent,
    })
  })

  // Built-in app notifications (opt-in through notifications.json). Mounted
  // before the body parser so request bodies reach the hub unchanged.
  const appNotifications = resolveAppNotifications(options.appNotifications, serviceName)
  if (appNotifications) {
    app.use(APP_NOTIFICATIONS_PREFIX, createAppNotificationsRouter(appNotifications))
  }

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
    buildIdentity: process.env.VROOLI_BUILD_IDENTITY,
    apiPort: parsedApiPort,
    apiHost,
  }))

  // Custom routes
  if (setupRoutes) {
    setupRoutes(app)
  }

  // Built-in embedded scenario proxy
  if (options.embeddedProxy) {
    const proxyOpts = options.embeddedProxy === true ? {} : options.embeddedProxy
    if (proxyOpts.enabled !== false) {
      app.use('/embedded', createEmbeddedProxyRouter(proxyOpts))
    }
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

  app.use(gzipStaticResponseMiddleware)

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
    if (requestPath.includes('/proxy') || requestPath.startsWith('/embedded/') || requestPath.startsWith(`${APP_NOTIFICATIONS_PREFIX}/`) || (localPortProxy && requestPath.startsWith(`${localPortPathPrefix}/`))) {
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
      let htmlContent = loadIndexHtml()
      // Inject base tag to ensure relative asset paths (./assets/...) resolve from root.
      // This is critical when accessing nested routes like /chat/uuid - without <base href="/">,
      // the browser would try to load /chat/assets/... which doesn't exist.
      htmlContent = injectBaseTag(htmlContent, '/')
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
    if (localPortProxy) console.log(`  Local Port Proxy: ${localPortPathPrefix}/<port>/ -> loopback`)
  }

  // Attach WebSocket upgrade handler if configured
  if (wsPathPrefix || localPortProxy) {
    // Express has no upgrade hook, so the handler is stored on the app and
    // bound to whichever http.Server ends up serving it. Both start paths bind
    // it: startScenarioServer binds its own server explicitly, and app.listen
    // is wrapped below so a caller that starts the app itself is not silently
    // left without WebSocket proxying.
    ;(app as any).__wsUpgradeHandler = (req: any, socket: any, head: any) => {
      if (!req.url || !wsPathPrefix || !req.url.startsWith(wsPathPrefix)) {
        if (!localPortProxy?.handleUpgrade(req, socket, head)) socket.destroy()
        return
      }

      // wsPathPrefix selects which upgrades to proxy; it does not rewrite them.
      // The upstream path is preserved unless a caller asks for a remap, so a
      // prefix the API also serves needs no transform to work.
      const transformedPath = wsPathTransform ? wsPathTransform(req.url) : req.url

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
        originPolicy,
      })
    }

    // A caller that starts the app itself (createScenarioServer + app.listen,
    // rather than startScenarioServer) would otherwise receive an app whose
    // wsPathPrefix is configured but never bound. The upgrade would fall
    // through to the ordinary HTTP proxy, reach the API as a plain GET, and be
    // rejected there — a 400 that names nothing about the missing binding.
    // Wrapping listen makes the option behave the same on both start paths.
    const originalListen = app.listen.bind(app)
    app.listen = ((...args: unknown[]) => {
      const server = originalListen(...(args as Parameters<typeof originalListen>))
      attachWsUpgradeHandler(server, app)
      return server
    }) as typeof app.listen
  }

  return app
}

/**
 * Bind an app's configured WebSocket upgrade handler to the server that serves
 * it. Safe to call more than once for the same pair: a second call for an
 * already-bound server is a no-op, so a caller that both wraps listen and binds
 * explicitly does not proxy each upgrade twice.
 */
export function attachWsUpgradeHandler(server: http.Server, app: Express): void {
  const handler = (app as any).__wsUpgradeHandler
  if (!handler) return

  const bound: Set<http.Server> =
    (app as any).__wsUpgradeBoundServers ??
    ((app as any).__wsUpgradeBoundServers = new Set<http.Server>())
  if (bound.has(server)) return

  bound.add(server)
  server.on('upgrade', handler)
  server.once('close', () => {
    bound.delete(server)
    ;(app as any).__localPortProxyDestroy?.()
  })
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
  attachWsUpgradeHandler(server, app)

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
