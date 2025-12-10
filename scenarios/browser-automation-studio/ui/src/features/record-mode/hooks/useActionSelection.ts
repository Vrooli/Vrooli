/**
 * useActionSelection Hook
 *
 * Manages selection state for recorded actions, supporting:
 * - Individual selection (click)
 * - Range selection (shift+click)
 * - Multi-range selection (ctrl+click to start new range)
 * - Select all / select none
 * - Invert selection
 */

import { useState, useCallback, useMemo } from 'react';

interface UseActionSelectionOptions {
  /** Total number of actions available for selection */
  actionCount: number;
}

interface UseActionSelectionReturn {
  /** Set of selected indices */
  selectedIndices: Set<number>;
  /** Array of selected indices (sorted) for easier iteration */
  selectedIndicesArray: number[];
  /** Whether selection mode is active */
  isSelectionMode: boolean;
  /** Last clicked index (for shift-click range selection) */
  lastClickedIndex: number | null;
  /** Enter selection mode */
  enterSelectionMode: () => void;
  /** Exit selection mode and clear selection */
  exitSelectionMode: () => void;
  /** Toggle selection mode */
  toggleSelectionMode: () => void;
  /** Handle click on an action (with modifier key support) */
  handleActionClick: (index: number, shiftKey: boolean, ctrlKey: boolean) => void;
  /** Toggle selection for a single action */
  toggleSelection: (index: number) => void;
  /** Select all actions */
  selectAll: () => void;
  /** Clear all selections */
  selectNone: () => void;
  /** Invert current selection */
  invertSelection: () => void;
  /** Select a range of actions (inclusive) */
  selectRange: (start: number, end: number) => void;
  /** Check if an action is selected */
  isSelected: (index: number) => boolean;
  /** Number of selected actions */
  selectionCount: number;
  /** Whether any actions are selected */
  hasSelection: boolean;
}

export function useActionSelection({ actionCount }: UseActionSelectionOptions): UseActionSelectionReturn {
  const [selectedIndices, setSelectedIndices] = useState<Set<number>>(new Set());
  const [isSelectionMode, setIsSelectionMode] = useState(false);
  const [lastClickedIndex, setLastClickedIndex] = useState<number | null>(null);

  // Convert Set to sorted array for easier consumption
  const selectedIndicesArray = useMemo(
    () => Array.from(selectedIndices).sort((a, b) => a - b),
    [selectedIndices]
  );

  const enterSelectionMode = useCallback(() => {
    setIsSelectionMode(true);
  }, []);

  const exitSelectionMode = useCallback(() => {
    setIsSelectionMode(false);
    setSelectedIndices(new Set());
    setLastClickedIndex(null);
  }, []);

  const toggleSelectionMode = useCallback(() => {
    if (isSelectionMode) {
      exitSelectionMode();
    } else {
      enterSelectionMode();
    }
  }, [isSelectionMode, enterSelectionMode, exitSelectionMode]);

  const toggleSelection = useCallback((index: number) => {
    setSelectedIndices((prev) => {
      const next = new Set(prev);
      if (next.has(index)) {
        next.delete(index);
      } else {
        next.add(index);
      }
      return next;
    });
    setLastClickedIndex(index);
  }, []);

  const selectRange = useCallback((start: number, end: number) => {
    const [from, to] = start <= end ? [start, end] : [end, start];
    setSelectedIndices((prev) => {
      const next = new Set(prev);
      for (let i = from; i <= to; i++) {
        next.add(i);
      }
      return next;
    });
  }, []);

  const handleActionClick = useCallback(
    (index: number, shiftKey: boolean, ctrlKey: boolean) => {
      if (!isSelectionMode) {
        // Auto-enter selection mode on first click
        setIsSelectionMode(true);
      }

      if (shiftKey && lastClickedIndex !== null) {
        // Range selection: select from lastClickedIndex to current index
        selectRange(lastClickedIndex, index);
        setLastClickedIndex(index);
      } else if (ctrlKey) {
        // Toggle selection without affecting others (for building multi-ranges)
        toggleSelection(index);
      } else {
        // Simple click: toggle this item
        toggleSelection(index);
      }
    },
    [isSelectionMode, lastClickedIndex, selectRange, toggleSelection]
  );

  const selectAll = useCallback(() => {
    const all = new Set<number>();
    for (let i = 0; i < actionCount; i++) {
      all.add(i);
    }
    setSelectedIndices(all);
  }, [actionCount]);

  const selectNone = useCallback(() => {
    setSelectedIndices(new Set());
    setLastClickedIndex(null);
  }, []);

  const invertSelection = useCallback(() => {
    const inverted = new Set<number>();
    for (let i = 0; i < actionCount; i++) {
      if (!selectedIndices.has(i)) {
        inverted.add(i);
      }
    }
    setSelectedIndices(inverted);
  }, [actionCount, selectedIndices]);

  const isSelected = useCallback((index: number) => selectedIndices.has(index), [selectedIndices]);

  const selectionCount = selectedIndices.size;
  const hasSelection = selectionCount > 0;

  return {
    selectedIndices,
    selectedIndicesArray,
    isSelectionMode,
    lastClickedIndex,
    enterSelectionMode,
    exitSelectionMode,
    toggleSelectionMode,
    handleActionClick,
    toggleSelection,
    selectAll,
    selectNone,
    invertSelection,
    selectRange,
    isSelected,
    selectionCount,
    hasSelection,
  };
}
