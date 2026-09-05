import { describe, expect, it } from 'vitest'
import { tuning } from '../config'
import type { Place } from '../sim'
import { buildRoomSlabs } from './Places'

describe('buildRoomSlabs', () => {
  it('preserves default slab dimensions and responds to configured surface dimensions', () => {
    const room: Place = { id: 'room:a', kind: 'room', teamId: 'a', position: [2, 3], rotation: 0, size: [8, 6], seats: [], label: 'A' }
    const original = buildRoomSlabs([room], [], tuning.layout, () => 4, true)
    expect(original.floors[0]).toMatchObject({ position: [2, 4.012, 3], scale: [8, 0.02, 6] })
    expect(original.walls[0]?.scale).toEqual([8.18, tuning.layout.wallHeight, 0.18])
    const layout = { ...tuning.layout, surfaces: { ...tuning.layout.surfaces, wallThickness: 0.3, floorLift: 0.05, floorThickness: 0.08, doorFrameScale: 2 } }
    const changed = buildRoomSlabs([room], [], layout, () => 4, true)
    expect(changed.floors[0]).toMatchObject({ position: [2, 4.05, 3], scale: [8, 0.08, 6] })
    expect(changed.walls[0]?.scale).toEqual([8.3, layout.wallHeight, 0.3])
    expect(changed.walls.find((wall) => wall.key.includes(':door-jamb:'))?.scale).toEqual([0.6, layout.wallHeight, 0.6])
  })

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
