import type { TeamInput } from '../../model'
import type { RoomLeaf } from './bsp'

export interface RoomAssignment {
  team: TeamInput
  leaf: RoomLeaf
}

/** Stable largest-demand-to-largest-leaf assignment; names and API order are irrelevant. */
export function assignRooms(teams: readonly TeamInput[], leaves: readonly RoomLeaf[]): RoomAssignment[] {
  const ordered = [...teams].sort((a, b) => b.memberIds.length - a.memberIds.length || a.id.localeCompare(b.id))
  return ordered.flatMap((team, index) => leaves[index] ? [{ team, leaf: leaves[index] }] : [])
}
