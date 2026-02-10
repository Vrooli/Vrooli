import { describe, it, expect } from 'vitest'
import { seatLocalToWorld, seatWorldToLocal, seatFacingArrowOffset, rotateAllSeats } from './world'

const PI = Math.PI
const HALF_PI = PI / 2

describe('seatLocalToWorld', () => {
  it('zero rotation + zero offset → identity', () => {
    const pos: [number, number, number] = [5, 0, 3]
    expect(seatLocalToWorld([0, 0, 0], pos, 0)).toEqual(pos)
  })

  it('non-zero offset with 0 rotation', () => {
    const result = seatLocalToWorld([1, 0.5, 2], [10, 0, 10], 0)
    expect(result[0]).toBeCloseTo(11)
    expect(result[1]).toBeCloseTo(0.5)
    expect(result[2]).toBeCloseTo(12)
  })

  it('offset with π/2 rotation', () => {
    // Rotating 90° CW: x→z, z→-x  (standard Y-up rotation matrix)
    const result = seatLocalToWorld([1, 0, 0], [0, 0, 0], HALF_PI)
    expect(result[0]).toBeCloseTo(0)  // cos(π/2)*1 ≈ 0
    expect(result[1]).toBeCloseTo(0)
    expect(result[2]).toBeCloseTo(1)  // sin(π/2)*1 ≈ 1
  })

  it('offset with π rotation', () => {
    const result = seatLocalToWorld([1, 0, 0], [0, 0, 0], PI)
    expect(result[0]).toBeCloseTo(-1) // cos(π)*1 = -1
    expect(result[1]).toBeCloseTo(0)
    expect(result[2]).toBeCloseTo(0)  // sin(π)*1 ≈ 0
  })

  it('offset with 3π/2 rotation', () => {
    const result = seatLocalToWorld([1, 0, 0], [5, 0, 5], 3 * HALF_PI)
    expect(result[0]).toBeCloseTo(5)  // 5 + cos(3π/2)*1 ≈ 5+0
    expect(result[1]).toBeCloseTo(0)
    expect(result[2]).toBeCloseTo(4)  // 5 + sin(3π/2)*1 ≈ 5-1
  })

  it('Y-axis preservation — rotation only affects XZ', () => {
    const result = seatLocalToWorld([1, 3, 1], [0, 2, 0], HALF_PI)
    expect(result[1]).toBeCloseTo(5) // 2 + 3
  })

  it('negative seat offset', () => {
    const result = seatLocalToWorld([-2, 0, -3], [0, 0, 0], 0)
    expect(result[0]).toBeCloseTo(-2)
    expect(result[2]).toBeCloseTo(-3)
  })
})

describe('seatWorldToLocal', () => {
  it('zero rotation → simple subtraction', () => {
    const result = seatWorldToLocal([11, 0.5, 12], [10, 0, 10], 0)
    expect(result[0]).toBeCloseTo(1)
    expect(result[1]).toBeCloseTo(0.5)
    expect(result[2]).toBeCloseTo(2)
  })

  it('π/2 rotation inverse', () => {
    const result = seatWorldToLocal([0, 0, 1], [0, 0, 0], HALF_PI)
    expect(result[0]).toBeCloseTo(1)
    expect(result[1]).toBeCloseTo(0)
    expect(result[2]).toBeCloseTo(0)
  })

  it('Y-axis preservation', () => {
    const result = seatWorldToLocal([5, 7, 5], [5, 2, 5], PI)
    expect(result[1]).toBeCloseTo(5) // 7 - 2
  })
})

describe('seatLocalToWorld ↔ seatWorldToLocal roundtrip', () => {
  const rotations = [0, HALF_PI, PI, 3 * HALF_PI, 0.7, -1.2]
  const furniturePos: [number, number, number] = [3, 1, -4]

  for (const rot of rotations) {
    it(`roundtrip at rotation=${rot.toFixed(2)}`, () => {
      const local: [number, number, number] = [1.5, 0.3, -0.8]
      const world = seatLocalToWorld(local, furniturePos, rot)
      const back = seatWorldToLocal(world, furniturePos, rot)
      expect(back[0]).toBeCloseTo(local[0])
      expect(back[1]).toBeCloseTo(local[1])
      expect(back[2]).toBeCloseTo(local[2])
    })
  }
})

describe('seatFacingArrowOffset', () => {
  it('0 rotation → offset along +Z', () => {
    const result = seatFacingArrowOffset(0)
    expect(result[0]).toBeCloseTo(0)
    expect(result[1]).toBeCloseTo(0)
    expect(result[2]).toBeCloseTo(0.15)
  })

  it('π/2 rotation → offset along +X', () => {
    const result = seatFacingArrowOffset(HALF_PI)
    expect(result[0]).toBeCloseTo(0.15)
    expect(result[1]).toBeCloseTo(0)
    expect(result[2]).toBeCloseTo(0)
  })

  it('custom length', () => {
    const result = seatFacingArrowOffset(0, 1)
    expect(result[0]).toBeCloseTo(0)
    expect(result[2]).toBeCloseTo(1)
  })

  it('Y is always 0', () => {
    const result = seatFacingArrowOffset(1.23, 5)
    expect(result[1]).toBe(0)
  })
})

describe('rotateAllSeats', () => {
  it('rotate by π/2 — XZ swaps', () => {
    const seats = [{ position: [1, 0.5, 0] as [number, number, number], rotation: 0 }]
    const result = rotateAllSeats(seats, HALF_PI)
    expect(result).toHaveLength(1)
    expect(result[0]!.position[0]).toBeCloseTo(0)
    expect(result[0]!.position[1]).toBeCloseTo(0.5)
    expect(result[0]!.position[2]).toBeCloseTo(1)
    expect(result[0]!.rotation).toBeCloseTo(HALF_PI)
  })

  it('rotate by 2π — identity', () => {
    const seats = [
      { position: [1, 2, 3] as [number, number, number], rotation: 1 },
      { position: [-0.5, 0, 0.5] as [number, number, number], rotation: 0.5 },
    ]
    const result = rotateAllSeats(seats, PI * 2)
    for (let i = 0; i < seats.length; i++) {
      expect(result[i]!.position[0]).toBeCloseTo(seats[i]!.position[0])
      expect(result[i]!.position[1]).toBeCloseTo(seats[i]!.position[1])
      expect(result[i]!.position[2]).toBeCloseTo(seats[i]!.position[2])
      expect(result[i]!.rotation).toBeCloseTo(seats[i]!.rotation)
    }
  })

  it('empty array → empty array', () => {
    expect(rotateAllSeats([], HALF_PI)).toEqual([])
  })

  it('rotation normalizes to [0, 2π)', () => {
    const seats = [{ position: [0, 0, 0] as [number, number, number], rotation: PI * 1.5 }]
    const result = rotateAllSeats(seats, PI)
    // 1.5π + π = 2.5π → normalized to 0.5π
    expect(result[0]!.rotation).toBeCloseTo(HALF_PI)
  })

  it('negative delta normalizes correctly', () => {
    const seats = [{ position: [0, 0, 0] as [number, number, number], rotation: 0.5 }]
    const result = rotateAllSeats(seats, -PI)
    // 0.5 - π ≈ -2.64 → normalized to ≈ 3.64
    expect(result[0]!.rotation).toBeGreaterThanOrEqual(0)
    expect(result[0]!.rotation).toBeLessThan(PI * 2)
    expect(result[0]!.rotation).toBeCloseTo(0.5 + PI)
  })
})
