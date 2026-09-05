import type { CameraPose, CameraTuning } from '../../config'
import { tuning } from '../../config'
import { MathUtils, type Box3 } from 'three'

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

/** Radian clamps for camera-controls derived from tuning and the scene's hero azimuth. */
export function orbitClamps(camera: CameraTuning, heroAzimuthDeg: number): OrbitClamps {
  return {
    minPolar: camera.polarMinDeg * DEG,
    maxPolar: camera.polarMaxDeg * DEG,
    minAzimuth: (heroAzimuthDeg - camera.azimuthRangeDeg) * DEG,
    maxAzimuth: (heroAzimuthDeg + camera.azimuthRangeDeg) * DEG,
    minDistance: camera.minDistance,
    maxDistance: camera.maxDistance,
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
