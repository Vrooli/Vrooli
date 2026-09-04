import parkRaw from './scenes/park.json'
import officeRaw from './scenes/office.json'
import { SceneSchema, type Scene } from './scenes.schema'
import { tuning } from './tuning'
import type { LightingPeriod, PeriodId, SceneId, TerrainTuning, WorldTuning } from './tuning.schema'

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

export function resolveTerrain(scene: Scene, base: Pick<WorldTuning, 'terrain'> = tuning): TerrainTuning {
  return scene.terrain ? { ...base.terrain, ...scene.terrain } : base.terrain
}
