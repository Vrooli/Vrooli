import type { LayoutTuning } from '../../../config'
import type { Rng } from '../../rng'
import { sortSteps, type WorkProgress } from '../../cooperative'
import type { CorridorPlan } from './corridors'

export type RoomLeaf = CorridorPlan['blocks'][number]

function area(rect: RoomLeaf): number { return rect.width * rect.depth }

/** Seeded binary subdivision parallel to the corridor-facing edge. */
export function bspLeaves(blocks: readonly RoomLeaf[], targetCount: number, tuning: LayoutTuning, rng: Rng): RoomLeaf[] {
  const steps = bspLeavesSteps(blocks, targetCount, tuning, rng)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

export function* bspLeavesSteps(blocks: readonly RoomLeaf[], targetCount: number, tuning: LayoutTuning, rng: Rng): Generator<WorkProgress, RoomLeaf[]> {
  if (targetCount <= 0) return []
  const order = (a: RoomLeaf, b: RoomLeaf) => area(b) - area(a) || a.x - b.x || a.z - b.z
  const sorted = yield* sortSteps(blocks, order)
  let leaves: RoomLeaf[] = []
  for (let index = 0; index < Math.min(targetCount, sorted.length); index++) {
    if (index % 128 === 0) yield { completed: index, total: sorted.length }
    const leaf = sorted[index]
    if (leaf) leaves.push(leaf)
  }
  while (leaves.length < targetCount) {
    let selected: { leaf: RoomLeaf; index: number } | undefined
    for (let index = 0; index < leaves.length; index++) {
      if (index % 128 === 0) yield { completed: index, total: leaves.length }
      const leaf = leaves[index]
      if (leaf && leaf.width >= tuning.floorplan.doorWidth * 2 && (!selected || area(leaf) > area(selected.leaf))) selected = { leaf, index }
    }
    if (!selected) break
    const ratio = rng.range(tuning.floorplan.splitRatio.min, tuning.floorplan.splitRatio.max)
    const gap = tuning.cellSize
    const usableWidth = selected.leaf.width - gap
    const leftWidth = usableWidth * ratio
    const rightWidth = usableWidth - leftWidth
    const left: RoomLeaf = { ...selected.leaf, x: selected.leaf.x - selected.leaf.width / 2 + leftWidth / 2, width: leftWidth }
    const right: RoomLeaf = { ...selected.leaf, x: selected.leaf.x + selected.leaf.width / 2 - rightWidth / 2, width: rightWidth }
    const next: RoomLeaf[] = []
    for (let index = 0; index < leaves.length; index++) {
      if (index % 128 === 0) yield { completed: index, total: leaves.length }
      const leaf = leaves[index]
      if (index === selected.index) next.push(left, right)
      else if (leaf) next.push(leaf)
    }
    leaves = next
  }
  return yield* sortSteps(leaves, order)
}
