/**
 * LOD (Level of Detail) store for managing object detail levels.
 * Optimizes rendering by reducing detail for distant objects.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#lod-system

import { create } from 'zustand'
import type { LODLevel, LODConfig, LODThresholds } from '@/types/lod'
import { DEFAULT_LOD_CONFIG } from '@/types/lod'

/**
 * Per-object LOD state stored in a Map for O(1) access.
 * Using a Map instead of object for better performance with many objects.
 */
interface ObjectLOD {
  level: LODLevel
  distance: number
  lastUpdated: number
}

interface LODState {
  /** LOD configuration */
  config: LODConfig
  /** Per-object LOD state - keyed by object ID */
  objectLODs: Map<string, ObjectLOD>
  /** Camera position (updated via ref, not state) */
  cameraPositionRef: [number, number, number]
  /** Total object count for statistics */
  objectCount: number
  /** Count of objects at each LOD level */
  levelCounts: Record<LODLevel, number>
}

interface LODActions {
  /** Update LOD config */
  setConfig: (config: Partial<LODConfig>) => void
  /** Set LOD thresholds */
  setThresholds: (thresholds: Partial<LODThresholds>) => void
  /** Update camera position (called from useFrame via getState) */
  updateCameraPosition: (position: [number, number, number]) => void
  /** Calculate LOD level for a given distance */
  calculateLODLevel: (distance: number) => LODLevel
  /** Update a single object's LOD */
  updateObjectLOD: (objectId: string, distance: number) => ObjectLOD
  /** Batch update multiple object LODs (more efficient) */
  batchUpdateLODs: (updates: Array<{ id: string; distance: number }>) => void
  /** Remove object from LOD tracking */
  removeObject: (objectId: string) => void
  /** Clear all object LODs */
  clearAll: () => void
  /** Get LOD for a specific object */
  getObjectLOD: (objectId: string) => ObjectLOD | undefined
  /** Check if object should track cursor */
  shouldTrackCursor: (objectId: string) => boolean
  /** Check if object should animate fully */
  shouldAnimateFully: (objectId: string) => boolean
  /** Check if object can receive hover */
  canReceiveHover: (objectId: string) => boolean
}

type LODStore = LODState & LODActions

const initialState: LODState = {
  config: DEFAULT_LOD_CONFIG,
  objectLODs: new Map(),
  cameraPositionRef: [0, 5, 10],
  objectCount: 0,
  levelCounts: { high: 0, medium: 0, low: 0, culled: 0 },
}

/**
 * Zustand store for LOD management.
 *
 * CRITICAL: LOD calculations happen in useFrame via getState() to avoid re-renders.
 * Only aggregate statistics trigger React updates.
 */
export const useLODStore = create<LODStore>((set, get) => ({
  ...initialState,

  setConfig: (config) =>
    set((state) => ({
      config: { ...state.config, ...config },
    })),

  setThresholds: (thresholds) =>
    set((state) => ({
      config: {
        ...state.config,
        thresholds: { ...state.config.thresholds, ...thresholds },
      },
    })),

  updateCameraPosition: (position) => {
    // Direct mutation for performance - no React update needed
    get().cameraPositionRef = position
  },

  calculateLODLevel: (distance) => {
    const { thresholds, hysteresis } = get().config

    // Apply hysteresis to prevent flickering at boundaries
    const h = hysteresis

    if (distance < thresholds.high) return 'high'
    if (distance < thresholds.medium * (1 + h)) return 'medium'
    if (distance < thresholds.low * (1 + h)) return 'low'
    if (distance < thresholds.culled * (1 + h)) return 'low'
    return 'culled'
  },

  updateObjectLOD: (objectId, distance) => {
    const state = get()
    const level = state.calculateLODLevel(distance)
    const now = performance.now()

    const objectLOD: ObjectLOD = {
      level,
      distance,
      lastUpdated: now,
    }

    // Direct Map mutation for performance
    state.objectLODs.set(objectId, objectLOD)

    return objectLOD
  },

  batchUpdateLODs: (updates) => {
    const state = get()
    const levelCounts: Record<LODLevel, number> = { high: 0, medium: 0, low: 0, culled: 0 }
    const now = performance.now()

    for (const { id, distance } of updates) {
      const level = state.calculateLODLevel(distance)
      state.objectLODs.set(id, {
        level,
        distance,
        lastUpdated: now,
      })
      levelCounts[level]++
    }

    // Only update React state for statistics (throttled)
    set({
      objectCount: updates.length,
      levelCounts,
    })
  },

  removeObject: (objectId) => {
    get().objectLODs.delete(objectId)
  },

  clearAll: () => {
    set({
      objectLODs: new Map(),
      objectCount: 0,
      levelCounts: { high: 0, medium: 0, low: 0, culled: 0 },
    })
  },

  getObjectLOD: (objectId) => {
    return get().objectLODs.get(objectId)
  },

  shouldTrackCursor: (objectId) => {
    const state = get()
    if (!state.config.enableCursorLOD) return true

    const lod = state.objectLODs.get(objectId)
    if (!lod) return true

    // Only track cursor for high and medium LOD
    return lod.level === 'high' || lod.level === 'medium'
  },

  shouldAnimateFully: (objectId) => {
    const state = get()
    if (!state.config.enableAnimationLOD) return true

    const lod = state.objectLODs.get(objectId)
    if (!lod) return true

    // Full animations only for high LOD
    return lod.level === 'high'
  },

  canReceiveHover: (objectId) => {
    const state = get()
    if (!state.config.enableHoverLOD) return true

    const lod = state.objectLODs.get(objectId)
    if (!lod) return true

    // Hover only for high and medium LOD
    return lod.level === 'high' || lod.level === 'medium'
  },
}))

/**
 * Selector for LOD statistics (safe to use in components)
 */
export const selectLODStats = (state: LODStore) => ({
  objectCount: state.objectCount,
  levelCounts: state.levelCounts,
})

/**
 * Selector for LOD config
 */
export const selectLODConfig = (state: LODStore) => state.config
