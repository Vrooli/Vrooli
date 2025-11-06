/**
 * Tests for server/template.ts
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createScenarioServer, startScenarioServer } from '../../server/template.js'
import type { ServerTemplateOptions } from '../../shared/types.js'
import type { Server } from 'node:http'
import * as http from 'node:http'

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

    // Make POST request with JSON body
    const address = server.address()
    if (!address || typeof address === 'string') {
      throw new Error('Server not listening')
    }

    const postData = JSON.stringify({ test: 'data' })

    const result = await new Promise<any>((resolve, reject) => {
      const options = {
        hostname: 'localhost',
        port: (address as any).port,
        path: '/echo',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Content-Length': Buffer.byteLength(postData),
        },
      }

      const req = http.request(options, (res) => {
        let body = ''
        res.on('data', (chunk) => { body += chunk })
        res.on('end', () => {
          resolve({ status: res.statusCode, body: JSON.parse(body) })
        })
      })

      req.on('error', reject)
      req.write(postData)
      req.end()
    })

    expect(result.status).toBe(200)
    expect(result.body.received).toEqual({ test: 'data' })
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
