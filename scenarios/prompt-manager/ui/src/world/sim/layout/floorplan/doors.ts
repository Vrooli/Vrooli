import type { LayoutTuning } from '../../../config'
import type { Place } from '../../model'
import type { RoomAssignment } from './assign'
import type { Rect } from './plate'

export function doorwayFor(assignment: RoomAssignment, primary: Rect, tuning: LayoutTuning): Place {
  const { team, leaf } = assignment
  const z = leaf.side === 'north' ? primary.z + primary.depth / 2 : primary.z - primary.depth / 2
  return { id: `door:${team.id}`, kind: 'door', teamId: team.id, parentId: `room:${team.id}`, position: [leaf.x, z], rotation: leaf.side === 'north' ? Math.PI : 0, size: [Math.min(tuning.floorplan.doorWidth, leaf.width), tuning.cellSize], seats: [], label: `${team.name} door` }
}
