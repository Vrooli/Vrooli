import { afterEach, describe, expect, it } from 'vitest'
import express from 'express'
import * as http from 'node:http'
import { ProxyMetadataUnavailableError, createScenarioProxyHost } from '../../server/host.js'

const cleanup: Array<() => Promise<void> | void> = []
afterEach(async () => { for (const close of cleanup.splice(0).reverse()) await close() })

async function listen(server: http.Server): Promise<number> {
  await new Promise<void>(resolve => server.listen(0, '127.0.0.1', resolve))
  cleanup.push(() => !server.listening ? undefined : new Promise<void>((resolve, reject) => server.close(error => error ? reject(error) : resolve())))
  const address = server.address()
  if (!address || typeof address === 'string') throw new Error('Test listener unavailable')
  return address.port
}

async function request(port: number): Promise<{ status: number; body: string }> {
  return await new Promise((resolve, reject) => {
    http.get({ hostname: '127.0.0.1', port, path: '/apps/notes/proxy/', headers: { accept: 'text/html' } }, response => {
      const chunks: Buffer[] = []
      response.on('data', chunk => chunks.push(Buffer.from(chunk)))
      response.on('end', () => resolve({ status: response.statusCode ?? 0, body: Buffer.concat(chunks).toString('utf8') }))
      response.on('error', reject)
    }).on('error', reject)
  })
}

async function writeChildNote(port: number, method: 'GET' | 'POST', note?: string): Promise<{ status: number; body: string }> {
  return await new Promise((resolve, reject) => {
    const request = http.request({
      hostname: '127.0.0.1',
      port,
      path: '/apps/notes/proxy/api/state',
      method,
      headers: { accept: 'application/json', ...(note === undefined ? {} : { 'content-type': 'application/json' }) },
    }, response => {
      const chunks: Buffer[] = []
      response.on('data', chunk => chunks.push(Buffer.from(chunk)))
      response.on('end', () => resolve({ status: response.statusCode ?? 0, body: Buffer.concat(chunks).toString('utf8') }))
      response.on('error', reject)
    })
    request.on('error', reject)
    if (note !== undefined) request.write(JSON.stringify({ note }))
    request.end()
  })
}

const html = '<!doctype html><html><head></head><body><main>Saved work stays open</main></body></html>'

describe('bounded proxy metadata recovery', () => {
  it('keeps a healthy app reachable during a short metadata outage, then expires the stale route', async () => {
    let savedNote = 'draft created before the outage'
    const upstream = http.createServer(async (req, res) => {
      if (req.url === '/api/state') {
        if (req.method === 'POST') {
          const chunks: Buffer[] = []
          for await (const chunk of req) chunks.push(Buffer.from(chunk))
          savedNote = (JSON.parse(Buffer.concat(chunks).toString('utf8')) as { note: string }).note
        }
        res.writeHead(200, { 'content-type': 'application/json' })
        res.end(JSON.stringify({ note: savedNote }))
        return
      }
      res.writeHead(200, { 'content-type': 'text/html' })
      res.end(html)
    })
    const upstreamPort = await listen(upstream)
    let metadataAvailable = true
    let metadataCalls = 0
    const proxy = createScenarioProxyHost({
      hostScenario: 'portal',
      cacheTtlMs: 0,
      metadataStaleIfErrorMs: 60,
      metadataFailureRetryIntervalMs: 20,
      healthCheckIntervalMs: 5,
      healthCheckTimeoutMs: 20,
      enableServerTiming: false,
      fetchAppMetadata: async appId => {
        metadataCalls++
        if (!metadataAvailable) throw new ProxyMetadataUnavailableError()
        return { id: appId, scenario_name: appId, port_mappings: { UI_PORT: upstreamPort, API_PORT: upstreamPort } }
      },
    })
    cleanup.push(() => proxy.destroy())
    const app = express()
    app.use(proxy.router)
    const hostPort = await listen(http.createServer(app))

    const first = await request(hostPort)
    expect(first.status).toBe(200)
    expect(first.body).toContain('Saved work stays open')
    expect(JSON.parse((await writeChildNote(hostPort, 'GET')).body)).toEqual({ note: 'draft created before the outage' })
    metadataAvailable = false

    const duringLoss = await writeChildNote(hostPort, 'POST', 'edited while catalog is unavailable')
    expect(duringLoss.status).toBe(200)
    expect(JSON.parse(duringLoss.body)).toEqual({ note: 'edited while catalog is unavailable' })
    expect(metadataCalls).toBe(3)

    const repeatedDuringLoss = await writeChildNote(hostPort, 'GET')
    expect(repeatedDuringLoss.status).toBe(200)
    expect(JSON.parse(repeatedDuringLoss.body)).toEqual({ note: 'edited while catalog is unavailable' })
    expect(metadataCalls).toBe(3)

    await new Promise(resolve => setTimeout(resolve, 75))
    const afterExpiry = await request(hostPort)
    expect(afterExpiry.status).toBe(502)
    expect(savedNote).toBe('edited while catalog is unavailable')
    expect(metadataCalls).toBe(4)
  })

  it('moves to a changed child port without serving cached old UI or losing shared app data', async () => {
    const stored = new Map<string, string>([['note', 'draft before restart']])
    let oldCalls = 0
    let newCalls = 0
    const createChild = (heading: string, onRequest: () => void) => http.createServer(async (req, res) => {
      onRequest()
      if (req.url === '/api/state') {
        if (req.method === 'POST') {
          const chunks: Buffer[] = []
          for await (const chunk of req) chunks.push(Buffer.from(chunk))
          stored.set('note', (JSON.parse(Buffer.concat(chunks).toString('utf8')) as { note: string }).note)
        }
        res.writeHead(200, { 'content-type': 'application/json' })
        res.end(JSON.stringify({ note: stored.get('note') }))
        return
      }
      res.writeHead(200, { 'content-type': 'text/html' })
      res.end(`<!doctype html><html><head></head><body><h1>${heading}</h1></body></html>`)
    })
    const oldChild = createChild('Old child', () => { oldCalls++ })
    const oldPort = await listen(oldChild)
    let currentPort = oldPort
    let metadataCalls = 0
    const proxy = createScenarioProxyHost({
      hostScenario: 'portal',
      cacheTtlMs: 50,
      proxyHtmlCacheTtlMs: 5000,
      metadataStaleIfErrorMs: 1000,
      healthCheckIntervalMs: 5,
      healthCheckTimeoutMs: 20,
      enableServerTiming: false,
      fetchAppMetadata: async appId => {
        metadataCalls++
        return { id: appId, scenario_name: appId, port_mappings: { UI_PORT: currentPort, API_PORT: currentPort } }
      },
    })
    cleanup.push(() => proxy.destroy())
    const app = express()
    app.use(proxy.router)
    const hostPort = await listen(http.createServer(app))

    expect((await request(hostPort)).body).toContain('Old child')
    const draft = await writeChildNote(hostPort, 'POST', 'edit saved before restart')
    expect(JSON.parse(draft.body)).toEqual({ note: 'edit saved before restart' })
    expect(oldCalls).toBe(2)

    const newChild = createChild('New child', () => { newCalls++ })
    currentPort = await listen(newChild)
    await new Promise(resolve => setTimeout(resolve, 60))
    const updatedPage = await request(hostPort)
    expect(updatedPage.status).toBe(200)
    expect(updatedPage.body).toContain('New child')
    expect(updatedPage.body).not.toContain('Old child')
    expect(oldCalls).toBe(2)
    expect(newCalls).toBe(1)
    expect(JSON.parse((await writeChildNote(hostPort, 'GET')).body)).toEqual({ note: 'edit saved before restart' })
    expect(metadataCalls).toBe(2)
    expect(oldCalls).toBe(2)
    expect(newCalls).toBe(2)
  })

  it('does not use stale metadata for an app that is explicitly missing', async () => {
    const upstream = http.createServer((_req, res) => {
      res.writeHead(200, { 'content-type': 'text/html' })
      res.end(html)
    })
    const upstreamPort = await listen(upstream)
    let missing = false
    const proxy = createScenarioProxyHost({
      hostScenario: 'portal',
      cacheTtlMs: 0,
      metadataStaleIfErrorMs: 1000,
      enableServerTiming: false,
      fetchAppMetadata: async appId => {
        if (missing) throw new Error('Application not found')
        return { id: appId, scenario_name: appId, port_mappings: { UI_PORT: upstreamPort } }
      },
    })
    cleanup.push(() => proxy.destroy())
    const app = express()
    app.use(proxy.router)
    const hostPort = await listen(http.createServer(app))

    expect((await request(hostPort)).status).toBe(200)
    missing = true
    const response = await request(hostPort)
    expect(response.status).toBe(502)
    expect(response.body).not.toContain('Saved work stays open')
  })

  it('does not use stale metadata when the cached app itself has gone offline', async () => {
    const upstream = http.createServer((_req, res) => {
      res.writeHead(200, { 'content-type': 'text/html' })
      res.end(html)
    })
    const upstreamPort = await listen(upstream)
    let metadataAvailable = true
    const proxy = createScenarioProxyHost({
      hostScenario: 'portal',
      cacheTtlMs: 0,
      metadataStaleIfErrorMs: 1000,
      healthCheckIntervalMs: 5,
      healthCheckTimeoutMs: 20,
      enableServerTiming: false,
      fetchAppMetadata: async appId => {
        if (!metadataAvailable) throw new ProxyMetadataUnavailableError()
        return { id: appId, scenario_name: appId, port_mappings: { UI_PORT: upstreamPort } }
      },
    })
    cleanup.push(() => proxy.destroy())
    const app = express()
    app.use(proxy.router)
    const hostPort = await listen(http.createServer(app))

    expect((await request(hostPort)).status).toBe(200)
    await new Promise<void>((resolve, reject) => upstream.close(error => error ? reject(error) : resolve()))
    await new Promise(resolve => setTimeout(resolve, 25))
    metadataAvailable = false
    const response = await request(hostPort)
    expect(response.status).toBe(502)
    expect(response.body).not.toContain('Saved work stays open')
  })

  it('fails closed when the metadata owner is unavailable before any app route is cached', async () => {
    const proxy = createScenarioProxyHost({
      hostScenario: 'portal',
      metadataStaleIfErrorMs: 1000,
      enableServerTiming: false,
      fetchAppMetadata: async () => { throw new ProxyMetadataUnavailableError() },
    })
    cleanup.push(() => proxy.destroy())
    const app = express()
    app.use(proxy.router)
    const hostPort = await listen(http.createServer(app))

    const response = await request(hostPort)
    expect(response.status).toBe(502)
    expect(response.body).not.toContain('Saved work stays open')
  })
})
