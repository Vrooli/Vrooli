import parkRaw from './scenes/park.json'
import officeRaw from './scenes/office.json'
import { SceneSchema, type Scene } from './scenes.schema'
import { tuning } from './tuning'
import type { LightingPeriod, PeriodId, SceneId, TerrainTuning, WorldTuning } from './tuning.schema'
import { centreWeight, type CentreRegion } from './regions'

function parseScene(raw: unknown, name: string): Scene {
  const result = SceneSchema.safeParse(raw)
  if (!result.success) {
    const issues = result.error.issues.map((i) => `${i.path.join('.')}: ${i.message}`).join('\n')
    throw new Error(`scenes/${name}.json is invalid:\n${issues}`)
  }
  return result.data
}

export const scenes: Record<SceneId, Scene> = {
  park: parseScene(parkRaw, 'park'),
  office: parseScene(officeRaw, 'office'),
}

export function isSceneId(value: string | null | undefined): value is SceneId {
  return value === 'park' || value === 'office'
}

/** The lighting period for a scene: the global preset with the scene's overrides applied. */
export function resolvePeriod(scene: Scene, period: PeriodId, base: WorldTuning = tuning): LightingPeriod {
  const global = base.lighting.periods[period]
  const override = scene.lighting?.periods?.[period]
  return override ? { ...global, ...override } : global
}

export interface TerrainResolver {
  at(x: number, z: number): TerrainTuning
  base(): TerrainTuning
}

export function uniformTerrain(value: TerrainTuning): TerrainResolver {
  return { at: () => value, base: () => value }
}

/** Preserve position-aware sampling when a consumer overrides one terrain policy. */
export function overrideTerrain(source: TerrainResolver, override: Partial<TerrainTuning>): TerrainResolver {
  const cache = new WeakMap<TerrainTuning, TerrainTuning>()
  const merge = (value: TerrainTuning) => {
    let result = cache.get(value)
    if (!result) {
      result = { ...value, ...override }
      cache.set(value, result)
    }
    return result
  }
  return { at: (x, z) => merge(source.at(x, z)), base: () => merge(source.base()) }
}

export function resolveTerrain(scene: Scene, base: Pick<WorldTuning, 'terrain'> = tuning, region?: CentreRegion): TerrainResolver {
  const global = scene.terrain ? { ...base.terrain, ...scene.terrain } : base.terrain
  const override = scene.centre?.terrain
  if (!region || !override) return uniformTerrain(global)
  const centre = { ...global, ...override }
  const keys = Object.keys(override) as Array<keyof TerrainTuning>
  return {
    base: () => global,
    at: (x, z) => {
      const weight = centreWeight(region, x, z)
      if (weight === 0) return global
      if (weight === 1) return centre
      const local = { ...global }
      for (const key of keys) local[key] = global[key] + (centre[key] - global[key]) * weight
      return local
    },
  }
}
