/**
 * AgentOverlayGroup - Combines all overlays for an agent.
 * Manages the composition and layering of name tags, status icons, etc.
 */

import { useMemo } from 'react'
import { NameTag } from './NameTag'
import { StatusIcon } from './StatusIcon'
import { ThinkingBubble } from './ThinkingBubble'
import { SpeechBubble } from './SpeechBubble'
import { WorldErrorBoundary } from '../WorldErrorBoundary'
import { useAccessoryStore } from '@/stores/accessoryStore'
import type { AgentStatusType } from '@/types/accessory'

interface AgentOverlayGroupProps {
  /** Agent ID */
  agentId: string
  /** Agent display name */
  name: string
  /** Base position of the agent */
  position: [number, number, number]
  /** Whether the agent is currently hovered */
  isHovered?: boolean
  /** Override status (if not using store) */
  status?: AgentStatusType
  /** Whether to show overlays at all */
  enabled?: boolean
}

/**
 * Renders all overlays for an agent in the correct order.
 * Overlays are stacked vertically with appropriate spacing.
 */
export function AgentOverlayGroup({
  agentId,
  name,
  position,
  isHovered = false,
  status,
  enabled = true,
}: AgentOverlayGroupProps) {
  // Get stable reference to agent accessories state
  const agentAccessories = useAccessoryStore((state) => state.agentAccessories)

  // Memoize status lookup to avoid creating new objects
  const { effectiveStatus, statusMessage } = useMemo(() => {
    if (status) return { effectiveStatus: status, statusMessage: undefined }
    const agentState = agentAccessories[agentId]
    return {
      effectiveStatus: agentState?.status?.type ?? ('normal' as AgentStatusType),
      statusMessage: agentState?.status?.message,
    }
  }, [status, agentAccessories, agentId])

  // Early return for disabled or invalid agentId
  if (!enabled || !agentId) {
    return null
  }

  return (
    <>
      {/* Name tag - lowest layer */}
      <WorldErrorBoundary componentName="NameTag" minimal>
        <NameTag
          agentId={agentId}
          name={name}
          position={position}
          isHovered={isHovered}
          yOffset={1.0}
        />
      </WorldErrorBoundary>

      {/* Status icon - above name tag */}
      <WorldErrorBoundary componentName="StatusIcon" minimal>
        <StatusIcon
          status={effectiveStatus}
          position={position}
          yOffset={1.3}
          message={statusMessage}
        />
      </WorldErrorBoundary>

      {/* Thinking bubble - above status */}
      <WorldErrorBoundary componentName="ThinkingBubble" minimal>
        <ThinkingBubble
          agentId={agentId}
          position={position}
          yOffset={1.5}
        />
      </WorldErrorBoundary>

      {/* Speech bubble - topmost layer */}
      <WorldErrorBoundary componentName="SpeechBubble" minimal>
        <SpeechBubble
          agentId={agentId}
          position={position}
          yOffset={1.7}
        />
      </WorldErrorBoundary>
    </>
  )
}
