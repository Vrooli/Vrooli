import { describe, expect, it } from 'vitest'
import { ambientPolicy } from '../../config/ambient'
import { runCooperatively } from '../cooperative'
import { habitatRegionSteps } from '../terrain/habitats'
import { fireflyPose, fireflySiteSteps } from './fireflies'

function fixture(values: number[], cols = values.length) {
  return { habitats: Uint8Array.from(values), field: {
    cols, rows: values.length / cols, originX: -10, originZ: 20, cellSize: 2, radius: 100,
    height: new Float32Array(values.length).fill(3), moisture: new Float32Array(values.length),
  } }
}
async function sites(values: number[], cols?: number, seed = 1) {
  const { field, habitats } = fixture(values, cols)
  const index = await runCooperatively(habitatRegionSteps(field, habitats))
  return runCooperatively(fireflySiteSteps(seed, field, habitats, index))
}

describe('wetland firefly sites', () => {
  it('uses wetland interfaces, excluding interiors, protected gaps and row wrapping', async () => {
    expect((await sites([0, 4, 4, 4, 1])).map(site => site.sample)).toEqual([3])
    expect(await sites([0, 0, 4, 1, 0, 0], 3)).toEqual([])
    expect(await sites([0, 0, 0, 0, 4, 0, 1, 0, 0], 3)).toEqual([])
    const [edge] = await sites([4, 5])
    expect(edge?.position).toEqual([-10, 3, 20])
    expect(edge?.region).toBe(1)
  })
  it('caps dense habitat at 48 with stable profile prefixes and independent seed variation', async () => {
    const values = Array.from({ length: 1000 }, (_, i) => i % 2 ? 1 : 4)
    const first = await sites(values)
    expect(first).toHaveLength(48)
    expect(await sites(values)).toEqual(first)
    expect(await sites(values, undefined, 2)).not.toEqual(first)
    expect(new Set(first.map(site => site.id)).size).toBe(48)
    expect(first.map(site => site.rank)).toEqual(first.map(site => site.rank).sort((a, b) => a - b))
    for (const maximum of Object.values(ambientPolicy.fireflies.maximumVisible)) {
      expect(first.slice(0, maximum).every(site => values[site.sample] === 4)).toBe(true)
    }
  })
  it('retains exact habitat coordinates and bounded height/glow at arbitrary replay times', async () => {
    const [site] = await sites([1, 4])
    if (!site) throw new Error('Missing fixture site')
    for (const seconds of [-1e12, -1, 0, 1, 100.5, 1e12]) {
      const pose = fireflyPose(site, seconds)
      expect(pose.position[0]).toBe(site.position[0])
      expect(pose.position[2]).toBe(site.position[2])
      expect(pose.position[1]).toBeGreaterThanOrEqual(3.3)
      expect(pose.position[1]).toBeLessThanOrEqual(3.9)
      expect(pose.glow).toBeGreaterThanOrEqual(0.15)
      expect(pose.glow).toBeLessThanOrEqual(1)
      expect(fireflyPose(site, seconds)).toEqual(pose)
    }
  })
  it('cancels unpublished work and loses sites when regeneration removes wetlands', async () => {
    const { field, habitats } = fixture([1, 4, 4, 1])
    const before = habitats.slice()
    const index = await runCooperatively(habitatRegionSteps(field, habitats))
    const controller = new AbortController()
    await expect(runCooperatively(fireflySiteSteps(1, field, habitats, index), {
      signal: controller.signal, yieldTask: () => { controller.abort(); return Promise.resolve() },
    })).rejects.toThrow()
    expect(habitats).toEqual(before)
    expect(await sites([1, 1, 1, 1])).toEqual([])
  })
})
