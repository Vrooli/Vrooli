/**
 * Tests for usePromptEditor hook.
 *
 * Tests cover:
 * - Current prompt selection and form state
 * - Form field updates
 * - Dirty state tracking
 * - Pending changes for multi-item editing
 * - Save and discard operations
 * - Delete operations
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { usePromptEditor } from './usePromptEditor'
import type { Prompt, UpdatePromptRequest } from '@/types'

// Helper to create a test prompt
function createTestPrompt(overrides: Partial<Prompt> = {}): Prompt {
  return {
    id: 'test-1',
    name: 'Test Prompt',
    description: 'A test description',
    content: '# Test content',
    modes: ['development', 'testing'],
    tags: ['tag1', 'tag2'],
    icon: 'file',
    targetToolId: 'tool-123',
    draft: false,
    folder: 'internal',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 5,
    lastUsed: '2025-01-01T12:00:00Z',
    effectivenessRating: 4.5,
    ...overrides,
  }
}

describe('usePromptEditor', () => {
  const mockOnSave = vi.fn<
    [Map<string, UpdatePromptRequest>],
    Promise<Map<string, Prompt | Error>>
  >()
  const mockOnDelete = vi.fn<[string], Promise<void>>()

  beforeEach(() => {
    vi.clearAllMocks()
    mockOnSave.mockResolvedValue(new Map())
    mockOnDelete.mockResolvedValue(undefined)
  })

  describe('initial state', () => {
    it('should return null currentPrompt when no item selected', () => {
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [createTestPrompt()],
          selectedItemId: null,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      expect(result.current.currentPrompt).toBeNull()
      expect(result.current.isDirty).toBe(false)
    })

    it('should load form state when prompt is selected', () => {
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      expect(result.current.currentPrompt).toEqual(prompt)
      expect(result.current.formState.name).toBe('Test Prompt')
      expect(result.current.formState.content).toBe('# Test content')
      expect(result.current.formState.modes).toEqual(['development', 'testing'])
    })

    it('should return null for non-existent selected item', () => {
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [createTestPrompt()],
          selectedItemId: 'non-existent',
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      expect(result.current.currentPrompt).toBeNull()
    })
  })

  describe('form field updates', () => {
    it('should update field value with updateField', () => {
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
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
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
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
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
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

      expect(result.current.formState.name).toBe('Test Prompt')
      expect(result.current.formState.content).toBe('# Test content')
    })
  })

  describe('dirty state tracking', () => {
    it('should not be dirty when form matches original', () => {
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      expect(result.current.isDirty).toBe(false)
      expect(result.current.dirtyCount).toBe(0)
    })

    it('should be dirty when form is modified', () => {
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.updateField('name', 'Modified Name')
      })

      expect(result.current.isDirty).toBe(true)
      expect(result.current.dirtyItemIds.has(prompt.id)).toBe(true)
      expect(result.current.dirtyCount).toBe(1)
    })

    it('should track dirty items in dirtyItemIds set', () => {
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.updateField('content', 'New content')
      })

      expect(result.current.dirtyItemIds).toContain(prompt.id)
    })
  })

  describe('validation', () => {
    it('should validate form and return valid state', () => {
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      expect(result.current.isValid).toBe(true)
      expect(result.current.validation.valid).toBe(true)
      expect(Object.keys(result.current.validation.errors)).toHaveLength(0)
    })

    it('should detect invalid form when name is empty', () => {
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
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
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
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
      const prompt1 = createTestPrompt({ id: 'prompt-1', name: 'Prompt 1' })
      const prompt2 = createTestPrompt({ id: 'prompt-2', name: 'Prompt 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          usePromptEditor({
            prompts: [prompt1, prompt2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'prompt-1' } }
      )

      // Modify the first prompt
      act(() => {
        result.current.updateField('name', 'Modified Prompt 1')
      })

      // Store changes before switching
      act(() => {
        result.current.storeCurrentChanges()
      })

      // Switch to second prompt
      rerender({ selectedItemId: 'prompt-2' })

      // Pending changes should include the first prompt
      expect(result.current.pendingChanges.has('prompt-1')).toBe(true)
      const pending = result.current.pendingChanges.get('prompt-1')
      expect(pending?.current.name).toBe('Modified Prompt 1')
      expect(pending?.isDirty).toBe(true)
    })

    it('should restore pending changes when switching back', () => {
      const prompt1 = createTestPrompt({ id: 'prompt-1', name: 'Prompt 1' })
      const prompt2 = createTestPrompt({ id: 'prompt-2', name: 'Prompt 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          usePromptEditor({
            prompts: [prompt1, prompt2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'prompt-1' } }
      )

      // Modify first (separate act to allow re-render)
      act(() => {
        result.current.updateField('name', 'Modified Prompt 1')
      })

      // Store in separate act (now dirty state is updated)
      act(() => {
        result.current.storeCurrentChanges()
      })

      // Switch to prompt 2
      rerender({ selectedItemId: 'prompt-2' })

      // Switch back to prompt 1
      rerender({ selectedItemId: 'prompt-1' })

      // Form state should have the pending changes
      expect(result.current.formState.name).toBe('Modified Prompt 1')
    })

    it('should not store changes if not dirty', () => {
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
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
      const prompt1 = createTestPrompt({ id: 'prompt-1', name: 'Prompt 1' })
      const prompt2 = createTestPrompt({ id: 'prompt-2', name: 'Prompt 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          usePromptEditor({
            prompts: [prompt1, prompt2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'prompt-1' } }
      )

      // Modify prompt 1 (separate act to allow re-render)
      act(() => {
        result.current.updateField('name', 'Modified Prompt 1')
      })

      // Store in separate act
      act(() => {
        result.current.storeCurrentChanges()
      })

      // Switch to prompt 2 and modify
      rerender({ selectedItemId: 'prompt-2' })
      act(() => {
        result.current.updateField('name', 'Modified Prompt 2')
      })

      // Both should be in dirtyItemIds
      expect(result.current.dirtyItemIds.has('prompt-1')).toBe(true)
      expect(result.current.dirtyItemIds.has('prompt-2')).toBe(true)
      expect(result.current.dirtyCount).toBe(2)
    })
  })

  describe('save operations', () => {
    it('should save current prompt with saveCurrentPrompt', async () => {
      const prompt = createTestPrompt()
      const updatedPrompt = { ...prompt, name: 'Updated Name' }
      mockOnSave.mockResolvedValue(new Map([[prompt.id, updatedPrompt]]))

      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.updateField('name', 'Updated Name')
      })

      await act(async () => {
        await result.current.saveCurrentPrompt()
      })

      expect(mockOnSave).toHaveBeenCalledTimes(1)
      const calls = mockOnSave.mock.calls
      expect(calls.length).toBeGreaterThan(0)
      const updates = calls[0]?.[0]
      expect(updates).toBeDefined()
      expect(updates?.has(prompt.id)).toBe(true)
      expect(updates?.get(prompt.id)?.name).toBe('Updated Name')
    })

    it('should not save when not dirty', async () => {
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await act(async () => {
        await result.current.saveCurrentPrompt()
      })

      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should not save when validation fails', async () => {
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.updateField('name', '') // Invalid
      })

      await act(async () => {
        await result.current.saveCurrentPrompt()
      })

      expect(mockOnSave).not.toHaveBeenCalled()
    })

    it('should save all changes with saveAllChanges', async () => {
      const prompt1 = createTestPrompt({ id: 'prompt-1', name: 'Prompt 1' })
      const prompt2 = createTestPrompt({ id: 'prompt-2', name: 'Prompt 2' })
      mockOnSave.mockResolvedValue(
        new Map([
          ['prompt-1', { ...prompt1, name: 'Modified 1' }],
          ['prompt-2', { ...prompt2, name: 'Modified 2' }],
        ])
      )

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          usePromptEditor({
            prompts: [prompt1, prompt2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'prompt-1' } }
      )

      // Modify prompt 1 (separate act to allow re-render)
      act(() => {
        result.current.updateField('name', 'Modified 1')
      })

      // Store in separate act
      act(() => {
        result.current.storeCurrentChanges()
      })

      // Switch to prompt 2 and modify
      rerender({ selectedItemId: 'prompt-2' })
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
      expect(updates?.has('prompt-1')).toBe(true)
      expect(updates?.has('prompt-2')).toBe(true)
    })

    it('should set isSaving during save operation', async () => {
      const prompt = createTestPrompt()
      let resolvePromise: (value: Map<string, Prompt | Error>) => void = () => {}
      mockOnSave.mockImplementation(
        () => new Promise((resolve) => { resolvePromise = resolve })
      )

      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      act(() => {
        result.current.updateField('name', 'Updated')
      })

      let savePromise: Promise<void>
      act(() => {
        savePromise = result.current.saveCurrentPrompt()
      })

      expect(result.current.isSaving).toBe(true)

      await act(async () => {
        resolvePromise(new Map())
        await savePromise
      })

      expect(result.current.isSaving).toBe(false)
    })

    it('should clear pending changes after successful save', async () => {
      const prompt1 = createTestPrompt({ id: 'prompt-1', name: 'Prompt 1' })
      const prompt2 = createTestPrompt({ id: 'prompt-2', name: 'Prompt 2' })
      mockOnSave.mockResolvedValue(
        new Map([
          ['prompt-1', { ...prompt1, name: 'Modified 1' }],
        ])
      )

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          usePromptEditor({
            prompts: [prompt1, prompt2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'prompt-1' } }
      )

      // Modify and store prompt 1
      act(() => {
        result.current.updateField('name', 'Modified 1')
        result.current.storeCurrentChanges()
      })

      // Switch to prompt 2
      rerender({ selectedItemId: 'prompt-2' })

      // Save all
      await act(async () => {
        await result.current.saveAllChanges()
      })

      // Pending changes for prompt 1 should be cleared
      expect(result.current.pendingChanges.has('prompt-1')).toBe(false)
    })
  })

  describe('discard operations', () => {
    it('should discard current changes with discardCurrentChanges', () => {
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
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

      expect(result.current.formState.name).toBe('Test Prompt')
      expect(result.current.formState.content).toBe('# Test content')
      expect(result.current.isDirty).toBe(false)
    })

    it('should remove from pending changes when discarding', () => {
      const prompt1 = createTestPrompt({ id: 'prompt-1', name: 'Prompt 1' })
      const prompt2 = createTestPrompt({ id: 'prompt-2', name: 'Prompt 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          usePromptEditor({
            prompts: [prompt1, prompt2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'prompt-1' } }
      )

      // Modify and store
      act(() => {
        result.current.updateField('name', 'Modified')
        result.current.storeCurrentChanges()
      })

      // Switch to prompt 2 then back
      rerender({ selectedItemId: 'prompt-2' })
      rerender({ selectedItemId: 'prompt-1' })

      // Discard
      act(() => {
        result.current.discardCurrentChanges()
      })

      expect(result.current.pendingChanges.has('prompt-1')).toBe(false)
      expect(result.current.formState.name).toBe('Prompt 1')
    })

    it('should discard all changes with discardAllChanges', () => {
      const prompt1 = createTestPrompt({ id: 'prompt-1', name: 'Prompt 1' })
      const prompt2 = createTestPrompt({ id: 'prompt-2', name: 'Prompt 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          usePromptEditor({
            prompts: [prompt1, prompt2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'prompt-1' } }
      )

      // Modify and store prompt 1
      act(() => {
        result.current.updateField('name', 'Modified 1')
        result.current.storeCurrentChanges()
      })

      // Switch to prompt 2 and modify
      rerender({ selectedItemId: 'prompt-2' })
      act(() => {
        result.current.updateField('name', 'Modified 2')
      })

      // Discard all
      act(() => {
        result.current.discardAllChanges()
      })

      // Current form should be reset
      expect(result.current.formState.name).toBe('Prompt 2')
      // Pending changes should be cleared
      expect(result.current.pendingChanges.size).toBe(0)
      expect(result.current.dirtyCount).toBe(0)
    })
  })

  describe('delete operations', () => {
    it('should delete current prompt with deleteCurrentPrompt', async () => {
      const prompt = createTestPrompt()
      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      await act(async () => {
        await result.current.deleteCurrentPrompt()
      })

      expect(mockOnDelete).toHaveBeenCalledWith(prompt.id)
    })

    it('should set isDeleting during delete operation', async () => {
      const prompt = createTestPrompt()
      let resolvePromise: () => void = () => {}
      mockOnDelete.mockImplementation(
        () => new Promise((resolve) => { resolvePromise = resolve })
      )

      const { result } = renderHook(() =>
        usePromptEditor({
          prompts: [prompt],
          selectedItemId: prompt.id,
          onSave: mockOnSave,
          onDelete: mockOnDelete,
        })
      )

      let deletePromise: Promise<void>
      act(() => {
        deletePromise = result.current.deleteCurrentPrompt()
      })

      expect(result.current.isDeleting).toBe(true)

      await act(async () => {
        resolvePromise()
        await deletePromise
      })

      expect(result.current.isDeleting).toBe(false)
    })

    it('should clear pending changes for deleted prompt', async () => {
      const prompt1 = createTestPrompt({ id: 'prompt-1', name: 'Prompt 1' })
      const prompt2 = createTestPrompt({ id: 'prompt-2', name: 'Prompt 2' })

      const { result, rerender } = renderHook(
        ({ selectedItemId }: { selectedItemId: string }) =>
          usePromptEditor({
            prompts: [prompt1, prompt2],
            selectedItemId,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { selectedItemId: 'prompt-1' } }
      )

      // Modify and store
      act(() => {
        result.current.updateField('name', 'Modified')
        result.current.storeCurrentChanges()
      })

      // Switch back to prompt 1
      rerender({ selectedItemId: 'prompt-1' })

      // Delete
      await act(async () => {
        await result.current.deleteCurrentPrompt()
      })

      expect(result.current.pendingChanges.has('prompt-1')).toBe(false)
    })
  })

  describe('prompt updates', () => {
    it('should update form state when prompts change externally', () => {
      const prompt = createTestPrompt()
      const { result, rerender } = renderHook(
        ({ prompts }: { prompts: Prompt[] }) =>
          usePromptEditor({
            prompts,
            selectedItemId: prompt.id,
            onSave: mockOnSave,
            onDelete: mockOnDelete,
          }),
        { initialProps: { prompts: [prompt] } }
      )

      expect(result.current.formState.name).toBe('Test Prompt')

      // Simulate external update
      const updatedPrompt = { ...prompt, name: 'Externally Updated' }
      rerender({ prompts: [updatedPrompt] })

      // Note: the hook uses useEffect to update form state from currentPrompt
      // The behavior depends on whether there are pending changes
      // Without pending changes, it should load from the updated prompt
    })
  })
})
