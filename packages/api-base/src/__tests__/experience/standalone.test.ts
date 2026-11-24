import { afterEach, describe, expect, it } from 'vitest'
import express from 'express'
import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'
import type { Server } from 'node:http'
import { installExperienceTelemetry } from '../../experience/standalone.js'

const servers: Server[] = []
const dirs: string[] = []

function setup(): { base: string; dir: string; dataDir: string } {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'experience-standalone-'))
  dirs.push(dir)
  fs.mkdirSync(path.join(dir, 'dist'))
  fs.writeFileSync(path.join(dir, 'dist', 'index.html'), '<!doctype html><html><head><title>x</title></head><body>root</body></html>')
  fs.writeFileSync(path.join(dir, 'dist', 'deep.html'), '<!doctype html><html><head></head><body>deep</body></html>')
  fs.writeFileSync(path.join(dir, 'agent.js'), 'export const installed = true\n')
  const dataDir = path.join(dir, 'sessions')

  const app = express()
  const telemetry = installExperienceTelemetry(app, {
    serviceName: 'custom',
    distDir: path.join(dir, 'dist'),
    dataDir,
    assetFile: path.join(dir, 'agent.js'),
    enabled: true,
  })
  if (!telemetry) throw new Error('telemetry did not resolve')

  app.use(express.static(path.join(dir, 'dist'), { index: false }))
  app.get('/fallback', (_req, res) => {
    res.sendFile(path.join(dir, 'dist', 'deep.html'))
  })
  app.post('/api/echo', express.json(), (req, res) => res.json(req.body))

  const server = app.listen(0)
  servers.push(server)
  const address = server.address()
  if (!address || typeof address === 'string') throw new Error('no address')
  return { base: `http://127.0.0.1:${address.port}`, dir, dataDir }
}

afterEach(async () => {
  await Promise.all(servers.splice(0).map((server) => new Promise<void>((resolve) => server.close(() => resolve()))))
  while (dirs.length > 0) fs.rmSync(dirs.pop()!, { recursive: true, force: true })
})

const headers = { host: '127.0.0.1', accept: 'text/html' }

describe('installExperienceTelemetry for custom servers', () => {
  it('injects the root document and serves the agent', async () => {
    const { base } = setup()
    const root = await fetch(`${base}/`, { headers })
    const body = await root.text()
    expect(root.status).toBe(200)
    expect(body).toContain('data-vrooli-experience="1"')
    expect(body).toContain('data-scenario="custom"')

    const asset = await fetch(`${base}/__vrooli/experience/agent.js`)
    expect(asset.status).toBe(200)
  })

  it('binds standalone telemetry to the lifecycle served build identity', async () => {
    const previous = process.env.VROOLI_BUILD_IDENTITY
    process.env.VROOLI_BUILD_IDENTITY = 'sha256:custom-server-build'
    try {
      const { base } = setup()
      const root = await fetch(`${base}/`, { headers })
      expect(await root.text()).toContain('data-build="sha256:custom-server-build"')
    } finally {
      if (previous === undefined) delete process.env.VROOLI_BUILD_IDENTITY
      else process.env.VROOLI_BUILD_IDENTITY = previous
    }
  })

  it('injects HTML served through sendFile', async () => {
    const { base } = setup()
    const response = await fetch(`${base}/fallback`, { headers })
    expect(await response.text()).toContain('data-vrooli-experience="1"')
  })

  it('does not alter JSON responses or API routes', async () => {
    const { base } = setup()
    const response = await fetch(`${base}/api/echo`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ hello: 'world' }),
    })
    expect(await response.json()).toEqual({ hello: 'world' })
  })

  it('persists ingested events', async () => {
    const { base, dataDir } = setup()
    const response = await fetch(`${base}/__vrooli/experience/events`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({
        v: 1,
        sessionId: 'abcdefgh12345678',
        events: [{ type: 'action', ts: Date.now(), seq: 0, action: 'click', element: { tag: 'button', path: 'button', fp: 'f' } }],
      }),
    })
    expect(response.status).toBe(204)
    expect(fs.readdirSync(dataDir)).toEqual(['abcdefgh12345678.jsonl'])
  })
})
