/**
 * Tests for usePromptEditor hook.
 *
 * Tests cover:
 * - Current skill selection and form state
 * - Form field updates
 * - Dirty state tracking
 * - Undo/redo operations
 * - Multi-prompt editing persistence
 * - Save and discard operations
 * - Delete operations
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act, waitFor } from '@testing-library/react'
import { usePromptEditor, type SaveResult } from './usePromptEditor'
import { useEditorStore } from '@/stores/editorStore'
import type { Skill, UpdateSkillRequest } from '@/types'

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {}
  return {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key]
    }),
    clear: vi.fn(() => {
      store = {}
    }),
  }
})()

Object.defineProperty(window, 'localStorage', { value: localStorageMock })

// Helper to create a test skill
function createTestSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'test-1',
    name: 'Test Skill',
    description: 'A test description',
    content: '# Test content',
    modes: ['development', 'testing'],
    tags: ['tag1', 'tag2'],
    icon: 'file',
    targetToolId: 'tool-123',
    draft: false,
    folder: 'local',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 5,
    lastUsed: '2025-01-01T12:00:00Z',
    effectivenessRating: 4.5,
    ...overrides,
  }
}

// Reset store between tests
function resetStore() {
  useEditorStore.setState({
    prompts: new Map(),
    isHydrated: true,
  })
}

describe('usePromptEditor', () => {
  const mockOnSave = vi.fn<
    [Map<string, UpdateSkillRequest>],
    Promise<Map<string, Skill | Error>>
  >()
  const mockOnDelete = vi.fn<[string], Promise<void>>()

  beforeEach(() => {
    vi.clearAllMocks()
    localStorageMock.clear()
    resetStore()
    mockOnSave.mockResolvedValue(new Map())
    mockOnDelete.mockResolvedValue(undefined)
  })

  afterEach(() => {
    vi.clearAllTimers()
  })

  describe('initial state', () => {
    it('should return null currentSkill when no item selected', async () => {
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [createTestSkill()],
          selectedItemId: null,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      expect(result.current.currentSkill).toBeNull()
      expect(result.current.isDirty).toBe(false)
    })

    it('should load form state when skill is selected', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.currentSkill).toEqual(skill)
        expect(result.current.formState.name).toBe('Test Skill')
        expect(result.current.formState.content).toBe('# Test content')
        expect(result.current.formState.modes).toEqual(['development', 'testing'])
        // Tags are now an array, not comma-separated string
        expect(result.current.formState.tags).toEqual(['tag1', 'tag2'])
      })
    })

    it('should return null for non-existent selected item', async () => {
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [createTestSkill()],
          selectedItemId: 'non-existent',
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      expect(result.current.currentSkill).toBeNull()
    })
  })

  describe('form field updates', () => {
    it('should update field value with updateField', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      act(() => {
        result.current.updateField('name', 'Updated Name')
      })

      expect(result.current.formState.name).toBe('Updated Name')
    })

    it('should update modes with setModes', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.modes).toEqual(['development', 'testing'])
      })

      act(() => {
        result.current.setModes(['new-mode-1', 'new-mode-2'])
      })

      expect(result.current.formState.modes).toEqual(['new-mode-1', 'new-mode-2'])
    })

    it('should update tags with setTags', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.tags).toEqual(['tag1', 'tag2'])
      })

      act(() => {
        result.current.setTags(['new-tag', 'another-tag'])
      })

      expect(result.current.formState.tags).toEqual(['new-tag', 'another-tag'])
    })

    it('should reset form to original with resetForm', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      act(() => {
        result.current.updateField('name', 'Changed Name')
        result.current.updateField('content', 'Changed content')
      })

      expect(result.current.formState.name).toBe('Changed Name')

      act(() => {
        result.current.resetForm()
      })

      expect(result.current.formState.name).toBe('Test Skill')
      expect(result.current.formState.content).toBe('# Test content')
    })
  })

  describe('dirty state tracking', () => {
    it('should not be dirty when form matches original', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      expect(result.current.isDirty).toBe(false)
      expect(result.current.dirtyCount).toBe(0)
    })

    it('should be dirty when form is modified', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      act(() => {
        result.current.updateField('name', 'Modified Name')
      })

      expect(result.current.isDirty).toBe(true)
      expect(result.current.dirtyItemIds.has(skill.id)).toBe(true)
      expect(result.current.dirtyCount).toBe(1)
    })
  })

  describe('undo/redo operations', () => {
    it('should undo a change', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      // Initially no undo available
      expect(result.current.canUndo).toBe(false)

      // Make a change
      act(() => {
        result.current.updateField('name', 'Changed Name')
      })

      expect(result.current.formState.name).toBe('Changed Name')
      expect(result.current.canUndo).toBe(true)

      // Undo
      act(() => {
        result.current.undo()
      })

      expect(result.current.formState.name).toBe('Test Skill')
      expect(result.current.canUndo).toBe(false)
      expect(result.current.canRedo).toBe(true)
    })

    it('should redo an undone change', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      // Make a change
      act(() => {
        result.current.updateField('name', 'Changed Name')
      })

      // Undo
      act(() => {
        result.current.undo()
      })

      expect(result.current.formState.name).toBe('Test Skill')
      expect(result.current.canRedo).toBe(true)

      // Redo
      act(() => {
        result.current.redo()
      })

      expect(result.current.formState.name).toBe('Changed Name')
      expect(result.current.canRedo).toBe(false)
    })

    it('should clear redo stack on new edit', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      // Make changes
      act(() => {
        result.current.updateField('name', 'Change 1')
      })
      act(() => {
        result.current.updateField('name', 'Change 2')
      })

      // Undo
      act(() => {
        result.current.undo()
      })

      expect(result.current.canRedo).toBe(true)

      // Make new edit
      act(() => {
        result.current.updateField('name', 'New Change')
      })

      // Redo stack should be cleared
      expect(result.current.canRedo).toBe(false)
    })
  })

  describe('validation', () => {
    it('should validate form and return valid state', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      expect(result.current.isValid).toBe(true)
      expect(result.current.validation.valid).toBe(true)
      expect(Object.keys(result.current.validation.errors)).toHaveLength(0)
    })

    it('should detect invalid form when name is empty', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      act(() => {
        result.current.updateField('name', '')
      })

      expect(result.current.isValid).toBe(false)
      expect(result.current.validation.errors.name).toBeDefined()
    })

    it('should detect invalid form when content is empty', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      act(() => {
        result.current.updateField('content', '')
      })

      expect(result.current.isValid).toBe(false)
      expect(result.current.validation.errors.content).toBeDefined()
    })
  })

  describe('multi-prompt editing', () => {
    it('should preserve changes when switching prompts', async () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'Skill 1' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Skill 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          usePromptEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'skill-1' } }
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Skill 1')
      })

      // Modify the first skill
      act(() => {
        result.current.updateField('name', 'Modified Skill 1')
      })

      // Switch to second skill
      rerender({ selectedItemId: 'skill-2' })

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Skill 2')
      })

      // Switch back to first skill
      rerender({ selectedItemId: 'skill-1' })

      await waitFor(() => {
        // Changes should be preserved (new store persists automatically)
        expect(result.current.formState.name).toBe('Modified Skill 1')
      })
    })

    it('should track multiple dirty prompts', async () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'Skill 1' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Skill 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          usePromptEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'skill-1' } }
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Skill 1')
      })

      // Modify skill 1
      act(() => {
        result.current.updateField('name', 'Modified Skill 1')
      })

      // Switch to skill 2 and modify
      rerender({ selectedItemId: 'skill-2' })

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Skill 2')
      })

      act(() => {
        result.current.updateField('name', 'Modified Skill 2')
      })

      // Both should be dirty
      expect(result.current.dirtyItemIds.has('skill-1')).toBe(true)
      expect(result.current.dirtyItemIds.has('skill-2')).toBe(true)
      expect(result.current.dirtyCount).toBe(2)
    })
  })

  describe('save operations', () => {
    it('should save current skill with saveCurrentSkill', async () => {
      const skill = createTestSkill()
      const updatedSkill = { ...skill, name: 'Updated Name' }
      mockOnSave.mockResolvedValue(new Map([[skill.id, updatedSkill]]))

      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      act(() => {
        result.current.updateField('name', 'Updated Name')
      })

      await act(async () => {
        await result.current.saveCurrentSkill()
      })

      expect(mockOnSave).toHaveBeenCalledTimes(1)
      const calls = mockOnSave.mock.calls
      expect(calls.length).toBeGreaterThan(0)
      const updates = calls[0]?.[0]
      expect(updates).toBeDefined()
      expect(updates?.has(skill.id)).toBe(true)
      expect(updates?.get(skill.id)?.name).toBe('Updated Name')
    })

    it('should not save when not dirty', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      await act(async () => {
        await result.current.saveCurrentSkill()
      })

      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should not save when validation fails', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      act(() => {
        result.current.updateField('name', '') // Invalid
      })

      await act(async () => {
        await result.current.saveCurrentSkill()
      })

      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should set isSaving during save operation', async () => {
      const skill = createTestSkill()
      let resolvePromise: (value: Map<string, Skill | Error>) => void = () => {}
      mockOnSave.mockImplementation(
        () => new Promise((resolve) => { resolvePromise = resolve })
      )

      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      act(() => {
        result.current.updateField('name', 'Updated')
      })

      let savePromise: Promise<SaveResult>
      act(() => {
        savePromise = result.current.saveCurrentSkill()
      })

      expect(result.current.isSaving).toBe(true)

      await act(async () => {
        resolvePromise(new Map())
        await savePromise
      })

      expect(result.current.isSaving).toBe(false)
    })
  })

  describe('discard operations', () => {
    it('should discard current changes with discardCurrentChanges', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      act(() => {
        result.current.updateField('name', 'Modified Name')
        result.current.updateField('content', 'Modified content')
      })

      expect(result.current.isDirty).toBe(true)

      act(() => {
        result.current.discardCurrentChanges()
      })

      expect(result.current.formState.name).toBe('Test Skill')
      expect(result.current.formState.content).toBe('# Test content')
      expect(result.current.isDirty).toBe(false)
    })

    it('should discard all changes with discardAllChanges', async () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'Skill 1' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Skill 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          usePromptEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'skill-1' } }
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Skill 1')
      })

      // Modify skill 1
      act(() => {
        result.current.updateField('name', 'Modified 1')
      })

      // Switch to skill 2 and modify
      rerender({ selectedItemId: 'skill-2' })

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Skill 2')
      })

      act(() => {
        result.current.updateField('name', 'Modified 2')
      })

      // Discard all
      act(() => {
        result.current.discardAllChanges()
      })

      // Current form should be reset
      expect(result.current.formState.name).toBe('Skill 2')
      expect(result.current.dirtyCount).toBe(0)
    })
  })

  describe('delete operations', () => {
    it('should delete current skill with deleteCurrentSkill', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      await act(async () => {
        await result.current.deleteCurrentSkill()
      })

      expect(mockOnDelete).toHaveBeenCalledWith(skill.id)
    })

    it('should set isDeleting during delete operation', async () => {
      const skill = createTestSkill()
      let resolvePromise: () => void = () => {}
      mockOnDelete.mockImplementation(
        () => new Promise((resolve) => { resolvePromise = resolve })
      )

      const { result } = renderHook(() =>
        usePromptEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await waitFor(() => {
        expect(result.current.formState.name).toBe('Test Skill')
      })

      let deletePromise: Promise<void>
      act(() => {
        deletePromise = result.current.deleteCurrentSkill()
      })

      expect(result.current.isDeleting).toBe(true)

      await act(async () => {
        resolvePromise()
        await deletePromise
      })

      expect(result.current.isDeleting).toBe(false)
    })
  })
})
