export type NavigationMode = 'explore' | 'first-person' | 'third-person'
export type NavigationTool = 'orbit' | 'pan'
export type NavigationPreset = 'top' | 'front' | 'isometric'
export type NavigationCommand = 'zoom-in' | 'zoom-out' | 'left' | 'right' | 'up' | 'down' | 'orbit-left' | 'orbit-right' | 'stop' | 'jump'
export interface NavigationState { mode: NavigationMode; heading: number; blocked: boolean; locked: boolean; message: string }
export const INITIAL_NAVIGATION_STATE: NavigationState = { mode: 'explore', heading: 0, blocked: false, locked: false, message: '' }
export interface NavigationPreferences {
  device: 'mouse' | 'trackpad'
  sensitivity: number
  invertZoom: boolean
  invertLook: boolean
  smoothing: boolean
}
export const DEFAULT_NAVIGATION: NavigationPreferences = { device: 'mouse', sensitivity: 1, invertZoom: false, invertLook: false, smoothing: false }
export const NAVIGATION_STORAGE_KEY = 'prompt-manager.world.navigation.v1'
export function parseNavigationPreferences(value: unknown): NavigationPreferences {
  const p = value && typeof value === 'object' ? value as Partial<NavigationPreferences> : {}
  return {
    device: p.device === 'trackpad' ? 'trackpad' : 'mouse',
    sensitivity: typeof p.sensitivity === 'number' && Number.isFinite(p.sensitivity) ? Math.max(.25, Math.min(3, p.sensitivity)) : 1,
    invertZoom: p.invertZoom === true, invertLook: p.invertLook === true, smoothing: p.smoothing === true,
  }
}


export const WALK = {
  radius: .3, height: 1.8, eyeHeight: 1.6, speed: 3.5, runSpeed: 7,
  jumpSpeed: 5, gravity: 12, physicsStep: 1 / 120,
  stepHeight: .45, kerbTolerance: .05, slopeSampleMetres: .1, maxGrade: Math.tan(.45),
  stepMetres: .08, spawnRings: 40, spawnRingMetres: .5,
  lookRadiansPerPixel: .0025, maxPitch: 1.4,
  boom: 4, minBoom: .8, maxBoom: 8, boomStep: .5, boomMetresPerPixel: .005,
  boomRadius: .2, boomSampleMetres: .1, avatarHideDistance: .65,
  near: .08, fov: 60,
} as const

export const NAV_MOTION = {
  maxOrdinaryFrame: .25, integrationStep: .05, dragSmoothing: .06,
  wheelPixelsPerUnit: 30, blockedNoticeMs: 600, telemetryMs: 100,
  panButtonPixels: 60, zoomButtonPixels: 90, orbitButtonRadians: .15,
  lookButtonPixels: 100, walkButtonSeconds: .1, topPolar: .001,
  telemetryPriority: -.25, walkingPriority: -.5, positionEpsilon: 1e-6, motionEpsilon: 1e-8,
} as const

export const NAV_VISUALS = {
  iconPixels: 15, pivotOrder: 10, pivotGeometry: [.65, 1, 24],
  accent: '#38bdf8', pivotOpacity: .8, pivotMinScale: .1, pivotDistanceScale: .008,
  bodyPosition: [0, .8, 0], bodyGeometry: [.28, 1, 4, 8],
  facePosition: [0, 1.55, -.15], faceGeometry: [.18, 12, 8], faceColor: '#f8fafc',
} as const


export interface SavedWalkingView { position: [number, number, number]; yaw: number; pitch: number; boom: number }
export interface SavedCameraView {
  version: 1
  mode: NavigationMode
  explore: { position: [number, number, number]; target: [number, number, number]; zoom: number }
  body?: SavedWalkingView
}
export interface CameraMemory {
  read(): SavedCameraView | null
  save(view: SavedCameraView): void
  flush(): void
}
