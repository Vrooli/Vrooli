/**
 * Tests for editorService.ts
 *
 * Tests cover:
 * - Converting between Skill and SkillFormState
 * - Tag parsing and formatting
 * - Form validation
 * - Dirty state detection
 */

import { describe, it, expect } from 'vitest'
import {
  skillToFormState,
  formStateToUpdateRequest,
  parseTags,
  formatTags,
  validateFormState,
  isDirty,
  createEmptyFormState,
  getChangeSummary,
} from './editorService'
import type { Skill } from '@/types'
import type { SkillFormState } from '@/types/editor'

// Helper to create a minimal skill for testing
function createTestSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'test-1',
    name: 'Test Skill',
    description: 'A test description',
    content: '# Test content',
    modes: ['development', 'testing'],
    tags: ['tag1', 'tag2'],
    icon: 'file',
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

describe('skillToFormState', () => {
  it('should convert a skill to form state with all fields', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)

    expect(formState.name).toBe('Test Skill')
    expect(formState.description).toBe('A test description')
    expect(formState.content).toBe('# Test content')
    expect(formState.modes).toEqual(['development', 'testing'])
    expect(formState.tags).toBe('tag1, tag2')
    expect(formState.icon).toBe('file')
    expect(formState.draft).toBe(false)
    expect(formState.folder).toBe('local')
  })

  it('should handle undefined icon', () => {
    const skill = createTestSkill({ icon: undefined })
    const formState = skillToFormState(skill)

    expect(formState.icon).toBe('')
  })

  it('should create a copy of modes array', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)

    // Modify the form state modes
    formState.modes.push('new-mode')

    // Original skill should not be affected
    expect(skill.modes).toEqual(['development', 'testing'])
  })
})

describe('formStateToUpdateRequest', () => {
  it('should convert form state to update request', () => {
    const formState: SkillFormState = {
      name: 'Updated Name',
      description: 'Updated description',
      content: 'Updated content',
      modes: ['mode1', 'mode2'],
      tags: 'tag1, tag2, tag3',
      icon: 'star',
      draft: true,
      folder: 'local',
    }

    const request = formStateToUpdateRequest(formState)

    expect(request.name).toBe('Updated Name')
    expect(request.description).toBe('Updated description')
    expect(request.content).toBe('Updated content')
    expect(request.modes).toEqual(['mode1', 'mode2'])
    expect(request.tags).toEqual(['tag1', 'tag2', 'tag3'])
    expect(request.icon).toBe('star')
    expect(request.draft).toBe(true)
    expect(request.folder).toBe('local')
  })

  it('should convert empty icon to undefined', () => {
    const formState: SkillFormState = {
      name: 'Test',
      description: '',
      content: 'Content',
      modes: [],
      tags: '',
      icon: '',
      draft: false,
      folder: 'local',
    }

    const request = formStateToUpdateRequest(formState)

    expect(request.icon).toBeUndefined()
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
    const formState: SkillFormState = {
      name: 'Valid Name',
      description: 'Valid description',
      content: 'Valid content',
      modes: ['mode1'],
      tags: 'tag1',
      icon: 'file',
      draft: false,
      folder: 'local',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(true)
    expect(Object.keys(result.errors)).toHaveLength(0)
  })

  it('should require name', () => {
    const formState: SkillFormState = {
      name: '',
      description: '',
      content: 'Content',
      modes: [],
      tags: '',
      icon: '',
      draft: false,
      folder: 'local',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(result.errors.name).toBe('Name is required')
  })

  it('should reject whitespace-only name', () => {
    const formState: SkillFormState = {
      name: '   ',
      description: '',
      content: 'Content',
      modes: [],
      tags: '',
      icon: '',
      draft: false,
      folder: 'local',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(result.errors.name).toBe('Name is required')
  })

  it('should reject name longer than 100 characters', () => {
    const formState: SkillFormState = {
      name: 'a'.repeat(101),
      description: '',
      content: 'Content',
      modes: [],
      tags: '',
      icon: '',
      draft: false,
      folder: 'local',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(result.errors.name).toBe('Name must be 100 characters or less')
  })

  it('should accept name with exactly 100 characters', () => {
    const formState: SkillFormState = {
      name: 'a'.repeat(100),
      description: '',
      content: 'Content',
      modes: [],
      tags: '',
      icon: '',
      draft: false,
      folder: 'local',
    }

    const result = validateFormState(formState)

    expect(result.errors.name).toBeUndefined()
  })

  it('should reject description longer than 500 characters', () => {
    const formState: SkillFormState = {
      name: 'Valid',
      description: 'a'.repeat(501),
      content: 'Content',
      modes: [],
      tags: '',
      icon: '',
      draft: false,
      folder: 'local',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(result.errors.description).toBe('Description must be 500 characters or less')
  })

  it('should require content', () => {
    const formState: SkillFormState = {
      name: 'Valid',
      description: '',
      content: '',
      modes: [],
      tags: '',
      icon: '',
      draft: false,
      folder: 'local',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(result.errors.content).toBe('Content is required')
  })

  it('should reject whitespace-only content', () => {
    const formState: SkillFormState = {
      name: 'Valid',
      description: '',
      content: '   \n\t  ',
      modes: [],
      tags: '',
      icon: '',
      draft: false,
      folder: 'local',
    }

    const result = validateFormState(formState)

    expect(result.valid).toBe(false)
    expect(result.errors.content).toBe('Content is required')
  })

  it('should return multiple errors when multiple fields are invalid', () => {
    const formState: SkillFormState = {
      name: '',
      description: 'a'.repeat(501),
      content: '',
      modes: [],
      tags: '',
      icon: '',
      draft: false,
      folder: 'local',
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
  it('should return false when form state matches skill', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)

    expect(isDirty(skill, formState)).toBe(false)
  })

  it('should detect name change', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.name = 'Changed Name'

    expect(isDirty(skill, formState)).toBe(true)
  })

  it('should detect description change', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.description = 'Changed description'

    expect(isDirty(skill, formState)).toBe(true)
  })

  it('should detect content change', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.content = 'Changed content'

    expect(isDirty(skill, formState)).toBe(true)
  })

  it('should detect draft change', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.draft = !formState.draft

    expect(isDirty(skill, formState)).toBe(true)
  })

  it('should detect icon change', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.icon = 'new-icon'

    expect(isDirty(skill, formState)).toBe(true)
  })

  it('should detect folder change', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.folder = 'core'

    expect(isDirty(skill, formState)).toBe(true)
  })

  it('should detect tag addition', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.tags = 'tag1, tag2, tag3'

    expect(isDirty(skill, formState)).toBe(true)
  })

  it('should detect tag removal', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.tags = 'tag1'

    expect(isDirty(skill, formState)).toBe(true)
  })

  it('should detect tag reordering', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.tags = 'tag2, tag1'

    expect(isDirty(skill, formState)).toBe(true)
  })

  it('should detect mode addition', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.modes = [...formState.modes, 'new-mode']

    expect(isDirty(skill, formState)).toBe(true)
  })

  it('should detect mode removal', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.modes = ['development']

    expect(isDirty(skill, formState)).toBe(true)
  })

  it('should handle undefined icon in skill vs empty string in form', () => {
    const skill = createTestSkill({ icon: undefined })
    const formState = skillToFormState(skill)

    // Form state should have empty string, and this should not be considered dirty
    expect(isDirty(skill, formState)).toBe(false)
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
    expect(formState.draft).toBe(true)
    expect(formState.folder).toBe('local')
  })
})

describe('getChangeSummary', () => {
  it('should return empty array when nothing changed', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)

    expect(getChangeSummary(skill, formState)).toEqual([])
  })

  it('should list changed fields', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.name = 'New Name'
    formState.content = 'New Content'
    formState.draft = true

    const changes = getChangeSummary(skill, formState)

    expect(changes).toContain('name')
    expect(changes).toContain('content')
    expect(changes).toContain('draft status')
    expect(changes).not.toContain('description')
  })

  it('should include tags when changed', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.tags = 'new-tag'

    const changes = getChangeSummary(skill, formState)

    expect(changes).toContain('tags')
  })

  it('should include modes when changed', () => {
    const skill = createTestSkill()
    const formState = skillToFormState(skill)
    formState.modes = ['new-mode']

    const changes = getChangeSummary(skill, formState)

    expect(changes).toContain('modes')
  })
})
