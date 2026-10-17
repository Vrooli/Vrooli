import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { makeWorldInput } from '../__tests__/fixtures'
import { runCooperatively } from '../cooperative'
import { generateWorld } from '../world'
import type { WaterGeometryData } from '../terrain/waterSurface'
import { fishCandidate, fishEventsAt, fishPondSteps, fishPose, nextFishBoundary } from './fish'

const field = { cols: 21, rows: 21, originX: -5, originZ: -5, cellSize: 1, radius: 100,
  height: new Float32Array(441), moisture: new Float32Array(441) }
const habitats = new Uint8Array(441).fill(5)
function pond(width: number, depth: number, component = 0): WaterGeometryData {
  return { component, positions: new Float32Array([0,1,0, width,1,0, width,1,depth, 0,1,depth]),
    normals: new Float32Array(12), shore: new Float32Array(4), indices: new Uint32Array([0,1,2, 0,2,3]), sphere: { center: [0,1,0], radius: 20 } }
}

describe('fish pond authority and schedule', () => {
  it('uses exact connected rendered area and does not combine separate undersized ponds', async () => {
    expect(await runCooperatively(fishPondSteps([pond(3,3), pond(3,3,1)], field, habitats))).toEqual([])
    const selected = await runCooperatively(fishPondSteps([pond(4,3)], field, habitats))
    expect(selected).toHaveLength(1); expect(selected[0]?.area).toBe(12)
    expect(await runCooperatively(fishPondSteps([pond(4,3)], field, new Uint8Array(441)))).toEqual([])
  })
  it('keeps complete site discs inside their actual water triangle', async () => {
    const surface = pond(4,3)
    const ponds = await runCooperatively(fishPondSteps([surface], field, habitats))
    for (const site of ponds[0]?.sites ?? []) {
      const vertices = [...surface.indices.slice(site.triangle * 3, site.triangle * 3 + 3)].map(index => [surface.positions[index * 3] ?? 0, surface.positions[index * 3 + 2] ?? 0])
      for (let angle = 0; angle < Math.PI * 2; angle += .03) {
        const x = site.position[0] + site.radius * Math.cos(angle), z = site.position[2] + site.radius * Math.sin(angle)
        const signs = vertices.map((a, i) => { const b = vertices[(i + 1) % 3] ?? a; return ((b[0] ?? 0) - (a[0] ?? 0)) * (z - (a[1] ?? 0)) - ((b[1] ?? 0) - (a[1] ?? 0)) * (x - (a[0] ?? 0)) })
        expect(signs.every(value => value >= 0) || signs.every(value => value <= 0)).toBe(true)
      }
    }
  })
  it('has one deterministic opportunity per 90 seconds without historical replay', async () => {
    const ponds = await runCooperatively(fishPondSteps([pond(4,3)], field, habitats))
    let previous = 0, totalSpacing = 0
    for (let bucket = 0; bucket < 10000; bucket++) {
      const event = fishCandidate(1, bucket, ponds)
      if (!event) throw new Error('Missing event')
      expect(event.start).toBeGreaterThanOrEqual(bucket * 90)
      expect(event.start).toBeLessThan((bucket + 1) * 90)
      if (bucket) totalSpacing += event.start - previous
      previous = event.start
      expect(fishEventsAt(1, event.start, ponds).map(value => value.id)).toContain(event.id)
      expect(fishEventsAt(1, event.end, ponds).map(value => value.id)).not.toContain(event.id)
      expect(nextFishBoundary(1, event.start, ponds)).toBeGreaterThan(event.start)
    }
    expect(totalSpacing / 9999).toBeCloseTo(90, 1)
    const event = fishCandidate(1, 3, ponds)
    expect(fishCandidate(1, 3, ponds)).toEqual(event)
    expect(fishCandidate(2, 3, ponds)).not.toEqual(event)
    expect(fishEventsAt(1, 1000000000, ponds).every(event => event.start <= 1000000000 && event.end > 1000000000)).toBe(true)
    expect(nextFishBoundary(1, 0, [])).toBeNull()
  })
  it('points the nose upward on ascent and downward on descent, then expands a bounded ripple', async () => {
    const ponds = await runCooperatively(fishPondSteps([pond(4,3)], field, habitats))
    const event = fishCandidate(1, 0, ponds)
    if (!event) throw new Error('Missing event')
    expect(fishPose(event, event.start + .1).pitch).toBeLessThan(0)
    expect(fishPose(event, event.start + .7).pitch).toBeGreaterThan(0)
    expect(fishPose(event, event.start + .4).height).toBeCloseTo(.65)
    expect(fishPose(event, event.start + 1).jumping).toBe(false)
    expect(fishPose(event, event.end).radius).toBeCloseTo(event.site.radius)
    expect(fishPose(event, event.end).opacity).toBeCloseTo(0)
  })
  it.each(['park', 'office'] as const)('selects eligible real %s ponds and loses them after habitat removal', async scene => {
    const world = generateWorld(makeWorldInput({ seed: 1, scene, teams: 5, agents: 25 }), tuning)
    const ponds = await runCooperatively(fishPondSteps(world.waterGeometry, world.terrain, world.habitats))
    expect(ponds.length).toBeGreaterThan(0)
    for (const pond of ponds) {
      expect(pond.area).toBeGreaterThanOrEqual(12)
      expect(pond.sites.length).toBeLessThanOrEqual(8)
      for (const site of pond.sites) for (const place of Object.values(world.places)) {
        const dx = site.position[0] - place.position[0], dz = site.position[2] - place.position[1]
        const cos = Math.cos(place.rotation), sin = Math.sin(place.rotation)
        expect(Math.abs(dx * cos - dz * sin) > place.size[0] / 2 || Math.abs(dx * sin + dz * cos) > place.size[1] / 2).toBe(true)
      }
    }
    expect(await runCooperatively(fishPondSteps(world.waterGeometry, world.terrain, new Uint8Array(world.habitats.length)))).toEqual([])
  })
})
