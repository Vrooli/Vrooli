/**
 * Tests for server/config.ts
 */

import { describe, it, expect, vi } from 'vitest'
import { buildScenarioConfig, createConfigEndpoint } from '../../server/config.js'
import type { ConfigEndpointOptions } from '../../shared/types.js'
import { mockRequest, mockResponse } from '../helpers/mock-request.js'

describe('buildScenarioConfig', () => {
  it('builds complete configuration', () => {
    const options: ConfigEndpointOptions = {
      apiPort: '8080',
      wsPort: '8081',
      uiPort: '3000',
      serviceName: 'test-scenario',
      version: '1.0.0',
    }

    const config = buildScenarioConfig(options)

    expect(config.apiUrl).toBe('http://127.0.0.1:8080/api/v1')
    expect(config.wsUrl).toBe('ws://127.0.0.1:8081/ws')
    expect(config.apiPort).toBe('8080')
    expect(config.wsPort).toBe('8081')
    expect(config.uiPort).toBe('3000')
    expect(config.service).toBe('test-scenario')
    expect(config.version).toBe('1.0.0')
  })

  it('uses custom host names', () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      apiHost: 'api.example.com',
      wsPort: 8081,
      wsHost: 'ws.example.com',
      uiPort: 3000,
    }

    const config = buildScenarioConfig(options)

    expect(config.apiUrl).toBe('http://api.example.com:8080/api/v1')
    expect(config.wsUrl).toBe('ws://ws.example.com:8081/ws')
  })

  it('defaults WS port to API port', () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      uiPort: 3000,
    }

    const config = buildScenarioConfig(options)

    expect(config.wsPort).toBe('8080')
    expect(config.wsUrl).toContain(':8080')
  })

  it('defaults WS host to API host', () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      apiHost: 'api.example.com',
      wsPort: 8081,
      uiPort: 3000,
    }

    const config = buildScenarioConfig(options)

    expect(config.wsUrl).toBe('ws://api.example.com:8081/ws')
  })

  it('includes additional config', () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      uiPort: 3000,
      additionalConfig: {
        customField: 'custom-value',
        anotherField: 42,
      },
    }

    const config = buildScenarioConfig(options)

    expect(config.customField).toBe('custom-value')
    expect(config.anotherField).toBe(42)
  })

  it('parses port from string', () => {
    const options: ConfigEndpointOptions = {
      apiPort: '8080',
      uiPort: '3000',
    }

    const config = buildScenarioConfig(options)

    expect(config.apiPort).toBe('8080')
    expect(config.apiUrl).toContain(':8080')
  })

  it('parses port from number', () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      uiPort: 3000,
    }

    const config = buildScenarioConfig(options)

    expect(config.apiPort).toBe('8080')
    expect(config.apiUrl).toContain(':8080')
  })

  it('throws on invalid API port', () => {
    const options: ConfigEndpointOptions = {
      apiPort: 'invalid',
      uiPort: 3000,
    }

    expect(() => buildScenarioConfig(options)).toThrow('Invalid API port')
  })

  it('throws on invalid UI port', () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      uiPort: 'invalid',
    }

    expect(() => buildScenarioConfig(options)).toThrow('Invalid UI port')
  })

  it('omits version if not provided', () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      uiPort: 3000,
    }

    const config = buildScenarioConfig(options)

    expect(config.version).toBeUndefined()
  })

  it('omits service if not provided', () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      uiPort: 3000,
    }

    const config = buildScenarioConfig(options)

    expect(config.service).toBeUndefined()
  })
})

describe('createConfigEndpoint', () => {
  it('creates middleware that returns config', async () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      wsPort: 8081,
      uiPort: 3000,
      serviceName: 'test-scenario',
      version: '1.0.0',
    }

    const middleware = createConfigEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.status).toBe(200)
    expect(result.json).toMatchObject({
      apiUrl: 'http://127.0.0.1:8080/api/v1',
      wsUrl: 'ws://127.0.0.1:8081/ws',
      apiPort: '8080',
      wsPort: '8081',
      uiPort: '3000',
      service: 'test-scenario',
      version: '1.0.0',
    })
  })

  it('uses environment variables when provided', async () => {
    const options: ConfigEndpointOptions = {
      apiPort: process.env.API_PORT || '8080',
      wsPort: process.env.WS_PORT || '8081',
      uiPort: process.env.UI_PORT || '3000',
    }

    const middleware = createConfigEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.status).toBe(200)
    expect(result.json).toBeDefined()
    expect(result.json.apiPort).toBeDefined()
  })

  it('handles CORS headers when enabled', async () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      uiPort: 3000,
      cors: true,
    }

    const middleware = createConfigEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.headers['access-control-allow-origin']).toBe('*')
  })

  it('does not set CORS headers when disabled', async () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      uiPort: 3000,
      cors: false,
    }

    const middleware = createConfigEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.headers['access-control-allow-origin']).toBeUndefined()
  })

  it('uses custom configBuilder if provided', async () => {
    const customBuilder = () => ({
      apiUrl: 'http://custom.example.com/api',
      wsUrl: 'ws://custom.example.com/ws',
      apiPort: '9999',
      wsPort: '9999',
      uiPort: '8888',
      customField: 'test',
    })

    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      uiPort: 3000,
      configBuilder: customBuilder,
    }

    const middleware = createConfigEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.json.apiUrl).toBe('http://custom.example.com/api')
    expect(result.json.customField).toBe('test')
  })

  it('handles errors in configBuilder', async () => {
    const customBuilder = () => {
      throw new Error('Config build failed')
    }

    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      uiPort: 3000,
      configBuilder: customBuilder,
    }

    const middleware = createConfigEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.status).toBe(500)
    expect(result.json.error).toBe('Failed to build configuration')
  })

  it('adds cache control headers when enabled', async () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      uiPort: 3000,
      cacheControl: true,
    }

    const middleware = createConfigEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.headers['cache-control']).toBeDefined()
    expect(result.headers['cache-control']).toContain('no-cache')
  })

  it('includes timestamp when enabled', async () => {
    const options: ConfigEndpointOptions = {
      apiPort: 8080,
      uiPort: 3000,
      includeTimestamp: true,
    }

    const middleware = createConfigEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    const before = Date.now()
    middleware(req, res, vi.fn())
    const after = Date.now()

    const result = getResult()
    expect(result.json.timestamp).toBeDefined()
    expect(result.json.timestamp).toBeGreaterThanOrEqual(before)
    expect(result.json.timestamp).toBeLessThanOrEqual(after)
  })
})
