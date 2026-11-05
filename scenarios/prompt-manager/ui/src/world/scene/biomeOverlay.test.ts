import { describe, expect, it } from 'vitest'
import { biomeSets } from '../config'
import { runCooperatively } from '../sim/cooperative'
import { biomeLegend, biomeOverlaySteps, habitatLegend, habitatOverlaySteps } from './classificationOverlay'

describe('committed biome diagnostic markers', () => {
  const field = { radius: 2, cols: 2, rows: 2, originX: 10, originZ: 20, cellSize: 2,
    height: new Float32Array([1, 2, 3, 4]), moisture: new Float32Array(4) }
  it('shows excluded habitat samples distinctly and retains stored class counts', async () => {
    const result = await runCooperatively(habitatOverlaySteps(field, new Uint8Array([0, 1, 2, 5])))
    expect(result.counts).toEqual([1, 1, 1, 0, 0, 1])
    expect(habitatLegend()[0]).toEqual({ label: 'none', color: '#777777' })
    expect(result.summary).toContain('open-water: 1')
    expect(result.positions[0]).toBe(10)
  })
  it('uses terrain sample positions and the stored classification rather than classifying again', async () => {
    const ids = new Uint8Array([0, 1, 2, 0])
    const result = await runCooperatively(biomeOverlaySteps(field, ids, biomeSets.park))
    expect(result.count).toBe(4)
    expect([...result.positions].filter((_, index) => index % 3 !== 1)).toEqual([10, 20, 12, 20, 10, 22, 12, 22])
    expect(result.positions[1]).toBeCloseTo(1.05)
    expect(result.positions[10]).toBeCloseTo(4.05)
    expect(result.counts.slice(0, 3)).toEqual([2, 1, 1])
    expect(result.bytes).toBe(96)
    expect([...result.colors.slice(0, 3)]).toEqual([...result.colors.slice(9, 12)])
    expect([...result.colors.slice(0, 3)]).not.toEqual([...result.colors.slice(3, 6)])
    expect(biomeLegend(biomeSets.park).map(entry => entry.label)).toEqual(biomeSets.park.biomes.map(biome => biome.id))
    expect([...ids]).toEqual([0, 1, 2, 0])
  })
  it('rejects mismatched sample counts and unknown stored biome indexes', async () => {
    await expect(runCooperatively(biomeOverlaySteps(field, new Uint8Array(3), biomeSets.park))).rejects.toThrow('Invalid biome overlay field')
    await expect(runCooperatively(biomeOverlaySteps(field, new Uint8Array([0, 255, 0, 0]), biomeSets.park))).rejects.toThrow('unknown biome')
  })
})
