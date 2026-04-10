/**
 * Tests for self-proxy prevention in createScenarioProxyHost
 *
 * DOC: scenarios/app-monitor/docs/internal/SEAMS.md#recursive-self-embedding-prevention
 */

import { describe, it, expect, vi, afterEach } from 'vitest'
import { createScenarioProxyHost } from '../../server/host.js'
import type { ScenarioProxyAppMetadata } from '../../shared/types.js'

function buildHost(overrides: {
  hostScenario?: string
  fetchAppMetadata?: (appId: string) => Promise<ScenarioProxyAppMetadata | null>
  hostHtmlFingerprint?: string
} = {}) {
  const fetchAppMetadata = overrides.fetchAppMetadata ?? (async (appId: string) => ({
    id: appId,
    name: appId,
    scenario_name: appId,
    port_mappings: { ui: 4000, api: 4001 },
  }))

  return createScenarioProxyHost({
    hostScenario: overrides.hostScenario ?? 'app-monitor',
    fetchAppMetadata,
    hostHtmlFingerprint: overrides.hostHtmlFingerprint,
    verbose: false,
    enableServerTiming: false,
    enableMetrics: false,
  })
}

function fakeReq(params: Record<string, string>, url?: string): any {
  const reqUrl = url ?? '/'
  return {
    method: 'GET',
    url: reqUrl,
    originalUrl: reqUrl,
    path: reqUrl.split('?')[0],
    params,
    headers: { accept: 'text/html' },
    get: (name: string) => undefined,
  }
}

function fakeRes(): { res: any; status: () => number; json: () => any } {
  let statusCode = 200
  let jsonData: any = null
  const headers: Record<string, any> = {}
  const res: any = {
    headersSent: false,
    status(code: number) { statusCode = code; return this },
    json(data: any) { jsonData = data; return this },
    send(data: any) { return this },
    write(chunk: any) { return true },
    end() { return this },
    setHeader(name: string, value: any) { headers[name] = value; return this },
    getHeader(name: string) { return headers[name] },
    removeHeader(name: string) { delete headers[name]; return this },
    destroy() {},
  }
  return { res, status: () => statusCode, json: () => jsonData }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('self-proxy blocking in handleScenarioProxyRequest', () => {
  it('blocks appId === hostScenario with 403', async () => {
    const host = buildHost()
    const req = fakeReq({ appId: 'app-monitor' }, '/apps/app-monitor/proxy/')
    const { res, status, json } = fakeRes()

    // Use the router to handle the request
    await new Promise<void>((resolve) => {
      host.router.handle(
        { ...req, originalUrl: '/apps/app-monitor/proxy/', url: '/apps/app-monitor/proxy/' },
        res,
        () => resolve(),
      )
      // Give async handler time to execute
      setTimeout(resolve, 100)
    })

    expect(status()).toBe(403)
    expect(json()).toMatchObject({ code: 'SELF_PROXY_BLOCKED' })

    host.destroy()
  })

  it('blocks case-insensitive match', async () => {
    const host = buildHost()
    const req = fakeReq({ appId: 'App-Monitor' }, '/apps/App-Monitor/proxy/')
    const { res, status, json } = fakeRes()

    await new Promise<void>((resolve) => {
      host.router.handle(
        { ...req, originalUrl: '/apps/App-Monitor/proxy/', url: '/apps/App-Monitor/proxy/' },
        res,
        () => resolve(),
      )
      setTimeout(resolve, 100)
    })

    expect(status()).toBe(403)
    expect(json()).toMatchObject({ code: 'SELF_PROXY_BLOCKED' })

    host.destroy()
  })

  it('allows non-host appId through', async () => {
    const host = buildHost()
    const { res, status } = fakeRes()

    await new Promise<void>((resolve) => {
      host.router.handle(
        {
          method: 'GET',
          url: '/apps/other-scenario/proxy/',
          originalUrl: '/apps/other-scenario/proxy/',
          path: '/apps/other-scenario/proxy/',
          params: { appId: 'other-scenario' },
          headers: { accept: 'text/html' },
          get: () => undefined,
        },
        res,
        () => resolve(),
      )
      setTimeout(resolve, 200)
    })

    // Should not be 403 — it will either proxy or fail with a connection error
    expect(status()).not.toBe(403)

    host.destroy()
  })
})

describe('indirect self-proxy via metadata', () => {
  it('blocks appId that resolves to hostScenario via scenario_name', async () => {
    const host = buildHost({
      fetchAppMetadata: async (appId: string) => ({
        id: appId,
        scenario_name: 'app-monitor',
        port_mappings: { ui: 4000 },
      }),
    })
    const { res, status, json } = fakeRes()

    await new Promise<void>((resolve) => {
      host.router.handle(
        {
          method: 'GET',
          url: '/apps/sneaky-alias/proxy/',
          originalUrl: '/apps/sneaky-alias/proxy/',
          path: '/apps/sneaky-alias/proxy/',
          params: { appId: 'sneaky-alias' },
          headers: { accept: 'text/html' },
          get: () => undefined,
        },
        res,
        () => resolve(),
      )
      setTimeout(resolve, 200)
    })

    // The getProxyContext throws, which results in a 502
    expect(status()).toBe(502)
    expect(json()?.details).toContain('Self-proxy blocked')

    host.destroy()
  })
})

describe('self-proxy in handlePortProxyRequest', () => {
  it('blocks port proxy for hostScenario with 403', async () => {
    const host = buildHost()
    const { res, status, json } = fakeRes()

    await new Promise<void>((resolve) => {
      host.router.handle(
        {
          method: 'GET',
          url: '/apps/app-monitor/ports/ui/proxy/',
          originalUrl: '/apps/app-monitor/ports/ui/proxy/',
          path: '/apps/app-monitor/ports/ui/proxy/',
          params: { appId: 'app-monitor', portKey: 'ui' },
          headers: {},
          get: () => undefined,
        },
        res,
        () => resolve(),
      )
      setTimeout(resolve, 100)
    })

    expect(status()).toBe(403)
    expect(json()).toMatchObject({ code: 'SELF_PROXY_BLOCKED' })

    host.destroy()
  })
})

describe('WebSocket upgrade self-proxy', () => {
  it('destroys socket for self-proxy upgrade', async () => {
    const host = buildHost()
    const socket = {
      destroyed: false,
      destroy() { this.destroyed = true },
      on() { return this },
      write() { return true },
      end() {},
    }

    const handled = await host.handleUpgrade(
      {
        url: '/apps/app-monitor/proxy/ws',
        headers: { host: 'localhost:3000' },
      },
      socket,
      Buffer.alloc(0),
    )

    expect(handled).toBe(true)
    expect(socket.destroyed).toBe(true)

    host.destroy()
  })

  it('destroys socket for port-specific self-proxy upgrade', async () => {
    const host = buildHost()
    const socket = {
      destroyed: false,
      destroy() { this.destroyed = true },
      on() { return this },
      write() { return true },
      end() {},
    }

    const handled = await host.handleUpgrade(
      {
        url: '/apps/app-monitor/ports/ws/proxy/ws',
        headers: { host: 'localhost:3000' },
      },
      socket,
      Buffer.alloc(0),
    )

    expect(handled).toBe(true)
    expect(socket.destroyed).toBe(true)

    host.destroy()
  })
})
