import type { LayoutTuning } from '../../../config'
import type { Rng } from '../../rng'
import type { CorridorPlan } from './corridors'

export type RoomLeaf = CorridorPlan['blocks'][number]

function area(rect: RoomLeaf): number { return rect.width * rect.depth }

/** Seeded binary subdivision parallel to the corridor-facing edge. */
export function bspLeaves(blocks: readonly RoomLeaf[], targetCount: number, tuning: LayoutTuning, rng: Rng): RoomLeaf[] {
  if (targetCount <= 0) return []
  const leaves = [...blocks]
    .sort((a, b) => area(b) - area(a) || a.x - b.x || a.z - b.z)
    .slice(0, Math.min(targetCount, blocks.length))
  while (leaves.length < targetCount) {
    const candidates = leaves
      .map((leaf, index) => ({ leaf, index }))
      .filter(({ leaf }) => leaf.width >= tuning.floorplan.doorWidth * 2)
      .sort((a, b) => area(b.leaf) - area(a.leaf) || a.index - b.index)
    const selected = candidates[0]
    if (!selected) break
    const ratio = rng.range(tuning.floorplan.splitRatio.min, tuning.floorplan.splitRatio.max)
    const gap = tuning.cellSize
    const usableWidth = selected.leaf.width - gap
    const leftWidth = usableWidth * ratio
    const rightWidth = usableWidth - leftWidth
    const left: RoomLeaf = { ...selected.leaf, x: selected.leaf.x - selected.leaf.width / 2 + leftWidth / 2, width: leftWidth }
    const right: RoomLeaf = { ...selected.leaf, x: selected.leaf.x + selected.leaf.width / 2 - rightWidth / 2, width: rightWidth }
    leaves.splice(selected.index, 1, left, right)
  }
  return leaves.sort((a, b) => area(b) - area(a) || a.x - b.x || a.z - b.z).slice(0, targetCount)
}
