/**
 * MemberOverlayGroup - Combines all overlays for a member.
 * Manages the composition and layering of name tags, status icons, etc.
 */

import { NameTag } from './NameTag'
import { StatusIcon } from './StatusIcon'
import { ThinkingBubble } from './ThinkingBubble'
import { SpeechBubble } from './SpeechBubble'
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

  if (!enabled) {
    return null
  }

  const effectiveStatus = status ?? storedStatus?.type ?? 'normal'

  return (
    <>
      {/* Name tag - lowest layer */}
      <NameTag
        memberId={memberId}
        name={name}
        position={position}
        isHovered={isHovered}
        yOffset={1.0}
      />

      {/* Status icon - above name tag */}
      <StatusIcon
        status={effectiveStatus}
        position={position}
        yOffset={1.3}
      />

      {/* Thinking bubble - above status */}
      <ThinkingBubble
        memberId={memberId}
        position={position}
        yOffset={1.5}
      />

      {/* Speech bubble - topmost layer */}
      <SpeechBubble
        memberId={memberId}
        position={position}
        yOffset={1.7}
      />
    </>
  )
}
