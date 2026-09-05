import type { LayoutTuning } from '../../../config'
import type { Rng } from '../../rng'
import type { Rect } from './plate'

export interface CorridorPlan {
  primary: Rect
  secondary: Rect[]
  blocks: Array<Rect & { side: 'north' | 'south' }>
}

/** Cut the corridor spine before subdivision so every returned block touches it. */
export function cutCorridors(plate: Rect, tuning: LayoutTuning, rng: Rng): CorridorPlan {
  const width = tuning.floorplan.corridorWidth
  const margin = tuning.floorplan.plateMargin
  const primaryZ = rng.range(-width * tuning.floorplan.primaryOffset, width * tuning.floorplan.primaryOffset)
  const primary: Rect = { x: plate.x, z: primaryZ, width: plate.width - margin * 2, depth: width }
  const minCount = Math.round(tuning.floorplan.secondaryCorridors.min)
  const maxCount = Math.round(tuning.floorplan.secondaryCorridors.max)
  const count = minCount + rng.int(Math.max(1, maxCount - minCount + 1))
  const usableLeft = plate.x - plate.width / 2 + margin
  const usableRight = plate.x + plate.width / 2 - margin
  const secondary = Array.from({ length: count }, (_, index): Rect => {
    const even = usableLeft + ((index + 1) / (count + 1)) * (usableRight - usableLeft)
    const jitter = rng.range(-tuning.floorplan.secondaryJitter, tuning.floorplan.secondaryJitter) * (usableRight - usableLeft) / (count + 1)
    return { x: even + jitter, z: plate.z, width, depth: plate.depth - margin * 2 }
  }).sort((a, b) => a.x - b.x)
  const edges = [usableLeft, ...secondary.flatMap((corridor) => [corridor.x - width / 2, corridor.x + width / 2]), usableRight]
  const intervals: Array<[number, number]> = []
  for (let index = 0; index < edges.length - 1; index += 2) {
    const left = edges[index]
    const right = edges[index + 1]
    if (left !== undefined && right !== undefined && right - left >= tuning.floorplan.doorWidth) intervals.push([left, right])
  }
  const northEdge = plate.z + plate.depth / 2 - margin
  const southEdge = plate.z - plate.depth / 2 + margin
  const northStart = primaryZ + width / 2
  const southEnd = primaryZ - width / 2
  const blocks = intervals.flatMap(([left, right]) => {
    const centerX = (left + right) / 2
    return [
      { x: centerX, z: (northStart + northEdge) / 2, width: right - left, depth: northEdge - northStart, side: 'north' as const },
      { x: centerX, z: (southEdge + southEnd) / 2, width: right - left, depth: southEnd - southEdge, side: 'south' as const },
    ]
  }).filter((block) => block.width > 0 && block.depth > 0)
  return { primary, secondary, blocks }
}
