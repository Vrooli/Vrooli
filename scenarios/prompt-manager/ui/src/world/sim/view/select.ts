/**
 * The read model for the HUD. Memoised on revision so React work happens only
 * when something discrete changed; the scene reads state directly per frame.
 */
import type { ActorTuning } from '../../config'
import type { Actor, ActorState, Place, WorldEvent, WorldState } from '../model'

export interface SummaryView {
  total: number
  running: number
  walking: number
  gathered: number
  idle: number
  failed: number
  socializing: number
  nextHeartbeat?: { teamId: string; scheduledAt: number }
}

export interface ActorView {
  id: string
  name: string
  teamId?: string
  state: ActorState
  stateSince: number
  runId?: string
  lastRun?: Actor['lastRun']
  failedError?: string
  skillCount: number
  equipmentTier: number
  colors: Actor['colors']
  message?: Actor['message']
  /** Higher renders first when labels collide: focused > failed > working > gathered > idle. */
  labelPriority: number
  seated: boolean
}

export interface TeamView {
  id: string
  roomId: string
  label: string
  memberIds: string[]
  states: Record<ActorState, number>
}

export interface WorldView {
  revision: number
  time: number
  summary: SummaryView
  actors: ActorView[]
  teams: TeamView[]
  places: Place[]
  events: WorldEvent[]
  bounds: WorldState['bounds']
}

const PRIORITY: Record<ActorState, number> = {
  failed: 5,
  working: 4,
  walkingToDesk: 4,
  gathered: 3,
  walkingToTable: 3,
  socializing: 2,
  idle: 1,
}

export function equipmentTier(skillCount: number, tiers: readonly number[]): number {
  let tier = 0
  for (let i = 0; i < tiers.length; i += 1) if (skillCount >= (tiers[i] ?? Infinity)) tier = i
  return tier
}

function emptyStates(): Record<ActorState, number> {
  return { idle: 0, walkingToDesk: 0, working: 0, failed: 0, walkingToTable: 0, gathered: 0, socializing: 0 }
}

export function buildView(state: WorldState, actor: ActorTuning): WorldView {
  const summary: SummaryView = { total: 0, running: 0, walking: 0, gathered: 0, idle: 0, failed: 0, socializing: 0 }
  const actors: ActorView[] = []
  const teamMap = new Map<string, TeamView>()
  for (const id of state.placeOrder) {
    const place = state.places[id]
    if (place && place.kind === 'room' && place.teamId) {
      teamMap.set(place.teamId, { id: place.teamId, roomId: place.id, label: place.label, memberIds: [], states: emptyStates() })
    }
  }
  for (const id of state.actorOrder) {
    const a = state.actors[id]
    if (!a) continue
    summary.total += 1
    switch (a.state) {
      case 'working':
      case 'walkingToDesk':
        summary.running += 1
        break
      case 'gathered':
      case 'walkingToTable':
        summary.gathered += 1
        break
      case 'failed':
        summary.failed += 1
        break
      case 'socializing':
        summary.socializing += 1
        break
      default:
        summary.idle += 1
    }
    if (a.path.length > 0) summary.walking += 1
    actors.push({
      id: a.id,
      name: a.name,
      teamId: a.teamId,
      state: a.state,
      stateSince: a.stateSince,
      runId: a.runId,
      lastRun: a.lastRun,
      failedError: a.failedError,
      skillCount: a.skillCount,
      equipmentTier: equipmentTier(a.skillCount, actor.equipmentTiers),
      colors: a.colors,
      message: a.message,
      labelPriority: PRIORITY[a.state],
      seated: a.anim.seated,
    })
    if (a.teamId) {
      const team = teamMap.get(a.teamId)
      if (team) {
        team.memberIds.push(a.id)
        team.states[a.state] += 1
      }
    }
  }
  let nextHeartbeat: SummaryView['nextHeartbeat']
  for (const g of Object.values(state.gatherings)) {
    if (g.until <= state.time) continue
    if (!nextHeartbeat || g.scheduledAt < nextHeartbeat.scheduledAt) nextHeartbeat = { teamId: g.teamId, scheduledAt: g.scheduledAt }
  }
  summary.nextHeartbeat = nextHeartbeat
  return {
    revision: state.revision,
    time: state.time,
    summary,
    actors,
    teams: [...teamMap.values()],
    places: state.placeOrder.map((id) => state.places[id]).filter((p): p is Place => p !== undefined),
    events: [...state.events].reverse(),
    bounds: state.bounds,
  }
}

/** Memoising selector: same revision and gathering clock → same object. */
export function createViewSelector(actor: ActorTuning): (state: WorldState) => WorldView {
  let lastRevision = -1
  let lastView: WorldView | null = null
  return (state) => {
    if (lastView && lastRevision === state.revision) return lastView
    lastView = buildView(state, actor)
    lastRevision = state.revision
    return lastView
  }
}
