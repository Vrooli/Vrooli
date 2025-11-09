import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import { setupTestEnvironment, cleanupTestEnvironment, makeRequest } from './setup.js'
import type { TestContext } from './setup.js'
import type { HostEndpointDefinition } from '../../../server/index.js'

describe('Host scenario host-endpoint metadata', () => {
  const childId = 'test-child'
  const hostEndpoints: HostEndpointDefinition[] = [
    { path: '/api/v1/summary', method: 'GET' },
    { path: '/api/v1/apps/{id}/diagnostics', method: 'GET' },
  ]

  let ctx: TestContext

  beforeAll(async () => {
    ctx = await setupTestEnvironment(5000, { hostEndpoints })
  }, 60000)

  afterAll(async () => {
    await cleanupTestEnvironment(ctx)
  })

  it('injects host endpoint metadata into proxied HTML', async () => {
    const htmlResponse = await makeRequest(ctx.hostUiPort, `/apps/${ctx.childId}/proxy/index.html`)

    expect(htmlResponse.status).toBe(200)
    expect(htmlResponse.body).toContain('id="vrooli-proxy-metadata"')
    expect(htmlResponse.body).toContain('"hostEndpoints"')
    expect(htmlResponse.body).toContain('/api/v1/summary')
    expect(htmlResponse.body).toContain('/api/v1/apps/{id}/diagnostics')
  })
})
