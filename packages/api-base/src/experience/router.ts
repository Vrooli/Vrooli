/**
 * Local experience telemetry endpoints.
 *
 * - `GET  <assetPath>`   the browser agent module
 * - `POST <ingestPath>`  batched events, appended to a per-session JSONL file
 * - `GET  <statusPath>`  bounded local status (no event contents)
 *
 * Mounted by `createScenarioServer` behind the scenario origin policy, so only
 * same-origin pages can write. Bodies are size-capped, session ids are
 * validated, and the scenario name is server-owned so a page cannot write into
 * another scenario's history.
 */

import express, { type Request, type Response, type NextFunction, type Router } from 'express'
import * as fs from 'node:fs'
import { fileURLToPath } from 'node:url'
import {
  EXPERIENCE_PROTOCOL_VERSION,
  type ExperienceBatch,
  type ExperienceEvent,
  type ResolvedExperienceTelemetry,
} from './types.js'
import { appendBatch, isSessionId, pruneSessions, sessionStatus } from './store.js'

const EVENT_TYPE = /^[a-z][a-z0-9_]{0,40}$/

function readPayload(req: Request): unknown {
  if (typeof req.body === 'string') {
    try {
      return JSON.parse(req.body)
    } catch {
      return null
    }
  }
  return req.body
}

interface ParsedBatch {
  sessionId: string
  build?: string
  events: ExperienceEvent[]
}

function parseBatchRequest(input: unknown, maxEvents: number): ParsedBatch | { error: string } {
  if (!input || typeof input !== 'object') return { error: 'body must be a JSON object' }
  const body = input as Record<string, unknown>
  if (body.v !== EXPERIENCE_PROTOCOL_VERSION) return { error: 'unsupported protocol version' }
  if (!isSessionId(body.sessionId)) return { error: 'invalid sessionId' }
  if (!Array.isArray(body.events)) return { error: 'events must be an array' }
  if (body.events.length === 0) return { error: 'events must not be empty' }
  if (body.events.length > maxEvents) return { error: `events must not exceed ${maxEvents}` }

  const events: ExperienceEvent[] = []
  for (const raw of body.events) {
    if (!raw || typeof raw !== 'object') return { error: 'event must be an object' }
    const event = raw as Record<string, unknown>
    if (typeof event.type !== 'string' || !EVENT_TYPE.test(event.type)) return { error: 'invalid event type' }
    if (typeof event.ts !== 'number' || !Number.isFinite(event.ts)) return { error: 'invalid event timestamp' }
    events.push(event as unknown as ExperienceEvent)
  }

  const build = typeof body.build === 'string' && body.build.length <= 128 ? body.build : undefined
  return { sessionId: body.sessionId, events, ...(build ? { build } : {}) }
}

function resolveAgentFile(resolved: ResolvedExperienceTelemetry): string | null {
  const candidates: string[] = []
  if (resolved.assetFile) candidates.push(resolved.assetFile)
  try {
    candidates.push(fileURLToPath(new URL('./agent.js', import.meta.url)))
  } catch {
    // ignore
  }
  for (const candidate of candidates) {
    try {
      if (fs.statSync(candidate).isFile()) return candidate
    } catch {
      // try the next candidate
    }
  }
  return null
}

let cachedAgent: { file: string; mtimeMs: number; source: string } | null = null

function readAgent(resolved: ResolvedExperienceTelemetry): string | null {
  const file = resolveAgentFile(resolved)
  if (!file) return null
  try {
    const stat = fs.statSync(file)
    if (!cachedAgent || cachedAgent.file !== file || cachedAgent.mtimeMs !== stat.mtimeMs) {
      cachedAgent = { file, mtimeMs: stat.mtimeMs, source: fs.readFileSync(file, 'utf-8') }
    }
    return cachedAgent.source
  } catch {
    return null
  }
}

export function createExperienceTelemetryRouter(resolved: ResolvedExperienceTelemetry): Router {
  const router = express.Router()

  try {
    pruneSessions(resolved.dataDir, { maxAgeDays: resolved.maxAgeDays, maxSessions: resolved.maxSessions })
  } catch {
    // Retention is best-effort and must never block server start.
  }
  let lastPruneAt = Date.now()

  const parseBody = express.text({
    type: ['application/json', 'text/plain'],
    limit: resolved.maxBodyBytes,
  })

  router.get(resolved.assetPath, (_req: Request, res: Response) => {
    const source = readAgent(resolved)
    if (source === null) {
      res.status(404).type('text/plain').send(
        'Experience agent is not built. Run `pnpm --filter @vrooli/api-base build`.',
      )
      return
    }
    res.setHeader('Content-Type', 'text/javascript; charset=utf-8')
    res.setHeader('Cache-Control', 'no-cache, must-revalidate')
    res.setHeader('X-Content-Type-Options', 'nosniff')
    res.status(200).send(source)
  })

  router.post(resolved.ingestPath, parseBody, (req: Request, res: Response) => {
    const parsed = parseBatchRequest(readPayload(req), resolved.maxEventsPerBatch)
    if ('error' in parsed) {
      res.status(400).json({ error: parsed.error })
      return
    }

    const batch: ExperienceBatch = {
      v: EXPERIENCE_PROTOCOL_VERSION,
      sessionId: parsed.sessionId,
      scenario: resolved.scenario,
      receivedAt: Date.now(),
      ...(parsed.build ? { build: parsed.build } : {}),
      events: parsed.events,
    }

    try {
      appendBatch(resolved.dataDir, batch)
    } catch (error) {
      if (resolved.verbose) {
        console.error('[experience] failed to persist batch', error)
      }
      res.status(500).json({ error: 'failed to persist events' })
      return
    }

    if (Date.now() - lastPruneAt > 60 * 60 * 1000) {
      lastPruneAt = Date.now()
      try {
        pruneSessions(resolved.dataDir, { maxAgeDays: resolved.maxAgeDays, maxSessions: resolved.maxSessions })
      } catch {
        // ignore
      }
    }

    if (resolved.verbose) {
      console.log(`[experience] ${batch.events.length} events (${resolved.scenario}/${batch.sessionId})`)
    }
    res.status(204).end()
  })

  router.get(resolved.statusPath, (_req: Request, res: Response) => {
    const status = sessionStatus(resolved.dataDir)
    res.setHeader('Cache-Control', 'no-store')
    res.status(200).json({
      v: EXPERIENCE_PROTOCOL_VERSION,
      enabled: resolved.enabled,
      configuredBy: resolved.source,
      scenario: resolved.scenario,
      sessionDir: status.sessionDir,
      sessions: status.sessions,
      bytes: status.bytes,
      oldest: status.oldest ? new Date(status.oldest).toISOString() : null,
      newest: status.newest ? new Date(status.newest).toISOString() : null,
      limits: {
        maxAgeDays: resolved.maxAgeDays,
        maxSessions: resolved.maxSessions,
        maxEventsPerSession: resolved.maxEventsPerSession,
        maxBodyBytes: resolved.maxBodyBytes,
      },
    })
  })

  // Express error middleware for body-parser failures (413, bad encoding).
  router.use((error: unknown, _req: Request, res: Response, next: NextFunction) => {
    const status = typeof (error as { status?: number }).status === 'number'
      ? (error as { status: number }).status
      : 500
    if (res.headersSent) {
      next(error)
      return
    }
    res.status(status).json({ error: status === 413 ? 'payload too large' : 'invalid request body' })
  })

  return router
}
