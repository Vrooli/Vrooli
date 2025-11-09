/**
 * Shared test setup for Host Scenario E2E tests
 *
 * Provides common utilities and server setup for all host scenario browser tests.
 * This reduces duplication and makes tests easier to maintain.
 */

import type { Server } from 'node:http'
import * as http from 'node:http'
import { chromium, type Browser, type Page } from 'playwright'
import {
  createScenarioServer,
  injectBaseTag,
  createScenarioProxyHost,
  proxyWebSocketUpgrade,
  type ScenarioProxyHostController,
  type HostEndpointDefinition,
} from '../../../server/index.js'
import { WebSocketServer } from 'ws'
import { isAssetRequest } from '../../../shared/utils.js'

function hasPathTraversal(value: string | undefined | null): boolean {
  if (!value) {
    return false
  }
  let decoded = value
  try {
    decoded = decodeURIComponent(value)
  } catch {
    decoded = value
  }
  const normalized = decoded
    .replace(/\\/g, '/')
    .replace(new RegExp('/+', 'g'), '/')
  return normalized.split('/').some((segment) => segment === '..')
}

/**
 * Find available port for testing
 */
export async function findAvailablePort(startPort: number): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = http.createServer()

    server.listen(startPort, '127.0.0.1', () => {
      const address = server.address()
      if (address && typeof address === 'object') {
        const port = address.port
        server.close(() => {
          resolve(port)
        })
      } else {
        reject(new Error('Failed to get server address'))
      }
    })

    server.on('error', (err: any) => {
      if (err.code === 'EADDRINUSE') {
        resolve(findAvailablePort(startPort + 1))
      } else {
        reject(err)
      }
    })
  })
}

/**
 * Test context shared across all tests
 */
export interface TestContext {
  browser: Browser
  page: Page
  childUiServer: Server
  childApiServer: Server
  hostUiServer: Server
  hostApiServer: Server
  childWsServer?: WebSocketServer
  hostWsServer?: WebSocketServer
  childUiPort: number
  childApiPort: number
  hostUiPort: number
  hostApiPort: number
  childId: string
}

interface HostScenarioOptions {
  hostEndpoints?: HostEndpointDefinition[]
}

/**
 * Setup child scenario (the embedded app)
 */
export async function setupChildScenario(
  childId: string,
  childUiPort: number,
  childApiPort: number
): Promise<{ uiServer: Server; apiServer: Server; wsServer: WebSocketServer }> {
  // Child API server
  const apiServer = http.createServer((req, res) => {
    if (req.url === '/api/v1/health') {
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ status: 'healthy', service: 'child-api' }))
      return
    }

    if (req.url === '/api/v1/data') {
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ data: 'child-data', timestamp: Date.now() }))
      return
    }

    if (req.url?.startsWith('/api/v1/issues')) {
      const issuesUrl = new URL(`http://127.0.0.1${req.url}`)
      const parsedLimit = Number(issuesUrl.searchParams.get('limit') ?? '20')
      const limit = Number.isFinite(parsedLimit) ? parsedLimit : 20
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ limit, count: limit, service: 'child-api' }))
      return
    }

    // Child should NOT have /summary or /resources endpoints
    // If these are called, it means routing is broken (host requests going to child)
    if (req.url === '/api/v1/summary' || req.url === '/api/v1/resources') {
      res.writeHead(404, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ error: 'Endpoint not found in child API', service: 'child-api' }))
      return
    }

    res.writeHead(404)
    res.end()
  })

  const childWsServer = new WebSocketServer({ noServer: true })
  childWsServer.on('connection', (ws) => {
    ws.send(JSON.stringify({ type: 'welcome', source: 'child-api' }))
    ws.on('message', (data) => {
      ws.send(
        JSON.stringify({
          type: 'echo',
          source: 'child-api',
          payload: { raw: data.toString() },
        })
      )
    })
  })

  apiServer.on('upgrade', (req, socket, head) => {
    if (req.url?.startsWith('/api/v1/ws')) {
      childWsServer.handleUpgrade(req, socket, head, (ws) => {
        childWsServer.emit('connection', ws, req)
      })
    } else {
      socket.destroy()
    }
  })

  await new Promise<void>((resolve) => {
    apiServer.listen(childApiPort, '127.0.0.1', resolve)
  })

  // Child UI server
  const childApp = createScenarioServer({
    uiPort: childUiPort,
    apiPort: childApiPort,
    distDir: './dist',
    serviceName: 'child-scenario',
    verbose: false,
    setupRoutes: (app) => {
      app.get('/index.html', (req, res) => {
        res.setHeader('Content-Type', 'text/html')
        res.send(`<!DOCTYPE html>
<html>
<head>
  <title>Child Scenario</title>
  <meta charset="UTF-8">
  <link rel="stylesheet" href="styles.css">
</head>
<body>
  <div id="child-app" data-testid="child-content">
    <h1>Child Scenario</h1>
    <p>This is embedded content</p>
  </div>
  <script src="main.js"></script>
  <script>
    // This will help us test if scripts execute in child
    window.childLoaded = true;
    console.log('Child scenario loaded');

    // Test that base tag exists and is correct
    const base = document.querySelector('base');
    if (base) {
      window.childBasePath = base.getAttribute('href');
      console.log('Child base path:', window.childBasePath);
    } else {
      console.error('Child: No base tag found!');
    }

    // Test proxy metadata
    if (window.__VROOLI_PROXY_INFO__) {
      window.childProxyInfo = window.__VROOLI_PROXY_INFO__;
      console.log('Child has proxy metadata:', window.childProxyInfo);
    } else {
      console.error('Child: No proxy metadata found!');
    }
  </script>
</body>
</html>`)
      })

      // Serve CSS
      app.get('/styles.css', (req, res) => {
        res.setHeader('Content-Type', 'text/css')
        res.send('body { background: blue; }')
      })

      // Serve JS
      app.get('/main.js', (req, res) => {
        res.setHeader('Content-Type', 'application/javascript')
        res.send('console.log("Child main.js loaded");')
      })
    },
  })

  const uiServer = await new Promise<Server>((resolve) => {
    const server = childApp.listen(childUiPort, '127.0.0.1', () => {
      resolve(server)
    })
  })

  uiServer.on('upgrade', (req, socket, head) => {
    if (req.url?.startsWith('/api')) {
      proxyWebSocketUpgrade(req as any, socket, head, {
        apiPort: childApiPort,
        apiHost: '127.0.0.1',
        verbose: false,
      })
    } else {
      socket.destroy()
    }
  })

  return { uiServer, apiServer, wsServer: childWsServer }
}

/**
 * Setup host scenario (the container app)
 */
export async function setupHostScenario(
  childId: string,
  childUiPort: number,
  childApiPort: number,
  hostUiPort: number,
  hostApiPort: number,
  hostOptions?: HostScenarioOptions
): Promise<{ uiServer: Server; apiServer: Server; wsServer: WebSocketServer }> {
  let hostProxyController: ScenarioProxyHostController | null = null
  const hostWsServer = new WebSocketServer({ noServer: true })
  hostWsServer.on('connection', (ws) => {
    ws.send('host-welcome')
    ws.on('message', (data) => {
      ws.send(`host-echo:${data.toString()}`)
    })
  })
  // Host API server
  const apiServer = http.createServer((req, res) => {
    if (req.url === '/api/v1/health') {
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ status: 'healthy', service: 'host-api' }))
      return
    }

    // Mock endpoint to get child app metadata
    if (req.url === `/api/v1/apps/${childId}`) {
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(
        JSON.stringify({
          data: {
            id: childId,
            name: childId,
            port_mappings: {
              UI_PORT: childUiPort,
              API_PORT: childApiPort,
            },
          },
        })
      )
      return
    }

    // Mock /summary endpoint (host-specific)
    if (req.url === '/api/v1/summary') {
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ summary: 'host-summary', apps: [], service: 'host-api' }))
      return
    }

    // Mock /resources endpoint (host-specific)
    if (req.url === '/api/v1/resources') {
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ resources: ['postgres', 'redis'], service: 'host-api' }))
      return
    }

    res.writeHead(404)
    res.end()
  })

  apiServer.on('upgrade', (req, socket, head) => {
    if (req.url?.startsWith('/api/v1/ws')) {
      hostWsServer.handleUpgrade(req, socket, head, (ws) => {
        hostWsServer.emit('connection', ws, req)
      })
    } else {
      socket.destroy()
    }
  })

  await new Promise<void>((resolve) => {
    apiServer.listen(hostApiPort, '127.0.0.1', resolve)
  })

  // Host UI server
  const hostApp = createScenarioServer({
    uiPort: hostUiPort,
    apiPort: hostApiPort,
    distDir: './dist',
    serviceName: 'host-scenario',
    verbose: false,
    setupRoutes: (app) => {
      app.use((req, res, next) => {
        const target = req.originalUrl || req.url || req.path
        if (hasPathTraversal(target)) {
          res.status(400).json({ error: 'Invalid path' })
          return
        }
        next()
      })

      // Host's own base tag injection
      // Add additional proxy routes like app-monitor has
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
        app.use(route, (req, res, next) => {
          const candidatePath = req.originalUrl || req.url || req.path
          if (hasPathTraversal(candidatePath)) {
            res.status(400).json({ error: 'Invalid path' })
            return
          }
          // Proxy to host API (just like createProxyMiddleware would)
          const targetUrl = `http://127.0.0.1:${hostApiPort}${req.url}`
          http.get(targetUrl, (apiRes) => {
            res.writeHead(apiRes.statusCode || 500, apiRes.headers)
            apiRes.pipe(res)
          }).on('error', (err) => {
            res.status(502).json({ error: 'Proxy failed', details: err.message })
          })
        })
      })

      app.use((req, res, next) => {
        if (req.path.startsWith('/apps/') && req.path.includes('/proxy')) {
          return next()
        }

        const candidatePath = req.originalUrl || req.url || req.path
        if (hasPathTraversal(candidatePath)) {
          res.status(400).json({ error: 'Invalid path' })
          return
        }

        const originalSend = res.send
        res.send = function (body: any) {
          const contentType = res.getHeader('content-type')
          const isHtml =
            contentType &&
            typeof contentType === 'string' &&
            contentType.includes('text/html')

          if (isHtml && typeof body === 'string') {
            const modifiedBody = injectBaseTag(body, '/', {
              skipIfExists: true,
              dataAttribute: 'data-host-self',
            })
            return originalSend.call(this, modifiedBody)
          }

          return originalSend.call(this, body)
        }

        next()
      })

      // Host's own pages
      app.get('/index.html', (req, res) => {
        res.setHeader('Content-Type', 'text/html')
        res.send(`<!DOCTYPE html>
<html>
<head>
  <title>Host Scenario</title>
  <meta charset="UTF-8">
  <link rel="stylesheet" href="host-styles.css">
</head>
<body>
  <div id="host-app" data-testid="host-content">
    <h1>Host Scenario</h1>
    <p>This is the host application</p>
  </div>

  <iframe
    id="child-iframe"
    src="/apps/${childId}/proxy/index.html"
    width="800"
    height="600"
    data-testid="child-iframe"
  ></iframe>

  <script src="host-main.js"></script>
  <script>
    window.hostLoaded = true;
    console.log('Host scenario loaded');

    // Test that base tag is correct
    const base = document.querySelector('base');
    if (base) {
      window.hostBasePath = base.getAttribute('href');
      console.log('Host base path:', window.hostBasePath);
    } else {
      console.error('Host: No base tag found!');
    }

    // Wait for iframe to load
    const iframe = document.getElementById('child-iframe');
    iframe.addEventListener('load', () => {
      console.log('Child iframe loaded');
      window.iframeLoaded = true;
    });
  </script>
</body>
</html>`)
      })

      // Host assets
      app.get('/host-styles.css', (req, res) => {
        res.setHeader('Content-Type', 'text/css')
        res.send('body { background: red; }')
      })

      app.get('/host-main.js', (req, res) => {
        res.setHeader('Content-Type', 'application/javascript')
        res.send('console.log("Host main.js loaded");')
      })

      // SPA Fallback for nested host routes (like /apps/:id/preview, /dashboard, etc.)
      // This must come BEFORE the proxy route so it doesn't catch everything
      // This simulates the SPA fallback that would normally serve dist/index.html
      const hostHtmlContent = `<!DOCTYPE html>
<html>
<head>
  <title>Host Scenario</title>
  <meta charset="UTF-8">
  <link rel="stylesheet" href="host-styles.css">
</head>
<body>
  <div id="host-app" data-testid="host-content">
    <h1>Host Scenario</h1>
    <p>This is the host application</p>
  </div>

  <iframe
    id="child-iframe"
    src="/apps/${childId}/proxy/index.html"
    width="800"
    height="600"
    data-testid="child-iframe"
  ></iframe>

  <script src="host-main.js"></script>
  <script>
    window.hostLoaded = true;
    console.log('Host scenario loaded');

    // Test that base tag is correct
    const base = document.querySelector('base');
    if (base) {
      window.hostBasePath = base.getAttribute('href');
      console.log('Host base path:', window.hostBasePath);
    } else {
      console.error('Host: No base tag found!');
    }

    // Wait for iframe to load
    const iframe = document.getElementById('child-iframe');
    iframe.addEventListener('load', () => {
      console.log('Child iframe loaded');
      window.iframeLoaded = true;
    });
  </script>
</body>
</html>`

      // SPA Fallback for nested host routes (before proxy route)
      // This handles routes like /apps/:id/preview, /dashboard, etc.
      // Must come AFTER specific routes but BEFORE proxy route
      app.get('*', (req, res, next) => {
        // Skip proxy routes (let proxy handler deal with them)
        if (req.path.includes('/proxy')) {
          return next()
        }

        // Skip asset routes (let createScenarioServer's SPA fallback handle 404s)
        if (isAssetRequest(req.path)) {
          return next()
        }

        const candidatePath = req.originalUrl || req.url || req.path
        if (hasPathTraversal(candidatePath)) {
          res.status(400).send('Invalid path')
          return
        }

        // Serve host HTML for everything else (SPA fallback)
        // This simulates what would happen in production with dist/index.html
        res.setHeader('Content-Type', 'text/html')
        res.send(hostHtmlContent)
      })

      const controller = createScenarioProxyHost({
        hostScenario: 'host-scenario',
        patchFetch: true,
        childBaseTagAttribute: 'data-host-proxy',
        hostEndpoints: hostOptions?.hostEndpoints,
        fetchAppMetadata: async (appId) => {
          const result = await makeRequest(hostApiPort, `/api/v1/apps/${appId}`)
          try {
            const parsed = JSON.parse(result.body || '{}')
            return parsed.data || parsed
          } catch {
            return null
          }
        },
      })

      hostProxyController = controller
      app.use(controller.router)
    },
  })

  const uiServer = await new Promise<Server>((resolve) => {
    const server = hostApp.listen(hostUiPort, '127.0.0.1', () => {
      resolve(server)
    })
  })

  uiServer.on('upgrade', async (req, socket, head) => {
    if (hostProxyController && (await hostProxyController.handleUpgrade(req, socket, head))) {
      return
    }
    if (req.url?.startsWith('/api')) {
      proxyWebSocketUpgrade(req as any, socket, head, {
        apiPort: hostApiPort,
        apiHost: '127.0.0.1',
        verbose: false,
      })
      return
    }
    socket.destroy()
  })

  return { uiServer, apiServer, wsServer: hostWsServer }
}

/**
 * Setup complete test environment with browser and servers
 *
 * @param portOffset - Offset to add to base ports (for parallel test execution)
 */
export async function setupTestEnvironment(portOffset = 0, options?: HostScenarioOptions): Promise<TestContext> {
  // Launch browser
  const browser = await chromium.launch({
    headless: true, // Set to false for debugging
  })
  const page = await browser.newPage()

  // Allocate ports with offset to avoid conflicts when running tests in parallel
  const childUiPort = await findAvailablePort(45000 + portOffset)
  const childApiPort = await findAvailablePort(45100 + portOffset)
  const hostUiPort = await findAvailablePort(45200 + portOffset)
  const hostApiPort = await findAvailablePort(45300 + portOffset)

  const childId = 'test-child'

  // Setup child scenario
  const child = await setupChildScenario(childId, childUiPort, childApiPort)

  // Setup host scenario
  const host = await setupHostScenario(
    childId,
    childUiPort,
    childApiPort,
    hostUiPort,
    hostApiPort,
    options
  )

  return {
    browser,
    page,
    childUiServer: child.uiServer,
    childApiServer: child.apiServer,
    childWsServer: child.wsServer,
    hostUiServer: host.uiServer,
    hostApiServer: host.apiServer,
    hostWsServer: host.wsServer,
    childUiPort,
    childApiPort,
    hostUiPort,
    hostApiPort,
    childId,
  }
}

/**
 * Cleanup test environment
 */
export async function cleanupTestEnvironment(ctx: TestContext): Promise<void> {
  await ctx.browser.close()

  if (ctx.hostWsServer) {
    await new Promise<void>((resolve) => {
      ctx.hostWsServer!.close(() => resolve())
    })
  }

  if (ctx.childWsServer) {
    await new Promise<void>((resolve) => {
      ctx.childWsServer!.close(() => resolve())
    })
  }

  await new Promise<void>((resolve) => {
    ctx.hostApiServer.close(() => {
      ctx.childApiServer.close(() => {
        ctx.hostUiServer.close(() => {
          ctx.childUiServer.close(() => {
            resolve()
          })
        })
      })
    })
  })
}

/**
 * Make HTTP request for testing
 */
export async function makeRequest(
  port: number,
  path: string,
  options: {
    method?: string
    headers?: Record<string, string>
  } = {}
): Promise<{
  status: number
  body: string
  headers: http.IncomingHttpHeaders
}> {
  return new Promise((resolve, reject) => {
    const req = http.request(
      {
        hostname: '127.0.0.1',
        port,
        path,
        method: options.method || 'GET',
        headers: options.headers || {},
      },
      (res) => {
        let body = ''
        res.on('data', (chunk) => {
          body += chunk
        })
        res.on('end', () => {
          resolve({
            status: res.statusCode || 500,
            body,
            headers: res.headers,
          })
        })
      }
    )

    req.on('error', reject)
    req.setTimeout(5000, () => {
      req.destroy()
      reject(new Error('Request timeout'))
    })
    req.end()
  })
}
