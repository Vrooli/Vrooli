import { describe, it, expect } from 'vitest'
import { seatLocalToWorld, seatWorldToLocal, seatFacingArrowOffset, rotateAllSeats } from './world'

const PI = Math.PI
const HALF_PI = PI / 2

/** Asserts array element exists (avoids non-null assertions flagged by eslint). */
function el<T>(arr: T[], i: number): T {
  const v = arr[i]
  if (v === undefined) throw new Error(`No element at index ${i}`)
  return v
}

// ---------------------------------------------------------------------------
// Three.js Ry(θ) reference implementation for test verification.
// Ry(θ) = [[cos θ, 0, sin θ], [0, 1, 0], [-sin θ, 0, cos θ]]
// ---------------------------------------------------------------------------

/** Apply Three.js Ry(θ) matrix to a point, then add an offset. */
function threeJsRotateY(
  point: [number, number, number],
  offset: [number, number, number],
  theta: number,
): [number, number, number] {
  const cos = Math.cos(theta)
  const sin = Math.sin(theta)
  return [
    offset[0] + point[0] * cos + point[2] * sin,
    offset[1] + point[1],
    offset[2] - point[0] * sin + point[2] * cos,
  ]
}

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

  it('offset with π/2 rotation — matches Three.js Ry(π/2)', () => {
    // Three.js Ry(π/2): (1,0,0) → (0, 0, -1)
    const result = seatLocalToWorld([1, 0, 0], [0, 0, 0], HALF_PI)
    expect(result[0]).toBeCloseTo(0)
    expect(result[1]).toBeCloseTo(0)
    expect(result[2]).toBeCloseTo(-1)
  })

  it('offset with π rotation — matches Three.js Ry(π)', () => {
    // Three.js Ry(π): (1,0,0) → (-1, 0, 0)
    const result = seatLocalToWorld([1, 0, 0], [0, 0, 0], PI)
    expect(result[0]).toBeCloseTo(-1)
    expect(result[1]).toBeCloseTo(0)
    expect(result[2]).toBeCloseTo(0)
  })

  it('offset with 3π/2 rotation — matches Three.js Ry(3π/2)', () => {
    // Three.js Ry(3π/2): (1,0,0) → (0, 0, 1)
    const result = seatLocalToWorld([1, 0, 0], [5, 0, 5], 3 * HALF_PI)
    expect(result[0]).toBeCloseTo(5)
    expect(result[1]).toBeCloseTo(0)
    expect(result[2]).toBeCloseTo(6)
  })

  it('Z offset with π/2 rotation — matches Three.js', () => {
    // Three.js Ry(π/2): (0,0,1) → (1, 0, 0)
    const result = seatLocalToWorld([0, 0, 1], [0, 0, 0], HALF_PI)
    expect(result[0]).toBeCloseTo(1)
    expect(result[1]).toBeCloseTo(0)
    expect(result[2]).toBeCloseTo(0)
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

  it('π/2 rotation inverse — undoes Three.js Ry(π/2)', () => {
    // Forward: (1,0,0) at π/2 → (0,0,-1), so inverse: (0,0,-1) at π/2 → (1,0,0)
    const result = seatWorldToLocal([0, 0, -1], [0, 0, 0], HALF_PI)
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

describe('seatLocalToWorld matches Three.js Ry(θ)', () => {
  const cases: Array<{
    label: string
    local: [number, number, number]
    furnPos: [number, number, number]
    rotation: number
  }> = [
    { label: 'X-axis seat at 0°', local: [1, 0, 0], furnPos: [0, 0, 0], rotation: 0 },
    { label: 'X-axis seat at 45°', local: [1, 0, 0], furnPos: [0, 0, 0], rotation: PI / 4 },
    { label: 'X-axis seat at 90°', local: [1, 0, 0], furnPos: [0, 0, 0], rotation: HALF_PI },
    { label: 'X-axis seat at 180°', local: [1, 0, 0], furnPos: [0, 0, 0], rotation: PI },
    { label: 'X-axis seat at 270°', local: [1, 0, 0], furnPos: [0, 0, 0], rotation: 3 * HALF_PI },
    { label: 'Z-axis seat at 90°', local: [0, 0, 1], furnPos: [0, 0, 0], rotation: HALF_PI },
    { label: 'diagonal seat at 90°', local: [1, 0, 1], furnPos: [0, 0, 0], rotation: HALF_PI },
    { label: 'bench left seat at 45°', local: [-0.4, 0.5, 0.05], furnPos: [2, 0, 3], rotation: PI / 4 },
    { label: 'bench right seat at 225°', local: [0.4, 0.5, 0.05], furnPos: [-1, 0, 2], rotation: 5 * PI / 4 },
    { label: 'arbitrary offset + rotation', local: [1.5, 0.3, -0.8], furnPos: [3, 1, -4], rotation: 2.1 },
  ]

  for (const { label, local, furnPos, rotation } of cases) {
    it(label, () => {
      const result = seatLocalToWorld(local, furnPos, rotation)
      const expected = threeJsRotateY(local, furnPos, rotation)
      expect(result[0]).toBeCloseTo(expected[0], 10)
      expect(result[1]).toBeCloseTo(expected[1], 10)
      expect(result[2]).toBeCloseTo(expected[2], 10)
    })
  }
})

describe('bench seat alignment across rotations', () => {
  const benchSeats: [number, number, number][] = [
    [-0.4, 0.5, 0.05],
    [0, 0.5, 0.05],
    [0.4, 0.5, 0.05],
  ]
  const furnPos: [number, number, number] = [0, 0, 0]

  it('seats at 0° are spread along X axis', () => {
    const worldSeats = benchSeats.map((s) => seatLocalToWorld(s, furnPos, 0))
    // At 0° rotation, seats should be at x = -0.4, 0, 0.4
    expect(el(worldSeats, 0)[0]).toBeCloseTo(-0.4)
    expect(el(worldSeats, 1)[0]).toBeCloseTo(0)
    expect(el(worldSeats, 2)[0]).toBeCloseTo(0.4)
    // Z should all be 0.05
    for (const ws of worldSeats) {
      expect(ws[2]).toBeCloseTo(0.05)
    }
  })

  it('seats at 90° are spread along -Z axis', () => {
    const worldSeats = benchSeats.map((s) => seatLocalToWorld(s, furnPos, HALF_PI))
    // Ry(π/2) rotates +X → -Z, so seats spread along -Z
    expect(el(worldSeats, 0)[2]).toBeCloseTo(0.4)  // -(-0.4)*sin(π/2)
    expect(el(worldSeats, 1)[2]).toBeCloseTo(0)
    expect(el(worldSeats, 2)[2]).toBeCloseTo(-0.4) // -(0.4)*sin(π/2)
    // X should all be ~0.05*sin(π/2) = 0.05
    for (const ws of worldSeats) {
      expect(ws[0]).toBeCloseTo(0.05)
    }
  })

  it('seats at 180° are spread along -X axis (mirrored)', () => {
    const worldSeats = benchSeats.map((s) => seatLocalToWorld(s, furnPos, PI))
    expect(el(worldSeats, 0)[0]).toBeCloseTo(0.4)
    expect(el(worldSeats, 1)[0]).toBeCloseTo(0)
    expect(el(worldSeats, 2)[0]).toBeCloseTo(-0.4)
  })

  it('seat spread distance is preserved across all rotations', () => {
    const rotations = [0, PI / 4, HALF_PI, PI, 3 * HALF_PI, 5.5]
    for (const rot of rotations) {
      const worldSeats = benchSeats.map((s) => seatLocalToWorld(s, furnPos, rot))
      // Distance between left and right seats should always be 0.8
      const dx = el(worldSeats, 2)[0] - el(worldSeats, 0)[0]
      const dz = el(worldSeats, 2)[2] - el(worldSeats, 0)[2]
      const dist = Math.hypot(dx, dz)
      expect(dist).toBeCloseTo(0.8, 5)
    }
  })
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

  it('matches Three.js Ry(θ) applied to +Z unit vector', () => {
    const rotations = [0, PI / 4, HALF_PI, PI, 3 * HALF_PI, 2.7]
    for (const rot of rotations) {
      const arrow = seatFacingArrowOffset(rot, 1)
      const expected = threeJsRotateY([0, 0, 1], [0, 0, 0], rot)
      expect(arrow[0]).toBeCloseTo(expected[0], 10)
      expect(arrow[1]).toBeCloseTo(expected[1], 10)
      expect(arrow[2]).toBeCloseTo(expected[2], 10)
    }
  })
})

describe('rotateAllSeats', () => {
  it('rotate by π/2 — matches Three.js convention', () => {
    const seats = [{ position: [1, 0.5, 0] as [number, number, number], rotation: 0 }]
    const result = rotateAllSeats(seats, HALF_PI)
    expect(result).toHaveLength(1)
    // Ry(π/2) * (1, 0.5, 0) → (0, 0.5, -1)
    const r0 = el(result, 0)
    expect(r0.position[0]).toBeCloseTo(0)
    expect(r0.position[1]).toBeCloseTo(0.5)
    expect(r0.position[2]).toBeCloseTo(-1)
    expect(r0.rotation).toBeCloseTo(HALF_PI)
  })

  it('rotate by 2π — identity', () => {
    const seats = [
      { position: [1, 2, 3] as [number, number, number], rotation: 1 },
      { position: [-0.5, 0, 0.5] as [number, number, number], rotation: 0.5 },
    ]
    const result = rotateAllSeats(seats, PI * 2)
    for (let i = 0; i < seats.length; i++) {
      const ri = el(result, i)
      const si = el(seats, i)
      expect(ri.position[0]).toBeCloseTo(si.position[0])
      expect(ri.position[1]).toBeCloseTo(si.position[1])
      expect(ri.position[2]).toBeCloseTo(si.position[2])
      expect(ri.rotation).toBeCloseTo(si.rotation)
    }
  })

  it('empty array → empty array', () => {
    expect(rotateAllSeats([], HALF_PI)).toEqual([])
  })

  it('rotation normalizes to [0, 2π)', () => {
    const seats = [{ position: [0, 0, 0] as [number, number, number], rotation: PI * 1.5 }]
    const result = rotateAllSeats(seats, PI)
    // 1.5π + π = 2.5π → normalized to 0.5π
    expect(el(result, 0).rotation).toBeCloseTo(HALF_PI)
  })

  it('negative delta normalizes correctly', () => {
    const seats = [{ position: [0, 0, 0] as [number, number, number], rotation: 0.5 }]
    const result = rotateAllSeats(seats, -PI)
    // 0.5 - π ≈ -2.64 → normalized to ≈ 3.64
    const r0 = el(result, 0)
    expect(r0.rotation).toBeGreaterThanOrEqual(0)
    expect(r0.rotation).toBeLessThan(PI * 2)
    expect(r0.rotation).toBeCloseTo(0.5 + PI)
  })

  it('XZ transform matches Three.js Ry(δ)', () => {
    const seats = [
      { position: [1, 0, 0] as [number, number, number], rotation: 0 },
      { position: [0, 0, 1] as [number, number, number], rotation: 0 },
      { position: [1, 0, 1] as [number, number, number], rotation: 0 },
    ]
    const delta = PI / 3
    const result = rotateAllSeats(seats, delta)
    for (let i = 0; i < seats.length; i++) {
      const expected = threeJsRotateY(el(seats, i).position, [0, 0, 0], delta)
      const ri = el(result, i)
      expect(ri.position[0]).toBeCloseTo(expected[0], 10)
      expect(ri.position[1]).toBeCloseTo(expected[1], 10)
      expect(ri.position[2]).toBeCloseTo(expected[2], 10)
    }
  })
})
