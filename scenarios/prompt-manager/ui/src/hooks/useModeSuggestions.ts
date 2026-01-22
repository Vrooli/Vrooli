/**
 * useModeSuggestions - Mode autocomplete suggestions for CategoryPathEditor.
 *
 * Provides context-aware suggestions based on existing prompt modes.
 */

import { useCallback, useMemo } from 'react'
import type { Prompt } from '@/types'
import { getModesAtLevel } from '@/services/treeService'

interface UseModeSuggestionsProps {
  prompts: Prompt[]
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
   * Check if a mode path would be new (not existing in any prompt).
   *
   * @param modes - The full mode path to check
   * @returns True if this would create a new category
   */
  isNewPath: (modes: string[]) => boolean
}

/**
 * Hook for getting mode suggestions based on existing prompts.
 */
export function useModeSuggestions({ prompts }: UseModeSuggestionsProps): UseModeSuggestionsReturn {
  // Memoize top-level modes
  const topLevelModes = useMemo(() => getModesAtLevel(prompts, 0, []), [prompts])

  // Get suggestions for a specific level
  const getSuggestionsAtLevel = useCallback(
    (level: number, parentPath: string[]): string[] => {
      return getModesAtLevel(prompts, level, parentPath)
    },
    [prompts]
  )

  // Check if a mode path is new
  const isNewPath = useCallback(
    (modes: string[]): boolean => {
      if (modes.length === 0) return false

      // Check if any prompt has exactly this mode path
      return !prompts.some((p) => {
        if (p.modes.length !== modes.length) return false
        return modes.every((m, i) => p.modes[i] === m)
      })
    },
    [prompts]
  )

  return {
    getSuggestionsAtLevel,
    topLevelModes,
    isNewPath,
  }
}
