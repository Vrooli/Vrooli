import { describe, expect, it } from 'vitest'
import { roomId } from '../layout/generate'
import { canRedo, canUndo, commit, emptyHistory, redo, removeOverride, snapPosition, undo, upsertOverride } from '../layout/overrides'
import { makeWorld } from './fixtures'

describe('override history', () => {
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
