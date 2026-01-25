/**
 * MemberOverlayGroup - Combines all overlays for a member.
 * Manages the composition and layering of name tags, status icons, etc.
 */

import { NameTag } from './NameTag'
import { StatusIcon } from './StatusIcon'
import { ThinkingBubble } from './ThinkingBubble'
import { SpeechBubble } from './SpeechBubble'
import { WorldErrorBoundary } from '../WorldErrorBoundary'
import { useAccessoryStore } from '@/stores/accessoryStore'
import type { MemberStatusType } from '@/types/accessory'

interface MemberOverlayGroupProps {
  /** Member ID */
  memberId: string
  /** Member display name */
  name: string
  /** Base position of the member */
  position: [number, number, number]
  /** Whether the member is currently hovered */
  isHovered?: boolean
  /** Override status (if not using store) */
  status?: MemberStatusType
  /** Whether to show overlays at all */
  enabled?: boolean
}

/**
 * Renders all overlays for a member in the correct order.
 * Overlays are stacked vertically with appropriate spacing.
 */
export function MemberOverlayGroup({
  memberId,
  name,
  position,
  isHovered = false,
  status,
  enabled = true,
}: MemberOverlayGroupProps) {
  const storedStatus = useAccessoryStore((state) => state.getMemberStatus(memberId))

  // Early return for disabled or invalid memberId
  if (!enabled || !memberId) {
    return null
  }

  const effectiveStatus = status ?? storedStatus?.type ?? 'normal'

  return (
    <>
      {/* Name tag - lowest layer */}
      <WorldErrorBoundary componentName="NameTag" minimal>
        <NameTag
          memberId={memberId}
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
        />
      </WorldErrorBoundary>

      {/* Thinking bubble - above status */}
      <WorldErrorBoundary componentName="ThinkingBubble" minimal>
        <ThinkingBubble
          memberId={memberId}
          position={position}
          yOffset={1.5}
        />
      </WorldErrorBoundary>

      {/* Speech bubble - topmost layer */}
      <WorldErrorBoundary componentName="SpeechBubble" minimal>
        <SpeechBubble
          memberId={memberId}
          position={position}
          yOffset={1.7}
        />
      </WorldErrorBoundary>
    </>
  )
}
