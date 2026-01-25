/**
 * DragPlane - Invisible plane that catches pointer events for drag operations.
 * Placed at ground level to enable smooth dragging across the scene.
 */

import { useCallback } from 'react'
import { useInteractionStore } from '@/stores/interactionStore'

interface DragPlaneProps {
  /** Size of the plane (width/depth) */
  size?: number
  /** Y position of the plane */
  y?: number
  /** Whether the plane is visible (for debugging) */
  visible?: boolean
}

/**
 * Invisible ground plane that captures pointer events during drag operations.
 * This ensures smooth dragging even when the pointer moves off the dragged object.
 */
export function DragPlane({
  size = 100,
  y = 0,
  visible = false,
}: DragPlaneProps) {
  const isDragging = useInteractionStore((state) => state.isDragging)
  const updateDrag = useInteractionStore((state) => state.updateDrag)
  const endDrag = useInteractionStore((state) => state.endDrag)

  const handlePointerMove = useCallback(
    (e: { point: { x: number; y: number; z: number } }) => {
      if (!isDragging) return
      updateDrag([e.point.x, y, e.point.z])
    },
    [isDragging, updateDrag, y]
  )

  const handlePointerUp = useCallback(() => {
    if (!isDragging) return
    endDrag()
  }, [isDragging, endDrag])

  // Only render when dragging to avoid interfering with other interactions
  if (!isDragging && !visible) {
    return null
  }

  return (
    <mesh
      position={[0, y, 0]}
      rotation={[-Math.PI / 2, 0, 0]}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerUp}
    >
      <planeGeometry args={[size, size]} />
      <meshBasicMaterial
        color="#00ff00"
        transparent
        opacity={visible ? 0.1 : 0}
        depthWrite={false}
      />
    </mesh>
  )
}
