import type { LayoutTuning } from '../../../config'
import type { Vec2 } from '../../model'
import type { Rng } from '../../rng'

export interface Rect {
  x: number
  z: number
  width: number
  depth: number
}

/** Size a single plate from the furniture minimums and headcount area demand. */
export function floorplate(roomSizes: readonly Vec2[], memberCount: number, tuning: LayoutTuning, rng: Rng): Rect {
  const margin = tuning.floorplan.plateMargin
  const corridor = tuning.floorplan.corridorWidth
  const widestRoom = Math.max(tuning.roomWidth, ...roomSizes.map((size) => size[0]), tuning.tableSeatRadius * 4 + tuning.deskInset * 2)
  const corridorColumns = Math.round(tuning.floorplan.secondaryCorridors.max) + 1
  const roomColumns = Math.max(corridorColumns, Math.ceil(roomSizes.length / 2) + 1)
  const minimumWidth = Math.max(
    roomSizes.reduce((sum, size) => sum + size[0], 0) / 2 + margin * 2 + corridor,
    widestRoom * roomColumns + corridor * (corridorColumns - 1) + margin * 2,
  )
  const minimumDepth = Math.max(tuning.roomDepth, ...roomSizes.map((size) => size[1])) * 2 + corridor * 2 + margin * 2
  const requestedArea = roomSizes.reduce((sum, size) => sum + Math.max(tuning.floorplan.roomMinArea, size[0] * size[1]), 0)
    + memberCount * tuning.floorplan.roomAreaPerMember
    + margin * margin * 4
  const aspect = rng.range(tuning.floorplan.plateAspect.min, tuning.floorplan.plateAspect.max)
  const width = Math.max(minimumWidth, Math.sqrt(requestedArea * aspect))
  const depth = Math.max(minimumDepth, requestedArea / width)
  return { x: 0, z: 0, width, depth }
}
