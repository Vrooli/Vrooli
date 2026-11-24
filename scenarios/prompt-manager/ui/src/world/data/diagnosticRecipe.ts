import { isTimeZone } from '../config/clock'
import { z } from 'zod'
import { WorldTuningSchema, SCENE_IDS, PERIOD_IDS, QUALITY_PROFILE_IDS, WeatherIdSchema, scenes, biomeSets } from '../config'
import { canonical, WORLD_GENERATOR_VERSION } from '../sim/recipe'
import { hashString } from '../sim/rng'
import { syntheticRoster } from './synthetic'
import { BuildProvenanceSchema, readBuildProvenance } from '../config/buildProvenance'

const coordinate = z.number().min(-100000).max(100000)
const point = z.tuple([coordinate, coordinate])
const position = z.tuple([coordinate, coordinate, coordinate])
const syntheticPlace = z.string().regex(/^(?:room:demo-team-\d+|gathering|hearth)$/)

/** Only this schema may cross the diagnostic download/import boundary. */
export const DiagnosticRecipeSchema = z.object({
  schemaVersion: z.literal(1),
  source: z.object({
    generatorVersion: z.string().max(80),
    environmentFingerprint: z.string().regex(/^[0-9a-f]{8}$/),
    build: BuildProvenanceSchema.nullable().default(null),
  }).strict(),
  seed: z.number().int().min(0).max(0xffffffff),
  scene: z.enum(SCENE_IDS),
  roster: z.object({ kind: z.literal('synthetic'), actors: z.number().int().min(1).max(10000), perTeam: z.number().int().min(1).max(10000) }).strict(),
  tuning: WorldTuningSchema,
  layout: z.array(z.object({ placeId: syntheticPlace, position: point.optional(), rotation: z.number().optional(), removed: z.boolean().optional() }).strict()).max(10000),
  clearPoints: z.array(point).max(100),
  view: z.object({
    period: z.enum(PERIOD_IDS), weather: WeatherIdSchema,
    profile: z.enum(QUALITY_PROFILE_IDS), zoomTarget: z.enum(['cursor', 'center']),
    ambientLife: z.boolean().default(true),
    lightingMode: z.enum(['clock', 'preset']).default('preset'),
    clock: z.object({ mode: z.literal('fixed'), utcMilliseconds: z.number().min(-1e15).max(1e15),
      timeZone: z.string().min(1).max(100).refine(isTimeZone, 'Unknown civil timezone').optional(),
    }).strict().nullable().default(null),
    navigationOverlay: z.boolean().default(false),
    biomeOverlay: z.boolean().default(false),
    habitatOverlay: z.boolean().default(false),
    cameraCollisionOverlay: z.boolean().default(false),
    camera: z.object({ position, target: position }).strict(),
  }).strict().refine(view => [view.navigationOverlay, view.biomeOverlay, view.habitatOverlay].filter(Boolean).length <= 1, 'Choose only one grid overlay.'),
  renderer: z.object({
    vendor: z.string().max(256).nullable(), renderer: z.string().max(512).nullable(),
    width: z.number().int().min(1).max(32768), height: z.number().int().min(1).max(32768),
    dpr: z.number().positive().max(8),
  }).strict(),
}).strict()

export type DiagnosticRecipe = z.infer<typeof DiagnosticRecipeSchema>
export type DiagnosticRecipeInput = Omit<DiagnosticRecipe, 'schemaVersion' | 'source'>
export const RECIPE_SESSION_KEY = 'world.synthetic-recipe.v1'

export function parseRecipeFile(text: string) {
  if (new TextEncoder().encode(text).byteLength > 1024 * 1024) throw new Error('Recipe files must be at most 1 MiB.')
  return restoreDiagnosticRecipe(JSON.parse(text)).recipe
}

export function recipeLocation(recipe: DiagnosticRecipe) {
  return `/world?${new URLSearchParams({ recipe: 'session', workbench: '1', scene: recipe.scene,
    seed: String(recipe.seed), actors: String(recipe.roster.actors), profile: recipe.view.profile,
    period: recipe.view.period, weather: recipe.view.weather })}`
}

function environmentFingerprint(scene: DiagnosticRecipe['scene']) {
  return hashString(canonical({ scene: scenes[scene], biomes: biomeSets[scenes[scene].biomeSet] })).toString(16).padStart(8, '0')
}

/** Explicit projection: callers cannot accidentally export their extra state. */
export function createDiagnosticRecipe(input: DiagnosticRecipeInput): DiagnosticRecipe {
  return DiagnosticRecipeSchema.parse({
    schemaVersion: 1,
    source: { generatorVersion: WORLD_GENERATOR_VERSION, environmentFingerprint: environmentFingerprint(input.scene), build: readBuildProvenance() },
    seed: input.seed, scene: input.scene,
    roster: { kind: 'synthetic', actors: input.roster.actors, perTeam: input.roster.perTeam },
    tuning: WorldTuningSchema.parse(input.tuning),
    layout: input.layout.map(item => ({ placeId: item.placeId, position: item.position, rotation: item.rotation, removed: item.removed })),
    clearPoints: input.clearPoints,
    view: {
      period: input.view.period, weather: input.view.weather, profile: input.view.profile, zoomTarget: input.view.zoomTarget,
      ambientLife: input.view.ambientLife,
      lightingMode: input.view.lightingMode,
      clock: input.view.clock,
      navigationOverlay: input.view.navigationOverlay,
      biomeOverlay: input.view.biomeOverlay,
      habitatOverlay: input.view.habitatOverlay,
      cameraCollisionOverlay: input.view.cameraCollisionOverlay,
      camera: { position: input.view.camera.position, target: input.view.camera.target },
    },
    renderer: { vendor: input.renderer.vendor, renderer: input.renderer.renderer, width: input.renderer.width, height: input.renderer.height, dpr: input.renderer.dpr },
  })
}

/** Restore only compatible generation inputs, with a fixed synthetic clock. */
export function restoreDiagnosticRecipe(value: unknown) {
  const recipe = DiagnosticRecipeSchema.parse(value)
  if (recipe.source.generatorVersion !== WORLD_GENERATOR_VERSION || recipe.source.environmentFingerprint !== environmentFingerprint(recipe.scene)) {
    throw new Error('This recipe uses a different world generator or environment definition.')
  }
  return {
    recipe,
    input: { seed: recipe.seed, now: 0, scene: recipe.scene,
      ...syntheticRoster(recipe.roster.actors, recipe.roster.perTeam, recipe.seed),
      overrides: recipe.layout, clearPoints: recipe.clearPoints },
    tuning: recipe.tuning,
  }
}
