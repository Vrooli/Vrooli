/**
 * MemberWithAccessories - Wrapper component that combines a member with accessories and overlays.
 * Provides a complete member representation with all visual enhancements.
 */

import { useMemo } from 'react'
import { GeometricMember } from './GeometricMember'
import { BackpackAccessory } from '../accessories/BackpackAccessory'
import { HeadAccessory } from '../accessories/HeadAccessory'
import { HeldItemAccessory } from '../accessories/HeldItemAccessory'
import { ClothingTop, ClothingBottom, FootwearAccessory } from '../accessories/ClothingAccessory'
import { MemberOverlayGroup } from '../overlays/MemberOverlayGroup'
import { HoverGlow } from '../effects'
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
  /** Whether member is seated on furniture */
  isSeated?: boolean
  /** Rotation when seated (radians) */
  seatRotation?: number
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
  selectedNodes: selectedNodesProp,
  isAnimating,
  onAnimationComplete,
  onMemberClick,
  colors,
  showOverlays = true,
  showAccessories = true,
  enableHover = true,
  isSeated = false,
  seatRotation = 0,
}: MemberWithAccessoriesProps) {
  // Defensive: ensure selectedNodes is always an array
  const selectedNodes = selectedNodesProp ?? []
  // Defensive: ensure skills is an array before accessing length
  const skillCount = Array.isArray(member.skills) ? member.skills.length : 0
  // Compute backpack type from skill count
  const backpackType = useSkillBackpack(skillCount)

  // Get stable references from accessory store
  const memberAccessoriesState = useAccessoryStore((state) => state.memberAccessories)
  const accessoryDefaults = useAccessoryStore((state) => state.defaults)

  // Memoize accessory resolution to avoid creating new objects on each render
  const storedAccessories = useMemo(() => {
    const memberId = member.id
    const memberState = memberAccessoriesState[memberId]
    const defaults = accessoryDefaults
    return {
      head: memberState?.accessories.head ?? defaults.head ?? { type: 'none' as const },
      back: memberState?.accessories.back ?? defaults.back ?? { type: 'none' as const },
      held: memberState?.accessories.held ?? defaults.held ?? { type: 'none' as const },
      clothingTop: memberState?.accessories.clothingTop ?? defaults.clothingTop ?? { type: 'none' as const },
      clothingBottom: memberState?.accessories.clothingBottom ?? defaults.clothingBottom ?? { type: 'none' as const },
      footwear: memberState?.accessories.footwear ?? defaults.footwear ?? { type: 'none' as const },
    }
  }, [memberAccessoriesState, member.id, accessoryDefaults])

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
        isSeated={isSeated}
        seatRotation={seatRotation}
      />

      {/* Accessories */}
      {showAccessories && (
        <group position={position}>
          {/* Auto-computed backpack based on skills */}
          <BackpackAccessory
            type={backpackType}
            skillCount={skillCount}
          />

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

      {/* Hover glow effect - uses member's accent color */}
      <HoverGlow
        isActive={isHovered}
        position={position}
        size={0.8}
        color={memberColors.accent || '#6366f1'}
        intensity={0.6}
      />

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
