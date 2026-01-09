import { describe, it, expect } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useActionSelection } from './useActionSelection';

describe('useActionSelection', () => {
  describe('basic selection', () => {
    it('starts with empty selection and not in selection mode', () => {
      const { result } = renderHook(() => useActionSelection({ actionCount: 10 }));

      expect(result.current.isSelectionMode).toBe(false);
      expect(result.current.selectedIndicesArray).toEqual([]);
      expect(result.current.selectionCount).toBe(0);
      expect(result.current.hasSelection).toBe(false);
    });

    it('toggles selection mode', () => {
      const { result } = renderHook(() => useActionSelection({ actionCount: 10 }));

      act(() => {
        result.current.toggleSelectionMode();
      });
      expect(result.current.isSelectionMode).toBe(true);

      act(() => {
        result.current.toggleSelectionMode();
      });
      expect(result.current.isSelectionMode).toBe(false);
    });

    it('toggles individual item selection', () => {
      const { result } = renderHook(() => useActionSelection({ actionCount: 10 }));

      act(() => {
        result.current.enterSelectionMode();
        result.current.toggleSelection(3);
      });

      expect(result.current.isSelected(3)).toBe(true);
      expect(result.current.selectedIndicesArray).toEqual([3]);

      act(() => {
        result.current.toggleSelection(3);
      });

      expect(result.current.isSelected(3)).toBe(false);
      expect(result.current.selectedIndicesArray).toEqual([]);
    });
  });

  describe('range selection', () => {
    it('selects a range of items', () => {
      const { result } = renderHook(() => useActionSelection({ actionCount: 10 }));

      act(() => {
        result.current.enterSelectionMode();
        result.current.selectRange(2, 5);
      });

      expect(result.current.selectedIndicesArray).toEqual([2, 3, 4, 5]);
      expect(result.current.selectionCount).toBe(4);
    });

    it('handles reversed range (end < start)', () => {
      const { result } = renderHook(() => useActionSelection({ actionCount: 10 }));

      act(() => {
        result.current.selectRange(5, 2);
      });

      expect(result.current.selectedIndicesArray).toEqual([2, 3, 4, 5]);
    });

    it('handles shift+click for range selection', () => {
      const { result } = renderHook(() => useActionSelection({ actionCount: 10 }));

      // First click sets lastClickedIndex
      act(() => {
        result.current.handleActionClick(2, false, false);
      });

      // Shift+click extends selection from lastClickedIndex to current
      act(() => {
        result.current.handleActionClick(6, true, false);
      });

      expect(result.current.selectedIndicesArray).toEqual([2, 3, 4, 5, 6]);
    });
  });

  describe('multi-selection', () => {
    it('ctrl+click toggles individual items without affecting others', () => {
      const { result } = renderHook(() => useActionSelection({ actionCount: 10 }));

      act(() => {
        result.current.handleActionClick(2, false, false);
        result.current.handleActionClick(5, false, true); // Ctrl+click
      });

      expect(result.current.isSelected(2)).toBe(true);
      expect(result.current.isSelected(5)).toBe(true);
      expect(result.current.selectionCount).toBe(2);

      // Ctrl+click again to deselect
      act(() => {
        result.current.handleActionClick(2, false, true);
      });

      expect(result.current.isSelected(2)).toBe(false);
      expect(result.current.isSelected(5)).toBe(true);
    });
  });

  describe('bulk operations', () => {
    it('selectAll selects all items', () => {
      const { result } = renderHook(() => useActionSelection({ actionCount: 5 }));

      act(() => {
        result.current.selectAll();
      });

      expect(result.current.selectedIndicesArray).toEqual([0, 1, 2, 3, 4]);
      expect(result.current.selectionCount).toBe(5);
    });

    it('selectNone clears all selections', () => {
      const { result } = renderHook(() => useActionSelection({ actionCount: 5 }));

      act(() => {
        result.current.selectAll();
        result.current.selectNone();
      });

      expect(result.current.selectedIndicesArray).toEqual([]);
      expect(result.current.hasSelection).toBe(false);
    });

    it('invertSelection inverts the selection', () => {
      const { result } = renderHook(() => useActionSelection({ actionCount: 5 }));

      act(() => {
        result.current.toggleSelection(1);
        result.current.toggleSelection(3);
      });

      expect(result.current.selectedIndicesArray).toEqual([1, 3]);

      act(() => {
        result.current.invertSelection();
      });

      expect(result.current.selectedIndicesArray).toEqual([0, 2, 4]);
    });
  });

  describe('exit selection mode', () => {
    it('clears selection when exiting selection mode', () => {
      const { result } = renderHook(() => useActionSelection({ actionCount: 5 }));

      act(() => {
        result.current.enterSelectionMode();
        result.current.selectAll();
      });

      expect(result.current.selectionCount).toBe(5);

      act(() => {
        result.current.exitSelectionMode();
      });

      expect(result.current.isSelectionMode).toBe(false);
      expect(result.current.selectionCount).toBe(0);
    });
  });

  describe('auto-enter selection mode', () => {
    it('auto-enters selection mode on first click when not in selection mode', () => {
      const { result } = renderHook(() => useActionSelection({ actionCount: 10 }));

      expect(result.current.isSelectionMode).toBe(false);

      act(() => {
        result.current.handleActionClick(5, false, false);
      });

      expect(result.current.isSelectionMode).toBe(true);
      expect(result.current.isSelected(5)).toBe(true);
    });
  });
});
