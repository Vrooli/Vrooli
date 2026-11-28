import type { TerrainResolver } from '../../config'
import type { TerrainField } from './field'
import { shoreDistance, waterComponentLabelsSteps, wetHeight } from './water'

export interface WaterSurfaceData {
  component: number
  positions: number[]
  shore: number[]
  indices: number[]
}

interface Point {
  x: number
  z: number
  signed: number
  label: number
}

function crossing(a: Point, b: Point): Point {
  const denominator = a.signed - b.signed
  const t = Math.abs(denominator) < Number.EPSILON ? 0.5 : a.signed / denominator
  return { x: a.x + (b.x - a.x) * t, z: a.z + (b.z - a.z) * t, signed: 0, label: a.signed < 0 ? a.label : b.label }
}

function clippedPolygon(corners: readonly Point[]): Point[] {
  const polygon: Point[] = []
  for (let index = 0; index < corners.length; index += 1) {
    const current = corners[index]
    const next = corners[(index + 1) % corners.length]
    if (!current || !next) continue
    if (current.signed < 0) polygon.push(current)
    if ((current.signed < 0) !== (next.signed < 0)) polygon.push(crossing(current, next))
  }
  return polygon
}

function addPolygon(surface: WaterSurfaceData, polygon: readonly Point[], field: TerrainField, tuning: TerrainResolver): void {
  if (polygon.length < 3) return
  const base = surface.positions.length / 3
  for (const point of polygon) {
    const local = tuning.at(point.x, point.z)
    surface.positions.push(point.x, local.waterLevel + local.waterSurfaceLift, point.z)
    surface.shore.push(Math.max(0, -shoreDistance(field, tuning, point.x, point.z)))
  }
  for (let index = 1; index < polygon.length - 1; index += 1) surface.indices.push(base, base + index, base + index + 1)
}

/** Marching-squares clipping, grouped into one triangle buffer per connected pond. */
export function waterSurfaceComponents(...args: Parameters<typeof waterSurfaceComponentsSteps>): WaterSurfaceData[] {
  const steps = waterSurfaceComponentsSteps(...args)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

export function* waterSurfaceComponentsSteps(field: TerrainField, tuning: TerrainResolver): Generator<{ completed: number; total: number }, WaterSurfaceData[]> {
  const { components, labels } = yield* waterComponentLabelsSteps(field, tuning)
  const surfaces = Array.from({ length: components }, (_, component): WaterSurfaceData => ({ component, positions: [], shore: [], indices: [] }))
  const point = (col: number, row: number): Point => {
    const x = field.originX + col * field.cellSize
    const z = field.originZ + row * field.cellSize
    return { x, z, signed: wetHeight(field, tuning, x, z) - tuning.at(x, z).waterLevel, label: labels[row * field.cols + col] ?? -1 }
  }
  for (let row = 0; row < field.rows - 1; row += 1) {
    for (let col = 0; col < field.cols - 1; col += 1) {
      if (col % 128 === 0) yield { completed: row, total: field.rows }
      const corners = [point(col, row), point(col + 1, row), point(col + 1, row + 1), point(col, row + 1)] as const
      const wet = corners.map((corner) => corner.signed < 0)
      const wetIndices = wet.flatMap((value, index) => value ? [index] : [])
      const firstWet = wetIndices[0]
      if (firstWet === undefined) continue
      const cornerAt = (index: number): Point => corners[index] ?? corners[0]
      // Alternating saddle cases are two disjoint triangles, not one bow-tie polygon.
      const secondWet = wetIndices[1]
      if (wetIndices.length === 2 && secondWet !== undefined && Math.abs(firstWet - secondWet) === 2) {
        for (const cornerIndex of wetIndices) {
          const corner = cornerAt(cornerIndex)
          const previous = cornerAt((cornerIndex + 3) % 4)
          const next = cornerAt((cornerIndex + 1) % 4)
          const surface = surfaces[corner.label]
          if (surface) addPolygon(surface, [corner, crossing(corner, next), crossing(previous, corner)], field, tuning)
        }
        continue
      }
      const component = cornerAt(firstWet).label
      const surface = surfaces[component]
      if (surface) addPolygon(surface, clippedPolygon(corners), field, tuning)
    }
  }
  return surfaces.filter((surface) => surface.indices.length > 0)
}


export interface WaterGeometryData {
  component: number
  positions: Float32Array
  normals: Float32Array
  sphere: { center: [number, number, number]; radius: number }
  shore: Float32Array
  indices: Uint32Array
}

/** Pure mesh preparation. Published buffers are borrowed by the renderer. */
export function* waterGeometrySteps(field: TerrainField, tuning: TerrainResolver): Generator<{ completed: number; total: number }, WaterGeometryData[]> {
  const surfaces = yield* waterSurfaceComponentsSteps(field, tuning)
  const geometries: WaterGeometryData[] = []
  for (const surface of surfaces) {
    const positions = new Float32Array(surface.positions)
    const normals = new Float32Array(positions.length)
    const indices = new Uint32Array(surface.indices)
    for (let i = 0; i < indices.length; i += 3) {
      if (i % 384 === 0) yield { completed: i, total: indices.length }
      const a = (indices[i] ?? 0) * 3
      const b = (indices[i + 1] ?? 0) * 3
      const c = (indices[i + 2] ?? 0) * 3
      const cb = [0, 1, 2].map(axis => (positions[c + axis] ?? 0) - (positions[b + axis] ?? 0))
      const ab = [0, 1, 2].map(axis => (positions[a + axis] ?? 0) - (positions[b + axis] ?? 0))
      const cross = [(cb[1] ?? 0) * (ab[2] ?? 0) - (cb[2] ?? 0) * (ab[1] ?? 0), (cb[2] ?? 0) * (ab[0] ?? 0) - (cb[0] ?? 0) * (ab[2] ?? 0), (cb[0] ?? 0) * (ab[1] ?? 0) - (cb[1] ?? 0) * (ab[0] ?? 0)]
      for (const vertex of [a, b, c]) for (let axis = 0; axis < 3; axis++) normals[vertex + axis] = (normals[vertex + axis] ?? 0) + (cross[axis] ?? 0)
    }
    for (let i = 0; i < normals.length; i += 3) {
      if (i % 384 === 0) yield { completed: i, total: normals.length }
      const length = Math.hypot(normals[i] ?? 0, normals[i + 1] ?? 0, normals[i + 2] ?? 0) || 1
      for (let axis = 0; axis < 3; axis++) normals[i + axis] = (normals[i + axis] ?? 0) / length
    }
    const min = [Infinity, Infinity, Infinity]
    const max = [-Infinity, -Infinity, -Infinity]
    for (let i = 0; i < positions.length; i += 3) {
      if (i % 384 === 0) yield { completed: i, total: positions.length }
      for (let axis = 0; axis < 3; axis++) {
        min[axis] = Math.min(min[axis] ?? Infinity, positions[i + axis] ?? 0)
        max[axis] = Math.max(max[axis] ?? -Infinity, positions[i + axis] ?? 0)
      }
    }
    const center: [number, number, number] = [0, 1, 2].map(axis => ((min[axis] ?? 0) + (max[axis] ?? 0)) / 2) as [number, number, number]
    let radiusSquared = 0
    for (let i = 0; i < positions.length; i += 3) {
      if (i % 384 === 0) yield { completed: i, total: positions.length }
      const x = (positions[i] ?? 0) - center[0]
      const y = (positions[i + 1] ?? 0) - center[1]
      const z = (positions[i + 2] ?? 0) - center[2]
      radiusSquared = Math.max(radiusSquared, x * x + y * y + z * z)
    }
    geometries.push({ sphere: { center, radius: Math.sqrt(radiusSquared) }, component: surface.component, positions, normals, indices, shore: new Float32Array(surface.shore) })
  }
  return geometries
}
