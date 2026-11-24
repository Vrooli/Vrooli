import { create } from '@bufbuild/protobuf'
import { WorldEventKind, WorldEventSchema, type WorldEvent } from '@vrooli/proto-types/prompt-manager/v1/world/world_pb'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { tuning } from '../../config'
import type { Signal } from '../../sim'
import { backoffMs, startFeed, type FeedStatus } from '../feed'

const data = tuning.data

function snapshot(seq: number): WorldEvent {
  return create(WorldEventSchema, { kind: WorldEventKind.SNAPSHOT, seq: BigInt(seq), at: '2026-09-02T12:00:00Z' })
}

/** A stream that yields the given events, then either ends, throws or hangs until aborted. */
function fakeStream(events: WorldEvent[], mode: 'hang' | 'end' | 'throw' = 'hang') {
  const calls: bigint[] = []
  const api = {
    streamFeed: (sinceSeq: bigint, signal: AbortSignal): AsyncIterable<WorldEvent> => {
      calls.push(sinceSeq)
      return {
        async *[Symbol.asyncIterator]() {
          for (const event of events) yield event
          if (mode === 'throw') throw new Error('stream broke')
          if (mode === 'end') return
          await new Promise<void>((resolve) => signal.addEventListener('abort', () => resolve()))
        },
      }
    },
  }
  return { api, calls }
}

describe('feed', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('backoff doubles from the base and caps at the max', () => {
    expect(backoffMs(0, data)).toBe(data.reconnectBaseMs)
    expect(backoffMs(1, data)).toBe(data.reconnectBaseMs * 2)
    expect(backoffMs(30, data)).toBe(data.reconnectMaxMs)
  })

  it('delivers stream events as signals and reports stream mode', async () => {
    const { api } = fakeStream([snapshot(1)])
    const received: Signal[][] = []
    const statuses: FeedStatus[] = []
    const handle = startFeed({ api, pollRunning: () => Promise.resolve([]), onSignals: (s) => received.push(s), onStatus: (s) => statuses.push(s), tuning: data })
    await vi.advanceTimersByTimeAsync(10)
    expect(handle.status().mode).toBe('stream')
    expect(received.flat()).toEqual([])
    expect(statuses.some((s) => s.mode === 'stream')).toBe(true)
    handle.stop()
    expect(handle.status().mode).toBe('stopped')
  })

  it('starts polling when the stream fails, then resumes the stream', async () => {
    let attempt = 0
    const polls: number[] = []
    const api = {
      streamFeed: (_since: bigint, signal: AbortSignal): AsyncIterable<WorldEvent> => ({
        async *[Symbol.asyncIterator]() {
          attempt += 1
          if (attempt === 1) throw new Error('down')
          yield snapshot(1)
          await new Promise<void>((resolve) => signal.addEventListener('abort', () => resolve()))
        },
      }),
    }
    const received: Signal[][] = []
    const handle = startFeed({
      api,
      pollRunning: () => {
        polls.push(Date.now())
        return Promise.resolve([{ agentId: 'a', teamId: 't', runId: 'r1', startedAt: 0 }])
      },
      onSignals: (s) => received.push(s),
      tuning: data,
    })
    await vi.advanceTimersByTimeAsync(5)
    expect(handle.status().mode).toBe('polling')
    expect(polls).toHaveLength(1)
    expect(received.flat()[0]).toMatchObject({ kind: 'run.started', runId: 'r1' })
    await vi.advanceTimersByTimeAsync(data.reconnectBaseMs + 5)
    expect(handle.status().mode).toBe('stream')
    expect(handle.status().reconnects).toBe(1)
    await vi.advanceTimersByTimeAsync(data.pollIntervalMs * 2)
    expect(polls).toHaveLength(1)
    handle.stop()
  })

  it('a silent stream falls back to polling after fallbackAfterMs', async () => {
    const { api } = fakeStream([])
    const polls: number[] = []
    const handle = startFeed({ api, pollRunning: () => { polls.push(1); return Promise.resolve([]) }, onSignals: () => {}, tuning: data })
    await vi.advanceTimersByTimeAsync(data.fallbackAfterMs - 1)
    expect(handle.status().mode).toBe('connecting')
    expect(polls).toHaveLength(0)
    await vi.advanceTimersByTimeAsync(2)
    expect(handle.status().mode).toBe('polling')
    expect(polls).toHaveLength(1)
    handle.stop()
  })

  it('reconnects with the last seen seq after the stream ends', async () => {
    const { api, calls } = fakeStream([snapshot(7)], 'end')
    const handle = startFeed({ api, pollRunning: () => Promise.resolve([]), onSignals: () => {}, tuning: data })
    await vi.advanceTimersByTimeAsync(5)
    await vi.advanceTimersByTimeAsync(data.reconnectBaseMs + 5)
    expect(calls[0]).toBe(0n)
    expect(calls[1]).toBe(7n)
    handle.stop()
  })
})
