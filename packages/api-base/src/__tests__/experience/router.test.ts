import { afterEach, describe, expect, it } from 'vitest'
import express from 'express'
import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'
import type { Server } from 'node:http'
import { createExperienceTelemetryRouter } from '../../experience/router.js'
import { resolveExperienceTelemetry } from '../../experience/gate.js'

const servers: Server[] = []
const dirs: string[] = []

function setup(options: { maxBodyBytes?: number } = {}): { base: string; dataDir: string } {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'experience-router-'))
  dirs.push(dir)
  const agentFile = path.join(dir, 'agent.js')
  fs.writeFileSync(agentFile, 'export const installed = true\n', 'utf-8')
  const dataDir = path.join(dir, 'sessions')

  const resolved = resolveExperienceTelemetry(
    { enabled: true, dataDir, assetFile: agentFile, ...(options.maxBodyBytes ? { maxBodyBytes: options.maxBodyBytes } : {}) },
    { NODE_ENV: 'development' } as NodeJS.ProcessEnv,
    'demo',
  )!

  const app = express()
  app.use(createExperienceTelemetryRouter(resolved))
  const server = app.listen(0)
  servers.push(server)
  const address = server.address()
  if (!address || typeof address === 'string') throw new Error('no address')
  return { base: `http://127.0.0.1:${address.port}`, dataDir }
}

afterEach(async () => {
  await Promise.all(servers.splice(0).map((server) => new Promise<void>((resolve) => server.close(() => resolve()))))
  while (dirs.length > 0) fs.rmSync(dirs.pop()!, { recursive: true, force: true })
})

const event = { type: 'action', ts: Date.now(), seq: 0, action: 'click', element: { tag: 'button', path: 'button', fp: 'f' } }

describe('experience telemetry router', () => {
  it('serves the agent with a JavaScript content type', async () => {
    const { base } = setup()
    const response = await fetch(`${base}/__vrooli/experience/agent.js`)
    expect(response.status).toBe(200)
    expect(response.headers.get('content-type')).toContain('text/javascript')
    expect(await response.text()).toContain('installed')
  })

  it('accepts JSON and text/plain batches (sendBeacon sends text/plain)', async () => {
    const { base, dataDir } = setup()
    const body = JSON.stringify({ v: 1, sessionId: 'abcdefgh12345678', events: [event] })

    const json = await fetch(`${base}/__vrooli/experience/events`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body,
    })
    expect(json.status).toBe(204)

    const plain = await fetch(`${base}/__vrooli/experience/events`, {
      method: 'POST',
      headers: { 'content-type': 'text/plain' },
      body: body.replace('abcdefgh12345678', 'abcdefgh87654321'),
    })
    expect(plain.status).toBe(204)

    const files = fs.readdirSync(dataDir).sort()
    expect(files).toEqual(['abcdefgh12345678.jsonl', 'abcdefgh87654321.jsonl'])
    const line = JSON.parse(fs.readFileSync(path.join(dataDir, files[0]!), 'utf-8').trim())
    expect(line.scenario).toBe('demo')
    expect(line.events).toHaveLength(1)
  })

  it('rejects invalid ids, empty batches, and unknown protocol versions', async () => {
    const { base } = setup()
    const send = (payload: unknown) => fetch(`${base}/__vrooli/experience/events`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify(payload),
    })

    expect((await send({ v: 1, sessionId: '../x', events: [event] })).status).toBe(400)
    expect((await send({ v: 1, sessionId: 'abcdefgh12345678', events: [] })).status).toBe(400)
    expect((await send({ v: 99, sessionId: 'abcdefgh12345678', events: [event] })).status).toBe(400)
    expect((await send({ v: 1, sessionId: 'abcdefgh12345678', events: [{ type: 'Action!', ts: 1 }] })).status).toBe(400)
  })

  it('caps the request body', async () => {
    const { base } = setup({ maxBodyBytes: 512 })
    const response = await fetch(`${base}/__vrooli/experience/events`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ v: 1, sessionId: 'abcdefgh12345678', events: [event], padding: 'x'.repeat(2048) }),
    })
    expect(response.status).toBe(413)
  })

  it('reports bounded status without event contents', async () => {
    const { base, dataDir } = setup()
    await fetch(`${base}/__vrooli/experience/events`, {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ v: 1, sessionId: 'abcdefgh12345678', events: [event] }),
    })
    const response = await fetch(`${base}/__vrooli/experience/status`)
    const status = await response.json()
    expect(response.status).toBe(200)
    expect(status.scenario).toBe('demo')
    expect(status.sessions).toBe(1)
    expect(status.sessionDir).toBe(dataDir)
    expect(JSON.stringify(status)).not.toContain('click')
  })
})
