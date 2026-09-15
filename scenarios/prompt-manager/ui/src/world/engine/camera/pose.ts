import type { CameraPose, CameraTuning } from '../../config'
import { tuning } from '../../config'
import { MathUtils, Vector3, type Box3 } from 'three'
import type CameraControls from 'camera-controls'
import type { WorldExtent } from '../types'

/** CameraControls.stop jumps to its endpoints. Replace them with the current pose first. */
export function freezeCamera(controls: CameraControls): void {
  const position = controls.getPosition(new Vector3(), false)
  const target = controls.getTarget(new Vector3(), false)
  const offset = controls.getFocalOffset(new Vector3(), false)
  const zoom = controls.camera.zoom
  void controls.setLookAt(position.x, position.y, position.z, target.x, target.y, target.z, false)
  void controls.setFocalOffset(offset.x, offset.y, offset.z, false)
  void controls.zoomTo(zoom, false)
  controls.stop()
}

export type Vec3 = readonly [number, number, number]

const DEG = MathUtils.DEG2RAD

/** Convert a scene camera pose (spherical around a target) into a world position. */
export function poseToPosition(
  pose: CameraPose,
  center: readonly [number, number],
  fit: number,
): { position: Vec3; target: Vec3; distance: number } {
  const polar = pose.polarDeg * DEG
  const azimuth = pose.azimuthDeg * DEG
  const distance = pose.distanceFactor * fit
  const target: Vec3 = [center[0], pose.targetY, center[1]]
  const position: Vec3 = [
    target[0] + distance * Math.sin(polar) * Math.sin(azimuth),
    target[1] + distance * Math.cos(polar),
    target[2] + distance * Math.sin(polar) * Math.cos(azimuth),
  ]
  return { position, target, distance }
}

export interface Footprint {
  width: number
  depth: number
  center: readonly [number, number]
}

export interface FrameInput {
  minimumProjectionAspect?: number
  minimumFrameFill?: number
  /** Bottom of an elevated box; world footprints default to ground zero. */
  baseY?: number
  /** Ground points to frame, in world space; the box they span is centred on `center`. */
  points: ReadonlyArray<readonly [number, number]>
  /** The look-at point on the ground. */
  center: readonly [number, number]
  /** Height framed above the ground (actors, walls, labels). */
  height: number
  polarDeg: number
  azimuthDeg: number
  targetY: number
  fovDeg: number
  aspect: number
}

interface ViewBasis {
  toCamera: Vec3
  forward: Vec3
  right: Vec3
  up: Vec3
}

function dot(a: Vec3, b: Vec3): number {
  return a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

function cross(a: Vec3, b: Vec3): Vec3 {
  return [a[1] * b[2] - a[2] * b[1], a[2] * b[0] - a[0] * b[2], a[0] * b[1] - a[1] * b[0]]
}

function normalize(v: Vec3): Vec3 {
  const length = Math.hypot(v[0], v[1], v[2]) || 1
  return [v[0] / length, v[1] / length, v[2] / length]
}

function viewBasis(polarDeg: number, azimuthDeg: number): ViewBasis {
  const polar = polarDeg * DEG
  const azimuth = azimuthDeg * DEG
  const toCamera: Vec3 = [Math.sin(polar) * Math.sin(azimuth), Math.cos(polar), Math.sin(polar) * Math.cos(azimuth)]
  const forward: Vec3 = [-toCamera[0], -toCamera[1], -toCamera[2]]
  const right = normalize(cross(forward, [0, 1, 0]))
  const up = cross(right, forward)
  return { toCamera, forward, right, up }
}

/** Every framed point at ground level and at `height`, relative to the look-at target. */
function frameCorners({ points, center, height, targetY, baseY = 0 }: FrameInput): Vec3[] {
  const corners: Vec3[] = []
  for (const [x, z] of points) for (const y of [baseY, baseY + height]) corners.push([x - center[0], y - targetY, z - center[1]])
  return corners
}

/** The four ground corners of an extent, for callers that frame a box rather than an outline. */
export function extentPoints(footprint: Footprint): Array<readonly [number, number]> {
  const halfW = footprint.width / 2
  const halfD = footprint.depth / 2
  const points: Array<readonly [number, number]> = []
  for (const sx of [-1, 1]) for (const sz of [-1, 1]) points.push([footprint.center[0] + sx * halfW, footprint.center[1] + sz * halfD])
  return points
}

function halfTangents(fovDeg: number, aspect: number, minimumAspect: number): { tanV: number; tanH: number } {
  const tanV = Math.tan((fovDeg * DEG) / 2)
  return { tanV, tanH: tanV * Math.max(aspect, minimumAspect) }
}

/**
 * Distance from the look-at target at which the framed points occupy
 * `fill` of the viewport on their widest axis, seen from the given pose.
 * Closed form: for a corner at camera-space (x, y) and depth offset f from
 * the target, |x| <= tanH * fill * (D + f) bounds D from below.
 */
export function frameDistance(input: FrameInput, fill: number): number {
  const { forward, right, up } = viewBasis(input.polarDeg, input.azimuthDeg)
  const { tanV, tanH } = halfTangents(input.fovDeg, input.aspect, input.minimumProjectionAspect ?? tuning.camera.minimumProjectionAspect)
  const share = Math.max(fill, input.minimumFrameFill ?? tuning.camera.minimumFrameFill)
  let distance = 0
  for (const corner of frameCorners(input)) {
    const depth = dot(corner, forward)
    distance = Math.max(distance, Math.abs(dot(corner, right)) / (tanH * share) - depth, Math.abs(dot(corner, up)) / (tanV * share) - depth)
  }
  return distance
}

/** Share of the viewport the framed points occupy at `distance` (1 touches the frame edge; above 1 is cropped). */
export function footprintFill(input: FrameInput, distance: number): number {
  const { forward, right, up } = viewBasis(input.polarDeg, input.azimuthDeg)
  const { tanV, tanH } = halfTangents(input.fovDeg, input.aspect, input.minimumProjectionAspect ?? tuning.camera.minimumProjectionAspect)
  let fill = 0
  for (const corner of frameCorners(input)) {
    const depth = distance + dot(corner, forward)
    if (depth <= 0) return Infinity
    fill = Math.max(fill, Math.abs(dot(corner, right)) / (tanH * depth), Math.abs(dot(corner, up)) / (tanV * depth))
  }
  return fill
}

export interface OrbitClamps {
  minPolar: number
  maxPolar: number
  minAzimuth: number
  maxAzimuth: number
  minDistance: number
  maxDistance: number
}

/** Sphere around the eye that contains every corner of the perspective near plane. */
export function nearPlaneRadius(near: number, fov: number, aspect: number, zoom = 1): number {
  const halfHeight = near * Math.tan(fov * Math.PI / 360) / zoom
  return Math.hypot(near, halfHeight, halfHeight * aspect)
}

/** Radian clamps and lens-aware distance limits. */
export function cameraRange(camera: CameraTuning, aspect = 1, extent?: Pick<WorldExtent, 'width' | 'depth'>, framingFactor = 1): { minDistance: number; maxDistance: number; far: number } {
  const minDistance = Math.max(camera.minDistance, nearPlaneRadius(camera.near, camera.fov, aspect))
  const diameter = extent ? Math.hypot(extent.width, extent.depth, camera.frameHeight) : 0
  const halfAngle = Math.atan(Math.tan(camera.fov * Math.PI / 360) * Math.max(Number.EPSILON, Math.min(1, aspect)) * camera.frameFill)
  const fit = diameter / (2 * Math.sin(halfAngle)) * framingFactor
  const maxDistance = Math.max(minDistance, camera.maxDistance, fit)
  return { minDistance, maxDistance, far: Math.max(camera.far, maxDistance + diameter + camera.near) }
}

export function orbitClamps(camera: CameraTuning, heroAzimuthDeg: number, aspect = 1, extent?: Pick<WorldExtent, 'width' | 'depth'>, framingFactor = 1): OrbitClamps {
  const range = cameraRange(camera, aspect, extent, framingFactor)
  return {
    minPolar: camera.polarMinDeg * DEG,
    maxPolar: camera.polarMaxDeg * DEG,
    minAzimuth: camera.azimuthRangeDeg >= 180 ? -Infinity : (heroAzimuthDeg - camera.azimuthRangeDeg) * DEG,
    maxAzimuth: camera.azimuthRangeDeg >= 180 ? Infinity : (heroAzimuthDeg + camera.azimuthRangeDeg) * DEG,
    minDistance: range.minDistance,
    maxDistance: range.maxDistance,
  }
}

export function clamp(value: number, min: number, max: number): number {
  return value < min ? min : value > max ? max : value
}

/** Clamp a pose into the orbit clamps so a stored or requested pose can never escape the diorama. */
export function clampPose(pose: CameraPose, clamps: OrbitClamps, fit: number): CameraPose {
  return {
    ...pose,
    polarDeg: clamp(pose.polarDeg, clamps.minPolar / DEG, clamps.maxPolar / DEG),
    azimuthDeg: clamp(pose.azimuthDeg, clamps.minAzimuth / DEG, clamps.maxAzimuth / DEG),
    distanceFactor: clamp(pose.distanceFactor * fit, clamps.minDistance, clamps.maxDistance) / fit,
  }
}

export interface FocusedPose extends CameraPose {
  frame: FrameInput
  fill: number
}

/** Adapt any world-space box to the same framing solver used by home. */
export function poseForBox(
  box: Box3,
  current: Pick<CameraPose, 'polarDeg' | 'azimuthDeg'>,
  camera: CameraTuning,
  aspect: number,
  clamps: OrbitClamps,
): FocusedPose {
  if (box.isEmpty()) throw new Error('Cannot focus an empty box')
  const center: readonly [number, number] = [(box.min.x + box.max.x) / 2, (box.min.z + box.max.z) / 2]
  const targetY = (box.min.y + box.max.y) / 2
  const angles = clampPose({ ...current, targetY, distanceFactor: 1 }, clamps, 1)
  const frame: FrameInput = {
    points: extentPoints({ width: box.max.x - box.min.x, depth: box.max.z - box.min.z, center }),
    center, baseY: box.min.y, height: box.max.y - box.min.y,
    polarDeg: angles.polarDeg, azimuthDeg: angles.azimuthDeg, targetY,
    fovDeg: camera.fov, aspect, minimumProjectionAspect: camera.minimumProjectionAspect, minimumFrameFill: camera.minimumFrameFill,
  }
  const fill = camera.frameFill / camera.focusPadding
  // A point-sized box still respects the configured closest camera distance.
  const fit = Math.max(frameDistance(frame, fill), Number.EPSILON)
  return { ...clampPose({ ...angles, distanceFactor: 1 }, clamps, fit), frame, fill }
}

/** Prefer the current view; inspect a bounded set of alternatives when occluded. */
export function visiblePoseForBox(
  box: Box3, current: Pick<CameraPose, 'polarDeg' | 'azimuthDeg'>,
  camera: CameraTuning, aspect: number, clamps: OrbitClamps,
  visible: (eye: Vector3, target: Vector3) => boolean,
): FocusedPose {
  let best = poseForBox(box, current, camera, aspect, clamps)
  let bestScore = -1
  const target = box.getCenter(new Vector3())
  const eye = new Vector3()
  // Quarter-turn candidates and an elevated view preserve the configured clamps.
  for (const polarDeg of [current.polarDeg, camera.polarMinDeg]) {
    for (const turn of camera.focusVisibilityTurnsDeg) {
      const pose = poseForBox(box, { polarDeg, azimuthDeg: current.azimuthDeg + turn }, camera, aspect, clamps)
      const resolved = poseToPosition(pose, pose.frame.center, Math.max(frameDistance(pose.frame, pose.fill), Number.EPSILON))
      eye.set(...resolved.position)
      let score = 0
      for (const fraction of camera.focusVisibilityHeights) {
        target.y = box.min.y + (box.max.y - box.min.y) * fraction
        if (visible(eye, target)) score++
      }
      if (score > bestScore) { best = pose; bestScore = score }
      if (score === camera.focusVisibilityHeights.length) return best
    }
  }
  return best
}
