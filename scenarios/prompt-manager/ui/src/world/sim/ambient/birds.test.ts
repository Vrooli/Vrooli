import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { ambientPolicy } from '../../config/ambient'
import { makeWorldInput } from '../__tests__/fixtures'
import { runCooperatively } from '../cooperative'
import { generateWorld } from '../world'
import { birdPose, birdRouteSteps } from './birds'

const field = { cols: 80, rows: 80, originX: -40, originZ: -40, cellSize: 1, radius: 100,
  height: new Float32Array(6400).fill(7), moisture: new Float32Array(6400) }
const habitats = new Uint8Array(6400).fill(1)

describe('coarse bird habitat routes', () => {
  it('selects sparse deterministic separated loops and changes them with the seed', async () => {
    const routes = await runCooperatively(birdRouteSteps(1, field, habitats))
    expect(routes).toHaveLength(4)
    expect(await runCooperatively(birdRouteSteps(1, field, habitats))).toEqual(routes)
    expect(await runCooperatively(birdRouteSteps(2, field, habitats))).not.toEqual(routes)
    expect(new Set(routes.map(route => route.phase)).size).toBe(routes.length)
    for (const a of routes) for (const b of routes) if (a !== b) expect(Math.hypot(a.x - b.x, a.z - b.z)).toBeGreaterThanOrEqual(a.radius * 3)
    expect(routes.map(route => route.rank)).toEqual(routes.map(route => route.rank).sort((a, b) => a - b))
  })
  it('rejects protected gaps, water and habitat patches too small for a whole loop', async () => {
    expect(await runCooperatively(birdRouteSteps(1, field, new Uint8Array(6400).fill(5)))).toEqual([])
    const narrow = new Uint8Array(6400)
    for (let row = 20; row < 30; row++) for (let col = 20; col < 30; col++) narrow[row * 80 + col] = 2
    expect(await runCooperatively(birdRouteSteps(1, field, narrow))).toEqual([])
    const striped = habitats.slice()
    for (let row = 0; row < 80; row += 8) striped.fill(0, row * 80, (row + 1) * 80)
    expect(await runCooperatively(birdRouteSteps(1, field, striped))).toEqual([])
  })
  it('replays bounded airborne loops without touching generated terrain', async () => {
    const [route] = await runCooperatively(birdRouteSteps(1, field, habitats))
    if (!route) throw new Error('Missing route')
    for (let frame = 0; frame < 32 * 60; frame++) {
      const time = 1700000000 + frame / 60
      const pose = birdPose(route, time)
      expect(Math.hypot(pose.position[0] - route.x, pose.position[2] - route.z)).toBeCloseTo(route.radius)
      expect(pose.position[1]).toBeGreaterThanOrEqual(7 + ambientPolicy.birds.clearance - .5)
      expect(pose.position[1]).toBeLessThanOrEqual(7 + ambientPolicy.birds.clearance + .5)
      expect(birdPose(route, time)).toEqual(pose)
    }
    expect(field.height.every(height => height === 7)).toBe(true)
  })
  it('cancels selection before publishing routes', async () => {
    const controller = new AbortController()
    await expect(runCooperatively(birdRouteSteps(1, field, habitats), {
      signal: controller.signal, yieldTask: () => { controller.abort(); return Promise.resolve() },
    })).rejects.toThrow()
  })
  it.each(['park', 'office'] as const)('keeps real %s routes over allowed habitats', async scene => {
    const world = generateWorld(makeWorldInput({ seed: 1, scene, teams: 5, agents: 25 }), tuning)
    const routes = await runCooperatively(birdRouteSteps(1, world.terrain, world.habitats))
    expect(routes.length).toBeGreaterThan(0)
    for (const route of routes) for (let time = 0; time < 32; time += .25) {
      const pose = birdPose(route, time)
      const col = Math.round((pose.position[0] - world.terrain.originX) / world.terrain.cellSize)
      const row = Math.round((pose.position[2] - world.terrain.originZ) / world.terrain.cellSize)
      expect([1, 2]).toContain(world.habitats[row * world.terrain.cols + col])
      for (const place of Object.values(world.places)) {
        const dx = pose.position[0] - place.position[0], dz = pose.position[2] - place.position[1]
        const cos = Math.cos(place.rotation), sin = Math.sin(place.rotation)
        expect(Math.abs(dx * cos - dz * sin) > place.size[0] / 2 || Math.abs(dx * sin + dz * cos) > place.size[1] / 2).toBe(true)
      }
    }
  })
})
