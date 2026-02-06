/**
 * NameTag - Billboard name label for agents.
 * Floats above the agent and always faces the camera.
 */

import { useMemo } from 'react'
import { Html } from '@react-three/drei'
import { useOverlayStore } from '@/stores/overlayStore'

interface NameTagProps {
  /** Member ID for visibility checks */
  agentId: string
  /** Display name */
  name: string
  /** Position offset from agent center */
  position: [number, number, number]
  /** Whether the agent is currently hovered */
  isHovered?: boolean
  /** Custom Y offset above the agent */
  yOffset?: number
}

/**
 * Renders a name tag that floats above a agent.
 * Visibility is controlled by the overlay store settings.
 */
export function NameTag({
  agentId,
  name,
  position,
  isHovered = false,
  yOffset = 1.2,
}: NameTagProps) {
  // Get stable references to config values
  const overlaysVisible = useOverlayStore((state) => state.overlaysVisible)
  const nameTagConfig = useOverlayStore((state) => state.nameTagConfig)

  // Compute visibility based on config and hover state
  const shouldShow = useMemo(() => {
    if (!overlaysVisible) return false
    if (nameTagConfig.neverShowFor.includes(agentId)) return false
    if (nameTagConfig.alwaysShowFor.includes(agentId)) return true
    if (nameTagConfig.showOnHover) return isHovered
    return nameTagConfig.showAll
  }, [overlaysVisible, nameTagConfig, agentId, isHovered])

  if (!shouldShow) {
    return null
  }

  return (
    <Html
      position={[position[0], position[1] + yOffset, position[2]]}
      center
      distanceFactor={10}
      occlude
      zIndexRange={[10, 0]}
      style={{
        pointerEvents: 'none',
        userSelect: 'none',
      }}
    >
      <div
        className={`
          px-2 py-1
          bg-card/90 backdrop-blur-sm
          rounded-full
          text-sm font-medium
          shadow-lg
          border border-border/50
          transition-opacity duration-200
          ${isHovered ? 'opacity-100' : 'opacity-80'}
        `}
      >
        {name}
      </div>
    </Html>
  )
}
