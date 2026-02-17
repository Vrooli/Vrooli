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
  /** Uniform scale multiplier for all overlays */
  overlayScale?: number
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
  overlayScale = 1,
}: AgentOverlayGroupProps) {
  // Subscribe to a single agent slice to avoid rerendering all overlays on any status change.
  const agentAccessoryState = useAccessoryStore((state) => state.agentAccessories[agentId])

  // AI_CHECK: R3F_AGENT_STATUS_SUBSCRIPTION=1 | LAST: 2026-02-17
  const { effectiveStatus, statusMessage } = useMemo(() => {
    if (status) return { effectiveStatus: status, statusMessage: undefined }
    return {
      effectiveStatus: agentAccessoryState?.status?.type ?? ('normal' as AgentStatusType),
      statusMessage: agentAccessoryState?.status?.message,
    }
  }, [status, agentAccessoryState])

  // Early return for disabled or invalid agentId
  if (!enabled || !agentId) {
    return null
  }

  return (
    <group scale={[overlayScale, overlayScale, overlayScale]}>
      {/* Name tag - lowest layer */}
      <WorldErrorBoundary componentName="NameTag" minimal>
        <NameTag
          agentId={agentId}
          name={name}
          position={position}
          isHovered={isHovered}
          yOffset={1.4}
        />
      </WorldErrorBoundary>

      {/* Status icon - above name tag */}
      <WorldErrorBoundary componentName="StatusIcon" minimal>
        <StatusIcon
          status={effectiveStatus}
          position={position}
          yOffset={1.7}
          message={statusMessage}
        />
      </WorldErrorBoundary>

      {/* Thinking bubble - above status */}
      <WorldErrorBoundary componentName="ThinkingBubble" minimal>
        <ThinkingBubble
          agentId={agentId}
          position={position}
          yOffset={1.9}
        />
      </WorldErrorBoundary>

      {/* Speech bubble - topmost layer */}
      <WorldErrorBoundary componentName="SpeechBubble" minimal>
        <SpeechBubble
          agentId={agentId}
          position={position}
          yOffset={2.1}
        />
      </WorldErrorBoundary>
    </group>
  )
}
