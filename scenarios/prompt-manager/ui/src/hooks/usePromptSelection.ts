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

interface UsePromptSelectionOptions {
  maxSelection?: number
  onSelectionChange?: (selectedIds: string[]) => void
}

export function usePromptSelection(options: UsePromptSelectionOptions = {}) {
  const { maxSelection = Infinity, onSelectionChange } = options
  const [state, setState] = useState<SelectionState>(INITIAL_STATE)

  /**
   * Select a single prompt, clearing previous selection.
   */
  const selectSingle = useCallback(
    (promptId: string) => {
      setState({
        selectedIds: [promptId],
        mode: 'single',
        anchorId: promptId,
      })
      onSelectionChange?.([promptId])
    },
    [onSelectionChange]
  )

  /**
   * Toggle a prompt's selection (for Cmd/Ctrl+click).
   */
  const toggleSelection = useCallback(
    (promptId: string) => {
      setState((prev) => {
        const isSelected = prev.selectedIds.includes(promptId)
        let newSelectedIds: string[]

        if (isSelected) {
          newSelectedIds = prev.selectedIds.filter((id) => id !== promptId)
        } else if (prev.selectedIds.length < maxSelection) {
          newSelectedIds = [...prev.selectedIds, promptId]
        } else {
          return prev
        }

        onSelectionChange?.(newSelectedIds)
        return {
          selectedIds: newSelectedIds,
          mode: 'toggle' as SelectionMode,
          anchorId: isSelected ? prev.anchorId : promptId,
        }
      })
    },
    [maxSelection, onSelectionChange]
  )

  /**
   * Add to selection (for Shift+click range selection).
   */
  const addToSelection = useCallback(
    (promptId: string) => {
      setState((prev) => {
        if (prev.selectedIds.includes(promptId)) {
          return prev
        }

        if (prev.selectedIds.length >= maxSelection) {
          return prev
        }

        const newSelectedIds = [...prev.selectedIds, promptId]
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
    (promptId: string) => {
      setState((prev) => {
        const newSelectedIds = prev.selectedIds.filter((id) => id !== promptId)
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
    (promptIds: string[]) => {
      const limitedIds = promptIds.slice(0, maxSelection)
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
    (promptId: string, event: { shiftKey: boolean; metaKey: boolean; ctrlKey: boolean }) => {
      if (event.metaKey || event.ctrlKey) {
        toggleSelection(promptId)
      } else if (event.shiftKey && state.anchorId) {
        addToSelection(promptId)
      } else {
        selectSingle(promptId)
      }
    },
    [toggleSelection, addToSelection, selectSingle, state.anchorId]
  )

  /**
   * Check if a prompt is selected.
   */
  const isSelected = useCallback(
    (promptId: string) => state.selectedIds.includes(promptId),
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
