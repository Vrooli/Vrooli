/**
 * Tests for usePromptSelection hook.
 *
 * Tests cover:
 * - Single selection
 * - Multi-selection (toggle)
 * - Adding to selection
 * - Removing from selection
 * - Clearing selection
 * - Selection limits
 * - Click handling with modifiers
 */

import { describe, it, expect, vi } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { usePromptSelection } from './usePromptSelection'

describe('usePromptSelection', () => {
  describe('initial state', () => {
    it('should start with empty selection', () => {
      const { result } = renderHook(() => usePromptSelection())

      expect(result.current.selectedIds).toEqual([])
      expect(result.current.selectionCount).toBe(0)
      expect(result.current.hasSelection).toBe(false)
    })

    it('should report mode as single', () => {
      const { result } = renderHook(() => usePromptSelection())

      expect(result.current.mode).toBe('single')
    })
  })

  describe('selectSingle', () => {
    it('should select a single prompt', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
      })

      expect(result.current.selectedIds).toEqual(['prompt-1'])
      expect(result.current.hasSelection).toBe(true)
    })

    it('should replace previous selection', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
      })

      act(() => {
        result.current.selectSingle('prompt-2')
      })

      expect(result.current.selectedIds).toEqual(['prompt-2'])
    })

    it('should call onSelectionChange callback', () => {
      const onChange = vi.fn()
      const { result } = renderHook(() =>
        usePromptSelection({ onSelectionChange: onChange })
      )

      act(() => {
        result.current.selectSingle('prompt-1')
      })

      expect(onChange).toHaveBeenCalledWith(['prompt-1'])
    })

    it('should set anchor ID', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
      })

      expect(result.current.anchorId).toBe('prompt-1')
    })
  })

  describe('toggleSelection', () => {
    it('should add unselected prompt to selection', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
      })

      act(() => {
        result.current.toggleSelection('prompt-2')
      })

      expect(result.current.selectedIds).toContain('prompt-1')
      expect(result.current.selectedIds).toContain('prompt-2')
    })

    it('should remove selected prompt from selection', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
      })

      act(() => {
        result.current.toggleSelection('prompt-1')
      })

      expect(result.current.selectedIds).not.toContain('prompt-1')
    })

    it('should respect maxSelection limit', () => {
      const { result } = renderHook(() =>
        usePromptSelection({ maxSelection: 2 })
      )

      act(() => {
        result.current.selectSingle('prompt-1')
        result.current.toggleSelection('prompt-2')
        result.current.toggleSelection('prompt-3')
      })

      expect(result.current.selectedIds).toHaveLength(2)
      expect(result.current.selectedIds).not.toContain('prompt-3')
    })
  })

  describe('addToSelection', () => {
    it('should add prompt to existing selection', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
        result.current.addToSelection('prompt-2')
      })

      expect(result.current.selectedIds).toEqual(['prompt-1', 'prompt-2'])
    })

    it('should not duplicate already selected prompt', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
        result.current.addToSelection('prompt-1')
      })

      expect(result.current.selectedIds).toEqual(['prompt-1'])
    })

    it('should respect maxSelection limit', () => {
      const { result } = renderHook(() =>
        usePromptSelection({ maxSelection: 1 })
      )

      act(() => {
        result.current.selectSingle('prompt-1')
        result.current.addToSelection('prompt-2')
      })

      expect(result.current.selectedIds).toEqual(['prompt-1'])
    })
  })

  describe('removeFromSelection', () => {
    it('should remove prompt from selection', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
        result.current.toggleSelection('prompt-2')
        result.current.removeFromSelection('prompt-1')
      })

      expect(result.current.selectedIds).toEqual(['prompt-2'])
    })

    it('should handle removing non-selected prompt gracefully', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
        result.current.removeFromSelection('prompt-999')
      })

      expect(result.current.selectedIds).toEqual(['prompt-1'])
    })
  })

  describe('clearSelection', () => {
    it('should clear all selections', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
        result.current.toggleSelection('prompt-2')
        result.current.clearSelection()
      })

      expect(result.current.selectedIds).toEqual([])
      expect(result.current.hasSelection).toBe(false)
    })

    it('should call onSelectionChange with empty array', () => {
      const onChange = vi.fn()
      const { result } = renderHook(() =>
        usePromptSelection({ onSelectionChange: onChange })
      )

      act(() => {
        result.current.selectSingle('prompt-1')
      })

      onChange.mockClear()

      act(() => {
        result.current.clearSelection()
      })

      expect(onChange).toHaveBeenCalledWith([])
    })
  })

  describe('selectAll', () => {
    it('should select all provided IDs', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectAll(['prompt-1', 'prompt-2', 'prompt-3'])
      })

      expect(result.current.selectedIds).toEqual([
        'prompt-1',
        'prompt-2',
        'prompt-3',
      ])
    })

    it('should respect maxSelection limit', () => {
      const { result } = renderHook(() =>
        usePromptSelection({ maxSelection: 2 })
      )

      act(() => {
        result.current.selectAll(['prompt-1', 'prompt-2', 'prompt-3'])
      })

      expect(result.current.selectedIds).toHaveLength(2)
    })

    it('should replace existing selection', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('old-prompt')
        result.current.selectAll(['new-1', 'new-2'])
      })

      expect(result.current.selectedIds).toEqual(['new-1', 'new-2'])
    })
  })

  describe('handleClick', () => {
    it('should select single on plain click', () => {
      const { result } = renderHook(() => usePromptSelection())
      const event = { shiftKey: false, metaKey: false, ctrlKey: false }

      act(() => {
        result.current.handleClick('prompt-1', event)
      })

      expect(result.current.selectedIds).toEqual(['prompt-1'])
    })

    it('should toggle on meta key click', () => {
      const { result } = renderHook(() => usePromptSelection())
      const plainEvent = { shiftKey: false, metaKey: false, ctrlKey: false }
      const metaEvent = { shiftKey: false, metaKey: true, ctrlKey: false }

      act(() => {
        result.current.handleClick('prompt-1', plainEvent)
        result.current.handleClick('prompt-2', metaEvent)
      })

      expect(result.current.selectedIds).toContain('prompt-1')
      expect(result.current.selectedIds).toContain('prompt-2')
    })

    it('should toggle on ctrl key click', () => {
      const { result } = renderHook(() => usePromptSelection())
      const plainEvent = { shiftKey: false, metaKey: false, ctrlKey: false }
      const ctrlEvent = { shiftKey: false, metaKey: false, ctrlKey: true }

      act(() => {
        result.current.handleClick('prompt-1', plainEvent)
        result.current.handleClick('prompt-2', ctrlEvent)
      })

      expect(result.current.selectedIds).toContain('prompt-1')
      expect(result.current.selectedIds).toContain('prompt-2')
    })

    it('should add to selection on shift click when anchor exists', () => {
      const { result } = renderHook(() => usePromptSelection())
      const plainEvent = { shiftKey: false, metaKey: false, ctrlKey: false }
      const shiftEvent = { shiftKey: true, metaKey: false, ctrlKey: false }

      // First establish an anchor with a plain click
      act(() => {
        result.current.handleClick('prompt-1', plainEvent)
      })

      // Then shift-click should add to selection
      act(() => {
        result.current.handleClick('prompt-2', shiftEvent)
      })

      expect(result.current.selectedIds).toContain('prompt-1')
      expect(result.current.selectedIds).toContain('prompt-2')
    })
  })

  describe('isSelected', () => {
    it('should return true for selected prompt', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
      })

      expect(result.current.isSelected('prompt-1')).toBe(true)
    })

    it('should return false for unselected prompt', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
      })

      expect(result.current.isSelected('prompt-2')).toBe(false)
    })
  })

  describe('hasMultipleSelected', () => {
    it('should return false for single selection', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
      })

      expect(result.current.hasMultipleSelected).toBe(false)
    })

    it('should return true for multiple selections', () => {
      const { result } = renderHook(() => usePromptSelection())

      act(() => {
        result.current.selectSingle('prompt-1')
        result.current.toggleSelection('prompt-2')
      })

      expect(result.current.hasMultipleSelected).toBe(true)
    })
  })
})
