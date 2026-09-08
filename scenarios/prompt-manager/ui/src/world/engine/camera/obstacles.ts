import { NAV_MOTION } from '../../config/navigation'
import { Box3, InstancedMesh, Matrix4, Mesh, Vector3, type Object3D } from 'three'

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
  const visitBoxes = (radius: number, visit: () => void, verticalSpan = 0) => {
    let boxes = 0
    root.updateWorldMatrix(true, true)
    const test = (mesh: Mesh, transform: Matrix4) => {
      if (Math.abs(transform.determinant()) < 1e-12) return
      if (!mesh.geometry.boundingBox) mesh.geometry.computeBoundingBox()
      if (!mesh.geometry.boundingBox) return
      boxes++
      worldTransform.copy(transform)
      inverse.copy(transform).invert()
      const e = inverse.elements
      padding.set(Math.hypot(e[0], e[4], e[8]), Math.hypot(e[1], e[5], e[9]), Math.hypot(e[2], e[6], e[10])).multiplyScalar(radius)
      // A vertical capsule is a sphere swept along a vertical segment. Expand
      // by that segment's projection into each box-local axis as well.
      padding.x += Math.abs(e[4]) * verticalSpan / 2
      padding.y += Math.abs(e[5]) * verticalSpan / 2
      padding.z += Math.abs(e[6]) * verticalSpan / 2
      expanded.copy(mesh.geometry.boundingBox).expandByVector(padding)
      visit()
    }
    root.traverseVisible(object => {
      if (!(object instanceof Mesh)) return
      const mesh = object as Mesh
      let eligible = mesh.geometry.userData.cameraObstacle === 'box'
      if (includeFurniture) {
        for (let parent: Object3D | null = object; parent && !eligible; parent = parent.parent) eligible = parent.userData.walkObstacle === true
      }
      if (!eligible) return
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
    const boxes = visitBoxes(radius, () => {
      localFrom.copy(from).applyMatrix4(inverse)
      localTo.copy(to).applyMatrix4(inverse)
      const entry = segmentEntry(localFrom, localTo, expanded)
      if (entry < 1) fraction = Math.min(fraction, Math.max(0, entry - 1e-5 / distance))
    }, verticalSpan)
    measure?.({ fraction, boxes, queryMs: performance.now() - started })
    return fraction
  }
  const intervals: Array<[number, number]> = []
  const overlaps = (position: Vector3, radius: number, verticalSpan = 0): boolean => {
    let occupied = false
    visitBoxes(radius, () => {
      localFrom.copy(position).applyMatrix4(inverse)
      if (localFrom.x > expanded.min.x + NAV_MOTION.positionEpsilon && localFrom.x < expanded.max.x - NAV_MOTION.positionEpsilon &&
          localFrom.y > expanded.min.y + NAV_MOTION.positionEpsilon && localFrom.y < expanded.max.y - NAV_MOTION.positionEpsilon &&
          localFrom.z > expanded.min.z + NAV_MOTION.positionEpsilon && localFrom.z < expanded.max.z - NAV_MOTION.positionEpsilon) occupied = true
    }, verticalSpan)
    return occupied
  }
  const up = new Vector3()
  const recover = (position: Vector3, radius: number): number => {
    const started = performance.now()
    intervals.length = 0
    const boxes = visitBoxes(radius, () => {
      localFrom.copy(position).applyMatrix4(inverse)
      const e = inverse.elements
      // World vertical expressed in box space; retain its scale so interval
      // endpoints remain metres of vertical motion, not local units.
      up.set(e[4], e[5], e[6])
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
    })
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
    visitBoxes(radius, () => {
      worldBox.copy(expanded).applyMatrix4(worldTransform)
      if (worldBox.max.x < Math.min(from.x, to.x) || worldBox.min.x > Math.max(from.x, to.x) ||
          worldBox.max.z < Math.min(from.z, to.z) || worldBox.min.z > Math.max(from.z, to.z)) return
      result.union(worldBox)
    })
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
