import { describe, it, expect, beforeAll, afterAll } from 'vitest'
import type { Server } from 'node:http'
import {
  setupChildScenario,
  setupHostScenario,
  makeRequest,
} from './setup.js'
import type { HostEndpointDefinition } from '../../../server/index.js'

interface HarnessContext {
  childUiServer: Server
  childApiServer: Server
  hostUiServer: Server
  hostApiServer: Server
  hostUiPort: number
  childId: string
}

describe('Host scenario host-endpoint metadata', () => {
  const childId = 'test-child'
  const hostEndpoints: HostEndpointDefinition[] = [
    { path: '/api/v1/summary', method: 'GET' },
    { path: '/api/v1/apps/{id}/diagnostics', method: 'GET' },
  ]

  let ctx: HarnessContext

  beforeAll(async () => {
    const childUiPort = 46000
    const childApiPort = 46050
    const hostUiPort = 46100
    const hostApiPort = 46150

    const child = await setupChildScenario(childId, childUiPort, childApiPort)
    const host = await setupHostScenario(
      childId,
      childUiPort,
      childApiPort,
      hostUiPort,
      hostApiPort,
      { hostEndpoints }
    )

    ctx = {
      childUiServer: child.uiServer,
      childApiServer: child.apiServer,
      hostUiServer: host.uiServer,
      hostApiServer: host.apiServer,
      hostUiPort,
      childId,
    }
  }, 60000)

  afterAll(async () => {
    await new Promise<void>((resolve) => ctx.childUiServer.close(() => resolve()))
    await new Promise<void>((resolve) => ctx.childApiServer.close(() => resolve()))
    await new Promise<void>((resolve) => ctx.hostUiServer.close(() => resolve()))
    await new Promise<void>((resolve) => ctx.hostApiServer.close(() => resolve()))
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
