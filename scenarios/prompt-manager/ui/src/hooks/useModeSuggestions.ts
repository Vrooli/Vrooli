/**
 * useModeSuggestions - Mode autocomplete suggestions for CategoryPathEditor.
 *
 * Provides context-aware suggestions based on existing skill modes.
 */

import { useCallback, useMemo } from 'react'
import type { Skill } from '@/types'
import { getModesAtLevel } from '@/services/treeService'

interface UseModeSuggestionsProps {
  skills: Skill[]
}

interface UseModeSuggestionsReturn {
  /**
   * Get mode suggestions for a specific level in the hierarchy.
   *
   * @param level - Zero-based level (0 = top level)
   * @param parentPath - Modes at levels above this one
   * @returns Array of suggested mode values
   */
  getSuggestionsAtLevel: (level: number, parentPath: string[]) => string[]

  /**
   * Get all unique top-level modes.
   */
  topLevelModes: string[]

  /**
   * Check if a mode path would be new (not existing in any skill).
   *
   * @param modes - The full mode path to check
   * @returns True if this would create a new category
   */
  isNewPath: (modes: string[]) => boolean
}

/**
 * Hook for getting mode suggestions based on existing skills.
 */
export function useModeSuggestions({ skills }: UseModeSuggestionsProps): UseModeSuggestionsReturn {
  // Memoize top-level modes
  const topLevelModes = useMemo(() => getModesAtLevel(skills, 0, []), [skills])

  // Get suggestions for a specific level
  const getSuggestionsAtLevel = useCallback(
    (level: number, parentPath: string[]): string[] => {
      return getModesAtLevel(skills, level, parentPath)
    },
    [skills]
  )

  // Check if a mode path is new
  const isNewPath = useCallback(
    (modes: string[]): boolean => {
      if (modes.length === 0) return false

      // Check if any skill has exactly this mode path
      return !skills.some((p) => {
        if (p.modes.length !== modes.length) return false
        return modes.every((m, i) => p.modes[i] === m)
      })
    },
    [skills]
  )

  return {
    getSuggestionsAtLevel,
    topLevelModes,
    isNewPath,
  }
}
