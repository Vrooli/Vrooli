/**
 * Tests for the decoration store.
 *
 * Covers per-scene state, CRUD operations, and — critically — reference
 * stability of selector hooks to prevent infinite re-render loops (React
 * error #185).
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { useDecorationStore } from './decorationStore'
import { useEnvironmentStore } from './environmentStore'

/** Reset both stores to a known-clean state before each test. */
function resetStores() {
  useDecorationStore.setState({ scenes: {} })
  // Ensure the environment store points at 'abstract-space' (the default).
  const envState = useEnvironmentStore.getState()
  if (envState.current.type !== 'abstract-space') {
    useEnvironmentStore.getState().setSceneType('abstract-space')
  }
}

describe('decorationStore', () => {
  beforeEach(resetStores)

  // ---------------------------------------------------------------------------
  // Reference stability — regression tests for React error #185
  // ---------------------------------------------------------------------------
  describe('reference stability', () => {
    it('unvisited scene key should be undefined in the raw state', () => {
      // Selector hooks use ?? EMPTY_DECORATIONS for stability;
      // the raw scenes map should simply not have the key.
      expect(useDecorationStore.getState().scenes['outdoor-park']).toBeUndefined()
    })

    it('should not create a new scenes object when reading an unrelated scene', () => {
      const before = useDecorationStore.getState().scenes
      // Reading a scene key doesn't mutate state
      void useDecorationStore.getState().scenes['outdoor-park']
      const after = useDecorationStore.getState().scenes
      expect(before).toBe(after)
    })

    it('seeding a scene should not affect unrelated scene entries', () => {
      useDecorationStore.getState().seedScene('outdoor-park', [
        { type: 'oak-tree', position: [0, 0, 0], rotation: 0, scale: 1 },
      ])
      // abstract-space should still be undefined (untouched)
      expect(useDecorationStore.getState().scenes['abstract-space']).toBeUndefined()
    })
  })

  // ---------------------------------------------------------------------------
  // Per-scene isolation
  // ---------------------------------------------------------------------------
  describe('per-scene state', () => {
    it('should add decorations to the active scene only', () => {
      const store = useDecorationStore.getState()

      // Add to abstract-space (the active scene)
      store.addDecoration('oak-tree', [1, 0, 1])

      const abstractDecos = useDecorationStore.getState().scenes['abstract-space']
      const parkDecos = useDecorationStore.getState().scenes['outdoor-park']

      expect(abstractDecos).toHaveLength(1)
      expect(parkDecos).toBeUndefined()
    })

    it('should keep scenes independent after switching', () => {
      // Seed abstract-space with one item
      useDecorationStore.getState().addDecoration('oak-tree', [0, 0, 0])

      // Switch to outdoor-park and add a different item
      useEnvironmentStore.getState().setSceneType('outdoor-park')
      useDecorationStore.getState().addDecoration('pine-tree', [5, 0, 5])

      // Switch back — abstract-space should still have 1 item
      useEnvironmentStore.getState().setSceneType('abstract-space')
      expect(useDecorationStore.getState().scenes['abstract-space']).toHaveLength(1)
      expect(useDecorationStore.getState().scenes['outdoor-park']).toHaveLength(1)
    })
  })

  // ---------------------------------------------------------------------------
  // CRUD
  // ---------------------------------------------------------------------------
  describe('addDecoration', () => {
    it('should return an id string', () => {
      const id = useDecorationStore.getState().addDecoration('oak-tree', [0, 0, 0])
      expect(typeof id).toBe('string')
      expect(id.length).toBeGreaterThan(0)
    })
  })

  describe('removeDecoration', () => {
    it('should remove the specified decoration', () => {
      const id = useDecorationStore.getState().addDecoration('oak-tree', [0, 0, 0])
      useDecorationStore.getState().removeDecoration(id)
      expect(useDecorationStore.getState().scenes['abstract-space']).toHaveLength(0)
    })
  })

  describe('moveDecoration', () => {
    it('should update the position', () => {
      const id = useDecorationStore.getState().addDecoration('oak-tree', [0, 0, 0])
      useDecorationStore.getState().moveDecoration(id, [5, 0, 5])

      const d = useDecorationStore.getState().getDecoration(id)
      expect(d?.position[0]).toBe(5)
      expect(d?.position[2]).toBe(5)
    })
  })

  describe('reset', () => {
    it('should set the active scene to an empty array (not undefined)', () => {
      useDecorationStore.getState().addDecoration('oak-tree', [0, 0, 0])
      useDecorationStore.getState().reset()

      // `[]` (explicitly cleared) — not `undefined` (never visited)
      expect(useDecorationStore.getState().scenes['abstract-space']).toEqual([])
    })
  })

  describe('seedScene', () => {
    it('should populate a scene with the given items', () => {
      useDecorationStore.getState().seedScene('outdoor-park', [
        { type: 'oak-tree', position: [1, 0, 1], rotation: 0, scale: 1 },
        { type: 'pine-tree', position: [2, 0, 2], rotation: 0, scale: 1 },
      ])

      expect(useDecorationStore.getState().scenes['outdoor-park']).toHaveLength(2)
    })
  })
})

