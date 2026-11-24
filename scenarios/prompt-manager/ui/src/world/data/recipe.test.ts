import { generationLayoutTuning, layoutSurfaceImpacts } from '../config/settingImpact'
import { describe, expect, it } from 'vitest'
import { tuning, WorldTuningSchema } from '../config'
import { collectLevers } from '../config/tuningDocs'
import { makeWorldInput, makeWorldStore } from '../sim/__tests__/fixtures'
import { canonical, canonicalRoster, canonicalRosterSteps, canonicalTopologySteps, structuralRoster, worldRecipe } from '../sim/recipe'
import { runCooperatively } from '../sim/cooperative'

// Generated case inventory; expected clearance ownership is independently fixed here.
const impactCases = collectLevers().filter(row => row.path.startsWith('actor.') || row.path.startsWith('layout.') || row.path.startsWith('sim.') || row.path.startsWith('labels.') || row.path.startsWith('camera.') && !row.readOnly)
function differentValue(value: unknown): unknown {
  if (typeof value === 'boolean') return !value
  if (typeof value === 'number') return value === 0 ? 1 : Number.isInteger(value) ? value + 1 : value * .9
  if (typeof value === 'string') return value.startsWith('rgba') ? 'rgba(255,0,0,0.5)' : value === '#abcdef' ? '#123456' : '#abcdef'
  if (Array.isArray(value)) return value.map((item, index) => index === 0 ? differentValue(item) : item)
  if (typeof value === 'object' && value !== null) {
    const entries = Object.entries(value)
    return Object.fromEntries(entries.map(([key, item], index) => [key, index === 0 ? differentValue(item) : item]))
  }
  throw new Error('Unsupported setting case')
}

describe('world generation identity', () => {
  it.each(impactCases)('routes $path through its declared generation impact', row => {
    expect(row.impact).toBeDefined()
    const next = structuredClone(tuning)
    const keys = row.path.split('.')
    let parent = next as unknown as Record<string, unknown>
    for (const key of keys.slice(0, -1)) parent = parent[key] as Record<string, unknown>
    const key = keys[keys.length - 1]
    if (!key) throw new Error('Missing setting key')
    const original = parent[key]
    parent[key] = row.choices ? row.choices.find(choice => choice !== original) : differentValue(original)
    if (typeof original === 'number') {
      for (const candidate of [parent[key], original - 1, original * .9, original + .01]) {
        parent[key] = candidate
        if (candidate !== original && WorldTuningSchema.safeParse(next).success) break
      }
    }
    expect(parent[key]).not.toEqual(original)
    expect(WorldTuningSchema.safeParse(next).success, row.path).toBe(true)
    const input = makeWorldInput()
    const before = worldRecipe(input, tuning), after = worldRecipe(input, next)
    expect(after.terrainIdentity).toBe(before.terrainIdentity)
    const rendererLayout = row.path.startsWith('layout.surfaces.') || ['layout.wallHeight', 'layout.lampInsetRatio', 'layout.corridorLampSpacing', 'layout.corridorLampScale'].includes(row.path)
    if (row.path === 'actor.bodyRadius' || row.path.startsWith('layout.') && !rendererLayout) {
      expect(row.impact).toBe('world')
      expect(after.layoutIdentity).not.toBe(before.layoutIdentity)
      expect(after.dressingIdentity).not.toBe(before.dressingIdentity)
    } else {
      expect(row.impact).not.toBe('world')
      expect(after).toEqual(before)
    }
  })
  it('preserves the existing body-clearance recipe format', () => {
    const input = makeWorldInput()
    const recipe = worldRecipe(input, tuning)
    expect(recipe.layoutIdentity).toBe(canonical({
      terrainIdentity: recipe.terrainIdentity, topology: structuralRoster(input),
      layout: generationLayoutTuning(tuning.layout), bodyRadius: tuning.actor.bodyRadius, overrides: input.overrides ?? [],
    }))
  })
  it('preserves exact canonical topology bytes including escaped and Unicode IDs', async () => {
    const ids = ['z', 'a"b', 'line\nfeed', '\\', 'é', '😀']
    const input = { agents: ids.map(id => ({ id, name: id })), teams: [{ id: 'team"', name: '', memberIds: [...ids, ...ids] }] }
    const before = structuredClone(input)
    const roster = await runCooperatively(canonicalRosterSteps(input))
    expect(roster.agents.map(agent => agent.id)).toEqual([...ids].sort())
    expect(roster.teams[0]?.memberIds).toEqual([...ids].sort())
    expect(await runCooperatively(canonicalTopologySteps(roster))).toBe(canonical(structuralRoster(input)))
    expect(input).toEqual(before)
  })

  it('cancels a large roster scan while retaining the caller input', async () => {
    const input = makeWorldInput({ teams: 2, agents: 512 })
    const before = structuredClone(input)
    const steps = canonicalRosterSteps(input)
    const controller = new AbortController()
    await expect(runCooperatively(steps, {
      signal: controller.signal,
      onProgress: event => { if (event.completed === 128) controller.abort(new Error('cancel roster')) },
    })).rejects.toThrow('cancel roster')
    expect(input).toEqual(before)
    expect(steps.next().done).toBe(true)
  })

  it('ignores cosmetic data and semantic roster ordering', () => {
    const input = makeWorldInput()
    const cosmetic = {
      ...input,
      now: input.now + 100,
      teams: [...input.teams].reverse().map(team => ({ ...team, name: 'Renamed', memberIds: [...team.memberIds].reverse() })),
      agents: [...input.agents].reverse().map(agent => ({ ...agent, name: 'Display', colors: { body: '#ffffff' }, skillCount: 99 })),
    }
    expect(canonicalRoster(cosmetic).teams.map(team => team.memberIds)).toEqual(canonicalRoster(input).teams.map(team => team.memberIds))
    expect(worldRecipe(cosmetic, tuning)).toEqual(worldRecipe(input, tuning))
    expect(canonical({ b: 2, a: 1 })).toBe(canonical({ a: 1, b: 2 }))
  })
  it('separates live tuning, membership, layout, and terrain changes', () => {
    const input = makeWorldInput()
    const before = worldRecipe(input, tuning)
    const live = worldRecipe(input, { ...tuning, camera: { ...tuning.camera, fov: 65 } })
    expect(live).toEqual(before)
    const membership = worldRecipe({ ...input, teams: [] }, tuning)
    expect(membership.terrainIdentity).toBe(before.terrainIdentity)
    expect(membership.layoutIdentity).not.toBe(before.layoutIdentity)
    expect(worldRecipe({ ...input, seed: input.seed + 1 }, tuning).terrainIdentity).not.toBe(before.terrainIdentity)
  })
  it.each([-1, 0x100000000, NaN, Infinity, 0.5])('rejects invalid seed %s before generation', seed => {
    expect(() => worldRecipe({ ...makeWorldInput(), seed }, tuning)).toThrow(/seed/)
  })
  it.each(Object.keys(layoutSurfaceImpacts) as Array<keyof typeof layoutSurfaceImpacts>)('does not regenerate for renderer-owned surface %s', key => {
    const input = makeWorldInput()
    const next = { ...tuning, layout: { ...tuning.layout, surfaces: { ...tuning.layout.surfaces, [key]: key === 'commonsSegments' ? tuning.layout.surfaces[key] + 1 : tuning.layout.surfaces[key] === 0 ? .01 : tuning.layout.surfaces[key] * .9 } } }
    expect(worldRecipe(input, next)).toEqual(worldRecipe(input, tuning))
  })
  it('retains regeneration for structural layout edits', () => {
    const input = makeWorldInput()
    const next = { ...tuning, layout: { ...tuning.layout, roomWidth: tuning.layout.roomWidth + 1 } }
    expect(worldRecipe(input, next).layoutIdentity).not.toBe(worldRecipe(input, tuning).layoutIdentity)
  })
  it('updates presentation without replacing generated buffers or resetting live state', () => {
    const input = makeWorldInput()
    const store = makeWorldStore()
    const agent = input.agents[0]
    if (!agent) throw new Error('Missing actor fixture')
    store.dispatch([{ kind: 'run.started', agentId: agent.id, runId: 'active-run', at: input.now }])
    store.advance(tuning.sim.tickSeconds)
    const before = store.getState()
    const actor = before.actors[agent.id]
    const updated = input.agents.map(item => ({ ...item, name: `New ${item.name}`, colors: { body: '#aabbcc' }, skillCount: 10 }))
    store.updatePresentation(input.teams, updated)
    const after = store.getState()
    expect(after.terrain).toBe(before.terrain)
    expect(after.nav).toBe(before.nav)
    expect(after.decor).toBe(before.decor)
    expect(after.actors[agent.id]).toMatchObject({ name: `New ${agent.name}`, runId: actor?.runId, state: actor?.state, position: actor?.position, skillCount: 10 })
    expect(after.actors[agent.id]?.colors.body).toBe('#aabbcc')
    const revision = after.revision
    store.updatePresentation(input.teams, updated)
    expect(store.getState().revision).toBe(revision)
  })
})
