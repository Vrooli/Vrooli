import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { cameraFacing, facingWeight } from './faceCamera'

describe('presentation facing', () => {
  it('returns simulation facing exactly at zero weight and viewer yaw at full weight', () => {
    expect(cameraFacing(9, 0, 0, Math.PI)).toBe(9)
    expect(cameraFacing(0, 1.7, 1, Math.PI)).toBeCloseTo(1.7, 10)
  })
  it('uses the short arc across the signed-pi boundary', () => {
    const halfway = cameraFacing(3.1, -3.1, 0.5, Math.PI)
    expect(Math.abs(halfway)).toBeCloseTo(Math.PI, 10)
  })
  it('bounds total yaw and the interpolation weight', () => {
    expect(cameraFacing(0, Math.PI, 1, Math.PI / 4)).toBeCloseTo(Math.PI / 4, 10)
    expect(cameraFacing(0, Math.PI, 2, Math.PI / 4)).toBeCloseTo(Math.PI / 4, 10)
    expect(cameraFacing(0, Math.PI, 1, 0)).toBe(0)
  })
  it('ramps resting focus in and out and leaves moving actors unaffected', () => {
    const t = tuning.actor.facing
    const half = facingWeight(0, true, 0, t.blendSeconds / 2, t)
    expect(half).toBeCloseTo(0.5)
    expect(facingWeight(half, true, 0, t.blendSeconds / 2, t)).toBe(1)
    expect(facingWeight(1, false, 0, t.blendSeconds / 2, t)).toBeCloseTo(0.5)
    expect(facingWeight(1, true, t.restSpeed + 0.001, 1 / 60, t)).toBe(0)
  })
})
