/**
 * WorldEvent (wire) → Signal (sim). Also diffs running-agent snapshots into
 * run.started / run.finished signals so the poll fallback and the stream's
 * opening SNAPSHOT drive the sim through the same door.
 */
import { WorldEventKind, type WorldEvent } from '@vrooli/proto-types/prompt-manager/v1/world/world_pb'
import type { Signal } from '../sim'

export interface KnownRun {
  agentId: string
  teamId: string
  runId: string
  startedAt: number
}

export function parseSeconds(rfc3339: string, fallback: number): number {
  const ms = Date.parse(rfc3339)
  return Number.isFinite(ms) ? ms / 1000 : fallback
}

/** Diff the previously known active runs against a fresh list. */
export function diffRuns(known: readonly KnownRun[], fresh: readonly KnownRun[], now: number): { signals: Signal[]; next: KnownRun[] } {
  const signals: Signal[] = []
  const freshByRun = new Map(fresh.map((r) => [r.runId, r]))
  const knownByRun = new Map(known.map((r) => [r.runId, r]))
  for (const run of known) {
    if (!freshByRun.has(run.runId)) signals.push({ kind: 'run.finished', agentId: run.agentId, runId: run.runId, at: now })
  }
  for (const run of fresh) {
    if (!knownByRun.has(run.runId)) {
      signals.push({ kind: 'run.started', agentId: run.agentId, teamId: run.teamId, runId: run.runId, at: run.startedAt })
    }
  }
  return { signals, next: [...fresh] }
}

export interface AdapterState {
  known: KnownRun[]
  lastSeq: bigint
}

export function initialAdapterState(): AdapterState {
  return { known: [], lastSeq: 0n }
}

/** Map one wire event onto sim signals, updating the adapter's run bookkeeping. */
export function eventToSignals(event: WorldEvent, state: AdapterState, now: number): Signal[] {
  if (event.seq > state.lastSeq) state.lastSeq = event.seq
  const at = parseSeconds(event.at, now)
  switch (event.kind) {
    case WorldEventKind.SNAPSHOT: {
      const fresh = event.activeRuns.map((r) => ({ agentId: r.agentId, teamId: r.teamId, runId: r.runId, startedAt: parseSeconds(r.startedAt, at) }))
      const { signals, next } = diffRuns(state.known, fresh, at)
      state.known = next
      for (const up of event.upcoming) {
        signals.push({ kind: 'heartbeat.upcoming', teamId: up.teamId, scheduledAt: parseSeconds(up.scheduledAt, at), at })
      }
      return signals
    }
    case WorldEventKind.RUN_STARTED: {
      if (!state.known.some((r) => r.runId === event.runId)) {
        state.known.push({ agentId: event.agentId, teamId: event.teamId, runId: event.runId, startedAt: at })
      }
      return [{ kind: 'run.started', agentId: event.agentId, teamId: event.teamId, runId: event.runId, at }]
    }
    case WorldEventKind.RUN_FINISHED:
      state.known = state.known.filter((r) => r.runId !== event.runId)
      return [{ kind: 'run.finished', agentId: event.agentId, runId: event.runId, at }]
    case WorldEventKind.RUN_FAILED:
      state.known = state.known.filter((r) => r.runId !== event.runId)
      return [{ kind: 'run.failed', agentId: event.agentId, runId: event.runId, error: event.message, at }]
    case WorldEventKind.HEARTBEAT_UPCOMING:
      return [{ kind: 'heartbeat.upcoming', teamId: event.teamId, scheduledAt: parseSeconds(event.scheduledAt, at), at }]
    case WorldEventKind.HEARTBEAT_CANCELLED:
      return [{ kind: 'heartbeat.cancelled', teamId: event.teamId, at }]
    case WorldEventKind.AGENT_MESSAGE:
      return [{ kind: 'agent.message', agentId: event.agentId, message: event.message, at }]
    default:
      return []
  }
}
