import { Frustum, Sphere, Vector3 } from 'three'

export const MATRIX_ELEMENTS = 16

export interface VegetationCullItem {
  key: string
  center: readonly [number, number, number]
  radius: number
  matrix: Float32Array
  color?: readonly [number, number, number]
}

interface VisibleItem {
  item: VegetationCullItem
  distanceSq: number
  order: number
}

/**
 * Compact the visible vegetation matrices into `target`, nearest first when a
 * budget applies. Equal-distance items retain layout order so the budget edge
 * cannot flicker between frames.
 */
export function cullVegetation(
  items: readonly VegetationCullItem[],
  frustum: Frustum,
  cameraPosition: Vector3,
  target: Float32Array,
  budget: number,
  allowedKeys?: ReadonlySet<string>,
  targetColors?: Float32Array,
): number {
  const sphere = new Sphere()
  const center = sphere.center
  const visible: VisibleItem[] = []
  for (let order = 0; order < items.length; order += 1) {
    const item = items[order]
    if (!item) continue
    if (allowedKeys && !allowedKeys.has(item.key)) continue
    center.set(item.center[0], item.center[1], item.center[2])
    sphere.radius = item.radius
    if (!frustum.intersectsSphere(sphere)) continue
    visible.push({ item, distanceSq: cameraPosition.distanceToSquared(center), order })
  }
  if (visible.length > budget) visible.sort((a, b) => a.distanceSq - b.distanceSq || a.order - b.order)
  const count = Math.min(visible.length, Math.max(0, budget), Math.floor(target.length / MATRIX_ELEMENTS))
  for (let index = 0; index < count; index += 1) {
    const item = visible[index]?.item
    if (item) {
      target.set(item.matrix, index * MATRIX_ELEMENTS)
      if (targetColors) targetColors.set(item.color ?? [1, 1, 1], index * 3)
    }
  }
  return count
}

/** Select the nearest visible instances once across all prop groups. */
export function visibleVegetationKeys(items: readonly VegetationCullItem[], frustum: Frustum, cameraPosition: Vector3, budget: number): ReadonlySet<string> {
  const sphere = new Sphere()
  const visible: VisibleItem[] = []
  for (let order = 0; order < items.length; order += 1) {
    const item = items[order]
    if (!item) continue
    sphere.center.set(item.center[0], item.center[1], item.center[2])
    sphere.radius = item.radius
    if (!frustum.intersectsSphere(sphere)) continue
    visible.push({ item, distanceSq: cameraPosition.distanceToSquared(sphere.center), order })
  }
  visible.sort((a, b) => a.distanceSq - b.distanceSq || a.order - b.order)
  return new Set(visible.slice(0, Math.max(0, budget)).map(({ item }) => item.key))
}

export function matrixBufferChanged(previous: Float32Array, next: Float32Array, count: number, previousCount: number): boolean {
  if (count !== previousCount) return true
  const length = count * MATRIX_ELEMENTS
  for (let index = 0; index < length; index += 1) if (previous[index] !== next[index]) return true
  return false
}
