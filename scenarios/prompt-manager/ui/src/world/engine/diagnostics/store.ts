/**
 * Live renderer diagnostics. Written by the in-canvas probe every frame,
 * read by the overlay (throttled) and by the smoke tool through
 * `window.__worldDiagnostics`.
 */
import type { PeriodId, QualityProfileId, SceneId, WeatherId } from '../../config'
import type { WebGLProbeResult } from '../webgl'
import type { QualityVerdictRecord } from '../quality/governor'

export interface WorldDiagnostics {
  webgl: WebGLProbeResult | null
  /** Run expensive scene-graph, raycast, and framing measurements on demand. */
  measure: () => void
  ready: boolean
  assetsLoaded: boolean
  introDone: boolean
  /** Rendered frames in the most recent wall-clock second. */
  framesRendered: number
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
  qualityHistory: QualityVerdictRecord[]
  /** Top-level scene groups with child counts and the world-space bounds of their content (debugging). */
  sceneGraph: Array<{ name: string; type: string; visible: boolean; children: number; minY: number; maxY: number; instances: number }>
}

const FRAME_WINDOW = 120

const initial: WorldDiagnostics = {
  webgl: null,
  measure: () => undefined,
  ready: false,
  assetsLoaded: false,
  introDone: false,
  framesRendered: 0,
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
  state = { ...initial }
  frameTimes.length = 0
  publish()
}

function percentile(sorted: number[], p: number): number {
  if (sorted.length === 0) return 0
  const index = Math.min(sorted.length - 1, Math.floor(p * (sorted.length - 1)))
  return sorted[index] ?? 0
}

/** Record one frame's delta (seconds) and recompute the timing percentiles. */
export function recordFrame(deltaSeconds: number): void {
  frameTimes.push(deltaSeconds * 1000)
  if (frameTimes.length > FRAME_WINDOW) frameTimes.shift()
}

export function frameStats(): { p50: number; p95: number } {
  const sorted = [...frameTimes].sort((a, b) => a - b)
  return { p50: percentile(sorted, 0.5), p95: percentile(sorted, 0.95) }
}

export const READY_FRAMES = 12

export function updateDiagnostics(patch: Partial<WorldDiagnostics>): void {
  const next = { ...state, ...patch }
  next.ready = next.assetsLoaded && next.introDone && next.framesRendered >= READY_FRAMES
  state = next
  publish()
}

// Publish immediately so 2D fallback pages expose diagnostic state without a Canvas.
publish()

/** Frames that must render after assets and intro settle before `ready` flips. */
