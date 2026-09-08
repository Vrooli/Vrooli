import { sortSteps, type WorkProgress } from '../../cooperative'
import type { TeamInput } from '../../model'
import type { RoomLeaf } from './bsp'

export interface RoomAssignment {
  team: TeamInput
  leaf: RoomLeaf
}

/** Stable largest-demand-to-largest-leaf assignment; names and API order are irrelevant. */
export function assignRooms(teams: readonly TeamInput[], leaves: readonly RoomLeaf[]): RoomAssignment[] {
  const steps = assignRoomsSteps(teams, leaves)
  let step = steps.next()
  while (!step.done) step = steps.next()
  return step.value
}

export function* assignRoomsSteps(teams: readonly TeamInput[], leaves: readonly RoomLeaf[]): Generator<WorkProgress, RoomAssignment[]> {
  const ordered = yield* sortSteps(teams, (a, b) => b.memberIds.length - a.memberIds.length || a.id.localeCompare(b.id))
  const assignments: RoomAssignment[] = []
  for (const [index, team] of ordered.entries()) {
    if (index % 128 === 0) yield { completed: index, total: ordered.length }
    const leaf = leaves[index]
    if (leaf) assignments.push({ team, leaf })
  }
  return assignments
}
