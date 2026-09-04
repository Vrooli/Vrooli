import { describe, expect, it } from 'vitest'
import type { TerrainTuning } from '../../../config'
import { buildTerrain, heightAt, slopeAt } from '..'

const tuning: TerrainTuning = {
  radius: 24,
  cellSize: 1,
  amplitude: 1.5,
  frequency: 0.04,
  octaves: 4,
  lacunarity: 2,
  gain: 0.5,
  moistureFrequency: 0.015,
  falloffStart: 0.55,
  waterLevel: -0.45,
  shoreMargin: 1.2,
  maxSiteSlope: 0.14,
  maxWalkSlope: 0.45,
  kerbWidth: 2,
  pathWidth: 1.4,
  innerCellSize: 1,
  innerRadius: 20,
  ringFalloff: 2,
  tileSize: 12,
  moistureBasinDepth: 0.35,
  shoreMinGrade: 0.03,
  padClearance: 0.75,
  siteLevelTolerance: 0.05,
}

describe('terrain field', () => {
  it('is byte-deterministic and seed-sensitive', () => {
    const first = buildTerrain({ seed: 7, tuning })
    const again = buildTerrain({ seed: 7, tuning })
    const other = buildTerrain({ seed: 8, tuning })
    expect(new Uint8Array(first.height.buffer)).toEqual(new Uint8Array(again.height.buffer))
    expect(first.height).not.toEqual(other.height)
  })

  it('stays within amplitude and reaches zero at and outside the radius', () => {
    const field = buildTerrain({ seed: 11, tuning })
    expect(Math.max(...field.height.map(Math.abs))).toBeLessThanOrEqual(tuning.amplitude)
    expect(heightAt(field, tuning.radius, 0)).toBe(0)
    expect(heightAt(field, tuning.radius + 1, 0)).toBe(0)
  })

  it('interpolates continuously across cell boundaries', () => {
    const field = buildTerrain({ seed: 17, tuning })
    const left = heightAt(field, 3 - 1e-5, -2.25)
    const right = heightAt(field, 3 + 1e-5, -2.25)
    expect(Math.abs(left - right)).toBeLessThan(1e-4)
  })

  it('reports the same slope as a numeric gradient', () => {
    const field = buildTerrain({ seed: 23, tuning })
    const epsilon = field.cellSize * 0.5
    const dx = (heightAt(field, 2 + epsilon, 3) - heightAt(field, 2 - epsilon, 3)) / (epsilon * 2)
    const dz = (heightAt(field, 2, 3 + epsilon) - heightAt(field, 2, 3 - epsilon)) / (epsilon * 2)
    expect(slopeAt(field, 2, 3)).toBeCloseTo(Math.atan(Math.hypot(dx, dz)), 8)
  })
})
