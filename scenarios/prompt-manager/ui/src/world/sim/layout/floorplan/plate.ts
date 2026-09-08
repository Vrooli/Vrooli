import type { LayoutTuning } from '../../../config'
import type { Vec2 } from '../../model'
import type { WorkProgress } from '../../cooperative'
import type { Rng } from '../../rng'

export interface Rect {
  x: number
  z: number
  width: number
  depth: number
}

/** Size a single plate from the furniture minimums and headcount area demand. */
export function floorplate(roomSizes: readonly Vec2[], memberCount: number, tuning: LayoutTuning, rng: Rng): Rect {
  const steps = floorplateSteps(roomSizes, memberCount, tuning, rng)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

export function* floorplateSteps(roomSizes: readonly Vec2[], memberCount: number, tuning: LayoutTuning, rng: Rng): Generator<WorkProgress, Rect> {
  const margin = tuning.floorplan.plateMargin
  const corridor = tuning.floorplan.corridorWidth
  let widestRoom = Math.max(tuning.roomWidth, tuning.tableSeatRadius * 4 + tuning.deskInset * 2)
  let deepestRoom = tuning.roomDepth
  let totalWidth = 0
  let roomArea = 0
  for (const [index, size] of roomSizes.entries()) {
    if (index % 128 === 0) yield { completed: index, total: roomSizes.length }
    widestRoom = Math.max(widestRoom, size[0])
    deepestRoom = Math.max(deepestRoom, size[1])
    totalWidth += size[0]
    roomArea += Math.max(tuning.floorplan.roomMinArea, size[0] * size[1])
  }
  const corridorColumns = Math.round(tuning.floorplan.secondaryCorridors.max) + 1
  const roomColumns = Math.max(corridorColumns, Math.ceil(roomSizes.length / 2) + 1)
  const minimumWidth = Math.max(
    totalWidth / 2 + margin * 2 + corridor,
    widestRoom * roomColumns + corridor * (corridorColumns - 1) + margin * 2,
  )
  const minimumDepth = deepestRoom * 2 + corridor * 2 + margin * 2
  const requestedArea = roomArea
    + memberCount * tuning.floorplan.roomAreaPerMember
    + margin * margin * 4
  const aspect = rng.range(tuning.floorplan.plateAspect.min, tuning.floorplan.plateAspect.max)
  const width = Math.max(minimumWidth, Math.sqrt(requestedArea * aspect))
  const depth = Math.max(minimumDepth, requestedArea / width)
  return { x: 0, z: 0, width, depth }
}
