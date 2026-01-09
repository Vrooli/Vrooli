/**
 * Tests for server/health.ts
 */

import { describe, it, expect, vi } from 'vitest'
import { createHealthEndpoint, createSimpleHealthEndpoint } from '../../server/health.js'
import type { HealthOptions } from '../../shared/types.js'
import { mockRequest, mockResponse } from '../helpers/mock-request.js'

describe('createHealthEndpoint', () => {
  it('returns healthy status without API check', async () => {
    const options: HealthOptions = {
      serviceName: 'test-scenario',
      version: '1.0.0',
    }

    const middleware = createHealthEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    await middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.status).toBe(200)
    expect(result.json).toMatchObject({
      status: 'healthy',
      service: 'test-scenario',
      version: '1.0.0',
      readiness: true,
    })
    expect(result.json.timestamp).toBeDefined()
  })

  it('includes API connectivity check when apiPort configured', async () => {
    const options: HealthOptions = {
      serviceName: 'test-scenario',
      apiPort: 99999, // Will fail but we can test structure
      apiHost: 'localhost',
      timeout: 1000,
    }

    const middleware = createHealthEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    await middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.json.api_connectivity).toBeDefined()
    expect(result.json.api_connectivity).toHaveProperty('connected')
    expect(result.json.api_connectivity.connected).toBe(false) // Since no server running
    expect(result.json.api_connectivity.error).toBeDefined()
  })

  it('omits API check when apiPort not provided', async () => {
    const options: HealthOptions = {
      serviceName: 'test-scenario',
    }

    const middleware = createHealthEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    await middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.json.api_connectivity).toBeUndefined()
  })

  it('runs custom health check function', async () => {
    const customHealthCheck = async () => ({
      database: { healthy: true },
      redis: { healthy: false },
    })

    const options: HealthOptions = {
      serviceName: 'test-scenario',
      customHealthCheck,
    }

    const middleware = createHealthEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    await middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.json.database).toMatchObject({ healthy: true })
    expect(result.json.redis).toMatchObject({ healthy: false })
  })

  it('handles errors in custom health check', async () => {
    const customHealthCheck = async () => {
      throw new Error('Health check failed')
    }

    const options: HealthOptions = {
      serviceName: 'test-scenario',
      customHealthCheck,
    }

    const middleware = createHealthEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    await middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.json.custom_health_error).toBeDefined()
  })

  it('returns degraded status when API check fails', async () => {
    const options: HealthOptions = {
      serviceName: 'test-scenario',
      apiPort: 99999,
      timeout: 1000,
    }

    const middleware = createHealthEndpoint(options)
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    await middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.status).toBe(503)
    expect(result.json.status).toBe('degraded')
  })
})

describe('createSimpleHealthEndpoint', () => {
  it('returns minimal healthy status', () => {
    const middleware = createSimpleHealthEndpoint('test-service', '1.0.0')
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.status).toBe(200)
    expect(result.json).toMatchObject({
      status: 'healthy',
      service: 'test-service',
      version: '1.0.0',
      readiness: true,
    })
    expect(result.json.timestamp).toBeDefined()
  })

  it('works without version', () => {
    const middleware = createSimpleHealthEndpoint('test-service')
    const req = mockRequest()
    const { res, getResult } = mockResponse()

    middleware(req, res, vi.fn())

    const result = getResult()
    expect(result.json.version).toBeUndefined()
    expect(result.json.service).toBe('test-service')
  })
})
