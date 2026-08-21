import { describe, it, expect } from 'vitest'
import {
  SkillSchema,
  SkillArraySchema,
  CreateSkillRequestSchema,
  TagSchema,
} from '../skill.schema'
import { FolderTypeSchema } from '../common.schema'

describe('FolderTypeSchema', () => {
  it('should accept valid folder types', () => {
    expect(FolderTypeSchema.parse('core')).toBe('core')
    expect(FolderTypeSchema.parse('local')).toBe('local')
    expect(FolderTypeSchema.parse('drafts')).toBe('drafts')
  })

  it('should reject invalid folder types', () => {
    expect(FolderTypeSchema.safeParse('invalid').success).toBe(false)
    expect(FolderTypeSchema.safeParse('').success).toBe(false)
    expect(FolderTypeSchema.safeParse(123).success).toBe(false)
  })
})

describe('SkillSchema', () => {
  const minimalSkill = {
    id: 'test-skill',
    file: 'test-skill.md',
    name: 'Test Skill',
    folder: 'core',
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
    usageCount: 0,
  }

  it('should parse a minimal skill with defaults', () => {
    const result = SkillSchema.parse(minimalSkill)

    expect(result.id).toBe('test-skill')
    expect(result.name).toBe('Test Skill')
    expect(result.folder).toBe('core')
    // Verify defaults are applied
    expect(result.description).toBe('')
    expect(result.content).toBe('')
    expect(result.tags).toEqual([])
    expect(result.modes).toEqual([])
    expect(result.draft).toBe(false)
  })

  it('should default an omitted protobuf usage counter to zero', () => {
    const { usageCount: _omitted, ...protobufJson } = minimalSkill

    expect(SkillSchema.parse(protobufJson).usageCount).toBe(0)
  })

  it('should normalize legacy null timestamps without dropping the list item', () => {
    const result = SkillSchema.parse({ ...minimalSkill, createdAt: null, updatedAt: null })

    expect(result.createdAt).toBe('')
    expect(result.updatedAt).toBe('')
  })

  it('should provide defaults for optional arrays - prevents crashes', () => {
    // This tests the key safety feature: when API omits arrays, we get [] not undefined
    const skill = SkillSchema.parse(minimalSkill)

    // These should be arrays, not undefined, so iteration is safe
    expect(Array.isArray(skill.tags)).toBe(true)
    expect(Array.isArray(skill.modes)).toBe(true)
    expect(skill.tags.length).toBe(0)
    expect(skill.modes.length).toBe(0)

    // Safe to iterate without crashing
    expect(() => skill.tags.map((t) => t.toUpperCase())).not.toThrow()
    expect(() => skill.modes.filter((m) => m.includes('x'))).not.toThrow()
  })

  it('should preserve provided array values', () => {
    const skillWithArrays = {
      ...minimalSkill,
      tags: ['tag1', 'tag2'],
      modes: ['agent', 'ide'],
    }

    const result = SkillSchema.parse(skillWithArrays)

    expect(result.tags).toEqual(['tag1', 'tag2'])
    expect(result.modes).toEqual(['agent', 'ide'])
  })

  it('should reject invalid folder type', () => {
    const invalidSkill = {
      ...minimalSkill,
      folder: 'invalid',
    }

    expect(SkillSchema.safeParse(invalidSkill).success).toBe(false)
  })

  it('should reject missing required fields', () => {
    expect(SkillSchema.safeParse({}).success).toBe(false)
    expect(SkillSchema.safeParse({ id: 'test' }).success).toBe(false)
    expect(SkillSchema.safeParse({ id: 'test', name: 'Test' }).success).toBe(false)
  })

  it('should handle nullable fields correctly', () => {
    const skillWithNulls = {
      ...minimalSkill,
      targetToolId: null,
      lastUsed: null,
      effectivenessRating: null,
    }

    const result = SkillSchema.parse(skillWithNulls)

    expect(result.targetToolId).toBeNull()
    expect(result.lastUsed).toBeNull()
    expect(result.effectivenessRating).toBeNull()
  })
})

describe('SkillArraySchema', () => {
  it('should parse an array of skills', () => {
    const skills = [
      {
        id: 'skill-1',
        file: 'skill-1.md',
        name: 'Skill 1',
        folder: 'core',
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
        usageCount: 5,
      },
      {
        id: 'skill-2',
        file: 'skill-2.md',
        name: 'Skill 2',
        folder: 'local',
        createdAt: '2024-01-02T00:00:00Z',
        updatedAt: '2024-01-02T00:00:00Z',
        usageCount: 0,
      },
    ]

    const result = SkillArraySchema.parse(skills)

    expect(result).toHaveLength(2)
    expect(result[0]?.id).toBe('skill-1')
    expect(result[1]?.id).toBe('skill-2')
    // Verify defaults are applied to each skill
    expect(result[0]?.tags).toEqual([])
    expect(result[1]?.tags).toEqual([])
  })

  it('should parse empty array', () => {
    const result = SkillArraySchema.parse([])
    expect(result).toEqual([])
  })
})

describe('CreateSkillRequestSchema', () => {
  it('should validate a valid create request', () => {
    const request = {
      name: 'New Skill',
      description: 'A new skill',
      content: 'Skill content here',
      folder: 'drafts',
    }

    const result = CreateSkillRequestSchema.parse(request)

    expect(result.name).toBe('New Skill')
    expect(result.folder).toBe('drafts')
  })

  it('should reject empty name', () => {
    const request = {
      name: '',
      description: 'A skill',
      content: 'Content',
      folder: 'core',
    }

    const parseResult = CreateSkillRequestSchema.safeParse(request)
    expect(parseResult.success).toBe(false)
  })

  it('should reject empty content', () => {
    const request = {
      name: 'Skill',
      description: 'A skill',
      content: '',
      folder: 'core',
    }

    const parseResult = CreateSkillRequestSchema.safeParse(request)
    expect(parseResult.success).toBe(false)
  })

  it('should reject invalid folder', () => {
    const request = {
      name: 'Skill',
      description: 'A skill',
      content: 'Content',
      folder: 'invalid',
    }

    const parseResult = CreateSkillRequestSchema.safeParse(request)
    expect(parseResult.success).toBe(false)
  })
})

describe('TagSchema', () => {
  it('should parse a minimal tag', () => {
    const tag = {
      id: 'tag-1',
      name: 'Test Tag',
    }

    const result = TagSchema.parse(tag)

    expect(result.id).toBe('tag-1')
    expect(result.name).toBe('Test Tag')
    expect(result.color).toBeUndefined()
    expect(result.description).toBeUndefined()
  })

  it('should parse a tag with all fields', () => {
    const tag = {
      id: 'tag-2',
      name: 'Full Tag',
      color: '#FF5733',
      description: 'A fully specified tag',
    }

    const result = TagSchema.parse(tag)

    expect(result.id).toBe('tag-2')
    expect(result.color).toBe('#FF5733')
    expect(result.description).toBe('A fully specified tag')
  })
})
