import { Camera, Frustum, InstancedMesh, Matrix4, Sphere, Vector3 } from 'three'
import { CameraMotionGate } from '../engine/camera/motion'

/* eslint-disable @typescript-eslint/no-non-null-assertion -- Array indices are bounded by their owning lengths; heap indices are bounded by size. */

export const MATRIX_ELEMENTS = 16
export interface VegetationCullItem {
  key: string
  group: number
  center: readonly [number, number, number]
  radius: number
  matrix: Float32Array
  color?: readonly [number, number, number]
}

/** The global driver owns buffers; mesh components only register consumers. */
export class VegetationBuffer {
  readonly matrices: Float32Array
  readonly colors: Float32Array
  readonly meshes: Array<InstancedMesh | null> = []
  count = 0
  constructor(readonly capacity: number) {
    this.matrices = new Float32Array(capacity * MATRIX_ELEMENTS)
    this.colors = new Float32Array(capacity * 3)
  }
  upload(mesh: InstancedMesh): void {
    mesh.count = this.count
    mesh.instanceMatrix.array.set(this.matrices)
    mesh.instanceMatrix.needsUpdate = true
    if (mesh.instanceColor) {
      mesh.instanceColor.array.set(this.colors)
      mesh.instanceColor.needsUpdate = true
    }
  }
}

/** Reusable nearest-K heap: O(N log K), no frame-path temporary collections. */
export class VegetationCuller {
  private readonly sphere = new Sphere()
  private readonly frustum = new Frustum()
  private readonly viewProjection = new Matrix4()
  private readonly motion = new CameraMotionGate()
  private readonly heap: Int32Array
  private readonly candidates: Int32Array
  private readonly distances: Float64Array
  private readonly selected: Uint8Array
  runs = 0
  skips = 0
  constructor(readonly items: readonly VegetationCullItem[], readonly buffers: readonly VegetationBuffer[], readonly budget: number) {
    this.heap = new Int32Array(items.length)
    this.candidates = new Int32Array(items.length)
    this.distances = new Float64Array(items.length)
    this.selected = new Uint8Array(items.length)
  }
  private farther(a: number, b: number): boolean {
    return this.distances[a]! > this.distances[b]! || (this.distances[a] === this.distances[b] && a > b)
  }
  update(camera: Camera, metres: number, radians: number): boolean {
    if (!this.motion.changed(camera, metres, radians)) {
      this.skips += 1
      return false
    }
    this.viewProjection.multiplyMatrices(camera.projectionMatrix, camera.matrixWorldInverse)
    this.frustum.setFromProjectionMatrix(this.viewProjection)
    this.select(this.frustum, this.motion.position)
    this.runs += 1
    return true
  }
  select(frustum: Frustum, position: Vector3): void {
    let size = 0
    const limit = Math.max(0, Math.min(this.items.length, Math.floor(this.budget)))
    this.selected.fill(0)
    let visibleCount = 0
    for (let i = 0; i < this.items.length && limit > 0; i += 1) {
      const item = this.items[i]!
      this.sphere.center.set(item.center[0], item.center[1], item.center[2])
      this.sphere.radius = item.radius
      if (!frustum.intersectsSphere(this.sphere)) continue
      this.distances[i] = position.distanceToSquared(this.sphere.center)
      this.candidates[visibleCount++] = i
    }
    if (visibleCount <= limit) {
      for (let i = 0; i < visibleCount; i += 1) this.selected[this.candidates[i]!] = 1
    } else {
      for (let candidate = 0; candidate < visibleCount; candidate += 1) {
        const i = this.candidates[candidate]!
        if (size < limit) {
          let child = size++
          while (child > 0) {
            const parent = (child - 1) >> 1
            if (!this.farther(i, this.heap[parent]!)) break
            this.heap[child] = this.heap[parent]!
            child = parent
          }
          this.heap[child] = i
        } else if (this.farther(this.heap[0]!, i)) {
          let parent = 0
          while (parent * 2 + 1 < size) {
            let child = parent * 2 + 1
            if (child + 1 < size && this.farther(this.heap[child + 1]!, this.heap[child]!)) child += 1
            if (!this.farther(this.heap[child]!, i)) break
            this.heap[parent] = this.heap[child]!
            parent = child
          }
          this.heap[parent] = i
        }
      }
      for (let i = 0; i < size; i += 1) this.selected[this.heap[i]!] = 1
    }
    for (let i = 0; i < this.buffers.length; i += 1) this.buffers[i]!.count = 0
    for (let i = 0; i < this.items.length; i += 1) {
      if (!this.selected[i]) continue
      const item = this.items[i]!
      const buffer = this.buffers[item.group]!
      if (buffer.count >= buffer.capacity) continue
      const slot = buffer.count++
      buffer.matrices.set(item.matrix, slot * MATRIX_ELEMENTS)
      for (let channel = 0; channel < 3; channel += 1) buffer.colors[slot * 3 + channel] = item.color?.[channel] ?? 1
    }
    for (let i = 0; i < this.buffers.length; i += 1) {
      const buffer = this.buffers[i]!
      for (let j = 0; j < buffer.meshes.length; j += 1) {
        const mesh = buffer.meshes[j]
        if (mesh) buffer.upload(mesh)
      }
    }
  }
}
