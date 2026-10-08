/**
 * Live renderer diagnostics. Written by the in-canvas probe every frame,
 * read by the overlay (throttled) and by the smoke tool through
 * `window.__worldDiagnostics`.
 */
import type { PeriodId, QualityProfileId, SceneId, WeatherId } from '../../config'
import { tuning } from '../../config'
import type { WebGLProbeResult } from '../webgl'
import type { QualityVerdictRecord } from '../quality/governor'
import { preparedAssets } from '../assets/cache'
import { ambientMeasurements } from './ambient'
import type { CameraCommandEvent, CameraOwnership } from '../camera/controller'
import type { ObstacleSweepSample } from '../camera/obstacles'
import type { PoseRouteStatus } from '../camera/poseRoute'
import type { NavigationState } from '../../config/navigation'

export interface WorldDiagnostics {
  cameraNavigation: (NavigationState & { position: number[]; target: number[]; playerPosition: number[] | null; lensZoom: number; inputSequence: number; inputAction: 'pan' | 'dolly' | null }) | null
  beginInteractionSample: () => void
  endInteractionSample: () => InteractionSample | null
  ambientCosts: () => ReturnType<typeof ambientMeasurements.snapshot>
  resetAmbientCosts: () => void
  cameraPoseRoute: (PoseRouteStatus & { owner: 'overview' | 'intro' | 'focus' }) | null
  cameraObstacleGuard: ObstacleSweepSample | null
  cameraZoomGuard: { travel: number | null; limited: boolean; queryMs: number } | null
  cameraOwnership: CameraOwnership | null
  cameraHistory: CameraCommandEvent[]
  webgl: WebGLProbeResult | null
  /** Run expensive scene-graph, raycast, and framing measurements on demand. */
  measure: () => void
  terrainPreparations: number
  terrainMeshMode: 'worker' | 'fallback' | 'cache' | null
  terrainMeshCache: { entries: number; estimatedBytes: number; budgetBytes: number; hits: number; misses: number; evictions: number } | null
  ready: boolean
  presented: boolean
  presentationEpoch: number
  epochReadyAt: number | null
  /** First readiness event since navigation, in performance.now() milliseconds. */
  firstReadyAt: number | null
  preparedAssets: ReturnType<typeof preparedAssets.stats>
  /** Legacy configured performance floor; it does not gate presentation readiness. */
  minimumReadyFps: number
  assetsLoaded: boolean
  introDone: boolean
  /** Rendered frames in the most recent wall-clock second. */
  framesRendered: number
  /** Monotonic rendered-frame counter for capture settling; never an FPS gauge. */
  totalFrames: number
  scene: SceneId
  profile: QualityProfileId
  auto: boolean
  period: PeriodId
  weather: WeatherId
  weatherPressure: number
  drawCalls: number
  triangles: number
  programs: number
  geometries: number
  textures: number
  frameMsP50: number
  frameMsP95: number
  gpuMsP50: number
  gpuMsP95: number
  gpuSamples: number
  gpuTimerReason: string
  passMs: { shadow: number; main: number; post: number; total: number }
  toneMapping: string
  ao: boolean
  bloom: boolean
  dpr: number
  msaa: number
  cameraPosition: [number, number, number]
  cameraTarget: [number, number, number]
  /** Distance from the camera to the nearest geometry along its view axis; -1 when nothing is hit. */
  nearestHit: number
  /** Share of the viewport the layout footprint occupies on its tighter axis (1 touches the edge). */
  footprintFill: number
  /** The extent the camera rig frames (metres), so evidence records what the fill was measured against. */
  footprint: { width: number; depth: number; center: [number, number] }
  gpu: string
  /** Direct scene groups plus explicit shadow/post pass attribution. */
  groupCosts: Array<{ name: string; calls: number; triangles: number }>
  drawCallsUnattributed: number
  trianglesUnattributed: number
  shadowRefreshes: number
  vegetationCullRuns: number
  vegetationCullSkips: number
  lampLightsMounted: number
  lampLightSelections: number
  qualityHistory: QualityVerdictRecord[]
  /** Top-level scene groups with child counts and the world-space bounds of their content (debugging). */
  sceneGraph: Array<{ name: string; type: string; visible: boolean; children: number; minY: number; maxY: number; instances: number }>
}

const initial: WorldDiagnostics = {
  cameraNavigation: null,
  beginInteractionSample,
  endInteractionSample,
  ambientCosts: () => ambientMeasurements.snapshot(),
  resetAmbientCosts: () => ambientMeasurements.reset(),
  cameraPoseRoute: null,
  cameraObstacleGuard: null,
  cameraZoomGuard: null,
  cameraOwnership: null,
  cameraHistory: [],
  webgl: null,
  measure: () => undefined,
  terrainPreparations: 0,
  terrainMeshMode: null,
  terrainMeshCache: null,
  ready: false,
  presented: false,
  presentationEpoch: 0,
  epochReadyAt: null,
  firstReadyAt: null,
  preparedAssets: preparedAssets.stats(),
  minimumReadyFps: tuning.quality.diagnostics.minimumReadyFps,
  assetsLoaded: false,
  introDone: false,
  framesRendered: 0,
  totalFrames: 0,
  scene: 'park',
  profile: 'high',
  auto: true,
  period: 'day',
  weather: 'clear',
  weatherPressure: 0,
  drawCalls: 0,
  triangles: 0,
  programs: 0,
  geometries: 0,
  textures: 0,
  frameMsP50: 0,
  frameMsP95: 0,
  gpuMsP50: 0,
  gpuMsP95: 0,
  gpuSamples: 0,
  gpuTimerReason: 'timer not initialized',
  passMs: { shadow: 0, main: 0, post: 0, total: 0 },
  toneMapping: 'none',
  ao: false,
  bloom: false,
  dpr: 1,
  msaa: 0,
  cameraPosition: [0, 0, 0],
  cameraTarget: [0, 0, 0],
  nearestHit: -1,
  footprintFill: 0,
  footprint: { width: 0, depth: 0, center: [0, 0] },
  gpu: '',
  groupCosts: [],
  drawCallsUnattributed: 0,
  trianglesUnattributed: 0,
  shadowRefreshes: 0,
  vegetationCullRuns: 0,
  vegetationCullSkips: 0,
  lampLightsMounted: 0,
  lampLightSelections: 0,
  qualityHistory: [],
  sceneGraph: [],
}

declare global {
  interface Window {
    __worldDiagnostics?: WorldDiagnostics
  }
}

type Listener = () => void

let state: WorldDiagnostics = { ...initial }

/** Mutate counters without allocating a diagnostic snapshot on each frame. */
export function recordVegetationCull(ran: boolean): void {
  if (ran) state.vegetationCullRuns += 1
  else state.vegetationCullSkips += 1
}

export function recordLampSelection(): void {
  state.lampLightSelections += 1
}
const listeners = new Set<Listener>()
const frameTimes: number[] = []

function publish() {
  if (typeof window !== 'undefined') window.__worldDiagnostics = state
  for (const listener of listeners) listener()
}

export function readDiagnostics(): WorldDiagnostics {
  return state
}

export function subscribeDiagnostics(listener: Listener): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function resetDiagnostics(): void {
  ambientMeasurements.reset()
  state = { ...initial }
  interaction = null
  frameTimes.length = 0
  publish()
}

function percentile(sorted: number[], p: number): number {
  if (sorted.length === 0) return 0
  const index = Math.min(sorted.length - 1, Math.floor(p * (sorted.length - 1)))
  return sorted[index] ?? 0
}

/** Record one frame's delta (seconds) and recompute the timing percentiles. */
export function recordFrame(deltaSeconds: number, windowSize = tuning.quality.diagnostics.frameWindow): void {
  if (interaction) {
    if (interaction.skipFirst) interaction.skipFirst = false
    else if (interaction.frames.length < tuning.quality.diagnostics.interactionSampleLimit) interaction.frames.push(deltaSeconds * 1000)
    else interaction.truncated = true
  }
  state.totalFrames += 1
  frameTimes.push(deltaSeconds * 1000)
  if (frameTimes.length > windowSize) frameTimes.splice(0, frameTimes.length - windowSize)
}

export function frameStats(): { p50: number; p95: number } {
  const sorted = [...frameTimes].sort((a, b) => a - b)
  return { p50: percentile(sorted, 0.5), p95: percentile(sorted, 0.95) }
}

export function updateDiagnostics(patch: Partial<WorldDiagnostics>): void {
  if (interaction && patch.cameraZoomGuard && interaction.zoom.length < tuning.quality.diagnostics.interactionSampleLimit) interaction.zoom.push(patch.cameraZoomGuard.queryMs)
  const next = { ...state, ...patch }
  if (next.terrainPreparations > 0 || !next.assetsLoaded || !next.introDone || next.scene !== state.scene) next.presented = false
  next.ready = next.terrainPreparations === 0 && next.assetsLoaded && next.introDone && next.presented
  // Capture at the producer, before harness polling, settling, or screenshots.
  next.epochReadyAt = next.epochReadyAt ?? (next.ready ? performance.now() : null)
  next.firstReadyAt = state.firstReadyAt ?? (next.ready ? performance.now() : null)
  state = next
  publish()
}

/** A preparation lease prevents readiness until its committed mesh is attached. */
export function beginTerrainPreparation(): () => void {
  updateDiagnostics({ terrainPreparations: state.terrainPreparations + 1 })
  let released = false
  return () => {
    if (released) return
    released = true
    updateDiagnostics({ terrainPreparations: Math.max(0, state.terrainPreparations - 1) })
  }
}

/** A committed world must earn readiness from its own assets, intro and frame. */
export function beginPresentation(epoch: number): void {
  if (epoch <= state.presentationEpoch) return
  updateDiagnostics({ presentationEpoch: epoch, epochReadyAt: null, presented: false, assetsLoaded: false, introDone: false })
}

/** Called after a frame from this Canvas completes, never from an FPS timer. */
export function recordPresentedFrame(epoch = state.presentationEpoch): void {
  if (epoch !== state.presentationEpoch) return
  if (interaction?.inputAt !== null && interaction?.inputAt !== undefined) {
    if (interaction.input.length < tuning.quality.diagnostics.interactionSampleLimit) interaction.input.push(performance.now() - interaction.inputAt)
    interaction.inputAt = null
  }
  if (state.terrainPreparations === 0 && state.assetsLoaded && state.introDone && !state.presented) updateDiagnostics({ presented: true })
}

export interface InteractionSample {
  durationMs: number
  frames: ReturnType<typeof summarize>
  inputToRenderMs: ReturnType<typeof summarize>
  zoomQueryMs: ReturnType<typeof summarize>
  truncated: boolean
}
let interaction: { startedAt: number; skipFirst: boolean; frames: number[]; input: number[]; zoom: number[]; inputAt: number | null; truncated: boolean } | null = null
function summarize(values: number[]) {
  const sorted = [...values].sort((a, b) => a - b)
  return { count: sorted.length, p50: percentile(sorted, 0.5), p95: percentile(sorted, 0.95), p99: percentile(sorted, 0.99), max: sorted[sorted.length - 1] ?? 0, over50ms: sorted.filter(value => value > tuning.quality.diagnostics.interactionLongFrameMs).length }
}
export function beginInteractionSample(): void {
  interaction = { startedAt: performance.now(), skipFirst: true, frames: [], input: [], zoom: [], inputAt: null, truncated: false }
}
/** First canvas input until its completed render; this is not display/input-to-photon latency. */
export function recordInteractionInput(): void {
  if (interaction && interaction.inputAt === null) interaction.inputAt = performance.now()
}
export function endInteractionSample(): InteractionSample | null {
  const sample = interaction
  interaction = null
  return sample ? { durationMs: performance.now() - sample.startedAt, frames: summarize(sample.frames), inputToRenderMs: summarize(sample.input), zoomQueryMs: summarize(sample.zoom), truncated: sample.truncated } : null
}

// Publish immediately so 2D fallback pages expose diagnostic state without a Canvas.
publish()
