import { beforeAll, describe, expect, it } from 'vitest'
import { WILDLIFE_PREVIEWS } from '../config/ambient'
import { makeWorld } from '../sim/__tests__/fixtures'
import { fishEventsAt, fishPondSteps, fishPose } from '../sim/ambient/fish'
import { rabbitPose, rabbitRouteSteps } from '../sim/ambient/rabbits'
import { runCooperatively } from '../sim/cooperative'
import type { WorldState } from '../sim/model'
import { prepareWildlifePreview } from './wildlifePreview'

const utc = 1788616800000
let world: WorldState
beforeAll(() => { world = makeWorld({ seed: 1, teams: 5, agents: 25 }) })

describe('normal-schedule wildlife previews', () => {
  it.each(WILDLIFE_PREVIEWS)('prepares $label from committed habitat without changing world state', async ({ id }) => {
    const actors = JSON.stringify(world.actors), occupancy = JSON.stringify(world.occupancy)
    const terrain = world.terrain.height.slice()
    const preview = await prepareWildlifePreview(world, id, utc)
    expect(preview).not.toBeNull()
    expect(preview?.position.every(Number.isFinite)).toBe(true)
    expect(preview?.period).toBe(id === 'fireflies' ? 'night' : 'day')
    expect(Math.abs((preview?.utcMilliseconds ?? 0) - utc)).toBeLessThan(90000)
    expect(world.terrain.height).toEqual(terrain)
    expect(JSON.stringify(world.actors)).toBe(actors)
    expect(JSON.stringify(world.occupancy)).toBe(occupancy)
  })
  it('seeks fish jump, splash and ripple within the actual scheduled event', async () => {
    const ponds = await runCooperatively(fishPondSteps(world.waterGeometry, world.terrain, world.habitats))
    for (const kind of ['fish-jump', 'fish-splash', 'fish-ripple'] as const) {
      const preview = await prepareWildlifePreview(world, kind, utc)
      if (!preview) throw new Error('Missing preview')
      const events = fishEventsAt(world.seed, preview.utcMilliseconds / 1000, ponds)
      expect(events.length).toBeGreaterThan(0)
      const event = events.find(event => event.site.position[0] === preview.position[0] && event.site.position[2] === preview.position[2])
      if (!event) throw new Error('Preview invented a fish site')
      const pose = fishPose(event, preview.utcMilliseconds / 1000)
      expect(pose.jumping).toBe(kind === 'fish-jump')
      if (kind === 'fish-ripple') expect(pose.ripple).toBeCloseTo(.5)
    }
  })
  it('seeks genuine rabbit idle and hopping states', async () => {
    const [route] = await runCooperatively(rabbitRouteSteps(world.seed, world.terrain, world.habitats, world.nav))
    if (!route) throw new Error('Missing route')
    for (const kind of ['rabbit-idle', 'rabbit-travel'] as const) {
      const preview = await prepareWildlifePreview(world, kind, utc)
      if (!preview) throw new Error('Missing preview')
      const pose = rabbitPose(route, world.terrain, preview.utcMilliseconds / 1000)
      expect(pose.moving).toBe(kind === 'rabbit-travel')
      if (pose.moving) expect(pose.hop).toBeCloseTo(.08)
    }
  })
  it('does not invent habitat after regeneration and respects cancellation', async () => {
    const empty = { ...world, habitats: new Uint8Array(world.habitats.length), habitatRegions: { labels: new Uint32Array(world.habitats.length), members: new Uint32Array(), regions: [] } }
    for (const { id } of WILDLIFE_PREVIEWS) expect(await prepareWildlifePreview(empty, id, utc)).toBeNull()
    const controller = new AbortController(); controller.abort()
    await expect(prepareWildlifePreview(world, 'birds', utc, controller.signal)).rejects.toThrow()
  })
})
