import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { generateLayout, scatterTrees, COMMONS_ID, CAMPFIRE_ID, BOARD_ID, roomId, deskId, tableId } from '../layout/generate'
import { makeTeams } from './fixtures'

const opts = { seed: 3, trees: true, treeVariants: 3 }

describe('layout generation', () => {
  it('produces one room, one table per team and one desk per member, keyed by ids', () => {
    const { teams, agents } = makeTeams(3, 4)
    const layout = generateLayout(teams, agents, tuning.layout, opts)
    const kinds = layout.places.reduce<Record<string, number>>((acc, p) => ({ ...acc, [p.kind]: (acc[p.kind] ?? 0) + 1 }), {})
    expect(kinds.room).toBe(3)
    expect(kinds.table).toBe(3)
    expect(kinds.desk).toBe(12)
    expect(kinds.commons).toBe(1)
    expect(kinds.campfire).toBe(1)
    expect(kinds.board).toBe(1)
    expect(layout.places.map((p) => p.id)).toContain(roomId('team-1'))
    expect(layout.places.map((p) => p.id)).toContain(deskId('agent-2-3'))
    expect(layout.places.map((p) => p.id)).toContain(tableId('team-0'))
    expect(Object.keys(layout.deskSeatByAgent)).toHaveLength(12)
  })

  it('is stable across calls and after a team rename', () => {
    const { teams, agents } = makeTeams(2, 3)
    const a = generateLayout(teams, agents, tuning.layout, opts)
    const renamed = teams.map((t) => ({ ...t, name: `${t.name} renamed` }))
    const b = generateLayout(renamed, agents, tuning.layout, opts)
    expect(a.places.map((p) => [p.id, p.position])).toEqual(b.places.map((p) => [p.id, p.position]))
    expect(a.decor).toEqual(b.decor)
    expect(a.bounds).toEqual(b.bounds)
  })

  it('adding a member keeps other rooms in place', () => {
    const { teams, agents } = makeTeams(2, 2)
    const before = generateLayout(teams, agents, tuning.layout, opts)
    const grown = { teams: teams.map((t, i) => (i === 1 ? { ...t, memberIds: [...t.memberIds, 'new'] } : t)), agents: [...agents, { id: 'new', name: 'New' }] }
    const after = generateLayout(grown.teams, grown.agents, tuning.layout, opts)
    const room0Before = before.places.find((p) => p.id === roomId('team-0'))
    const room0After = after.places.find((p) => p.id === roomId('team-0'))
    expect(room0After?.position).toEqual(room0Before?.position)
    expect(after.places.filter((p) => p.kind === 'desk')).toHaveLength(5)
  })

  it('lays rooms behind the commons on a grid that wraps at maxRoomsPerRow', () => {
    const { teams, agents } = makeTeams(tuning.layout.maxRoomsPerRow + 2, 1)
    const layout = generateLayout(teams, agents, tuning.layout, opts)
    const rooms = layout.places.filter((p) => p.kind === 'room')
    const rowsZ = new Set(rooms.map((r) => r.position[1].toFixed(3)))
    expect(rowsZ.size).toBe(2)
    for (const room of rooms) expect(room.position[1]).toBeLessThan(-tuning.layout.commonsRadius)
    expect(layout.bounds.width).toBeGreaterThanOrEqual(tuning.layout.minSlabWidth)
    expect(layout.bounds.depth).toBeGreaterThan(tuning.layout.minSlabDepth)
  })

  it('an empty team graph still yields the commons, campfire and board on the minimum slab', () => {
    const layout = generateLayout([], [], tuning.layout, opts)
    expect(layout.places.map((p) => p.id).sort()).toEqual([BOARD_ID, CAMPFIRE_ID, COMMONS_ID].sort())
    expect(layout.bounds.width).toBeGreaterThanOrEqual(tuning.layout.minSlabWidth)
    expect(layout.bounds.depth).toBeGreaterThanOrEqual(tuning.layout.minSlabDepth)
  })

  it('desks sit inside their room and their seats face the desk', () => {
    const { teams, agents } = makeTeams(1, 5)
    const layout = generateLayout(teams, agents, tuning.layout, opts)
    const room = layout.places.find((p) => p.kind === 'room')
    expect(room).toBeDefined()
    if (!room) return
    for (const desk of layout.places.filter((p) => p.kind === 'desk')) {
      expect(Math.abs(desk.position[0] - room.position[0])).toBeLessThan(room.size[0] / 2)
      expect(Math.abs(desk.position[1] - room.position[1])).toBeLessThan(room.size[1] / 2)
      expect(desk.seats[0]?.position[1]).toBeGreaterThan(desk.position[1])
      expect(desk.seats[0]?.facing).toBeCloseTo(Math.PI, 6)
    }
  })

  it('places no tree within the clearing radius of any place or clear point', () => {
    const { teams, agents } = makeTeams(3, 3)
    const clearPoints = [[0, 12] as const]
    const layout = generateLayout(teams, agents, tuning.layout, { ...opts, clearPoints: [...clearPoints] })
    expect(layout.decor.length).toBeGreaterThan(0)
    for (const tree of layout.decor) {
      for (const place of layout.places) {
        const reach = Math.hypot(place.size[0], place.size[1]) / 2 + tuning.layout.clearingRadius
        expect(Math.hypot(tree.position[0] - place.position[0], tree.position[1] - place.position[1])).toBeGreaterThanOrEqual(reach)
      }
      for (const point of clearPoints) {
        expect(Math.hypot(tree.position[0] - point[0], tree.position[1] - point[1])).toBeGreaterThanOrEqual(tuning.layout.clearingRadius)
      }
      expect(Math.abs(tree.position[0] - layout.bounds.center[0])).toBeLessThanOrEqual(layout.bounds.width / 2 - tuning.layout.treeMargin)
      expect(tree.variant).toBeLessThan(3)
    }
  })

  it('tree scatter is deterministic for a seed and honours spacing', () => {
    const { teams, agents } = makeTeams(1, 1)
    const layout = generateLayout(teams, agents, tuning.layout, opts)
    const again = scatterTrees(layout.places, layout.bounds, tuning.layout, 3, [], 3)
    expect(again).toEqual(layout.decor)
    for (let i = 0; i < layout.decor.length; i += 1) {
      for (let j = i + 1; j < layout.decor.length; j += 1) {
        const a = layout.decor[i]
        const b = layout.decor[j]
        if (!a || !b) continue
        expect(Math.hypot(a.position[0] - b.position[0], a.position[1] - b.position[1])).toBeGreaterThanOrEqual(tuning.layout.treeMargin)
      }
    }
  })

  it('indoor scenes get no trees', () => {
    const { teams, agents } = makeTeams(1, 1)
    expect(generateLayout(teams, agents, tuning.layout, { ...opts, trees: false }).decor).toEqual([])
  })

  it('overrides move a room with its desks and seats, and remove a room with its children', () => {
    const { teams, agents } = makeTeams(2, 2)
    const moved = generateLayout(teams, agents, tuning.layout, { ...opts, overrides: [{ placeId: roomId('team-0'), position: [40, -30] }] })
    const room = moved.places.find((p) => p.id === roomId('team-0'))
    const desk = moved.places.find((p) => p.id === deskId('agent-0-0'))
    expect(room?.position).toEqual([40, -30])
    expect(desk?.position[0]).toBeGreaterThan(30)
    expect(desk?.seats[0]?.position[1]).toBeGreaterThan(-40)
    const removed = generateLayout(teams, agents, tuning.layout, { ...opts, overrides: [{ placeId: roomId('team-1'), removed: true }] })
    expect(removed.places.find((p) => p.id === roomId('team-1'))).toBeUndefined()
    expect(removed.places.find((p) => p.id === deskId('agent-1-0'))).toBeUndefined()
    expect(removed.places.find((p) => p.id === tableId('team-1'))).toBeUndefined()
    expect(removed.deskSeatByAgent['agent-1-0']).toBeUndefined()
    expect(removed.deskSeatByAgent['agent-0-0']).toBeDefined()
  })
})
