/**
 * engine layer — canvas, lighting, post, camera, quality, diagnostics, assets.
 * Imports `config` only. Knows nothing about agents, teams or places.
 */
export { WorldCanvas } from './WorldCanvas'
export { FrameDriver } from './frameDriver'
export { LightingRig } from './lighting/Rig'
export { applyWeather } from './lighting/weather'
export { PostChain } from './post/Chain'
export { CameraRig, type CameraRigHandle } from './camera/Rig'
export { poseToPosition, orbitClamps, clampPose, fitDistance, frameDistance, footprintFill, extentPoints } from './camera/pose'
export { decideIntro } from './camera/intro'
export { QualityGovernor } from './quality/QualityGovernor'
export { applyVerdict, chooseInitialProfile, pickProfile, setAuto, governorBounds, isQualityProfileId, type QualityVerdictRecord } from './quality/governor'
export { DiagnosticsProbe } from './diagnostics/Probe'
export { DiagnosticsOverlay } from './diagnostics/Overlay'
export { readDiagnostics, subscribeDiagnostics, resetDiagnostics, updateDiagnostics, type WorldDiagnostics } from './diagnostics/store'
export { probeWebGL, resolveTwoD, retryWebGL, type WebGLProbeResult } from './webgl'
export { worldAssetUrl, WORLD_ASSETS } from './assets/urls'
export type { WorldBounds } from './types'
