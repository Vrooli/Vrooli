/**
 * AgentWithAccessories - Wrapper component that combines an agent with accessories and overlays.
 * Provides a complete agent representation with all visual enhancements.
 */

import { useMemo } from 'react'
import { GeometricAgent } from './GeometricAgent'
import { BackpackAccessory } from '../accessories/BackpackAccessory'
import { HeadAccessory } from '../accessories/HeadAccessory'
import { HeldItemAccessory } from '../accessories/HeldItemAccessory'
import { ClothingTop, ClothingBottom, FootwearAccessory } from '../accessories/ClothingAccessory'
import { AgentOverlayGroup } from '../overlays/AgentOverlayGroup'
import { HoverGlow } from '../effects'
import { useAccessoryStore } from '@/stores/accessoryStore'
import { useHoverHighlight } from '@/hooks/useHoverHighlight'
import type { AgentProps } from '@/types/world'
import type { Agent } from '@/types/agent'
import { DEFAULT_AGENT_COLORS } from '@/types/agent'

interface AgentWithAccessoriesProps extends Omit<AgentProps, 'agentId'> {
  /** Full agent data */
  agent: Agent
  /** Enable overlays (name tags, status icons) */
  showOverlays?: boolean
  /** Enable accessories rendering */
  showAccessories?: boolean
  /** Enable hover highlighting */
  enableHover?: boolean
  /** Whether agent is seated on furniture */
  isSeated?: boolean
  /** Rotation when seated (radians) */
  seatRotation?: number
}

/**
 * Complete agent representation with accessories and overlays.
 *
 * @example
 * ```tsx
 * <AgentWithAccessories
 *   agent={agent}
 *   position={[0, 0, 0]}
 *   cursorPosition={cursor}
 *   selectedNodes={[]}
 *   isAnimating={false}
 *   onAgentClick={() => handleClick(agent.id)}
 * />
 * ```
 */
export function AgentWithAccessories({
  agent,
  position,
  cursorPosition,
  selectedNodes: selectedNodesProp,
  isAnimating,
  onAnimationComplete,
  onAgentClick,
  colors,
  showOverlays = true,
  showAccessories = true,
  enableHover = true,
  isSeated = false,
  seatRotation = 0,
}: AgentWithAccessoriesProps) {
  // Defensive: ensure selectedNodes is always an array
  const selectedNodes = selectedNodesProp ?? []
  // Get stable references from accessory store
  const agentAccessoriesState = useAccessoryStore((state) => state.agentAccessories)
  const accessoryDefaults = useAccessoryStore((state) => state.defaults)

  // Memoize accessory resolution to avoid creating new objects on each render
  const storedAccessories = useMemo(() => {
    const agentIdLocal = agent.id
    const agentState = agentAccessoriesState[agentIdLocal]
    const defaults = accessoryDefaults
    return {
      head: agentState?.accessories.head ?? defaults.head ?? { type: 'none' as const },
      back: agentState?.accessories.back ?? defaults.back ?? { type: 'none' as const },
      held: agentState?.accessories.held ?? defaults.held ?? { type: 'none' as const },
      clothingTop: agentState?.accessories.clothingTop ?? defaults.clothingTop ?? { type: 'none' as const },
      clothingBottom: agentState?.accessories.clothingBottom ?? defaults.clothingBottom ?? { type: 'none' as const },
      footwear: agentState?.accessories.footwear ?? defaults.footwear ?? { type: 'none' as const },
    }
  }, [agentAccessoriesState, agent.id, accessoryDefaults])

  // Hover highlighting
  const { isHovered, hoverProps } = useHoverHighlight(agent.id, {
    enabled: enableHover,
  })

  // Merge provided colors with agent colors
  const agentColors = useMemo(
    () =>
      colors ?? {
        body: agent.appearance?.body ?? DEFAULT_AGENT_COLORS.body,
        head: agent.appearance?.head ?? DEFAULT_AGENT_COLORS.head,
        accent: agent.appearance?.accent ?? DEFAULT_AGENT_COLORS.accent,
      },
    [colors, agent.appearance]
  )

  return (
    <group {...(enableHover ? hoverProps : {})}>
      {/* Base agent model */}
      <GeometricAgent
        agentId={agent.id}
        position={position}
        cursorPosition={cursorPosition}
        selectedNodes={selectedNodes}
        isAnimating={isAnimating}
        onAnimationComplete={onAnimationComplete}
        onAgentClick={onAgentClick}
        colors={agentColors}
        isSeated={isSeated}
        seatRotation={seatRotation}
      />

      {/* Accessories */}
      {showAccessories && (
        <group position={position}>
          {/* Back accessory from store */}
          {storedAccessories.back.type !== 'none' && (
            <BackpackAccessory
              type={storedAccessories.back.type}
              scale={storedAccessories.back.scale}
            />
          )}

          {/* Head accessory from store */}
          {storedAccessories.head.type !== 'none' && (
            <HeadAccessory
              type={storedAccessories.head.type}
              color={storedAccessories.head.color}
            />
          )}

          {/* Held item from store */}
          {storedAccessories.held.type !== 'none' && (
            <HeldItemAccessory
              type={storedAccessories.held.type}
              hand={storedAccessories.held.hand}
              color={storedAccessories.held.color}
            />
          )}

          {/* Clothing - top */}
          {storedAccessories.clothingTop.type !== 'none' && (
            <ClothingTop
              type={storedAccessories.clothingTop.type}
              color={storedAccessories.clothingTop.color}
              accentColor={storedAccessories.clothingTop.accentColor}
            />
          )}

          {/* Clothing - bottom */}
          {storedAccessories.clothingBottom.type !== 'none' && (
            <ClothingBottom
              type={storedAccessories.clothingBottom.type}
              color={storedAccessories.clothingBottom.color}
            />
          )}

          {/* Footwear */}
          {storedAccessories.footwear.type !== 'none' && (
            <FootwearAccessory
              type={storedAccessories.footwear.type}
              color={storedAccessories.footwear.color}
            />
          )}
        </group>
      )}

      {/* Hover glow effect - uses agent's accent color */}
      <HoverGlow
        isActive={isHovered}
        position={position}
        size={0.8}
        color={agentColors.accent || '#6366f1'}
        intensity={0.6}
      />

      {/* Overlays (name tag, status, speech) */}
      {showOverlays && (
        <AgentOverlayGroup
          agentId={agent.id}
          name={agent.displayName}
          position={position}
          isHovered={isHovered}
        />
      )}
    </group>
  )
}
