import { hashString } from '../rng'

function fade(value: number): number {
  return value * value * (3 - 2 * value)
}

function corner(ix: number, iz: number, seed: number): number {
  return (hashString(`${seed}:${ix}:${iz}`) / 0xffffffff) * 2 - 1
}

/** Smooth deterministic lattice value noise in [-1, 1]. */
export function valueNoise2D(x: number, z: number, seed: number): number {
  const x0 = Math.floor(x)
  const z0 = Math.floor(z)
  const tx = fade(x - x0)
  const tz = fade(z - z0)
  const a = corner(x0, z0, seed)
  const b = corner(x0 + 1, z0, seed)
  const c = corner(x0, z0 + 1, seed)
  const d = corner(x0 + 1, z0 + 1, seed)
  const top = a + (b - a) * tx
  const bottom = c + (d - c) * tx
  return top + (bottom - top) * tz
}

/** Normalised fractal Brownian motion in [-1, 1]. */
export function fbm(x: number, z: number, seed: number, octaves: number, lacunarity: number, gain: number): number {
  let frequency = 1
  let amplitude = 1
  let total = 0
  let weight = 0
  for (let octave = 0; octave < octaves; octave += 1) {
    total += valueNoise2D(x * frequency, z * frequency, seed + octave * 0x9e3779b9) * amplitude
    weight += amplitude
    frequency *= lacunarity
    amplitude *= gain
  }
  return weight === 0 ? 0 : total / weight
}
