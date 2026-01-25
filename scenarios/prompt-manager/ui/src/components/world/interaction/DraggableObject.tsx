/**
 * DraggableObject - Wrapper component that makes any 3D object draggable.
 * Uses useDragDrop hook and renders children with drag visual feedback.
 */

import { useRef, useState, useCallback, type ReactNode } from 'react'
import { useFrame } from '@react-three/fiber'
import type { Group } from 'three'
import * as THREE from 'three'
import { useDragDrop } from '@/hooks/useDragDrop'

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
}

// Stable reference for animation
const DRAG_LIFT_HEIGHT = 0.2
const DRAG_SCALE_FACTOR = 1.05

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
}: DraggableObjectProps) {
  const groupRef = useRef<Group>(null)
  const [currentPosition, setCurrentPosition] = useState(initialPosition)

  const handleDragStart = useCallback(
    (_pos: [number, number, number]) => {
      onDragStart?.()
    },
    [onDragStart]
  )

  const handleDrag = useCallback(
    (_pos: [number, number, number], offset: [number, number, number]) => {
      // Apply offset to initial position
      const newPos: [number, number, number] = [
        initialPosition[0] + offset[0],
        initialPosition[1],
        initialPosition[2] + offset[2],
      ]
      setCurrentPosition(newPos)
    },
    [initialPosition]
  )

  const handleDragEnd = useCallback(
    (_pos: [number, number, number]) => {
      onPositionChange?.(currentPosition)
      onDragEnd?.()
    },
    [currentPosition, onPositionChange, onDragEnd]
  )

  const { isDragging, dragProps } = useDragDrop(objectId, currentPosition, {
    enabled,
    onDragStart: handleDragStart,
    onDrag: handleDrag,
    onDragEnd: handleDragEnd,
    constrainToPlane: true,
    planeY: dragPlaneY ?? initialPosition[1],
  })

  // Animate lift and scale during drag
  useFrame(() => {
    if (!groupRef.current) return

    const targetY = isDragging
      ? currentPosition[1] + DRAG_LIFT_HEIGHT
      : currentPosition[1]
    const targetScale = isDragging ? DRAG_SCALE_FACTOR : 1

    // Smooth interpolation
    groupRef.current.position.x = THREE.MathUtils.lerp(
      groupRef.current.position.x,
      currentPosition[0],
      0.3
    )
    groupRef.current.position.y = THREE.MathUtils.lerp(
      groupRef.current.position.y,
      targetY,
      0.2
    )
    groupRef.current.position.z = THREE.MathUtils.lerp(
      groupRef.current.position.z,
      currentPosition[2],
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
