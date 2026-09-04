/**
 * config layer — the world's control surface.
 *
 * Imports nothing from the other world layers. Everything numeric the world
 * uses comes from here; sim, engine, scene and hud read it, never define it.
 */
export { tuning, parseTuning, withTuningOverride, type TuningOverride } from './tuning'
export {
  WorldTuningSchema,
  QUALITY_PROFILE_IDS,
  PERIOD_IDS,
  SCENE_IDS,
  type WorldTuning,
  type SimTuning,
  type LayoutTuning,
  type TerrainTuning,
  type CameraTuning,
  type LightingTuning,
  type WeatherTuning,
  type LightingPeriod,
  type LabelsTuning,
  type ActorTuning,
  type QualityTuning,
  type QualityProfile,
  type QualityProfileId,
  type DataTuning,
  type EditorTuning,
  type BudgetsTuning,
  type SceneBudget,
  type PeriodId,
  type SceneId,
} from './tuning.schema'
export { scenes, isSceneId, resolvePeriod } from './scenes'
export { SceneSchema, type Scene, type CameraPose } from './scenes.schema'
export { BiomeSchema, BiomeSetSchema, BiomeSetsSchema, type Biome, type BiomeSet } from './biomes.schema'
export { biomeSets } from './biomes'
export { periodForHour, isPeriodId } from './periods'
export { WeatherIdSchema, WeatherPresetSchema, WeatherTuningSchema, type WeatherId, type WeatherPreset } from './weather.schema'
export { isQualityProfileId, type QualityState } from './quality'
