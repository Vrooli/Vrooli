/**
 * MemberWithAccessories - Wrapper component that combines a member with accessories and overlays.
 * Provides a complete member representation with all visual enhancements.
 */

import { useMemo } from 'react'
import { GeometricMember } from './GeometricMember'
import { BackpackAccessory } from '../accessories/BackpackAccessory'
import { HeadAccessory } from '../accessories/HeadAccessory'
import { HeldItemAccessory } from '../accessories/HeldItemAccessory'
import { MemberOverlayGroup } from '../overlays/MemberOverlayGroup'
import { useSkillBackpack } from '../accessories/hooks/useSkillBackpack'
import { useAccessoryStore } from '@/stores/accessoryStore'
import { useHoverHighlight } from '@/hooks/useHoverHighlight'
import type { MemberProps } from '@/types/world'
import type { Member } from '@/types/member'

interface MemberWithAccessoriesProps extends Omit<MemberProps, 'memberId'> {
  /** Full member data */
  member: Member
  /** Enable overlays (name tags, status icons) */
  showOverlays?: boolean
  /** Enable accessories rendering */
  showAccessories?: boolean
  /** Enable hover highlighting */
  enableHover?: boolean
}

/**
 * Complete member representation with accessories and overlays.
 *
 * @example
 * ```tsx
 * <MemberWithAccessories
 *   member={member}
 *   position={[0, 0, 0]}
 *   cursorPosition={cursor}
 *   selectedNodes={[]}
 *   isAnimating={false}
 *   onMemberClick={() => handleClick(member.id)}
 * />
 * ```
 */
export function MemberWithAccessories({
  member,
  position,
  cursorPosition,
  selectedNodes,
  isAnimating,
  onAnimationComplete,
  onMemberClick,
  colors,
  showOverlays = true,
  showAccessories = true,
  enableHover = true,
}: MemberWithAccessoriesProps) {
  // Compute backpack type from skill count
  const backpackType = useSkillBackpack(member.skills.length)

  // Get stored accessories for this member
  const storedAccessories = useAccessoryStore((state) =>
    state.getMemberAccessories(member.id)
  )

  // Hover highlighting
  const { isHovered, hoverProps } = useHoverHighlight(member.id, {
    enabled: enableHover,
  })

  // Merge provided colors with member colors
  const memberColors = useMemo(
    () =>
      colors ?? {
        body: member.bodyColor,
        head: member.headColor,
        accent: member.accentColor,
      },
    [colors, member.bodyColor, member.headColor, member.accentColor]
  )

  return (
    <group {...(enableHover ? hoverProps : {})}>
      {/* Base member model */}
      <GeometricMember
        memberId={member.id}
        position={position}
        cursorPosition={cursorPosition}
        selectedNodes={selectedNodes}
        isAnimating={isAnimating}
        onAnimationComplete={onAnimationComplete}
        onMemberClick={onMemberClick}
        colors={memberColors}
      />

      {/* Accessories */}
      {showAccessories && (
        <group position={position}>
          {/* Auto-computed backpack based on skills */}
          <BackpackAccessory
            type={backpackType}
            skillCount={member.skills.length}
          />

          {/* Head accessory from store */}
          {storedAccessories.head && storedAccessories.head.type !== 'none' && (
            <HeadAccessory
              type={storedAccessories.head.type}
              color={storedAccessories.head.color}
            />
          )}

          {/* Held item from store */}
          {storedAccessories.held && storedAccessories.held.type !== 'none' && (
            <HeldItemAccessory
              type={storedAccessories.held.type}
              hand={storedAccessories.held.hand}
              color={storedAccessories.held.color}
            />
          )}
        </group>
      )}

      {/* Hover glow effect */}
      {isHovered && (
        <mesh position={position}>
          <sphereGeometry args={[0.6, 16, 16]} />
          <meshBasicMaterial color="#ffffff" transparent opacity={0.1} />
        </mesh>
      )}

      {/* Overlays (name tag, status, speech) */}
      {showOverlays && (
        <MemberOverlayGroup
          memberId={member.id}
          name={member.name}
          position={position}
          isHovered={isHovered}
        />
      )}
    </group>
  )
}
