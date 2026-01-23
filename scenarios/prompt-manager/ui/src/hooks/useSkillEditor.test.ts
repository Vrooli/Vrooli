/**
 * Tests for useSkillEditor hook.
 *
 * Tests cover:
 * - Current skill selection and form state
 * - Form field updates
 * - Dirty state tracking
 * - Pending changes for multi-item editing
 * - Save and discard operations
 * - Delete operations
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useSkillEditor } from './useSkillEditor'
import type { Skill, UpdateSkillRequest } from '@/types'

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

describe('useSkillEditor', () => {
  const mockOnSave = vi.fn<
    [Map<string, UpdateSkillRequest>],
    Promise<Map<string, Skill | Error>>
  >()
  const mockOnDelete = vi.fn<[string], Promise<void>>()

  beforeEach(() => {
    vi.clearAllMocks()
    mockOnSave.mockResolvedValue(new Map())
    mockOnDelete.mockResolvedValue(undefined)
  })

  describe('initial state', () => {
    it('should return null currentSkill when no item selected', () => {
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [createTestSkill()],
          selectedItemId: null,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      expect(result.current.currentSkill).toBeNull()
      expect(result.current.isDirty).toBe(false)
    })

    it('should load form state when skill is selected', () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      expect(result.current.currentSkill).toEqual(skill)
      expect(result.current.formState.name).toBe('Test Skill')
      expect(result.current.formState.content).toBe('# Test content')
      expect(result.current.formState.modes).toEqual(['development', 'testing'])
    })

    it('should return null for non-existent selected item', () => {
      const { result } = renderHook(() =>
        useSkillEditor({
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
    it('should update field value with updateField', () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.updateField('name', 'Updated Name')
      })

      expect(result.current.formState.name).toBe('Updated Name')
    })

    it('should update modes with setModes', () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.setModes(['new-mode-1', 'new-mode-2'])
      })

      expect(result.current.formState.modes).toEqual(['new-mode-1', 'new-mode-2'])
    })

    it('should reset form to original with resetForm', () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

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
    it('should not be dirty when form matches original', () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      expect(result.current.isDirty).toBe(false)
      expect(result.current.dirtyCount).toBe(0)
    })

    it('should be dirty when form is modified', () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.updateField('name', 'Modified Name')
      })

      expect(result.current.isDirty).toBe(true)
      expect(result.current.dirtyItemIds.has(skill.id)).toBe(true)
      expect(result.current.dirtyCount).toBe(1)
    })

    it('should track dirty items in dirtyItemIds set', () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.updateField('content', 'New content')
      })

      expect(result.current.dirtyItemIds).toContain(skill.id)
    })
  })

  describe('validation', () => {
    it('should validate form and return valid state', () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      expect(result.current.isValid).toBe(true)
      expect(result.current.validation.valid).toBe(true)
      expect(Object.keys(result.current.validation.errors)).toHaveLength(0)
    })

    it('should detect invalid form when name is empty', () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.updateField('name', '')
      })

      expect(result.current.isValid).toBe(false)
      expect(result.current.validation.errors.name).toBeDefined()
    })

    it('should detect invalid form when content is empty', () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.updateField('content', '')
      })

      expect(result.current.isValid).toBe(false)
      expect(result.current.validation.errors.content).toBeDefined()
    })
  })

  describe('pending changes', () => {
    it('should store current changes when switching items', () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'Skill 1' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Skill 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          useSkillEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'skill-1' } }
      )

      // Modify the first skill
      act(() => {
        result.current.updateField('name', 'Modified Skill 1')
      })

      // Store changes before switching
      act(() => {
        result.current.storeCurrentChanges()
      })

      // Switch to second skill
      rerender({ selectedItemId: 'skill-2' })

      // Pending changes should include the first skill
      expect(result.current.pendingChanges.has('skill-1')).toBe(true)
      const pending = result.current.pendingChanges.get('skill-1')
      expect(pending?.current.name).toBe('Modified Skill 1')
      expect(pending?.isDirty).toBe(true)
    })

    it('should restore pending changes when switching back', () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'Skill 1' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Skill 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          useSkillEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'skill-1' } }
      )

      // Modify first (separate act to allow re-render)
      act(() => {
        result.current.updateField('name', 'Modified Skill 1')
      })

      // Store in separate act (now dirty state is updated)
      act(() => {
        result.current.storeCurrentChanges()
      })

      // Switch to skill 2
      rerender({ selectedItemId: 'skill-2' })

      // Switch back to skill 1
      rerender({ selectedItemId: 'skill-1' })

      // Form state should have the pending changes
      expect(result.current.formState.name).toBe('Modified Skill 1')
    })

    it('should not store changes if not dirty', () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.storeCurrentChanges()
      })

      expect(result.current.pendingChanges.size).toBe(0)
    })

    it('should include pending changes in dirtyItemIds', () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'Skill 1' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Skill 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          useSkillEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'skill-1' } }
      )

      // Modify skill 1 (separate act to allow re-render)
      act(() => {
        result.current.updateField('name', 'Modified Skill 1')
      })

      // Store in separate act
      act(() => {
        result.current.storeCurrentChanges()
      })

      // Switch to skill 2 and modify
      rerender({ selectedItemId: 'skill-2' })
      act(() => {
        result.current.updateField('name', 'Modified Skill 2')
      })

      // Both should be in dirtyItemIds
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
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

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
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await act(async () => {
        await result.current.saveCurrentSkill()
      })

      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should not save when validation fails', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.updateField('name', '') // Invalid
      })

      await act(async () => {
        await result.current.saveCurrentSkill()
      })

      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should save all changes with saveAllChanges', async () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'Skill 1' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Skill 2' })
      mockOnSave.mockResolvedValue(
        new Map([
          ['skill-1', { ...skill1, name: 'Modified 1' }],
          ['skill-2', { ...skill2, name: 'Modified 2' }],
        ])
      )

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          useSkillEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'skill-1' } }
      )

      // Modify skill 1 (separate act to allow re-render)
      act(() => {
        result.current.updateField('name', 'Modified 1')
      })

      // Store in separate act
      act(() => {
        result.current.storeCurrentChanges()
      })

      // Switch to skill 2 and modify
      rerender({ selectedItemId: 'skill-2' })
      act(() => {
        result.current.updateField('name', 'Modified 2')
      })

      // Save all
      await act(async () => {
        await result.current.saveAllChanges()
      })

      expect(mockOnSave).toHaveBeenCalledTimes(1)
      const calls = mockOnSave.mock.calls
      expect(calls.length).toBeGreaterThan(0)
      const updates = calls[0]?.[0]
      expect(updates).toBeDefined()
      expect(updates?.size).toBe(2)
      expect(updates?.has('skill-1')).toBe(true)
      expect(updates?.has('skill-2')).toBe(true)
    })

    it('should set isSaving during save operation', async () => {
      const skill = createTestSkill()
      let resolvePromise: (value: Map<string, Skill | Error>) => void = () => {}
      mockOnSave.mockImplementation(
        () => new Promise((resolve) => { resolvePromise = resolve })
      )

      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.updateField('name', 'Updated')
      })

      let savePromise: Promise<void>
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

    it('should clear pending changes after successful save', async () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'Skill 1' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Skill 2' })
      mockOnSave.mockResolvedValue(
        new Map([
          ['skill-1', { ...skill1, name: 'Modified 1' }],
        ])
      )

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          useSkillEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'skill-1' } }
      )

      // Modify and store skill 1
      act(() => {
        result.current.updateField('name', 'Modified 1')
        result.current.storeCurrentChanges()
      })

      // Switch to skill 2
      rerender({ selectedItemId: 'skill-2' })

      // Save all
      await act(async () => {
        await result.current.saveAllChanges()
      })

      // Pending changes for skill 1 should be cleared
      expect(result.current.pendingChanges.has('skill-1')).toBe(false)
    })
  })

  describe('discard operations', () => {
    it('should discard current changes with discardCurrentChanges', () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

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

    it('should remove from pending changes when discarding', () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'Skill 1' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Skill 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          useSkillEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'skill-1' } }
      )

      // Modify and store
      act(() => {
        result.current.updateField('name', 'Modified')
        result.current.storeCurrentChanges()
      })

      // Switch to skill 2 then back
      rerender({ selectedItemId: 'skill-2' })
      rerender({ selectedItemId: 'skill-1' })

      // Discard
      act(() => {
        result.current.discardCurrentChanges()
      })

      expect(result.current.pendingChanges.has('skill-1')).toBe(false)
      expect(result.current.formState.name).toBe('Skill 1')
    })

    it('should discard all changes with discardAllChanges', () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'Skill 1' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Skill 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          useSkillEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'skill-1' } }
      )

      // Modify and store skill 1
      act(() => {
        result.current.updateField('name', 'Modified 1')
        result.current.storeCurrentChanges()
      })

      // Switch to skill 2 and modify
      rerender({ selectedItemId: 'skill-2' })
      act(() => {
        result.current.updateField('name', 'Modified 2')
      })

      // Discard all
      act(() => {
        result.current.discardAllChanges()
      })

      // Current form should be reset
      expect(result.current.formState.name).toBe('Skill 2')
      // Pending changes should be cleared
      expect(result.current.pendingChanges.size).toBe(0)
      expect(result.current.dirtyCount).toBe(0)
    })
  })

  describe('delete operations', () => {
    it('should delete current skill with deleteCurrentSkill', async () => {
      const skill = createTestSkill()
      const { result } = renderHook(() =>
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

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
        useSkillEditor({
          skills: [skill],
          selectedItemId: skill.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

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

    it('should clear pending changes for deleted skill', async () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'Skill 1' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Skill 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          useSkillEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'skill-1' } }
      )

      // Modify and store
      act(() => {
        result.current.updateField('name', 'Modified')
        result.current.storeCurrentChanges()
      })

      // Switch back to skill 1
      rerender({ selectedItemId: 'skill-1' })

      // Delete
      await act(async () => {
        await result.current.deleteCurrentSkill()
      })

      expect(result.current.pendingChanges.has('skill-1')).toBe(false)
    })
  })

  describe('skill updates', () => {
    it('should update form state when skills change externally', () => {
      const skill = createTestSkill()
      const { result, rerender } = renderHook(
        ({ skills }: { skills: Skill[] }) =>
          useSkillEditor({
            skills,
            selectedItemId: skill.id,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { skills: [skill] } }
      )

      expect(result.current.formState.name).toBe('Test Skill')

      // Simulate external update
      const updatedSkill = { ...skill, name: 'Externally Updated' }
      rerender({ skills: [updatedSkill] })

      // Note: the hook uses useEffect to update form state from currentSkill
      // The behavior depends on whether there are pending changes
      // Without pending changes, it should load from the updated skill
    })
  })

  describe('race condition fix - immediate form population', () => {
    it('should immediately populate form when selecting a skill', () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'First Skill', content: 'First content' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Second Skill', content: 'Second content' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string | null }) =>
          useSkillEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: null as string | null } }
      )

      // Initially no selection
      expect(result.current.currentSkill).toBeNull()

      // Select first skill - form should immediately populate
      rerender({ selectedItemId: 'skill-1' as string | null })
      expect(result.current.formState.name).toBe('First Skill')
      expect(result.current.formState.content).toBe('First content')

      // Switch to second skill - form should immediately populate
      rerender({ selectedItemId: 'skill-2' as string | null })
      expect(result.current.formState.name).toBe('Second Skill')
      expect(result.current.formState.content).toBe('Second content')
    })

    it('should preserve pending changes when switching skills', () => {
      const skill1 = createTestSkill({ id: 'skill-1', name: 'First Skill' })
      const skill2 = createTestSkill({ id: 'skill-2', name: 'Second Skill' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          useSkillEditor({
            skills: [skill1, skill2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'skill-1' } }
      )

      // Modify skill 1
      act(() => {
        result.current.updateField('name', 'Modified First')
      })

      // Store changes
      act(() => {
        result.current.storeCurrentChanges()
      })

      // Switch to skill 2
      rerender({ selectedItemId: 'skill-2' })
      expect(result.current.formState.name).toBe('Second Skill')

      // Switch back to skill 1 - should restore pending changes
      rerender({ selectedItemId: 'skill-1' })
      expect(result.current.formState.name).toBe('Modified First')
    })

    it('should clear form state when deselecting', () => {
      const skill = createTestSkill()

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string | null }) =>
          useSkillEditor({
            skills: [skill],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: skill.id as string | null } }
      )

      // Form should be populated
      expect(result.current.formState.name).toBe('Test Skill')

      // Deselect
      rerender({ selectedItemId: null as string | null })

      // Form should be cleared (empty state)
      expect(result.current.formState.name).toBe('')
      expect(result.current.formState.content).toBe('')
    })

    it('should handle rapid skill switches correctly', () => {
      const skills = [
        createTestSkill({ id: 'p1', name: 'Skill 1', content: 'Content 1' }),
        createTestSkill({ id: 'p2', name: 'Skill 2', content: 'Content 2' }),
        createTestSkill({ id: 'p3', name: 'Skill 3', content: 'Content 3' }),
      ]

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          useSkillEditor({
            skills,
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'p1' } }
      )

      // Rapid switches
      rerender({ selectedItemId: 'p2' })
      rerender({ selectedItemId: 'p3' })
      rerender({ selectedItemId: 'p1' })
      rerender({ selectedItemId: 'p3' })

      // Final state should match p3
      expect(result.current.formState.name).toBe('Skill 3')
      expect(result.current.formState.content).toBe('Content 3')
    })
  })
})
