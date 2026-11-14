/**
 * Built-in app notifications for scenario web servers.
 *
 * A scenario opts in with `"appDelivery": true` in its
 * `.vrooli/notifications.json` (or the `appNotifications` server option).
 * The server then serves, under /vrooli-notify/:
 *
 *   GET  /vrooli-notify/sw.js            the shared push worker (scope /vrooli-notify/)
 *   POST /vrooli-notify/rpc/:method      Notification Hub's AppDeliveryService
 *   GET  /vrooli-notify/catalog          this scenario's notification categories
 *   GET  /vrooli-notify/preferences      the operator's choices for them
 *   PUT  /vrooli-notify/preferences      change one choice
 *
 * Every hub call names this scenario in x-vrooli-app-scenario from the
 * server's own configuration, never from the browser, and the preference
 * routes are pinned to this scenario's source. Only the operator's sign-in
 * headers are forwarded. Contract:
 * scenarios/notification-hub/docs/concepts/NOTIFICATIONS.md §"App delivery".
 */

import * as fs from 'node:fs'
import * as path from 'node:path'
import { Router, type Request, type Response } from 'express'
import { createPortResolver } from './embedded.js'
import { APP_NOTIFICATIONS_WORKER_SOURCE } from './app-notifications-worker.js'
import type { AppNotificationsOptions } from '../shared/types.js'

export type { AppNotificationsOptions }

export const APP_NOTIFICATIONS_PREFIX = '/vrooli-notify'
export const APP_DELIVERY_SERVICE = 'vrooli.notification_hub.v1.appdelivery.AppDeliveryService'
export const APP_SCENARIO_HEADER = 'x-vrooli-app-scenario'

const RPC_METHODS = new Set(['GetAppState', 'EnableApp', 'ConfirmApp', 'DisableApp'])
const FORWARDED_HEADERS = [
  'authorization',
  'cf-access-jwt-assertion',
  'cf-access-client-id',
  'cf-access-client-secret',
  'connect-protocol-version',
  'content-type',
  'user-agent',
  'x-request-id',
  'x-vrooli-device-id',
  'x-vrooli-test-mode',
]
const MAX_BODY_BYTES = 64 * 1024

/** True when the scenario's notifications manifest sets "appDelivery": true. */
export function appDeliveryDeclared(manifestPath: string): boolean {
  try {
    const parsed = JSON.parse(fs.readFileSync(manifestPath, 'utf-8')) as { appDelivery?: unknown }
    return parsed.appDelivery === true
  } catch {
    return false
  }
}

export function resolveAppNotifications(
  setting: AppNotificationsOptions | boolean | undefined,
  serviceName: string,
  cwd: string = process.cwd(),
): Required<Pick<AppNotificationsOptions, 'scenario'>> & AppNotificationsOptions | null {
  if (setting === false) return null
  const options: AppNotificationsOptions = setting === true || setting === undefined ? {} : setting
  if (options.enabled === false) return null
  const manifestPath = options.manifestPath ?? path.resolve(cwd, '..', '.vrooli', 'notifications.json')
  const enabled = setting === true || options.enabled === true || appDeliveryDeclared(manifestPath)
  if (!enabled) return null
  return { ...options, scenario: options.scenario ?? serviceName }
}

function hubBaseResolver(options: AppNotificationsOptions): () => string | null {
  if (options.hubBaseUrl) {
    const base = options.hubBaseUrl.replace(/\/+$/, '')
    return () => base
  }
  const resolvePort = createPortResolver(30_000, 'API_PORT')
  return () => {
    const port = resolvePort('notification-hub')
    return port ? `http://127.0.0.1:${port}` : null
  }
}

async function readBody(req: Request): Promise<Buffer> {
  if (Buffer.isBuffer(req.body)) return req.body
  if (req.body && typeof req.body === 'object' && Object.keys(req.body as object).length > 0) {
    return Buffer.from(JSON.stringify(req.body))
  }
  const chunks: Buffer[] = []
  let size = 0
  for await (const chunk of req) {
    const buf = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk as string)
    size += buf.length
    if (size > MAX_BODY_BYTES) throw new Error('request body is too large')
    chunks.push(buf)
  }
  return Buffer.concat(chunks)
}

/**
 * Router mounted at /vrooli-notify by createScenarioServer when app
 * notifications are enabled.
 */
export function createAppNotificationsRouter(options: AppNotificationsOptions & { scenario: string }): Router {
  const router = Router()
  const resolveHub = hubBaseResolver(options)
  const timeoutMs = options.timeoutMs ?? 15_000
  const scenario = options.scenario

  async function forward(req: Request, res: Response, targetPath: string, body?: Buffer) {
    const base = resolveHub()
    if (!base) {
      res.status(503).json({ code: 'unavailable', message: 'Notification Hub is not running' })
      return
    }
    const headers: Record<string, string> = {}
    for (const name of FORWARDED_HEADERS) {
      const value = req.headers[name]
      if (typeof value === 'string' && value !== '') headers[name] = value
    }
    headers[APP_SCENARIO_HEADER] = scenario
    try {
      const upstream = await fetch(base + targetPath, {
        method: req.method,
        headers,
        body: body && body.length > 0 ? new Uint8Array(body) : undefined,
        signal: AbortSignal.timeout(timeoutMs),
      })
      res.status(upstream.status)
      const contentType = upstream.headers.get('content-type')
      if (contentType) res.setHeader('Content-Type', contentType)
      res.setHeader('Cache-Control', 'no-store')
      res.send(Buffer.from(await upstream.arrayBuffer()))
    } catch {
      res.status(503).json({ code: 'unavailable', message: 'Notification Hub did not answer' })
    }
  }

  router.get('/sw.js', (_req, res) => {
    res.setHeader('Content-Type', 'application/javascript; charset=utf-8')
    res.setHeader('Cache-Control', 'no-cache')
    res.send(APP_NOTIFICATIONS_WORKER_SOURCE)
  })

  router.post('/rpc/:method', async (req, res) => {
    const method = req.params.method ?? ''
    if (!RPC_METHODS.has(method)) {
      res.status(404).json({ code: 'not_found', message: 'Unknown app-notification method' })
      return
    }
    let body: Buffer
    try {
      body = await readBody(req)
    } catch {
      res.status(413).json({ code: 'invalid_argument', message: 'Request body is too large' })
      return
    }
    await forward(req, res, `/${APP_DELIVERY_SERVICE}/${method}`, body)
  })

  router.get('/catalog', (req, res) => {
    void forward(req, res, `/api/v1/notifications/catalog?source_scenario=${encodeURIComponent(scenario)}`)
  })

  router.get('/preferences', (req, res) => {
    void forward(req, res, `/api/v1/notifications/preferences?source_scenario=${encodeURIComponent(scenario)}`)
  })

  router.put('/preferences', async (req, res) => {
    let parsed: Record<string, unknown>
    try {
      const raw = (await readBody(req)).toString('utf-8')
      parsed = JSON.parse(raw || '{}') as Record<string, unknown>
    } catch {
      res.status(400).json({ code: 'invalid_argument', message: 'Preference must be JSON' })
      return
    }
    // A scenario's page may change only its own scenario's preferences.
    parsed.source_scenario = scenario
    req.headers['content-type'] = 'application/json'
    await forward(req, res, '/api/v1/notifications/preferences', Buffer.from(JSON.stringify(parsed)))
  })

  return router
}
