import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { makeWorldInput } from '../__tests__/fixtures'
import { runCooperatively } from '../cooperative'
import type { DecorSpot } from '../model'
import { generateWorld } from '../world'
import { butterflyPose, butterflySiteSteps } from './butterflies'

const field = { cols: 25, rows: 25, originX: 0, originZ: 0, cellSize: 1, radius: 100,
  height: new Float32Array(625).fill(2), moisture: new Float32Array(625) }
const meadow = new Uint8Array(625).fill(1)
const spot = (id: string, x: number, z: number): DecorSpot => ({ id, kind: 'ground', scaleRef: 'prop', variant: 0,
  position: [x, z], rotation: 0, scale: 1 })

describe('meadow butterflies', () => {
  it('requires vegetation, meadow surroundings and no protected-room decoration', async () => {
    const select = (habitats: Uint8Array, decor: DecorSpot[]) => runCooperatively(butterflySiteSteps(1, field, habitats, decor))
    expect(await select(meadow, [])).toEqual([])
    expect(await select(meadow, [{ ...spot('indoor', 10, 10), roomId: 'office' }])).toEqual([])
    expect(await select(meadow, [{ ...spot('lamp', 10, 10), kind: 'lamp' }])).toEqual([])
    expect(await select(meadow, [spot('edge', 0, 10)])).toEqual([])
    for (const code of [0, 2, 3, 4, 5]) {
      const habitats = meadow.slice(); habitats[9 * 25 + 9] = code
      expect(await select(habitats, [spot('plant', 10, 10)])).toEqual([])
    }
    expect(await select(meadow, [spot('plant', 10.2, 10.3)])).toHaveLength(1)
  })
  it('bounds and deduplicates sites independently of input ordering and quality', async () => {
    const decor = Array.from({ length: 400 }, (_, i) => spot(`plant-${i}`, i % 20 + 2, Math.floor(i / 20) + 2))
    decor.push(spot('duplicate', 2, 2))
    const first = await runCooperatively(butterflySiteSteps(3, field, meadow, decor))
    expect(first).toHaveLength(16)
    expect(new Set(first.map(site => site.sample)).size).toBe(16)
    expect(await runCooperatively(butterflySiteSteps(3, field, meadow, [...decor].reverse()))).toEqual(first)
    expect(await runCooperatively(butterflySiteSteps(4, field, meadow, decor))).not.toEqual(first)
    expect(decor).toHaveLength(401)
  })
  it('keeps the entire flight loop over meadow with bounded clearance and exact replay', async () => {
    const [site] = await runCooperatively(butterflySiteSteps(1, field, meadow, [spot('plant', 10, 10)]))
    if (!site) throw new Error('Missing fixture')
    for (let frame = 0; frame < 24 * 60; frame++) {
      const seconds = 1700000000 + frame / 60
      const pose = butterflyPose(site, field, seconds)
      expect(Math.abs(pose.position[0] - site.x)).toBeLessThanOrEqual(.300001)
      expect(Math.abs(pose.position[2] - site.z)).toBeLessThanOrEqual(.150001)
      expect(pose.position[1]).toBeGreaterThanOrEqual(2.33)
      expect(pose.position[1]).toBeLessThanOrEqual(2.57)
      expect(butterflyPose(site, field, seconds)).toEqual(pose)
    }
  })
  it('cancels unpublished site selection without mutating habitat or vegetation', async () => {
    const controller = new AbortController()
    const decor = [spot('plant', 10, 10)]
    const before = meadow.slice()
    await expect(runCooperatively(butterflySiteSteps(1, field, meadow, decor), {
      signal: controller.signal, yieldTask: () => { controller.abort(); return Promise.resolve() },
    })).rejects.toThrow()
    expect(meadow).toEqual(before)
    expect(decor).toEqual([spot('plant', 10, 10)])
  })
  it.each(['park', 'office'] as const)('selects real %s vegetation without changing generated state', async scene => {
    const world = generateWorld(makeWorldInput({ seed: 1, scene, teams: 5, agents: 25 }), tuning)
    const before = JSON.stringify(world.decor)
    const sites = await runCooperatively(butterflySiteSteps(world.seed, world.terrain, world.habitats, world.decor))
    expect(sites.length).toBeGreaterThan(0)
    for (const site of sites) {
      const plant = world.decor.find(spot => spot.id === site.vegetationId)
      expect(plant).toBeDefined(); expect(plant?.roomId).toBeUndefined()
      expect(world.habitats[site.sample]).toBe(1)
      for (let second = 0; second < 24; second += .5) {
        const pose = butterflyPose(site, world.terrain, second)
        for (const place of Object.values(world.places)) {
          const dx = pose.position[0] - place.position[0], dz = pose.position[2] - place.position[1]
          const cos = Math.cos(place.rotation), sin = Math.sin(place.rotation)
          expect(Math.abs(dx * cos - dz * sin) > place.size[0] / 2 || Math.abs(dx * sin + dz * cos) > place.size[1] / 2).toBe(true)
        }
      }
    }
    expect(JSON.stringify(world.decor)).toBe(before)
    expect(await runCooperatively(butterflySiteSteps(world.seed, world.terrain, new Uint8Array(world.habitats.length), world.decor))).toEqual([])
  })
})
