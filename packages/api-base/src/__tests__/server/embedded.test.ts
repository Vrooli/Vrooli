/**
 * Tests for server/embedded.ts
 *
 * Covers:
 * - createPortResolver: env var lookup, CLI fallback, TTL caching, negative lookups
 * - resolveExternalUrl: localhost, tunnel/remote, fallback
 * - createEmbeddedProxyRouter: external-url endpoint, proxy passthrough, error paths,
 *   allowedScenarios gate, timeout, proxy error handling
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as http from 'node:http'
import express from 'express'
import type { Server } from 'node:http'
import {
  createPortResolver,
  resolveExternalUrl,
  createEmbeddedProxyRouter,
} from '../../server/embedded.js'
import { mockRequest, mockResponse } from '../helpers/mock-request.js'

// ──────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────

/** Start an Express app with the embedded router mounted at /embedded and return server + port */
async function startEmbeddedServer(
  options: Parameters<typeof createEmbeddedProxyRouter>[0] = {},
): Promise<{ server: Server; port: number }> {
  const app = express()
  app.use('/embedded', createEmbeddedProxyRouter(options))

  return new Promise((resolve) => {
    const server = app.listen(0, '127.0.0.1', () => {
      const addr = server.address()
      if (!addr || typeof addr === 'string') throw new Error('No address')
      resolve({ server, port: addr.port })
    })
  })
}

/** Start a tiny upstream HTTP server that echoes back request info */
async function startUpstreamServer(): Promise<{ server: Server; port: number }> {
  const server = http.createServer((req, res) => {
    if (req.url === '/health') {
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ ok: true }))
      return
    }
    if (req.url === '/') {
      res.writeHead(200, { 'Content-Type': 'text/html' })
      res.end('<html><body>hello</body></html>')
      return
    }
    if (req.url?.startsWith('/api/data')) {
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ path: req.url, method: req.method }))
      return
    }
    // Slow endpoint for timeout testing — never responds
    if (req.url === '/hang') {
      // intentionally do not respond
      return
    }
    res.writeHead(404)
    res.end('not found')
  })

  return new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      const addr = server.address()
      if (!addr || typeof addr === 'string') throw new Error('No address')
      resolve({ server, port: addr.port })
    })
  })
}

/** Make an HTTP request and return status + parsed JSON body */
async function makeRequest(
  port: number,
  path: string,
  options: { method?: string; headers?: Record<string, string> } = {},
): Promise<{ status: number; body: any; headers: http.IncomingHttpHeaders }> {
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
        let raw = ''
        res.on('data', (chunk) => { raw += chunk })
        res.on('end', () => {
          let body: any = raw
          try { body = JSON.parse(raw) } catch { /* keep raw */ }
          resolve({ status: res.statusCode || 500, body, headers: res.headers })
        })
      },
    )
    req.on('error', reject)
    req.setTimeout(5000, () => {
      req.destroy()
      reject(new Error('Test request timeout'))
    })
    req.end()
  })
}

function closeServer(server: Server): Promise<void> {
  return new Promise((resolve) => server.close(() => resolve()))
}

// ──────────────────────────────────────────────────────────────────────
// createPortResolver
// ──────────────────────────────────────────────────────────────────────

describe('createPortResolver', () => {
  const originalEnv = { ...process.env }

  afterEach(() => {
    // Restore env
    for (const key of Object.keys(process.env)) {
      if (!(key in originalEnv)) delete process.env[key]
    }
    Object.assign(process.env, originalEnv)
  })

  it('resolves port from environment variable', () => {
    process.env.MY_SCENARIO_UI_PORT = '12345'
    const resolve = createPortResolver()
    expect(resolve('my-scenario')).toBe('12345')
  })

  it('converts hyphens to underscores and uppercases for env lookup', () => {
    process.env.AGENT_INBOX_UI_PORT = '9999'
    const resolve = createPortResolver()
    expect(resolve('agent-inbox')).toBe('9999')
  })

  it('returns null when env var is missing and vrooli CLI fails', () => {
    // No env var set and vrooli CLI won't be available in test
    const resolve = createPortResolver()
    expect(resolve('nonexistent-scenario')).toBeNull()
  })

  it('caches resolved port for cacheTtlMs', () => {
    process.env.CACHED_SCENARIO_UI_PORT = '1111'
    const resolve = createPortResolver(60_000)

    // First call resolves
    expect(resolve('cached-scenario')).toBe('1111')

    // Remove env var — should still return cached value
    delete process.env.CACHED_SCENARIO_UI_PORT
    expect(resolve('cached-scenario')).toBe('1111')
  })

  it('expires cache after TTL', async () => {
    process.env.TTL_SCENARIO_UI_PORT = '2222'
    const resolve = createPortResolver(50) // 50ms TTL

    expect(resolve('ttl-scenario')).toBe('2222')

    // Wait for TTL to expire
    await new Promise((r) => setTimeout(r, 80))

    // Remove env var — cache expired, should return null (CLI also fails)
    delete process.env.TTL_SCENARIO_UI_PORT
    expect(resolve('ttl-scenario')).toBeNull()
  })

  it('does not cache negative lookups', () => {
    const resolve = createPortResolver(60_000)

    // First call — not found
    expect(resolve('missing-scenario')).toBeNull()

    // Set env var — should find it immediately (not cached as null)
    process.env.MISSING_SCENARIO_UI_PORT = '3333'
    expect(resolve('missing-scenario')).toBe('3333')

    delete process.env.MISSING_SCENARIO_UI_PORT
  })

  it('uses default TTL of 30000ms', () => {
    process.env.DEFAULT_TTL_UI_PORT = '4444'
    const resolve = createPortResolver() // default TTL

    expect(resolve('default-ttl')).toBe('4444')

    // Remove env var — should still return cached value (within 30s)
    delete process.env.DEFAULT_TTL_UI_PORT
    expect(resolve('default-ttl')).toBe('4444')
  })

  it('handles multiple scenarios independently', () => {
    process.env.SCENARIO_A_UI_PORT = '5555'
    process.env.SCENARIO_B_UI_PORT = '6666'
    const resolve = createPortResolver()

    expect(resolve('scenario-a')).toBe('5555')
    expect(resolve('scenario-b')).toBe('6666')
  })
})

// ──────────────────────────────────────────────────────────────────────
// resolveExternalUrl
// ──────────────────────────────────────────────────────────────────────

describe('resolveExternalUrl', () => {
  it('returns localhost URL when host is localhost', () => {
    const req = mockRequest({ headers: { host: 'localhost:3000' } })
    const url = resolveExternalUrl(req, 'my-scenario', '9000')
    expect(url).toBe('http://localhost:9000')
  })

  it('returns localhost URL when host is 127.0.0.1', () => {
    const req = mockRequest({ headers: { host: '127.0.0.1:3000' } })
    const url = resolveExternalUrl(req, 'my-scenario', '9000')
    expect(url).toBe('http://localhost:9000')
  })

  it('returns localhost URL when host is 0.0.0.0', () => {
    const req = mockRequest({ headers: { host: '0.0.0.0:3000' } })
    const url = resolveExternalUrl(req, 'my-scenario', '9000')
    expect(url).toBe('http://localhost:9000')
  })

  it('returns localhost URL when host header is missing', () => {
    const req = mockRequest({ headers: {} })
    const url = resolveExternalUrl(req, 'my-scenario', '9000')
    expect(url).toBe('http://localhost:9000')
  })

  it('swaps first subdomain for tunnel URLs with 3+ domain parts', () => {
    const req = mockRequest({ headers: { host: 'git-control-tower.example.com' } })
    const url = resolveExternalUrl(req, 'my-scenario', '9000')
    expect(url).toBe('https://my-scenario.example.com')
  })

  it('swaps subdomain for deep domain names', () => {
    const req = mockRequest({ headers: { host: 'app.staging.example.com' } })
    const url = resolveExternalUrl(req, 'test-app', '9000')
    expect(url).toBe('https://test-app.staging.example.com')
  })

  it('falls back to localhost for two-part domain names', () => {
    const req = mockRequest({ headers: { host: 'example.com' } })
    const url = resolveExternalUrl(req, 'my-scenario', '9000')
    expect(url).toBe('http://localhost:9000')
  })

  it('falls back to localhost for single-word hosts', () => {
    const req = mockRequest({ headers: { host: 'myhost' } })
    const url = resolveExternalUrl(req, 'my-scenario', '9000')
    expect(url).toBe('http://localhost:9000')
  })

  it('strips port from host header before domain analysis', () => {
    const req = mockRequest({ headers: { host: 'app.example.com:8080' } })
    const url = resolveExternalUrl(req, 'my-scenario', '9000')
    expect(url).toBe('https://my-scenario.example.com')
  })
})

// ──────────────────────────────────────────────────────────────────────
// createEmbeddedProxyRouter — external-url endpoint
// ──────────────────────────────────────────────────────────────────────

describe('createEmbeddedProxyRouter', () => {
  let upstream: { server: Server; port: number }
  let embedded: { server: Server; port: number }

  afterEach(async () => {
    if (embedded?.server) await closeServer(embedded.server)
    if (upstream?.server) await closeServer(upstream.server)
  })

  describe('GET /:scenario/external-url', () => {
    it('returns context-aware URL for a known scenario', async () => {
      upstream = await startUpstreamServer()
      // Set env var so port resolves
      process.env.TEST_SCENARIO_UI_PORT = String(upstream.port)

      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/test-scenario/external-url')

      expect(res.status).toBe(200)
      expect(res.body.url).toBe(`http://localhost:${upstream.port}`)
      expect(res.body.scenario).toBe('test-scenario')

      delete process.env.TEST_SCENARIO_UI_PORT
    })

    it('returns 503 when scenario port cannot be resolved', async () => {
      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/unknown-scenario/external-url')

      expect(res.status).toBe(503)
      expect(res.body.error).toBe('Scenario not available')
      expect(res.body.scenario).toBe('unknown-scenario')
    })

    it('returns 403 when scenario is not in allowedScenarios', async () => {
      process.env.BLOCKED_SCENARIO_UI_PORT = '9999'
      embedded = await startEmbeddedServer({ allowedScenarios: ['allowed-only'] })
      const res = await makeRequest(embedded.port, '/embedded/blocked-scenario/external-url')

      expect(res.status).toBe(403)
      expect(res.body.error).toBe('Scenario not allowed')
      expect(res.body.scenario).toBe('blocked-scenario')

      delete process.env.BLOCKED_SCENARIO_UI_PORT
    })

    it('allows scenario that IS in allowedScenarios list', async () => {
      upstream = await startUpstreamServer()
      process.env.ALLOWED_APP_UI_PORT = String(upstream.port)

      embedded = await startEmbeddedServer({ allowedScenarios: ['allowed-app'] })
      const res = await makeRequest(embedded.port, '/embedded/allowed-app/external-url')

      expect(res.status).toBe(200)
      expect(res.body.scenario).toBe('allowed-app')

      delete process.env.ALLOWED_APP_UI_PORT
    })
  })

  describe('proxy passthrough (USE /:scenario)', () => {
    it('proxies GET request to upstream and returns response', async () => {
      upstream = await startUpstreamServer()
      process.env.PROXY_TARGET_UI_PORT = String(upstream.port)

      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/proxy-target/health')

      expect(res.status).toBe(200)
      expect(res.body).toEqual({ ok: true })

      delete process.env.PROXY_TARGET_UI_PORT
    })

    it('proxies root path when only scenario name is given', async () => {
      upstream = await startUpstreamServer()
      process.env.ROOT_PROXY_UI_PORT = String(upstream.port)

      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/root-proxy/')

      expect(res.status).toBe(200)
      expect(res.body).toContain('hello')

      delete process.env.ROOT_PROXY_UI_PORT
    })

    it('strips /embedded/{scenario} prefix before proxying', async () => {
      upstream = await startUpstreamServer()
      process.env.STRIP_TEST_UI_PORT = String(upstream.port)

      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/strip-test/api/data?foo=bar')

      expect(res.status).toBe(200)
      expect(res.body.path).toBe('/api/data?foo=bar')

      delete process.env.STRIP_TEST_UI_PORT
    })

    it('returns 503 when scenario is not available for proxy', async () => {
      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/no-such-scenario/anything')

      expect(res.status).toBe(503)
      expect(res.body.error).toBe('Scenario not available')
    })

    it('returns 403 when scenario not in allowed list for proxy', async () => {
      process.env.FORBIDDEN_UI_PORT = '9999'
      embedded = await startEmbeddedServer({ allowedScenarios: ['only-this'] })
      const res = await makeRequest(embedded.port, '/embedded/forbidden/')

      expect(res.status).toBe(403)
      expect(res.body.error).toBe('Scenario not allowed')

      delete process.env.FORBIDDEN_UI_PORT
    })

    it('returns 502 when upstream is unreachable', async () => {
      // Point to a port with nothing listening
      process.env.DEAD_UPSTREAM_UI_PORT = '19'

      const originalError = console.error
      console.error = vi.fn() // Suppress expected error log

      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/dead-upstream/')

      expect(res.status).toBe(502)
      expect(res.body.error).toBe('Failed to proxy to scenario UI')
      expect(res.body.scenario).toBe('dead-upstream')

      console.error = originalError
      delete process.env.DEAD_UPSTREAM_UI_PORT
    })

    it('respects custom upstreamHost', async () => {
      upstream = await startUpstreamServer()
      // Use 127.0.0.1 explicitly — same as default but confirming it works
      process.env.CUSTOM_HOST_UI_PORT = String(upstream.port)

      embedded = await startEmbeddedServer({ upstreamHost: '127.0.0.1' })
      const res = await makeRequest(embedded.port, '/embedded/custom-host/health')

      expect(res.status).toBe(200)
      expect(res.body).toEqual({ ok: true })

      delete process.env.CUSTOM_HOST_UI_PORT
    })

    it('returns 404 for paths upstream doesn\'t handle', async () => {
      upstream = await startUpstreamServer()
      process.env.NOTFOUND_TEST_UI_PORT = String(upstream.port)

      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/notfound-test/nonexistent')

      expect(res.status).toBe(404)

      delete process.env.NOTFOUND_TEST_UI_PORT
    })
  })

  describe('proxy timeout', () => {
    it('returns 504 when upstream does not respond within timeoutMs', async () => {
      upstream = await startUpstreamServer()
      process.env.TIMEOUT_SCENARIO_UI_PORT = String(upstream.port)

      const originalError = console.error
      console.error = vi.fn()

      embedded = await startEmbeddedServer({ timeoutMs: 200 })
      const res = await makeRequest(embedded.port, '/embedded/timeout-scenario/hang')

      expect(res.status).toBe(504)
      expect(res.body.error).toBe('Proxy timeout')
      expect(res.body.scenario).toBe('timeout-scenario')

      console.error = originalError
      delete process.env.TIMEOUT_SCENARIO_UI_PORT
    })
  })

  describe('port resolution caching in router context', () => {
    it('caches port resolution across multiple requests', async () => {
      upstream = await startUpstreamServer()
      process.env.CACHE_HIT_UI_PORT = String(upstream.port)

      embedded = await startEmbeddedServer({ cacheTtlMs: 60_000 })

      // First request — resolves and caches
      const res1 = await makeRequest(embedded.port, '/embedded/cache-hit/health')
      expect(res1.status).toBe(200)

      // Remove env var — should still work from cache
      delete process.env.CACHE_HIT_UI_PORT

      const res2 = await makeRequest(embedded.port, '/embedded/cache-hit/health')
      expect(res2.status).toBe(200)
    })
  })
})

// ──────────────────────────────────────────────────────────────────────
// Template integration: embeddedProxy option
// ──────────────────────────────────────────────────────────────────────

describe('createScenarioServer with embeddedProxy', () => {
  // These tests verify that template.ts correctly wires in the embedded router.
  // We use a valid high port number for uiPort since parsePort rejects 0.
  const TEMPLATE_UI_PORT = 33050

  let server: Server

  /** Listen on port 0 and wait until server is ready */
  function listenAndWait(app: any): Promise<number> {
    return new Promise((resolve, reject) => {
      server = app.listen(0, '127.0.0.1', () => {
        const addr = server.address()
        if (!addr || typeof addr === 'string') return reject(new Error('No address'))
        resolve(addr.port)
      })
    })
  }

  afterEach(async () => {
    if (server) await closeServer(server)
  })

  it('registers /embedded routes when embeddedProxy is true', async () => {
    const { createScenarioServer } = await import('../../server/template.js')
    process.env.TMPL_SCENARIO_UI_PORT = '12345'

    const app = createScenarioServer({
      uiPort: TEMPLATE_UI_PORT,
      apiPort: 8080,
      distDir: './dist',
      embeddedProxy: true,
    })

    const port = await listenAndWait(app)
    const res = await makeRequest(port, '/embedded/tmpl-scenario/external-url')
    expect(res.status).toBe(200)
    expect(res.body.scenario).toBe('tmpl-scenario')

    delete process.env.TMPL_SCENARIO_UI_PORT
  })

  it('registers /embedded routes when embeddedProxy is an options object', async () => {
    const { createScenarioServer } = await import('../../server/template.js')
    process.env.OPT_SCENARIO_UI_PORT = '12345'

    const app = createScenarioServer({
      uiPort: TEMPLATE_UI_PORT,
      apiPort: 8080,
      distDir: './dist',
      embeddedProxy: { cacheTtlMs: 5000 },
    })

    const port = await listenAndWait(app)
    const res = await makeRequest(port, '/embedded/opt-scenario/external-url')
    expect(res.status).toBe(200)
    expect(res.body.scenario).toBe('opt-scenario')

    delete process.env.OPT_SCENARIO_UI_PORT
  })

  it('does not register /embedded routes when embeddedProxy.enabled is false', async () => {
    const { createScenarioServer } = await import('../../server/template.js')

    const app = createScenarioServer({
      uiPort: TEMPLATE_UI_PORT,
      apiPort: 8080,
      distDir: './dist',
      embeddedProxy: { enabled: false },
    })

    const port = await listenAndWait(app)
    const res = await makeRequest(port, '/embedded/any-scenario/external-url')
    // Should fall through to SPA fallback skip (404 for /embedded/ paths)
    expect(res.status).toBe(404)
  })

  it('does not register /embedded routes when embeddedProxy is not set', async () => {
    const { createScenarioServer } = await import('../../server/template.js')

    const app = createScenarioServer({
      uiPort: TEMPLATE_UI_PORT,
      apiPort: 8080,
      distDir: './dist',
    })

    const port = await listenAndWait(app)
    const res = await makeRequest(port, '/embedded/any-scenario/external-url')
    // Should fall through to SPA fallback skip (404 for /embedded/ paths)
    expect(res.status).toBe(404)
  })

  it('SPA fallback skips /embedded/ paths', async () => {
    const { createScenarioServer } = await import('../../server/template.js')

    const app = createScenarioServer({
      uiPort: TEMPLATE_UI_PORT,
      apiPort: 8080,
      distDir: './dist',
      verbose: true,
    })

    const port = await listenAndWait(app)
    const res = await makeRequest(port, '/embedded/some-scenario/page')
    // Must NOT serve index.html — should be 404
    expect(res.status).toBe(404)
    expect(typeof res.body === 'string' ? res.body : '').not.toContain('<!DOCTYPE html>')
  })
})
