/**
 * DecorationManager - Renders all decorations in the scene.
 * Connects to the decoration store and renders DecorationItem components.
 */

import { memo, useCallback, useMemo } from 'react'
import { DecorationItem } from './DecorationItem'
import { DraggableObject } from '../interaction'
import { useDecorationStore, useDecorationList } from '@/stores/decorationStore'
import { useWorldScaleStore } from '@/stores/worldScaleStore'
import { useEnvironmentStore } from '@/stores/environmentStore'
import type { DecorationInstance } from '@/types/decoration'
import { DECORATION_CONFIGS } from '@/types/decoration'
import { applyPlacementConstraints } from '@/lib/world'

interface DecorationManagerProps {
  /** Called when decoration is clicked */
  onDecorationClick?: (decoration: DecorationInstance) => void
  /** Whether decorations are interactive (clickable) */
  interactive?: boolean
  /** Whether decorations are draggable */
  draggable?: boolean
}

const LOCAL_ORIGIN: [number, number, number] = [0, 0, 0]

interface DecorationNodeProps {
  decoration: DecorationInstance
  decorationScale: number
  draggable: boolean
  interactive: boolean
  isNightTime: boolean
  constrainPosition: (position: [number, number, number]) => [number, number, number]
  onDecorationClick?: (decoration: DecorationInstance) => void
  onPositionChange: (decorationId: string, newPosition: [number, number, number]) => void
}

const DecorationNode = memo(function DecorationNode({
  decoration,
  decorationScale,
  draggable,
  interactive,
  isNightTime,
  constrainPosition,
  onDecorationClick,
  onPositionChange,
}: DecorationNodeProps) {
  const handleClick = useCallback(() => {
    onDecorationClick?.(decoration)
  }, [onDecorationClick, decoration])

  // Resolve effective light state from mode + time of day
  const config = DECORATION_CONFIGS[decoration.type]
  let effectiveLightOn: boolean | undefined
  if (config.emitsLight) {
    const mode = decoration.lightMode ?? 'auto'
    effectiveLightOn = mode === 'on' ? true : mode === 'off' ? false : isNightTime
  }

  const item = (
    <DecorationItem
      id={decoration.id}
      type={decoration.type}
      position={draggable ? LOCAL_ORIGIN : decoration.position}
      rotation={decoration.rotation}
      scale={(decoration.scale ?? 1) * decorationScale}
      color={decoration.color}
      lightOn={effectiveLightOn}
      onClick={interactive ? handleClick : undefined}
    />
  )

  if (draggable) {
    return (
      <DraggableObject
        objectId={decoration.id}
        position={decoration.position}
        onPositionChange={(pos) => onPositionChange(decoration.id, pos)}
        constrainPosition={constrainPosition}
      >
        {item}
      </DraggableObject>
    )
  }

  return item
})

/**
 * Manages rendering of all decoration instances in the world.
 */
export function DecorationManager({
  onDecorationClick,
  interactive = true,
  draggable = false,
}: DecorationManagerProps) {
  const decorationList = useDecorationList()
  const decorationScale = useWorldScaleStore((state) => state.decoration)
  const moveDecoration = useDecorationStore((state) => state.moveDecoration)
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
      {decorationList.map((decoration) => (
        <DecorationNode
          key={decoration.id}
          decoration={decoration}
          decorationScale={decorationScale}
          draggable={draggable}
          interactive={interactive}
          isNightTime={isNightTime}
          constrainPosition={constrainPosition}
          onDecorationClick={handleClick}
          onPositionChange={handlePositionChange}
        />
      ))}
    </group>
  )
}

/**
 * Hook to add decoration at a position
 */
