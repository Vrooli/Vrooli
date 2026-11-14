import { Box3, Vector3, type BufferGeometry } from 'three'

export interface HullPlane { normal: Vector3; limit: number }
export interface TriangleHull { bounds: Box3; planes: HullPlane[] }
const cache = new WeakMap<BufferGeometry, TriangleHull[]>()

/** Each rendered triangle is a convex sheet. Capsule expansion gives it volume. */
export function triangleHulls(geometry: BufferGeometry): readonly TriangleHull[] {
  const found = cache.get(geometry)
  if (found) return found
  const position = geometry.getAttribute('position'), index = geometry.getIndex()
  const hulls: TriangleHull[] = []
  for (let offset = 0; offset + 2 < (index?.count ?? position.count); offset += 3) {
    const points = [0, 1, 2].map(i => new Vector3().fromBufferAttribute(position, index ? index.getX(offset + i) : offset + i))
    const [a, b, c] = points
    if (!a || !b || !c) continue
    const normal = b.clone().sub(a).cross(c.clone().sub(a))
    if (normal.lengthSq() < 1e-20) continue
    normal.normalize()
    const planes = [{ normal, limit: normal.dot(a) }, { normal: normal.clone().negate(), limit: -normal.dot(a) }]
    points.forEach((point, i) => {
      const next = points[(i + 1) % 3]
      if (!next) return
      const outward = next.clone().sub(point).cross(normal).normalize()
      planes.push({ normal: outward, limit: outward.dot(point) })
    })
    hulls.push({ bounds: new Box3().setFromPoints(points), planes })
  }
  cache.set(geometry, hulls)
  return hulls
}

/** Clip an infinite line to a convex hull represented by outward half-spaces. */
export function hullInterval(point: Vector3, direction: Vector3, planes: readonly HullPlane[]): [number, number] | null {
  let enter = -Infinity, leave = Infinity
  for (const { normal, limit } of planes) {
    const delta = normal.dot(direction), distance = limit - normal.dot(point)
    if (Math.abs(delta) < 1e-14) { if (distance < 0) return null; continue }
    const t = distance / delta
    if (delta < 0) enter = Math.max(enter, t)
    else leave = Math.min(leave, t)
    if (enter > leave) return null
  }
  return [enter, leave]
}

export function hullEntry(from: Vector3, to: Vector3, planes: readonly HullPlane[], direction: Vector3): number {
  direction.subVectors(to, from)
  let inside = true, depth = Infinity, escape = 0
  for (const { normal, limit } of planes) {
    const distance = limit - normal.dot(from)
    if (distance < 0) inside = false
    if (distance < depth) { depth = distance; escape = normal.dot(direction) }
  }
  if (inside) return escape > 0 || (depth <= 1e-8 && escape >= 0) ? 1 : 0
  const interval = hullInterval(from, direction, planes)
  return interval && interval[1] >= 0 && interval[0] <= 1 ? Math.max(0, interval[0]) : 1
}
