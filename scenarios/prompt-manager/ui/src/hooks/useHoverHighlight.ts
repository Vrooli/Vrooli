/**
 * useHoverHighlight - Hook for managing hover state on 3D objects.
 * Provides event handlers and state for highlighting objects on hover.
 */
// AI_CHECK: R3F_HOVER_EVENT_CHURN=1 | LAST: 2026-02-17

import { useCallback, useMemo } from 'react'
import { useInteractionStore } from '@/stores/interactionStore'

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

  const onPointerOver = useCallback(
    (e: { stopPropagation: () => void }) => {
      if (!enabled || useInteractionStore.getState().isDragging) return
      e.stopPropagation()
      setHovered(objectId)
      if (document.body.style.cursor !== cursor) {
        document.body.style.cursor = cursor
      }
    },
    [objectId, setHovered, cursor, enabled]
  )

  const onPointerOut = useCallback(() => {
    if (!enabled) return
    const { hoveredObjectId } = useInteractionStore.getState()
    if (hoveredObjectId === objectId) {
      setHovered(null)
    }
    if (document.body.style.cursor !== 'auto') {
      document.body.style.cursor = 'auto'
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
