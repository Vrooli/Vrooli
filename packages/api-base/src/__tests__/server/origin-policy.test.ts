/**
 * Browser origin policy: a page the operator happens to visit must not be able
 * to read or change scenario data through a scenario UI server, while the
 * scenario's own pages, declared origins and non-browser callers keep working.
 */

import { afterEach, describe, expect, it } from 'vitest'
import * as http from 'node:http'
import * as net from 'node:net'
import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'
import type { AddressInfo } from 'node:net'
import { createOriginPolicy, normalizeAllowedOrigins, type OriginPolicyRequest } from '../../server/origin-policy.js'
import { attachWsUpgradeHandler, createScenarioServer } from '../../server/template.js'
import type { ServerTemplateOptions } from '../../shared/types.js'

// ---------------------------------------------------------------------------
// Browser request vocabulary. Each helper is the header set a real browser
// sends for that situation, so cases read as the situation, not as headers.
// ---------------------------------------------------------------------------

type Headers = http.IncomingHttpHeaders

const SCENARIO = 'http://localhost:3000'
const ATTACKER = 'https://attacker.example'
const SIBLING = 'http://localhost:4000'

const browser = {
  sameOriginFetch: (origin = SCENARIO): Headers => ({ origin, 'sec-fetch-site': 'same-origin', 'sec-fetch-mode': 'cors', 'sec-fetch-dest': 'empty' }),
  typedAddress: (): Headers => ({ 'sec-fetch-site': 'none', 'sec-fetch-mode': 'navigate', 'sec-fetch-dest': 'document', 'sec-fetch-user': '?1' }),
  crossSiteFetch: (origin = ATTACKER): Headers => ({ origin, 'sec-fetch-site': 'cross-site', 'sec-fetch-mode': 'cors', 'sec-fetch-dest': 'empty' }),
  sameSiteFetch: (origin = SIBLING): Headers => ({ origin, 'sec-fetch-site': 'same-site', 'sec-fetch-mode': 'cors', 'sec-fetch-dest': 'empty' }),
  crossSiteNoCors: (): Headers => ({ 'sec-fetch-site': 'cross-site', 'sec-fetch-mode': 'no-cors', 'sec-fetch-dest': 'image' }),
  crossSiteFormPost: (origin = ATTACKER): Headers => ({ origin, 'sec-fetch-site': 'cross-site', 'sec-fetch-mode': 'navigate', 'sec-fetch-dest': 'document', 'content-type': 'application/x-www-form-urlencoded' }),
  crossSiteLink: (): Headers => ({ 'sec-fetch-site': 'cross-site', 'sec-fetch-mode': 'navigate', 'sec-fetch-dest': 'document', 'sec-fetch-user': '?1' }),
  crossSiteFrame: (): Headers => ({ 'sec-fetch-site': 'cross-site', 'sec-fetch-mode': 'navigate', 'sec-fetch-dest': 'iframe' }),
  crossSiteEmbed: (): Headers => ({ 'sec-fetch-site': 'cross-site', 'sec-fetch-mode': 'navigate', 'sec-fetch-dest': 'embed' }),
  opaqueSandboxScript: (): Headers => ({ origin: 'null', 'sec-fetch-site': 'cross-site', 'sec-fetch-mode': 'cors', 'sec-fetch-dest': 'script' }),
  preflight: (origin: string, method: string, requestHeaders = 'content-type'): Headers => ({
    origin, 'sec-fetch-site': 'cross-site', 'sec-fetch-mode': 'cors', 'access-control-request-method': method, 'access-control-request-headers': requestHeaders,
  }),
  /** Browsers without Fetch Metadata send only Origin on cross-origin and on non-GET requests. */
  legacy: (origin: string): Headers => ({ origin }),
}

function req(method: string, headers: Headers, requestPath = '/api/v1/items', host = 'localhost:3000'): OriginPolicyRequest {
  return { method, path: requestPath, headers: { host, ...headers } }
}

// ---------------------------------------------------------------------------
// Decision table
// ---------------------------------------------------------------------------

describe('createOriginPolicy().evaluate', () => {
  const policy = createOriginPolicy({
    allowedOrigins: ['chrome-extension://abcdefghijklmnop', 'https://Partner.Example:443'],
    publicPaths: ['/preview/runtime/', /^\/api\/v1\/sessions\/[^/]+\/runtime\//],
  })

  it.each([
    ['same-origin fetch', req('POST', browser.sameOriginFetch()), 'same-origin'],
    ['typed address or bookmark', req('GET', browser.typedAddress(), '/'), 'same-origin'],
    ['CLI or service (no browser context)', req('DELETE', {}), 'no-browser-context'],
    ['legacy browser, same-origin through the tunnel', req('POST', browser.legacy('https://notes.tunnel.example'), '/api/v1/items', 'notes.tunnel.example'), 'same-origin'],
    ['legacy browser, explicit default port in Host', req('POST', browser.legacy('https://notes.tunnel.example'), '/api/v1/items', 'notes.tunnel.example:443'), 'same-origin'],
    ['cross-site link to the app', req('GET', browser.crossSiteLink(), '/'), 'navigation'],
    ['cross-site frame navigation', req('GET', browser.crossSiteFrame(), '/'), 'navigation'],
    ['declared extension origin', req('POST', browser.crossSiteFetch('chrome-extension://abcdefghijklmnop')), 'allowed-origin'],
    ['declared origin, canonicalized', req('GET', browser.crossSiteFetch('https://partner.example')), 'allowed-origin'],
    ['public prefix read from an opaque sandbox', req('GET', browser.opaqueSandboxScript(), '/preview/runtime/react@18/index.js'), 'public-path'],
    ['public pattern read', req('HEAD', browser.crossSiteNoCors(), '/api/v1/sessions/s1/runtime/app.css'), 'public-path'],
    ['public preflight for a read', req('OPTIONS', browser.preflight('null', 'GET'), '/preview/runtime/x.js'), 'public-path'],
  ])('admits %s', (_label, request, verdict) => {
    expect(policy.evaluate(request).verdict).toBe(verdict)
  })

  it.each([
    ['cross-site fetch', req('GET', browser.crossSiteFetch())],
    ['same-site fetch from another localhost port', req('GET', browser.sameSiteFetch())],
    ['cross-site no-cors subresource (no Origin header)', req('GET', browser.crossSiteNoCors(), '/api/v1/secrets.js')],
    ['cross-site form POST (a simple request CORS never preflights)', req('POST', browser.crossSiteFormPost())],
    ['cross-site object/embed navigation', req('GET', browser.crossSiteEmbed(), '/')],
    ['opaque sandbox origin outside public paths', req('GET', browser.opaqueSandboxScript(), '/api/v1/items')],
    ['write to a public path', req('POST', browser.crossSiteFetch(), '/preview/runtime/x.js')],
    ['preflight for a write to a public path', req('OPTIONS', browser.preflight(ATTACKER, 'PUT'), '/preview/runtime/x.js')],
    ['preflight from an undeclared origin', req('OPTIONS', browser.preflight(ATTACKER, 'POST'))],
    ['legacy browser, cross-origin', req('POST', browser.legacy(ATTACKER))],
    ['legacy browser, sandboxed frame', req('POST', browser.legacy('null'))],
    ['look-alike of a declared origin', req('GET', browser.crossSiteFetch('https://partner.example.attacker.example'))],
  ])('refuses %s', (_label, request) => {
    const decision = policy.evaluate(request)
    expect(decision.verdict).toBe('refused')
    expect(decision.reason).toMatch(/cross-origin/)
  })

  it('applies to WebSocket handshakes without the navigation and public-path exemptions', () => {
    const upgrade = (headers: Headers, host = 'localhost:3000') => policy.evaluateUpgrade({ headers: { host, ...headers } }).verdict
    expect(upgrade({ origin: SCENARIO })).toBe('same-origin')
    expect(upgrade({ origin: SCENARIO, 'sec-fetch-site': 'same-origin', 'sec-fetch-mode': 'websocket' })).toBe('same-origin')
    expect(upgrade({})).toBe('no-browser-context')
    expect(upgrade({ origin: 'chrome-extension://abcdefghijklmnop' })).toBe('allowed-origin')
    expect(upgrade({ origin: ATTACKER })).toBe('refused')
    expect(upgrade({ origin: SIBLING, 'sec-fetch-site': 'same-site', 'sec-fetch-mode': 'websocket' })).toBe('refused')
    expect(upgrade({ origin: 'null' })).toBe('refused')
  })
})

describe('origin policy configuration', () => {
  it.each([
    ['the wildcard', ['*'], /wildcard/],
    ['a subdomain pattern', ['https://*.tunnel.example'], /wildcard/],
    ['the opaque origin', ['null'], /"null" is refused/],
    ['a URL with a path', ['https://partner.example/app'], /not an exact origin/],
    ['a comma-joined string', 'https://a.example,https://b.example', /must be an array/],
    ['a bare string', '*', /must be an array/],
    ['a non-string entry', [42], /must be strings/],
  ])('refuses %s in allowedOrigins', (_label, value, message) => {
    expect(() => normalizeAllowedOrigins(value)).toThrow(message)
  })

  it.each([
    ['the root path', ['/']],
    ['a wildcard path', ['/api/*']],
    ['a relative path', ['preview/']],
  ])('refuses %s in publicPaths', (_label, value) => {
    expect(() => createOriginPolicy({ publicPaths: value })).toThrow(/publicPaths/)
  })

  it('canonicalizes declared origins', () => {
    expect([...normalizeAllowedOrigins(['HTTPS://Partner.Example:443', 'http://127.0.0.1:8080', 'Chrome-Extension://ABC'])])
      .toEqual(['https://partner.example', 'http://127.0.0.1:8080', 'chrome-extension://abc'])
  })

  it('refuses the removed corsOrigins option at server construction, naming the replacement', () => {
    const legacy = { uiPort: 3000, apiPort: 8080, disableLifecycleGuard: true, corsOrigins: '*' } as unknown as ServerTemplateOptions
    expect(() => createScenarioServer(legacy)).toThrow(/corsOrigins option was removed.*allowedOrigins/)
  })

  it('refuses a wildcard allowedOrigins at server construction', () => {
    expect(() => createScenarioServer({ uiPort: 3000, apiPort: 8080, disableLifecycleGuard: true, allowedOrigins: ['*'] })).toThrow(/wildcard/)
  })
})

// ---------------------------------------------------------------------------
// Scenario server end to end: real sockets, a real upstream API, real relays.
// ---------------------------------------------------------------------------

interface Recorded { method: string; url: string; headers: http.IncomingHttpHeaders }

interface Response { status: number; headers: http.IncomingHttpHeaders; body: string }

async function listen(server: http.Server): Promise<number> {
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject)
    server.listen(0, '127.0.0.1', resolve)
  })
  return (server.address() as AddressInfo).port
}

function send(port: number, method: string, requestPath: string, headers: http.OutgoingHttpHeaders = {}, body?: string): Promise<Response> {
  return new Promise((resolve, reject) => {
    const request = http.request({ hostname: '127.0.0.1', port, method, path: requestPath, headers: { host: `localhost:${port}`, ...headers } }, (res) => {
      let text = ''
      res.setEncoding('utf8')
      res.on('data', (chunk) => { text += chunk })
      res.on('end', () => resolve({ status: res.statusCode ?? 0, headers: res.headers, body: text }))
    })
    request.on('error', reject)
    request.end(body)
  })
}

/** Raw handshake so the 403 status line is observable. */
function handshake(port: number, requestPath: string, headers: Record<string, string>): Promise<string> {
  return new Promise((resolve, reject) => {
    const socket = net.connect(port, '127.0.0.1', () => {
      const lines = [
        `GET ${requestPath} HTTP/1.1`,
        `Host: localhost:${port}`,
        'Connection: Upgrade',
        'Upgrade: websocket',
        'Sec-WebSocket-Version: 13',
        'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==',
        ...Object.entries(headers).map(([key, value]) => `${key}: ${value}`),
      ]
      socket.write(`${lines.join('\r\n')}\r\n\r\n`)
    })
    let received = ''
    socket.setEncoding('utf8')
    socket.on('data', (chunk) => {
      received += chunk
      if (received.includes('\r\n')) {
        socket.destroy()
        resolve(received.split('\r\n')[0]!)
      }
    })
    socket.on('error', reject)
    socket.on('close', () => resolve(received.split('\r\n')[0] ?? ''))
  })
}

describe('scenario server origin policy end to end', () => {
  const servers: http.Server[] = []
  const distDirs: string[] = []

  afterEach(async () => {
    for (const server of servers.splice(0)) {
      server.closeAllConnections()
      await new Promise<void>((resolve) => server.close(() => resolve()))
    }
    for (const dir of distDirs.splice(0)) fs.rmSync(dir, { recursive: true, force: true })
  })

  /** An API that records every request and answers with its own wildcard CORS, as many scenario APIs do. */
  async function api(): Promise<{ port: number; seen: Recorded[] }> {
    const seen: Recorded[] = []
    const server = http.createServer((request, response) => {
      seen.push({ method: request.method ?? '', url: request.url ?? '', headers: request.headers })
      request.resume()
      request.on('end', () => {
        response.setHeader('Access-Control-Allow-Origin', '*')
        response.setHeader('Vary', 'Accept-Encoding')
        response.setHeader('Content-Type', 'application/json')
        response.end(JSON.stringify({ ok: true }))
      })
    })
    server.on('upgrade', (request, socket) => {
      seen.push({ method: 'UPGRADE', url: request.url ?? '', headers: request.headers })
      socket.end('HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=\r\n\r\n')
    })
    const port = await listen(server)
    servers.push(server)
    return { port, seen }
  }

  async function scenario(apiPort: number, options: Partial<ServerTemplateOptions> = {}): Promise<number> {
    const distDir = fs.mkdtempSync(path.join(os.tmpdir(), 'origin-policy-dist-'))
    distDirs.push(distDir)
    fs.mkdirSync(path.join(distDir, 'assets'))
    fs.writeFileSync(path.join(distDir, 'index.html'), '<!doctype html><html><head></head><body>app</body></html>')
    fs.writeFileSync(path.join(distDir, 'assets', 'index-abc123.js'), 'export {}')
    const app = createScenarioServer({ uiPort: 3000, apiPort, distDir, disableLifecycleGuard: true, wsPathPrefix: '/ws', ...options })
    const server = http.createServer(app)
    attachWsUpgradeHandler(server, app)
    const port = await listen(server)
    servers.push(server)
    return port
  }

  it('serves the scenario\'s own pages: same-origin reads and writes reach the API', async () => {
    const upstream = await api()
    const ui = await scenario(upstream.port)
    const origin = `http://localhost:${ui}`

    const read = await send(ui, 'GET', '/api/v1/items', browser.sameOriginFetch(origin))
    const write = await send(ui, 'POST', '/api/v1/items', { ...browser.sameOriginFetch(origin), 'content-type': 'application/json' }, '{"name":"x"}')
    expect(read.status).toBe(200)
    expect(write.status).toBe(200)
    expect(upstream.seen.map((entry) => entry.method)).toEqual(['GET', 'POST'])
    // The scenario's own API still sees the browser context and may apply its own checks.
    expect(upstream.seen[1]!.headers['sec-fetch-site']).toBe('same-origin')
    // Same-origin responses carry no CORS grant, and the API's wildcard is not passed through.
    expect(read.headers['access-control-allow-origin']).toBeUndefined()
  })

  it('serves module scripts that send Origin on same-origin loads, in browsers with and without Fetch Metadata', async () => {
    const upstream = await api()
    const ui = await scenario(upstream.port)
    const origin = `http://localhost:${ui}`
    expect((await send(ui, 'GET', '/assets/index-abc123.js', { ...browser.sameOriginFetch(origin), 'sec-fetch-dest': 'script' })).status).toBe(200)
    expect((await send(ui, 'GET', '/assets/index-abc123.js', browser.legacy(origin))).status).toBe(200)
  })

  it('refuses a page the operator happens to visit, before the API is contacted', async () => {
    const upstream = await api()
    const ui = await scenario(upstream.port)

    const read = await send(ui, 'GET', '/api/v1/items', browser.crossSiteFetch())
    const write = await send(ui, 'POST', '/api/v1/items', browser.crossSiteFormPost(), 'name=x')
    const rpc = await send(ui, 'POST', '/vrooli.example.v1.ItemService/Delete', { ...browser.crossSiteFetch(), 'content-type': 'text/plain' }, '{}')
    const preflight = await send(ui, 'OPTIONS', '/api/v1/items', browser.preflight(ATTACKER, 'DELETE'))
    const sibling = await send(ui, 'GET', '/config', browser.sameSiteFetch())

    for (const response of [read, write, rpc, preflight, sibling]) {
      expect(response.status).toBe(403)
      expect(response.headers['access-control-allow-origin']).toBeUndefined()
      expect(response.headers['cache-control']).toBe('no-store')
    }
    expect(JSON.parse(read.body)).toMatchObject({ error: 'Cross-origin request refused' })
    expect(upstream.seen).toEqual([])
  })

  it('serves non-browser callers such as CLIs and services', async () => {
    const upstream = await api()
    const ui = await scenario(upstream.port)
    expect((await send(ui, 'DELETE', '/api/v1/items/1')).status).toBe(200)
    expect(upstream.seen).toHaveLength(1)
  })

  it('lets the operator follow a link to the app from another site', async () => {
    const upstream = await api()
    const ui = await scenario(upstream.port)
    const response = await send(ui, 'GET', '/dashboard', browser.crossSiteLink())
    expect(response.status).toBe(200)
    expect(response.body).toContain('app')
  })

  it('grants credentialed CORS only to a declared origin and replaces the API\'s wildcard with it', async () => {
    const upstream = await api()
    const extension = 'chrome-extension://abcdefghijklmnop'
    const ui = await scenario(upstream.port, { allowedOrigins: [extension] })

    const preflight = await send(ui, 'OPTIONS', '/api/v1/items', browser.preflight(extension, 'POST', 'content-type, x-custom'))
    expect(preflight.status).toBe(204)
    expect(preflight.headers['access-control-allow-origin']).toBe(extension)
    expect(preflight.headers['access-control-allow-credentials']).toBe('true')
    expect(preflight.headers['access-control-allow-headers']).toBe('content-type, x-custom')
    expect(upstream.seen).toEqual([])

    const call = await send(ui, 'POST', '/api/v1/items', { ...browser.crossSiteFetch(extension), 'content-type': 'application/json' }, '{}')
    expect(call.status).toBe(200)
    expect(call.headers['access-control-allow-origin']).toBe(extension)
    expect(call.headers['access-control-allow-credentials']).toBe('true')
    expect(String(call.headers.vary)).toMatch(/Origin/)
    expect(String(call.headers.vary)).toMatch(/Accept-Encoding/)

    expect((await send(ui, 'POST', '/api/v1/items', browser.crossSiteFetch())).status).toBe(403)
  })

  it('lets opaque sandbox frames read declared public paths without credentials, and nothing else', async () => {
    const upstream = await api()
    const ui = await scenario(upstream.port, {
      publicPaths: ['/api/v1/preview/runtime/'],
    })
    const runtime = await send(ui, 'GET', '/api/v1/preview/runtime/react@18/index.js', browser.opaqueSandboxScript())
    expect(runtime.status).toBe(200)
    expect(runtime.headers['access-control-allow-origin']).toBe('*')
    expect(runtime.headers['access-control-allow-credentials']).toBeUndefined()
    expect((await send(ui, 'POST', '/api/v1/preview/runtime/x', browser.opaqueSandboxScript())).status).toBe(403)
    expect((await send(ui, 'GET', '/api/v1/items', browser.opaqueSandboxScript())).status).toBe(403)
  })

  it('refuses cross-origin WebSocket handshakes and relays same-origin ones', async () => {
    const upstream = await api()
    const ui = await scenario(upstream.port)
    expect(await handshake(ui, '/ws/events', { Origin: ATTACKER })).toBe('HTTP/1.1 403 Forbidden')
    expect(await handshake(ui, '/ws/events', { Origin: 'null' })).toBe('HTTP/1.1 403 Forbidden')
    expect(upstream.seen).toEqual([])
    expect(await handshake(ui, '/ws/events', { Origin: `http://localhost:${ui}` })).toBe('HTTP/1.1 101 Switching Protocols')
    expect(await handshake(ui, '/ws/events', {})).toBe('HTTP/1.1 101 Switching Protocols')
    expect(upstream.seen.map((entry) => entry.method)).toEqual(['UPGRADE', 'UPGRADE'])
  })

  it('reaches a scenario through Web Console\'s /_local relay, which drops the sandbox frame\'s browser context', async () => {
    const targetApi = await api()
    const target = await scenario(targetApi.port)
    const consoleApi = await api()
    const consoleUi = await scenario(consoleApi.port, { localPortProxy: true })

    // The operator opens the app from Web Console: the relay issues a capability.
    const bootstrap = await send(consoleUi, 'GET', `/_local/${target}/`, { accept: 'text/html', 'sec-fetch-site': 'same-origin', 'sec-fetch-mode': 'navigate', 'sec-fetch-dest': 'document' })
    expect(bootstrap.status).toBe(302)
    const base = bootstrap.headers.location!.replace(/\/$/, '')

    // The app runs in an opaque sandbox and writes to its own API through the relay.
    const write = await send(consoleUi, 'POST', `${base}/api/v1/items`, {
      origin: 'null', 'sec-fetch-site': 'cross-site', 'sec-fetch-mode': 'cors', 'sec-fetch-dest': 'empty', referer: `http://localhost:${consoleUi}${base}/`, 'content-type': 'application/json',
    }, '{}')
    expect(write.status).toBe(200)
    const forwarded = targetApi.seen.at(-1)!.headers
    for (const header of ['origin', 'referer', 'sec-fetch-site', 'sec-fetch-mode', 'sec-fetch-dest']) {
      expect(forwarded[header]).toBeUndefined()
    }

    // Without the capability, the same request is refused by the relay itself.
    expect((await send(consoleUi, 'POST', `/_local/${target}/api/v1/items`, browser.crossSiteFetch())).status).toBe(403)
  })
})
