/**
 * Tests for the furniture store.
 *
 * Covers per-scene state, seating, and — critically — reference stability of
 * selector hooks to prevent infinite re-render loops (React error #185).
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { useFurnitureStore } from './furnitureStore'
import { useEnvironmentStore } from './environmentStore'

/** Reset both stores to a known-clean state before each test. */
function resetStores() {
  useFurnitureStore.setState({ scenes: {}, seatedAgentsByScene: {} })
  const envState = useEnvironmentStore.getState()
  if (envState.current.type !== 'abstract-space') {
    useEnvironmentStore.getState().setSceneType('abstract-space')
  }
}

describe('furnitureStore', () => {
  beforeEach(resetStores)

  // ---------------------------------------------------------------------------
  // Reference stability — regression tests for React error #185
  // ---------------------------------------------------------------------------
  describe('reference stability', () => {
    it('should return the SAME empty array reference for an unvisited scene (furniture)', () => {
      const state = useFurnitureStore.getState()
      // Both reads of an undefined scene key should yield the same reference
      // when using the module-level EMPTY_FURNITURE constant.
      const scenesObj = state.scenes
      expect(scenesObj['outdoor-park']).toBeUndefined()
    })

    it('should return the SAME empty object reference for an unvisited scene (seating)', () => {
      const state = useFurnitureStore.getState()
      expect(state.seatedAgentsByScene['outdoor-park']).toBeUndefined()
    })

    it('seeding furniture should NOT create a new seatedAgentsByScene entry', () => {
      // This is the critical regression: seedScene should only touch `scenes`,
      // not `seatedAgentsByScene`. If it did, useSeatedAgents() would get a
      // new object reference on every seed, potentially causing re-render loops.
      const seatingBefore = useFurnitureStore.getState().seatedAgentsByScene

      useFurnitureStore.getState().seedScene('abstract-space', [
        { type: 'desk', position: [0, 0, 0], rotation: 0, color: '#fff', occupiedBy: null },
      ])

      const seatingAfter = useFurnitureStore.getState().seatedAgentsByScene
      // The seating map itself changes (new state object) but the CONTENT for
      // the seeded scene should still be undefined (no agents seated).
      expect(seatingAfter['abstract-space']).toBeUndefined()
      // And the overall seatingByScene should be the same reference since
      // seedScene doesn't spread seatedAgentsByScene.
      expect(seatingBefore).toBe(seatingAfter)
    })
  })

  // ---------------------------------------------------------------------------
  // Per-scene isolation
  // ---------------------------------------------------------------------------
  describe('per-scene state', () => {
    it('should add furniture to the active scene only', () => {
      useFurnitureStore.getState().addFurniture('desk', [1, 0, 1])

      expect(useFurnitureStore.getState().scenes['abstract-space']).toHaveLength(1)
      expect(useFurnitureStore.getState().scenes['outdoor-park']).toBeUndefined()
    })

    it('should isolate seating between scenes', () => {
      // Add desk + seat agent in abstract-space
      const deskId = useFurnitureStore.getState().addFurniture('desk', [0, 0, 0])
      useFurnitureStore.getState().seatAgent('agent-1', deskId)

      const seatedAbstract = useFurnitureStore.getState().seatedAgentsByScene['abstract-space']
      expect(seatedAbstract).toBeDefined()
      expect(seatedAbstract?.['agent-1']).toBeDefined()

      // Outdoor-park should have no seating data
      expect(useFurnitureStore.getState().seatedAgentsByScene['outdoor-park']).toBeUndefined()
    })
  })

  // ---------------------------------------------------------------------------
  // CRUD
  // ---------------------------------------------------------------------------
  describe('addFurniture', () => {
    it('should return an id string', () => {
      const id = useFurnitureStore.getState().addFurniture('desk', [0, 0, 0])
      expect(typeof id).toBe('string')
      expect(id.length).toBeGreaterThan(0)
    })
  })

  describe('removeFurniture', () => {
    it('should remove the specified furniture', () => {
      const id = useFurnitureStore.getState().addFurniture('desk', [0, 0, 0])
      useFurnitureStore.getState().removeFurniture(id)
      expect(useFurnitureStore.getState().scenes['abstract-space']).toHaveLength(0)
    })

    it('should unseat agents from removed furniture', () => {
      const id = useFurnitureStore.getState().addFurniture('desk', [0, 0, 0])
      useFurnitureStore.getState().seatAgent('agent-1', id)
      useFurnitureStore.getState().removeFurniture(id)

      const seated = useFurnitureStore.getState().seatedAgentsByScene['abstract-space']
      expect(seated?.['agent-1']).toBeUndefined()
    })
  })

  describe('moveFurniture', () => {
    it('should update the position', () => {
      const id = useFurnitureStore.getState().addFurniture('desk', [0, 0, 0])
      useFurnitureStore.getState().moveFurniture(id, [5, 0, 5])

      const f = useFurnitureStore.getState().getFurniture(id)
      expect(f?.position).toEqual([5, 0, 5])
    })
  })

  describe('reset', () => {
    it('should set active scene to empty array and clear seating', () => {
      useFurnitureStore.getState().addFurniture('desk', [0, 0, 0])
      useFurnitureStore.getState().reset()

      expect(useFurnitureStore.getState().scenes['abstract-space']).toEqual([])
      expect(useFurnitureStore.getState().seatedAgentsByScene['abstract-space']).toEqual({})
    })
  })

  describe('seedScene', () => {
    it('should populate a scene with the given items', () => {
      useFurnitureStore.getState().seedScene('outdoor-park', [
        { type: 'desk', position: [1, 0, 1], rotation: 0, color: '#fff', occupiedBy: null },
      ])

      expect(useFurnitureStore.getState().scenes['outdoor-park']).toHaveLength(1)
    })

    it('should assign unique IDs to seeded items', () => {
      useFurnitureStore.getState().seedScene('outdoor-park', [
        { type: 'desk', position: [0, 0, 0], rotation: 0, color: '#fff', occupiedBy: null },
        { type: 'desk', position: [2, 0, 0], rotation: 0, color: '#fff', occupiedBy: null },
      ])

      const items = useFurnitureStore.getState().scenes['outdoor-park'] ?? []
      expect(items).toHaveLength(2)
      expect(items[0]?.id).not.toBe(items[1]?.id)
    })
  })

  // ---------------------------------------------------------------------------
  // Seating
  // ---------------------------------------------------------------------------
  describe('seatAgent / unseatAgent', () => {
    it('should seat and unseat an agent', () => {
      const id = useFurnitureStore.getState().addFurniture('desk', [0, 0, 0])

      const result = useFurnitureStore.getState().seatAgent('agent-1', id)
      expect(result).toBe(true)

      const seated = useFurnitureStore.getState().seatedAgentsByScene['abstract-space']
      expect(seated?.['agent-1']?.furnitureId).toBe(id)

      useFurnitureStore.getState().unseatAgent('agent-1')
      const after = useFurnitureStore.getState().seatedAgentsByScene['abstract-space']
      expect(after?.['agent-1']).toBeUndefined()
    })

    it('should return false when furniture does not exist', () => {
      const result = useFurnitureStore.getState().seatAgent('agent-1', 'nonexistent')
      expect(result).toBe(false)
    })
  })
})
