import { describe, expect, it } from 'vitest'
import { roomId } from '../layout/generate'
import { canRedo, canUndo, commit, emptyHistory, redo, removeOverride, snapPosition, undo, upsertOverride } from '../layout/overrides'
import { makeTeams, makeWorld } from './fixtures'
import { checkWorldInvariants } from '../invariants'
import { heightAt } from '../terrain/field'
import { insideSpace, spacePoint } from '../layout/spaces'
import { isWalkable } from '../nav/grid'
import { biomeGrid } from '../terrain/biomes'
import { biomeSets, resolveTerrain, scenes, tuning } from '../../config'

describe('override history', () => {
  it.each([0, Math.PI / 4, -Math.PI / 2])('repairs an overlapping office save without losing its orientation (%f)', rotation => {
    const input = { scene: 'office' as const, teams: 2, agents: 16,
      overrides: [{ placeId: roomId('team-0'), position: [0, 0] as const, rotation }] }
    const state = makeWorld(input), room = state.places[roomId('team-0')]
    if (!room) throw new Error('Missing saved room')
    expect(room.rotation).toBe(rotation)
    expect(Math.hypot(...room.position)).toBeLessThan(25)
    expect(checkWorldInvariants(state, tuning)).toEqual([])
    expect(makeWorld(input).places).toEqual(state.places)
  })
  it('keeps moved office doors on continuous floors and includes the final rooms in the building bounds', () => {
    const original = makeWorld({ scene: 'office', teams: 2, agents: 8 })
    const room = original.places[roomId('team-0')]
    if (!room) throw new Error('Missing office')
    const state = makeWorld({ scene: 'office', teams: 2, agents: 8,
      overrides: [{ placeId: room.id, position: [room.position[0] + 2, room.position[1] - 3], rotation: room.rotation }] })
    const moved = state.places[room.id]
    if (!moved?.space) throw new Error('Missing moved office')
    const halls = Object.values(state.places).filter(p => p.kind === 'corridor')
    for (const distance of [.1, .5, 1, 2, 3]) {
      for (const across of [-.7, 0, .7]) {
        const point = spacePoint(moved, [across, moved.size[1] / 2 + distance])
        expect(halls.some(hall => insideSpace(hall, point)), 'doorway connects to the hall on a continuous full-width floor').toBe(true)
      }
    }
    expect(checkWorldInvariants(state, tuning)).toEqual([])
  })
  it('keeps older position-only campground saves oriented consistently through growth', () => {
    const overrides = [{ placeId: roomId('team-0'), position: [35, 25] as const }]
    const small = makeWorld({ teams: 1, agents: 4, overrides }), grown = makeWorld({ teams: 1, agents: 12, overrides })
    expect(grown.places[roomId('team-0')]?.rotation).toBe(small.places[roomId('team-0')]?.rotation)
  })
  it('keeps a growing saved campsite together on nearby clear ground', () => {
    const roster = makeTeams(2, 4)
    const overrides = [{ placeId: roomId('team-0'), position: [-5, 19.5] as const, rotation: -2.8797932658 }]
    const small = makeWorld({ ...roster, seed: 7, overrides })
    for (let member = 4; member < 12; member++) {
      const id = `agent-0-${member}`
      roster.teams[0]?.memberIds.push(id)
      roster.agents.push({ id, name: id })
    }
    const grown = makeWorld({ ...roster, seed: 7, overrides })
    const room = grown.places[roomId('team-0')], previous = small.places[roomId('team-0')]
    if (!room?.space || !previous) throw new Error('Missing saved campsite')
    expect(room.space.occupantIds).toHaveLength(12)
    expect(Math.hypot(room.position[0] - previous.position[0], room.position[1] - previous.position[1])).toBeLessThan(10)
    expect(room.rotation).toBe(previous.rotation)
    for (const id of small.actorOrder) expect(grown.actors[id]?.deskSeatId).toBe(small.actors[id]?.deskSeatId)
    expect(checkWorldInvariants(grown, tuning).filter(v => v.rule === 'sites-disjoint' || v.rule === 'desk-in-room')).toEqual([])
    const reloaded = makeWorld({ ...roster, seed: 7, overrides })
    expect(reloaded.places).toEqual(grown.places)
  })
  it.each([4, 12])('restores a moved, rotated campground with level support for %i members', agents => {
    const state = makeWorld({ scene: 'park', seed: 7, teams: 1, agents,
      overrides: [{ placeId: roomId('team-0'), position: [35, 25], rotation: Math.PI / 4 }] })
    const room = state.places[roomId('team-0')]
    if (!room?.space) throw new Error('Missing restored campsite')
    expect(room.position).toEqual([35, 25])
    expect(room.space.occupantIds).toHaveLength(agents)
    const ground = heightAt(state.terrain, ...room.position)
    for (const x of [-.48, 0, .48]) for (const z of [-.48, 0, .48]) {
      const point = spacePoint(room, [room.size[0] * x, room.size[1] * z])
      expect(heightAt(state.terrain, ...point), 'saved campsite keeps its level ground').toBeCloseTo(ground, 3)
    }
    expect(isWalkable(state.nav, spacePoint(room, room.space.meeting))).toBe(true)
    const expectedBiomes = biomeGrid(state.terrain, resolveTerrain(scenes.park, tuning), biomeSets.park)
    expect(state.biomes.reduce((count, biome, i) => count + Number(biome !== expectedBiomes[i]), 0), 'vegetation and ground colors follow the final terrain').toBe(0)
  })
  it('commit, undo and redo round trip and the stack is bounded', () => {
    let h = emptyHistory()
    h = commit(h, [{ placeId: 'a', position: [1, 1] }], 2)
    h = commit(h, [{ placeId: 'a', position: [2, 2] }], 2)
    h = commit(h, [{ placeId: 'a', position: [3, 3] }], 2)
    expect(h.past).toHaveLength(2)
    expect(canUndo(h)).toBe(true)
    expect(canRedo(h)).toBe(false)
    h = undo(h)
    expect(h.current[0]?.position).toEqual([2, 2])
    expect(canRedo(h)).toBe(true)
    h = redo(h)
    expect(h.current[0]?.position).toEqual([3, 3])
    h = undo(undo(h))
    expect(h.current[0]?.position).toEqual([1, 1])
    expect(undo(h)).toBe(h)
    h = commit(h, [], 2)
    expect(canRedo(h)).toBe(false)
  })

  it('upsert merges by place id and remove drops it', () => {
    let set = upsertOverride([], { placeId: 'r', position: [1, 2] })
    set = upsertOverride(set, { placeId: 'r', rotation: 0.5 })
    expect(set).toEqual([{ placeId: 'r', position: [1, 2], rotation: 0.5 }])
    set = upsertOverride(set, { placeId: 'q', removed: true })
    expect(removeOverride(set, 'r')).toEqual([{ placeId: 'q', removed: true }])
  })

  it('snaps to the grid', () => {
    expect(snapPosition([1.3, -0.7], 0.5)).toEqual([1.5, -0.5])
    expect(snapPosition([1.3, -0.7], 0)).toEqual([1.3, -0.7])
  })

  it('a removed room sends its members home to the commons on the next tick', () => {
    const state = makeWorld({ teams: 2, agents: 4, overrides: [{ placeId: roomId('team-0'), removed: true }] })
    expect(state.places[roomId('team-0')]).toBeUndefined()
    const member = state.actors['agent-0-0']
    expect(member?.deskSeatId).toBeUndefined()
  })
})
