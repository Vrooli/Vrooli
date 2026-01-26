/**
 * Tests for skillDisplayService.ts
 *
 * Tests cover:
 * - Displaying skills to XML format
 * - Displaying skills to Markdown format
 * - Displaying skills to JSON format
 * - Preview generation
 * - Validation
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  displaySkills,
  generatePreview,
  validateForDisplay,
  copyToClipboard,
} from './skillDisplayService'
import type { Skill } from '@/types'

// Helper to create a minimal skill for testing
function createTestSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    id: 'test-1',
    name: 'Test Skill',
    description: 'A test description',
    content: '# Test content',
    modes: [],
    tags: [],
    draft: false,
    folder: 'local',
    file: 'test-skill.md',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

describe('displaySkills', () => {
  it('should return empty result for empty array', () => {
    const result = displaySkills([], 'xml')

    expect(result.combined).toBe('')
    expect(result.skillCount).toBe(0)
    expect(result.totalTokens).toBe(0)
  })

  describe('XML format', () => {
    it('should generate valid XML structure', () => {
      const skills = [createTestSkill({ id: '1', name: 'Test', content: 'Hello' })]

      const result = displaySkills(skills, 'xml')

      expect(result.combined).toContain('<skills count="1">')
      expect(result.combined).toContain('</skills>')
    })

    it('should include skill ID and name as attributes', () => {
      const skills = [createTestSkill({ id: 'my-skill', name: 'My Skill' })]

      const result = displaySkills(skills, 'xml')

      expect(result.combined).toContain('id="my-skill"')
      expect(result.combined).toContain('name="My Skill"')
    })

    it('should include content in CDATA section', () => {
      const skills = [createTestSkill({ content: 'Some <special> content' })]

      const result = displaySkills(skills, 'xml')

      expect(result.combined).toContain('<![CDATA[')
      expect(result.combined).toContain('Some <special> content')
      expect(result.combined).toContain(']]>')
    })

    it('should escape XML special characters in attributes', () => {
      const skills = [createTestSkill({ name: 'Test "Skill" & More' })]

      const result = displaySkills(skills, 'xml')

      expect(result.combined).toContain('&quot;')
      expect(result.combined).toContain('&amp;')
    })

    it('should not include extra metadata tags in XML', () => {
      const skills = [createTestSkill({ description: 'My description', tags: ['tag1'], modes: ['coding'] })]

      const result = displaySkills(skills, 'xml')

      expect(result.combined).not.toContain('<description>')
      expect(result.combined).not.toContain('<tags>')
      expect(result.combined).not.toContain('modes=')
    })
  })

  describe('Markdown format', () => {
    it('should generate valid Markdown structure', () => {
      const skills = [createTestSkill({ name: 'Test Skill' })]

      const result = displaySkills(skills, 'markdown')

      expect(result.combined).toContain('# Combined Skills')
      expect(result.combined).toContain('## 1. Test Skill')
    })

    it('should include description as blockquote', () => {
      const skills = [createTestSkill({ description: 'My description' })]

      const result = displaySkills(skills, 'markdown')

      expect(result.combined).toContain('> My description')
    })

    it('should format modes with bold label', () => {
      const skills = [createTestSkill({ modes: ['coding', 'review'] })]

      const result = displaySkills(skills, 'markdown')

      expect(result.combined).toContain('**Modes:**')
      expect(result.combined).toContain('coding / review')
    })

    it('should format tags with code backticks', () => {
      const skills = [createTestSkill({ tags: ['tag1', 'tag2'] })]

      const result = displaySkills(skills, 'markdown')

      expect(result.combined).toContain('**Tags:**')
      expect(result.combined).toContain('`tag1`')
      expect(result.combined).toContain('`tag2`')
    })

    it('should wrap content in code block', () => {
      const skills = [createTestSkill({ content: 'function test() {}' })]

      const result = displaySkills(skills, 'markdown')

      expect(result.combined).toContain('```')
      expect(result.combined).toContain('function test() {}')
    })
  })

  describe('JSON format', () => {
    // Type for parsed JSON output
    interface DisplayJsonOutput {
      combined: boolean
      count: number
      generated: string
      skills: Array<{
        id: string
        name: string
        description: string
        modes: string[]
        tags: string[]
        content: string
      }>
    }

    it('should generate valid JSON', () => {
      const skills = [createTestSkill({ id: '1', name: 'Test' })]

      const result = displaySkills(skills, 'json')

      expect(() => JSON.parse(result.combined) as unknown).not.toThrow()
    })

    it('should include combined flag and count', () => {
      const skills = [createTestSkill(), createTestSkill({ id: '2' })]

      const result = displaySkills(skills, 'json')
      const parsed: DisplayJsonOutput = JSON.parse(result.combined) as DisplayJsonOutput

      expect(parsed.combined).toBe(true)
      expect(parsed.count).toBe(2)
    })

    it('should include all skill fields', () => {
      const skills = [
        createTestSkill({
          id: 'test-id',
          name: 'Test Name',
          description: 'Test Desc',
          modes: ['m1'],
          tags: ['t1'],
          content: 'Test Content',
        }),
      ]

      const result = displaySkills(skills, 'json')
      const parsed: DisplayJsonOutput = JSON.parse(result.combined) as DisplayJsonOutput
      const firstSkill = parsed.skills[0]

      expect(firstSkill).toBeDefined()
      expect(firstSkill?.id).toBe('test-id')
      expect(firstSkill?.name).toBe('Test Name')
      expect(firstSkill?.description).toBe('Test Desc')
      expect(firstSkill?.modes).toEqual(['m1'])
      expect(firstSkill?.tags).toEqual(['t1'])
      expect(firstSkill?.content).toBe('Test Content')
    })
  })

  it('should default to XML format', () => {
    const skills = [createTestSkill()]

    const result = displaySkills(skills)

    expect(result.format).toBe('xml')
    expect(result.combined).toContain('<skills')
  })

  it('should calculate token estimate', () => {
    const skills = [createTestSkill({ content: 'A'.repeat(400) })]

    const result = displaySkills(skills, 'xml')

    // Roughly 4 chars per token
    expect(result.totalTokens).toBeGreaterThan(100)
  })

  it('should report correct skill count', () => {
    const skills = [
      createTestSkill({ id: '1' }),
      createTestSkill({ id: '2' }),
      createTestSkill({ id: '3' }),
    ]

    const result = displaySkills(skills, 'xml')

    expect(result.skillCount).toBe(3)
  })
})

describe('generatePreview', () => {
  it('should return full content if under max length', () => {
    const skills = [createTestSkill({ content: 'Short' })]

    const preview = generatePreview(skills, 'xml', 10000)

    expect(preview).not.toContain('truncated')
  })

  it('should truncate long content', () => {
    const skills = [createTestSkill({ content: 'A'.repeat(1000) })]

    const preview = generatePreview(skills, 'xml', 100)

    expect(preview.length).toBeLessThan(200) // Truncated + indicator
    expect(preview).toContain('truncated')
  })

  it('should use default max length', () => {
    const skills = [createTestSkill()]

    const preview = generatePreview(skills, 'xml')

    expect(typeof preview).toBe('string')
  })
})

describe('validateForDisplay', () => {
  it('should error on empty array', () => {
    const result = validateForDisplay([])

    expect(result.valid).toBe(false)
    expect(result.errors).toContain('No skills selected')
  })

  it('should warn on single skill', () => {
    const result = validateForDisplay([createTestSkill()])

    expect(result.valid).toBe(true)
    expect(result.warnings.length).toBeGreaterThan(0)
    expect(result.warnings.some((w) => w.includes('one skill'))).toBe(true)
  })

  it('should be valid for multiple skills', () => {
    const result = validateForDisplay([
      createTestSkill({ id: '1' }),
      createTestSkill({ id: '2' }),
    ])

    expect(result.valid).toBe(true)
    expect(result.errors).toEqual([])
  })

  it('should warn on empty content', () => {
    const result = validateForDisplay([
      createTestSkill({ id: '1', content: '' }),
      createTestSkill({ id: '2', content: 'Has content' }),
    ])

    expect(result.warnings.some((w) => w.includes('no content'))).toBe(true)
  })

  it('should warn on draft skills', () => {
    const result = validateForDisplay([
      createTestSkill({ id: '1', draft: true, name: 'Draft Skill' }),
      createTestSkill({ id: '2', draft: false }),
    ])

    expect(result.warnings.some((w) => w.includes('drafts'))).toBe(true)
    expect(result.warnings.some((w) => w.includes('Draft Skill'))).toBe(true)
  })

  it('should warn on large displayed size', () => {
    const largeSkills = Array.from({ length: 10 }, (_, i) =>
      createTestSkill({ id: `${i}`, content: 'A'.repeat(10000) })
    )

    const result = validateForDisplay(largeSkills)

    expect(result.warnings.some((w) => w.includes('large'))).toBe(true)
  })
})

describe('copyToClipboard', () => {
  const mockClipboard = {
    writeText: vi.fn(),
  }

  beforeEach(() => {
    vi.stubGlobal('navigator', { clipboard: mockClipboard })
    mockClipboard.writeText.mockReset()
  })

  it('should copy displayed content to clipboard', async () => {
    mockClipboard.writeText.mockResolvedValue(undefined)
    const skills = [createTestSkill()]

    const result = await copyToClipboard(skills, 'xml')

    expect(result.success).toBe(true)
    expect(mockClipboard.writeText).toHaveBeenCalled()
  })

  it('should return error on clipboard failure', async () => {
    mockClipboard.writeText.mockRejectedValue(new Error('Permission denied'))
    const skills = [createTestSkill()]

    const result = await copyToClipboard(skills, 'xml')

    expect(result.success).toBe(false)
    expect(result.error).toBe('Permission denied')
  })
})
