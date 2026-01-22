/**
 * Tests for editorService.ts
 *
 * Tests cover:
 * - Converting between Prompt and PromptFormState
 * - Tag parsing and formatting
 * - Form validation
 * - Dirty state detection
 */

import { describe, it, expect } from 'vitest'
import {
  promptToFormState,
  formStateToUpdateRequest,
  parseTags,
  formatTags,
  validateFormState,
  isDirty,
  createEmptyFormState,
  getChangeSummary,
} from './editorService'
import type { Prompt } from '@/types'
import type { PromptFormState } from '@/types/editor'

// Helper to create a minimal prompt for testing
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

describe('promptToFormState', () => {
  it('should convert a prompt to form state with all fields', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)

    expect(formState.name).toBe('Test Prompt')
    expect(formState.description).toBe('A test description')
    expect(formState.content).toBe('# Test content')
    expect(formState.modes).toEqual(['development', 'testing'])
    expect(formState.tags).toBe('tag1, tag2')
    expect(formState.icon).toBe('file')
    expect(formState.targetToolId).toBe('tool-123')
    expect(formState.draft).toBe(false)
  })

  it('should handle undefined icon and targetToolId', () => {
    const prompt = createTestPrompt({ icon: undefined, targetToolId: undefined })
    const formState = promptToFormState(prompt)

    expect(formState.icon).toBe('')
    expect(formState.targetToolId).toBe('')
  })

  it('should handle null targetToolId', () => {
    const prompt = createTestPrompt({ targetToolId: null })
    const formState = promptToFormState(prompt)

    expect(formState.targetToolId).toBe('')
  })

  it('should create a copy of modes array', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)

    // Modify the form state modes
    formState.modes.push('new-mode')

    // Original prompt should not be affected
    expect(prompt.modes).toEqual(['development', 'testing'])
  })
})

describe('formStateToUpdateRequest', () => {
  it('should convert form state to update request', () => {
    const formState: PromptFormState = {
      name: 'Updated Name',
      description: 'Updated description',
      content: 'Updated content',
      modes: ['mode1', 'mode2'],
      tags: 'tag1, tag2, tag3',
      icon: 'star',
      targetToolId: 'new-tool',
      draft: true,
      folder: 'internal',
    }

    const request = formStateToUpdateRequest(formState)

    expect(request.name).toBe('Updated Name')
    expect(request.description).toBe('Updated description')
    expect(request.content).toBe('Updated content')
    expect(request.modes).toEqual(['mode1', 'mode2'])
    expect(request.tags).toEqual(['tag1', 'tag2', 'tag3'])
    expect(request.icon).toBe('star')
    expect(request.targetToolId).toBe('new-tool')
    expect(request.draft).toBe(true)
  })

  it('should convert empty icon to undefined', () => {
    const formState: PromptFormState = {
      name: 'Test',
      description: '',
      content: 'Content',
      modes: [],
      tags: '',
      icon: '',
      targetToolId: '',
      draft: false,
      folder: 'internal',
    }

    const request = formStateToUpdateRequest(formState)

    expect(request.icon).toBeUndefined()
    expect(request.targetToolId).toBeUndefined()
  })
})

describe('parseTags', () => {
  it('should parse comma-separated tags', () => {
    expect(parseTags('tag1, tag2, tag3')).toEqual(['tag1', 'tag2', 'tag3'])
  })

  it('should trim whitespace from tags', () => {
    expect(parseTags('  tag1  ,  tag2  ,  tag3  ')).toEqual(['tag1', 'tag2', 'tag3'])
  })

  it('should filter out empty tags', () => {
    expect(parseTags('tag1, , tag2, , , tag3')).toEqual(['tag1', 'tag2', 'tag3'])
  })

  it('should handle empty string', () => {
    expect(parseTags('')).toEqual([])
  })

  it('should handle single tag', () => {
    expect(parseTags('single')).toEqual(['single'])
  })

  it('should handle tags with only whitespace between commas', () => {
    expect(parseTags(',   , ,, ')).toEqual([])
  })
})

describe('formatTags', () => {
  it('should format tags as comma-separated string', () => {
    expect(formatTags(['tag1', 'tag2', 'tag3'])).toBe('tag1, tag2, tag3')
  })

  it('should handle empty array', () => {
    expect(formatTags([])).toBe('')
  })

  it('should handle single tag', () => {
    expect(formatTags(['single'])).toBe('single')
  })
})

describe('validateFormState', () => {
  it('should pass validation for valid form state', () => {
    const formState: PromptFormState = {
      name: 'Valid Name',
      description: 'Valid description',
      content: 'Valid content',
      modes: ['mode1'],
      tags: 'tag1',
      icon: 'file',
      targetToolId: '',
      draft: false,
      folder: 'internal',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(true)
    expect(Object.keys(result.errors)).toHaveLength(0)
  })

  it('should require name', () => {
    const formState: PromptFormState = {
      name: '',
      description: '',
      content: 'Content',
      modes: [],
      tags: '',
      icon: '',
      targetToolId: '',
      draft: false,
      folder: 'internal',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(result.errors.name).toBe('Name is required')
  })

  it('should reject whitespace-only name', () => {
    const formState: PromptFormState = {
      name: '   ',
      description: '',
      content: 'Content',
      modes: [],
      tags: '',
      icon: '',
      targetToolId: '',
      draft: false,
      folder: 'internal',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(result.errors.name).toBe('Name is required')
  })

  it('should reject name longer than 100 characters', () => {
    const formState: PromptFormState = {
      name: 'a'.repeat(101),
      description: '',
      content: 'Content',
      modes: [],
      tags: '',
      icon: '',
      targetToolId: '',
      draft: false,
      folder: 'internal',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(result.errors.name).toBe('Name must be 100 characters or less')
  })

  it('should accept name with exactly 100 characters', () => {
    const formState: PromptFormState = {
      name: 'a'.repeat(100),
      description: '',
      content: 'Content',
      modes: [],
      tags: '',
      icon: '',
      targetToolId: '',
      draft: false,
      folder: 'internal',
    }

    const result = validateFormState(formState)

    expect(result.errors.name).toBeUndefined()
  })

  it('should reject description longer than 500 characters', () => {
    const formState: PromptFormState = {
      name: 'Valid',
      description: 'a'.repeat(501),
      content: 'Content',
      modes: [],
      tags: '',
      icon: '',
      targetToolId: '',
      draft: false,
      folder: 'internal',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(result.errors.description).toBe('Description must be 500 characters or less')
  })

  it('should require content', () => {
    const formState: PromptFormState = {
      name: 'Valid',
      description: '',
      content: '',
      modes: [],
      tags: '',
      icon: '',
      targetToolId: '',
      draft: false,
      folder: 'internal',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(result.errors.content).toBe('Content is required')
  })

  it('should reject whitespace-only content', () => {
    const formState: PromptFormState = {
      name: 'Valid',
      description: '',
      content: '   \n\t  ',
      modes: [],
      tags: '',
      icon: '',
      targetToolId: '',
      draft: false,
      folder: 'internal',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(result.errors.content).toBe('Content is required')
  })

  it('should return multiple errors when multiple fields are invalid', () => {
    const formState: PromptFormState = {
      name: '',
      description: 'a'.repeat(501),
      content: '',
      modes: [],
      tags: '',
      icon: '',
      targetToolId: '',
      draft: false,
      folder: 'internal',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(Object.keys(result.errors)).toHaveLength(3)
    expect(result.errors.name).toBeDefined()
    expect(result.errors.description).toBeDefined()
    expect(result.errors.content).toBeDefined()
  })
})

describe('isDirty', () => {
  it('should return false when form state matches prompt', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)

    expect(isDirty(prompt, formState)).toBe(false)
  })

  it('should detect name change', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.name = 'Changed Name'

    expect(isDirty(prompt, formState)).toBe(true)
  })

  it('should detect description change', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.description = 'Changed description'

    expect(isDirty(prompt, formState)).toBe(true)
  })

  it('should detect content change', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.content = 'Changed content'

    expect(isDirty(prompt, formState)).toBe(true)
  })

  it('should detect draft change', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.draft = !formState.draft

    expect(isDirty(prompt, formState)).toBe(true)
  })

  it('should detect icon change', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.icon = 'new-icon'

    expect(isDirty(prompt, formState)).toBe(true)
  })

  it('should detect targetToolId change', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.targetToolId = 'new-tool-id'

    expect(isDirty(prompt, formState)).toBe(true)
  })

  it('should detect tag addition', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.tags = 'tag1, tag2, tag3'

    expect(isDirty(prompt, formState)).toBe(true)
  })

  it('should detect tag removal', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.tags = 'tag1'

    expect(isDirty(prompt, formState)).toBe(true)
  })

  it('should detect tag reordering', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.tags = 'tag2, tag1'

    expect(isDirty(prompt, formState)).toBe(true)
  })

  it('should detect mode addition', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.modes = [...formState.modes, 'new-mode']

    expect(isDirty(prompt, formState)).toBe(true)
  })

  it('should detect mode removal', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.modes = ['development']

    expect(isDirty(prompt, formState)).toBe(true)
  })

  it('should handle undefined icon in prompt vs empty string in form', () => {
    const prompt = createTestPrompt({ icon: undefined })
    const formState = promptToFormState(prompt)

    // Form state should have empty string, and this should not be considered dirty
    expect(isDirty(prompt, formState)).toBe(false)
  })
})

describe('createEmptyFormState', () => {
  it('should create empty form state with correct defaults', () => {
    const formState = createEmptyFormState()

    expect(formState.name).toBe('')
    expect(formState.description).toBe('')
    expect(formState.content).toBe('')
    expect(formState.modes).toEqual([])
    expect(formState.tags).toBe('')
    expect(formState.icon).toBe('')
    expect(formState.targetToolId).toBe('')
    expect(formState.draft).toBe(true)
    expect(formState.folder).toBe('internal')
  })
})

describe('getChangeSummary', () => {
  it('should return empty array when nothing changed', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)

    expect(getChangeSummary(prompt, formState)).toEqual([])
  })

  it('should list changed fields', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.name = 'New Name'
    formState.content = 'New Content'
    formState.draft = true

    const changes = getChangeSummary(prompt, formState)

    expect(changes).toContain('name')
    expect(changes).toContain('content')
    expect(changes).toContain('draft status')
    expect(changes).not.toContain('description')
  })

  it('should include tags when changed', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.tags = 'new-tag'

    const changes = getChangeSummary(prompt, formState)

    expect(changes).toContain('tags')
  })

  it('should include modes when changed', () => {
    const prompt = createTestPrompt()
    const formState = promptToFormState(prompt)
    formState.modes = ['new-mode']

    const changes = getChangeSummary(prompt, formState)

    expect(changes).toContain('modes')
  })
})
