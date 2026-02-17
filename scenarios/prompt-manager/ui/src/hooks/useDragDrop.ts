/**
 * useDragDrop - Hook for drag-and-drop functionality in 3D space.
 * Manages drag state and provides event handlers.
 */
// AI_CHECK: R3F_DRAG_RENDER_PATH=1 | LAST: 2026-02-17

import { useCallback, useRef } from 'react'
import { useThree } from '@react-three/fiber'
import * as THREE from 'three'
import { useInteractionStore } from '@/stores/interactionStore'
import { usePerformanceStore } from '@/stores/performanceStore'

interface DragDropResult {
  /** Whether this object is currently being dragged */
  isDragging: boolean
  /** Start drag operation */
  startDrag: (e: { stopPropagation: () => void; point: THREE.Vector3 }) => void
  /** Update drag position */
  updateDrag: (e: { point: THREE.Vector3 }) => void
  /** End drag operation */
  endDrag: () => void
  /** Cancel drag operation */
  cancelDrag: () => void
  /** Current drag offset from start position */
  dragOffset: [number, number, number] | null
  /** Props for pointer events */
  dragProps: {
    onPointerDown: (e: { stopPropagation: () => void; point: THREE.Vector3 }) => void
    onPointerMove: (e: { point: THREE.Vector3 }) => void
    onPointerUp: () => void
  }
}

interface DragDropOptions {
  /** Whether dragging is enabled */
  enabled?: boolean
  /** Callback when drag starts */
  onDragStart?: (position: [number, number, number]) => void
  /** Callback during drag */
  onDrag?: (position: [number, number, number], offset: [number, number, number]) => void
  /** Callback when drag ends */
  onDragEnd?: (position: [number, number, number]) => void
  /** Constrain drag to horizontal plane (Y = startY) */
  constrainToPlane?: boolean
  /** Y value for plane constraint */
  planeY?: number
}

/**
 * Hook for enabling drag-and-drop on a 3D object.
 *
 * @param objectId - Unique ID for this object
 * @param currentPosition - Current position of the object
 * @param options - Configuration options
 *
 * @example
 * ```tsx
 * function DraggableCube({ id, position }) {
 *   const [pos, setPos] = useState(position)
 *   const { isDragging, dragProps } = useDragDrop(id, pos, {
 *     onDrag: (newPos) => setPos(newPos),
 *   })
 *
 *   return (
 *     <mesh position={pos} {...dragProps}>
 *       <boxGeometry />
 *       <meshStandardMaterial color={isDragging ? 'orange' : 'blue'} />
 *     </mesh>
 *   )
 * }
 * ```
 */
export function useDragDrop(
  objectId: string,
  currentPosition: [number, number, number],
  options: DragDropOptions = {}
): DragDropResult {
  const {
    enabled = true,
    onDragStart,
    onDrag,
    onDragEnd,
    constrainToPlane = true,
    planeY,
  } = options

  const { raycaster } = useThree()

  const startDragFn = useInteractionStore((state) => state.startDrag)
  const updateDragFn = useInteractionStore((state) => state.updateDrag)
  const endDragFn = useInteractionStore((state) => state.endDrag)
  const cancelDragFn = useInteractionStore((state) => state.cancelDrag)
  const isDragging = useInteractionStore((state) => state.draggedObjectId === objectId)
  const dragOffset = useInteractionStore((state) =>
    state.draggedObjectId === objectId && state.dragState ? state.dragState.offset : null
  )

  const dragPlaneRef = useRef(
    new THREE.Plane(new THREE.Vector3(0, 1, 0), -(planeY ?? currentPosition[1]))
  )
  const intersectionPoint = useRef(new THREE.Vector3())
  const dragUpdateCountRef = useRef(0)

  const startDrag = useCallback(
    (e: { stopPropagation: () => void; point: THREE.Vector3 }) => {
      if (!enabled) return
      e.stopPropagation()

      const startPos: [number, number, number] = [
        e.point.x,
        constrainToPlane ? (planeY ?? currentPosition[1]) : e.point.y,
        e.point.z,
      ]

      startDragFn(objectId, startPos)
      onDragStart?.(startPos)
      usePerformanceStore.getState().recordTraceMarker('drag-start', `Drag start: ${objectId}`)

      // Update drag plane height
      dragPlaneRef.current.constant = -(planeY ?? currentPosition[1])
    },
    [objectId, enabled, startDragFn, onDragStart, constrainToPlane, planeY, currentPosition]
  )

  const updateDrag = useCallback(
    (e: { point: THREE.Vector3 }) => {
      if (!isDragging || !enabled) return
      const t0 = performance.now()
      usePerformanceStore.getState().recordPointerMoveEvent()

      let newPos: [number, number, number]

      if (constrainToPlane) {
        // Project mouse onto drag plane
        const raycastStart = performance.now()
        raycaster.ray.intersectPlane(dragPlaneRef.current, intersectionPoint.current)
        usePerformanceStore.getState().recordRaycastSample(performance.now() - raycastStart)
        newPos = [
          intersectionPoint.current.x,
          planeY ?? currentPosition[1],
          intersectionPoint.current.z,
        ]
      } else {
        newPos = [e.point.x, e.point.y, e.point.z]
      }

      updateDragFn(newPos)

      const dragState = useInteractionStore.getState().dragState
      if (dragState) {
        const offset: [number, number, number] = [
          newPos[0] - dragState.startPosition[0],
          newPos[1] - dragState.startPosition[1],
          newPos[2] - dragState.startPosition[2],
        ]
        onDrag?.(newPos, offset)
      }

      dragUpdateCountRef.current++
      if (dragUpdateCountRef.current % 5 === 0) {
        usePerformanceStore.getState().recordSubsystemSample(
          'interaction.drag.update',
          performance.now() - t0
        )
      }
    },
    [
      isDragging,
      enabled,
      constrainToPlane,
      planeY,
      currentPosition,
      raycaster,
      updateDragFn,
      onDrag,
    ]
  )

  const endDrag = useCallback(() => {
    if (!isDragging) return

    const dragState = useInteractionStore.getState().dragState
    if (dragState) {
      onDragEnd?.(dragState.currentPosition)
      usePerformanceStore.getState().recordTraceMarker('drag-end', `Drag end: ${objectId}`)
    }

    endDragFn()
  }, [isDragging, objectId, onDragEnd, endDragFn])

  const cancelDrag = useCallback(() => {
    if (isDragging) {
      usePerformanceStore.getState().recordTraceMarker('drag-end', `Drag cancel: ${objectId}`)
    }
    cancelDragFn()
  }, [cancelDragFn, isDragging, objectId])

  return {
    isDragging,
    startDrag,
    updateDrag,
    endDrag,
    cancelDrag,
    dragOffset,
    dragProps: {
      onPointerDown: startDrag,
      onPointerMove: updateDrag,
      onPointerUp: endDrag,
    },
  }
}

/**
 * Hook to check if anything is currently being dragged.
 */
export function useIsDragging(): boolean {
  return useInteractionStore((state) => state.isDragging)
}

/**
 * Hook to get the currently dragged object ID.
 */
export function useDraggedObjectId(): string | null {
  return useInteractionStore((state) => state.draggedObjectId)
}
