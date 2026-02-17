/**
 * FurnitureManager - Renders all furniture in the scene.
 * Connects to the furniture store and renders FurnitureItem components.
 */

import { memo, useCallback, useMemo } from 'react'
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
const LOCAL_ORIGIN: [number, number, number] = [0, 0, 0]

interface FurnitureManagerProps {
  /** Called when furniture is clicked */
  onFurnitureClick?: (furniture: FurnitureInstance) => void
  /** Whether furniture is interactive (clickable) */
  interactive?: boolean
  /** Whether furniture is draggable */
  draggable?: boolean
}

interface FurnitureNodeProps {
  furniture: FurnitureInstance
  furnitureScale: number
  draggable: boolean
  interactive: boolean
  isNightTime: boolean
  constrainPosition: (position: [number, number, number]) => [number, number, number]
  onFurnitureClick?: (furniture: FurnitureInstance) => void
  onPositionChange: (furnitureId: string, newPosition: [number, number, number]) => void
}

const FurnitureNode = memo(function FurnitureNode({
  furniture,
  furnitureScale,
  draggable,
  interactive,
  isNightTime,
  constrainPosition,
  onFurnitureClick,
  onPositionChange,
}: FurnitureNodeProps) {
  const handleClick = useCallback(() => {
    onFurnitureClick?.(furniture)
  }, [onFurnitureClick, furniture])

  // Resolve effective light state from mode + time of day
  const config = FURNITURE_CONFIGS[furniture.type]
  let effectiveLightOn: boolean | undefined
  if (config.emitsLight) {
    const mode = furniture.lightMode ?? 'auto'
    effectiveLightOn = mode === 'on' ? true : mode === 'off' ? false : isNightTime
  }

  const item = (
    <FurnitureItem
      id={furniture.id}
      type={furniture.type}
      position={draggable ? LOCAL_ORIGIN : furniture.position}
      rotation={furniture.rotation}
      scale={furnitureScale}
      color={furniture.color}
      lightOn={effectiveLightOn}
      onClick={interactive ? handleClick : undefined}
    />
  )

  if (draggable) {
    return (
      <DraggableObject
        objectId={furniture.id}
        position={furniture.position}
        onPositionChange={(pos) => onPositionChange(furniture.id, pos)}
        constrainPosition={constrainPosition}
      >
        {item}
      </DraggableObject>
    )
  }

  return item
})

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
  // Subscribe to derived day/night state so manager doesn't rerender on every time tick.
  const isNightTime = useEnvironmentStore((state) => state.timeValue < 6 || state.timeValue >= 18)

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
      {furnitureList.map((furniture) => (
        <FurnitureNode
          key={furniture.id}
          furniture={furniture}
          furnitureScale={furnitureScale}
          draggable={draggable}
          interactive={interactive}
          isNightTime={isNightTime}
          constrainPosition={constrainPosition}
          onFurnitureClick={handleClick}
          onPositionChange={handlePositionChange}
        />
      ))}

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
