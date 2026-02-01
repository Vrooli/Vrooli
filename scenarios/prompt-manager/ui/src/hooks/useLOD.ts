/**
 * useLOD - Hook for managing Level of Detail for 3D objects.
 * Provides distance-based optimization for cursor tracking, animations, and rendering.
 *
 * CRITICAL: This hook uses refs and getState() to avoid re-renders in the animation loop.
 * LOD calculations happen in useFrame and results are accessed via refs.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#lod-system

import { useRef, useEffect, useCallback } from 'react'
import { useFrame, useThree } from '@react-three/fiber'
import { useLODStore } from '@/stores/lodStore'
import type { LODLevel, LODState } from '@/types/lod'

interface UseLODOptions {
  /** Object ID for LOD tracking */
  objectId: string
  /** Whether LOD is enabled for this object */
  enabled?: boolean
  /** Update frequency in frames (1 = every frame, 10 = every 10 frames) */
  updateFrequency?: number
}

interface UseLODResult {
  /** Current LOD level (via ref to avoid re-renders) */
  lodLevelRef: React.MutableRefObject<LODLevel>
  /** Current distance from camera (via ref) */
  distanceRef: React.MutableRefObject<number>
  /** Whether cursor tracking should be active */
  shouldTrackCursorRef: React.MutableRefObject<boolean>
  /** Whether full animations should play */
  shouldAnimateFullyRef: React.MutableRefObject<boolean>
  /** Whether hover is enabled */
  canReceiveHoverRef: React.MutableRefObject<boolean>
  /** Whether object is visible (not culled) */
  isVisibleRef: React.MutableRefObject<boolean>
  /** Get current LOD state (snapshot) */
  getLODState: () => LODState
}

/**
 * Hook for managing LOD for a single 3D object.
 *
 * Uses refs exclusively to avoid triggering React re-renders from useFrame.
 * Call getLODState() when you need the current values.
 *
 * @example
 * ```tsx
 * function Agent({ id, position }) {
 *   const groupRef = useRef<Group>(null)
 *   const {
 *     shouldTrackCursorRef,
 *     shouldAnimateFullyRef,
 *     isVisibleRef
 *   } = useLOD({ objectId: id })
 *
 *   useFrame(() => {
 *     if (!groupRef.current || !isVisibleRef.current) return
 *
 *     // Only track cursor if LOD allows
 *     if (shouldTrackCursorRef.current && cursorPosition) {
 *       // ... cursor tracking logic
 *     }
 *
 *     // Only run complex animations if LOD allows
 *     if (shouldAnimateFullyRef.current) {
 *       // ... full animation
 *     } else {
 *       // ... simplified animation
 *     }
 *   })
 *
 *   if (!isVisibleRef.current) return null
 *   return <group ref={groupRef} position={position}>...</group>
 * }
 * ```
 */
export function useLOD({
  objectId,
  enabled = true,
  updateFrequency = 5, // Update every 5 frames by default
}: UseLODOptions): UseLODResult {
  const { camera } = useThree()

  // Refs to store LOD state without triggering re-renders
  const lodLevelRef = useRef<LODLevel>('high')
  const distanceRef = useRef<number>(0)
  const shouldTrackCursorRef = useRef<boolean>(true)
  const shouldAnimateFullyRef = useRef<boolean>(true)
  const canReceiveHoverRef = useRef<boolean>(true)
  const isVisibleRef = useRef<boolean>(true)
  const frameCountRef = useRef<number>(0)

  // Object position ref (set by parent component)
  const objectPositionRef = useRef<[number, number, number]>([0, 0, 0])

  // Store access via getState() to avoid subscriptions
  const updateObjectLOD = useCallback(
    (distance: number) => {
      return useLODStore.getState().updateObjectLOD(objectId, distance)
    },
    [objectId]
  )

  const calculateLODLevel = useCallback((distance: number) => {
    return useLODStore.getState().calculateLODLevel(distance)
  }, [])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      useLODStore.getState().removeObject(objectId)
    }
  }, [objectId])

  // Update LOD in animation frame (no React state updates!)
  useFrame(() => {
    if (!enabled) {
      lodLevelRef.current = 'high'
      shouldTrackCursorRef.current = true
      shouldAnimateFullyRef.current = true
      canReceiveHoverRef.current = true
      isVisibleRef.current = true
      return
    }

    // Throttle updates based on frequency
    frameCountRef.current++
    if (frameCountRef.current % updateFrequency !== 0) return

    // Get object position (stored by parent) and camera position
    const [ox, oy, oz] = objectPositionRef.current
    const cx = camera.position.x
    const cy = camera.position.y
    const cz = camera.position.z

    // Calculate distance (avoiding Math.hypot for performance)
    const dx = ox - cx
    const dy = oy - cy
    const dz = oz - cz
    const distance = Math.sqrt(dx * dx + dy * dy + dz * dz)

    // Update distance ref
    distanceRef.current = distance

    // Calculate LOD level
    const level = calculateLODLevel(distance)
    lodLevelRef.current = level

    // Update derived state based on LOD level
    isVisibleRef.current = level !== 'culled'
    shouldTrackCursorRef.current = level === 'high' || level === 'medium'
    shouldAnimateFullyRef.current = level === 'high'
    canReceiveHoverRef.current = level === 'high' || level === 'medium'

    // Update store (for aggregate statistics only)
    updateObjectLOD(distance)
  })

  const getLODState = useCallback((): LODState => {
    return {
      level: lodLevelRef.current,
      distance: distanceRef.current,
      isVisible: isVisibleRef.current,
      shouldTrackCursor: shouldTrackCursorRef.current,
      useSimplifiedAnimations: !shouldAnimateFullyRef.current,
      canReceiveHover: canReceiveHoverRef.current,
    }
  }, [])

  return {
    lodLevelRef,
    distanceRef,
    shouldTrackCursorRef,
    shouldAnimateFullyRef,
    canReceiveHoverRef,
    isVisibleRef,
    getLODState,
  }
}

/**
 * Hook for batch LOD updates - more efficient for many objects.
 * Use this at the scene level to update all agents at once.
 *
 * @example
 * ```tsx
 * function WorldScene({ agents }) {
 *   const { updateAllLODs } = useLODManager()
 *
 *   useFrame(() => {
 *     // Update all agent LODs in one batch
 *     updateAllLODs(agents.map(m => ({
 *       id: m.id,
 *       position: m.position
 *     })))
 *   })
 *
 *   return agents.map(m => <Member key={m.id} {...m} />)
 * }
 * ```
 */
export function useLODManager() {
  const { camera } = useThree()
  const frameCountRef = useRef(0)

  const updateAllLODs = useCallback(
    (
      objects: Array<{ id: string; position: [number, number, number] }>,
      frequency = 10
    ) => {
      frameCountRef.current++
      if (frameCountRef.current % frequency !== 0) return

      const cx = camera.position.x
      const cy = camera.position.y
      const cz = camera.position.z

      const updates = objects.map(({ id, position }) => {
        const [ox, oy, oz] = position
        const dx = ox - cx
        const dy = oy - cy
        const dz = oz - cz
        const distance = Math.sqrt(dx * dx + dy * dy + dz * dz)

        return { id, distance }
      })

      useLODStore.getState().batchUpdateLODs(updates)
    },
    [camera]
  )

  const getLODStats = useCallback(() => {
    const state = useLODStore.getState()
    return {
      objectCount: state.objectCount,
      levelCounts: state.levelCounts,
    }
  }, [])

  return {
    updateAllLODs,
    getLODStats,
  }
}

/**
 * Hook to get LOD level for a specific object (reactive).
 * Use sparingly - prefer refs for animation loops.
 */
export function useObjectLODLevel(objectId: string): LODLevel {
  const lod = useLODStore((state) => state.getObjectLOD(objectId))
  return lod?.level ?? 'high'
}

/**
 * Hook to get LOD statistics (for UI display).
 */
export function useLODStats() {
  const objectCount = useLODStore((state) => state.objectCount)
  const levelCounts = useLODStore((state) => state.levelCounts)

  return {
    objectCount,
    levelCounts,
    highCount: levelCounts.high,
    mediumCount: levelCounts.medium,
    lowCount: levelCounts.low,
    culledCount: levelCounts.culled,
  }
}
