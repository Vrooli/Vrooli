import { describe, expect, it } from 'vitest'
import { biomeSets, tuning } from '../../../config'
import { HABITAT_IDS } from '../../../config/habitats'
import { habitatFieldSteps, habitatRegionSteps } from '../habitats'
import { runCooperatively } from '../../cooperative'
import { generateWorld } from '../../world'
import { makeWorldInput } from '../../__tests__/fixtures'
import type { Place } from '../../model'

describe('generated habitat field', () => {
  it('partitions orthogonal regions without joining diagonals or wrapping rows', async () => {
    const field = { cols: 3, rows: 3, originX: 10, originZ: 20, cellSize: 2, radius: 20,
      height: new Float32Array(9), moisture: new Float32Array(9) }
    const values = new Uint8Array([1, 1, 0, 0, 0, 1, 2, 2, 1])
    const result = await runCooperatively(habitatRegionSteps(field, values))
    expect([...result.labels]).toEqual([1, 1, 0, 0, 0, 6, 7, 7, 6])
    expect(result.regions.map(region => ({ id: region.id, habitat: region.habitat, samples: region.samples }))).toEqual([
      { id: 1, habitat: 'meadow', samples: 2 }, { id: 6, habitat: 'meadow', samples: 2 }, { id: 7, habitat: 'woodland', samples: 2 },
    ])
    expect(result.regions[1]?.bounds).toEqual({ minX: 14, maxX: 14, minZ: 22, maxZ: 24 })
    const members = result.regions.flatMap(region => [...result.members.slice(region.offset, region.offset + region.samples)])
    expect([...members].sort((a, b) => a - b)).toEqual([0, 1, 5, 6, 7, 8])
    expect(new Set(members).size).toBe(members.length)
    expect(await runCooperatively(habitatRegionSteps(field, values))).toEqual(result)
    await expect(runCooperatively(habitatRegionSteps(field, values, 2))).rejects.toThrow('region budget')
  })
  it('maps habitat classes and excludes paths, rotated built footprints and the outside disc', async () => {
    const field = { cols: 10, rows: 1, originX: -4, originZ: 0, cellSize: 1, radius: 4,
      height: new Float32Array(10), moisture: new Float32Array(10) }
    const names = ['meadow', 'forest', 'rocky', 'wetland', 'water', 'meadow', 'meadow', 'meadow', 'meadow', 'meadow']
    const biomes = Uint8Array.from(names.map(name => biomeSets.park.biomes.findIndex(biome => biome.id === name)))
    const paths = new Float32Array(10)
    paths[5] = 0.01
    const place: Place = { id: 'room:test', kind: 'room', position: [3, 0], size: [2, 0.5], rotation: Math.PI / 2, seats: [], label: 'Test' }
    const result = await runCooperatively(habitatFieldSteps(field, biomes, biomeSets.park, paths, [place]))
    expect([...result].map(index => HABITAT_IDS[index])).toEqual([
      'meadow', 'woodland', 'rocky-slope', 'wetland', 'open-water', 'none', 'meadow', 'none', 'meadow', 'none',
    ])
    expect(paths[5]).toBeCloseTo(0.01)
    expect(biomes[7]).toBe(biomeSets.park.biomes.findIndex(biome => biome.id === 'meadow'))
  })
  it('publishes all required terrestrial habitat types for the default seed deterministically', () => {
    const input = makeWorldInput({ seed: 1, teams: 5, agents: 25 })
    const first = generateWorld(input, tuning)
    const second = generateWorld(input, tuning)
    expect(first.habitats.length).toBe(first.terrain.cols * first.terrain.rows)
    expect(first.habitats).toEqual(second.habitats)
    expect(first.habitatRegions).toEqual(second.habitatRegions)
    expect(first.habitatRegions.regions.reduce((sum, region) => sum + region.samples, 0)).toBe([...first.habitats].filter(Boolean).length)
    const ids = new Set([...first.habitats].map(index => HABITAT_IDS[index]))
    for (const id of ['meadow', 'woodland', 'rocky-slope', 'wetland']) expect(ids.has(id as typeof HABITAT_IDS[number])).toBe(true)
    expect(first.habitats.byteLength).toBe(first.terrain.height.length)
  })
  it('does not publish partially prepared fields after cancellation', async () => {
    const field = { cols: 1, rows: 1, originX: 0, originZ: 0, cellSize: 1, radius: 1, height: new Float32Array(1), moisture: new Float32Array(1) }
    const controller = new AbortController()
    await expect(runCooperatively(habitatFieldSteps(field, new Uint8Array(1), biomeSets.park, new Float32Array(1), []), {
      signal: controller.signal, yieldTask: () => { controller.abort(); return Promise.resolve() },
    })).rejects.toMatchObject({ name: 'AbortError' })
    const regionController = new AbortController()
    await expect(runCooperatively(habitatRegionSteps(field, new Uint8Array([1])), {
      signal: regionController.signal, yieldTask: () => { regionController.abort(); return Promise.resolve() },
    })).rejects.toMatchObject({ name: 'AbortError' })
  })
})
