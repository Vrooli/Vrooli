import { describe, expect, it } from 'vitest'
import { heightAt, maximumHeightInRegion, type TerrainField } from './field'

const field: TerrainField = { radius: 20, originX: -2, originZ: -2, cellSize: 1, cols: 5, rows: 5, height: new Float32Array([0, 0, 0, 0, 0, 0, 1, 4, 2, 0, 0, 3, 9, 1, 0, 0, 2, 1, 4, 0, 0, 0, 0, 0, 0]), moisture: new Float32Array(25) }

describe('terrain clearance ceiling', () => {
  it('bounds all sampled points, including an interior peak missed by rectangle corners', () => {
    const ceiling = maximumHeightInRegion(field, -1.5, -1.5, 1.5, 1.5)
    expect(ceiling).toBe(9)
    for (let x = -1.5; x <= 1.5; x += 0.1) for (let z = -1.5; z <= 1.5; z += 0.1) expect(heightAt(field, x, z)).toBeLessThanOrEqual(ceiling)
  })
  it('includes the full interpolation cell even for a small footprint', () => {
    expect(maximumHeightInRegion(field, -0.9, -0.9, -0.8, -0.8)).toBe(9)
    expect(maximumHeightInRegion(field, 50, 50, 51, 51)).toBe(0)
    expect(() => maximumHeightInRegion(field, NaN, 0, 1, 1)).toThrow('Invalid terrain query')
  })
})
