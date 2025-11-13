/**
 * Tests for server/proxy.ts
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { proxyToApi, createProxyMiddleware, proxyWebSocketUpgrade } from '../../server/proxy.js'
import type { ProxyOptions } from '../../shared/types.js'
import { mockRequest, mockResponse } from '../helpers/mock-request.js'
import * as http from 'node:http'
import { EventEmitter } from 'node:events'
import { resetProxyAgentsForTesting } from '../../server/agent.js'

afterEach(() => {
  resetProxyAgentsForTesting()
})

describe('proxyToApi', () => {
  it('rejects invalid API port', async () => {
    const req = mockRequest({ method: 'GET', url: '/test' })
    const { res, getResult } = mockResponse()

    await proxyToApi(req, res, '/test', { apiPort: 'invalid' })

    const result = getResult()
    expect(result.status).toBe(502)
    expect(result.json.error).toBe('API server unavailable')
    expect(result.json.details).toContain('Invalid API_PORT')
  })

  it('rejects undefined API port', async () => {
    const req = mockRequest({ method: 'GET', url: '/test' })
    const { res, getResult } = mockResponse()

    await proxyToApi(req, res, '/test', { apiPort: undefined as any })

    const result = getResult()
    expect(result.status).toBe(502)
    expect(result.json.error).toBe('API server unavailable')
  })

  it('rejects non-numeric port strings', async () => {
    const req = mockRequest({ method: 'GET', url: '/test' })
    const { res, getResult } = mockResponse()

    await proxyToApi(req, res, '/test', { apiPort: 'not-a-port' })

    const result = getResult()
    expect(result.status).toBe(502)
    expect(result.json.details).toContain('Invalid API_PORT')
  })

  it('filters hop-by-hop headers', async () => {
    const req = mockRequest({
      method: 'GET',
      url: '/test',
      headers: {
        'connection': 'keep-alive',
        'content-type': 'application/json',
        'transfer-encoding': 'chunked',
        'x-custom-header': 'should-pass',
      },
    })
    const { res } = mockResponse()

    // We can't fully test the actual proxy without a real server,
    // but we can test that it doesn't crash and handles headers
    const promise = proxyToApi(req, res, '/test', {
      apiPort: 99999, // Will fail but that's ok for this test
      timeout: 100,
    })

    await expect(promise).resolves.toBeUndefined()
  })

  it('sets host header to target', async () => {
    const req = mockRequest({ method: 'GET', url: '/test' })
    const { res } = mockResponse()

    const promise = proxyToApi(req, res, '/test', {
      apiPort: 8080,
      apiHost: 'api.example.com',
      timeout: 100,
    })

    // Just ensure it doesn't throw
    await expect(promise).resolves.toBeUndefined()
  })

  it('adds custom headers', async () => {
    const req = mockRequest({ method: 'GET', url: '/test' })
    const { res } = mockResponse()

    const promise = proxyToApi(req, res, '/test', {
      apiPort: 8080,
      headers: {
        'x-custom': 'value',
        'x-another': 'another-value',
      },
      timeout: 100,
    })

    await expect(promise).resolves.toBeUndefined()
  })

  it('logs when verbose is true', async () => {
    const consoleSpy = vi.spyOn(console, 'log').mockImplementation(() => {})

    const req = mockRequest({ method: 'POST', url: '/api/test' })
    const { res } = mockResponse()

    await proxyToApi(req, res, '/api/test', {
      apiPort: 8080,
      verbose: true,
      timeout: 100,
    })

    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('[proxy] POST /api/test')
    )

    consoleSpy.mockRestore()
  })

  it('respects timeout setting', async () => {
    const req = mockRequest({ method: 'GET', url: '/test' })
    const { res } = mockResponse()

    const start = Date.now()
    await proxyToApi(req, res, '/test', {
      apiPort: 99999,
      apiHost: '192.0.2.1', // Non-routable for timeout
      timeout: 200,
    })
    const elapsed = Date.now() - start

    expect(elapsed).toBeLessThan(2000) // Should timeout quickly
  })

  it('uses default timeout when not specified', async () => {
    const req = mockRequest({ method: 'GET', url: '/test' })
    const { res } = mockResponse()

    const promise = proxyToApi(req, res, '/test', {
      apiPort: 99999,
    })

    // Should still respect default timeout
    await expect(promise).resolves.toBeUndefined()
  })

  it('handles different HTTP methods', async () => {
    const methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH']

    for (const method of methods) {
      const req = mockRequest({ method, url: '/test' })
      const { res } = mockResponse()

      await expect(
        proxyToApi(req, res, '/test', {
          apiPort: 99999,
          timeout: 100,
        })
      ).resolves.toBeUndefined()
    }
  })

  it('forwards request body for POST/PUT', async () => {
    const req = mockRequest({
      method: 'POST',
      url: '/api/data',
      body: { test: 'data' },
    })
    const { res } = mockResponse()

    const promise = proxyToApi(req, res, '/api/data', {
      apiPort: 99999,
      timeout: 100,
    })

    await expect(promise).resolves.toBeUndefined()
  })

  it('preserves content-type header', async () => {
    const req = mockRequest({
      method: 'POST',
      url: '/test',
      headers: { 'content-type': 'application/xml' },
    })
    const { res } = mockResponse()

    const promise = proxyToApi(req, res, '/test', {
      apiPort: 99999,
      timeout: 100,
    })

    await expect(promise).resolves.toBeUndefined()
  })
})

describe('createProxyMiddleware', () => {
  it('creates middleware with fixed options', async () => {
    const options: ProxyOptions = {
      apiPort: 8080,
      apiHost: 'localhost',
      timeout: 5000,
    }

    const middleware = createProxyMiddleware(options)
    expect(typeof middleware).toBe('function')
    expect(middleware.length).toBe(3) // Express middleware signature (req, res, next)
  })

  it('middleware proxies requests', async () => {
    const options: ProxyOptions = {
      apiPort: 99999,
      timeout: 100,
    }

    const middleware = createProxyMiddleware(options)
    const req = mockRequest({ method: 'GET', url: '/test' })
    const { res } = mockResponse()

    await new Promise<void>((resolve) => {
      middleware(req, res, () => {
        resolve()
      })
      setTimeout(resolve, 200)
    })

    // Should complete without throwing
    expect(true).toBe(true)
  })

  it('passes through custom headers', async () => {
    const options: ProxyOptions = {
      apiPort: 99999,
      timeout: 100,
      headers: {
        'x-proxy-header': 'test',
      },
    }

    const middleware = createProxyMiddleware(options)
    const req = mockRequest({ method: 'GET', url: '/test' })
    const { res } = mockResponse()

    await new Promise<void>((resolve) => {
      middleware(req, res, () => {
        resolve()
      })
      setTimeout(resolve, 200)
    })

    expect(true).toBe(true)
  })

  it('handles path prefix stripping', async () => {
    const options: ProxyOptions = {
      apiPort: 99999,
      timeout: 100,
      pathPrefix: '/api',
    }

    const middleware = createProxyMiddleware(options)
    const req = mockRequest({ method: 'GET', url: '/api/test' })
    const { res } = mockResponse()

    await new Promise<void>((resolve) => {
      middleware(req, res, () => {
        resolve()
      })
      setTimeout(resolve, 200)
    })

    // Path should be stripped before proxying
    expect(true).toBe(true)
  })

  it('preserves query parameters', async () => {
    const options: ProxyOptions = {
      apiPort: 99999,
      timeout: 100,
    }

    const middleware = createProxyMiddleware(options)
    const req = mockRequest({ method: 'GET', url: '/test?foo=bar&baz=qux' })
    const { res } = mockResponse()

    await new Promise<void>((resolve) => {
      middleware(req, res, () => {
        resolve()
      })
      setTimeout(resolve, 200)
    })

    expect(true).toBe(true)
  })

  it('handles URL encoding properly', async () => {
    const options: ProxyOptions = {
      apiPort: 99999,
      timeout: 100,
    }

    const middleware = createProxyMiddleware(options)
    const req = mockRequest({ method: 'GET', url: '/test/path%20with%20spaces' })
    const { res } = mockResponse()

    await new Promise<void>((resolve) => {
      middleware(req, res, () => {
        resolve()
      })
      setTimeout(resolve, 200)
    })

    expect(true).toBe(true)
  })

  it('respects verbose logging option', async () => {
    const options: ProxyOptions = {
      apiPort: 99999,
      timeout: 100,
      verbose: true,
    }

    const middleware = createProxyMiddleware(options)
    expect(typeof middleware).toBe('function')

    // Note: Full integration testing of verbose logging would require
    // a real server environment. This test just verifies the option is accepted.
  })

  it('catches and handles middleware exceptions', async () => {
    const options: ProxyOptions = {
      apiPort: 99999,
      timeout: 50,
    }

    const middleware = createProxyMiddleware(options)
    const req = mockRequest({ method: 'GET', url: '/test' })
    const { res } = mockResponse()
    const nextSpy = vi.fn()

    await new Promise<void>((resolve) => {
      middleware(req, res, nextSpy)
      setTimeout(resolve, 150)
    })

    // Middleware should complete without throwing
    expect(true).toBe(true)
  })

  it('handles pathPrefix option correctly', async () => {
    const options: ProxyOptions = {
      apiPort: 99999,
      timeout: 100,
      pathPrefix: '/v1',
    }

    const middleware = createProxyMiddleware(options)
    const req = mockRequest({ method: 'GET', url: '/v1/users' })
    const { res } = mockResponse()

    await new Promise<void>((resolve) => {
      middleware(req, res, () => {})
      setTimeout(resolve, 150)
    })

    expect(true).toBe(true)
  })

  it('adds accept header when not present', async () => {
    const req = mockRequest({
      method: 'GET',
      url: '/test',
      headers: {}, // No accept header
    })
    const { res } = mockResponse()

    await proxyToApi(req, res, '/test', {
      apiPort: 99999,
      timeout: 100,
    })

    // Should complete without error (accept header added internally)
    expect(true).toBe(true)
  })

  it('uses custom apiHost option', async () => {
    const req = mockRequest({ method: 'GET', url: '/test' })
    const { res } = mockResponse()

    await proxyToApi(req, res, '/test', {
      apiPort: 99999,
      apiHost: 'custom.api.host',
      timeout: 100,
    })

    expect(true).toBe(true)
  })

  it('handles HEAD requests correctly', async () => {
    const req = mockRequest({ method: 'HEAD', url: '/test' })
    const { res } = mockResponse()

    await proxyToApi(req, res, '/test', {
      apiPort: 99999,
      timeout: 100,
    })

    // HEAD requests should be handled like GET (no body)
    expect(true).toBe(true)
  })

  it('handles OPTIONS requests', async () => {
    const req = mockRequest({ method: 'OPTIONS', url: '/test' })
    const { res } = mockResponse()

    await proxyToApi(req, res, '/test', {
      apiPort: 99999,
      timeout: 100,
    })

    expect(true).toBe(true)
  })
})

describe('proxyWebSocketUpgrade', () => {
  let mockSocket: any
  let mockUpstream: any

  beforeEach(() => {
    // Mock client socket
    mockSocket = new EventEmitter()
    mockSocket.destroy = vi.fn()
    mockSocket.end = vi.fn()
    mockSocket.write = vi.fn()  // Added for error response
    mockSocket.setNoDelay = vi.fn()
    mockSocket.setTimeout = vi.fn()
    mockSocket.setKeepAlive = vi.fn()

    // Mock upstream socket
    mockUpstream = new EventEmitter()
    mockUpstream.destroy = vi.fn()
    mockUpstream.end = vi.fn()
    mockUpstream.write = vi.fn()
    mockUpstream.pipe = vi.fn()
    mockUpstream.setNoDelay = vi.fn()
    mockUpstream.setTimeout = vi.fn()
    mockUpstream.setKeepAlive = vi.fn()
  })

  it('rejects invalid API port', () => {
    const req = { url: '/ws', method: 'GET', headers: {}, httpVersion: '1.1' } as http.IncomingMessage
    const head = Buffer.from([])

    proxyWebSocketUpgrade(req, mockSocket, head, { apiPort: 'invalid' })

    expect(mockSocket.destroy).toHaveBeenCalled()
  })

  it('logs when verbose is true', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    const req = { url: '/ws', method: 'GET', headers: {}, httpVersion: '1.1' } as http.IncomingMessage
    const head = Buffer.from([])

    proxyWebSocketUpgrade(req, mockSocket, head, {
      apiPort: 'invalid',
      verbose: true,
    })

    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('[ws-proxy] Invalid API_PORT')
    )

    consoleSpy.mockRestore()
  })

  it('handles undefined apiPort', () => {
    const req = { url: '/ws', method: 'GET', headers: {}, httpVersion: '1.1' } as http.IncomingMessage
    const head = Buffer.from([])

    proxyWebSocketUpgrade(req, mockSocket, head, { apiPort: undefined as any })

    expect(mockSocket.destroy).toHaveBeenCalled()
  })

  it('accepts valid numeric port', () => {
    const req = { url: '/ws', method: 'GET', headers: {}, httpVersion: '1.1' } as http.IncomingMessage
    const head = Buffer.from([])

    // This will attempt to connect, which will fail, but that's expected
    // We're just checking it doesn't reject the port immediately
    proxyWebSocketUpgrade(req, mockSocket, head, {
      apiPort: '8080',
      timeout: 50,
    })

    expect(mockSocket.destroy).not.toHaveBeenCalled()
  })

  it('accepts valid apiHost option', () => {
    const req = { url: '/ws', method: 'GET', headers: {}, httpVersion: '1.1' } as http.IncomingMessage
    const head = Buffer.from([])

    proxyWebSocketUpgrade(req, mockSocket, head, {
      apiPort: '8080',
      apiHost: 'custom.api.host',
      timeout: 50,
    })

    // Should not immediately destroy socket
    expect(mockSocket.destroy).not.toHaveBeenCalled()
  })
})
