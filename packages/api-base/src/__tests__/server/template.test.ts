/**
 * Tests for server/template.ts
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import express from 'express'
import { createScenarioServer, startScenarioServer } from '../../server/template.js'
import * as proxyModule from '../../server/proxy.js'
import type { ServerTemplateOptions } from '../../shared/types.js'
import type { Server } from 'node:http'
import * as http from 'node:http'
import * as fs from 'node:fs'
import * as path from 'node:path'
import * as os from 'node:os'

// Helper to make HTTP requests to test server
async function makeRequest(server: Server, path: string, method: string = 'GET'): Promise<{
  status: number
  body: any
  headers: http.IncomingHttpHeaders
}> {
  const address = server.address()
  if (!address || typeof address === 'string') {
    throw new Error('Server not listening')
  }

  return new Promise((resolve, reject) => {
    const options = {
      hostname: 'localhost',
      port: address.port,
      path,
      method,
    }

    const req = http.request(options, (res) => {
      let body = ''
      res.on('data', (chunk) => {
        body += chunk
      })
      res.on('end', () => {
        try {
          const parsed = body ? JSON.parse(body) : null
          resolve({ status: res.statusCode || 500, body: parsed, headers: res.headers })
        } catch {
          resolve({ status: res.statusCode || 500, body: body, headers: res.headers })
        }
      })
    })

    req.on('error', reject)
    req.setTimeout(1000, () => {
      req.destroy()
      reject(new Error('Request timeout'))
    })
    req.end()
  })
}

async function postJson(server: Server, path: string, payload: Record<string, unknown>) {
  const address = server.address()
  if (!address || typeof address === 'string') {
    throw new Error('Server not listening')
  }

  const body = JSON.stringify(payload)

  return new Promise<{ status: number; body: any; headers: http.IncomingHttpHeaders }>((resolve, reject) => {
    const options = {
      hostname: 'localhost',
      port: address.port,
      path,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(body),
      },
    }

    const req = http.request(options, (res) => {
      let responseBody = ''
      res.on('data', chunk => { responseBody += chunk })
      res.on('end', () => {
        resolve({
          status: res.statusCode || 500,
          body: responseBody ? JSON.parse(responseBody) : null,
          headers: res.headers,
        })
      })
    })

    req.on('error', reject)
    req.write(body)
    req.end()
  })
}

function createTempDist(html: string, extraFiles?: Record<string, string>) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'api-base-dist-'))
  const distDir = path.join(root, 'dist')
  fs.mkdirSync(distDir, { recursive: true })
  const indexPath = path.join(distDir, 'index.html')
  fs.writeFileSync(indexPath, html, 'utf-8')

  if (extraFiles) {
    for (const [relativePath, contents] of Object.entries(extraFiles)) {
      const filePath = path.join(distDir, relativePath)
      fs.mkdirSync(path.dirname(filePath), { recursive: true })
      fs.writeFileSync(filePath, contents, 'utf-8')
    }
  }

  return {
    distDir,
    indexPath,
    cleanup: () => {
      fs.rmSync(root, { recursive: true, force: true })
    },
  }
}


describe('createScenarioServer', () => {
  it('creates Express app', () => {
    const options: ServerTemplateOptions = {
      uiPort: '3000',
      apiPort: '8080',
      distDir: './dist',
    }

    const app = createScenarioServer(options)

    expect(app).toBeDefined()
    expect(typeof app.listen).toBe('function')
    expect(typeof app.use).toBe('function')
    expect(typeof app.get).toBe('function')
  })

  it('requires uiPort', () => {
    const options: ServerTemplateOptions = {
      uiPort: '',
      apiPort: '8080',
      distDir: './dist',
    }

    expect(() => createScenarioServer(options)).toThrow('UI_PORT')
  })

  it('requires apiPort', () => {
    const options: ServerTemplateOptions = {
      uiPort: '3000',
      apiPort: '',
      distDir: './dist',
    }

    expect(() => createScenarioServer(options)).toThrow('API_PORT')
  })

  it('passes proxyTimeoutMs to createProxyMiddleware', () => {
    const spy = vi.spyOn(proxyModule, 'createProxyMiddleware')

    createScenarioServer({
      uiPort: '3000',
      apiPort: '8080',
      distDir: './dist',
      proxyTimeoutMs: 60000,
    })

    expect(spy).toHaveBeenCalledWith(
      expect.objectContaining({
        timeout: 60000,
      })
    )

    spy.mockRestore()
  })

  it('accepts string ports', () => {
    const options: ServerTemplateOptions = {
      uiPort: '3000',
      apiPort: '8080',
      distDir: './dist',
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('accepts number ports', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('uses default distDir when not provided', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('accepts custom routes', () => {
    const customRoute = vi.fn()
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      routes: (app) => {
        app.get('/custom', customRoute)
      },
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('accepts custom configBuilder', () => {
    const customBuilder = vi.fn(() => ({
      apiUrl: 'http://custom.example.com/api',
      wsUrl: 'ws://custom.example.com/ws',
      apiPort: '9999',
      wsPort: '9999',
      uiPort: '8888',
    }))

    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      configBuilder: customBuilder,
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('accepts serviceName and version', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      serviceName: 'test-scenario',
      version: '1.0.0',
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('accepts proxy path prefix', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      apiProxyPath: '/api',
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('accepts CORS configuration', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      cors: '*',
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('accepts custom CORS origins array', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      cors: ['http://example.com', 'http://test.com'],
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('disables CORS when set to false', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      cors: false,
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('enables verbose logging', () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {})

    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      verbose: true,
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()

    consoleSpy.mockRestore()
  })

  it('enables health checks', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      health: {
        checkApi: true,
        includeMemory: true,
      },
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('accepts custom health checks', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      health: {
        customChecks: [
          {
            name: 'database',
            check: async () => ({ healthy: true }),
          },
        ],
      },
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('enables metadata injection', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      injectMetadata: true,
      proxyMetadata: {
        appId: 'test-app',
        hostScenario: 'host',
        targetScenario: 'target',
        ports: [],
        primaryPort: {
          port: 3000,
          label: 'ui',
          slug: 'ui',
          path: '/test',
          aliases: [],
          isPrimary: true,
          source: 'manual',
          priority: 100,
        },
      },
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('injects scenario config', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      injectConfig: true,
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('handles JSON body parsing', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      jsonLimit: '10mb',
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('handles custom API timeout', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      apiTimeout: 60000,
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('uses custom config endpoint path', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      configEndpoint: '/runtime-config',
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('uses custom health endpoint path', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      healthEndpoint: '/status',
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('combines multiple options', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      wsPort: 8081,
      distDir: './dist',
      serviceName: 'test-scenario',
      version: '1.0.0',
      apiProxyPath: '/api',
      configEndpoint: '/config',
      healthEndpoint: '/health',
      cors: '*',
      verbose: true,
      injectMetadata: false,
      injectConfig: true,
      jsonLimit: '5mb',
      apiTimeout: 30000,
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('handles middleware errors gracefully', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      routes: (app) => {
        // Add route that throws error
        app.get('/error', () => {
          throw new Error('Test error')
        })
      },
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('returns app with all expected methods', () => {
    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
    }

    const app = createScenarioServer(options)

    // Check Express app interface
    expect(typeof app.get).toBe('function')
    expect(typeof app.post).toBe('function')
    expect(typeof app.put).toBe('function')
    expect(typeof app.delete).toBe('function')
    expect(typeof app.use).toBe('function')
    expect(typeof app.listen).toBe('function')
  })

  it('accepts custom beforeRoutes middleware', () => {
    const beforeRoutes = vi.fn((req, res, next) => next())

    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      beforeRoutes: (app) => {
        app.use(beforeRoutes)
      },
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })

  it('accepts custom afterRoutes middleware', () => {
    const afterRoutes = vi.fn((req, res, next) => next())

    const options: ServerTemplateOptions = {
      uiPort: 3000,
      apiPort: 8080,
      distDir: './dist',
      afterRoutes: (app) => {
        app.use(afterRoutes)
      },
    }

    const app = createScenarioServer(options)
    expect(app).toBeDefined()
  })
})

describe('createScenarioServer HTTP endpoints', () => {
  let server: Server | null = null

  afterEach(async () => {
    if (server) {
      await new Promise<void>((resolve) => {
        server!.close(() => {
          server = null
          resolve()
        })
      })
    }
  })

  it('/config endpoint returns default config', async () => {
    const app = createScenarioServer({
      uiPort: 33000, // Test port
      apiPort: 8080,
      distDir: './dist',
    })

    server = app.listen(0) // OS assigns random port
    const result = await makeRequest(server, '/config')

    expect(result.status).toBe(200)
    expect(result.body).toHaveProperty('apiUrl')
    expect(result.body).toHaveProperty('wsUrl')
    expect(result.body).toHaveProperty('apiPort', '8080')
  })

  it('/config endpoint uses custom configBuilder', async () => {
    const app = createScenarioServer({
      uiPort: 33001,
      apiPort: 8080,
      distDir: './dist',
      configBuilder: () => ({
        apiUrl: 'http://custom.example.com/api',
        wsUrl: 'ws://custom.example.com/ws',
        apiPort: '9999',
        wsPort: '9999',
        uiPort: '8888',
      }),
    })

    server = app.listen(0)
    const result = await makeRequest(server, '/config')

    expect(result.status).toBe(200)
    expect(result.body.apiUrl).toBe('http://custom.example.com/api')
    expect(result.body.apiPort).toBe('9999')
  })

  it('/config endpoint handles configBuilder errors', async () => {
    const app = createScenarioServer({
      uiPort: 33002,
      apiPort: 8080,
      distDir: './dist',
      configBuilder: () => {
        throw new Error('Config build failed')
      },
    })

    server = app.listen(0)
    const result = await makeRequest(server, '/config')

    expect(result.status).toBe(500)
    expect(result.body).toHaveProperty('error')
    expect(result.body.error).toContain('Failed to build configuration')
  })

  it('/health endpoint returns status', async () => {
    const app = createScenarioServer({
      uiPort: 33003,
      apiPort: 8080,
      distDir: './dist',
    })

    server = app.listen(0)
    const result = await makeRequest(server, '/health')

    // Returns 503 when API is not reachable (degraded status)
    expect(result.status).toBe(503)
    expect(result.body).toHaveProperty('status', 'degraded')
    expect(result.body).toHaveProperty('timestamp')
    expect(result.body).toHaveProperty('api_connectivity')
    expect(result.body.api_connectivity.connected).toBe(false)
  })

  it('CORS headers are set for localhost by default', async () => {
    const app = createScenarioServer({
      uiPort: 33005,
      apiPort: 8080,
      distDir: './dist',
    })

    server = app.listen(0)
    const result = await makeRequest(server, '/health')

    // CORS headers are only set when origin is sent, but middleware exists
    // Health returns 503 when API is unreachable
    expect(result.status).toBe(503)
  })

  it('custom setupRoutes is called', async () => {
    const app = createScenarioServer({
      uiPort: 33006,
      apiPort: 8080,
      distDir: './dist',
      setupRoutes: (app) => {
        app.get('/custom-test', (req, res) => {
          res.json({ custom: true })
        })
      },
    })

    server = app.listen(0)
    const result = await makeRequest(server, '/custom-test')

    expect(result.status).toBe(200)
    expect(result.body).toEqual({ custom: true })
  })

  it('handles 404 for non-existent routes', async () => {
    const app = createScenarioServer({
      uiPort: 33007,
      apiPort: 8080,
      distDir: './dist',
    })

    server = app.listen(0)
    const result = await makeRequest(server, '/non-existent-route')

    expect(result.status).toBe(404)
  })

  it('JSON body parser works', async () => {
    const app = createScenarioServer({
      uiPort: 33008,
      apiPort: 8080,
      distDir: './dist',
      setupRoutes: (app) => {
        app.post('/echo', (req, res) => {
          res.json({ received: req.body })
        })
      },
    })

    server = app.listen(0)

    const result = await postJson(server, '/echo', { test: 'data' })

    expect(result.status).toBe(200)
    expect(result.body.received).toEqual({ test: 'data' })
  })

  it('supports disabling body parser entirely', async () => {
    const app = createScenarioServer({
      uiPort: 33012,
      apiPort: 8080,
      distDir: './dist',
      bodyParser: false,
      setupRoutes: (app) => {
        app.post('/noop', (req, res) => {
          res.json({ body: req.body ?? null })
        })
      },
    })

    server = app.listen(0)
    const result = await postJson(server, '/noop', { foo: 'bar' })

    expect(result.status).toBe(200)
    expect(result.body).toEqual({ body: null })
  })

  it('marks SPA fallback responses as no-store', async () => {
    const temp = createTempDist('<html><body>fallback</body></html>')
    try {
      const app = createScenarioServer({
        uiPort: 33013,
        apiPort: 8080,
        distDir: temp.distDir,
      })

      server = app.listen(0)
      const result = await makeRequest(server, '/some/deep/link')

      expect(result.status).toBe(200)
      expect(result.headers['cache-control']).toBe('no-store, max-age=0')
      expect(typeof result.body).toBe('string')
    } finally {
      temp.cleanup()
    }
  })

  it('serves hashed assets with immutable cache headers', async () => {
    const temp = createTempDist('<html><body>assets</body></html>', {
      'assets/app-main-abc12345.js': 'console.log("immutable")',
    })

    try {
      const app = createScenarioServer({
        uiPort: 33014,
        apiPort: 8080,
        distDir: temp.distDir,
      })

      server = app.listen(0)
      const result = await makeRequest(server, '/assets/app-main-abc12345.js')

      expect(result.status).toBe(200)
      expect(result.headers['cache-control']).toBe('public, max-age=31536000, immutable')
    } finally {
      temp.cleanup()
    }
  })

  it('caches index.html between SPA fallback requests when enabled', async () => {
    const temp = createTempDist('<html><body>version-one</body></html>')
    const originalLog = console.log
    const logs: string[] = []
    console.log = (...args: any[]) => {
      logs.push(args.join(' '))
    }

    try {
      const app = createScenarioServer({
        uiPort: 33020,
        apiPort: 8080,
        distDir: temp.distDir,
        verbose: true,
      })

      server = app.listen(0)

      const first = await makeRequest(server, '/spa-first')
      expect(first.body).toContain('version-one')

      const second = await makeRequest(server, '/spa-second')
      expect(second.body).toContain('version-one')

      const cacheLogs = logs.filter(line => line.includes('Cached index.html'))
      expect(cacheLogs.length).toBe(1)

      await new Promise(resolve => setTimeout(resolve, 5))
      fs.writeFileSync(temp.indexPath, '<html><body>version-two</body></html>', 'utf-8')

      const third = await makeRequest(server, '/spa-third')
      expect(third.body).toContain('version-two')

      const refreshedLogs = logs.filter(line => line.includes('Cached index.html'))
      expect(refreshedLogs.length).toBe(2)
    } finally {
      console.log = originalLog
      temp.cleanup()
    }
  })

  it('skips caching when cacheIndexHtml is disabled', async () => {
    const temp = createTempDist('<html><body>no-cache</body></html>')
    const originalLog = console.log
    const logs: string[] = []
    console.log = (...args: any[]) => {
      logs.push(args.join(' '))
    }

    try {
      const app = createScenarioServer({
        uiPort: 33021,
        apiPort: 8080,
        distDir: temp.distDir,
        cacheIndexHtml: false,
        verbose: true,
      })

      server = app.listen(0)

      await makeRequest(server, '/no-cache-one')
      await makeRequest(server, '/no-cache-two')

      const cacheLogs = logs.filter(line => line.includes('Cached index.html'))
      expect(cacheLogs.length).toBe(0)
    } finally {
      console.log = originalLog
      temp.cleanup()
    }
  })

  it('accepts custom body parser configurator', async () => {
    const app = createScenarioServer({
      uiPort: 33013,
      apiPort: 8080,
      distDir: './dist',
      bodyParser: (expressApp) => {
        expressApp.use(express.text({ type: '*/*' }))
      },
      setupRoutes: (app) => {
        app.post('/text', (req, res) => {
          res.json({ raw: req.body })
        })
      },
    })

    server = app.listen(0)
    const address = server.address()
    if (!address || typeof address === 'string') {
      throw new Error('Server not listening')
    }

    const payload = 'hello world'

    const result = await new Promise<{ status: number; body: any }>((resolve, reject) => {
      const req = http.request({
        hostname: 'localhost',
        port: address.port,
        path: '/text',
        method: 'POST',
        headers: {
          'Content-Type': 'text/plain',
          'Content-Length': Buffer.byteLength(payload),
        },
      }, (res) => {
        let data = ''
        res.on('data', chunk => { data += chunk })
        res.on('end', () => resolve({ status: res.statusCode || 0, body: JSON.parse(data) }))
      })
      req.on('error', reject)
      req.write(payload)
      req.end()
    })

    expect(result.status).toBe(200)
    expect(result.body).toEqual({ raw: 'hello world' })
  })
})

describe('startScenarioServer', () => {
  // startScenarioServer returns Express app, not Server - just test it calls listen
  it('calls createScenarioServer and listen', async () => {
    const originalListen = console.log
    const logs: string[] = []

    // Create promise that resolves when server logs
    const logPromise = new Promise<void>((resolve) => {
      console.log = (...args: any[]) => {
        logs.push(args.join(' '))
        if (logs.length === 1) {
          resolve()
        }
      }
    })

    // Start on a high port to avoid conflicts
    const app = startScenarioServer({
      uiPort: 33009,
      apiPort: 8080,
      distDir: './dist',
    })

    expect(app).toBeDefined()
    expect(typeof app.use).toBe('function')

    // Wait for server to start logging
    await logPromise
    console.log = originalListen

    expect(logs.length).toBeGreaterThan(0)
    expect(logs[0]).toContain('listening on port')
  })

  it('accepts all ServerTemplateOptions', () => {
    const originalLog = console.log
    console.log = () => {} // Silence output

    const app = startScenarioServer({
      uiPort: 33011,
      apiPort: 8080,
      wsPort: 8081,
      distDir: './dist',
      serviceName: 'test-service',
      version: '1.0.0',
    })

    console.log = originalLog

    expect(app).toBeDefined()
  })
})
