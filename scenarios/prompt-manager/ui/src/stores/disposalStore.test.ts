/**
 * Tests for Asset Disposal store.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useDisposalStore } from '@/stores/disposalStore'
import * as THREE from 'three'

describe('Disposal Store', () => {
  beforeEach(() => {
    // Reset store state before each test
    useDisposalStore.getState().disposeAll()
    useDisposalStore.getState().stopPeriodicCleanup()
    // Reset stats completely
    useDisposalStore.setState({
      assets: new Map(),
      stats: {
        totalTracked: 0,
        byType: { material: 0, geometry: 0, texture: 0, object3d: 0 },
        lastCleanupCount: 0,
        lastCleanupTime: null,
        totalDisposed: 0,
      },
    })
  })

  describe('trackAsset', () => {
    it('tracks a new material', () => {
      const material = new THREE.MeshStandardMaterial({ color: 'red' })
      const id = 'test-material-1'

      useDisposalStore.getState().trackAsset(id, 'material', material, 'test-owner')

      expect(useDisposalStore.getState().isTracked(id)).toBe(true)
      expect(useDisposalStore.getState().stats.totalTracked).toBe(1)
      expect(useDisposalStore.getState().stats.byType.material).toBe(1)
    })

    it('tracks a new geometry', () => {
      const geometry = new THREE.BoxGeometry(1, 1, 1)
      const id = 'test-geometry-1'

      useDisposalStore.getState().trackAsset(id, 'geometry', geometry, 'test-owner')

      expect(useDisposalStore.getState().isTracked(id)).toBe(true)
      expect(useDisposalStore.getState().stats.byType.geometry).toBe(1)
    })

    it('tracks a new texture', () => {
      const texture = new THREE.Texture()
      const id = 'test-texture-1'

      useDisposalStore.getState().trackAsset(id, 'texture', texture, 'test-owner')

      expect(useDisposalStore.getState().isTracked(id)).toBe(true)
      expect(useDisposalStore.getState().stats.byType.texture).toBe(1)
    })

    it('does not track duplicate IDs', () => {
      const material1 = new THREE.MeshStandardMaterial({ color: 'red' })
      const material2 = new THREE.MeshStandardMaterial({ color: 'blue' })
      const id = 'duplicate-id'

      useDisposalStore.getState().trackAsset(id, 'material', material1, 'owner-1')
      useDisposalStore.getState().trackAsset(id, 'material', material2, 'owner-2')

      expect(useDisposalStore.getState().stats.totalTracked).toBe(1)
    })
  })

  describe('disposeAsset', () => {
    it('disposes a tracked material', () => {
      const material = new THREE.MeshStandardMaterial({ color: 'red' })
      const disposeSpy = vi.spyOn(material, 'dispose')
      const id = 'test-material-1'

      useDisposalStore.getState().trackAsset(id, 'material', material, 'test-owner')
      const result = useDisposalStore.getState().disposeAsset(id)

      expect(result).toBe(true)
      expect(disposeSpy).toHaveBeenCalled()
      expect(useDisposalStore.getState().isTracked(id)).toBe(false)
      expect(useDisposalStore.getState().stats.totalDisposed).toBe(1)
    })

    it('returns false for unknown asset', () => {
      const result = useDisposalStore.getState().disposeAsset('unknown-id')
      expect(result).toBe(false)
    })
  })

  describe('disposeOwnerAssets', () => {
    it('disposes all assets for an owner', () => {
      const material1 = new THREE.MeshStandardMaterial({ color: 'red' })
      const material2 = new THREE.MeshStandardMaterial({ color: 'blue' })
      const material3 = new THREE.MeshStandardMaterial({ color: 'green' })

      useDisposalStore.getState().trackAsset('mat-1', 'material', material1, 'owner-a')
      useDisposalStore.getState().trackAsset('mat-2', 'material', material2, 'owner-a')
      useDisposalStore.getState().trackAsset('mat-3', 'material', material3, 'owner-b')

      const count = useDisposalStore.getState().disposeOwnerAssets('owner-a')

      expect(count).toBe(2)
      expect(useDisposalStore.getState().isTracked('mat-1')).toBe(false)
      expect(useDisposalStore.getState().isTracked('mat-2')).toBe(false)
      expect(useDisposalStore.getState().isTracked('mat-3')).toBe(true)
    })
  })

  describe('disposeAll', () => {
    it('disposes all tracked assets', () => {
      const material = new THREE.MeshStandardMaterial({ color: 'red' })
      const geometry = new THREE.BoxGeometry(1, 1, 1)

      useDisposalStore.getState().trackAsset('mat-1', 'material', material, 'owner')
      useDisposalStore.getState().trackAsset('geo-1', 'geometry', geometry, 'owner')

      const count = useDisposalStore.getState().disposeAll()

      expect(count).toBe(2)
      expect(useDisposalStore.getState().stats.totalTracked).toBe(0)
    })
  })

  describe('statistics', () => {
    it('tracks disposal counts correctly', () => {
      const material = new THREE.MeshStandardMaterial({ color: 'red' })
      const geometry = new THREE.BoxGeometry(1, 1, 1)

      useDisposalStore.getState().trackAsset('mat-1', 'material', material, 'owner')
      useDisposalStore.getState().trackAsset('geo-1', 'geometry', geometry, 'owner')

      let stats = useDisposalStore.getState().getStats()
      expect(stats.totalTracked).toBe(2)
      expect(stats.byType.material).toBe(1)
      expect(stats.byType.geometry).toBe(1)
      expect(stats.totalDisposed).toBe(0)

      useDisposalStore.getState().disposeAsset('mat-1')

      stats = useDisposalStore.getState().getStats()
      expect(stats.totalTracked).toBe(1)
      expect(stats.byType.material).toBe(0)
      expect(stats.totalDisposed).toBe(1)
    })
  })

  describe('config', () => {
    it('allows updating config', () => {
      useDisposalStore.getState().setConfig({ debug: true })
      expect(useDisposalStore.getState().config.debug).toBe(true)
    })
  })
})
