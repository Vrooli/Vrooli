/**
 * Tests for server/app-notifications.ts: the /vrooli-notify routes a scenario
 * web server mounts when its notifications.json opts into app delivery.
 */

import { describe, it, expect, afterEach } from 'vitest'
import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'
import * as http from 'node:http'
import express from 'express'
import type { Server } from 'node:http'
import {
  APP_DELIVERY_SERVICE,
  appDeliveryDeclared,
  createAppNotificationsRouter,
  resolveAppNotifications,
} from '../../server/app-notifications.js'

interface Seen {
  method: string
  url: string
  headers: http.IncomingHttpHeaders
  body: string
}

const servers: Server[] = []

afterEach(async () => {
  await Promise.all(servers.splice(0).map((server) => new Promise((resolve) => server.close(resolve))))
})

function listen(server: Server): Promise<number> {
  servers.push(server)
  return new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      const addr = server.address()
      if (!addr || typeof addr === 'string') throw new Error('No address')
      resolve(addr.port)
    })
  })
}

/** A stand-in Notification Hub that records each request. */
async function startHub(): Promise<{ base: string; seen: Seen[] }> {
  const seen: Seen[] = []
  const server = http.createServer((req, res) => {
    const chunks: Buffer[] = []
    req.on('data', (chunk: Buffer) => chunks.push(chunk))
    req.on('end', () => {
      seen.push({ method: req.method ?? '', url: req.url ?? '', headers: req.headers, body: Buffer.concat(chunks).toString('utf-8') })
      res.writeHead(200, { 'Content-Type': 'application/json' })
      res.end('{"ok":true}')
    })
  })
  return { base: `http://127.0.0.1:${await listen(server)}`, seen }
}

async function startScenarioServer(hubBaseUrl: string, withJsonParserAfter = true): Promise<string> {
  const app = express()
  app.use('/vrooli-notify', createAppNotificationsRouter({ scenario: 'web-console', hubBaseUrl }))
  if (withJsonParserAfter) app.use(express.json())
  return `http://127.0.0.1:${await listen(http.createServer(app))}`
}

describe('appDeliveryDeclared / resolveAppNotifications', () => {
  function manifestDir(contents: string): string {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), 'app-notify-'))
    fs.mkdirSync(path.join(root, '.vrooli'))
    fs.mkdirSync(path.join(root, 'ui'))
    fs.writeFileSync(path.join(root, '.vrooli', 'notifications.json'), contents)
    return root
  }

  it('turns on only when the scenario manifest opts in', () => {
    const on = manifestDir('{"displayName":"Web Console","appDelivery":true}')
    const off = manifestDir('{"displayName":"Web Console"}')
    expect(appDeliveryDeclared(path.join(on, '.vrooli', 'notifications.json'))).toBe(true)
    expect(resolveAppNotifications(undefined, 'web-console', path.join(on, 'ui'))?.scenario).toBe('web-console')
    expect(resolveAppNotifications(undefined, 'web-console', path.join(off, 'ui'))).toBeNull()
    expect(resolveAppNotifications(false, 'web-console', path.join(on, 'ui'))).toBeNull()
    expect(resolveAppNotifications(true, 'web-console', path.join(off, 'ui'))?.scenario).toBe('web-console')
    expect(appDeliveryDeclared('/nonexistent/notifications.json')).toBe(false)
  })
})

describe('createAppNotificationsRouter', () => {
  it('serves the shared push worker as JavaScript', async () => {
    const hub = await startHub()
    const base = await startScenarioServer(hub.base)
    const res = await fetch(`${base}/vrooli-notify/sw.js`)
    expect(res.status).toBe(200)
    expect(res.headers.get('content-type')).toContain('application/javascript')
    const source = await res.text()
    expect(source).toContain('addEventListener("push"')
    expect(source).toContain('showNotification')
    expect(hub.seen).toHaveLength(0)
  })

  it('forwards app-delivery calls with the server-named scenario and only sign-in headers', async () => {
    const hub = await startHub()
    const base = await startScenarioServer(hub.base)
    const res = await fetch(`${base}/vrooli-notify/rpc/GetAppState`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Cf-Access-Jwt-Assertion': 'access-jwt',
        'X-Vrooli-App-Scenario': 'agent-manager',
        Cookie: 'session=secret',
      },
      body: JSON.stringify({ endpoint: 'https://web.push.apple.com/abc' }),
    })
    expect(res.status).toBe(200)
    const [call] = hub.seen
    expect(call?.url).toBe(`/${APP_DELIVERY_SERVICE}/GetAppState`)
    expect(call?.headers['x-vrooli-app-scenario']).toBe('web-console')
    expect(call?.headers['cf-access-jwt-assertion']).toBe('access-jwt')
    expect(call?.headers.cookie).toBeUndefined()
    expect(JSON.parse(call?.body ?? '{}')).toEqual({ endpoint: 'https://web.push.apple.com/abc' })
  })

  it('refuses methods outside the scenario-scoped set', async () => {
    const hub = await startHub()
    const base = await startScenarioServer(hub.base)
    const res = await fetch(`${base}/vrooli-notify/rpc/RemoveDeviceApp`, { method: 'POST', body: '{}' })
    expect(res.status).toBe(404)
    expect(hub.seen).toHaveLength(0)
  })

  it('pins catalog reads and preference writes to this scenario', async () => {
    const hub = await startHub()
    const base = await startScenarioServer(hub.base)
    await fetch(`${base}/vrooli-notify/catalog?source_scenario=agent-manager`)
    await fetch(`${base}/vrooli-notify/preferences`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ source_scenario: 'agent-manager', category: 'session-finished', enabled: false }),
    })
    expect(hub.seen[0]?.url).toBe('/api/v1/notifications/catalog?source_scenario=web-console')
    expect(hub.seen[1]?.method).toBe('PUT')
    expect(JSON.parse(hub.seen[1]?.body ?? '{}')).toEqual({ source_scenario: 'web-console', category: 'session-finished', enabled: false })
  })

  it('answers 503 when Notification Hub is unreachable', async () => {
    const base = await startScenarioServer('http://127.0.0.1:1')
    const res = await fetch(`${base}/vrooli-notify/rpc/ConfirmApp`, { method: 'POST', body: '{}' })
    expect(res.status).toBe(503)
  })
})

describe('createScenarioServer appNotifications', () => {
  it('routes /vrooli-notify ahead of the body parser and the SPA fallback', async () => {
    const { createScenarioServer } = await import('../../server/template.js')
    const hub = await startHub()
    const app = createScenarioServer({
      uiPort: 33001,
      apiPort: 8080,
      distDir: './dist',
      serviceName: 'web-console',
      appNotifications: { enabled: true, hubBaseUrl: hub.base },
    })
    const base = `http://127.0.0.1:${await listen(http.createServer(app))}`
    const rpc = await fetch(`${base}/vrooli-notify/rpc/EnableApp`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ endpoint: 'e', pairWithGroup: 'g' }),
    })
    expect(rpc.status).toBe(200)
    expect(JSON.parse(hub.seen[0]?.body ?? '{}')).toEqual({ endpoint: 'e', pairWithGroup: 'g' })
    const worker = await fetch(`${base}/vrooli-notify/sw.js`)
    expect(worker.headers.get('content-type')).toContain('application/javascript')
    const unknown = await fetch(`${base}/vrooli-notify/missing`)
    expect(unknown.status).toBe(404)
  })

  it('stays off for a scenario that has not opted in', async () => {
    const { createScenarioServer } = await import('../../server/template.js')
    const app = createScenarioServer({ uiPort: 33002, apiPort: 8080, distDir: './dist', serviceName: 'no-notify' })
    const base = `http://127.0.0.1:${await listen(http.createServer(app))}`
    const res = await fetch(`${base}/vrooli-notify/sw.js`)
    expect(res.headers.get('content-type') ?? '').not.toContain('application/javascript')
  })
})
