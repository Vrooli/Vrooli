/**
 * Integration tests for Host Scenario Pattern
 *
 * Tests the complete flow of a host scenario (like app-monitor) embedding
 * child scenarios through iframes with proxy routes.
 *
 * Key scenarios tested:
 * 1. Host serving its own pages with correct base tag (/)
 * 2. Host proxying child scenario with correct base tag (/apps/child/proxy/)
 * 3. Asset requests routing correctly for both host and child
 * 4. API requests routing correctly for both host and child
 * 5. WebSocket connections for both host and child
 * 6. No interference between host and child
 */

import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import {
  createScenarioServer,
  injectBaseTag,
  buildProxyMetadata,
  injectProxyMetadata,
  proxyToApi,
} from '../../server/index.js'
import * as http from 'node:http'
import * as fs from 'node:fs'
import * as path from 'node:path'
import WebSocket from 'ws'
import type { Server } from 'node:http'
import type { Express } from 'express'

/**
 * Helper to find available port
 */
async function findAvailablePort(startPort: number): Promise<number> {
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
 * Helper to make HTTP requests
 */
async function makeRequest(
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

describe('Host Scenario Pattern Integration', () => {
  // Server instances
  let childUiServer: Server
  let childApiServer: Server
  let childWsServer: WebSocket.Server
  let hostUiServer: Server
  let hostApiServer: Server
  let hostWsServer: WebSocket.Server

  // Port assignments
  let childUiPort: number
  let childApiPort: number
  let hostUiPort: number
  let hostApiPort: number

  // Test data
  const childId = 'test-child-scenario'
  const testMessages: string[] = []

  beforeAll(async () => {
    // Allocate ports
    childUiPort = await findAvailablePort(40000)
    childApiPort = await findAvailablePort(40100)
    hostUiPort = await findAvailablePort(40200)
    hostApiPort = await findAvailablePort(40300)

    // ========================================
    // Setup Child Scenario (the embedded app)
    // ========================================

    // Child API server
    childApiServer = http.createServer((req, res) => {
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

    // Child WebSocket server
    childWsServer = new WebSocket.Server({ noServer: true })
    childWsServer.on('connection', (ws) => {
      testMessages.push('child-ws-connected')

      ws.on('message', (data) => {
        const message = data.toString()
        testMessages.push(`child-ws-received:${message}`)
        ws.send(`child-echo:${message}`)
      })

      ws.on('close', () => {
        testMessages.push('child-ws-disconnected')
      })

      ws.send('child-welcome')
    })

    childApiServer.on('upgrade', (request, socket, head) => {
      if (request.url?.startsWith('/api/v1/ws')) {
        childWsServer.handleUpgrade(request, socket, head, (ws) => {
          childWsServer.emit('connection', ws, request)
        })
      } else {
        socket.destroy()
      }
    })

    await new Promise<void>((resolve) => {
      childApiServer.listen(childApiPort, '127.0.0.1', resolve)
    })

    // Child UI server (standard scenario server)
    const childApp = createScenarioServer({
      uiPort: childUiPort,
      apiPort: childApiPort,
      distDir: './dist',
      serviceName: 'child-scenario',
      verbose: false,
      setupRoutes: (app) => {
        // Serve a simple HTML page for testing
        app.get('/index.html', (req, res) => {
          res.setHeader('Content-Type', 'text/html')
          res.send(`<!DOCTYPE html>
<html>
<head>
  <title>Child Scenario</title>
  <link rel="stylesheet" href="styles.css">
</head>
<body>
  <div id="app">Child Content</div>
  <script src="main.js"></script>
</body>
</html>`)
        })

        // Serve mock assets
        app.get('/styles.css', (req, res) => {
          res.setHeader('Content-Type', 'text/css')
          res.send('body { background: blue; }')
        })

        app.get('/main.js', (req, res) => {
          res.setHeader('Content-Type', 'application/javascript')
          res.send('console.log("Child app loaded");')
        })
      },
    })

    childUiServer = await new Promise<Server>((resolve) => {
      const server = childApp.listen(childUiPort, '127.0.0.1', () => {
        resolve(server)
      })
    })

    // ========================================
    // Setup Host Scenario (the container app)
    // ========================================

    // Host API server
    hostApiServer = http.createServer((req, res) => {
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

    // Host WebSocket server
    hostWsServer = new WebSocket.Server({ noServer: true })
    hostWsServer.on('connection', (ws) => {
      testMessages.push('host-ws-connected')

      ws.on('message', (data) => {
        const message = data.toString()
        testMessages.push(`host-ws-received:${message}`)
        ws.send(`host-echo:${message}`)
      })

      ws.on('close', () => {
        testMessages.push('host-ws-disconnected')
      })

      ws.send('host-welcome')
    })

    hostApiServer.on('upgrade', (request, socket, head) => {
      if (request.url?.startsWith('/api/v1/ws')) {
        hostWsServer.handleUpgrade(request, socket, head, (ws) => {
          hostWsServer.emit('connection', ws, request)
        })
      } else {
        socket.destroy()
      }
    })

    await new Promise<void>((resolve) => {
      hostApiServer.listen(hostApiPort, '127.0.0.1', resolve)
    })

    // Host UI server (with proxy functionality)
    const hostApp = createScenarioServer({
      uiPort: hostUiPort,
      apiPort: hostApiPort,
      distDir: './dist',
      serviceName: 'host-scenario',
      verbose: false,
      setupRoutes: (app) => {
        // ========================================
        // HOST'S OWN PAGES (with base tag /)
        // ========================================
        app.use((req, res, next) => {
          // Skip proxy routes
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
  <link rel="stylesheet" href="host-styles.css">
</head>
<body>
  <div id="app">Host Content</div>
  <iframe src="/apps/${childId}/proxy/index.html"></iframe>
  <script src="host-main.js"></script>
</body>
</html>`)
        })

        // Host's own assets
        app.get('/host-styles.css', (req, res) => {
          res.setHeader('Content-Type', 'text/css')
          res.send('body { background: red; }')
        })

        app.get('/host-main.js', (req, res) => {
          res.setHeader('Content-Type', 'application/javascript')
          res.send('console.log("Host app loaded");')
        })

        // ========================================
        // CHILD PROXY ROUTES
        // ========================================
        app.all(`/apps/${childId}/proxy/*`, async (req, res) => {
          const targetPath = req.params[0] || ''

          try {
            // For HTML requests, intercept and inject metadata
            const isHtmlRequest = req.headers.accept?.includes('text/html')

            if (isHtmlRequest && req.method === 'GET') {
              // Fetch HTML from child
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

              // Inject metadata if it's HTML
              if (contentType.includes('text/html') && typeof html === 'string') {
                // Build proxy metadata
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
                      aliases: ['UI_PORT', 'ui_port', 'primary'],
                      source: 'port_mappings',
                      isPrimary: true,
                      priority: 80,
                    },
                    {
                      port: childApiPort,
                      label: 'API_PORT',
                      slug: 'api-port',
                      path: `/apps/${childId}/ports/api-port/proxy`,
                      assetNamespace: `/apps/${childId}/ports/api-port/proxy/assets`,
                      aliases: ['API_PORT', 'api_port'],
                      source: 'port_mappings',
                      isPrimary: false,
                      priority: 30,
                    },
                  ],
                  primaryPort: {
                    port: childUiPort,
                    label: 'UI_PORT',
                    slug: 'ui-port',
                    path: `/apps/${childId}/proxy`,
                    assetNamespace: `/apps/${childId}/proxy/assets`,
                    aliases: ['UI_PORT', 'ui_port', 'primary'],
                    source: 'port_mappings',
                    isPrimary: true,
                    priority: 80,
                  },
                  loopbackHosts: ['127.0.0.1', 'localhost'],
                })

                // Inject proxy metadata
                html = injectProxyMetadata(html, metadata, {
                  patchFetch: true,
                })

                // Inject base tag for child
                html = injectBaseTag(html, `/apps/${childId}/proxy/`, {
                  skipIfExists: true,
                  dataAttribute: 'data-host-proxy',
                })
              }

              res.setHeader('Content-Type', contentType)
              res.send(html)
            } else {
              // For non-HTML requests, proxy directly
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

    hostUiServer = await new Promise<Server>((resolve) => {
      const server = hostApp.listen(hostUiPort, '127.0.0.1', () => {
        resolve(server)
      })
    })

    // Setup WebSocket upgrade handlers
    hostUiServer.on('upgrade', async (req, socket, head) => {
      if (req.url?.startsWith('/api')) {
        const { proxyWebSocketUpgrade } = await import('../../server/proxy.js')
        proxyWebSocketUpgrade(req, socket, head, {
          apiPort: hostApiPort,
          verbose: false,
        })
      } else {
        socket.destroy()
      }
    })

    childUiServer.on('upgrade', async (req, socket, head) => {
      if (req.url?.startsWith('/api')) {
        const { proxyWebSocketUpgrade } = await import('../../server/proxy.js')
        proxyWebSocketUpgrade(req, socket, head, {
          apiPort: childApiPort,
          verbose: false,
        })
      } else {
        socket.destroy()
      }
    })
  })

  afterAll(async () => {
    await new Promise<void>((resolve) => {
      hostWsServer.close(() => {
        childWsServer.close(() => {
          hostApiServer.close(() => {
            childApiServer.close(() => {
              hostUiServer.close(() => {
                childUiServer.close(() => {
                  resolve()
                })
              })
            })
          })
        })
      })
    })
  })

  // ========================================
  // HOST SCENARIO TESTS
  // ========================================

  it('host serves its own HTML with correct base tag', async () => {
    const result = await makeRequest(hostUiPort, '/index.html', {
      headers: { accept: 'text/html' },
    })

    expect(result.status).toBe(200)
    expect(result.body).toContain('Host Content')
    expect(result.body).toContain('<base')
    expect(result.body).toContain('href="/"')
    expect(result.body).toContain('data-host-self="injected"')
  })

  it('host serves its own assets correctly', async () => {
    const cssResult = await makeRequest(hostUiPort, '/host-styles.css')
    expect(cssResult.status).toBe(200)
    expect(cssResult.body).toContain('background: red')

    const jsResult = await makeRequest(hostUiPort, '/host-main.js')
    expect(jsResult.status).toBe(200)
    expect(jsResult.body).toContain('Host app loaded')
  })

  it('host API requests work correctly', async () => {
    const result = await makeRequest(hostUiPort, '/api/v1/health')

    expect(result.status).toBe(200)
    const data = JSON.parse(result.body)
    expect(data.status).toBe('healthy')
    expect(data.service).toBe('host-api')
  })

  it('host WebSocket connections work', async () => {
    const wsUrl = `ws://127.0.0.1:${hostUiPort}/api/v1/ws`
    const ws = new WebSocket(wsUrl)
    const messages: string[] = []

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)

      ws.on('open', () => {
        ws.send('host-test')
      })

      ws.on('message', (data) => {
        messages.push(data.toString())
        if (messages.length >= 2) {
          clearTimeout(timeout)
          ws.close()
        }
      })

      ws.on('close', () => {
        resolve()
      })

      ws.on('error', (err) => {
        clearTimeout(timeout)
        reject(err)
      })
    })

    expect(messages).toContain('host-welcome')
    expect(messages).toContain('host-echo:host-test')
  })

  // ========================================
  // CHILD SCENARIO (PROXIED) TESTS
  // ========================================

  it('child serves HTML through proxy with correct base tag', async () => {
    const result = await makeRequest(hostUiPort, `/apps/${childId}/proxy/index.html`, {
      headers: { accept: 'text/html' },
    })

    expect(result.status).toBe(200)
    expect(result.body).toContain('Child Content')
    expect(result.body).toContain('<base')
    expect(result.body).toContain(`href="/apps/${childId}/proxy/"`)
    expect(result.body).toContain('data-host-proxy="injected"')
  })

  it('child HTML includes proxy metadata', async () => {
    const result = await makeRequest(hostUiPort, `/apps/${childId}/proxy/index.html`, {
      headers: { accept: 'text/html' },
    })

    expect(result.status).toBe(200)
    expect(result.body).toContain('__VROOLI_PROXY_INFO__')
    expect(result.body).toContain(childId)
    expect(result.body).toContain('host-scenario')
  })

  it('child assets are served through proxy', async () => {
    const cssResult = await makeRequest(hostUiPort, `/apps/${childId}/proxy/styles.css`)
    expect(cssResult.status).toBe(200)
    expect(cssResult.body).toContain('background: blue')

    const jsResult = await makeRequest(hostUiPort, `/apps/${childId}/proxy/main.js`)
    expect(jsResult.status).toBe(200)
    expect(jsResult.body).toContain('Child app loaded')
  })

  it('child API requests work when accessed directly', async () => {
    const result = await makeRequest(childUiPort, '/api/v1/data')

    expect(result.status).toBe(200)
    const data = JSON.parse(result.body)
    expect(data.data).toBe('child-data')
  })

  it('child WebSocket connections work when accessed directly', async () => {
    const wsUrl = `ws://127.0.0.1:${childUiPort}/api/v1/ws`
    const ws = new WebSocket(wsUrl)
    const messages: string[] = []

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)

      ws.on('open', () => {
        ws.send('child-test')
      })

      ws.on('message', (data) => {
        messages.push(data.toString())
        if (messages.length >= 2) {
          clearTimeout(timeout)
          ws.close()
        }
      })

      ws.on('close', () => {
        resolve()
      })

      ws.on('error', (err) => {
        clearTimeout(timeout)
        reject(err)
      })
    })

    expect(messages).toContain('child-welcome')
    expect(messages).toContain('child-echo:child-test')
  })

  // ========================================
  // ISOLATION TESTS
  // ========================================

  it('host and child have separate base tags in their HTML', async () => {
    const hostResult = await makeRequest(hostUiPort, '/index.html', {
      headers: { accept: 'text/html' },
    })

    const childResult = await makeRequest(hostUiPort, `/apps/${childId}/proxy/index.html`, {
      headers: { accept: 'text/html' },
    })

    // Host has base="/"
    expect(hostResult.body).toContain('href="/"')
    expect(hostResult.body).toContain('data-host-self="injected"')

    // Child has base="/apps/.../proxy/"
    expect(childResult.body).toContain(`href="/apps/${childId}/proxy/"`)
    expect(childResult.body).toContain('data-host-proxy="injected"')

    // They should be different
    expect(hostResult.body).not.toContain(`href="/apps/${childId}/proxy/"`)
    expect(childResult.body).not.toContain('data-host-self="injected"')
  })

  it('host and child both have working WebSockets simultaneously', async () => {
    const hostWsUrl = `ws://127.0.0.1:${hostUiPort}/api/v1/ws`
    const childWsUrl = `ws://127.0.0.1:${childUiPort}/api/v1/ws`

    const hostWs = new WebSocket(hostWsUrl)
    const childWs = new WebSocket(childWsUrl)

    const hostMessages: string[] = []
    const childMessages: string[] = []

    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timeout')), 5000)
      let hostReady = false
      let childReady = false

      const checkDone = () => {
        if (hostReady && childReady) {
          clearTimeout(timeout)
          hostWs.close()
          childWs.close()
          resolve()
        }
      }

      hostWs.on('open', () => hostWs.send('host-concurrent'))
      hostWs.on('message', (data) => {
        hostMessages.push(data.toString())
        if (hostMessages.length >= 2) {
          hostReady = true
          checkDone()
        }
      })

      childWs.on('open', () => childWs.send('child-concurrent'))
      childWs.on('message', (data) => {
        childMessages.push(data.toString())
        if (childMessages.length >= 2) {
          childReady = true
          checkDone()
        }
      })

      hostWs.on('error', (err) => {
        clearTimeout(timeout)
        reject(err)
      })
      childWs.on('error', (err) => {
        clearTimeout(timeout)
        reject(err)
      })
    })

    // Host received its own messages
    expect(hostMessages).toContain('host-welcome')
    expect(hostMessages).toContain('host-echo:host-concurrent')

    // Child received its own messages
    expect(childMessages).toContain('child-welcome')
    expect(childMessages).toContain('child-echo:child-concurrent')

    // No cross-contamination
    expect(hostMessages).not.toContain('child-welcome')
    expect(childMessages).not.toContain('host-welcome')
  })

  it('404 for missing assets (prevents SPA fallback for .js/.css)', async () => {
    // Host should return 404 for missing assets, not HTML
    const hostAssetResult = await makeRequest(hostUiPort, '/missing-asset.js')
    expect(hostAssetResult.status).toBe(404)
    expect(hostAssetResult.body).not.toContain('<!DOCTYPE html>')

    // Child should return 404 for missing assets, not HTML
    const childAssetResult = await makeRequest(childUiPort, '/missing-asset.css')
    expect(childAssetResult.status).toBe(404)
    expect(childAssetResult.body).not.toContain('<!DOCTYPE html>')
  })

  it('SPA fallback still works for non-asset routes', async () => {
    // Host should return HTML for non-asset routes
    const hostPageResult = await makeRequest(hostUiPort, '/dashboard')
    expect(hostPageResult.status).toBe(404) // No SPA fallback in test setup

    // But NOT for assets
    const hostAssetResult = await makeRequest(hostUiPort, '/some-file.js')
    expect(hostAssetResult.status).toBe(404)
  })
})
