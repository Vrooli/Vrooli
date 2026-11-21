import { describe, expect, it } from 'vitest'
import { tuning } from '../../config'
import { createWorld } from '../world'
import { makeTeams, makeWorld, makeWorldInput, makeWorldStore, NOW } from './fixtures'

describe('shared world fixture', () => {
  it('defaults omitted properties and makes total agent count explicit', () => {
    const state = makeWorld({ agents: 7 })
    expect(state.actorOrder).toHaveLength(7)
    expect(makeWorldInput({ agents: 7 }).teams.map((team) => team.memberIds.length)).toEqual([4, 3])
    expect(makeWorld({ teams: 0, agents: 0 }).actorOrder).toEqual([])
    expect(makeWorld({ teams: 0, agents: 1 }).actorOrder).toHaveLength(1)
  })

  it('is deterministic for a fixed seed and diverges for different seeds', () => {
    expect(makeWorld({ seed: 42, agents: 2 })).toEqual(makeWorld({ seed: 42, agents: 2 }))
    expect(makeWorld({ seed: 42, agents: 2 })).not.toEqual(makeWorld({ seed: 43, agents: 2 }))
  })

  it('preserves supplied identities and legacy fixtures byte-for-byte', () => {
    const input = { ...makeTeams(2, 3), seed: 7, now: NOW, scene: 'office' as const }
    const before = structuredClone(input)
    expect(makeWorld({ ...input, treeVariants: 3 })).toEqual(createWorld(input, tuning, 3))
    expect(makeWorldStore(input).getState()).toEqual(makeWorld(input))
    expect(input).toEqual(before)
  })

  it('rejects malformed roster counts', () => {
    expect(() => makeWorld({ teams: -1 })).toThrow(/counts/)
    expect(() => makeWorld({ agents: 1.5 })).toThrow(/counts/)
  })
})
