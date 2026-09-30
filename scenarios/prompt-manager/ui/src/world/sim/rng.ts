/**
 * mulberry32: a tiny seeded generator whose whole state is one 32-bit integer,
 * so it lives inside WorldState and every step is reproducible from the seed.
 * The magic constants below are the algorithm, not world levers.
 */

export function seedRng(seed: number): number {
  return (seed >>> 0) || 0x9e3779b9
}

/** Advance the generator; returns the next state and a float in [0, 1). */
export function nextRandom(state: number): [next: number, value: number] {
  const a = (state + 0x6d2b79f5) >>> 0
  let t = a
  t = Math.imul(t ^ (t >>> 15), t | 1)
  t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
  const value = ((t ^ (t >>> 14)) >>> 0) / 4294967296
  return [a, value]
}

/** Stable 32-bit hash of a string (FNV-1a). Used for per-actor cosmetic variation. */
export function hashString(input: string): number {
  let h = 0x811c9dc5
  for (let i = 0; i < input.length; i += 1) {
    h ^= input.charCodeAt(i)
    h = Math.imul(h, 0x01000193)
  }
  return h >>> 0
}

/** A stateless random stream bound to a state cell, for code that draws several values in a row. */
export class Rng {
  constructor(public state: number) {}

  next(): number {
    const [next, value] = nextRandom(this.state)
    this.state = next
    return value
  }

  range(min: number, max: number): number {
    return min + (max - min) * this.next()
  }

  int(maxExclusive: number): number {
    return Math.floor(this.next() * maxExclusive)
  }

  /** Weighted pick; weights need not sum to one. Returns -1 for all-zero weights. */
  weighted(weights: readonly number[]): number {
    const total = weights.reduce((sum, w) => sum + Math.max(0, w), 0)
    if (total <= 0) return -1
    let roll = this.next() * total
    for (let i = 0; i < weights.length; i += 1) {
      roll -= Math.max(0, weights[i] ?? 0)
      if (roll < 0) return i
    }
    return weights.length - 1
  }
}
