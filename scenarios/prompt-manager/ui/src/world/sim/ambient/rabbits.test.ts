import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { ambientPolicy } from '../../config/ambient'
import { makeWorldInput } from '../__tests__/fixtures'
import { runCooperatively } from '../cooperative'
import { isWalkable } from '../nav/grid'
import { generateWorld } from '../world'
import { nextRabbitBoundary, rabbitPose, rabbitRouteSteps, type RabbitRoute } from './rabbits'

const field = { cols: 41, rows: 41, originX: 0, originZ: 0, cellSize: 1, radius: 100,
  height: new Float32Array(1681).fill(2), moisture: new Float32Array(1681) }
const habitat = new Uint8Array(1681).fill(1)
const nav = { cols: 40, rows: 40, originX: 0, originZ: 0, cellSize: 1, walkable: new Uint8Array(1600).fill(1) }

describe('terrestrial rabbit routes', () => {
  it('has long idle periods, short reversible travel, and replayable boundary wakes', () => {
    const route: RabbitRoute = { id: 'fixture', start: [5, 5], end: [7, 5], phase: 0, rank: 0 }
    for (const seconds of [0, 5, 17.99, 22, 30, 39.99]) expect(rabbitPose(route, field, seconds).moving).toBe(false)
    expect(rabbitPose(route, field, 0).position).toEqual([5, 2, 5])
    expect(rabbitPose(route, field, 20).position).toEqual([6, 2, 5])
    expect(rabbitPose(route, field, 22).position).toEqual([7, 2, 5])
    expect(rabbitPose(route, field, 42).position).toEqual([6, 2, 5])
    expect(rabbitPose(route, field, 44)).toEqual(rabbitPose(route, field, 0))
    expect([0, 18, 22, 40, -1].map(now => nextRabbitBoundary(route, now))).toEqual([18, 22, 40, 44, 0])
    for (const now of [-1e9, -50.5, 19.1, 1e9]) expect(nextRabbitBoundary(route, now)).toBeGreaterThan(now)
    for (let index = 0; index < 1000; index++) {
      const shifted = { ...route, phase: index / 1000 }
      const boundary = 1e12 - shifted.phase * 44 + 18
      expect(nextRabbitBoundary(shifted, boundary)).toBeGreaterThan(boundary)
    }
  })
  it('selects bounded deterministic routes independently of agent occupancy', async () => {
    const first = await runCooperatively(rabbitRouteSteps(1, field, habitat, nav))
    expect(first).toHaveLength(2)
    expect(await runCooperatively(rabbitRouteSteps(1, field, habitat, nav))).toEqual(first)
    expect(await runCooperatively(rabbitRouteSteps(2, field, habitat, nav))).not.toEqual(first)
    expect(nav.walkable.every(value => value === 1)).toBe(true)
  })
  it('rejects nonterrestrial habitat and blocked navigation and cancels unpublished work', async () => {
    expect(await runCooperatively(rabbitRouteSteps(1, field, new Uint8Array(1681).fill(5), nav))).toEqual([])
    expect(await runCooperatively(rabbitRouteSteps(1, field, habitat, { ...nav, walkable: new Uint8Array(1600) }))).toEqual([])
    const controller = new AbortController()
    await expect(runCooperatively(rabbitRouteSteps(1, field, habitat, nav), {
      signal: controller.signal, yieldTask: () => { controller.abort(); return Promise.resolve() },
    })).rejects.toThrow()
  })
  it.each(['park', 'office'] as const)('keeps the whole animal footprint on valid %s ground', async scene => {
    const world = generateWorld(makeWorldInput({ seed: 1, scene, teams: 5, agents: 25 }), tuning)
    const original = world.nav.walkable.slice()
    const routes = await runCooperatively(rabbitRouteSteps(1, world.terrain, world.habitats, world.nav))
    expect(routes.length).toBeGreaterThan(0)
    const radius = ambientPolicy.rabbits.radius
    for (const route of routes) for (let seconds = 0; seconds < 44; seconds += .125) {
      const pose = rabbitPose(route, world.terrain, seconds)
      for (const dx of [-radius, 0, radius]) for (const dz of [-radius, 0, radius]) {
        const x = pose.position[0] + dx, z = pose.position[2] + dz
        expect(isWalkable(world.nav, [x, z])).toBe(true)
        const col = Math.round((x - world.terrain.originX) / world.terrain.cellSize)
        const row = Math.round((z - world.terrain.originZ) / world.terrain.cellSize)
        expect([1, 2]).toContain(world.habitats[row * world.terrain.cols + col])
        for (const place of Object.values(world.places)) {
          const px = x - place.position[0], pz = z - place.position[1]
          const cos = Math.cos(place.rotation), sin = Math.sin(place.rotation)
          expect(Math.abs(px * cos - pz * sin) > place.size[0] / 2 || Math.abs(px * sin + pz * cos) > place.size[1] / 2).toBe(true)
        }
      }
    }
    expect(world.nav.walkable).toEqual(original)
  })
})
