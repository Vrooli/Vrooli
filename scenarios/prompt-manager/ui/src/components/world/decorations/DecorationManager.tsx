/**
 * DecorationManager - Renders all decorations in the scene.
 * Connects to the decoration store and renders DecorationItem components.
 */

import { useCallback } from 'react'
import { DecorationItem } from './DecorationItem'
import { DraggableObject } from '../interaction'
import { useDecorationStore, useDecorationList } from '@/stores/decorationStore'
import type { DecorationInstance } from '@/types/decoration'

interface DecorationManagerProps {
  /** Called when decoration is clicked */
  onDecorationClick?: (decoration: DecorationInstance) => void
  /** Whether decorations are interactive (clickable) */
  interactive?: boolean
  /** Whether decorations are draggable */
  draggable?: boolean
}

/**
 * Manages rendering of all decoration instances in the world.
 */
export function DecorationManager({
  onDecorationClick,
  interactive = true,
  draggable = false,
}: DecorationManagerProps) {
  const decorationList = useDecorationList()
  const moveDecoration = useDecorationStore((state) => state.moveDecoration)

  const handleClick = useCallback(
    (decoration: DecorationInstance) => {
      onDecorationClick?.(decoration)
    },
    [onDecorationClick]
  )

  const handlePositionChange = useCallback(
    (decorationId: string, newPosition: [number, number, number]) => {
      moveDecoration(decorationId, newPosition)
    },
    [moveDecoration]
  )

  return (
    <group name="decoration-manager">
      {decorationList.map((decoration) => {
        const item = (
          <DecorationItem
            key={decoration.id}
            id={decoration.id}
            type={decoration.type}
            position={draggable ? [0, 0, 0] : decoration.position}
            rotation={decoration.rotation}
            scale={decoration.scale}
            color={decoration.color}
            lightOn={decoration.lightOn}
            onClick={interactive ? () => handleClick(decoration) : undefined}
          />
        )

        if (draggable) {
          return (
            <DraggableObject
              key={decoration.id}
              objectId={decoration.id}
              position={decoration.position}
              onPositionChange={(pos) => handlePositionChange(decoration.id, pos)}
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
 * Hook to add decoration at a position
 */
