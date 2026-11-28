import { tuning, type WorldTuning } from '../../config'
import type { AgentInput, CreateWorldInput, Signal, TeamInput } from '../model'
import { createWorld } from '../world'
import { step } from '../tick'
import { createWorldStore } from '../store'

export const NOW = 1_700_000_000

export interface WorldFixtureOptions extends Omit<Partial<CreateWorldInput>, 'teams' | 'agents'> {
  teams?: number | TeamInput[]
  /** Total agents, not agents per team. Arrays preserve test-specific identities. */
  agents?: number | AgentInput[]
  tuning?: WorldTuning
  treeVariants?: number
}

/** Shared input construction for state and store fixtures. */
export function makeWorldInput({ teams = 2, agents = 6, seed = 7, now = NOW, scene = 'park', tuning: _tuning, treeVariants: _treeVariants, ...extra }: WorldFixtureOptions = {}): CreateWorldInput {
  for (const value of [teams, agents]) {
    if (typeof value === 'number' && (!Number.isInteger(value) || value < 0)) throw new Error('Fixture roster counts must be nonnegative integers')
  }
  const teamCount = typeof teams === 'number' ? teams : teams.length
  const generatedTeams: TeamInput[] = []
  const generatedAgents: AgentInput[] = []
  const agentCount = typeof agents === 'number' ? agents : agents.length
  let cursor = 0
  for (let team = 0; team < Math.max(1, teamCount); team += 1) {
    const count = Math.floor(agentCount / Math.max(1, teamCount)) + (team < agentCount % Math.max(1, teamCount) ? 1 : 0)
    const memberIds: string[] = []
    for (let member = 0; member < count; member += 1) {
      const agent = typeof agents === 'number' ? { id: `agent-${team}-${member}`, name: `Agent ${team}.${member}`, skillCount: cursor } : agents[cursor]
      if (!agent) throw new Error('Fixture roster length mismatch')
      generatedAgents.push(agent)
      memberIds.push(agent.id)
      cursor += 1
    }
    if (teamCount > 0) generatedTeams.push({ id: `team-${team}`, name: `Team ${team}`, memberIds })
  }
  return { seed, now, scene, teams: typeof teams === 'number' ? generatedTeams : teams, agents: generatedAgents, ...extra }
}

/** Fully generated world; specify only the properties the test exercises. */
export function makeWorld(options: WorldFixtureOptions = {}) {
  return createWorld(makeWorldInput(options), options.tuning ?? tuning, options.treeVariants ?? 0)
}

export function makeWorldStore(options: WorldFixtureOptions = {}) {
  return createWorldStore(makeWorldInput(options), options.tuning ?? tuning, options.treeVariants ?? 0)
}

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
