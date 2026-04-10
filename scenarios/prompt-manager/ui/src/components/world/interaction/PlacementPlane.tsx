/**
 * PlacementPlane - Invisible plane used to place new objects on the ground.
 */

import { useCallback, useEffect, useRef } from 'react'
import type { ThreeEvent } from '@react-three/fiber'
import { useWorldEditorStore } from '@/stores/worldEditorStore'
import { useFurnitureStore } from '@/stores/furnitureStore'
import { useDecorationStore } from '@/stores/decorationStore'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { DECORATION_CONFIGS, type DecorationType } from '@/types/decoration'
import type { FurnitureType } from '@/types/furniture'
import { applyPlacementConstraints } from '@/lib/world'

interface PlacementPlaneProps {
  /** Size of the plane (width/depth) */
  size: number
  /** Y position of the plane */
  y: number
}

const ROTATION_STEP = Math.PI / 4 // 45 degrees

export function PlacementPlane({ size, y }: PlacementPlaneProps) {
  const placingObject = useWorldEditorStore((state) => state.placingObject)
  const confirmPlacement = useWorldEditorStore((state) => state.confirmPlacement)
  const cancelPlacing = useWorldEditorStore((state) => state.cancelPlacing)

  const addFurniture = useFurnitureStore((state) => state.addFurniture)
  const addDecoration = useDecorationStore((state) => state.addDecoration)

  const placementConfig = useEnvironmentStore((state) => state.current.placement)
  const boundaryConfig = useEnvironmentStore((state) => state.current.boundary)
  const groundSize = useEnvironmentStore((state) => state.current.ground.size)

  const placementRotation = useRef(0)

  const handlePointerDown = useCallback(
    (event: ThreeEvent<PointerEvent>) => {
      if (!placingObject) return
      event.stopPropagation()

      const rawPosition: [number, number, number] = [event.point.x, y, event.point.z]
      const finalPosition = applyPlacementConstraints(rawPosition, {
        placement: placementConfig,
        boundary: boundaryConfig,
        groundSize,
      })

      if (placingObject.type === 'furniture') {
        addFurniture(placingObject.subtype as FurnitureType, finalPosition, placementRotation.current)
      } else if (placingObject.type === 'decoration') {
        const type = placingObject.subtype as DecorationType
        const config = DECORATION_CONFIGS[type]
        addDecoration(type, [finalPosition[0], config.defaultY, finalPosition[2]])
      } else {
        cancelPlacing()
        return
      }

      placementRotation.current = 0
      confirmPlacement(finalPosition)
    },
    [
      placingObject,
      y,
      placementConfig,
      boundaryConfig,
      groundSize,
      addFurniture,
      addDecoration,
      cancelPlacing,
      confirmPlacement,
    ]
  )

  useEffect(() => {
    if (!placingObject) return

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        placementRotation.current = 0
        cancelPlacing()
      } else if (event.key === 'r' || event.key === 'R') {
        const delta = event.shiftKey ? -ROTATION_STEP : ROTATION_STEP
        const TAU = Math.PI * 2
        placementRotation.current = ((placementRotation.current + delta) % TAU + TAU) % TAU
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => {
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [placingObject, cancelPlacing])

  if (!placingObject) {
    return null
  }

  return (
    <mesh
      position={[0, y, 0]}
      rotation={[-Math.PI / 2, 0, 0]}
      onPointerDown={handlePointerDown}
    >
      <planeGeometry args={[size, size]} />
      <meshBasicMaterial transparent opacity={0} depthWrite={false} />
    </mesh>
  )
}
