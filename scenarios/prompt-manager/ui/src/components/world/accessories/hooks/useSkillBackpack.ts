/**
 * useSkillBackpack - Computes backpack type from skill count.
 * Automatically determines the appropriate accessory based on how many skills a member has.
 */

import { useMemo } from 'react'
import type { BackAccessoryType } from '@/types/accessory'
import { SKILL_BACKPACK_THRESHOLDS } from '@/types/accessory'

/**
 * Determine backpack type based on skill count.
 *
 * - 0 skills: none
 * - 1-2 skills: paper (loose sheets)
 * - 3-5 skills: folder
 * - 6-10 skills: briefcase
 * - 11+ skills: backpack
 */
function computeBackpackType(skillCount: number): BackAccessoryType {
  if (skillCount >= SKILL_BACKPACK_THRESHOLDS.backpack) return 'backpack'
  if (skillCount >= SKILL_BACKPACK_THRESHOLDS.briefcase) return 'briefcase'
  if (skillCount >= SKILL_BACKPACK_THRESHOLDS.folder) return 'folder'
  if (skillCount >= SKILL_BACKPACK_THRESHOLDS.paper) return 'paper'
  return 'none'
}

/**
 * Hook to compute backpack accessory type from skill count.
 *
 * @param skillCount - Number of skills the member has
 * @returns The appropriate backpack type
 *
 * @example
 * ```tsx
 * function MemberWithBackpack({ member }) {
 *   const backpackType = useSkillBackpack(member.skills.length)
 *
 *   return (
 *     <group>
 *       <GeometricMember {...props} />
 *       {backpackType !== 'none' && (
 *         <BackpackAccessory type={backpackType} />
 *       )}
 *     </group>
 *   )
 * }
 * ```
 */
export function useSkillBackpack(skillCount: number): BackAccessoryType {
  return useMemo(() => computeBackpackType(skillCount), [skillCount])
}

/**
 * Get description text for a backpack type
 */
export function getBackpackDescription(type: BackAccessoryType): string {
  const descriptions: Record<BackAccessoryType, string> = {
    none: 'No items',
    paper: 'A few loose sheets',
    folder: 'A neat folder',
    briefcase: 'A professional briefcase',
    backpack: 'A full backpack',
  }
  return descriptions[type]
}

/**
 * Get capacity info for display
 */
export function getBackpackCapacity(type: BackAccessoryType): {
  min: number
  max: number | null
  label: string
} {
  const capacities: Record<BackAccessoryType, { min: number; max: number | null; label: string }> = {
    none: { min: 0, max: 0, label: 'Empty' },
    paper: { min: 1, max: 2, label: '1-2 skills' },
    folder: { min: 3, max: 5, label: '3-5 skills' },
    briefcase: { min: 6, max: 10, label: '6-10 skills' },
    backpack: { min: 11, max: null, label: '11+ skills' },
  }
  return capacities[type]
}
