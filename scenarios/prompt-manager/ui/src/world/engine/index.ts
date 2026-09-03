/**
 * engine layer — canvas, lighting, post, camera, quality, diagnostics, assets.
 * Imports `config` only. Knows nothing about agents, teams or places.
 */
export { WorldCanvas } from './WorldCanvas'
export { LightingRig } from './lighting/Rig'
export { PostChain } from './post/Chain'
export { CameraRig, type CameraRigHandle } from './camera/Rig'
export { poseToPosition, orbitClamps, clampPose, fitDistance, frameDistance, footprintFill, extentPoints } from './camera/pose'
export { decideIntro } from './camera/intro'
export { QualityGovernor } from './quality/QualityGovernor'
export { applyVerdict, pickProfile, setAuto, governorBounds, isQualityProfileId } from './quality/governor'
export { DiagnosticsProbe } from './diagnostics/Probe'
export { DiagnosticsOverlay } from './diagnostics/Overlay'
export { readDiagnostics, subscribeDiagnostics, resetDiagnostics, updateDiagnostics, type WorldDiagnostics } from './diagnostics/store'
export { worldAssetUrl, WORLD_ASSETS } from './assets/urls'
export type { WorldBounds } from './types'
