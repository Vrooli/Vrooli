import { afterEach, describe, expect, it } from 'vitest'
import * as fs from 'node:fs'
import * as http from 'node:http'
import * as os from 'node:os'
import * as path from 'node:path'
import type { Server } from 'node:http'
import { createScenarioServer } from '../../server/template.js'

const servers: Server[] = []
const dirs: string[] = []

function tempDist(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'experience-template-'))
  dirs.push(dir)
  fs.writeFileSync(path.join(dir, 'index.html'), '<!doctype html><html><head><title>demo</title></head><body><div id="root"></div></body></html>')
  return dir
}

function request(server: Server, pathname: string, headers: Record<string, string> = {}): Promise<{ status: number; body: string; headers: http.IncomingHttpHeaders }> {
  const address = server.address()
  if (!address || typeof address === 'string') throw new Error('server not listening')
  return new Promise((resolve, reject) => {
    const req = http.request({ hostname: '127.0.0.1', port: address.port, path: pathname, method: 'GET', headers }, (res) => {
      let body = ''
      res.setEncoding('utf-8')
      res.on('data', (chunk) => { body += chunk })
      res.on('end', () => resolve({ status: res.statusCode || 0, body, headers: res.headers }))
    })
    req.on('error', reject)
    req.end()
  })
}

afterEach(async () => {
  await Promise.all(servers.splice(0).map((server) => new Promise<void>((resolve) => server.close(() => resolve()))))
  while (dirs.length > 0) fs.rmSync(dirs.pop()!, { recursive: true, force: true })
})

function start(options: Parameters<typeof createScenarioServer>[0]): Server {
  const app = createScenarioServer({
    uiPort: 43100,
    apiPort: 43101,
    serviceName: 'demo',
    disableLifecycleGuard: true,
    ...options,
  })
  const server = app.listen(0)
  servers.push(server)
  return server
}

describe('experience telemetry in the shared server', () => {
  it('injects the agent into the root document and deep routes', async () => {
    const distDir = tempDist()
    const server = start({ distDir, experienceTelemetry: { enabled: true, dataDir: path.join(distDir, 'sessions') } })

    const root = await request(server, '/', { host: '127.0.0.1:43100', accept: 'text/html' })
    expect(root.status).toBe(200)
    expect(root.body).toContain('data-vrooli-experience="1"')
    expect(root.body).toContain('src="/__vrooli/experience/agent.js"')
    expect(root.body).toContain('<base data-vrooli-base="injected" href="/">')
    expect(root.headers['cache-control']).toBe('no-cache')
    const deep = await request(server, '/deep/link', { host: '127.0.0.1:43100', accept: 'text/html' })
    expect(deep.status).toBe(200)
    expect(deep.body).toContain('data-vrooli-experience="1"')
  })

  it('stamps the served build identity onto the agent tag', async () => {
    const distDir = tempDist()
    const previous = process.env.VROOLI_BUILD_IDENTITY
    process.env.VROOLI_BUILD_IDENTITY = 'sha256:test-build'
    try {
      const server = start({
        distDir,
        version: '1.2.3',
        experienceTelemetry: { enabled: true, dataDir: path.join(distDir, 'sessions') },
      })
      const root = await request(server, '/', { host: '127.0.0.1:43100', accept: 'text/html' })
      expect(root.body).toContain('data-build="sha256:test-build"')
    } finally {
      if (previous === undefined) delete process.env.VROOLI_BUILD_IDENTITY
      else process.env.VROOLI_BUILD_IDENTITY = previous
    }
  })

  it('leaves tunnel and automation requests uninstrumented', async () => {
    const distDir = tempDist()
    const server = start({ distDir, experienceTelemetry: { enabled: true, dataDir: path.join(distDir, 'sessions') } })

    const tunnel = await request(server, '/', { host: 'abc.trycloudflare.com', accept: 'text/html' })
    expect(tunnel.status).toBe(200)
    expect(tunnel.body).not.toContain('data-vrooli-experience')

    const optedOut = await request(server, '/', {
      host: '127.0.0.1:43100',
      accept: 'text/html',
      'x-vrooli-experience': 'off',
    })
    expect(optedOut.body).not.toContain('data-vrooli-experience')
  })

  it('can be disabled explicitly', async () => {
    const distDir = tempDist()
    const server = start({ distDir, experienceTelemetry: false })

    const root = await request(server, '/', { host: '127.0.0.1:43100', accept: 'text/html' })
    expect(root.body).not.toContain('data-vrooli-experience')

    const ingest = await request(server, '/__vrooli/experience/events', { host: '127.0.0.1:43100' })
    expect(ingest.status).toBe(404)
  })

  it('persists events posted to the mounted ingest route', async () => {
    const distDir = tempDist()
    const dataDir = path.join(distDir, 'sessions')
    const server = start({ distDir, experienceTelemetry: { enabled: true, dataDir } })
    const address = server.address()
    if (!address || typeof address === 'string') throw new Error('no address')

    const response = await fetch(`http://127.0.0.1:${address.port}/__vrooli/experience/events`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        v: 1,
        sessionId: 'abcdefgh12345678',
        events: [{ type: 'action', ts: Date.now(), seq: 0, action: 'click', element: { tag: 'button', path: 'button', fp: 'f' } }],
      }),
    })
    expect(response.status).toBe(204)

    const files = fs.readdirSync(dataDir)
    expect(files).toEqual(['abcdefgh12345678.jsonl'])
    expect(fs.readFileSync(path.join(dataDir, files[0]!), 'utf-8')).toContain('"scenario":"demo"')
  })
})
