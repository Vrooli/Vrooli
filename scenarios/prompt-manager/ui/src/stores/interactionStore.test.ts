/**
 * Tests for the interaction store.
 *
 * Tests cover:
 * - Hover state management
 * - Drag operations
 * - Selection state machines
 * - Multi-select mode
 */

import { describe, it, expect, beforeEach } from 'vitest'
import {
  useInteractionStore,
  selectIsHovered,
  selectIsSelected,
  selectIsDragged,
} from './interactionStore'

describe('interactionStore', () => {
  beforeEach(() => {
    // Reset store state before each test
    useInteractionStore.getState().reset()
  })

  describe('initial state', () => {
    it('should start in navigate mode', () => {
      const state = useInteractionStore.getState()
      expect(state.mode).toBe('navigate')
    })

    it('should start with no hovered object', () => {
      const state = useInteractionStore.getState()
      expect(state.hoveredObjectId).toBeNull()
    })

    it('should start with no dragging', () => {
      const state = useInteractionStore.getState()
      expect(state.isDragging).toBe(false)
      expect(state.draggedObjectId).toBeNull()
      expect(state.dragState).toBeNull()
    })

    it('should start with empty selection', () => {
      const state = useInteractionStore.getState()
      expect(state.selectedObjectIds).toEqual([])
    })

    it('should start with multi-select inactive', () => {
      const state = useInteractionStore.getState()
      expect(state.isMultiSelectActive).toBe(false)
    })
  })

  describe('setMode', () => {
    it('should change interaction mode', () => {
      useInteractionStore.getState().setMode('select')
      expect(useInteractionStore.getState().mode).toBe('select')

      useInteractionStore.getState().setMode('drag')
      expect(useInteractionStore.getState().mode).toBe('drag')

      useInteractionStore.getState().setMode('place')
      expect(useInteractionStore.getState().mode).toBe('place')
    })
  })

  describe('setHovered', () => {
    it('should set hovered object id', () => {
      useInteractionStore.getState().setHovered('object-1')
      expect(useInteractionStore.getState().hoveredObjectId).toBe('object-1')
    })

    it('should clear hovered object', () => {
      useInteractionStore.getState().setHovered('object-1')
      useInteractionStore.getState().setHovered(null)
      expect(useInteractionStore.getState().hoveredObjectId).toBeNull()
    })

    it('should not change hover during drag', () => {
      useInteractionStore.getState().startDrag('object-1', [0, 0, 0])
      useInteractionStore.getState().setHovered('object-2')

      expect(useInteractionStore.getState().hoveredObjectId).toBeNull()
    })
  })

  describe('startDrag', () => {
    it('should start drag operation', () => {
      useInteractionStore.getState().startDrag('object-1', [1, 2, 3])

      const state = useInteractionStore.getState()
      expect(state.isDragging).toBe(true)
      expect(state.draggedObjectId).toBe('object-1')
      expect(state.mode).toBe('drag')
    })

    it('should initialize drag state', () => {
      useInteractionStore.getState().startDrag('object-1', [1, 2, 3])

      const state = useInteractionStore.getState()
      expect(state.dragState).toEqual({
        objectId: 'object-1',
        startPosition: [1, 2, 3],
        currentPosition: [1, 2, 3],
        offset: [0, 0, 0],
      })
    })
  })

  describe('updateDrag', () => {
    it('should update drag position', () => {
      useInteractionStore.getState().startDrag('object-1', [0, 0, 0])
      useInteractionStore.getState().updateDrag([5, 3, 2])

      const state = useInteractionStore.getState()
      expect(state.dragState?.currentPosition).toEqual([5, 3, 2])
      expect(state.dragState?.offset).toEqual([5, 3, 2])
    })

    it('should calculate correct offset', () => {
      useInteractionStore.getState().startDrag('object-1', [2, 4, 6])
      useInteractionStore.getState().updateDrag([5, 5, 5])

      const state = useInteractionStore.getState()
      expect(state.dragState?.offset).toEqual([3, 1, -1])
    })

    it('should do nothing if not dragging', () => {
      useInteractionStore.getState().updateDrag([5, 3, 2])

      const state = useInteractionStore.getState()
      expect(state.dragState).toBeNull()
    })
  })

  describe('endDrag', () => {
    it('should end drag operation', () => {
      useInteractionStore.getState().startDrag('object-1', [0, 0, 0])
      useInteractionStore.getState().endDrag()

      const state = useInteractionStore.getState()
      expect(state.isDragging).toBe(false)
      expect(state.draggedObjectId).toBeNull()
      expect(state.dragState).toBeNull()
      expect(state.mode).toBe('navigate')
    })
  })

  describe('cancelDrag', () => {
    it('should cancel drag operation', () => {
      useInteractionStore.getState().startDrag('object-1', [0, 0, 0])
      useInteractionStore.getState().updateDrag([10, 10, 10])
      useInteractionStore.getState().cancelDrag()

      const state = useInteractionStore.getState()
      expect(state.isDragging).toBe(false)
      expect(state.draggedObjectId).toBeNull()
      expect(state.dragState).toBeNull()
      expect(state.mode).toBe('navigate')
    })
  })

  describe('selectObject', () => {
    it('should select a single object', () => {
      useInteractionStore.getState().selectObject('object-1')

      const state = useInteractionStore.getState()
      expect(state.selectedObjectIds).toEqual(['object-1'])
    })

    it('should replace previous selection', () => {
      useInteractionStore.getState().selectObject('object-1')
      useInteractionStore.getState().selectObject('object-2')

      const state = useInteractionStore.getState()
      expect(state.selectedObjectIds).toEqual(['object-2'])
    })
  })

  describe('toggleSelection', () => {
    it('should add object to empty selection', () => {
      useInteractionStore.getState().toggleSelection('object-1')

      const state = useInteractionStore.getState()
      expect(state.selectedObjectIds).toContain('object-1')
    })

    it('should remove already selected object', () => {
      useInteractionStore.getState().selectObject('object-1')
      useInteractionStore.getState().toggleSelection('object-1')

      const state = useInteractionStore.getState()
      expect(state.selectedObjectIds).not.toContain('object-1')
    })

    it('should add to existing selection', () => {
      useInteractionStore.getState().selectObject('object-1')
      useInteractionStore.getState().toggleSelection('object-2')

      const state = useInteractionStore.getState()
      expect(state.selectedObjectIds).toContain('object-1')
      expect(state.selectedObjectIds).toContain('object-2')
    })
  })

  describe('addToSelection', () => {
    it('should add object to selection', () => {
      useInteractionStore.getState().selectObject('object-1')
      useInteractionStore.getState().addToSelection('object-2')

      const state = useInteractionStore.getState()
      expect(state.selectedObjectIds).toEqual(['object-1', 'object-2'])
    })

    it('should not duplicate already selected object', () => {
      useInteractionStore.getState().selectObject('object-1')
      useInteractionStore.getState().addToSelection('object-1')

      const state = useInteractionStore.getState()
      expect(state.selectedObjectIds).toEqual(['object-1'])
    })
  })

  describe('clearSelection', () => {
    it('should clear all selected objects', () => {
      useInteractionStore.getState().selectObject('object-1')
      useInteractionStore.getState().addToSelection('object-2')
      useInteractionStore.getState().clearSelection()

      const state = useInteractionStore.getState()
      expect(state.selectedObjectIds).toEqual([])
    })
  })

  describe('setMultiSelectActive', () => {
    it('should enable multi-select', () => {
      useInteractionStore.getState().setMultiSelectActive(true)
      expect(useInteractionStore.getState().isMultiSelectActive).toBe(true)
    })

    it('should disable multi-select', () => {
      useInteractionStore.getState().setMultiSelectActive(true)
      useInteractionStore.getState().setMultiSelectActive(false)
      expect(useInteractionStore.getState().isMultiSelectActive).toBe(false)
    })
  })

  describe('setLastClickPosition', () => {
    it('should record click position', () => {
      useInteractionStore.getState().setLastClickPosition([5, 10, 15])
      expect(useInteractionStore.getState().lastClickPosition).toEqual([5, 10, 15])
    })

    it('should clear click position', () => {
      useInteractionStore.getState().setLastClickPosition([5, 10, 15])
      useInteractionStore.getState().setLastClickPosition(null)
      expect(useInteractionStore.getState().lastClickPosition).toBeNull()
    })
  })

  describe('selectors', () => {
    describe('selectIsHovered', () => {
      it('should return true for hovered object', () => {
        useInteractionStore.getState().setHovered('object-1')

        const state = useInteractionStore.getState()
        expect(selectIsHovered(state, 'object-1')).toBe(true)
      })

      it('should return false for non-hovered object', () => {
        useInteractionStore.getState().setHovered('object-1')

        const state = useInteractionStore.getState()
        expect(selectIsHovered(state, 'object-2')).toBe(false)
      })
    })

    describe('selectIsSelected', () => {
      it('should return true for selected object', () => {
        useInteractionStore.getState().selectObject('object-1')

        const state = useInteractionStore.getState()
        expect(selectIsSelected(state, 'object-1')).toBe(true)
      })

      it('should return false for non-selected object', () => {
        useInteractionStore.getState().selectObject('object-1')

        const state = useInteractionStore.getState()
        expect(selectIsSelected(state, 'object-2')).toBe(false)
      })
    })

    describe('selectIsDragged', () => {
      it('should return true for dragged object', () => {
        useInteractionStore.getState().startDrag('object-1', [0, 0, 0])

        const state = useInteractionStore.getState()
        expect(selectIsDragged(state, 'object-1')).toBe(true)
      })

      it('should return false for non-dragged object', () => {
        useInteractionStore.getState().startDrag('object-1', [0, 0, 0])

        const state = useInteractionStore.getState()
        expect(selectIsDragged(state, 'object-2')).toBe(false)
      })
    })
  })

  describe('reset', () => {
    it('should reset all state', () => {
      useInteractionStore.getState().setMode('select')
      useInteractionStore.getState().setHovered('object-1')
      useInteractionStore.getState().startDrag('object-2', [0, 0, 0])
      useInteractionStore.getState().selectObject('object-3')
      useInteractionStore.getState().setMultiSelectActive(true)

      useInteractionStore.getState().reset()

      const state = useInteractionStore.getState()
      expect(state.mode).toBe('navigate')
      expect(state.hoveredObjectId).toBeNull()
      expect(state.isDragging).toBe(false)
      expect(state.selectedObjectIds).toEqual([])
      expect(state.isMultiSelectActive).toBe(false)
    })
  })
})
