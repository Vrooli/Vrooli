import type { BoundaryConfig, PlacementConfig } from '@/types/environment'

export type XZPoint = [number, number]

export interface ResolvedBoundary {
  shape: BoundaryConfig['shape']
  size: number
  radius: number
  points: XZPoint[]
  position: number
  color?: string
  opacity?: number
}

const DEFAULT_BOUNDARY_SIZE = 60
const DEFAULT_BOUNDARY_POSITION = 0.01

function squarePoints(size: number): XZPoint[] {
  const half = size / 2
  return [
    [-half, -half],
    [half, -half],
    [half, half],
    [-half, half],
  ]
}

function normalizePath(points?: XZPoint[]): XZPoint[] {
  if (!points || points.length < 3) {
    return []
  }
  return points.map(([x, z]) => [x, z])
}

export function resolveBoundary(
  boundary: BoundaryConfig | undefined,
  groundSize?: number
): ResolvedBoundary | null {
  if (!boundary || !boundary.visible) {
    return null
  }

  const size = boundary.size ?? (groundSize ? groundSize * 2 : DEFAULT_BOUNDARY_SIZE)
  const position = boundary.position ?? DEFAULT_BOUNDARY_POSITION
  const shape = boundary.shape

  if (shape === 'path') {
    const points = normalizePath(boundary.points)
    if (points.length >= 3) {
      return {
        shape,
        size,
        radius: size / 2,
        points,
        position,
        color: boundary.color,
        opacity: boundary.opacity,
      }
    }
  }

  if (shape === 'circle') {
    return {
      shape,
      size,
      radius: size / 2,
      points: [],
      position,
      color: boundary.color,
      opacity: boundary.opacity,
    }
  }

  return {
    shape: 'square',
    size,
    radius: size / 2,
    points: squarePoints(size),
    position,
    color: boundary.color,
    opacity: boundary.opacity,
  }
}

export function getBoundaryLinePoints(boundary: ResolvedBoundary, segments = 64): XZPoint[] {
  if (boundary.shape === 'circle') {
    const pts: XZPoint[] = []
    for (let i = 0; i <= segments; i++) {
      const t = (i / segments) * Math.PI * 2
      pts.push([Math.cos(t) * boundary.radius, Math.sin(t) * boundary.radius])
    }
    return pts
  }

  const points = boundary.points.length > 0
    ? boundary.points
    : squarePoints(boundary.size)

  if (points.length === 0) {
    return []
  }

  const first = points[0]
  if (!first) {
    return []
  }

  const closed = points[points.length - 1]
    ? points
    : [...points, first]

  return closed
}

export function isPointInsideBoundary(point: XZPoint, boundary: ResolvedBoundary): boolean {
  if (boundary.shape === 'circle') {
    const [x, z] = point
    return x * x + z * z <= boundary.radius * boundary.radius
  }

  const points = boundary.points.length > 0
    ? boundary.points
    : squarePoints(boundary.size)

  let inside = false
  for (let i = 0, j = points.length - 1; i < points.length; j = i++) {
    const [xi, zi] = points[i] ?? [0, 0]
    const [xj, zj] = points[j] ?? [0, 0]
    const intersect = (zi > point[1]) !== (zj > point[1])
      && point[0] < ((xj - xi) * (point[1] - zi)) / (zj - zi) + xi
    if (intersect) {
      inside = !inside
    }
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
  if (isPointInsideBoundary(point, boundary)) {
    return point
  }

  if (boundary.shape === 'circle') {
    const [x, z] = point
    const len = Math.hypot(x, z)
    if (len === 0) {
      return [boundary.radius, 0]
    }
    const scale = boundary.radius / len
    return [x * scale, z * scale]
  }

  const points = boundary.points.length > 0
    ? boundary.points
    : squarePoints(boundary.size)

  let closest: XZPoint = points[0] ?? point
  let minDistSq = Infinity

  for (let i = 0; i < points.length; i++) {
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

export function snapPointToGrid(point: XZPoint, snapSize: number): XZPoint {
  if (snapSize <= 0) {
    return point
  }
  return [
    Math.round(point[0] / snapSize) * snapSize,
    Math.round(point[1] / snapSize) * snapSize,
  ]
}

export function applyPlacementConstraints(
  position: [number, number, number],
  options: {
    placement?: PlacementConfig
    boundary?: BoundaryConfig
    groundSize?: number
  }
): [number, number, number] {
  const placement = options.placement
  const boundary = resolveBoundary(options.boundary, options.groundSize)
  const snapToGrid = placement?.snapToGrid ?? false
  const snapSize = placement?.snapSize ?? 1
  const clampToBoundary = placement?.clampToBoundary ?? false

  let xz: XZPoint = [position[0], position[2]]

  if (snapToGrid) {
    xz = snapPointToGrid(xz, snapSize)
  }

  if (clampToBoundary && boundary) {
    xz = clampPointToBoundary(xz, boundary)
  }

  return [xz[0], position[1], xz[1]]
}
