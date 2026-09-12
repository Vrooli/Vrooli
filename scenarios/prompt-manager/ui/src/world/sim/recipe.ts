import { generationActorTuning, generationLayoutTuning } from '../config/settingImpact'
import type { WorldTuning } from '../config'
import { sortSteps, type WorkProgress } from './cooperative'
import type { CreateWorldInput } from './model'

export const WORLD_GENERATOR_VERSION = 'living-world-v4'

/** Canonical serialization rejects values that JSON would silently discard or coerce. */
export function canonical(value: unknown): string {
  if (value === null || typeof value === 'string' || typeof value === 'boolean') return JSON.stringify(value)
  if (typeof value === 'number' && Number.isFinite(value)) return JSON.stringify(value)
  if (Array.isArray(value)) return `[${value.map(canonical).join(',')}]`
  if (typeof value === 'object') {
    return `{${Object.entries(value).filter(([, item]) => item !== undefined).sort(([a], [b]) => a < b ? -1 : a > b ? 1 : 0).map(([key, item]) => `${JSON.stringify(key)}:${canonical(item)}`).join(',')}}`
  }
  throw new Error('World recipe contains an unsupported or nonfinite value')
}

export function canonicalRoster(input: Pick<CreateWorldInput, 'teams' | 'agents'>) {
  const steps = canonicalRosterSteps(input)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

export function* canonicalRosterSteps(input: Pick<CreateWorldInput, 'teams' | 'agents'>): Generator<WorkProgress, Pick<CreateWorldInput, 'teams' | 'agents'>> {
  const order = (a: { id: string }, b: { id: string }) => a.id < b.id ? -1 : a.id > b.id ? 1 : 0
  for (const records of [input.teams, input.agents]) {
    const ids = new Set<string>()
    for (const [index, record] of records.entries()) {
      if (index % 128 === 0) yield { completed: index, total: records.length }
      if (ids.has(record.id)) throw new Error('World roster contains duplicate IDs')
      ids.add(record.id)
    }
  }
  const teams: CreateWorldInput['teams'] = []
  for (const [index, team] of input.teams.entries()) {
    if (index % 128 === 0) yield { completed: index, total: input.teams.length }
    const seen = new Set<string>()
    const members: string[] = []
    for (const [memberIndex, id] of team.memberIds.entries()) {
      if (memberIndex % 128 === 0) yield { completed: memberIndex, total: team.memberIds.length }
      if (!seen.has(id)) { seen.add(id); members.push(id) }
    }
    const memberIds = yield* sortSteps(members, (a, b) => a < b ? -1 : a > b ? 1 : 0)
    teams.push({ ...team, memberIds })
  }
  return { teams: yield* sortSteps(teams, order), agents: yield* sortSteps(input.agents, order) }
}

/** Serialize an already canonical roster without re-sorting it or building a second topology tree. */
export function* canonicalTopologySteps(roster: Pick<CreateWorldInput, 'teams' | 'agents'>): Generator<WorkProgress, string> {
  let identity = '{"agentIds":['
  for (const [index, agent] of roster.agents.entries()) {
    if (index % 128 === 0) yield { completed: index, total: roster.agents.length }
    identity += `${index ? ',' : ''}${JSON.stringify(agent.id)}`
  }
  identity += '],"teams":['
  for (const [index, team] of roster.teams.entries()) {
    if (index % 128 === 0) yield { completed: index, total: roster.teams.length }
    identity += `${index ? ',' : ''}{"id":${JSON.stringify(team.id)},"memberIds":[`
    for (const [memberIndex, id] of team.memberIds.entries()) {
      if (memberIndex % 128 === 0) yield { completed: memberIndex, total: team.memberIds.length }
      identity += `${memberIndex ? ',' : ''}${JSON.stringify(id)}`
    }
    identity += ']}'
  }
  return identity + ']}'
}

export function structuralRoster(input: Pick<CreateWorldInput, 'teams' | 'agents'>) {
  const roster = canonicalRoster(input)
  return {
    teams: roster.teams.map(team => ({ id: team.id, memberIds: team.memberIds })),
    agentIds: roster.agents.map(agent => agent.id),
  }
}

/** Full canonical identities avoid collisions while the worker contract is introduced. */
export function worldRecipe(input: Omit<CreateWorldInput, 'now'>, tuning: WorldTuning) {
  if (!Number.isInteger(input.seed) || input.seed < 0 || input.seed > 0xffffffff) throw new Error('World seed must be an unsigned 32-bit integer')
  const terrainIdentity = canonical({ version: WORLD_GENERATOR_VERSION, scene: input.scene, seed: input.seed, terrain: tuning.terrain })
  const layoutIdentity = canonical({ terrainIdentity, topology: structuralRoster(input), layout: generationLayoutTuning(tuning.layout), ...generationActorTuning(tuning.actor), overrides: input.overrides ?? [] })
  const dressingIdentity = canonical({ layoutIdentity, clearPoints: input.clearPoints ?? [] })
  return { schemaVersion: 1 as const, generatorVersion: WORLD_GENERATOR_VERSION, terrainIdentity, layoutIdentity, dressingIdentity }
}
