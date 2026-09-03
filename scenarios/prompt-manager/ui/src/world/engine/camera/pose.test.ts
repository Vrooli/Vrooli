import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { clampPose, extentPoints, fitDistance, footprintFill, frameDistance, orbitClamps, poseToPosition } from './pose'

describe('poseToPosition', () => {
  it('places the camera at factor × fit distance from the target', () => {
    const { position, target, distance } = poseToPosition({ azimuthDeg: 30, polarDeg: 45, distanceFactor: 2, targetY: 1 }, [2, -3], 10)
    const d = Math.hypot(position[0] - target[0], position[1] - target[1], position[2] - target[2])
    expect(d).toBeCloseTo(20, 6)
    expect(distance).toBe(20)
    expect(target).toEqual([2, 1, -3])
  })

  it('polar 0 looks straight down and polar 90 is level with the target', () => {
    const top = poseToPosition({ azimuthDeg: 0, polarDeg: 0, distanceFactor: 1, targetY: 0 }, [0, 0], 10)
    expect(top.position[1]).toBeCloseTo(10, 6)
    const level = poseToPosition({ azimuthDeg: 0, polarDeg: 90, distanceFactor: 1, targetY: 0 }, [0, 0], 10)
    expect(level.position[1]).toBeCloseTo(0, 6)
    expect(level.position[2]).toBeCloseTo(10, 6)
  })

  it('fitDistance grows with the slab and uses the narrower field of view', () => {
    const small = fitDistance({ width: 20, depth: 20, fovDeg: 32, aspect: 1.6 })
    const large = fitDistance({ width: 40, depth: 40, fovDeg: 32, aspect: 1.6 })
    expect(large).toBeCloseTo(small * 2, 6)
    const portrait = fitDistance({ width: 20, depth: 20, fovDeg: 32, aspect: 0.6 })
    expect(portrait).toBeGreaterThan(small)
    // At the fit distance the enclosing sphere just touches the frustum edge.
    const radius = 0.5 * Math.hypot(20, 20)
    expect(small * Math.sin((32 * Math.PI) / 360)).toBeCloseTo(radius, 6)
  })
})

describe('orbit clamps', () => {
  it('derive from tuning and centre the azimuth window on the hero pose', () => {
    const clamps = orbitClamps(tuning.camera, 20)
    expect(clamps.minPolar).toBeCloseTo((tuning.camera.polarMinDeg * Math.PI) / 180)
    expect(clamps.maxAzimuth - clamps.minAzimuth).toBeCloseTo((2 * tuning.camera.azimuthRangeDeg * Math.PI) / 180)
  })

  it('clampPose keeps every requested pose inside the diorama', () => {
    const clamps = orbitClamps(tuning.camera, 0)
    const wild = clampPose({ azimuthDeg: 170, polarDeg: 5, distanceFactor: 50, targetY: 0 }, clamps, 10)
    expect(wild.azimuthDeg).toBeCloseTo(tuning.camera.azimuthRangeDeg, 9)
    expect(wild.polarDeg).toBeCloseTo(tuning.camera.polarMinDeg, 9)
    expect(wild.distanceFactor * 10).toBeCloseTo(tuning.camera.maxDistance, 9)
  })
})

describe('footprint framing', () => {
  const box = { width: 40, depth: 26, center: [0, 0] as const }
  const base = { points: extentPoints(box), center: box.center, height: 2, polarDeg: 52, azimuthDeg: 20, targetY: 0.6, fovDeg: 38, aspect: 1.6 }

  it('frameDistance and footprintFill round-trip at the requested fill', () => {
    for (const fill of [0.6, 0.82, 1]) {
      const distance = frameDistance(base, fill)
      expect(footprintFill(base, distance)).toBeCloseTo(fill, 6)
    }
  })

  it('a bigger footprint needs a farther camera, and a farther camera fills less', () => {
    const near = frameDistance(base, 0.8)
    const wide = frameDistance({ ...base, points: extentPoints({ ...box, width: 80, depth: 52 }) }, 0.8)
    expect(wide).toBeGreaterThan(near)
    expect(footprintFill(base, near * 2)).toBeLessThan(0.8)
    expect(footprintFill(base, near * 0.5)).toBeGreaterThan(1)
  })

  it('a portrait viewport frames on width and lands the same fill', () => {
    const portrait = { ...base, aspect: 0.6 }
    const distance = frameDistance(portrait, 0.8)
    expect(distance).toBeGreaterThan(frameDistance(base, 0.8))
    expect(footprintFill(portrait, distance)).toBeCloseTo(0.8, 6)
  })

  it('framing depends on the outline extent, never on where it sits', () => {
    const shifted = { ...box, center: [35, -20] as const }
    const moved = { ...base, points: extentPoints(shifted), center: shifted.center }
    expect(frameDistance(moved, 0.8)).toBeCloseTo(frameDistance(base, 0.8), 9)
  })

  it('an outline with empty corners frames closer than its bounding box', () => {
    // A cross shape: same extent as the box, nothing in the corners.
    const cross: Array<readonly [number, number]> = [[-20, -3], [20, -3], [-20, 3], [20, 3], [-3, -13], [3, -13], [-3, 13], [3, 13]]
    expect(frameDistance({ ...base, points: cross }, 0.8)).toBeLessThan(frameDistance(base, 0.8))
  })
})
