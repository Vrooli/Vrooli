import { describe, expect, it } from 'vitest'
import { uniformTerrain, biomeSets, tuning } from '../../config'
import { biomeGrid, buildTerrain } from '../terrain'
import { generateLayout, GATHERING_ID, HEARTH_ID, BOARD_ID, roomId, deskId, tableId, type GenerateOptions } from '../layout/generate'
import { makeTeams } from './fixtures'

function options(overrides: Partial<GenerateOptions> = {}): GenerateOptions {
  const seed = overrides.seed ?? 3
  const terrain = overrides.terrain ?? buildTerrain({ seed, tuning: uniformTerrain(tuning.terrain) })
  const biomeSet = overrides.biomeSet ?? biomeSets.park
  return {
    seed,
    scatterDecor: true,
    treeVariants: 3,
    terrain,
    terrainTuning: uniformTerrain(tuning.terrain),
    biomeSet,
    biomes: overrides.biomes ?? biomeGrid(terrain, uniformTerrain(tuning.terrain), biomeSet),
    ...overrides,
  }
}

describe('layout generation', () => {
  it('produces one room, one table per team and one desk per member, keyed by ids', () => {
    const { teams, agents } = makeTeams(3, 4)
    const layout = generateLayout(teams, agents, tuning.layout, options())
    const kinds = layout.places.reduce<Record<string, number>>((acc, p) => ({ ...acc, [p.kind]: (acc[p.kind] ?? 0) + 1 }), {})
    expect(kinds.room).toBe(3)
    expect(kinds.table).toBe(3)
    expect(kinds.desk).toBe(12)
    expect(kinds.gathering).toBe(1)
    expect(kinds.hearth).toBe(1)
    expect(kinds.board).toBe(1)
    expect(layout.places.map((p) => p.id)).toContain(roomId('team-1'))
    expect(layout.places.map((p) => p.id)).toContain(deskId('agent-2-3'))
    expect(layout.places.map((p) => p.id)).toContain(tableId('team-0'))
    expect(Object.keys(layout.deskSeatByAgent)).toHaveLength(12)
  })

  it('is stable across calls and after a team rename', () => {
    const { teams, agents } = makeTeams(2, 3)
    const a = generateLayout(teams, agents, tuning.layout, options())
    const renamed = teams.map((t) => ({ ...t, name: `${t.name} renamed` }))
    const b = generateLayout(renamed, agents, tuning.layout, options())
    expect(a.places.map((p) => [p.id, p.position])).toEqual(b.places.map((p) => [p.id, p.position]))
    expect(a.decor).toEqual(b.decor)
    expect(a.bounds).toEqual(b.bounds)
  })

  it('adding a member keeps other rooms in place', () => {
    const { teams, agents } = makeTeams(2, 2)
    const before = generateLayout(teams, agents, tuning.layout, options())
    const grown = { teams: teams.map((t, i) => (i === 1 ? { ...t, memberIds: [...t.memberIds, 'new'] } : t)), agents: [...agents, { id: 'new', name: 'New' }] }
    const after = generateLayout(grown.teams, grown.agents, tuning.layout, options())
    const room0Before = before.places.find((p) => p.id === roomId('team-0'))
    const room0After = after.places.find((p) => p.id === roomId('team-0'))
    expect(room0After?.position).toEqual(room0Before?.position)
    expect(after.places.filter((p) => p.kind === 'desk')).toHaveLength(5)
  })

  it('adding a team keeps every existing team on its site', () => {
    const { teams, agents } = makeTeams(2, 2)
    const before = generateLayout(teams, agents, tuning.layout, options())
    const grown = makeTeams(3, 2)
    const after = generateLayout(grown.teams, grown.agents, tuning.layout, options())
    for (const team of teams) {
      expect(after.places.find((place) => place.id === roomId(team.id))?.position).toEqual(before.places.find((place) => place.id === roomId(team.id))?.position)
    }
  })

  it('places rooms on distinct seeded sites with snapped rotations', () => {
    const { teams, agents } = makeTeams(6, 1)
    const layout = generateLayout(teams, agents, tuning.layout, options())
    const rooms = layout.places.filter((p) => p.kind === 'room')
    expect(new Set(rooms.map((room) => room.position.join(','))).size).toBe(rooms.length)
    for (const room of rooms) expect(room.rotation / (Math.PI / 12)).toBeCloseTo(Math.round(room.rotation / (Math.PI / 12)), 8)
    expect(layout.bounds.width).toBe(tuning.terrain.radius * 2)
    expect(layout.bounds.depth).toBe(tuning.terrain.radius * 2)
  })

  it('an empty team graph still yields the commons, campfire and board in the terrain bounds', () => {
    const layout = generateLayout([], [], tuning.layout, options())
    expect(layout.places.map((p) => p.id).sort()).toEqual([BOARD_ID, HEARTH_ID, GATHERING_ID].sort())
    expect(layout.bounds.width).toBe(tuning.terrain.radius * 2)
    expect(layout.bounds.depth).toBe(tuning.terrain.radius * 2)
  })

  it('desks sit inside their room and their seats face the desk', () => {
    const { teams, agents } = makeTeams(1, 5)
    const layout = generateLayout(teams, agents, tuning.layout, options())
    const room = layout.places.find((p) => p.kind === 'room')
    expect(room).toBeDefined()
    if (!room) return
    for (const desk of layout.places.filter((p) => p.kind === 'desk')) {
      const dx = desk.position[0] - room.position[0]
      const dz = desk.position[1] - room.position[1]
      const localX = dx * Math.cos(room.rotation) - dz * Math.sin(room.rotation)
      const localZ = dx * Math.sin(room.rotation) + dz * Math.cos(room.rotation)
      expect(Math.abs(localX)).toBeLessThan(room.size[0] / 2)
      expect(Math.abs(localZ)).toBeLessThan(room.size[1] / 2)
      const seat = desk.seats[0]
      expect(seat).toBeDefined()
      if (!seat) continue
      const towardDesk = Math.atan2(desk.position[0] - seat.position[0], desk.position[1] - seat.position[1])
      expect(Math.cos(seat.facing)).toBeCloseTo(Math.cos(towardDesk), 6)
      expect(Math.sin(seat.facing)).toBeCloseTo(Math.sin(towardDesk), 6)
    }
  })

  it('places no tree within the clearing radius of any place or clear point', () => {
    const { teams, agents } = makeTeams(3, 3)
    const clearPoints = [[0, 12] as const]
    const layout = generateLayout(teams, agents, tuning.layout, options({ clearPoints: [...clearPoints] }))
    expect(layout.decor.length).toBeGreaterThan(0)
    for (const tree of layout.decor) {
      for (const place of layout.places) {
        const reach = Math.hypot(place.size[0], place.size[1]) / 2 + tuning.layout.clearingRadius
        expect(Math.hypot(tree.position[0] - place.position[0], tree.position[1] - place.position[1])).toBeGreaterThanOrEqual(reach)
      }
      for (const point of clearPoints) {
        expect(Math.hypot(tree.position[0] - point[0], tree.position[1] - point[1])).toBeGreaterThanOrEqual(tuning.layout.clearingRadius)
      }
      expect(Math.hypot(tree.position[0], tree.position[1])).toBeLessThanOrEqual(tuning.terrain.radius)
      expect(biomeSets.park.biomes.some((biome) => {
        const entry = [...Object.entries(biome.vegetation), ...Object.entries(biome.decor)][tree.variant]
        return entry !== undefined && entry[0] === tree.propId && entry[1].class === tree.kind && entry[1].scaleRef === tree.scaleRef
      })).toBe(true)
    }
  })

  it('biome decor scatter is deterministic for a seed', () => {
    const { teams, agents } = makeTeams(1, 1)
    const layout = generateLayout(teams, agents, tuning.layout, options())
    const again = generateLayout(teams, agents, tuning.layout, options())
    expect(again.decor).toEqual(layout.decor)
  })

  it('indoor scenes get no trees', () => {
    const { teams, agents } = makeTeams(1, 1)
    expect(generateLayout(teams, agents, tuning.layout, options({ scatterDecor: false })).decor).toEqual([])
  })

  it('overrides move a room with its desks and seats, and remove a room with its children', () => {
    const { teams, agents } = makeTeams(2, 2)
    const moved = generateLayout(teams, agents, tuning.layout, options({ overrides: [{ placeId: roomId('team-0'), position: [40, -30] }] }))
    const room = moved.places.find((p) => p.id === roomId('team-0'))
    const desk = moved.places.find((p) => p.id === deskId('agent-0-0'))
    expect(room?.position).toEqual([40, -30])
    expect(desk?.position[0]).toBeGreaterThan(30)
    expect(desk?.seats[0]?.position[1]).toBeGreaterThan(-40)
    const removed = generateLayout(teams, agents, tuning.layout, options({ overrides: [{ placeId: roomId('team-1'), removed: true }] }))
    expect(removed.places.find((p) => p.id === roomId('team-1'))).toBeUndefined()
    expect(removed.places.find((p) => p.id === deskId('agent-1-0'))).toBeUndefined()
    expect(removed.places.find((p) => p.id === tableId('team-1'))).toBeUndefined()
    expect(removed.deskSeatByAgent['agent-1-0']).toBeUndefined()
    expect(removed.deskSeatByAgent['agent-0-0']).toBeDefined()
  })
})
