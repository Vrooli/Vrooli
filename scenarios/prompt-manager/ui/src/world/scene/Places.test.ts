import { describe, expect, it } from 'vitest'
import { tuning } from '../config'
import type { Place } from '../sim'
import { buildRoomSlabs } from './Places'

describe('buildRoomSlabs', () => {
  it('encloses an indoor room around the configured doorway gap and frame', () => {
    const room: Place = { id: 'room:a', kind: 'room', teamId: 'a', position: [0, 0], rotation: 0, size: [8, 6], seats: [], label: 'A' }
    const door: Place = { id: 'door:a', kind: 'door', teamId: 'a', parentId: room.id, position: [0, 3], rotation: 0, size: [tuning.layout.floorplan.doorWidth, tuning.layout.cellSize], seats: [], label: 'A door' }
    const { walls, floors } = buildRoomSlabs([room], [door], tuning.layout, () => 0, true)
    expect(floors).toHaveLength(1)
    expect(walls.filter((wall) => wall.key.includes(':front:'))).toHaveLength(2)
    expect(walls.filter((wall) => wall.key.includes(':door-jamb:'))).toHaveLength(2)
    expect(walls.some((wall) => wall.key.endsWith(':door-lintel'))).toBe(true)
    const frontWidth = walls.filter((wall) => wall.key.includes(':front:')).reduce((sum, wall) => sum + wall.scale[0], 0)
    expect(room.size[0] - frontWidth).toBeCloseTo(door.size[0])
  })
})
