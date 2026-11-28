/**
 * Append-only session storage.
 *
 * One JSONL file per browser tab session, under the operator's home directory
 * by default. Writes are synchronous and append-only: a crash loses at most
 * the batch in flight, and there is no database to corrupt. The analyzer in
 * `report.ts` reads the same files back.
 */

import * as fs from 'node:fs'
import * as path from 'node:path'
import { EXPERIENCE_PROTOCOL_VERSION, type ExperienceBatch, type ExperienceEvent } from './types.js'

const SESSION_ID = /^[A-Za-z0-9_-]{8,64}$/

export interface SessionFileInfo {
  name: string
  file: string
  size: number
  mtimeMs: number
}

export interface ReadSessionsResult {
  batches: ExperienceBatch[]
  files: number
  malformed: number
}

export interface PruneResult {
  removed: number
  kept: number
}

export function isSessionId(value: unknown): value is string {
  return typeof value === 'string' && SESSION_ID.test(value)
}

function sessionFileName(sessionId: string): string {
  return `${sessionId}.jsonl`
}

/** Ensure the directory exists and return it. */
export function ensureSessionDir(dataDir: string): string {
  fs.mkdirSync(dataDir, { recursive: true })
  return dataDir
}

/** Append one batch. Callers must validate the session id first. */
export function appendBatch(dataDir: string, batch: ExperienceBatch): string {
  ensureSessionDir(dataDir)
  const file = path.join(dataDir, sessionFileName(batch.sessionId))
  fs.appendFileSync(file, `${JSON.stringify(batch)}\n`, { encoding: 'utf-8' })
  return file
}

export function listSessionFiles(dataDir: string): SessionFileInfo[] {
  let entries: string[]
  try {
    entries = fs.readdirSync(dataDir)
  } catch {
    return []
  }
  const files: SessionFileInfo[] = []
  entries.forEach((entry) => {
    if (!entry.endsWith('.jsonl')) return
    const file = path.join(dataDir, entry)
    try {
      const stat = fs.statSync(file)
      if (!stat.isFile()) return
      files.push({ name: entry, file, size: stat.size, mtimeMs: stat.mtimeMs })
    } catch {
      // A file removed between readdir and stat is not an error.
    }
  })
  files.sort((a, b) => a.mtimeMs - b.mtimeMs)
  return files
}

/** Delete expired and excess session files. Best-effort; never throws. */
export function pruneSessions(
  dataDir: string,
  options: { maxAgeDays: number; maxSessions: number; now?: number },
): PruneResult {
  const now = options.now ?? Date.now()
  const maxAgeMs = Math.max(1, options.maxAgeDays) * 24 * 60 * 60 * 1000
  const files = listSessionFiles(dataDir)
  let removed = 0
  const survivors: SessionFileInfo[] = []

  files.forEach((file) => {
    if (now - file.mtimeMs > maxAgeMs) {
      try {
        fs.unlinkSync(file.file)
        removed += 1
        return
      } catch {
        // Fall through and keep it counted as a survivor.
      }
    }
    survivors.push(file)
  })

  const excess = survivors.length - Math.max(1, options.maxSessions)
  if (excess > 0) {
    survivors.slice(0, excess).forEach((file) => {
      try {
        fs.unlinkSync(file.file)
        removed += 1
      } catch {
        // ignore
      }
    })
  }

  return { removed, kept: survivors.length - Math.max(0, excess) }
}

function parseEvent(value: unknown): ExperienceEvent | null {
  if (!value || typeof value !== 'object') return null
  const event = value as Record<string, unknown>
  if (typeof event.type !== 'string' || event.type.length === 0) return null
  if (typeof event.ts !== 'number' || !Number.isFinite(event.ts)) return null
  return event as unknown as ExperienceEvent
}

function parseBatch(value: unknown): ExperienceBatch | null {
  if (!value || typeof value !== 'object') return null
  const batch = value as Record<string, unknown>
  if (batch.v !== EXPERIENCE_PROTOCOL_VERSION) return null
  if (!isSessionId(batch.sessionId)) return null
  if (typeof batch.scenario !== 'string' || !batch.scenario) return null
  if (typeof batch.receivedAt !== 'number' || !Number.isFinite(batch.receivedAt)) return null
  if (!Array.isArray(batch.events)) return null
  const events: ExperienceEvent[] = []
  for (const raw of batch.events) {
    const event = parseEvent(raw)
    if (event) events.push(event)
  }
  if (events.length === 0) return null
  return {
    v: EXPERIENCE_PROTOCOL_VERSION,
    sessionId: batch.sessionId,
    scenario: batch.scenario,
    receivedAt: batch.receivedAt,
    ...(typeof batch.build === 'string' && batch.build ? { build: batch.build } : {}),
    events,
  }
}

/** Read every parseable batch in a directory. Malformed lines are counted. */
export function readSessions(dataDir: string): ReadSessionsResult {
  const files = listSessionFiles(dataDir)
  const batches: ExperienceBatch[] = []
  let malformed = 0

  files.forEach((file) => {
    let content = ''
    try {
      content = fs.readFileSync(file.file, 'utf-8')
    } catch {
      malformed += 1
      return
    }
    content.split('\n').forEach((line) => {
      if (!line.trim()) return
      try {
        const batch = parseBatch(JSON.parse(line))
        if (batch) batches.push(batch)
        else malformed += 1
      } catch {
        malformed += 1
      }
    })
  })

  batches.sort((a, b) => a.receivedAt - b.receivedAt)
  return { batches, files: files.length, malformed }
}

export interface SessionStatus {
  sessionDir: string
  sessions: number
  bytes: number
  oldest: number | null
  newest: number | null
}

export function sessionStatus(dataDir: string): SessionStatus {
  const files = listSessionFiles(dataDir)
  let bytes = 0
  files.forEach((file) => {
    bytes += file.size
  })
  return {
    sessionDir: dataDir,
    sessions: files.length,
    bytes,
    oldest: files.length > 0 ? files[0]!.mtimeMs : null,
    newest: files.length > 0 ? files[files.length - 1]!.mtimeMs : null,
  }
}
