/**
 * FurnitureManager - Renders all furniture in the scene.
 * Connects to the furniture store and renders FurnitureItem components.
 */

import { useCallback } from 'react'
import { FurnitureItem } from './FurnitureItem'
import { DraggableObject } from '../interaction'
import { useFurnitureStore, useFurnitureList } from '@/stores/furnitureStore'
import type { FurnitureInstance } from '@/types/furniture'

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
  const moveFurniture = useFurnitureStore((state) => state.moveFurniture)

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

  return (
    <group name="furniture-manager">
      {furnitureList.map((furniture) => {
        const item = (
          <FurnitureItem
            key={furniture.id}
            id={furniture.id}
            type={furniture.type}
            position={draggable ? [0, 0, 0] : furniture.position}
            rotation={furniture.rotation}
            color={furniture.color}
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
            >
              {item}
            </DraggableObject>
          )
        }

        return item
      })}
    </group>
  )
}

/**
 * Hook to add furniture at a position
 */
export function useAddFurniture() {
  return useFurnitureStore((state) => state.addFurniture)
}

/**
 * Hook to remove furniture
 */
export function useRemoveFurniture() {
  return useFurnitureStore((state) => state.removeFurniture)
}
