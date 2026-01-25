/**
 * Tests for LOD (Level of Detail) system.
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { useLODStore } from '@/stores/lodStore'
import { DEFAULT_LOD_CONFIG } from '@/types/lod'

describe('LOD Store', () => {
  beforeEach(() => {
    // Reset store state completely before each test
    useLODStore.getState().clearAll()
    // Reset config to defaults
    useLODStore.setState({ config: { ...DEFAULT_LOD_CONFIG } })
  })

  describe('calculateLODLevel', () => {
    it('returns high for close distances', () => {
      const level = useLODStore.getState().calculateLODLevel(3)
      expect(level).toBe('high')
    })

    it('returns medium for moderate distances', () => {
      const level = useLODStore.getState().calculateLODLevel(8)
      expect(level).toBe('medium')
    })

    it('returns low for far distances', () => {
      const level = useLODStore.getState().calculateLODLevel(20)
      expect(level).toBe('low')
    })

    it('returns culled for very far distances', () => {
      const level = useLODStore.getState().calculateLODLevel(60)
      expect(level).toBe('culled')
    })
  })

  describe('updateObjectLOD', () => {
    it('tracks object LOD state', () => {
      const objectId = 'test-object-1'
      const distance = 3 // Within high threshold (default < 5)

      const result = useLODStore.getState().updateObjectLOD(objectId, distance)

      expect(result.level).toBe('high')
      expect(result.distance).toBe(distance)
      expect(useLODStore.getState().getObjectLOD(objectId)).toBeDefined()
    })

    it('updates existing object LOD', () => {
      const objectId = 'test-object-1'

      useLODStore.getState().updateObjectLOD(objectId, 3)
      expect(useLODStore.getState().getObjectLOD(objectId)?.level).toBe('high')

      useLODStore.getState().updateObjectLOD(objectId, 30)
      expect(useLODStore.getState().getObjectLOD(objectId)?.level).toBe('low')
    })
  })

  describe('batchUpdateLODs', () => {
    it('updates multiple objects efficiently', () => {
      const updates = [
        { id: 'obj-1', distance: 3 },
        { id: 'obj-2', distance: 10 },
        { id: 'obj-3', distance: 30 },
        { id: 'obj-4', distance: 60 },
      ]

      useLODStore.getState().batchUpdateLODs(updates)

      const state = useLODStore.getState()
      expect(state.objectCount).toBe(4)
      expect(state.levelCounts.high).toBe(1)
      expect(state.levelCounts.medium).toBe(1)
      expect(state.levelCounts.low).toBe(1)
      expect(state.levelCounts.culled).toBe(1)
    })
  })

  describe('removeObject', () => {
    it('removes object from tracking', () => {
      const objectId = 'test-object-1'
      useLODStore.getState().updateObjectLOD(objectId, 5)

      expect(useLODStore.getState().getObjectLOD(objectId)).toBeDefined()

      useLODStore.getState().removeObject(objectId)

      expect(useLODStore.getState().getObjectLOD(objectId)).toBeUndefined()
    })
  })

  describe('shouldTrackCursor', () => {
    it('returns true for high LOD objects', () => {
      const objectId = 'test-object-1'
      useLODStore.getState().updateObjectLOD(objectId, 3)

      expect(useLODStore.getState().shouldTrackCursor(objectId)).toBe(true)
    })

    it('returns true for medium LOD objects', () => {
      const objectId = 'test-object-1'
      useLODStore.getState().updateObjectLOD(objectId, 10)

      expect(useLODStore.getState().shouldTrackCursor(objectId)).toBe(true)
    })

    it('returns false for low LOD objects', () => {
      const objectId = 'test-object-1'
      useLODStore.getState().updateObjectLOD(objectId, 30)

      expect(useLODStore.getState().shouldTrackCursor(objectId)).toBe(false)
    })

    it('returns false for culled objects', () => {
      const objectId = 'test-object-1'
      useLODStore.getState().updateObjectLOD(objectId, 60)

      expect(useLODStore.getState().shouldTrackCursor(objectId)).toBe(false)
    })
  })

  describe('config', () => {
    it('allows updating config', () => {
      useLODStore.getState().setConfig({ enableCursorLOD: false })

      expect(useLODStore.getState().config.enableCursorLOD).toBe(false)
    })

    it('allows updating thresholds', () => {
      useLODStore.getState().setThresholds({ high: 10, medium: 20 })

      const config = useLODStore.getState().config.thresholds
      expect(config.high).toBe(10)
      expect(config.medium).toBe(20)
    })

    it('disables cursor LOD when config says so', () => {
      const objectId = 'test-config-cursor'

      // Use distance 15 to get 'low' LOD (threshold for low is 25, medium is 12)
      // 15 is > 12*1.1=13.2 and < 25*1.1=27.5, so should be 'low'
      useLODStore.getState().updateObjectLOD(objectId, 15)

      // Verify we got low LOD
      const lod = useLODStore.getState().getObjectLOD(objectId)
      expect(lod?.level).toBe('low')

      // With LOD enabled, low LOD shouldn't track cursor
      expect(useLODStore.getState().shouldTrackCursor(objectId)).toBe(false)

      // Disable cursor LOD
      useLODStore.getState().setConfig({ enableCursorLOD: false })

      // Now should always track cursor regardless of LOD
      expect(useLODStore.getState().shouldTrackCursor(objectId)).toBe(true)
    })
  })
})
