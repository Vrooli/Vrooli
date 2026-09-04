import type { TerrainTuning } from '../../config'
import type { TerrainField } from './field'
import { shoreDistance, waterComponentLabels, wetHeight } from './water'

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

function addPolygon(surface: WaterSurfaceData, polygon: readonly Point[], field: TerrainField, tuning: TerrainTuning): void {
  if (polygon.length < 3) return
  const base = surface.positions.length / 3
  for (const point of polygon) {
    surface.positions.push(point.x, tuning.waterLevel + tuning.waterSurfaceLift, point.z)
    surface.shore.push(Math.max(0, -shoreDistance(field, tuning, point.x, point.z)))
  }
  for (let index = 1; index < polygon.length - 1; index += 1) surface.indices.push(base, base + index, base + index + 1)
}

/** Marching-squares clipping, grouped into one triangle buffer per connected pond. */
export function waterSurfaceComponents(field: TerrainField, tuning: TerrainTuning): WaterSurfaceData[] {
  const { components, labels } = waterComponentLabels(field, tuning)
  const surfaces = Array.from({ length: components }, (_, component): WaterSurfaceData => ({ component, positions: [], shore: [], indices: [] }))
  const point = (col: number, row: number): Point => {
    const x = field.originX + col * field.cellSize
    const z = field.originZ + row * field.cellSize
    return { x, z, signed: wetHeight(field, tuning, x, z) - tuning.waterLevel, label: labels[row * field.cols + col] ?? -1 }
  }
  for (let row = 0; row < field.rows - 1; row += 1) {
    for (let col = 0; col < field.cols - 1; col += 1) {
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
