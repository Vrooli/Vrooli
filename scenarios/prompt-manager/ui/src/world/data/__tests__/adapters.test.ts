import { create } from '@bufbuild/protobuf'
import { WorldEventKind, WorldEventSchema } from '@vrooli/proto-types/prompt-manager/v1/world/world_pb'
import { describe, expect, it } from 'vitest'
import { diffRuns, eventToSignals, initialAdapterState, parseSeconds } from '../adapters'
import { overridesFromWire } from '../layout'

const NOW = 1_700_000_000

describe('adapters', () => {
  it('migrates legacy gathering-place override ids on read', () => {
    expect(overridesFromWire({ overrides: [
      { placeId: 'commons', removed: false },
      { placeId: 'campfire', removed: false },
    ] }).map((override) => override.placeId)).toEqual(['gathering', 'hearth'])
  })
  it('parses RFC 3339 and falls back on garbage', () => {
    expect(parseSeconds('2026-09-02T12:00:00Z', 0)).toBe(Date.UTC(2026, 8, 2, 12) / 1000)
    expect(parseSeconds('nope', 42)).toBe(42)
  })

  it('diffRuns emits started for new runs and finished for vanished ones', () => {
    const known = [{ agentId: 'a', teamId: 't', runId: 'r1', startedAt: NOW - 10 }]
    const fresh = [{ agentId: 'b', teamId: 't', runId: 'r2', startedAt: NOW - 5 }]
    const { signals, next } = diffRuns(known, fresh, NOW)
    expect(signals).toEqual([
      { kind: 'run.finished', agentId: 'a', runId: 'r1', at: NOW },
      { kind: 'run.started', agentId: 'b', teamId: 't', runId: 'r2', at: NOW - 5 },
    ])
    expect(next).toEqual(fresh)
    expect(diffRuns(fresh, fresh, NOW).signals).toEqual([])
  })

  it('a SNAPSHOT diffs known runs and announces upcoming heartbeats', () => {
    const state = initialAdapterState()
    state.known = [{ agentId: 'gone', teamId: 't', runId: 'old', startedAt: NOW }]
    const event = create(WorldEventSchema, {
      kind: WorldEventKind.SNAPSHOT,
      seq: 9n,
      at: '2026-09-02T12:00:00Z',
      activeRuns: [{ teamId: 't', agentId: 'a', runId: 'r1', startedAt: '2026-09-02T11:59:00Z' }],
      upcoming: [{ teamId: 't', agentId: 'a', scheduledAt: '2026-09-02T12:30:00Z' }],
    })
    const signals = eventToSignals(event, state, NOW)
    expect(signals.map((s) => s.kind)).toEqual(['run.finished', 'run.started', 'heartbeat.upcoming'])
    expect(state.lastSeq).toBe(9n)
    expect(state.known.map((r) => r.runId)).toEqual(['r1'])
  })

  it('maps every live kind onto its signal and keeps run bookkeeping', () => {
    const state = initialAdapterState()
    const at = '2026-09-02T12:00:00Z'
    const started = eventToSignals(create(WorldEventSchema, { kind: WorldEventKind.RUN_STARTED, seq: 1n, at, agentId: 'a', teamId: 't', runId: 'r' }), state, NOW)
    expect(started[0]?.kind).toBe('run.started')
    expect(state.known).toHaveLength(1)
    const failed = eventToSignals(create(WorldEventSchema, { kind: WorldEventKind.RUN_FAILED, seq: 2n, at, agentId: 'a', runId: 'r', message: 'boom' }), state, NOW)
    expect(failed[0]).toMatchObject({ kind: 'run.failed', error: 'boom' })
    expect(state.known).toHaveLength(0)
    const cancelled = eventToSignals(create(WorldEventSchema, { kind: WorldEventKind.HEARTBEAT_CANCELLED, seq: 3n, at, teamId: 't' }), state, NOW)
    expect(cancelled[0]?.kind).toBe('heartbeat.cancelled')
    const message = eventToSignals(create(WorldEventSchema, { kind: WorldEventKind.AGENT_MESSAGE, seq: 4n, at, agentId: 'a', message: 'hi' }), state, NOW)
    expect(message[0]).toMatchObject({ kind: 'agent.message', message: 'hi' })
    expect(eventToSignals(create(WorldEventSchema, { kind: WorldEventKind.UNSPECIFIED, seq: 5n, at }), state, NOW)).toEqual([])
    expect(state.lastSeq).toBe(5n)
  })
})
