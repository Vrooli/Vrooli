import { describe, expect, it } from 'vitest'
import { clampPointToBoundary, isPointInsideBoundary, localToWorld2, resolveBoundary, rotateAllSeats, seatFacingArrowOffset, seatLocalToWorld, seatWorldToLocal, snapPointToGrid } from '../layout/seatMath'

describe('seat math (ported)', () => {
  it('local/world transforms are inverses under rotation', () => {
    const local: [number, number, number] = [1.5, 0.2, -0.7]
    const pos: [number, number, number] = [3, 0, -2]
    const world = seatLocalToWorld(local, pos, 0.8)
    const back = seatWorldToLocal(world, pos, 0.8)
    back.forEach((v, i) => expect(v).toBeCloseTo(local[i] ?? NaN, 9))
  })

  it('rotation follows the three.js Ry convention', () => {
    const w = seatLocalToWorld([1, 0, 0], [0, 0, 0], Math.PI / 2)
    expect(w[0]).toBeCloseTo(0, 9)
    expect(w[2]).toBeCloseTo(-1, 9)
  })

  it('rotateAllSeats rotates offsets and normalises facings', () => {
    const seats = rotateAllSeats([{ position: [1, 0, 0] as [number, number, number], rotation: Math.PI * 1.75 }], Math.PI / 2)
    expect(seats[0]?.position[2]).toBeCloseTo(-1, 9)
    expect(seats[0]?.rotation).toBeCloseTo(Math.PI * 0.25, 9)
  })

  it('facing arrow points along the yaw', () => {
    const [x, , z] = seatFacingArrowOffset(0, 2)
    expect([x, z]).toEqual([0, 2])
  })

  it('boundaries contain, clamp and snap', () => {
    const square = resolveBoundary('square', 10)
    expect(isPointInsideBoundary([4, 4], square)).toBe(true)
    expect(isPointInsideBoundary([6, 0], square)).toBe(false)
    expect(clampPointToBoundary([6, 0], square)[0]).toBeCloseTo(5, 9)
    const circle = resolveBoundary('circle', 10)
    expect(clampPointToBoundary([10, 0], circle)).toEqual([5, 0])
    expect(clampPointToBoundary([0, 0], circle)).toEqual([0, 0])
    const path = resolveBoundary('path', 10, [[0, 0], [4, 0], [4, 4], [0, 4]])
    expect(isPointInsideBoundary([2, 2], path)).toBe(true)
    expect(isPointInsideBoundary([5, 5], path)).toBe(false)
    expect(snapPointToGrid([1.3, -0.7], 0.5)).toEqual([1.5, -0.5])
    expect(snapPointToGrid([1.3, -0.7], 0)).toEqual([1.3, -0.7])
    expect(localToWorld2([1, 0], [2, 2], 0)).toEqual([3, 2])
  })
})
