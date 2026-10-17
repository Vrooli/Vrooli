import { describe, expect, it } from 'vitest'
import { makeWorld, makeWorldStore } from '../__tests__/fixtures'
import { isWalkable } from '../nav/grid'
import { findPath } from '../nav/astar'
import { insideSpace, spaceActors, spacePoint, spaceStructures } from './spaces'

describe('inhabited spaces', () => {
  it('lists visiting agents in shared rooms and retains assigned members who are away', () => {
    const state = makeWorld({ scene: 'office', teams: 2, agents: 4 })
    const lounge = state.places['shared:lounge'], team = state.places['room:team-0']
    const actor = state.actors[state.actorOrder[0] ?? '']
    if (!lounge || !team || !actor) throw new Error('Missing fixture')
    expect(spaceActors(state, lounge)).toEqual([])
    actor.position = spacePoint(lounge, [1, -1])
    expect(insideSpace(lounge, actor.position)).toBe(true)
    expect(spaceActors(state, lounge).map(a => a.id)).toEqual([actor.id])
    expect(spaceActors(state, team).map(a => a.id)).toContain(actor.id)
    expect(insideSpace(team, actor.position)).toBe(false)
    expect(insideSpace(lounge, spacePoint(lounge, [0, lounge.size[1] / 2 + .1]))).toBe(false)
  })
  it('builds office window openings, overhead ceilings, open door leaves and navigable shared facilities', () => {
    const state = makeWorld({ scene: 'office', teams: 1, agents: 4 })
    for (const id of ['room:team-0', 'shared:lounge', 'shared:kitchen']) {
      const room = state.places[id]
      if (!room?.space) throw new Error(`Missing ${id}`)
      const boxes = spaceStructures(room)
      expect(boxes.filter(b => b.surface === 'window')).toHaveLength(3)
      expect(boxes.find(b => b.surface === 'ceiling')?.position[1]).toBeGreaterThan(2.7)
      expect(boxes.find(b => b.id.endsWith('open-door'))).toBeDefined()
      expect(isWalkable(state.nav, spacePoint(room, room.space.meeting))).toBe(true)
    }
  })
  it.each(['park', 'office'] as const)('an invited %s occupant leaves its enclosure and reaches the visitor', scene => {
    const store = makeWorldStore({ scene, teams: 1, agents: 4 })
    const room = store.getState().places['room:team-0']
    if (!room?.space) throw new Error('Missing space')
    const id = room.space.occupantIds[0]
    if (!id) throw new Error('Missing occupant')
    const visitor = spacePoint(room, [room.space.entrance[0], room.space.entrance[1] + 4])
    const original = store.getState().actors[id]?.position
    store.setVisitorConversation({ agentId: id, position: visitor, yaw: room.rotation + Math.PI })
    for (let i = 0; i < 1200; i++) store.advance(.05)
    const state = store.getState(), actor = state.actors[id]
    if (!actor) throw new Error('Missing invited actor')
    expect(actor.position).not.toEqual(original)
    expect(state.visitorConversation?.path).toHaveLength(0)
    expect(Math.hypot(actor.position[0] - visitor[0], actor.position[1] - visitor[1])).toBeLessThan(3)
  })
  it.each(['park', 'office'] as const)('gives every %s team a named space, entrance, occupants and reachable meeting place', scene => {
    const state = makeWorld({ scene, teams: 2, agents: 8 })
    const rooms = Object.values(state.places).filter(p => p.kind === 'room' && p.teamId)
    expect(rooms).toHaveLength(2)
    for (const room of rooms) {
      expect(room.space?.kind).toBe(scene === 'park' ? 'campsite' : 'office')
      const space = room.space
      if (!space) throw new Error('Missing space')
      expect(space.occupantIds).toHaveLength(4)
      const meeting = spacePoint(room, space.meeting)
      expect(isWalkable(state.nav, meeting)).toBe(true)
      for (const id of space.occupantIds) {
        const actor = state.actors[id]
        if (!actor) throw new Error('Missing occupant')
        const route = findPath(state.nav, actor.position, meeting)
        expect(route?.length ?? 0, `${scene} ${id} can exit to the meeting point`).toBeGreaterThan(0)
      }
      for (const wall of spaceStructures(room).filter(b => b.surface === 'wall')) {
        expect(isWalkable(state.nav, [wall.position[0], wall.position[2]]), `${wall.id} remains solid`).toBe(false)
      }
    }
  })

  it('allocates enough shelters as a campground team grows, retaining named seat identities', () => {
    const small = makeWorld({ teams: 1, agents: 4 })
    const large = makeWorld({ teams: 1, agents: 20 })
    const room = large.places['room:team-0']
    if (!room?.space) throw new Error('Missing campsite')
    expect(room.space.shelters).toHaveLength(5)
    for (const id of small.actorOrder) expect(large.actors[id]?.deskSeatId).toBe(small.actors[id]?.deskSeatId)
    const positions = room.space.occupantIds.map(id => large.actors[id]?.position.join(','))
    expect(new Set(positions).size).toBe(20)
    expect(Object.values(large.places).find(p => p.parentId === room.id && p.kind === 'hearth')).toBeDefined()
  })
})
