/**
 * DraggableObject - Wrapper component that makes any 3D object draggable.
 * Uses useDragDrop hook and renders children with drag visual feedback.
 *
 * Position during drag is derived from the interaction store's dragState,
 * ensuring smooth tracking even when pointer events are captured by the
 * DragPlane (which happens as soon as the pointer leaves the object mesh).
 */

import { useRef, useState, useEffect, useCallback, type ReactNode } from 'react'
import { useFrame } from '@react-three/fiber'
import type { Group } from 'three'
import * as THREE from 'three'
import { useDragDrop } from '@/hooks/useDragDrop'
import { useInteractionStore } from '@/stores/interactionStore'

interface DraggableObjectProps {
  /** Unique ID for this draggable object */
  objectId: string
  /** Initial position */
  position: [number, number, number]
  /** Children to render (the actual 3D object) */
  children: ReactNode
  /** Whether dragging is enabled */
  enabled?: boolean
  /** Y position for drag plane (default: object's Y) */
  dragPlaneY?: number
  /** Callback when position changes */
  onPositionChange?: (newPosition: [number, number, number]) => void
  /** Callback when drag starts */
  onDragStart?: () => void
  /** Callback when drag ends */
  onDragEnd?: () => void
  /** Whether to show drag indicator */
  showDragIndicator?: boolean
  /** Optional constraint function for drag positions */
  constrainPosition?: (position: [number, number, number]) => [number, number, number]
}

// Stable reference for animation
const DRAG_LIFT_HEIGHT = 0.2
const DRAG_SCALE_FACTOR = 1.05

/**
 * Compute the constrained world position from the initial position + drag offset.
 * Exported for testing.
 */
// eslint-disable-next-line react-refresh/only-export-components -- exported for unit testing
export function computeDragPosition(
  initialPos: [number, number, number],
  offset: [number, number, number],
  constrainFn?: (pos: [number, number, number]) => [number, number, number],
): [number, number, number] {
  let pos: [number, number, number] = [
    initialPos[0] + offset[0],
    initialPos[1],
    initialPos[2] + offset[2],
  ]
  if (constrainFn) {
    pos = constrainFn(pos)
  }
  return pos
}

/**
 * Wrapper that makes children draggable in 3D space.
 * Shows visual feedback during drag (lift + scale).
 */
export function DraggableObject({
  objectId,
  position: initialPosition,
  children,
  enabled = true,
  dragPlaneY,
  onPositionChange,
  onDragStart,
  onDragEnd,
  showDragIndicator = true,
  constrainPosition,
}: DraggableObjectProps) {
  const groupRef = useRef<Group>(null)
  const [currentPosition, setCurrentPosition] = useState(initialPosition)

  // Subscribe to store drag state for this object — this is the source of
  // truth for position during drag, regardless of whether pointer events
  // arrive on the object mesh or on the DragPlane.
  const storeDragState = useInteractionStore((state) =>
    state.draggedObjectId === objectId ? state.dragState : null
  )
  const isDragging = useInteractionStore((state) => state.draggedObjectId === objectId)

  // Sync with external position prop changes when not dragging
  useEffect(() => {
    if (!isDragging) {
      setCurrentPosition(initialPosition)
    }
  }, [initialPosition, isDragging])

  // Track the last computed drag position so we can persist it on drag end.
  const lastDragPosRef = useRef<[number, number, number] | null>(null)
  // Track previous isDragging to detect the true→false transition.
  const prevIsDraggingRef = useRef(false)

  // Detect drag-end from store transition (handles DragPlane pointer-up
  // where useDragDrop.endDrag never fires on this object).
  const onPositionChangeRef = useRef(onPositionChange)
  onPositionChangeRef.current = onPositionChange
  const onDragStartRef = useRef(onDragStart)
  onDragStartRef.current = onDragStart
  const onDragEndRef = useRef(onDragEnd)
  onDragEndRef.current = onDragEnd

  useEffect(() => {
    if (prevIsDraggingRef.current && !isDragging && lastDragPosRef.current) {
      // Drag just ended (via DragPlane or object pointer-up).
      // The useDragDrop.endDrag path also clears lastDragPosRef, so this
      // only fires when DragPlane ended the drag.
      const finalPos = lastDragPosRef.current
      lastDragPosRef.current = null
      setCurrentPosition(finalPos)
      onPositionChangeRef.current?.(finalPos)
      onDragEndRef.current?.()
    }
    prevIsDraggingRef.current = isDragging
  }, [isDragging])

  const handleDragStart = useCallback(
    (_pos: [number, number, number]) => {
      onDragStartRef.current?.()
    },
    []
  )

  const handleDragEnd = useCallback(
    (_pos: [number, number, number]) => {
      // Pointer-up on the object itself — persist final position.
      const finalPos = lastDragPosRef.current ?? currentPosition
      lastDragPosRef.current = null // Clear so the useEffect doesn't double-fire
      setCurrentPosition(finalPos)
      onPositionChangeRef.current?.(finalPos)
      onDragEndRef.current?.()
    },
    [currentPosition]
  )

  const { dragProps } = useDragDrop(objectId, currentPosition, {
    enabled,
    onDragStart: handleDragStart,
    onDragEnd: handleDragEnd,
    constrainToPlane: true,
    planeY: dragPlaneY ?? initialPosition[1],
  })

  // Animate lift, scale, and position during drag
  useFrame(() => {
    if (!groupRef.current) return

    // Derive target position: from store during drag, from state otherwise
    let targetPos: [number, number, number]
    if (isDragging && storeDragState) {
      targetPos = computeDragPosition(
        initialPosition,
        storeDragState.offset,
        constrainPosition,
      )
      lastDragPosRef.current = targetPos
    } else {
      targetPos = currentPosition
    }

    const targetY = isDragging
      ? targetPos[1] + DRAG_LIFT_HEIGHT
      : targetPos[1]
    const targetScale = isDragging ? DRAG_SCALE_FACTOR : 1

    // Smooth interpolation
    groupRef.current.position.x = THREE.MathUtils.lerp(
      groupRef.current.position.x,
      targetPos[0],
      0.3
    )
    groupRef.current.position.y = THREE.MathUtils.lerp(
      groupRef.current.position.y,
      targetY,
      0.2
    )
    groupRef.current.position.z = THREE.MathUtils.lerp(
      groupRef.current.position.z,
      targetPos[2],
      0.3
    )

    const currentScale = groupRef.current.scale.x
    const newScale = THREE.MathUtils.lerp(currentScale, targetScale, 0.2)
    groupRef.current.scale.setScalar(newScale)
  })

  return (
    <group ref={groupRef} position={currentPosition} {...dragProps}>
      {children}

      {/* Drag indicator - shadow ring on ground */}
      {showDragIndicator && isDragging && (
        <mesh
          position={[0, -currentPosition[1] - 0.01, 0]}
          rotation={[-Math.PI / 2, 0, 0]}
        >
          <ringGeometry args={[0.3, 0.5, 32]} />
          <meshBasicMaterial color="#ffffff" transparent opacity={0.3} />
        </mesh>
      )}
    </group>
  )
}
