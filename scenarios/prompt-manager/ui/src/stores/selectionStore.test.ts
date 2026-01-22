/**
 * Tests for the centralized selection store.
 *
 * Tests cover:
 * - Single selection (for editing)
 * - Multi-selection (for combining)
 * - Toggle selection
 * - Add/remove from selection
 * - Clearing selection
 * - Two-way sync between single and multi selection
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { useSelectionStore } from './selectionStore'

describe('selectionStore', () => {
  beforeEach(() => {
    // Reset store state before each test
    useSelectionStore.setState({
      selectedPromptId: null,
      selectedPromptIds: [],
    })
  })

  describe('initial state', () => {
    it('should start with null selectedPromptId', () => {
      const state = useSelectionStore.getState()
      expect(state.selectedPromptId).toBeNull()
    })

    it('should start with empty selectedPromptIds', () => {
      const state = useSelectionStore.getState()
      expect(state.selectedPromptIds).toEqual([])
    })
  })

  describe('setSelectedPromptId', () => {
    it('should set single selection', () => {
      useSelectionStore.getState().setSelectedPromptId('prompt-1')

      const state = useSelectionStore.getState()
      expect(state.selectedPromptId).toBe('prompt-1')
    })

    it('should also update multi-selection to match', () => {
      useSelectionStore.getState().setSelectedPromptId('prompt-1')

      const state = useSelectionStore.getState()
      expect(state.selectedPromptIds).toEqual(['prompt-1'])
    })

    it('should clear multi-selection when setting to null', () => {
      useSelectionStore.getState().setSelectedPromptId('prompt-1')
      useSelectionStore.getState().setSelectedPromptId(null)

      const state = useSelectionStore.getState()
      expect(state.selectedPromptId).toBeNull()
      expect(state.selectedPromptIds).toEqual([])
    })
  })

  describe('togglePromptSelection', () => {
    it('should add unselected prompt to multi-selection', () => {
      useSelectionStore.getState().togglePromptSelection('prompt-1')

      const state = useSelectionStore.getState()
      expect(state.selectedPromptIds).toContain('prompt-1')
    })

    it('should remove selected prompt from multi-selection', () => {
      useSelectionStore.getState().setSelectedPromptIds(['prompt-1', 'prompt-2'])
      useSelectionStore.getState().togglePromptSelection('prompt-1')

      const state = useSelectionStore.getState()
      expect(state.selectedPromptIds).not.toContain('prompt-1')
      expect(state.selectedPromptIds).toContain('prompt-2')
    })

    it('should set single selection when toggling to single item', () => {
      useSelectionStore.getState().togglePromptSelection('prompt-1')

      const state = useSelectionStore.getState()
      expect(state.selectedPromptId).toBe('prompt-1')
    })

    it('should preserve single selection when adding more items', () => {
      useSelectionStore.getState().setSelectedPromptId('prompt-1')
      useSelectionStore.getState().togglePromptSelection('prompt-2')

      const state = useSelectionStore.getState()
      expect(state.selectedPromptId).toBe('prompt-1')
      expect(state.selectedPromptIds).toContain('prompt-1')
      expect(state.selectedPromptIds).toContain('prompt-2')
    })
  })

  describe('addToSelection', () => {
    it('should add prompt to multi-selection', () => {
      useSelectionStore.getState().setSelectedPromptIds(['prompt-1'])
      useSelectionStore.getState().addToSelection('prompt-2')

      const state = useSelectionStore.getState()
      expect(state.selectedPromptIds).toEqual(['prompt-1', 'prompt-2'])
    })

    it('should not duplicate already selected prompt', () => {
      useSelectionStore.getState().setSelectedPromptIds(['prompt-1'])
      useSelectionStore.getState().addToSelection('prompt-1')

      const state = useSelectionStore.getState()
      expect(state.selectedPromptIds).toEqual(['prompt-1'])
    })

    it('should set single selection when adding first item', () => {
      useSelectionStore.getState().addToSelection('prompt-1')

      const state = useSelectionStore.getState()
      expect(state.selectedPromptId).toBe('prompt-1')
    })
  })

  describe('removeFromSelection', () => {
    it('should remove prompt from multi-selection', () => {
      useSelectionStore.getState().setSelectedPromptIds(['prompt-1', 'prompt-2'])
      useSelectionStore.getState().removeFromSelection('prompt-1')

      const state = useSelectionStore.getState()
      expect(state.selectedPromptIds).toEqual(['prompt-2'])
    })

    it('should clear single selection when removing that prompt', () => {
      useSelectionStore.getState().setSelectedPromptId('prompt-1')
      useSelectionStore.getState().removeFromSelection('prompt-1')

      const state = useSelectionStore.getState()
      expect(state.selectedPromptId).toBeNull()
    })

    it('should preserve single selection when removing different prompt', () => {
      useSelectionStore.getState().setSelectedPromptIds(['prompt-1', 'prompt-2'])
      useSelectionStore.setState({ selectedPromptId: 'prompt-1' })
      useSelectionStore.getState().removeFromSelection('prompt-2')

      const state = useSelectionStore.getState()
      expect(state.selectedPromptId).toBe('prompt-1')
    })
  })

  describe('setSelectedPromptIds', () => {
    it('should set multi-selection', () => {
      useSelectionStore.getState().setSelectedPromptIds(['prompt-1', 'prompt-2'])

      const state = useSelectionStore.getState()
      expect(state.selectedPromptIds).toEqual(['prompt-1', 'prompt-2'])
    })

    it('should set single selection when array has one item', () => {
      useSelectionStore.getState().setSelectedPromptIds(['prompt-1'])

      const state = useSelectionStore.getState()
      expect(state.selectedPromptId).toBe('prompt-1')
    })

    it('should preserve single selection when array has multiple items', () => {
      useSelectionStore.getState().setSelectedPromptId('prompt-1')
      useSelectionStore.getState().setSelectedPromptIds(['prompt-1', 'prompt-2', 'prompt-3'])

      const state = useSelectionStore.getState()
      expect(state.selectedPromptId).toBe('prompt-1')
    })
  })

  describe('clearSelection', () => {
    it('should clear multi-selection', () => {
      useSelectionStore.getState().setSelectedPromptIds(['prompt-1', 'prompt-2'])
      useSelectionStore.getState().clearSelection()

      const state = useSelectionStore.getState()
      expect(state.selectedPromptIds).toEqual([])
    })

    it('should not affect single selection', () => {
      useSelectionStore.getState().setSelectedPromptId('prompt-1')
      useSelectionStore.setState({ selectedPromptIds: ['prompt-1', 'prompt-2'] })
      useSelectionStore.getState().clearSelection()

      const state = useSelectionStore.getState()
      expect(state.selectedPromptId).toBe('prompt-1')
    })
  })

  describe('clearAllSelection', () => {
    it('should clear both single and multi selection', () => {
      useSelectionStore.getState().setSelectedPromptId('prompt-1')
      useSelectionStore.setState({ selectedPromptIds: ['prompt-1', 'prompt-2'] })
      useSelectionStore.getState().clearAllSelection()

      const state = useSelectionStore.getState()
      expect(state.selectedPromptId).toBeNull()
      expect(state.selectedPromptIds).toEqual([])
    })
  })
})
