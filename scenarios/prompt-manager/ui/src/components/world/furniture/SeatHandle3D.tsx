/**
 * SeatHandle3D - 3D handle for visually editing a single seat position.
 * Renders a sphere at the seat position with a cone arrow showing facing direction.
 * Draggable on the XZ plane via DraggableObject.
 */

import { useMemo } from 'react'
import { Text } from '@react-three/drei'
import { DraggableObject } from '../interaction'
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
  const worldPosition = useMemo<[number, number, number]>(() => {
    const cos = Math.cos(furnitureRotation)
    const sin = Math.sin(furnitureRotation)
    const [sx, sy, sz] = seat.position
    return [
      furniturePosition[0] + sx * cos - sz * sin,
      furniturePosition[1] + sy,
      furniturePosition[2] + sx * sin + sz * cos,
    ]
  }, [seat.position, furniturePosition, furnitureRotation])

  // World-space facing direction
  const worldRotation = furnitureRotation + seat.rotation

  // Arrow direction vector for the cone
  const arrowOffset = useMemo<[number, number, number]>(() => {
    return [
      Math.sin(worldRotation) * 0.15,
      0,
      Math.cos(worldRotation) * 0.15,
    ]
  }, [worldRotation])

  const handleDragEnd = (newPos: [number, number, number]) => {
    // Convert world position back to seat-local offset
    const cos = Math.cos(-furnitureRotation)
    const sin = Math.sin(-furnitureRotation)
    const dx = newPos[0] - furniturePosition[0]
    const dy = newPos[1] - furniturePosition[1]
    const dz = newPos[2] - furniturePosition[2]
    onPositionChange([
      dx * cos - dz * sin,
      dy,
      dx * sin + dz * cos,
    ])
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
