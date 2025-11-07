/**
 * Shared test setup for Host Scenario E2E tests
 *
 * Provides common utilities and server setup for all host scenario browser tests.
 * This reduces duplication and makes tests easier to maintain.
 */

import { chromium, type Browser, type Page } from 'playwright'
import {
  createScenarioServer,
  injectBaseTag,
  buildProxyMetadata,
  injectProxyMetadata,
} from '../../../server/index.js'
import * as http from 'node:http'
import type { Server } from 'node:http'
import type { Express } from 'express'

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
  childUiPort: number
  childApiPort: number
  hostUiPort: number
  hostApiPort: number
  childId: string
}

/**
 * Setup child scenario (the embedded app)
 */
export async function setupChildScenario(
  childId: string,
  childUiPort: number,
  childApiPort: number
): Promise<{ uiServer: Server; apiServer: Server }> {
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

    res.writeHead(404)
    res.end()
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

  return { uiServer, apiServer }
}

/**
 * Setup host scenario (the container app)
 */
export async function setupHostScenario(
  childId: string,
  childUiPort: number,
  childApiPort: number,
  hostUiPort: number,
  hostApiPort: number
): Promise<{ uiServer: Server; apiServer: Server }> {
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
            name: 'Test Child',
            port_mappings: {
              UI_PORT: childUiPort,
              API_PORT: childApiPort,
            },
          },
        })
      )
      return
    }

    res.writeHead(404)
    res.end()
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
      // Host's own base tag injection
      app.use((req, res, next) => {
        if (req.path.startsWith('/apps/') && req.path.includes('/proxy')) {
          return next()
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

      // Child proxy route
      app.all(`/apps/${childId}/proxy/*`, async (req, res) => {
        const targetPath = req.params[0] || ''

        try {
          const isHtmlRequest = req.headers.accept?.includes('text/html')

          if (isHtmlRequest && req.method === 'GET') {
            const targetUrl = `http://127.0.0.1:${childUiPort}/${targetPath}`

            const childResponse = await new Promise<{
              data: string
              headers: http.IncomingHttpHeaders
            }>((resolve, reject) => {
              http.get(
                targetUrl,
                {
                  headers: {
                    ...req.headers,
                    host: `127.0.0.1:${childUiPort}`,
                  },
                },
                (childRes) => {
                  let data = ''
                  childRes.on('data', (chunk) => {
                    data += chunk
                  })
                  childRes.on('end', () => {
                    resolve({ data, headers: childRes.headers })
                  })
                }
              ).on('error', reject)
            })

            let html = childResponse.data
            const contentType = childResponse.headers['content-type'] || ''

            if (contentType.includes('text/html') && typeof html === 'string') {
              const metadata = buildProxyMetadata({
                appId: childId,
                hostScenario: 'host-scenario',
                targetScenario: childId,
                ports: [
                  {
                    port: childUiPort,
                    label: 'UI_PORT',
                    slug: 'ui-port',
                    path: `/apps/${childId}/proxy`,
                    assetNamespace: `/apps/${childId}/proxy/assets`,
                    aliases: ['UI_PORT', 'primary'],
                    source: 'port_mappings',
                    isPrimary: true,
                    priority: 80,
                  },
                ],
                primaryPort: {
                  port: childUiPort,
                  label: 'UI_PORT',
                  slug: 'ui-port',
                  path: `/apps/${childId}/proxy`,
                  assetNamespace: `/apps/${childId}/proxy/assets`,
                  aliases: ['UI_PORT', 'primary'],
                  source: 'port_mappings',
                  isPrimary: true,
                  priority: 80,
                },
                loopbackHosts: ['127.0.0.1', 'localhost'],
              })

              html = injectProxyMetadata(html, metadata, {
                patchFetch: true,
              })

              html = injectBaseTag(html, `/apps/${childId}/proxy/`, {
                skipIfExists: true,
                dataAttribute: 'data-host-proxy',
              })
            }

            res.setHeader('Content-Type', contentType)
            res.send(html)
          } else {
            // Proxy non-HTML requests
            const targetUrl = `http://127.0.0.1:${childUiPort}/${targetPath}`

            const proxyReq = http.request(
              targetUrl,
              {
                method: req.method,
                headers: {
                  ...req.headers,
                  host: `127.0.0.1:${childUiPort}`,
                },
              },
              (proxyRes) => {
                res.writeHead(proxyRes.statusCode || 500, proxyRes.headers)
                proxyRes.pipe(res)
              }
            )

            proxyReq.on('error', (err) => {
              res.status(502).json({ error: 'Proxy failed', details: err.message })
            })

            req.pipe(proxyReq)
          }
        } catch (error: any) {
          res.status(502).json({ error: 'Proxy failed', details: error.message })
        }
      })
    },
  })

  const uiServer = await new Promise<Server>((resolve) => {
    const server = hostApp.listen(hostUiPort, '127.0.0.1', () => {
      resolve(server)
    })
  })

  return { uiServer, apiServer }
}

/**
 * Setup complete test environment with browser and servers
 *
 * @param portOffset - Offset to add to base ports (for parallel test execution)
 */
export async function setupTestEnvironment(portOffset = 0): Promise<TestContext> {
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
    hostApiPort
  )

  return {
    browser,
    page,
    childUiServer: child.uiServer,
    childApiServer: child.apiServer,
    hostUiServer: host.uiServer,
    hostApiServer: host.apiServer,
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
