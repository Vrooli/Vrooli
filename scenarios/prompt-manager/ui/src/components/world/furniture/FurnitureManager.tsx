/**
 * FurnitureManager - Renders all furniture in the scene.
 * Connects to the furniture store and renders FurnitureItem components.
 */

import { useCallback, useMemo } from 'react'
import { FurnitureItem } from './FurnitureItem'
import { SeatHandle3D } from './SeatHandle3D'
import { DraggableObject } from '../interaction'
import { useFurnitureStore, useFurnitureList } from '@/stores/furnitureStore'
import { useWorldScaleStore } from '@/stores/worldScaleStore'
import { useWorldSeatsStore } from '@/stores/worldSeatsStore'
import { useWorldEditorStore } from '@/stores/worldEditorStore'
import { useEnvironmentStore } from '@/stores/environmentStore'
import type { FurnitureInstance, SeatPosition } from '@/types/furniture'
import { FURNITURE_CONFIGS } from '@/types/furniture'
import { applyPlacementConstraints } from '@/lib/world'

/** Stable empty array for selector fallback — avoids new references triggering R3F re-render loops. */
const EMPTY_SEATS: SeatPosition[] = []

interface FurnitureManagerProps {
  /** Called when furniture is clicked */
  onFurnitureClick?: (furniture: FurnitureInstance) => void
  /** Whether furniture is interactive (clickable) */
  interactive?: boolean
  /** Whether furniture is draggable */
  draggable?: boolean
}

/**
 * Manages rendering of all furniture instances in the world.
 */
export function FurnitureManager({
  onFurnitureClick,
  interactive = true,
  draggable = false,
}: FurnitureManagerProps) {
  const furnitureList = useFurnitureList()
  const furnitureScale = useWorldScaleStore((state) => state.furniture)
  const moveFurniture = useFurnitureStore((state) => state.moveFurniture)
  const placementConfig = useEnvironmentStore((state) => state.current.placement)
  const boundaryConfig = useEnvironmentStore((state) => state.current.boundary)
  const groundSize = useEnvironmentStore((state) => state.current.ground.size)
  const timeValue = useEnvironmentStore((state) => state.timeValue)

  // Campfire (and other light-emitting furniture) auto-lights at night
  const isNightTime = timeValue < 6 || timeValue >= 18

  const constrainPosition = useMemo(() => {
    return (position: [number, number, number]) =>
      applyPlacementConstraints(position, {
        placement: placementConfig,
        boundary: boundaryConfig,
        groundSize,
      })
  }, [placementConfig, boundaryConfig, groundSize])

  const handleClick = useCallback(
    (furniture: FurnitureInstance) => {
      onFurnitureClick?.(furniture)
    },
    [onFurnitureClick]
  )

  const handlePositionChange = useCallback(
    (furnitureId: string, newPosition: [number, number, number]) => {
      moveFurniture(furnitureId, newPosition)
    },
    [moveFurniture]
  )

  // Seat editing state
  const editingSeatFurnitureId = useWorldEditorStore((s) => s.editingSeatFurnitureId)
  const editingSeatType = useWorldEditorStore((s) => s.editingSeatType)
  const editingSeats = useWorldSeatsStore(
    useCallback((s) => editingSeatType ? s.seats[editingSeatType] ?? EMPTY_SEATS : EMPTY_SEATS, [editingSeatType])
  )
  const updateSeat = useWorldSeatsStore((s) => s.updateSeat)

  // Find the furniture instance being seat-edited
  const editingFurniture = editingSeatFurnitureId
    ? furnitureList.find((f) => f.id === editingSeatFurnitureId)
    : null

  const handleSeatPositionChange = useCallback(
    (index: number, newLocalPos: [number, number, number]) => {
      if (!editingSeatType) return
      const currentSeat = editingSeats[index]
      if (!currentSeat) return
      updateSeat(editingSeatType, index, { ...currentSeat, position: newLocalPos })
    },
    [editingSeatType, editingSeats, updateSeat]
  )

  return (
    <group name="furniture-manager">
      {furnitureList.map((furniture) => {
        // Resolve effective light state from mode + time of day
        const config = FURNITURE_CONFIGS[furniture.type]
        let effectiveLightOn: boolean | undefined
        if (config.emitsLight) {
          const mode = furniture.lightMode ?? 'auto'
          effectiveLightOn = mode === 'on' ? true : mode === 'off' ? false : isNightTime
        }

        const item = (
          <FurnitureItem
            key={furniture.id}
            id={furniture.id}
            type={furniture.type}
            position={draggable ? [0, 0, 0] : furniture.position}
            rotation={furniture.rotation}
            scale={furnitureScale}
            color={furniture.color}
            lightOn={effectiveLightOn}
            onClick={interactive ? () => handleClick(furniture) : undefined}
          />
        )

        if (draggable) {
          return (
            <DraggableObject
              key={furniture.id}
              objectId={furniture.id}
              position={furniture.position}
              onPositionChange={(pos) => handlePositionChange(furniture.id, pos)}
              constrainPosition={constrainPosition}
            >
              {item}
            </DraggableObject>
          )
        }

        return item
      })}

      {/* Seat editing handles */}
      {editingFurniture && editingSeats.map((seat, index) => (
        <SeatHandle3D
          key={`seat-${index}`}
          seat={seat}
          index={index}
          furniturePosition={editingFurniture.position}
          furnitureRotation={editingFurniture.rotation}
          onPositionChange={(newLocalPos) => handleSeatPositionChange(index, newLocalPos)}
        />
      ))}
    </group>
  )
}

/**
 * Hook to add furniture at a position
 */
