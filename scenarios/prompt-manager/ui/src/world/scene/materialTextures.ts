import {
  DataTexture,
  Color,
  LinearFilter,
  LinearMipmapLinearFilter,
  RGBFormat,
  RepeatWrapping,
  RGBAFormat,
  SRGBColorSpace,
  UnsignedByteType,
  Vector2,
  type Texture,
} from 'three'

/**
 * Small, tileable material treatments for the low-poly world palette.  The
 * source kits intentionally contain colour factors instead of raster maps;
 * these maps add scale, grain, and light response without changing their
 * silhouettes or adding an external asset dependency.
 */
export type WorldTextureKind = 'wood' | 'bark' | 'canvas' | 'fabric' | 'metal' | 'leaf' | 'grass' | 'stone' | 'dirt' | 'wall' | 'floor' | 'path' | 'roof'

export interface WorldTextureSet {
  map: DataTexture
  normalMap?: DataTexture
  roughnessMap?: DataTexture
  repeat: number
  normalScale: number
  envMapIntensity: number
}

const SIZE = 96
const cache = new Map<WorldTextureKind, WorldTextureSet>()
const propsCache = new Map<WorldTextureKind, ReturnType<typeof makeTextureProps>>()

function hash(x: number, y: number, seed: number): number {
  const value = Math.sin(x * 127.1 + y * 311.7 + seed * 74.3) * 43758.5453123
  return value - Math.floor(value)
}

function smoothNoise(x: number, y: number, seed: number): number {
  const ix = Math.floor(x), iy = Math.floor(y)
  const fx = x - ix, fy = y - iy
  const ux = fx * fx * (3 - 2 * fx), uy = fy * fy * (3 - 2 * fy)
  const a = hash(ix, iy, seed), b = hash(ix + 1, iy, seed)
  const c = hash(ix, iy + 1, seed), d = hash(ix + 1, iy + 1, seed)
  return (a + (b - a) * ux) + ((c + (d - c) * ux) - (a + (b - a) * ux)) * uy
}

function materialKind(name: string | undefined): WorldTextureKind {
  const value = (name ?? '').toLowerCase()
  if (/bark|woodinner|woodbark|stump/.test(value)) return 'bark'
  if (/wood|timber/.test(value)) return 'wood'
  if (/canvas|tent/.test(value)) return 'canvas'
  if (/carpet|rug|fabric|cloth/.test(value)) return 'fabric'
  if (/metal|chrome|steel|hub/.test(value)) return 'metal'
  if (/leaf|plant|grass|green/.test(value)) return 'leaf'
  if (/stone|rock|concrete/.test(value)) return 'stone'
  if (/dirt|soil|sand/.test(value)) return 'dirt'
  if (/roof|shingle/.test(value)) return 'roof'
  if (/floor|ground/.test(value)) return 'floor'
  if (/path|pavement/.test(value)) return 'path'
  if (/wall|plaster|paint/.test(value)) return 'wall'
  return 'floor'
}

function configure(texture: DataTexture, colorSpace: '' | typeof SRGBColorSpace): DataTexture {
  texture.wrapS = RepeatWrapping
  texture.wrapT = RepeatWrapping
  texture.magFilter = LinearFilter
  texture.minFilter = LinearMipmapLinearFilter
  texture.generateMipmaps = true
  texture.colorSpace = colorSpace
  texture.needsUpdate = true
  return texture
}

function build(kind: WorldTextureKind): WorldTextureSet {
  const albedo = new Uint8Array(SIZE * SIZE * 4)
  const height = new Uint8Array(SIZE * SIZE)
  const roughness = new Uint8Array(SIZE * SIZE * 4)
  const seed = kind.length * 17 + kind.charCodeAt(0)
  for (let y = 0; y < SIZE; y += 1) for (let x = 0; x < SIZE; x += 1) {
    const u = x / SIZE, v = y / SIZE
    const broad = smoothNoise(u * 7, v * 7, seed)
    const fine = smoothNoise(u * 23, v * 23, seed + 9)
    const grain = kind === 'wood' || kind === 'bark' || kind === 'roof'
      ? Math.sin((u * 18 + fine * 1.8) * Math.PI * (kind === 'bark' ? 1.2 : .75)) * .5 + .5
      : kind === 'metal' ? Math.sin((u * 42 + v * 3) * Math.PI) * .5 + .5
        : kind === 'fabric' || kind === 'canvas' ? ((x + y) % 4 < 2 ? .42 : .58)
          : broad
    // Keep the albedo treatment close to neutral so the authored palette stays
    // authoritative after the sRGB-to-linear conversion in the PBR shader.
    const value = Math.max(0, Math.min(255, Math.round(252 + (grain - .5) * 6 + (fine - .5) * 4)))
    const i = y * SIZE + x
    albedo[i * 4] = value
    albedo[i * 4 + 1] = value
    albedo[i * 4 + 2] = value
    albedo[i * 4 + 3] = 255
    height[i] = Math.max(0, Math.min(255, Math.round(128 + (grain - .5) * (kind === 'metal' ? 10 : 38))))
    const rough = kind === 'metal' ? 92 + Math.round(broad * 38)
      : kind === 'fabric' || kind === 'canvas' ? 218 + Math.round(broad * 30)
        : kind === 'grass' || kind === 'dirt' ? 224 + Math.round(broad * 26)
          : 198 + Math.round(broad * 48)
    roughness[i * 4] = rough
    roughness[i * 4 + 1] = rough
    roughness[i * 4 + 2] = rough
    roughness[i * 4 + 3] = 255
  }
  const normal = ['wood', 'bark', 'fabric', 'stone'].includes(kind) ? new Uint8Array(SIZE * SIZE * 3) : undefined
  if (normal) for (let y = 0; y < SIZE; y += 1) for (let x = 0; x < SIZE; x += 1) {
    const left = height[y * SIZE + (x + SIZE - 1) % SIZE] ?? 128
    const right = height[y * SIZE + (x + 1) % SIZE] ?? 128
    const down = height[((y + SIZE - 1) % SIZE) * SIZE + x] ?? 128
    const up = height[((y + 1) % SIZE) * SIZE + x] ?? 128
    const i = (y * SIZE + x) * 3
    normal[i] = Math.max(0, Math.min(255, 128 + (left - right) * .55))
    normal[i + 1] = Math.max(0, Math.min(255, 128 + (down - up) * .55))
    normal[i + 2] = 255
  }
  const map = configure(new DataTexture(albedo, SIZE, SIZE, RGBAFormat, UnsignedByteType), SRGBColorSpace)
  const normalMap = normal ? configure(new DataTexture(normal, SIZE, SIZE, RGBFormat, UnsignedByteType), '') : undefined
  // Keep roughness RGBA, rather than a red-only texture: WebGL1 and several
  // mobile drivers otherwise sample unsupported red formats as zero, turning
  // every affected surface into a mirror.
  const roughnessMap = configure(new DataTexture(roughness, SIZE, SIZE, RGBAFormat, UnsignedByteType), '')
  const normalScale = kind === 'fabric' ? .035 : kind === 'stone' ? .08 : kind === 'wood' ? .1 : kind === 'bark' ? .12 : .06
  // The HDR environment is the sky backdrop and ambient light source. World
  // surfaces use the keyed lights for their highlights so the backdrop can
  // never be mistaken for a loaded material texture.
  const envMapIntensity = 0
  return { map, normalMap, roughnessMap, repeat: kind === 'bark' || kind === 'leaf' ? 3.2 : kind === 'metal' ? 2.6 : 2.1, normalScale, envMapIntensity }
}

export function worldTextureKind(materialName?: string): WorldTextureKind {
  return materialKind(materialName)
}

export function worldTexture(kind: WorldTextureKind): WorldTextureSet {
  const existing = cache.get(kind)
  if (existing) return existing
  const result = build(kind)
  result.map.repeat.set(result.repeat, result.repeat)
  result.normalMap?.repeat.set(result.repeat, result.repeat)
  result.roughnessMap?.repeat.set(result.repeat, result.repeat)
  cache.set(kind, result)
  return result
}

export function applyWorldTexture(material: { map?: Texture | null; normalMap?: Texture | null; roughnessMap?: Texture | null; normalScale?: Vector2; envMapIntensity?: number; color?: Color; needsUpdate: boolean }, kind: WorldTextureKind): void {
  const set = worldTexture(kind)
  material.map = set.map
  material.normalMap = set.normalMap ?? null
  material.roughnessMap = set.roughnessMap ?? null
  material.normalScale?.set(set.normalScale, set.normalScale)
  if (material.envMapIntensity !== undefined) material.envMapIntensity = set.envMapIntensity
  if (kind === 'leaf' && material.color) {
    // The source kit's leaf factors are cyan. Keep each asset's lightness
    // variation, but put the hue in the natural green range so it cannot read
    // as a reflection of the sky.
    const hsl = { h: 0, s: 0, l: 0 }
    material.color.getHSL(hsl)
    material.color.setHSL(.29, Math.min(.72, Math.max(.3, hsl.s * .65)), Math.min(.64, Math.max(.2, hsl.l * .82)))
  }
  if (kind === 'metal' && material.color) {
    // Several kit metal factors are blue-grey. Metals should catch light, not
    // inherit the blue of the outdoor sky as their base colour.
    const hsl = { h: 0, s: 0, l: 0 }
    material.color.getHSL(hsl)
    material.color.setHSL(hsl.h, Math.min(.14, hsl.s * .35), hsl.l)
  }
  material.needsUpdate = true
}

export function worldTextureProps(kind: WorldTextureKind) {
  const cached = propsCache.get(kind)
  if (cached) return cached
  const props = makeTextureProps(kind)
  propsCache.set(kind, props)
  return props
}

function makeTextureProps(kind: WorldTextureKind) {
  const set = worldTexture(kind)
  return { map: set.map, normalMap: set.normalMap, roughnessMap: set.roughnessMap, normalScale: new Vector2(set.normalScale, set.normalScale), envMapIntensity: set.envMapIntensity }
}
