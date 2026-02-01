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
      selectedSkillId: null,
      selectedSkillIds: [],
      selectedAgentId: null,
    })
  })

  describe('initial state', () => {
    it('should start with null selectedSkillId', () => {
      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBeNull()
    })

    it('should start with empty selectedSkillIds', () => {
      const state = useSelectionStore.getState()
      expect(state.selectedSkillIds).toEqual([])
    })

    it('should start with null selectedAgentId', () => {
      const state = useSelectionStore.getState()
      expect(state.selectedAgentId).toBeNull()
    })
  })

  describe('setSelectedSkillId', () => {
    it('should set single selection', () => {
      useSelectionStore.getState().setSelectedSkillId('skill-1')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBe('skill-1')
    })

    it('should also update multi-selection to match', () => {
      useSelectionStore.getState().setSelectedSkillId('skill-1')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillIds).toEqual(['skill-1'])
    })

    it('should clear multi-selection when setting to null', () => {
      useSelectionStore.getState().setSelectedSkillId('skill-1')
      useSelectionStore.getState().setSelectedSkillId(null)

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBeNull()
      expect(state.selectedSkillIds).toEqual([])
    })
  })

  describe('toggleSkillSelection', () => {
    it('should add unselected skill to multi-selection', () => {
      useSelectionStore.getState().toggleSkillSelection('skill-1')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillIds).toContain('skill-1')
    })

    it('should remove selected skill from multi-selection', () => {
      useSelectionStore.getState().setSelectedSkillIds(['skill-1', 'skill-2'])
      useSelectionStore.getState().toggleSkillSelection('skill-1')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillIds).not.toContain('skill-1')
      expect(state.selectedSkillIds).toContain('skill-2')
    })

    it('should set single selection when toggling to single item', () => {
      useSelectionStore.getState().toggleSkillSelection('skill-1')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBe('skill-1')
    })

    it('should preserve single selection when adding more items', () => {
      useSelectionStore.getState().setSelectedSkillId('skill-1')
      useSelectionStore.getState().toggleSkillSelection('skill-2')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBe('skill-1')
      expect(state.selectedSkillIds).toContain('skill-1')
      expect(state.selectedSkillIds).toContain('skill-2')
    })
  })

  describe('addToSelection', () => {
    it('should add skill to multi-selection', () => {
      useSelectionStore.getState().setSelectedSkillIds(['skill-1'])
      useSelectionStore.getState().addToSelection('skill-2')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillIds).toEqual(['skill-1', 'skill-2'])
    })

    it('should not duplicate already selected skill', () => {
      useSelectionStore.getState().setSelectedSkillIds(['skill-1'])
      useSelectionStore.getState().addToSelection('skill-1')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillIds).toEqual(['skill-1'])
    })

    it('should set single selection when adding first item', () => {
      useSelectionStore.getState().addToSelection('skill-1')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBe('skill-1')
    })
  })

  describe('removeFromSelection', () => {
    it('should remove skill from multi-selection', () => {
      useSelectionStore.getState().setSelectedSkillIds(['skill-1', 'skill-2'])
      useSelectionStore.getState().removeFromSelection('skill-1')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillIds).toEqual(['skill-2'])
    })

    it('should clear single selection when removing that skill', () => {
      useSelectionStore.getState().setSelectedSkillId('skill-1')
      useSelectionStore.getState().removeFromSelection('skill-1')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBeNull()
    })

    it('should preserve single selection when removing different skill', () => {
      useSelectionStore.getState().setSelectedSkillIds(['skill-1', 'skill-2'])
      useSelectionStore.setState({ selectedSkillId: 'skill-1' })
      useSelectionStore.getState().removeFromSelection('skill-2')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBe('skill-1')
    })
  })

  describe('setSelectedSkillIds', () => {
    it('should set multi-selection', () => {
      useSelectionStore.getState().setSelectedSkillIds(['skill-1', 'skill-2'])

      const state = useSelectionStore.getState()
      expect(state.selectedSkillIds).toEqual(['skill-1', 'skill-2'])
    })

    it('should set single selection when array has one item', () => {
      useSelectionStore.getState().setSelectedSkillIds(['skill-1'])

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBe('skill-1')
    })

    it('should preserve single selection when array has multiple items', () => {
      useSelectionStore.getState().setSelectedSkillId('skill-1')
      useSelectionStore.getState().setSelectedSkillIds(['skill-1', 'skill-2', 'skill-3'])

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBe('skill-1')
    })
  })

  describe('clearSelection', () => {
    it('should clear multi-selection', () => {
      useSelectionStore.getState().setSelectedSkillIds(['skill-1', 'skill-2'])
      useSelectionStore.getState().clearSelection()

      const state = useSelectionStore.getState()
      expect(state.selectedSkillIds).toEqual([])
    })

    it('should not affect single selection', () => {
      useSelectionStore.getState().setSelectedSkillId('skill-1')
      useSelectionStore.setState({ selectedSkillIds: ['skill-1', 'skill-2'] })
      useSelectionStore.getState().clearSelection()

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBe('skill-1')
    })
  })

  describe('clearAllSelection', () => {
    it('should clear both single and multi selection', () => {
      useSelectionStore.getState().setSelectedSkillId('skill-1')
      useSelectionStore.setState({ selectedSkillIds: ['skill-1', 'skill-2'] })
      useSelectionStore.getState().clearAllSelection()

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBeNull()
      expect(state.selectedSkillIds).toEqual([])
    })

    it('should also clear agent selection', () => {
      useSelectionStore.getState().setSelectedAgentId('agent-1')
      useSelectionStore.getState().clearAllSelection()

      const state = useSelectionStore.getState()
      expect(state.selectedAgentId).toBeNull()
    })
  })

  describe('setSelectedAgentId', () => {
    it('should set agent selection', () => {
      useSelectionStore.getState().setSelectedAgentId('agent-1')

      const state = useSelectionStore.getState()
      expect(state.selectedAgentId).toBe('agent-1')
    })

    it('should clear skill selection when selecting agent', () => {
      useSelectionStore.getState().setSelectedSkillId('skill-1')
      useSelectionStore.getState().setSelectedAgentId('agent-1')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBeNull()
      expect(state.selectedSkillIds).toEqual([])
      expect(state.selectedAgentId).toBe('agent-1')
    })

    it('should clear agent selection when set to null', () => {
      useSelectionStore.getState().setSelectedAgentId('agent-1')
      useSelectionStore.getState().setSelectedAgentId(null)

      const state = useSelectionStore.getState()
      expect(state.selectedAgentId).toBeNull()
    })
  })

  describe('skill and agent selection mutual exclusivity', () => {
    it('should clear agent selection when selecting skill', () => {
      useSelectionStore.getState().setSelectedAgentId('agent-1')
      useSelectionStore.getState().setSelectedSkillId('skill-1')

      const state = useSelectionStore.getState()
      expect(state.selectedAgentId).toBeNull()
      expect(state.selectedSkillId).toBe('skill-1')
    })

    it('should clear skill selection when selecting agent', () => {
      useSelectionStore.getState().setSelectedSkillId('skill-1')
      useSelectionStore.getState().setSelectedAgentId('agent-1')

      const state = useSelectionStore.getState()
      expect(state.selectedSkillId).toBeNull()
      expect(state.selectedAgentId).toBe('agent-1')
    })
  })
})
