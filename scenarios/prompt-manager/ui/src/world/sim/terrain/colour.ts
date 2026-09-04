import type { Biome } from '../../config'
import { heightAt, type TerrainField } from './field'

export type Rgb = readonly [number, number, number]

function rgb(hex: string): Rgb {
  return [Number.parseInt(hex.slice(1, 3), 0x10) / 0xff, Number.parseInt(hex.slice(3, 5), 0x10) / 0xff, Number.parseInt(hex.slice(5, 7), 0x10) / 0xff]
}

function mix(a: Rgb, b: Rgb, t: number): Rgb {
  return [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t]
}

export interface ColourSample {
  moisture: number
  path: number
  ao: number
  /** 0..1 darkening immediately inside a water contour. */
  wetShore?: number
  wetShoreDarkening?: number
}

export function bakeVertexColour(sample: ColourSample, biome: Biome): Rgb {
  const first = rgb(biome.ramp[0] ?? '#808080')
  const last = rgb(biome.ramp[biome.ramp.length - 1] ?? '#808080')
  let colour = mix(first, last, Math.max(0, Math.min(1, sample.moisture)))
  colour = mix(colour, rgb(biome.path), Math.max(0, Math.min(1, sample.path)))
  const shade = 1 - Math.max(0, Math.min(1, sample.ao)) * biome.aoStrength
  const wetShade = 1 - Math.max(0, Math.min(1, sample.wetShore ?? 0)) * Math.max(0, Math.min(1, sample.wetShoreDarkening ?? 0))
  return [colour[0] * shade * wetShade, colour[1] * shade * wetShade, colour[2] * shade * wetShade]
}

export function heightFieldAo(field: TerrainField, x: number, z: number, radius: number, samples: number): number {
  const center = heightAt(field, x, z)
  let occluded = 0
  for (let index = 0; index < samples; index += 1) {
    const angle = (index / samples) * Math.PI * 2
    const nearby = heightAt(field, x + Math.sin(angle) * radius, z + Math.cos(angle) * radius)
    occluded += Math.max(0, nearby - center) / Math.max(radius, Number.EPSILON)
  }
  return Math.max(0, Math.min(1, occluded / Math.max(1, samples)))
}
