/**
 * Schema for world.tuning.json.
 *
 * Every lever the world uses is declared here with a bound and a description.
 * The description is the documentation: `pnpm world:tuning-docs` renders this
 * schema into docs/reference/configuration.md and a test keeps the two equal.
 *
 * Rules:
 * - Every leaf has `.describe()` naming the unit and the effect.
 * - Every number has a min and a max.
 * - Adding a lever without a description fails `tuning.test.ts`.
 */
import { z } from 'zod'
import { WeatherTuningSchema } from './weather.schema'

const seconds = (min: number, max: number, what: string) =>
  z.number().min(min).max(max).describe(`${what} (seconds)`)
const metres = (min: number, max: number, what: string) =>
  z.number().min(min).max(max).describe(`${what} (metres)`)
const ratio = (what: string) => z.number().min(0).max(1).describe(`${what} (0..1)`)
const degrees = (min: number, max: number, what: string) =>
  z.number().min(min).max(max).describe(`${what} (degrees)`)
const count = (min: number, max: number, what: string) =>
  z.number().int().min(min).max(max).describe(`${what} (count)`)
const hex = (what: string) =>
  z.string().regex(/^#[0-9a-fA-F]{6}$/).describe(`${what} (hex colour)`)

const range = (min: number, max: number, what: string, unit: string) =>
  z
    .object({
      min: z.number().min(min).max(max).describe(`${what} lower bound (${unit})`),
      max: z.number().min(min).max(max).describe(`${what} upper bound (${unit})`),
    })
    .refine((r) => r.min <= r.max, { message: 'min must be <= max' })

export const SimTuningSchema = z.object({
  tickSeconds: seconds(0.01, 1, 'Fixed simulation step; every rule advances by this much'),
  walkSpeed: z.number().min(0.1).max(10).describe('Ground speed of a walking actor (metres per second)'),
  hurrySpeed: z.number().min(0.1).max(12).describe('Ground speed when a run starts and the actor heads to its desk (metres per second)'),
  turnRateRadPerSec: z.number().min(0.5).max(30).describe('How fast an actor turns to face its heading (radians per second)'),
  arriveRadius: metres(0.05, 2, 'Distance from a target at which an actor counts as arrived'),
  accelSeconds: seconds(0.05, 5, 'Time for an actor to ramp between rest and its target speed'),
  gatherLeadSeconds: seconds(0, 3600, 'How long before a scheduled heartbeat a team walks to its table'),
  gatherWindowSeconds: seconds(0, 7200, 'How long a gathered team waits before giving up and going idle'),
  failedAckSeconds: seconds(0, 86400, 'How long an actor stays in Failed before returning to Idle on its own'),
  eventsRing: count(8, 1024, 'How many recent world events the state keeps for the ticker'),
  maxReplansPerTick: count(1, 64, 'Upper bound on A* replans per tick so pathing never blows the tick budget'),
  pathCacheSize: count(16, 4096, 'Number of cached paths kept per world'),
  idle: z.object({
    rollIntervalSeconds: seconds(0.5, 120, 'How often an idle actor rolls for a new idle activity'),
    weights: z.object({
      rest: z.number().min(0).max(100).describe('Weight of resting at home: the desk seat, or the commons for an unassigned agent (relative weight)'),
      wander: z.number().min(0).max(100).describe('Weight of an outing to a random commons spot (relative weight)'),
      socialize: z.number().min(0).max(100).describe('Weight of pairing up with another idle actor on the commons (relative weight)'),
      sit: z.number().min(0).max(100).describe('Weight of taking a free campfire seat (relative weight)'),
    }),
    maxMoversRatio: ratio('Largest share of idle actors allowed to be walking at once'),
    spacing: metres(0.5, 5, 'Minimum distance between an idle actor and every other actor when a commons spot is chosen'),
    spacingAttempts: count(1, 32, 'Random commons spots tried before the spacing rule is given up for that roll'),
    socializeSeconds: range(1, 600, 'Duration of a socializing pair', 'seconds'),
    sitSeconds: range(1, 3600, 'Duration of a sit at the campfire', 'seconds'),
    restSeconds: range(1, 600, 'Duration of standing still before the next roll', 'seconds'),
    socializeGap: metres(0.5, 5, 'Distance between two socializing actors'),
  }),
})

export const LayoutTuningSchema = z.object({
  cellSize: metres(0.1, 2, 'Navigation grid cell size'),
  roomWidth: metres(3, 30, 'Width (x) of a team room'),
  roomDepth: metres(3, 30, 'Depth (z) of a team room'),
  deskPitch: metres(0.8, 5, 'Distance between neighbouring desks along the back wall'),
  deskInset: metres(0.2, 5, 'Distance from the back wall to the desk row'),
  deskSeatOffset: metres(0.2, 3, 'Distance in front of a desk where its owner stands'),
  tableRadius: metres(0.3, 4, 'Radius of the team table prop footprint'),
  tableSeatRadius: metres(0.5, 6, 'Radius of the seat ring around a team table'),
  tableSeats: count(2, 16, 'Seats around a team table'),
  commonsRadius: metres(2, 20, 'Radius of the commons clearing'),
  commonsSeatRadius: metres(0.5, 10, 'Radius of the seat ring around the campfire'),
  commonsSeats: count(2, 24, 'Seats around the campfire'),
  clearingRadius: metres(0, 20, 'No tree spawns within this distance of a room or the hero camera'),
  wallHeight: metres(0, 3, 'Height of the low wall around a room'),
  boardOffset: metres(0, 20, 'Distance from the commons centre to the runs board'),
  outlineRimSamples: count(4, 64, 'Points sampled around the commons rim for the outline the camera frames'),
  siteCandidates: count(16, 4096, 'Seeded candidates scored for each settlement site'),
  siteRadiusMax: metres(10, 300, 'Maximum distance of a team site from the commons'),
  siteSpacing: metres(2, 100, 'Minimum separation between settlement site centres'),
  siteWeightFlat: z.number().min(0).max(10).describe('Buildability weight favouring flat ground (relative weight)'),
  siteWeightDry: z.number().min(0).max(10).describe('Buildability weight favouring ground outside water and shore (relative weight)'),
  siteWeightNear: z.number().min(0).max(10).describe('Buildability weight favouring sites near the commons (relative weight)'),
  siteWeightApart: z.number().min(0).max(10).describe('Buildability weight favouring separation from selected sites (relative weight)'),
  siteRotationSnapRad: z.number().min(0.01).max(Math.PI / 2).describe('Angular increment used to snap generated site rotations (radians)'),
  scatterJitter: ratio('Share of one terrain cell available for decor position jitter'),
  decorSpacingFactor: ratio('Decor spacing as a fraction of tree spacing'),
  decorScale: range(0.1, 4, 'Seeded decor scale', 'multiplier'),
  decorColorJitter: ratio('Maximum seeded per-channel vegetation colour variation'),
  floorplan: z.object({
    corridorWidth: metres(1, 10, 'Primary and secondary corridor width'),
    secondaryCorridors: range(0, 8, 'Secondary corridor count', 'count'),
    splitRatio: range(0.25, 0.75, 'Seeded room split ratio', 'ratio'),
    maxAspect: z.number().min(1).max(6).describe('Maximum room aspect ratio (ratio)'),
    roomAreaPerMember: z.number().min(2).max(50).describe('Target room area per team member (square metres)'),
    roomMinArea: z.number().min(8).max(200).describe('Minimum room area (square metres)'),
    plateMargin: metres(1, 30, 'Floorplate margin around rooms'),
    doorWidth: metres(0.8, 4, 'Room doorway width'),
    lobbyRadius: metres(1, 15, 'Lobby gathering radius'),
    plateAspect: range(1, 3, 'Seeded office floorplate aspect ratio', 'ratio'),
    primaryOffset: ratio('Maximum seeded primary-corridor offset as a fraction of corridor width'),
    secondaryJitter: ratio('Maximum seeded secondary-corridor jitter within its even spacing'),
  }),
  interior: z.object({
    tableMinMembers: count(1, 100, 'Minimum team size for a meeting table'),
    fillerMax: count(0, 3, 'Maximum seeded filler props per room'),
  }),
})

export const TerrainTuningSchema = z.object({
  radius: metres(10, 500, 'Radius of the generated terrain field'),
  cellSize: metres(0.25, 8, 'Spacing between terrain field samples'),
  amplitude: metres(0, 20, 'Maximum absolute terrain elevation'),
  frequency: z.number().min(0.001).max(1).describe('Base terrain noise frequency (cycles per metre)'),
  detailAmplitude: metres(0, 5, 'Higher-frequency surface-detail amplitude'),
  detailFrequency: z.number().min(0.001).max(2).describe('Surface-detail noise frequency (cycles per metre)'),
  octaves: count(1, 8, 'Fractal noise octaves used for height and moisture'),
  lacunarity: z.number().min(1).max(4).describe('Frequency multiplier between terrain noise octaves (multiplier)'),
  gain: ratio('Amplitude multiplier between terrain noise octaves'),
  moistureFrequency: z.number().min(0.001).max(1).describe('Base moisture noise frequency (cycles per metre)'),
  moistureWarp: metres(0, 50, 'Domain-warp distance applied to moisture sampling'),
  falloffStart: ratio('Fraction of terrain radius where elevation begins fading to zero'),
  waterLevel: metres(-20, 20, 'Water surface elevation'),
  shoreMargin: metres(0, 20, 'Dry navigation margin around water'),
  waterSurfaceLift: metres(0, 0.2, 'Water surface lift above its terrain threshold'),
  wetShoreWidth: metres(0.1, 10, 'Width of terrain darkening immediately inside water'),
  wetShoreDarkening: ratio('Maximum terrain darkening immediately inside water'),
  maxSiteSlope: z.number().min(0).max(Math.PI / 2).describe('Steepest ground eligible for a team site (radians)'),
  maxWalkSlope: z.number().min(0).max(Math.PI / 2).describe('Steepest ground eligible for navigation (radians)'),
  kerbWidth: metres(0.25, 10, 'Width over which a level site pad blends into terrain'),
  pathWidth: metres(0.25, 10, 'Width of paths painted into terrain colour'),
  innerCellSize: metres(0.25, 4, 'Terrain mesh spacing near the settlement'),
  innerRadius: metres(5, 300, 'Radius of the dense inner terrain mesh'),
  ringFalloff: z.number().min(1).max(8).describe('Terrain mesh spacing multiplier beyond the inner ring (multiplier)'),
  tileSize: metres(4, 100, 'Side length of one vegetation culling tile'),
  moistureBasinDepth: metres(0, 5, 'Maximum moisture bias subtracted when classifying water'),
  shoreMinGrade: z.number().min(0.001).max(1).describe('Minimum grade used to estimate horizontal shore distance (rise over run)'),
  padClearance: metres(0, 5, 'Minimum terrace elevation above the configured water surface'),
  siteLevelTolerance: metres(0.001, 1, 'Maximum elevation variation allowed across a site pad'),
})
export const TerrainOverrideSchema = TerrainTuningSchema.partial()

export const CameraTuningSchema = z.object({
  fov: degrees(10, 90, 'Vertical field of view'),
  near: metres(0.01, 10, 'Near clip plane'),
  far: metres(10, 2000, 'Far clip plane'),
  polarMinDeg: degrees(0, 89, 'Steepest allowed camera angle from straight above'),
  polarMaxDeg: degrees(1, 90, 'Shallowest allowed camera angle from straight above'),
  azimuthRangeDeg: degrees(0, 180, 'Orbit allowed either side of the hero azimuth'),
  minDistance: metres(1, 100, 'Closest dolly distance'),
  maxDistance: metres(2, 500, 'Farthest dolly distance'),
  introSeconds: seconds(0, 10, 'Length of the establishing-to-hero dolly on load'),
  smoothTime: seconds(0.01, 3, 'Camera-controls smoothing time for every move'),
  frameFill: ratio('Share of the viewport the layout outline fills at distanceFactor 1; poses scale from this'),
  focusPadding: z.number().min(1).max(5).describe('fitToBox padding multiplier when focusing an actor (multiplier)'),
  focusDistance: metres(1, 50, 'Dolly distance after focusing an actor'),
  minClearance: metres(0, 20, 'The first frame must have no geometry closer than this to the camera'),
  keyOrbitDegPerSec: degrees(1, 360, 'Orbit speed for keyboard arrows'),
  keyDollyPerSec: z.number().min(0.1).max(100).describe('Dolly speed for keyboard +/- (metres per second)'),
})

export const PeriodSchema = z.object({
  exposure: z.number().min(0).max(4).describe('Tone-mapping exposure (multiplier)'),
  envIntensity: z.number().min(0).max(4).describe('Environment map intensity (multiplier)'),
  keyIntensity: z.number().min(0).max(20).describe('Directional key light intensity (physical units)'),
  keyColor: hex('Key light colour'),
  ambientIntensity: z.number().min(0).max(4).describe('Hemisphere ambient intensity (multiplier)'),
  fogColor: hex('Height fog colour'),
  fogNear: z.number().min(0).max(10).describe('Fog starts at this multiple of the slab fit distance from the camera (multiplier)'),
  fogFar: z.number().min(0.1).max(20).describe('Fog is complete at this multiple of the slab fit distance (multiplier)'),
  skyIntensity: z.number().min(0).max(4).describe('Brightness of the HDRI sky drawn behind the diorama (multiplier)'),
  skyBlur: ratio('Blur applied to the HDRI sky background; 0 is sharp, 1 is a soft gradient'),
  sunElevationDeg: degrees(-30, 90, 'Sun elevation used to tilt the key light for the period'),
  lampEmissive: z.number().min(0).max(10).describe('Emissive intensity of lamps and the campfire; above 1 blooms (multiplier)'),
  backgroundColor: hex('Canvas clear colour behind the sky'),
})

const hourBand = z.object({
  from: z.number().min(0).max(24).describe('Band start (hour of day)'),
  to: z.number().min(0).max(24).describe('Band end (hour of day)'),
})

export const LightingTuningSchema = z.object({
  keyLight: z.object({
    elevationDeg: degrees(0, 90, 'Key light elevation above the slab'),
    azimuthDeg: degrees(-180, 180, 'Key light azimuth around the slab'),
    shadowBias: z.number().min(-0.01).max(0.01).describe('Shadow map depth bias (depth units)'),
    shadowNormalBias: z.number().min(0).max(0.5).describe('Shadow map normal bias (metres)'),
  }),
  periodHours: z.object({
    dawn: hourBand,
    day: hourBand,
    dusk: hourBand,
    night: hourBand,
  }),
  periods: z.object({
    dawn: PeriodSchema,
    day: PeriodSchema,
    dusk: PeriodSchema,
    night: PeriodSchema,
  }),
})

export const LabelsTuningSchema = z.object({
  budget: count(0, 200, 'Maximum labels drawn at once before clustering'),
  collapseDistance: metres(1, 500, 'Camera distance past which room labels collapse into one count label'),
  fontSize: z.number().min(0.05).max(2).describe('SDF label height in world units (metres)'),
  offsetY: metres(0, 5, 'Label height above the actor origin'),
  roomOffsetY: metres(0, 8, 'Static height for clustered room labels above props'),
  minScreenPx: z.number().min(4).max(64).describe('Labels never render smaller than this on screen (pixels)'),
  maxScreenPx: z.number().min(4).max(128).describe('Labels never render larger than this on screen (pixels)'),
  paddingPx: z.number().min(0).max(32).describe('Collision padding around a projected label (pixels)'),
})

export const ActorTuningSchema = z.object({
  bodyRadius: metres(0.1, 2, 'Slime body radius at rest'),
  breathAmplitude: z.number().min(0).max(0.5).describe('Idle breathing scale swing (scale units)'),
  breathHz: z.number().min(0.05).max(5).describe('Idle breathing rate (hertz)'),
  hopHeight: metres(0, 2, 'Peak height of a locomotion hop'),
  hopHz: z.number().min(0.2).max(10).describe('Hops per second while walking (hertz)'),
  squashOnLand: z.number().min(0.3).max(1).describe('Vertical scale at the moment of landing (scale units)'),
  squashRecoverPerSec: z.number().min(0.5).max(60).describe('How fast the landing squash relaxes back to 1 (per second)'),
  wobbleIntensity: z.number().min(0).max(0.5).describe('Vertex wobble amplitude from the slime shader (metres)'),
  blinkIntervalSeconds: range(0.2, 60, 'Interval between blinks', 'seconds'),
  blinkSeconds: seconds(0.02, 1, 'Length of one blink'),
  emoteSeconds: seconds(0.2, 10, 'How long an emote burst stays visible'),
  seatedScale: z.number().min(0.5).max(1).describe('Body scale while seated (scale units)'),
  equipmentTiers: z
    .array(z.number().int().min(0).max(1000))
    .length(5)
    .describe('Skill counts at which equipment upgrades: none, paper, folder, briefcase, backpack (counts)'),
  look: z.object({
    minDetailPx: z.number().min(0).max(128).describe('Projected body height below which face and equipment detail is culled (pixels)'),
    bodySquashY: ratio('Resting vertical scale of the slime body sphere; below 1 makes a blob'),
    eyeRadius: ratio('Eye radius as a fraction of the body radius'),
    eyeSpacing: ratio('Half distance between the eyes as a fraction of the body radius'),
    eyeHeight: ratio('Eye height above the body centre as a fraction of the body radius'),
    eyeForward: ratio('Eye distance forward of the body centre as a fraction of the body radius'),
    blinkScaleY: ratio('Vertical eye scale while blinking'),
    mouthWidth: ratio('Mouth width as a fraction of the body radius'),
    mouthHeight: ratio('Mouth height as a fraction of the body radius'),
    mouthForward: ratio('Mouth distance forward of the body centre as a fraction of the body radius'),
    mouthDrop: ratio('Mouth height below the eye line as a fraction of the body radius'),
    earSize: ratio('Ear nub size as a fraction of the body radius'),
    earHeight: ratio('Ear height above the body centre as a fraction of the body radius'),
    earSpread: ratio('Ear sideways offset as a fraction of the body radius'),
    equipmentScale: ratio('Equipment prop size as a fraction of the body radius per tier'),
    equipmentBack: ratio('Equipment distance behind the body centre as a fraction of the body radius'),
    equipmentHeight: ratio('Equipment height above the body centre as a fraction of the body radius'),
    markerHeight: z.number().min(0).max(5).describe('Status marker height above the actor origin (metres)'),
    markerRadius: metres(0.02, 1, 'Status marker ring radius'),
    emoteRise: metres(0, 3, 'How far an emote sprite rises over its lifetime'),
    emoteSize: metres(0.05, 2, 'Emote sprite size'),
    emoteHeight: metres(0, 5, 'Emote start height above the actor origin'),
    messageTtlSeconds: seconds(1, 120, 'How long a speech bubble stays readable'),
  }),
})

export const QualityProfileSchema = z.object({
  dpr: z.number().min(0.5).max(3).describe('Device pixel ratio cap (multiplier)'),
  shadows: z.boolean().describe('Directional shadow map on or off (flag)'),
  shadowMapSize: count(256, 8192, 'Shadow map resolution (pixels, square)'),
  shadowRefreshHz: count(0, 60, 'Maximum shadow-map refreshes per second while actors move'),
  ao: z.boolean().describe('N8AO ambient occlusion pass on or off (flag)'),
  aoQuality: z.enum(['off', 'low', 'medium']).describe('Ambient-occlusion quality; off is the mount switch'),
  bloom: z.boolean().describe('Selective bloom pass on or off (flag)'),
  msaa: count(0, 8, 'Multisample anti-aliasing samples on the composer target; 0 disables (samples)'),
  labelBudget: count(0, 200, 'Label budget override for this profile'),
  frameCapFps: count(15, 240, 'Frame rate this profile is designed for; the governor derives its degraded threshold from it'),
  wobble: z.boolean().describe('Slime vertex wobble on or off (flag)'),
  clouds: z.boolean().describe('Volumetric clouds on or off (flag)'),
  terrainInnerRadius: metres(5, 500, 'Radius rendered at the terrain profile base resolution'),
  terrainCellScale: z.number().min(0.5).max(8).describe('Terrain sample spacing multiplier (multiplier)'),
  vegetationDensityScale: ratio('Share of deterministic vegetation instances rendered'),
  weatherParticleScale: ratio('Share of weather particles rendered'),
  waterEnabled: z.boolean().describe('Whether the water surface is rendered (flag)'),
  vegetationInstanceBudget: count(0, 5000, 'Maximum visible vegetation instances rendered at once'),
})

export const QualityTuningSchema = z.object({
  defaultProfile: z.enum(['low', 'medium', 'high', 'ultra']).describe('Profile used before the governor or the user picks one (profile id)'),
  degradedRatio: ratio('Fraction of frameCapFps below which the governor steps down while auto is on'),
  recoverRatio: ratio('Fraction of frameCapFps above which the governor steps up while auto is on'),
  monitorFlipflops: count(1, 20, 'Consecutive up/down flips after which the governor stops adjusting'),
  profiles: z.object({
    low: QualityProfileSchema,
    medium: QualityProfileSchema,
    high: QualityProfileSchema,
    ultra: QualityProfileSchema,
  }),
})

export const DataTuningSchema = z.object({
  pollIntervalMs: z.number().min(500).max(120000).describe('Fallback poll cadence when the feed stream is unavailable (milliseconds)'),
  fallbackAfterMs: z.number().min(100).max(120000).describe('How long a silent stream is tolerated before polling starts (milliseconds)'),
  reconnectBaseMs: z.number().min(100).max(60000).describe('First reconnect delay after a stream error (milliseconds)'),
  reconnectMaxMs: z.number().min(100).max(600000).describe('Reconnect backoff ceiling (milliseconds)'),
  snapshotStaleMs: z.number().min(1000).max(600000).describe('A snapshot older than this is treated as stale by the HUD (milliseconds)'),
})

export const EditorTuningSchema = z.object({
  snap: metres(0.05, 5, 'Drag snapping grid in edit mode'),
  maxHistory: count(1, 500, 'Undo history depth'),
  saveDebounceMs: z.number().min(0).max(30000).describe('Debounce before an edit is persisted (milliseconds)'),
  aerialPolarDeg: degrees(0, 89, 'Camera angle from straight above while editing'),
})

export const SceneBudgetSchema = z.object({
  drawCalls: count(1, 5000, 'Maximum renderer draw calls per frame; a ceiling from the layer design (one instanced draw per slab kind, four for actors, one per prop material part, one per pooled label, post passes), never a reading copied from a run'),
  triangles: count(1, 50000000, 'Maximum triangles per frame'),
  p95Ms: z.number().min(1).max(200).describe('Maximum p95 frame time on the reference machine; one frame of jitter above the 60 Hz vsync is allowed (milliseconds)'),
  provenance: z.object({
    actors: count(1, 100000, 'Pinned synthetic actor count used for calibration'),
    gpu: z.string().min(1).describe('GPU renderer string used for calibration (renderer name)'),
    renderer: z.string().min(1).describe('Exact hardware renderer string reported by WebGL'),
    gpuTier: z.enum(['igpu', 'dgpu']).describe('Hardware tier used for the gating calibration'),
    deviceScaleFactor: z.number().min(0.5).max(3).describe('Device pixel ratio used for calibration'),
    measuredP95Ms: z.number().positive().describe('Observed GPU p95 before headroom was applied'),
    target: z.boolean().describe('True when p95Ms is a delivery target rather than observed-plus-headroom'),
    calibratedAt: z.string().min(1).describe('Calibration date (ISO 8601 date)'),
    method: z.string().min(1).describe('Frame-time measurement method (method name)'),
  }),
})

const profileBudgets = z.object({
  low: SceneBudgetSchema,
  medium: SceneBudgetSchema,
  high: SceneBudgetSchema,
  ultra: SceneBudgetSchema,
})

export const BudgetsTuningSchema = z.object({
  goldenThreshold: ratio('Fraction of pixels allowed to differ from a golden before the smoke tool fails'),
  propTriangles: count(1, 100000, 'Maximum triangles for one baked prop GLB'),
  actorDrawCalls: count(1, 100, 'Draw calls the actor layer may add on top of the set'),
  emptyStageDrawCalls: count(1, 500, 'Draw calls allowed for the empty slab and environment'),
  periodPixelDelta: ratio('Minimum fraction of pixels that must differ between day and night goldens'),
  framing: z.object({
    minFill: ratio('Smallest share of the viewport the layout outline may occupy on the hero pose'),
    maxFill: ratio('Largest share of the viewport the layout outline may occupy on the hero pose'),
  }),
  scenes: z.object({
    park: profileBudgets,
    office: profileBudgets,
  }),
})

export const WorldTuningSchema = z.object({
  version: z.literal(1).describe('Tuning file format version (integer)'),
  sim: SimTuningSchema,
  layout: LayoutTuningSchema,
  terrain: TerrainTuningSchema,
  camera: CameraTuningSchema,
  lighting: LightingTuningSchema,
  weather: WeatherTuningSchema,
  labels: LabelsTuningSchema,
  actor: ActorTuningSchema,
  quality: QualityTuningSchema,
  data: DataTuningSchema,
  editor: EditorTuningSchema,
  budgets: BudgetsTuningSchema,
})

export type WorldTuning = z.infer<typeof WorldTuningSchema>
export type SimTuning = z.infer<typeof SimTuningSchema>
export type LayoutTuning = z.infer<typeof LayoutTuningSchema>
export type TerrainTuning = z.infer<typeof TerrainTuningSchema>
export type CameraTuning = z.infer<typeof CameraTuningSchema>
export type LightingTuning = z.infer<typeof LightingTuningSchema>
export type WeatherTuning = z.infer<typeof WeatherTuningSchema>
export type LightingPeriod = z.infer<typeof PeriodSchema>
export type LabelsTuning = z.infer<typeof LabelsTuningSchema>
export type ActorTuning = z.infer<typeof ActorTuningSchema>
export type QualityTuning = z.infer<typeof QualityTuningSchema>
export type QualityProfile = z.infer<typeof QualityProfileSchema>
export type DataTuning = z.infer<typeof DataTuningSchema>
export type EditorTuning = z.infer<typeof EditorTuningSchema>
export type BudgetsTuning = z.infer<typeof BudgetsTuningSchema>
export type SceneBudget = z.infer<typeof SceneBudgetSchema>

export type QualityProfileId = keyof WorldTuning['quality']['profiles']
export type PeriodId = keyof WorldTuning['lighting']['periods']
export type SceneId = keyof WorldTuning['budgets']['scenes']

export const QUALITY_PROFILE_IDS: readonly QualityProfileId[] = ['low', 'medium', 'high', 'ultra']
export const PERIOD_IDS: readonly PeriodId[] = ['dawn', 'day', 'dusk', 'night']
export const SCENE_IDS: readonly SceneId[] = ['park', 'office']
