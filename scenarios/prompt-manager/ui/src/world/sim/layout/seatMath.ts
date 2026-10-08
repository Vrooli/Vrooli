/**
 * Seat and boundary transforms, ported from the legacy ui/src/lib/world.ts.
 * The maths is unchanged; the types are the sim's own.
 */
import type { Vec2 } from '../model'

export type XZPoint = [number, number]

export interface ResolvedBoundary {
  shape: 'square' | 'circle' | 'path'
  size: number
  radius: number
  points: XZPoint[]
}

function squarePoints(size: number): XZPoint[] {
  const half = size / 2
  return [
    [-half, -half],
    [half, -half],
    [half, half],
    [-half, half],
  ]
}

export function resolveBoundary(shape: ResolvedBoundary['shape'], size: number, points?: XZPoint[]): ResolvedBoundary {
  if (shape === 'path' && points && points.length >= 3) {
    return { shape, size, radius: size / 2, points: points.map(([x, z]) => [x, z]) }
  }
  if (shape === 'circle') return { shape, size, radius: size / 2, points: [] }
  return { shape: 'square', size, radius: size / 2, points: squarePoints(size) }
}

export function isPointInsideBoundary(point: XZPoint, boundary: ResolvedBoundary): boolean {
  if (boundary.shape === 'circle') {
    const [x, z] = point
    return x * x + z * z <= boundary.radius * boundary.radius
  }
  const points = boundary.points.length > 0 ? boundary.points : squarePoints(boundary.size)
  let inside = false
  for (let i = 0, j = points.length - 1; i < points.length; j = i++) {
    const [xi, zi] = points[i] ?? [0, 0]
    const [xj, zj] = points[j] ?? [0, 0]
    const intersect = zi > point[1] !== zj > point[1] && point[0] < ((xj - xi) * (point[1] - zi)) / (zj - zi) + xi
    if (intersect) inside = !inside
  }
  return inside
}

function closestPointOnSegment(point: XZPoint, a: XZPoint, b: XZPoint): XZPoint {
  const [px, pz] = point
  const [ax, az] = a
  const [bx, bz] = b
  const abx = bx - ax
  const abz = bz - az
  const apx = px - ax
  const apz = pz - az
  const abLenSq = abx * abx + abz * abz
  const t = abLenSq === 0 ? 0 : Math.max(0, Math.min(1, (apx * abx + apz * abz) / abLenSq))
  return [ax + abx * t, az + abz * t]
}

export function clampPointToBoundary(point: XZPoint, boundary: ResolvedBoundary): XZPoint {
  if (isPointInsideBoundary(point, boundary)) return point
  if (boundary.shape === 'circle') {
    const [x, z] = point
    const len = Math.hypot(x, z)
    if (len === 0) return [boundary.radius, 0]
    const scale = boundary.radius / len
    return [x * scale, z * scale]
  }
  const points = boundary.points.length > 0 ? boundary.points : squarePoints(boundary.size)
  let closest: XZPoint = points[0] ?? point
  let minDistSq = Infinity
  for (let i = 0; i < points.length; i += 1) {
    const a = points[i]
    const b = points[(i + 1) % points.length]
    if (!a || !b) continue
    const candidate = closestPointOnSegment(point, a, b)
    const dx = candidate[0] - point[0]
    const dz = candidate[1] - point[1]
    const distSq = dx * dx + dz * dz
    if (distSq < minDistSq) {
      minDistSq = distSq
      closest = candidate
    }
  }
  return closest
}

/**
 * Convert a seat's local offset to world coordinates by applying furniture
 * position and Y-axis rotation (three.js Ry(θ) convention).
 */
export function seatLocalToWorld(
  seatLocal: [number, number, number],
  furniturePos: [number, number, number],
  furnitureRotation: number,
): [number, number, number] {
  const cos = Math.cos(furnitureRotation)
  const sin = Math.sin(furnitureRotation)
  const [sx, sy, sz] = seatLocal
  return [furniturePos[0] + sx * cos + sz * sin, furniturePos[1] + sy, furniturePos[2] - sx * sin + sz * cos]
}

/** Inverse of `seatLocalToWorld` (applies Ry(θ)^T). */
export function seatWorldToLocal(
  worldPos: [number, number, number],
  furniturePos: [number, number, number],
  furnitureRotation: number,
): [number, number, number] {
  const cos = Math.cos(furnitureRotation)
  const sin = Math.sin(furnitureRotation)
  const dx = worldPos[0] - furniturePos[0]
  const dy = worldPos[1] - furniturePos[1]
  const dz = worldPos[2] - furniturePos[2]
  return [dx * cos - dz * sin, dy, dx * sin + dz * cos]
}

/** XZ offset for a seat's facing-direction marker. */
export function seatFacingArrowOffset(worldRotation: number, length: number): [number, number, number] {
  return [Math.sin(worldRotation) * length, 0, Math.cos(worldRotation) * length]
}

/** Rotate seat offsets around Y and add the delta to each facing, normalised to [0, 2π). */
export function rotateAllSeats<T extends { position: [number, number, number]; rotation: number }>(seats: T[], deltaRadians: number): T[] {
  const cos = Math.cos(deltaRadians)
  const sin = Math.sin(deltaRadians)
  const TAU = Math.PI * 2
  return seats.map((seat) => {
    const [x, y, z] = seat.position
    return {
      ...seat,
      position: [x * cos + z * sin, y, -x * sin + z * cos] as [number, number, number],
      rotation: (((seat.rotation + deltaRadians) % TAU) + TAU) % TAU,
    }
  })
}

export function snapPointToGrid(point: XZPoint, snapSize: number): XZPoint {
  if (snapSize <= 0) return point
  return [Math.round(point[0] / snapSize) * snapSize, Math.round(point[1] / snapSize) * snapSize]
}

/** Rotate a local XZ offset by a yaw and add it to an origin. */
export function localToWorld2(offset: Vec2, origin: Vec2, rotation: number): Vec2 {
  const cos = Math.cos(rotation)
  const sin = Math.sin(rotation)
  return [origin[0] + offset[0] * cos + offset[1] * sin, origin[1] - offset[0] * sin + offset[1] * cos]
}
