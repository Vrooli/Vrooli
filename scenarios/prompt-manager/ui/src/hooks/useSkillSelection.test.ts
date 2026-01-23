/**
 * Tests for useSkillSelection hook.
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
import { useSkillSelection } from './useSkillSelection'

describe('useSkillSelection', () => {
  describe('initial state', () => {
    it('should start with empty selection', () => {
      const { result } = renderHook(() => useSkillSelection())

      expect(result.current.selectedIds).toEqual([])
      expect(result.current.selectionCount).toBe(0)
      expect(result.current.hasSelection).toBe(false)
    })

    it('should report mode as single', () => {
      const { result } = renderHook(() => useSkillSelection())

      expect(result.current.mode).toBe('single')
    })
  })

  describe('selectSingle', () => {
    it('should select a single skill', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
      })

      expect(result.current.selectedIds).toEqual(['skill-1'])
      expect(result.current.hasSelection).toBe(true)
    })

    it('should replace previous selection', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
      })

      act(() => {
        result.current.selectSingle('skill-2')
      })

      expect(result.current.selectedIds).toEqual(['skill-2'])
    })

    it('should call onSelectionChange callback', () => {
      const onChange = vi.fn()
      const { result } = renderHook(() =>
        useSkillSelection({ onSelectionChange: onChange })
      )

      act(() => {
        result.current.selectSingle('skill-1')
      })

      expect(onChange).toHaveBeenCalledWith(['skill-1'])
    })

    it('should set anchor ID', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
      })

      expect(result.current.anchorId).toBe('skill-1')
    })
  })

  describe('toggleSelection', () => {
    it('should add unselected skill to selection', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
      })

      act(() => {
        result.current.toggleSelection('skill-2')
      })

      expect(result.current.selectedIds).toContain('skill-1')
      expect(result.current.selectedIds).toContain('skill-2')
    })

    it('should remove selected skill from selection', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
      })

      act(() => {
        result.current.toggleSelection('skill-1')
      })

      expect(result.current.selectedIds).not.toContain('skill-1')
    })

    it('should respect maxSelection limit', () => {
      const { result } = renderHook(() =>
        useSkillSelection({ maxSelection: 2 })
      )

      act(() => {
        result.current.selectSingle('skill-1')
        result.current.toggleSelection('skill-2')
        result.current.toggleSelection('skill-3')
      })

      expect(result.current.selectedIds).toHaveLength(2)
      expect(result.current.selectedIds).not.toContain('skill-3')
    })
  })

  describe('addToSelection', () => {
    it('should add skill to existing selection', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
        result.current.addToSelection('skill-2')
      })

      expect(result.current.selectedIds).toEqual(['skill-1', 'skill-2'])
    })

    it('should not duplicate already selected skill', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
        result.current.addToSelection('skill-1')
      })

      expect(result.current.selectedIds).toEqual(['skill-1'])
    })

    it('should respect maxSelection limit', () => {
      const { result } = renderHook(() =>
        useSkillSelection({ maxSelection: 1 })
      )

      act(() => {
        result.current.selectSingle('skill-1')
        result.current.addToSelection('skill-2')
      })

      expect(result.current.selectedIds).toEqual(['skill-1'])
    })
  })

  describe('removeFromSelection', () => {
    it('should remove skill from selection', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
        result.current.toggleSelection('skill-2')
        result.current.removeFromSelection('skill-1')
      })

      expect(result.current.selectedIds).toEqual(['skill-2'])
    })

    it('should handle removing non-selected skill gracefully', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
        result.current.removeFromSelection('skill-999')
      })

      expect(result.current.selectedIds).toEqual(['skill-1'])
    })
  })

  describe('clearSelection', () => {
    it('should clear all selections', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
        result.current.toggleSelection('skill-2')
        result.current.clearSelection()
      })

      expect(result.current.selectedIds).toEqual([])
      expect(result.current.hasSelection).toBe(false)
    })

    it('should call onSelectionChange with empty array', () => {
      const onChange = vi.fn()
      const { result } = renderHook(() =>
        useSkillSelection({ onSelectionChange: onChange })
      )

      act(() => {
        result.current.selectSingle('skill-1')
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
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectAll(['skill-1', 'skill-2', 'skill-3'])
      })

      expect(result.current.selectedIds).toEqual([
        'skill-1',
        'skill-2',
        'skill-3',
      ])
    })

    it('should respect maxSelection limit', () => {
      const { result } = renderHook(() =>
        useSkillSelection({ maxSelection: 2 })
      )

      act(() => {
        result.current.selectAll(['skill-1', 'skill-2', 'skill-3'])
      })

      expect(result.current.selectedIds).toHaveLength(2)
    })

    it('should replace existing selection', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('old-skill')
        result.current.selectAll(['new-1', 'new-2'])
      })

      expect(result.current.selectedIds).toEqual(['new-1', 'new-2'])
    })
  })

  describe('handleClick', () => {
    it('should select single on plain click', () => {
      const { result } = renderHook(() => useSkillSelection())
      const event = { shiftKey: false, metaKey: false, ctrlKey: false }

      act(() => {
        result.current.handleClick('skill-1', event)
      })

      expect(result.current.selectedIds).toEqual(['skill-1'])
    })

    it('should toggle on meta key click', () => {
      const { result } = renderHook(() => useSkillSelection())
      const plainEvent = { shiftKey: false, metaKey: false, ctrlKey: false }
      const metaEvent = { shiftKey: false, metaKey: true, ctrlKey: false }

      act(() => {
        result.current.handleClick('skill-1', plainEvent)
        result.current.handleClick('skill-2', metaEvent)
      })

      expect(result.current.selectedIds).toContain('skill-1')
      expect(result.current.selectedIds).toContain('skill-2')
    })

    it('should toggle on ctrl key click', () => {
      const { result } = renderHook(() => useSkillSelection())
      const plainEvent = { shiftKey: false, metaKey: false, ctrlKey: false }
      const ctrlEvent = { shiftKey: false, metaKey: false, ctrlKey: true }

      act(() => {
        result.current.handleClick('skill-1', plainEvent)
        result.current.handleClick('skill-2', ctrlEvent)
      })

      expect(result.current.selectedIds).toContain('skill-1')
      expect(result.current.selectedIds).toContain('skill-2')
    })

    it('should add to selection on shift click when anchor exists', () => {
      const { result } = renderHook(() => useSkillSelection())
      const plainEvent = { shiftKey: false, metaKey: false, ctrlKey: false }
      const shiftEvent = { shiftKey: true, metaKey: false, ctrlKey: false }

      // First establish an anchor with a plain click
      act(() => {
        result.current.handleClick('skill-1', plainEvent)
      })

      // Then shift-click should add to selection
      act(() => {
        result.current.handleClick('skill-2', shiftEvent)
      })

      expect(result.current.selectedIds).toContain('skill-1')
      expect(result.current.selectedIds).toContain('skill-2')
    })
  })

  describe('isSelected', () => {
    it('should return true for selected skill', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
      })

      expect(result.current.isSelected('skill-1')).toBe(true)
    })

    it('should return false for unselected skill', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
      })

      expect(result.current.isSelected('skill-2')).toBe(false)
    })
  })

  describe('hasMultipleSelected', () => {
    it('should return false for single selection', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
      })

      expect(result.current.hasMultipleSelected).toBe(false)
    })

    it('should return true for multiple selections', () => {
      const { result } = renderHook(() => useSkillSelection())

      act(() => {
        result.current.selectSingle('skill-1')
        result.current.toggleSelection('skill-2')
      })

      expect(result.current.hasMultipleSelected).toBe(true)
    })
  })
})
