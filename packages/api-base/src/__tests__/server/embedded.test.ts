/**
 * Tests for server/embedded.ts
 *
 * Covers:
 * - createPortResolver: peer-record lookup, CLI fallback, TTL caching, negative lookups
 * - resolveExternalUrl: localhost, tunnel/remote, fallback
 * - createEmbeddedProxyRouter: external-url endpoint, proxy passthrough, error paths,
 *   allowedScenarios gate, timeout, proxy error handling
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import * as http from 'node:http'
import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'
import { gzipSync } from 'node:zlib'
import express from 'express'
import type { Server } from 'node:http'
import {
  createPortResolver,
  createPortResolutionResolver,
  resolveScenarioPort,
  classifyPortLookupFailureCode,
  parseScenarioPortCLIResponse,
  resolveExternalUrl,
  createEmbeddedProxyRouter,
} from '../../server/embedded.js'
import { mockRequest, mockResponse } from '../helpers/mock-request.js'

const previousRuntimeHome = process.env.VROOLI_HOME
let testRuntimeHome = ''

beforeEach(() => {
  testRuntimeHome = fs.mkdtempSync(path.join(os.tmpdir(), 'api-base-embedded-test-'))
  process.env.VROOLI_HOME = testRuntimeHome
  fs.mkdirSync(path.join(testRuntimeHome, 'peers'), { recursive: true })
})

afterEach(() => {
  fs.rmSync(testRuntimeHome, { recursive: true, force: true })
  if (previousRuntimeHome === undefined) delete process.env.VROOLI_HOME
  else process.env.VROOLI_HOME = previousRuntimeHome
})

// ──────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────

function publishPeerPort(scenarioName: string, port: string | number, portKey = 'ui'): void {
  const [scenario, variant] = scenarioName.split('@')
  const instance = variant ?? 'live'
  const slug = instance === 'live' ? scenario! : `${scenario}@${instance}`
  const peerDir = path.join(process.env.VROOLI_HOME!, 'peers')
  fs.mkdirSync(peerDir, { recursive: true })
  fs.writeFileSync(path.join(peerDir, `${slug}.json`), JSON.stringify({
    schema_version: 1,
    scenario,
    instance,
    owner_pid: process.pid,
    ports: { [portKey]: Number(port) },
    provides: [portKey],
  }))
}

/** Start an Express app with the embedded router mounted at /embedded and return server + port */
async function startEmbeddedServer(
  options: Parameters<typeof createEmbeddedProxyRouter>[0] = {},
  bodyParser?: express.RequestHandler,
): Promise<{ server: Server; port: number }> {
  const app = express()
  if (bodyParser) app.use(bodyParser)
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
    if (req.url === '/agent_manager.v1.AgentManagerService/GetEffortBoard') {
      const chunks: Buffer[] = []
      req.on('data', (chunk: Buffer) => chunks.push(chunk))
      req.on('end', () => {
        const body = Buffer.concat(chunks)
        res.writeHead(200, { 'Content-Type': 'application/json' })
        res.end(JSON.stringify({
          method: req.method,
          body: body.toString('base64'),
          byteLength: body.length,
          headers: req.headers,
        }))
      })
      return
    }
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
  options: {
    method?: string
    headers?: Record<string, string>
    body?: string | Buffer
    writeBody?: (req: http.ClientRequest) => void
  } = {},
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
    if (options.writeBody) options.writeBody(req)
    else req.end(options.body)
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

  it('matches the shared Go/Node peer-record conformance corpus', () => {
    const corpusPath = path.resolve(process.cwd(), '../api-core/discovery/testdata/peer-record-conformance.json')
    const fixtures = JSON.parse(fs.readFileSync(corpusPath, 'utf8')) as Array<{
      name: string; target: string; port_key: string; owner: 'alive' | 'dead' | 'eperm' | 'invalid'
      record: Record<string, unknown>; want?: string
    }>
    for (const fixture of fixtures) {
      const slug = fixture.target.replace('@live', '')
      const recordPath = path.join(testRuntimeHome, 'peers', `${slug}.json`)
      const record = { ...fixture.record }
      if (record.owner_pid === '$PID') record.owner_pid = process.pid
      fs.writeFileSync(recordPath, JSON.stringify(record))
      const kill = vi.spyOn(process, 'kill').mockImplementation((() => {
        if (fixture.owner === 'eperm') throw Object.assign(new Error('permission denied'), { code: 'EPERM' })
        if (fixture.owner === 'dead') throw Object.assign(new Error('no such process'), { code: 'ESRCH' })
        return true
      }) as typeof process.kill)
      try {
        const actual = createPortResolutionResolver(30_000, fixture.port_key, {
          runtimeHome: () => testRuntimeHome,
          cliLookup: () => ({ port: null, reason: 'not-found' }),
        })(fixture.target)
        if (fixture.want) expect(actual, fixture.name).toEqual({ port: fixture.want, source: 'peer' })
        else expect(actual, fixture.name).toEqual({ port: null, reason: 'not-found' })
      } finally {
        kill.mockRestore()
        fs.rmSync(recordPath, { force: true })
      }
    }
  })

  afterEach(() => {
    // Restore env
    for (const key of Object.keys(process.env)) {
      if (!(key in originalEnv)) delete process.env[key]
    }
    Object.assign(process.env, originalEnv)
  })

  it('resolves a live peer port without launching the CLI', () => {
    publishPeerPort('my-scenario', '12345')
    const cliLookup = vi.fn(() => ({ port: null, reason: 'not-found' as const }))
    const resolve = createPortResolver(2_000, 'UI_PORT', { runtimeHome: () => testRuntimeHome, cliLookup })
    expect(resolve('my-scenario')).toBe('12345')
    expect(cliLookup).not.toHaveBeenCalled()
  })

  it('resolves a hyphenated peer name from its canonical peer slug', () => {
    publishPeerPort('agent-inbox', '9999')
    const resolve = createPortResolver()
    expect(resolve('agent-inbox')).toBe('9999')
  })

  it('resolves a live lifecycle peer record before the CLI fallback', () => {
    const runtimeHome = fs.mkdtempSync(path.join(os.tmpdir(), 'api-base-peer-'))
    fs.mkdirSync(path.join(runtimeHome, 'peers'), { recursive: true })
    fs.writeFileSync(path.join(runtimeHome, 'peers', 'peer-target.json'), JSON.stringify({
      schema_version: 1, scenario: 'peer-target', instance: 'live', owner_pid: process.pid,
      ports: { ui: 43123 }, provides: ['ui'],
    }))
    try {
      expect(createPortResolutionResolver(30_000, 'UI_PORT', { runtimeHome: () => runtimeHome })('peer-target')).toEqual({ port: '43123', source: 'peer' })
    } finally {
      fs.rmSync(runtimeHome, { recursive: true, force: true })
    }
  })

  it('honors an explicit VROOLI_HOME runtime root', () => {
    const runtimeHome = fs.mkdtempSync(path.join(os.tmpdir(), 'api-base-runtime-home-'))
    fs.mkdirSync(path.join(runtimeHome, 'peers'), { recursive: true })
    fs.writeFileSync(path.join(runtimeHome, 'peers', 'peer-target.json'), JSON.stringify({
      schema_version: 1, scenario: 'peer-target', instance: 'live', owner_pid: process.pid,
      ports: { ui: 43126 }, provides: ['ui'],
    }))
    const previous = process.env.VROOLI_HOME
    process.env.VROOLI_HOME = runtimeHome
    try {
      expect(resolveScenarioPort('peer-target', 'UI_PORT', {
        homeDir: () => { throw new Error('VROOLI_HOME must take precedence') },
        cliLookup: () => ({ port: null, reason: 'not-found' }),
      })).toEqual({ port: '43126', source: 'peer' })
    } finally {
      if (previous === undefined) delete process.env.VROOLI_HOME
      else process.env.VROOLI_HOME = previous
      fs.rmSync(runtimeHome, { recursive: true, force: true })
    }
  })

  it('uses variant peer records and normalizes public port aliases', () => {
    const runtimeHome = fs.mkdtempSync(path.join(os.tmpdir(), 'api-base-peer-variant-'))
    fs.mkdirSync(path.join(runtimeHome, 'peers'), { recursive: true })
    fs.writeFileSync(path.join(runtimeHome, 'peers', 'peer-target@shadow.json'), JSON.stringify({
      schema_version: 1, scenario: 'peer-target', instance: 'shadow', owner_pid: process.pid,
      ports: { api: 43125 }, provides: ['api'],
    }))
    try {
      expect(createPortResolutionResolver(30_000, 'API_PORT', { runtimeHome: () => runtimeHome })('peer-target@shadow'))
        .toEqual({ port: '43125', source: 'peer' })
    } finally {
      fs.rmSync(runtimeHome, { recursive: true, force: true })
    }
  })

  it('refreshes a cached peer port after a restart changes the published port', async () => {
    const runtimeHome = fs.mkdtempSync(path.join(os.tmpdir(), 'api-base-peer-drift-'))
    const recordPath = path.join(runtimeHome, 'peers', 'peer-target.json')
    fs.mkdirSync(path.dirname(recordPath), { recursive: true })
    const publish = (port: number) => fs.writeFileSync(recordPath, JSON.stringify({
      schema_version: 1, scenario: 'peer-target', instance: 'live', owner_pid: process.pid,
      ports: { ui: port }, provides: ['ui'],
    }))
    publish(43123)
    const resolve = createPortResolutionResolver(20, 'UI_PORT', { runtimeHome: () => runtimeHome })
    try {
      expect(resolve('peer-target')).toEqual({ port: '43123', source: 'peer' })
      publish(43124)
      await new Promise((r) => setTimeout(r, 35))
      expect(resolve('peer-target')).toEqual({ port: '43124', source: 'peer' })
    } finally {
      fs.rmSync(runtimeHome, { recursive: true, force: true })
    }
  })

  it('rejects stale, malformed, and out-of-range peer records', () => {
    const runtimeHome = fs.mkdtempSync(path.join(os.tmpdir(), 'api-base-peer-'))
    fs.mkdirSync(path.join(runtimeHome, 'peers'), { recursive: true })
    fs.writeFileSync(path.join(runtimeHome, 'peers', 'peer-target.json'), JSON.stringify({
      schema_version: 1, scenario: 'peer-target', instance: 'live', owner_pid: 2147483647,
      ports: { ui: 70000 },
    }))
    try {
      expect(createPortResolutionResolver(30_000, 'UI_PORT', {
        runtimeHome: () => runtimeHome,
        cliLookup: () => ({ port: null, reason: 'not-found', detail: 'No running target' }),
      })('peer-target')).toEqual({ port: null, reason: 'not-found', detail: 'No running target' })
    } finally {
      fs.rmSync(runtimeHome, { recursive: true, force: true })
    }
  })

  it('delegates desktop discovery without reading host peer files', () => {
    process.env.VROOLI_DESKTOP_MODE = 'true'
    const homeDir = vi.fn(() => '/must/not/be/read')
    const cliLookup = vi.fn(() => ({ port: '43210', source: 'cli' as const }))
    expect(resolveScenarioPort('notification-hub', 'API_PORT', { homeDir, cliLookup })).toEqual({ port: '43210', source: 'cli' })
    expect(homeDir).not.toHaveBeenCalled()
    expect(cliLookup).toHaveBeenCalledWith('notification-hub', 'API_PORT')
  })

  it('does not let a live peer record satisfy an explicit non-live variant', () => {
    publishPeerPort('peer-target', '43123')
    const cliLookup = vi.fn(() => ({ port: '43210', source: 'cli' as const }))
    expect(resolveScenarioPort('peer-target@shadow', 'UI_PORT', {
      runtimeHome: () => testRuntimeHome, cliLookup,
    })).toEqual({ port: '43210', source: 'cli' })
    expect(cliLookup).toHaveBeenCalledWith('peer-target@shadow', 'UI_PORT')
  })

  it('ignores scenario port environment variables and uses canonical CLI recovery', () => {
    const key = 'STALE_SCENARIO_UI_PORT'
    const previous = process.env[key]
    process.env[key] = '43210'
    const cliLookup = vi.fn(() => ({ port: '43211', source: 'cli' as const }))
    try {
      expect(resolveScenarioPort('stale-scenario', 'UI_PORT', {
        runtimeHome: () => testRuntimeHome,
        cliLookup,
      })).toEqual({ port: '43211', source: 'cli' })
      expect(cliLookup).toHaveBeenCalledWith('stale-scenario', 'UI_PORT')
    } finally {
      if (previous === undefined) delete process.env[key]
      else process.env[key] = previous
    }
  })

  it('returns distinct typed failures for invalid targets and CLI failures', () => {
    expect(resolveScenarioPort('../notification-hub', 'API_PORT')).toMatchObject({
      port: null, reason: 'invalid-target',
    })
    expect(resolveScenarioPort('notification-hub', 'API_PORT', {
      runtimeHome: () => path.join(os.tmpdir(), 'api-base-empty-home'),
      cliLookup: () => ({ port: null, reason: 'cli-failed', detail: 'fixture: CLI unavailable' }),
    })).toEqual({ port: null, reason: 'cli-failed', detail: 'fixture: CLI unavailable' })
  })

  it('classifies only the structured stopped-target CLI code as a negative lookup', () => {
    expect(parseScenarioPortCLIResponse('{"success":false,"error_code":"scenario_not_running","error":"Scenario is stopped"}')).toEqual({
      port: null, reason: 'not-found', detail: 'Scenario is stopped',
    })
    for (const code of [undefined, 'scenario_not_running', 'runtime_error', 'scenario_not_found']) {
      const response = JSON.stringify({ success: false, code, error: 'scenario is not running' })
      expect(parseScenarioPortCLIResponse(response)).toMatchObject({ port: null, reason: 'cli-failed' })
    }
    expect(classifyPortLookupFailureCode('scenario_not_running', 'Scenario is stopped')).toEqual({
      port: null, reason: 'not-found', detail: 'Scenario is stopped',
    })
    expect(parseScenarioPortCLIResponse('{"success":true,"port":"12345"}')).toMatchObject({
      port: null, reason: 'cli-failed',
    })
    expect(parseScenarioPortCLIResponse('{"success":false,"error":"scenario is not running"}')).toMatchObject({
      port: null, reason: 'cli-failed',
    })
  })

  it('returns null when the peer record and CLI recovery are unavailable', () => {
    const resolve = createPortResolver(2_000, 'UI_PORT', {
      runtimeHome: () => testRuntimeHome,
      cliLookup: () => ({ port: null, reason: 'not-found' }),
    })
    expect(resolve('nonexistent-scenario')).toBeNull()
  })

  it('caches resolved port for cacheTtlMs', () => {
    const resolve = createPortResolver(60_000)
    publishPeerPort('cached-scenario', '1111')

    // First call resolves
    expect(resolve('cached-scenario')).toBe('1111')

    // Remove peer record — should still return the cached value
    fs.rmSync(path.join(testRuntimeHome, 'peers', 'cached-scenario.json'))
    expect(resolve('cached-scenario')).toBe('1111')
  })

  it('expires cache after TTL', async () => {
    const resolve = createPortResolver(50) // 50ms TTL
    publishPeerPort('ttl-scenario', '2222')

    expect(resolve('ttl-scenario')).toBe('2222')

    // Wait for TTL to expire
    await new Promise((r) => setTimeout(r, 80))

    // Remove peer record — cache expired, so the CLI recovery miss is observed.
    fs.rmSync(path.join(testRuntimeHome, 'peers', 'ttl-scenario.json'))
    expect(resolve('ttl-scenario')).toBeNull()
  })

  it('retries a missing target after the short negative TTL when a peer appears', () => {
    vi.useFakeTimers()
    try {
      const cliLookup = vi.fn(() => ({ port: null, reason: 'not-found' as const }))
      const resolve = createPortResolver(60_000, 'UI_PORT', {
        runtimeHome: () => testRuntimeHome,
        cliLookup,
      })

      expect(resolve('missing-scenario')).toBeNull()
      publishPeerPort('missing-scenario', '3333')
      expect(resolve('missing-scenario')).toBeNull()
      expect(cliLookup).toHaveBeenCalledTimes(1)

      vi.advanceTimersByTime(501)
      expect(resolve('missing-scenario')).toBe('3333')
      expect(cliLookup).toHaveBeenCalledTimes(1)
    } finally {
      vi.useRealTimers()
    }
  })

  it('does not cache required desktop IPC or transient CLI failures', () => {
    const previous = process.env.VROOLI_DESKTOP_MODE
    process.env.VROOLI_DESKTOP_MODE = 'true'
    try {
      const cliLookup = vi.fn(() => ({ port: null, reason: 'runtime-required' as const }))
      const resolve = createPortResolutionResolver(2_000, 'UI_PORT', { cliLookup })
      expect(resolve('desktop-target')).toMatchObject({ reason: 'runtime-required' })
      expect(resolve('desktop-target')).toMatchObject({ reason: 'runtime-required' })
      expect(cliLookup).toHaveBeenCalledTimes(2)
    } finally {
      if (previous === undefined) delete process.env.VROOLI_DESKTOP_MODE
      else process.env.VROOLI_DESKTOP_MODE = previous
    }
  })

  it('uses the short default TTL for normal lookups', () => {
    const resolve = createPortResolver() // default TTL
    publishPeerPort('default-ttl', '4444')

    expect(resolve('default-ttl')).toBe('4444')

    // Remove peer record — it should still return the cached value within the TTL.
    fs.rmSync(path.join(testRuntimeHome, 'peers', 'default-ttl.json'))
    expect(resolve('default-ttl')).toBe('4444')
  })

  it('handles multiple scenarios independently', () => {
    publishPeerPort('scenario-a', '5555')
    publishPeerPort('scenario-b', '6666')
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

  it('handles bracketed IPv6 loopback addresses', () => {
    const req = mockRequest({ headers: { host: '[::1]:3000' } })
    expect(resolveExternalUrl(req, 'my-scenario', '9000')).toBe('http://localhost:9000')
  })

  it('uses the private LAN address for private IPv4 and IPv6 hosts', () => {
    expect(resolveExternalUrl(mockRequest({ headers: { host: '192.168.1.20:3000' } }), 'my-scenario', '9000'))
      .toBe('http://192.168.1.20:9000')
    expect(resolveExternalUrl(mockRequest({ headers: { host: '[fd12::20]:3000' } }), 'my-scenario', '9000'))
      .toBe('http://[fd12::20]:9000')
  })

  it('uses the first forwarded host when a proxy supplies browser context', () => {
    const req = mockRequest({ headers: { host: '127.0.0.1:3000', 'x-forwarded-host': 'web-console.example.com, proxy.internal' } })
    expect(resolveExternalUrl(req, 'my-scenario', '9000')).toBe('https://my-scenario.example.com')
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
      publishPeerPort('test-scenario', String(upstream.port))

      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/test-scenario/external-url')

      expect(res.status).toBe(200)
      expect(res.body.url).toBe(`http://localhost:${upstream.port}`)
      expect(res.body.scenario).toBe('test-scenario')
    })

    it('returns 503 when scenario port cannot be resolved', async () => {
      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/unknown-scenario/external-url')

      expect(res.status).toBe(503)
      expect(res.body.error).toBe('Scenario not available')
      expect(res.body.scenario).toBe('unknown-scenario')
    })

    it('returns 403 when scenario is not in allowedScenarios', async () => {
      publishPeerPort('blocked-scenario', '9999')
      embedded = await startEmbeddedServer({ allowedScenarios: ['allowed-only'] })
      const res = await makeRequest(embedded.port, '/embedded/blocked-scenario/external-url')

      expect(res.status).toBe(403)
      expect(res.body.error).toBe('Scenario not allowed')
      expect(res.body.scenario).toBe('blocked-scenario')
    })

    it('allows scenario that IS in allowedScenarios list', async () => {
      upstream = await startUpstreamServer()
      publishPeerPort('allowed-app', String(upstream.port))

      embedded = await startEmbeddedServer({ allowedScenarios: ['allowed-app'] })
      const res = await makeRequest(embedded.port, '/embedded/allowed-app/external-url')

      expect(res.status).toBe(200)
      expect(res.body.scenario).toBe('allowed-app')
    })
  })

  describe('proxy passthrough (USE /:scenario)', () => {
    it.each(['length', 'chunked', 'gzip'])('forwards parsed Connect JSON with correct UTF-8 framing (%s)', async (framing) => {
      upstream = await startUpstreamServer()
      publishPeerPort('body-proxy', String(upstream.port))
      embedded = await startEmbeddedServer({ timeoutMs: 200 }, express.json())
      const source = '{\n  "effortRef": "effort:café-東京-🚀", "pageSize": 3\n}\n'
      const expected = Buffer.from('{"effortRef":"effort:café-東京-🚀","pageSize":3}')
      const input = framing === 'gzip' ? gzipSync(source) : Buffer.from(source)

      const res = await makeRequest(embedded.port, '/embedded/body-proxy/agent_manager.v1.AgentManagerService/GetEffortBoard', {
        method: 'POST',
        headers: {
          'content-type': 'application/json; charset=utf-8',
          'connect-protocol-version': '1',
          ...(framing === 'chunked'
            ? { 'transfer-encoding': 'chunked' }
            : { 'content-length': String(input.length) }),
          ...(framing === 'gzip' ? { 'content-encoding': 'gzip' } : {}),
        },
        body: input,
      })

      expect(res.status).toBe(200)
      expect(res.body.method).toBe('POST')
      expect(res.body.body).toBe(expected.toString('base64'))
      expect(res.body.byteLength).toBe(expected.length)
      expect(res.body.headers['content-length']).toBe(String(expected.length))
      expect(res.body.headers['transfer-encoding']).toBeUndefined()
      expect(res.body.headers['content-encoding']).toBeUndefined()
      expect(res.body.headers['connect-protocol-version']).toBe('1')
    })

    it.each(['raw', 'text'])('forwards a retained %s parser body without JSON coercion', async (parser) => {
      upstream = await startUpstreamServer()
      publishPeerPort('body-proxy', String(upstream.port))
      const contentType = parser === 'raw' ? 'application/octet-stream' : 'text/plain'
      const source = parser === 'raw' ? Buffer.from([0, 255, 128, 13, 10]) : Buffer.from('café 東京 🚀')
      embedded = await startEmbeddedServer({}, parser === 'raw' ? express.raw() : express.text())

      const res = await makeRequest(embedded.port, '/embedded/body-proxy/agent_manager.v1.AgentManagerService/GetEffortBoard', {
        method: 'PUT',
        headers: { 'content-type': contentType, 'content-length': String(source.length) },
        body: source,
      })

      expect(res.status).toBe(200)
      expect(res.body.method).toBe('PUT')
      expect(res.body.body).toBe(source.toString('base64'))
      expect(res.body.headers['content-length']).toBe(String(source.length))
    })

    it.each(['missing', 'circular'])('rejects a consumed %s body before contacting upstream', async (representation) => {
      upstream = await startUpstreamServer()
      const received = vi.fn()
      upstream.server.on('request', received)
      publishPeerPort('body-proxy', String(upstream.port))
      embedded = await startEmbeddedServer({}, (req, res, next) => {
        express.json()(req, res, (error) => {
          if (error) return next(error)
          if (representation === 'missing') req.body = undefined
          else req.body.circular = req.body
          next()
        })
      })

      const res = await makeRequest(embedded.port, '/embedded/body-proxy/agent_manager.v1.AgentManagerService/GetEffortBoard', {
        method: 'POST',
        headers: { 'content-type': 'application/json', 'content-length': '2' },
        body: '{}',
      })

      expect(res.status).toBe(400)
      expect(res.body.error).toBe('Cannot forward consumed request body')
      expect(received).not.toHaveBeenCalled()
    })

    it('does not invent a body when the JSON parser receives an empty request', async () => {
      upstream = await startUpstreamServer()
      publishPeerPort('body-proxy', String(upstream.port))
      embedded = await startEmbeddedServer({}, express.json())

      const res = await makeRequest(embedded.port, '/embedded/body-proxy/agent_manager.v1.AgentManagerService/GetEffortBoard', {
        method: 'POST',
        headers: { 'content-type': 'application/json', 'content-length': '0' },
      })

      expect(res.status).toBe(200)
      expect(res.body.byteLength).toBe(0)
      expect(res.body.body).toBe('')
    })

    it('streams binary request chunks before the client has finished uploading', async () => {
      upstream = await startUpstreamServer()
      publishPeerPort('body-proxy', String(upstream.port))
      embedded = await startEmbeddedServer({}, express.json())
      const first = Buffer.from([0, 255, 128, 13, 10])
      const second = Buffer.from([254, 1, 2, 0])
      let firstObserved: Buffer | undefined
      const res = await makeRequest(embedded.port, '/embedded/body-proxy/agent_manager.v1.AgentManagerService/GetEffortBoard', {
        method: 'POST',
        headers: { 'content-type': 'application/octet-stream' },
        writeBody: (request) => {
          upstream.server.once('request', (incoming) => {
            incoming.once('data', (chunk: Buffer) => {
              firstObserved = chunk
              request.end(second)
            })
          })
          request.write(first)
        },
      })

      expect(res.status).toBe(200)
      expect(firstObserved).toEqual(first)
      expect(res.body.body).toBe(Buffer.concat([first, second]).toString('base64'))
      expect(res.body.headers['transfer-encoding']).toBe('chunked')
      expect(res.body.headers['content-length']).toBeUndefined()
    })

    it('proxies GET request to upstream and returns response', async () => {
      upstream = await startUpstreamServer()
      publishPeerPort('proxy-target', String(upstream.port))

      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/proxy-target/health')

      expect(res.status).toBe(200)
      expect(res.body).toEqual({ ok: true })
    })

    it('proxies root path when only scenario name is given', async () => {
      upstream = await startUpstreamServer()
      publishPeerPort('root-proxy', String(upstream.port))

      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/root-proxy/')

      expect(res.status).toBe(200)
      expect(res.body).toContain('hello')
    })

    it('strips /embedded/{scenario} prefix before proxying', async () => {
      upstream = await startUpstreamServer()
      publishPeerPort('strip-test', String(upstream.port))

      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/strip-test/api/data?foo=bar')

      expect(res.status).toBe(200)
      expect(res.body.path).toBe('/api/data?foo=bar')
    })

    it('returns 503 when scenario is not available for proxy', async () => {
      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/no-such-scenario/anything')

      expect(res.status).toBe(503)
      expect(res.body.error).toBe('Scenario not available')
    })

    it('returns 403 when scenario not in allowed list for proxy', async () => {
      publishPeerPort('forbidden', '9999')
      embedded = await startEmbeddedServer({ allowedScenarios: ['only-this'] })
      const res = await makeRequest(embedded.port, '/embedded/forbidden/')

      expect(res.status).toBe(403)
      expect(res.body.error).toBe('Scenario not allowed')
    })

    it('returns 502 when upstream is unreachable', async () => {
      // Point to a port with nothing listening
      publishPeerPort('dead-upstream', '19')

      const originalError = console.error
      console.error = vi.fn() // Suppress expected error log

      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/dead-upstream/')

      expect(res.status).toBe(502)
      expect(res.body.error).toBe('Failed to proxy to scenario UI')
      expect(res.body.scenario).toBe('dead-upstream')

      console.error = originalError
    })

    it('respects custom upstreamHost', async () => {
      upstream = await startUpstreamServer()
      // Use 127.0.0.1 explicitly — same as default but confirming it works
      publishPeerPort('custom-host', String(upstream.port))

      embedded = await startEmbeddedServer({ upstreamHost: '127.0.0.1' })
      const res = await makeRequest(embedded.port, '/embedded/custom-host/health')

      expect(res.status).toBe(200)
      expect(res.body).toEqual({ ok: true })
    })

    it('returns 404 for paths upstream doesn\'t handle', async () => {
      upstream = await startUpstreamServer()
      publishPeerPort('notfound-test', String(upstream.port))

      embedded = await startEmbeddedServer()
      const res = await makeRequest(embedded.port, '/embedded/notfound-test/nonexistent')

      expect(res.status).toBe(404)
    })
  })

  describe('proxy timeout', () => {
    it('returns 504 when upstream does not respond within timeoutMs', async () => {
      upstream = await startUpstreamServer()
      publishPeerPort('timeout-scenario', String(upstream.port))

      const originalError = console.error
      console.error = vi.fn()

      embedded = await startEmbeddedServer({ timeoutMs: 200 })
      const res = await makeRequest(embedded.port, '/embedded/timeout-scenario/hang')

      expect(res.status).toBe(504)
      expect(res.body.error).toBe('Proxy timeout')
      expect(res.body.scenario).toBe('timeout-scenario')

      console.error = originalError
    })
  })

  describe('port resolution caching in router context', () => {
    it('caches port resolution across multiple requests', async () => {
      upstream = await startUpstreamServer()
      publishPeerPort('cache-hit', String(upstream.port))

      embedded = await startEmbeddedServer({ cacheTtlMs: 60_000 })

      // First request — resolves and caches
      const res1 = await makeRequest(embedded.port, '/embedded/cache-hit/health')
      expect(res1.status).toBe(200)

      // Remove env var — should still work from cache
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
    publishPeerPort('tmpl-scenario', '12345')

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
  })

  it('registers /embedded routes when embeddedProxy is an options object', async () => {
    const { createScenarioServer } = await import('../../server/template.js')
    publishPeerPort('opt-scenario', '12345')

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
