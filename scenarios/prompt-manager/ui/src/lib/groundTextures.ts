import * as THREE from 'three'
import type { GroundTextureId } from '@/types/environment'

export interface GroundTextureSet {
  id: GroundTextureId
  albedo: THREE.Texture
  normal: THREE.Texture
  roughness: THREE.Texture
  ao: THREE.Texture
  macro: THREE.Texture
}

interface GroundTexturePreset {
  base: [number, number, number]
  secondary: [number, number, number]
  accent?: [number, number, number]
  noiseScale: number
  noiseOctaves: number
  noisePersistence: number
  detailScale: number
  detailStrength: number
  macroScale: number
  macroStrength: number
  roughnessBase: number
  roughnessVariance: number
  heightStrength: number
  pattern?: 'none' | 'speckle' | 'stripe' | 'stone' | 'panel'
}

const DEFAULT_RESOLUTION = 512
const DEFAULT_MACRO_RESOLUTION = 256

const PRESETS: Record<GroundTextureId, GroundTexturePreset> = {
  grass: {
    base: [0.19, 0.39, 0.2],
    secondary: [0.13, 0.28, 0.15],
    accent: [0.33, 0.45, 0.18],
    noiseScale: 6,
    noiseOctaves: 5,
    noisePersistence: 0.5,
    detailScale: 18,
    detailStrength: 0.25,
    macroScale: 1.6,
    macroStrength: 0.25,
    roughnessBase: 0.9,
    roughnessVariance: 0.08,
    heightStrength: 0.6,
    pattern: 'speckle',
  },
  concrete: {
    base: [0.46, 0.46, 0.48],
    secondary: [0.36, 0.37, 0.38],
    accent: [0.52, 0.52, 0.54],
    noiseScale: 10,
    noiseOctaves: 4,
    noisePersistence: 0.55,
    detailScale: 26,
    detailStrength: 0.35,
    macroScale: 2.2,
    macroStrength: 0.2,
    roughnessBase: 0.75,
    roughnessVariance: 0.12,
    heightStrength: 0.45,
    pattern: 'speckle',
  },
  'wood-plank': {
    base: [0.36, 0.24, 0.15],
    secondary: [0.47, 0.31, 0.2],
    accent: [0.28, 0.18, 0.12],
    noiseScale: 4,
    noiseOctaves: 4,
    noisePersistence: 0.6,
    detailScale: 12,
    detailStrength: 0.4,
    macroScale: 1.4,
    macroStrength: 0.3,
    roughnessBase: 0.65,
    roughnessVariance: 0.1,
    heightStrength: 0.55,
    pattern: 'stripe',
  },
  stone: {
    base: [0.34, 0.34, 0.36],
    secondary: [0.24, 0.25, 0.27],
    accent: [0.46, 0.46, 0.48],
    noiseScale: 7,
    noiseOctaves: 5,
    noisePersistence: 0.55,
    detailScale: 16,
    detailStrength: 0.3,
    macroScale: 1.8,
    macroStrength: 0.28,
    roughnessBase: 0.8,
    roughnessVariance: 0.15,
    heightStrength: 0.7,
    pattern: 'stone',
  },
  'metal-panel': {
    base: [0.35, 0.36, 0.38],
    secondary: [0.28, 0.29, 0.31],
    accent: [0.5, 0.5, 0.55],
    noiseScale: 8,
    noiseOctaves: 4,
    noisePersistence: 0.5,
    detailScale: 20,
    detailStrength: 0.2,
    macroScale: 2.4,
    macroStrength: 0.18,
    roughnessBase: 0.55,
    roughnessVariance: 0.08,
    heightStrength: 0.35,
    pattern: 'panel',
  },
}

const textureCache = new Map<string, GroundTextureSet>()

const lerp = (a: number, b: number, t: number) => a + (b - a) * t
const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value))
const smoothstep = (t: number) => t * t * (3 - 2 * t)

const hash2 = (x: number, y: number, seed: number) => {
  const value = Math.sin(x * 127.1 + y * 311.7 + seed * 74.7) * 43758.5453
  return value - Math.floor(value)
}

const valueNoise = (x: number, y: number, seed: number) => {
  const xi = Math.floor(x)
  const yi = Math.floor(y)
  const xf = x - xi
  const yf = y - yi

  const v00 = hash2(xi, yi, seed)
  const v10 = hash2(xi + 1, yi, seed)
  const v01 = hash2(xi, yi + 1, seed)
  const v11 = hash2(xi + 1, yi + 1, seed)

  const u = smoothstep(xf)
  const v = smoothstep(yf)

  const x1 = lerp(v00, v10, u)
  const x2 = lerp(v01, v11, u)

  return lerp(x1, x2, v)
}

const fbm = (x: number, y: number, seed: number, octaves: number, persistence: number) => {
  let value = 0
  let amplitude = 1
  let frequency = 1
  let max = 0

  for (let i = 0; i < octaves; i += 1) {
    value += valueNoise(x * frequency, y * frequency, seed + i * 13.13) * amplitude
    max += amplitude
    amplitude *= persistence
    frequency *= 2
  }

  return value / max
}

const mixColors = (a: [number, number, number], b: [number, number, number], t: number) => ([
  lerp(a[0], b[0], t),
  lerp(a[1], b[1], t),
  lerp(a[2], b[2], t),
] as [number, number, number])

const applyPattern = (preset: GroundTexturePreset, u: number, v: number, noise: number, detail: number) => {
  switch (preset.pattern) {
    case 'stripe': {
      const stripe = Math.sin((u * 10 + noise * 2) * Math.PI * 2) * 0.5 + 0.5
      return clamp(lerp(noise, stripe, 0.6), 0, 1)
    }
    case 'stone': {
      const cracks = Math.abs(Math.sin((u * 8 + v * 6 + detail * 2) * Math.PI)) * 0.7 + 0.3
      return clamp(noise * cracks, 0, 1)
    }
    case 'panel': {
      const panel = (Math.sin(u * 12 * Math.PI) * 0.5 + 0.5) * (Math.sin(v * 12 * Math.PI) * 0.5 + 0.5)
      return clamp(lerp(noise, panel, 0.4), 0, 1)
    }
    case 'speckle':
      return clamp(noise + detail * preset.detailStrength, 0, 1)
    default:
      return clamp(noise, 0, 1)
  }
}

const createDataTexture = (data: Uint8Array, size: number, colorSpace: THREE.ColorSpace) => {
  const texture = new THREE.DataTexture(data as unknown as BufferSource, size, size, THREE.RGBAFormat)
  const textureWithColorSpace = texture as unknown as THREE.Texture & { colorSpace: THREE.ColorSpace }
  textureWithColorSpace.colorSpace = colorSpace
  texture.wrapS = THREE.RepeatWrapping
  texture.wrapT = THREE.RepeatWrapping
  texture.generateMipmaps = true
  texture.minFilter = THREE.LinearMipmapLinearFilter
  texture.magFilter = THREE.LinearFilter
  texture.needsUpdate = true
  return texture
}

const createMacroTexture = (seed: number, size: number) => {
  const data = new Uint8Array(size * size * 4)

  for (let y = 0; y < size; y += 1) {
    for (let x = 0; x < size; x += 1) {
      const u = x / size
      const v = y / size
      const noise = fbm(u * 2, v * 2, seed + 91, 4, 0.55)
      const value = Math.round(clamp(noise, 0, 1) * 255)
      const idx = (y * size + x) * 4
      data[idx] = value
      data[idx + 1] = value
      data[idx + 2] = value
      data[idx + 3] = 255
    }
  }

  return createDataTexture(data, size, THREE.NoColorSpace)
}

const createGroundTextures = (id: GroundTextureId, size: number, macroSize: number): GroundTextureSet => {
  const preset = PRESETS[id]
  const seed = id.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0)
  const heightMap = new Float32Array(size * size)
  const albedoData = new Uint8Array(size * size * 4)
  const roughnessData = new Uint8Array(size * size * 4)
  const aoData = new Uint8Array(size * size * 4)

  for (let y = 0; y < size; y += 1) {
    for (let x = 0; x < size; x += 1) {
      const u = x / size
      const v = y / size

      const noise = fbm(
        u * preset.noiseScale,
        v * preset.noiseScale,
        seed,
        preset.noiseOctaves,
        preset.noisePersistence
      )
      const detail = fbm(u * preset.detailScale, v * preset.detailScale, seed + 37, 3, 0.6)
      const macro = fbm(u * preset.macroScale, v * preset.macroScale, seed + 91, 3, 0.55)
      const pattern = applyPattern(preset, u, v, noise, detail)
      const mixFactor = clamp(pattern * 0.8 + detail * 0.2, 0, 1)

      const baseColor = mixColors(preset.base, preset.secondary, mixFactor)
      const accentMix = preset.accent ? mixColors(baseColor, preset.accent, detail * 0.4) : baseColor
      const macroTint = 1 + (macro - 0.5) * preset.macroStrength
      const color: [number, number, number] = [
        clamp(accentMix[0] * macroTint, 0, 1),
        clamp(accentMix[1] * macroTint, 0, 1),
        clamp(accentMix[2] * macroTint, 0, 1),
      ]

      const height = clamp((pattern * 0.7 + detail * 0.3) * preset.heightStrength, 0, 1)
      const roughness = clamp(
        preset.roughnessBase
          + (noise - 0.5) * preset.roughnessVariance
          + (macro - 0.5) * preset.macroStrength * 0.3,
        0,
        1
      )
      const ao = clamp(0.7 + (detail - 0.5) * 0.2 + (macro - 0.5) * 0.1, 0, 1)

      const idx = (y * size + x) * 4
      albedoData[idx] = Math.round(color[0] * 255)
      albedoData[idx + 1] = Math.round(color[1] * 255)
      albedoData[idx + 2] = Math.round(color[2] * 255)
      albedoData[idx + 3] = 255

      roughnessData[idx] = Math.round(roughness * 255)
      roughnessData[idx + 1] = Math.round(roughness * 255)
      roughnessData[idx + 2] = Math.round(roughness * 255)
      roughnessData[idx + 3] = 255

      aoData[idx] = Math.round(ao * 255)
      aoData[idx + 1] = Math.round(ao * 255)
      aoData[idx + 2] = Math.round(ao * 255)
      aoData[idx + 3] = 255

      heightMap[y * size + x] = height
    }
  }

  const normalData = new Uint8Array(size * size * 4)

  for (let y = 0; y < size; y += 1) {
    for (let x = 0; x < size; x += 1) {
      const left = heightMap[y * size + ((x - 1 + size) % size)] ?? 0
      const right = heightMap[y * size + ((x + 1) % size)] ?? 0
      const down = heightMap[((y - 1 + size) % size) * size + x] ?? 0
      const up = heightMap[((y + 1) % size) * size + x] ?? 0

      const dx = (left - right) * preset.heightStrength
      const dy = (down - up) * preset.heightStrength

      const nx = dx
      const ny = dy
      const nz = 1.0

      const length = Math.sqrt(nx * nx + ny * ny + nz * nz) || 1
      const normalX = nx / length
      const normalY = ny / length
      const normalZ = nz / length

      const idx = (y * size + x) * 4
      normalData[idx] = Math.round((normalX * 0.5 + 0.5) * 255)
      normalData[idx + 1] = Math.round((normalY * 0.5 + 0.5) * 255)
      normalData[idx + 2] = Math.round((normalZ * 0.5 + 0.5) * 255)
      normalData[idx + 3] = 255
    }
  }

  return {
    id,
    albedo: createDataTexture(albedoData, size, THREE.SRGBColorSpace),
    normal: createDataTexture(normalData, size, THREE.NoColorSpace),
    roughness: createDataTexture(roughnessData, size, THREE.NoColorSpace),
    ao: createDataTexture(aoData, size, THREE.NoColorSpace),
    macro: createMacroTexture(seed + 127, macroSize),
  }
}

export const getGroundTextureSet = (
  id: GroundTextureId,
  options?: { resolution?: number; macroResolution?: number }
): GroundTextureSet => {
  const resolution = options?.resolution ?? DEFAULT_RESOLUTION
  const macroResolution = options?.macroResolution ?? DEFAULT_MACRO_RESOLUTION
  const key = `${id}:${resolution}:${macroResolution}`
  const cached = textureCache.get(key)
  if (cached) {
    return cached
  }

  const set = createGroundTextures(id, resolution, macroResolution)
  textureCache.set(key, set)
  return set
}
