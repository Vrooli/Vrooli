import { tuning, type WorldTuning } from '../../config'
import type { AgentInput, CreateWorldInput, Signal, TeamInput } from '../model'
import { createWorld } from '../world'
import { step } from '../tick'

export const NOW = 1_700_000_000

export function makeTeams(teamCount: number, membersPerTeam: number): { teams: TeamInput[]; agents: AgentInput[] } {
  const teams: TeamInput[] = []
  const agents: AgentInput[] = []
  for (let t = 0; t < teamCount; t += 1) {
    const memberIds: string[] = []
    for (let m = 0; m < membersPerTeam; m += 1) {
      const id = `agent-${t}-${m}`
      memberIds.push(id)
      agents.push({ id, name: `Agent ${t}.${m}`, skillCount: t * membersPerTeam + m })
    }
    teams.push({ id: `team-${t}`, name: `Team ${t}`, memberIds })
  }
  return { teams, agents }
}

export function makeInput(teamCount = 2, membersPerTeam = 3, extra: Partial<CreateWorldInput> = {}): CreateWorldInput {
  const { teams, agents } = makeTeams(teamCount, membersPerTeam)
  return { seed: 7, now: NOW, teams, agents, scene: 'park', ...extra }
}

export function world(teamCount = 2, membersPerTeam = 3, extra: Partial<CreateWorldInput> = {}, t: WorldTuning = tuning) {
  return createWorld(makeInput(teamCount, membersPerTeam, extra), t, 3)
}

/** Run n ticks with an optional signal script keyed by tick index. */
export function run(state: ReturnType<typeof createWorld>, ticks: number, script: Record<number, Signal[]> = {}, t: WorldTuning = tuning) {
  let s = state
  for (let i = 0; i < ticks; i += 1) s = step(s, t.sim.tickSeconds, script[i] ?? [], t)
  return s
}

/** Tuning with a fixed tick but idle rolls disabled, for transition tests that must not be disturbed by wandering. */
export function quietTuning(base: WorldTuning = tuning): WorldTuning {
  return {
    ...base,
    sim: { ...base.sim, idle: { ...base.sim.idle, weights: { rest: 1, wander: 0, socialize: 0, sit: 0 } } },
  }
}

/** Put an actor on the commons with no seat, mutating a fresh state, so a transition test can start it away from its desk. */
export function awayFromHome(state: ReturnType<typeof createWorld>, id: string): ReturnType<typeof createWorld> {
  const actor = state.actors[id]
  if (!actor) throw new Error(`no actor ${id}`)
  const commons = state.places.gathering
  if (actor.seatId) state.occupancy = Object.fromEntries(Object.entries(state.occupancy).filter(([seat]) => seat !== actor.seatId))
  actor.seatId = undefined
  actor.position = commons ? [commons.position[0] + 1, commons.position[1] + 1] : [1, 1]
  return state
}
