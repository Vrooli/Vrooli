/**
 * useHoverHighlight - Hook for managing hover state on 3D objects.
 * Provides event handlers and state for highlighting objects on hover.
 */
// AI_CHECK: R3F_HOVER_EVENT_CHURN=2 | LAST: 2026-02-17

import { useCallback, useMemo, useRef } from 'react'
import { useInteractionStore } from '@/stores/interactionStore'
import { usePerformanceStore } from '@/stores/performanceStore'
import { useLODStore } from '@/stores/lodStore'

const HOVER_UPDATE_INTERVAL_MS = 66 // ~15Hz
const HOVER_MARKER_INTERVAL_MS = 100

interface HoverHighlightResult {
  /** Whether this object is currently hovered */
  isHovered: boolean
  /** Handler for pointer enter */
  onPointerOver: (e: { stopPropagation: () => void }) => void
  /** Handler for pointer leave */
  onPointerOut: () => void
  /** Props object for easy spreading onto mesh */
  hoverProps: {
    onPointerOver: (e: { stopPropagation: () => void }) => void
    onPointerOut: () => void
  }
}

/**
 * Hook for managing hover highlighting on a 3D object.
 *
 * @param objectId - Unique ID for this object
 * @param options - Optional configuration
 * @returns Hover state and event handlers
 *
 * @example
 * ```tsx
 * function MyMesh({ id }) {
 *   const { isHovered, hoverProps } = useHoverHighlight(id)
 *
 *   return (
 *     <mesh {...hoverProps}>
 *       <boxGeometry />
 *       <meshStandardMaterial
 *         color={isHovered ? '#ff0000' : '#0000ff'}
 *       />
 *     </mesh>
 *   )
 * }
 * ```
 */
export function useHoverHighlight(
  objectId: string,
  options: {
    /** Custom cursor to show on hover */
    cursor?: string
    /** Whether hovering is enabled */
    enabled?: boolean
  } = {}
): HoverHighlightResult {
  const { cursor = 'pointer', enabled = true } = options

  const setHovered = useInteractionStore((state) => state.setHovered)
  const isHovered = useInteractionStore((state) => enabled && state.hoveredObjectId === objectId)
  const hoverEventCountRef = useRef(0)
  const lastHoverUpdateMsRef = useRef(0)
  const lastMarkerMsRef = useRef(0)

  const onPointerOver = useCallback(
    (e: { stopPropagation: () => void }) => {
      if (!enabled || useInteractionStore.getState().isDragging) return
      if (!useLODStore.getState().canReceiveHover(objectId)) return

      const t0 = performance.now()
      const now = t0
      const wasHovered = useInteractionStore.getState().hoveredObjectId === objectId
      if (
        !wasHovered &&
        now - lastHoverUpdateMsRef.current < HOVER_UPDATE_INTERVAL_MS
      ) {
        return
      }
      e.stopPropagation()
      setHovered(objectId)
      lastHoverUpdateMsRef.current = now
      if (document.body.style.cursor !== cursor) {
        document.body.style.cursor = cursor
      }
      if (!wasHovered && now - lastMarkerMsRef.current >= HOVER_MARKER_INTERVAL_MS) {
        usePerformanceStore.getState().recordTraceMarker('hover-start', `Hover start: ${objectId}`)
        lastMarkerMsRef.current = now
      }
      hoverEventCountRef.current++
      if (hoverEventCountRef.current % 10 === 0) {
        usePerformanceStore.getState().recordSubsystemSample(
          'interaction.hover',
          performance.now() - t0
        )
      }
    },
    [objectId, setHovered, cursor, enabled]
  )

  const onPointerOut = useCallback(() => {
    if (!enabled) return
    const t0 = performance.now()
    const now = t0
    const { hoveredObjectId } = useInteractionStore.getState()
    if (hoveredObjectId === objectId) {
      setHovered(null)
      lastHoverUpdateMsRef.current = now
      if (now - lastMarkerMsRef.current >= HOVER_MARKER_INTERVAL_MS) {
        usePerformanceStore.getState().recordTraceMarker('hover-end', `Hover end: ${objectId}`)
        lastMarkerMsRef.current = now
      }
    }
    if (document.body.style.cursor !== 'auto') {
      document.body.style.cursor = 'auto'
    }
    hoverEventCountRef.current++
    if (hoverEventCountRef.current % 10 === 0) {
      usePerformanceStore.getState().recordSubsystemSample(
        'interaction.hover',
        performance.now() - t0
      )
    }
  }, [objectId, setHovered, enabled])

  const hoverProps = useMemo(
    () => ({
      onPointerOver,
      onPointerOut,
    }),
    [onPointerOver, onPointerOut]
  )

  return {
    isHovered: enabled && isHovered,
    onPointerOver,
    onPointerOut,
    hoverProps,
  }
}

/**
 * Hook to check if any object is currently hovered.
 */
export function useIsAnythingHovered(): boolean {
  return useInteractionStore((state) => state.hoveredObjectId !== null)
}

/**
 * Hook to get the currently hovered object ID.
 */
export function useHoveredObjectId(): string | null {
  return useInteractionStore((state) => state.hoveredObjectId)
}
