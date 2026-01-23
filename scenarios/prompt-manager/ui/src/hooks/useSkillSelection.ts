/**
 * Hook for managing multi-select state in the skill tree.
 */

import { useCallback, useState } from 'react'
import type { SelectionMode, SelectionState } from '@/types/world'

const INITIAL_STATE: SelectionState = {
  selectedIds: [],
  mode: 'single',
  anchorId: null,
}

interface UseSkillSelectionOptions {
  maxSelection?: number
  onSelectionChange?: (selectedIds: string[]) => void
}

export function useSkillSelection(options: UseSkillSelectionOptions = {}) {
  const { maxSelection = Infinity, onSelectionChange } = options
  const [state, setState] = useState<SelectionState>(INITIAL_STATE)

  /**
   * Select a single skill, clearing previous selection.
   */
  const selectSingle = useCallback(
    (skillId: string) => {
      setState({
        selectedIds: [skillId],
        mode: 'single',
        anchorId: skillId,
      })
      onSelectionChange?.([skillId])
    },
    [onSelectionChange]
  )

  /**
   * Toggle a skill's selection (for Cmd/Ctrl+click).
   */
  const toggleSelection = useCallback(
    (skillId: string) => {
      setState((prev) => {
        const isSelected = prev.selectedIds.includes(skillId)
        let newSelectedIds: string[]

        if (isSelected) {
          newSelectedIds = prev.selectedIds.filter((id) => id !== skillId)
        } else if (prev.selectedIds.length < maxSelection) {
          newSelectedIds = [...prev.selectedIds, skillId]
        } else {
          return prev
        }

        onSelectionChange?.(newSelectedIds)
        return {
          selectedIds: newSelectedIds,
          mode: 'toggle' as SelectionMode,
          anchorId: isSelected ? prev.anchorId : skillId,
        }
      })
    },
    [maxSelection, onSelectionChange]
  )

  /**
   * Add to selection (for Shift+click range selection).
   */
  const addToSelection = useCallback(
    (skillId: string) => {
      setState((prev) => {
        if (prev.selectedIds.includes(skillId)) {
          return prev
        }

        if (prev.selectedIds.length >= maxSelection) {
          return prev
        }

        const newSelectedIds = [...prev.selectedIds, skillId]
        onSelectionChange?.(newSelectedIds)

        return {
          ...prev,
          selectedIds: newSelectedIds,
          mode: 'multi' as SelectionMode,
        }
      })
    },
    [maxSelection, onSelectionChange]
  )

  /**
   * Remove from selection.
   */
  const removeFromSelection = useCallback(
    (skillId: string) => {
      setState((prev) => {
        const newSelectedIds = prev.selectedIds.filter((id) => id !== skillId)
        onSelectionChange?.(newSelectedIds)

        return {
          ...prev,
          selectedIds: newSelectedIds,
        }
      })
    },
    [onSelectionChange]
  )

  /**
   * Clear all selections.
   */
  const clearSelection = useCallback(() => {
    setState(INITIAL_STATE)
    onSelectionChange?.([])
  }, [onSelectionChange])

  /**
   * Select all provided IDs.
   */
  const selectAll = useCallback(
    (skillIds: string[]) => {
      const limitedIds = skillIds.slice(0, maxSelection)
      setState({
        selectedIds: limitedIds,
        mode: 'multi',
        anchorId: limitedIds[0] || null,
      })
      onSelectionChange?.(limitedIds)
    },
    [maxSelection, onSelectionChange]
  )

  /**
   * Handle click with modifiers.
   */
  const handleClick = useCallback(
    (skillId: string, event: { shiftKey: boolean; metaKey: boolean; ctrlKey: boolean }) => {
      if (event.metaKey || event.ctrlKey) {
        toggleSelection(skillId)
      } else if (event.shiftKey && state.anchorId) {
        addToSelection(skillId)
      } else {
        selectSingle(skillId)
      }
    },
    [toggleSelection, addToSelection, selectSingle, state.anchorId]
  )

  /**
   * Check if a skill is selected.
   */
  const isSelected = useCallback(
    (skillId: string) => state.selectedIds.includes(skillId),
    [state.selectedIds]
  )

  return {
    selectedIds: state.selectedIds,
    selectionCount: state.selectedIds.length,
    mode: state.mode,
    anchorId: state.anchorId,
    selectSingle,
    toggleSelection,
    addToSelection,
    removeFromSelection,
    clearSelection,
    selectAll,
    handleClick,
    isSelected,
    hasSelection: state.selectedIds.length > 0,
    hasMultipleSelected: state.selectedIds.length > 1,
  }
}
