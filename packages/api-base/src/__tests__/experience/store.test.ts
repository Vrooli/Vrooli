import { afterEach, describe, expect, it } from 'vitest'
import * as fs from 'node:fs'
import * as os from 'node:os'
import * as path from 'node:path'
import {
  appendBatch,
  isSessionId,
  listSessionFiles,
  pruneSessions,
  readSessions,
  sessionStatus,
} from '../../experience/store.js'
import type { ExperienceBatch } from '../../experience/types.js'

const dirs: string[] = []

function tempDir(): string {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'experience-store-'))
  dirs.push(dir)
  return dir
}

afterEach(() => {
  while (dirs.length > 0) {
    const dir = dirs.pop()!
    fs.rmSync(dir, { recursive: true, force: true })
  }
})

function batch(sessionId: string, scenario = 'demo', events = 1): ExperienceBatch {
  return {
    v: 1,
    sessionId,
    scenario,
    receivedAt: Date.now(),
    events: Array.from({ length: events }, (_value, index) => ({
      type: 'action',
      ts: Date.now() + index,
      seq: index,
      action: 'click',
      element: { tag: 'button', path: 'button', fp: 'f' },
    })),
  }
}

describe('session ids', () => {
  it('accepts bounded opaque ids and rejects traversal', () => {
    expect(isSessionId('abcdefgh12345678')).toBe(true)
    expect(isSessionId('../etc/passwd')).toBe(false)
    expect(isSessionId('short')).toBe(false)
    expect(isSessionId('a'.repeat(65))).toBe(false)
  })
})

describe('appendBatch', () => {
  it('creates the directory and appends parseable JSONL', () => {
    const dir = path.join(tempDir(), 'nested')
    appendBatch(dir, batch('abcdefgh12345678', 'demo', 2))
    appendBatch(dir, batch('abcdefgh12345678', 'demo', 1))

    const lines = fs.readFileSync(path.join(dir, 'abcdefgh12345678.jsonl'), 'utf-8').trim().split('\n')
    expect(lines).toHaveLength(2)

    const read = readSessions(dir)
    expect(read.files).toBe(1)
    expect(read.batches).toHaveLength(2)
    expect(read.batches[0]!.events).toHaveLength(2)
    expect(read.malformed).toBe(0)
  })

  it('counts malformed lines without failing the read', () => {
    const dir = tempDir()
    appendBatch(dir, batch('abcdefgh12345678'))
    fs.appendFileSync(path.join(dir, 'abcdefgh12345678.jsonl'), '{not json}\n', 'utf-8')
    const read = readSessions(dir)
    expect(read.batches).toHaveLength(1)
    expect(read.malformed).toBe(1)
  })
})

describe('pruneSessions', () => {
  it('removes expired and excess files, keeping the newest', () => {
    const dir = tempDir()
    const now = Date.now()
    for (let index = 0; index < 4; index += 1) {
      const sessionId = `session0000000${index}`
      appendBatch(dir, batch(sessionId))
      const file = path.join(dir, `${sessionId}.jsonl`)
      fs.utimesSync(file, new Date(now - index * 1000), new Date(now - index * 1000))
    }

    const result = pruneSessions(dir, { maxAgeDays: 30, maxSessions: 2, now })
    expect(result.removed).toBe(2)
    expect(listSessionFiles(dir)).toHaveLength(2)

    fs.utimesSync(path.join(dir, 'session00000000.jsonl'), new Date(now - 40 * 24 * 3600 * 1000), new Date(now - 40 * 24 * 3600 * 1000))
    const expired = pruneSessions(dir, { maxAgeDays: 30, maxSessions: 10, now })
    expect(expired.removed).toBe(1)
  })

  it('reports size and bounds in status', () => {
    const dir = tempDir()
    appendBatch(dir, batch('abcdefgh12345678'))
    const status = sessionStatus(dir)
    expect(status.sessions).toBe(1)
    expect(status.bytes).toBeGreaterThan(0)
    expect(status.oldest).not.toBeNull()
  })

  it('tolerates a missing directory', () => {
    expect(sessionStatus(path.join(tempDir(), 'absent')).sessions).toBe(0)
    expect(readSessions(path.join(tempDir(), 'absent')).batches).toEqual([])
  })
})
