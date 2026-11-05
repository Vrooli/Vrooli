import { NAV_MOTION } from '../../config/navigation'
import { Box3, InstancedMesh, Matrix4, Mesh, Vector3, type Object3D } from 'three'
import { hullEntry, hullInterval, triangleHulls, type HullPlane } from './convex'

/** Fraction of a segment before entering a box; an existing overlap may escape. */
function segmentEntry(from: Vector3, to: Vector3, box: Box3): number {
  if (box.containsPoint(from)) {
    let depth = Infinity
    let escape = 0
    for (const axis of ['x', 'y', 'z'] as const) {
      const delta = to[axis] - from[axis]
      if (from[axis] - box.min[axis] < depth) { depth = from[axis] - box.min[axis]; escape = -delta }
      if (box.max[axis] - from[axis] < depth) { depth = box.max[axis] - from[axis]; escape = delta }
    }
    return escape > 0 || (depth <= 1e-8 && escape >= 0) ? 1 : 0
  }
  let enter = 0
  let leave = 1
  for (const axis of ['x', 'y', 'z'] as const) {
    const delta = to[axis] - from[axis]
    if (Math.abs(delta) < 1e-14) {
      if (from[axis] < box.min[axis] || from[axis] > box.max[axis]) return 1
      continue
    }
    const a = (box.min[axis] - from[axis]) / delta
    const b = (box.max[axis] - from[axis]) / delta
    enter = Math.max(enter, Math.min(a, b))
    leave = Math.min(leave, Math.max(a, b))
    if (enter > leave) return 1
  }
  return enter
}

export interface ObstacleSweepSample { fraction: number; boxes: number; queryMs: number; recoveryLift?: number }

/**
 * Sweep the near-plane sphere against explicitly declared box geometry.
 * Expanding each local axis encloses the transformed sphere conservatively,
 * including nonuniform scales. Rounded box corners may stop slightly early.
 */
export function createObstacleSweep(root: Object3D, measure?: (sample: ObstacleSweepSample) => void, includeFurniture = false) {
  const matrix = new Matrix4()
  const instance = new Matrix4()
  const inverse = new Matrix4()
  const worldTransform = new Matrix4()
  const worldBox = new Box3()
  const localFrom = new Vector3()
  const localTo = new Vector3()
  const padding = new Vector3()
  const expanded = new Box3()
  const expandedPlanes: HullPlane[] = []
  const direction = new Vector3()
  const queryBounds = new Box3()
  let activePlanes: readonly HullPlane[] | null = null
  const visitBoxes = (radius: number, visit: () => void, verticalSpan = 0, query?: Box3) => {
    let boxes = 0
    // Only collision meshes need current transforms. Updating every descendant
    // also updates thousands of unrelated actor and decorative instance nodes
    // on each walking/step/boom query.
    root.updateWorldMatrix(true, false)
    const test = (mesh: Mesh, transform: Matrix4) => {
      if (Math.abs(transform.determinant()) < 1e-12) return
      if (!mesh.geometry.boundingBox) mesh.geometry.computeBoundingBox()
      if (!mesh.geometry.boundingBox) return
      boxes++
      if (query) {
        worldBox.copy(mesh.geometry.boundingBox).applyMatrix4(transform)
        if (!worldBox.intersectsBox(query)) return
      }
      worldTransform.copy(transform)
      inverse.copy(transform).invert()
      const e = inverse.elements
      padding.set(Math.hypot(e[0], e[4], e[8]), Math.hypot(e[1], e[5], e[9]), Math.hypot(e[2], e[6], e[10])).multiplyScalar(radius)
      // A vertical capsule is a sphere swept along a vertical segment. Expand
      // by that segment's projection into each box-local axis as well.
      padding.x += Math.abs(e[4]) * verticalSpan / 2
      padding.y += Math.abs(e[5]) * verticalSpan / 2
      padding.z += Math.abs(e[6]) * verticalSpan / 2
      if (mesh.geometry.userData.cameraObstacle === 'triangles') {
        for (const hull of triangleHulls(mesh.geometry)) {
          expanded.copy(hull.bounds).expandByVector(padding)
          for (const [i, plane] of hull.planes.entries()) {
            const target = expandedPlanes[i] ?? (expandedPlanes[i] = { normal: new Vector3(), limit: 0 })
            const n = plane.normal
            target.normal.copy(n)
            // Support of the world-space capsule along the transformed normal.
            const nx = n.x * e[0] + n.y * e[1] + n.z * e[2]
            const ny = n.x * e[4] + n.y * e[5] + n.z * e[6]
            const nz = n.x * e[8] + n.y * e[9] + n.z * e[10]
            target.limit = plane.limit + radius * Math.hypot(nx, ny, nz) + Math.abs(ny) * verticalSpan / 2
          }
          expandedPlanes.length = hull.planes.length
          activePlanes = expandedPlanes
          visit()
        }
      } else {
        activePlanes = null
        expanded.copy(mesh.geometry.boundingBox).expandByVector(padding)
        visit()
      }
    }
    root.traverseVisible(object => {
      if (!(object instanceof Mesh)) return
      const mesh = object as Mesh
      // Effect cards can live beside solid props without inheriting their
      // collision policy (flames and smoke must never become invisible walls).
      if (mesh.userData.walkObstacle === false) return
      let eligible = mesh.geometry.userData.cameraObstacle === 'box' || mesh.geometry.userData.cameraObstacle === 'triangles'
      if (includeFurniture) {
        for (let parent: Object3D | null = object; parent && !eligible; parent = parent.parent) eligible = parent.userData.walkObstacle === true
      }
      if (!eligible) return
      mesh.updateWorldMatrix(true, false)
      if (object instanceof InstancedMesh) {
        for (let index = 0; index < object.count; index++) {
          object.getMatrixAt(index, instance)
          matrix.multiplyMatrices(object.matrixWorld, instance)
          test(object, matrix)
        }
      } else test(object as Mesh, object.matrixWorld)
    })
    return boxes
  }
  const sweep = (from: Vector3, to: Vector3, radius: number, verticalSpan = 0): number => {
    const started = performance.now()
    let fraction = 1
    const distance = from.distanceTo(to)
    if (distance < 1e-12) return fraction
    queryBounds.set(from, from).expandByPoint(to).expandByScalar(radius)
    queryBounds.min.y -= verticalSpan / 2; queryBounds.max.y += verticalSpan / 2
    const boxes = visitBoxes(radius, () => {
      localFrom.copy(from).applyMatrix4(inverse)
      localTo.copy(to).applyMatrix4(inverse)
      const entry = activePlanes ? hullEntry(localFrom, localTo, activePlanes, direction) : segmentEntry(localFrom, localTo, expanded)
      if (entry < 1) fraction = Math.min(fraction, Math.max(0, entry - 1e-5 / distance))
    }, verticalSpan, queryBounds)
    measure?.({ fraction, boxes, queryMs: performance.now() - started })
    return fraction
  }
  const intervals: Array<[number, number]> = []
  const overlaps = (position: Vector3, radius: number, verticalSpan = 0): boolean => {
    let occupied = false
    queryBounds.set(position, position).expandByScalar(radius)
    queryBounds.min.y -= verticalSpan / 2; queryBounds.max.y += verticalSpan / 2
    visitBoxes(radius, () => {
      localFrom.copy(position).applyMatrix4(inverse)
      if (activePlanes) {
        if (activePlanes.every(plane => plane.limit - plane.normal.dot(localFrom) > NAV_MOTION.positionEpsilon)) occupied = true
        return
      }
      if (localFrom.x > expanded.min.x + NAV_MOTION.positionEpsilon && localFrom.x < expanded.max.x - NAV_MOTION.positionEpsilon &&
          localFrom.y > expanded.min.y + NAV_MOTION.positionEpsilon && localFrom.y < expanded.max.y - NAV_MOTION.positionEpsilon &&
          localFrom.z > expanded.min.z + NAV_MOTION.positionEpsilon && localFrom.z < expanded.max.z - NAV_MOTION.positionEpsilon) occupied = true
    }, verticalSpan, queryBounds)
    return occupied
  }
  const up = new Vector3()
  const recover = (position: Vector3, radius: number): number => {
    const started = performance.now()
    intervals.length = 0
    queryBounds.set(position, position).expandByScalar(radius)
    queryBounds.min.y = -Infinity; queryBounds.max.y = Infinity
    const boxes = visitBoxes(radius, () => {
      localFrom.copy(position).applyMatrix4(inverse)
      const e = inverse.elements
      // World vertical expressed in box space; retain its scale so interval
      // endpoints remain metres of vertical motion, not local units.
      up.set(e[4], e[5], e[6])
      if (activePlanes) {
        const interval = hullInterval(localFrom, up, activePlanes)
        if (interval && interval[1] >= 0 && Number.isFinite(interval[1])) intervals.push(interval)
        return
      }
      let enter = -Infinity
      let leave = Infinity
      for (const axis of ['x', 'y', 'z'] as const) {
        if (Math.abs(up[axis]) < 1e-14) {
          if (localFrom[axis] < expanded.min[axis] || localFrom[axis] > expanded.max[axis]) return
          continue
        }
        const a = (expanded.min[axis] - localFrom[axis]) / up[axis]
        const b = (expanded.max[axis] - localFrom[axis]) / up[axis]
        enter = Math.max(enter, Math.min(a, b))
        leave = Math.min(leave, Math.max(a, b))
        if (enter > leave) return
      }
      if (leave >= 0 && Number.isFinite(leave)) intervals.push([enter, leave])
    }, 0, queryBounds)
    intervals.sort((a, b) => a[0] - b[0])
    let lift = 0
    // Merge all vertical overlaps before choosing an exit. Clearing one box
    // must not place the near plane inside a stacked or intersecting box.
    for (const [enter, leave] of intervals) {
      if (enter > lift) break
      if (leave >= lift) lift = leave + 1e-5
    }
    if (lift > 0) measure?.({ fraction: 0, boxes, queryMs: performance.now() - started, recoveryLift: lift })
    return lift
  }
  const bounds = (from: Vector3, to: Vector3, radius: number): Box3 | null => {
    const result = new Box3()
    queryBounds.set(from, from).expandByPoint(to).expandByScalar(radius)
    queryBounds.min.y = -Infinity; queryBounds.max.y = Infinity
    visitBoxes(radius, () => {
      worldBox.copy(expanded).applyMatrix4(worldTransform)
      if (worldBox.max.x < Math.min(from.x, to.x) || worldBox.min.x > Math.max(from.x, to.x) ||
          worldBox.max.z < Math.min(from.z, to.z) || worldBox.min.z > Math.max(from.z, to.z)) return
      result.union(worldBox)
    }, 0, queryBounds)
    return result.isEmpty() ? null : result
  }
  const ceiling = (from: Vector3, to: Vector3, radius: number): number => bounds(from, to, radius)?.max.y ?? -Infinity
  const outlines = (radius: number) => {
    const positions: number[] = []
    const corners = Array.from({ length: 8 }, () => new Vector3())
    const count = visitBoxes(radius, () => {
      for (let index = 0; index < 8; index++) corners[index]?.set(
        index & 1 ? expanded.max.x : expanded.min.x,
        index & 2 ? expanded.max.y : expanded.min.y,
        index & 4 ? expanded.max.z : expanded.min.z,
      ).applyMatrix4(worldTransform)
      for (let index = 0; index < 8; index++) {
        for (const bit of [1, 2, 4]) {
          if (index & bit) continue
          const from = corners[index]
          const to = corners[index | bit]
          if (from && to) positions.push(from.x, from.y, from.z, to.x, to.y, to.z)
        }
      }
    })
    return { count, positions: new Float32Array(positions) }
  }
  return Object.assign(sweep, { recover, ceiling, bounds, outlines, overlaps })
}
