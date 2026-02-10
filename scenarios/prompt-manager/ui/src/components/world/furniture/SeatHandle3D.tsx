/**
 * SeatHandle3D - 3D handle for visually editing a single seat position.
 * Renders a sphere at the seat position with a cone arrow showing facing direction.
 * Draggable on the XZ plane via DraggableObject.
 */

import { useMemo } from 'react'
import { Text } from '@react-three/drei'
import { DraggableObject } from '../interaction'
import { seatLocalToWorld, seatWorldToLocal, seatFacingArrowOffset } from '@/lib/world'
import type { SeatPosition } from '@/types/furniture'

/** Colors cycle for seats so they're visually distinguishable */
const SEAT_COLORS = ['#ef4444', '#3b82f6', '#22c55e', '#f59e0b', '#a855f7', '#ec4899']

interface SeatHandle3DProps {
  seat: SeatPosition
  index: number
  furniturePosition: [number, number, number]
  furnitureRotation: number
  onPositionChange: (worldPos: [number, number, number]) => void
}

export function SeatHandle3D({
  seat,
  index,
  furniturePosition,
  furnitureRotation,
  onPositionChange,
}: SeatHandle3DProps) {
  const color = SEAT_COLORS[index % SEAT_COLORS.length] ?? '#ffffff'

  // Compute world position: furniture origin + rotated seat offset
  const worldPosition = useMemo<[number, number, number]>(
    () => seatLocalToWorld(seat.position, furniturePosition, furnitureRotation),
    [seat.position, furniturePosition, furnitureRotation],
  )

  // World-space facing direction
  const worldRotation = furnitureRotation + seat.rotation

  // Arrow direction vector for the cone
  const arrowOffset = useMemo<[number, number, number]>(
    () => seatFacingArrowOffset(worldRotation),
    [worldRotation],
  )

  const handleDragEnd = (newPos: [number, number, number]) => {
    onPositionChange(seatWorldToLocal(newPos, furniturePosition, furnitureRotation))
  }

  return (
    <DraggableObject
      objectId={`seat-handle-${index}`}
      position={worldPosition}
      onPositionChange={handleDragEnd}
      showDragIndicator={false}
    >
      {/* Seat sphere */}
      <mesh>
        <sphereGeometry args={[0.08, 16, 16]} />
        <meshStandardMaterial color={color} />
      </mesh>

      {/* Direction arrow (cone) */}
      <mesh
        position={arrowOffset}
        rotation={[0, -worldRotation, 0]}
      >
        <coneGeometry args={[0.04, 0.1, 8]} />
        <meshStandardMaterial color={color} />
      </mesh>

      {/* Seat number label */}
      <Text
        position={[0, 0.15, 0]}
        fontSize={0.08}
        color="white"
        anchorX="center"
        anchorY="middle"
        outlineWidth={0.01}
        outlineColor="black"
      >
        {String(index + 1)}
      </Text>
    </DraggableObject>
  )
}
