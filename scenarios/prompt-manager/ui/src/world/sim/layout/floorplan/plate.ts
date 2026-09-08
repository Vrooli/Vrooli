import type { LayoutTuning } from '../../../config'
import type { Vec2 } from '../../model'
import type { WorkProgress } from '../../cooperative'
import type { Rng } from '../../rng'
import { architecture } from '../../../config/architecture'

export interface Rect {
  x: number
  z: number
  width: number
  depth: number
}

export interface OfficeWingPlan {
  plate: Rect
  rooms: Array<Rect & { side: 'north' | 'south' }>
  corridors: Rect[]
  lounge: Rect
  kitchen: Rect
}

/** Rooms occupy paired corridor wings. A shared spine connects every wing. */
export function officeWings(roomSizes: readonly Vec2[], tuning: LayoutTuning): OfficeWingPlan {
  const { sharedWidth, roomGap, margin } = architecture.office
  const columns = Math.max(1, Math.ceil(Math.sqrt(roomSizes.length / 2)))
  const wings = Math.max(1, Math.ceil(roomSizes.length / (columns * 2)))
  const corridor = tuning.floorplan.corridorWidth
  const widths = Array.from({ length: columns }, (_, column) => Math.max(tuning.roomWidth,
    ...Array.from({ length: wings * 2 }, (_, index) => roomSizes[Math.floor(index / 2) * columns * 2 + column * 2 + index % 2]?.[0] ?? 0)))
  const depths = Array.from({ length: wings }, (_, wing) => [0, 1].map(side => {
    let depth = 0
    for (let column = 0; column < columns; column++) {
      depth = Math.max(depth, roomSizes[wing * columns * 2 + column * 2 + side]?.[1] ?? 0)
    }
    return depth
  }))
  const width = sharedWidth + corridor + widths.reduce((sum, value) => sum + value + roomGap, 0) + margin * 2
  const depth = Math.max(sharedWidth * 2 + corridor, depths.reduce((sum, value) => sum + (value[0] ?? 0) + (value[1] ?? 0) + corridor + roomGap, 0)) + margin * 2
  const spineX = -width / 2 + margin + sharedWidth + corridor / 2
  const corridors: Rect[] = [{ x: spineX, z: 0, width: corridor, depth: depth - margin * 2 }]
  const rooms: OfficeWingPlan['rooms'] = []
  let bottom = -depth / 2 + margin
  for (let wing = 0; wing < wings; wing++) {
    const northDepth = depths[wing]?.[0] ?? 0
    const southDepth = depths[wing]?.[1] ?? 0
    const z = bottom + northDepth + corridor / 2
    const end = width / 2 - margin
    corridors.push({ x: (spineX + end) / 2, z, width: end - spineX, depth: corridor })
    let left = spineX + corridor / 2 + roomGap
    for (let column = 0; column < columns; column++) {
      for (let side = 0; side < 2; side++) {
        const index = wing * columns * 2 + column * 2 + side
        const size = roomSizes[index]
        if (!size) continue
        rooms[index] = { x: left + size[0] / 2, z: z + (side === 0 ? -1 : 1) * (corridor / 2 + size[1] / 2), width: size[0], depth: size[1], side: side === 0 ? 'south' : 'north' }
      }
      left += (widths[column] ?? tuning.roomWidth) + roomGap
    }
    bottom += northDepth + southDepth + corridor + roomGap
  }
  const sharedX = -width / 2 + margin + sharedWidth / 2
  return { plate: { x: 0, z: 0, width, depth }, rooms, corridors,
    lounge: { x: sharedX, z: -sharedWidth / 2 - corridor / 2, width: sharedWidth, depth: sharedWidth },
    kitchen: { x: sharedX, z: sharedWidth / 2 + corridor / 2, width: sharedWidth, depth: sharedWidth } }
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
