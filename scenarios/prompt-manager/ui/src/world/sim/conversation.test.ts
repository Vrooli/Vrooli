import { describe, expect, it } from 'vitest'
import { conversationReady } from './conversation'
import { makeWorld, makeWorldStore } from './__tests__/fixtures'

describe('conversation arrival', () => {
  it('holds an Explore conversation in place and restores approach behavior when switching to walking', () => {
    const store = makeWorldStore({ agents: 1, teams: 1 })
    const id = store.getState().actorOrder[0] ?? '', actor = store.getState().actors[id]
    if (!actor) throw Error('Missing fixture')
    const original = [...actor.position]
    store.setVisitorConversation({ agentId: id, position: actor.position, yaw: 0, stationary: true })
    for (let i = 0; i < 40; i++) store.advance(.1)
    expect(store.getState().actors[id]?.position).toEqual(original)
    expect(store.getState().actors[id]?.speed).toBe(0)
    store.setVisitorConversation({ agentId: id, position: [0, 5], yaw: 0 })
    expect(store.getState().visitorConversation?.stationary).toBeUndefined()
    store.advance(.1)
    expect(store.getState().visitorConversation?.goal).toBeDefined()
    store.setVisitorConversation(undefined)
    expect(store.getState().visitorConversation).toBeUndefined()
  })
  it('opens immediately in Explore, but waits for a reachable, stopped nearby agent when walking', () => {
    const state = makeWorld({ agents: 1, teams: 1 })
    const id = state.actorOrder[0] ?? '', actor = state.actors[id]
    if (!actor) throw Error('Missing fixture')
    actor.position = [0, 2]; actor.speed = 0
    expect(conversationReady(state, id, false)).toBe(true)
    expect(conversationReady(state, id, true)).toBe(false)
    state.visitorConversation = { agentId: id, position: [0, 0], yaw: 0, goal: [0, 2], path: [[0, 2]] }
    expect(conversationReady(state, id, true)).toBe(false)
    state.visitorConversation.path = []
    expect(conversationReady(state, id, true)).toBe(true)
    actor.position = [0, 10]
    expect(conversationReady(state, id, true)).toBe(false)
    expect(conversationReady(state, 'missing', false)).toBe(false)
  })
})
