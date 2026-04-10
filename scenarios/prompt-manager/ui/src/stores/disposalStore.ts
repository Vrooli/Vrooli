/**
 * Asset disposal store for tracking and cleaning up Three.js resources.
 * Prevents memory leaks by properly disposing materials, geometries, and textures.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#memory-management

import { create } from 'zustand'
import type { Material, Texture, BufferGeometry } from 'three'
import type {
  TrackedAsset,
  DisposableType,
  DisposalStats,
  DisposalConfig,
} from '@/types/disposal'
import { DEFAULT_DISPOSAL_CONFIG } from '@/types/disposal'

// WeakRef polyfill type for older TypeScript targets
declare const WeakRef: {
  prototype: WeakRef<object>
  new <T extends object>(target: T): WeakRef<T>
}
interface WeakRef<T extends object> {
  deref(): T | undefined
}

interface DisposalState {
  /** Tracked assets by ID */
  assets: Map<string, TrackedAsset>
  /** Configuration */
  config: DisposalConfig
  /** Statistics */
  stats: DisposalStats
  /** Cleanup interval ID */
  cleanupIntervalId: number | null
}

interface DisposalActions {
  /** Track a new asset */
  trackAsset: (
    id: string,
    type: DisposableType,
    ref: Material | Texture | BufferGeometry,
    owner: string
  ) => void
  /** Dispose a specific asset */
  disposeAsset: (id: string) => boolean
  /** Dispose all assets for an owner */
  disposeOwnerAssets: (owner: string) => number
  /** Dispose all tracked assets */
  disposeAll: () => number
  /** Run cleanup (dispose orphaned assets) */
  runCleanup: () => number
  /** Update configuration */
  setConfig: (config: Partial<DisposalConfig>) => void
  /** Start periodic cleanup */
  startPeriodicCleanup: () => void
  /** Stop periodic cleanup */
  stopPeriodicCleanup: () => void
  /** Get statistics */
  getStats: () => DisposalStats
  /** Check if asset is tracked */
  isTracked: (id: string) => boolean
}

type DisposalStore = DisposalState & DisposalActions

const initialStats: DisposalStats = {
  totalTracked: 0,
  byType: { material: 0, geometry: 0, texture: 0, object3d: 0 },
  lastCleanupCount: 0,
  lastCleanupTime: null,
  totalDisposed: 0,
}

const initialState: DisposalState = {
  assets: new Map(),
  config: DEFAULT_DISPOSAL_CONFIG,
  stats: initialStats,
  cleanupIntervalId: null,
}

/**
 * Dispose a Three.js object properly
 */
function disposeObject(
  type: DisposableType,
  ref: Material | Texture | BufferGeometry | null
): boolean {
  if (!ref) return false

  try {
    switch (type) {
      case 'material':
        (ref as Material).dispose()
        return true
      case 'texture':
        (ref as Texture).dispose()
        return true
      case 'geometry':
        (ref as BufferGeometry).dispose()
        return true
      case 'object3d':
        // Object3D doesn't have dispose, but we can clear it
        return true
      default:
        return false
    }
  } catch (error) {
    console.warn(`Failed to dispose ${type}:`, error)
    return false
  }
}

/**
 * Zustand store for disposal management.
 */
export const useDisposalStore = create<DisposalStore>((set, get) => ({
  ...initialState,

  trackAsset: (id, type, ref, owner) => {
    const state = get()

    // Skip if already tracked
    if (state.assets.has(id)) return

    const asset: TrackedAsset = {
      id,
      type,
      ref: new WeakRef(ref),
      owner,
      createdAt: performance.now(),
      disposed: false,
    }

    const newAssets = new Map(state.assets)
    newAssets.set(id, asset)

    const newByType = { ...state.stats.byType }
    newByType[type]++

    set({
      assets: newAssets,
      stats: {
        ...state.stats,
        totalTracked: state.stats.totalTracked + 1,
        byType: newByType,
      },
    })

    if (state.config.debug) {
      console.log(`[Disposal] Tracked ${type}: ${id} (owner: ${owner})`)
    }
  },

  disposeAsset: (id) => {
    const state = get()
    const asset = state.assets.get(id)

    if (!asset || asset.disposed) return false

    // Get actual object from WeakRef
    const obj = asset.ref.deref()

    // Dispose the object
    const disposed = disposeObject(asset.type, obj ?? null)

    if (disposed) {
      // Update asset as disposed
      asset.disposed = true

      const newAssets = new Map(state.assets)
      newAssets.delete(id)

      const newByType = { ...state.stats.byType }
      newByType[asset.type] = Math.max(0, newByType[asset.type] - 1)

      set({
        assets: newAssets,
        stats: {
          ...state.stats,
          totalTracked: state.stats.totalTracked - 1,
          byType: newByType,
          totalDisposed: state.stats.totalDisposed + 1,
        },
      })

      if (state.config.debug) {
        console.log(`[Disposal] Disposed ${asset.type}: ${id}`)
      }
    }

    return disposed
  },

  disposeOwnerAssets: (owner) => {
    const state = get()
    let count = 0

    const toDispose: string[] = []
    for (const [id, asset] of state.assets) {
      if (asset.owner === owner && !asset.disposed) {
        toDispose.push(id)
      }
    }

    for (const id of toDispose) {
      if (state.disposeAsset(id)) {
        count++
      }
    }

    if (state.config.debug && count > 0) {
      console.log(`[Disposal] Disposed ${count} assets for owner: ${owner}`)
    }

    return count
  },

  disposeAll: () => {
    const state = get()
    let count = 0

    const toDispose = Array.from(state.assets.keys())
    for (const id of toDispose) {
      if (state.disposeAsset(id)) {
        count++
      }
    }

    if (state.config.debug) {
      console.log(`[Disposal] Disposed all ${count} assets`)
    }

    return count
  },

  runCleanup: () => {
    const state = get()
    const now = performance.now()
    let count = 0

    // Find assets where WeakRef has been garbage collected
    const toRemove: string[] = []
    for (const [id, asset] of state.assets) {
      if (asset.disposed) {
        toRemove.push(id)
        continue
      }

      // Check if object still exists
      const obj = asset.ref.deref()
      if (!obj) {
        // Object was garbage collected, remove tracking
        toRemove.push(id)
        count++
      }
    }

    if (toRemove.length > 0) {
      const newAssets = new Map(state.assets)
      const newByType = { ...state.stats.byType }

      for (const id of toRemove) {
        const asset = newAssets.get(id)
        if (asset) {
          newByType[asset.type] = Math.max(0, newByType[asset.type] - 1)
        }
        newAssets.delete(id)
      }

      set({
        assets: newAssets,
        stats: {
          ...state.stats,
          totalTracked: newAssets.size,
          byType: newByType,
          lastCleanupCount: count,
          lastCleanupTime: now,
          totalDisposed: state.stats.totalDisposed + count,
        },
      })
    } else {
      set({
        stats: {
          ...state.stats,
          lastCleanupCount: 0,
          lastCleanupTime: now,
        },
      })
    }

    if (state.config.debug && count > 0) {
      console.log(`[Disposal] Cleanup removed ${count} orphaned assets`)
    }

    return count
  },

  setConfig: (config) => {
    const state = get()
    const newConfig = { ...state.config, ...config }

    set({ config: newConfig })

    // Handle cleanup interval changes
    if (config.cleanupInterval !== undefined) {
      state.stopPeriodicCleanup()
      if (newConfig.cleanupInterval > 0) {
        state.startPeriodicCleanup()
      }
    }
  },

  startPeriodicCleanup: () => {
    const state = get()

    // Clear existing interval
    if (state.cleanupIntervalId !== null) {
      clearInterval(state.cleanupIntervalId)
    }

    if (state.config.cleanupInterval <= 0) return

    const intervalId = window.setInterval(() => {
      get().runCleanup()
    }, state.config.cleanupInterval)

    set({ cleanupIntervalId: intervalId })
  },

  stopPeriodicCleanup: () => {
    const state = get()
    if (state.cleanupIntervalId !== null) {
      clearInterval(state.cleanupIntervalId)
      set({ cleanupIntervalId: null })
    }
  },

  getStats: () => get().stats,

  isTracked: (id) => get().assets.has(id),
}))

/**
 * Selector for disposal stats
 */
export const selectDisposalStats = (state: DisposalStore) => state.stats

/**
 * Selector for tracked count
 */
export const selectTrackedCount = (state: DisposalStore) => state.stats.totalTracked
