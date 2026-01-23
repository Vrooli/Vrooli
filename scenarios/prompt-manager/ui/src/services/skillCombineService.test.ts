/**
 * Tests for promptCombineService.ts
 *
 * Tests cover:
 * - Combining prompts to XML format
 * - Combining prompts to Markdown format
 * - Combining prompts to JSON format
 * - Preview generation
 * - Validation
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  combinePrompts,
  generatePreview,
  validateForCombine,
  copyToClipboard,
} from './promptCombineService'
import type { Prompt } from '@/types'

// Helper to create a minimal prompt for testing
function createTestPrompt(overrides: Partial<Prompt> = {}): Prompt {
  return {
    id: 'test-1',
    name: 'Test Prompt',
    description: 'A test description',
    content: '# Test content',
    modes: [],
    tags: [],
    draft: false,
    folder: 'local',
    createdAt: '2025-01-01T00:00:00Z',
    updatedAt: '2025-01-01T00:00:00Z',
    usageCount: 0,
    ...overrides,
  }
}

describe('combinePrompts', () => {
  it('should return empty result for empty array', () => {
    const result = combinePrompts([], 'xml')

    expect(result.combined).toBe('')
    expect(result.promptCount).toBe(0)
    expect(result.totalTokens).toBe(0)
  })

  describe('XML format', () => {
    it('should generate valid XML structure', () => {
      const prompts = [createTestPrompt({ id: '1', name: 'Test', content: 'Hello' })]

      const result = combinePrompts(prompts, 'xml')

      expect(result.combined).toContain('<?xml version="1.0"')
      expect(result.combined).toContain('<combined-prompts')
      expect(result.combined).toContain('</combined-prompts>')
    })

    it('should include prompt ID and name as attributes', () => {
      const prompts = [createTestPrompt({ id: 'my-prompt', name: 'My Prompt' })]

      const result = combinePrompts(prompts, 'xml')

      expect(result.combined).toContain('id="my-prompt"')
      expect(result.combined).toContain('name="My Prompt"')
    })

    it('should include content in CDATA section', () => {
      const prompts = [createTestPrompt({ content: 'Some <special> content' })]

      const result = combinePrompts(prompts, 'xml')

      expect(result.combined).toContain('<![CDATA[')
      expect(result.combined).toContain('Some <special> content')
      expect(result.combined).toContain(']]>')
    })

    it('should escape XML special characters in attributes', () => {
      const prompts = [createTestPrompt({ name: 'Test "Prompt" & More' })]

      const result = combinePrompts(prompts, 'xml')

      expect(result.combined).toContain('&quot;')
      expect(result.combined).toContain('&amp;')
    })

    it('should include modes when present', () => {
      const prompts = [createTestPrompt({ modes: ['coding', 'review'] })]

      const result = combinePrompts(prompts, 'xml')

      expect(result.combined).toContain('modes="coding/review"')
    })

    it('should include description when present', () => {
      const prompts = [createTestPrompt({ description: 'My description' })]

      const result = combinePrompts(prompts, 'xml')

      expect(result.combined).toContain('<description>My description</description>')
    })

    it('should include tags when present', () => {
      const prompts = [createTestPrompt({ tags: ['tag1', 'tag2'] })]

      const result = combinePrompts(prompts, 'xml')

      expect(result.combined).toContain('<tags>tag1, tag2</tags>')
    })
  })

  describe('Markdown format', () => {
    it('should generate valid Markdown structure', () => {
      const prompts = [createTestPrompt({ name: 'Test Prompt' })]

      const result = combinePrompts(prompts, 'markdown')

      expect(result.combined).toContain('# Combined Prompts')
      expect(result.combined).toContain('## 1. Test Prompt')
    })

    it('should include description as blockquote', () => {
      const prompts = [createTestPrompt({ description: 'My description' })]

      const result = combinePrompts(prompts, 'markdown')

      expect(result.combined).toContain('> My description')
    })

    it('should format modes with bold label', () => {
      const prompts = [createTestPrompt({ modes: ['coding', 'review'] })]

      const result = combinePrompts(prompts, 'markdown')

      expect(result.combined).toContain('**Modes:**')
      expect(result.combined).toContain('coding / review')
    })

    it('should format tags with code backticks', () => {
      const prompts = [createTestPrompt({ tags: ['tag1', 'tag2'] })]

      const result = combinePrompts(prompts, 'markdown')

      expect(result.combined).toContain('**Tags:**')
      expect(result.combined).toContain('`tag1`')
      expect(result.combined).toContain('`tag2`')
    })

    it('should wrap content in code block', () => {
      const prompts = [createTestPrompt({ content: 'function test() {}' })]

      const result = combinePrompts(prompts, 'markdown')

      expect(result.combined).toContain('```')
      expect(result.combined).toContain('function test() {}')
    })
  })

  describe('JSON format', () => {
    // Type for parsed JSON output
    interface CombinedJsonOutput {
      combined: boolean
      count: number
      generated: string
      prompts: Array<{
        id: string
        name: string
        description: string
        modes: string[]
        tags: string[]
        content: string
      }>
    }

    it('should generate valid JSON', () => {
      const prompts = [createTestPrompt({ id: '1', name: 'Test' })]

      const result = combinePrompts(prompts, 'json')

      expect(() => JSON.parse(result.combined) as unknown).not.toThrow()
    })

    it('should include combined flag and count', () => {
      const prompts = [createTestPrompt(), createTestPrompt({ id: '2' })]

      const result = combinePrompts(prompts, 'json')
      const parsed: CombinedJsonOutput = JSON.parse(result.combined) as CombinedJsonOutput

      expect(parsed.combined).toBe(true)
      expect(parsed.count).toBe(2)
    })

    it('should include all prompt fields', () => {
      const prompts = [
        createTestPrompt({
          id: 'test-id',
          name: 'Test Name',
          description: 'Test Desc',
          modes: ['m1'],
          tags: ['t1'],
          content: 'Test Content',
        }),
      ]

      const result = combinePrompts(prompts, 'json')
      const parsed: CombinedJsonOutput = JSON.parse(result.combined) as CombinedJsonOutput
      const firstPrompt = parsed.prompts[0]

      expect(firstPrompt).toBeDefined()
      expect(firstPrompt?.id).toBe('test-id')
      expect(firstPrompt?.name).toBe('Test Name')
      expect(firstPrompt?.description).toBe('Test Desc')
      expect(firstPrompt?.modes).toEqual(['m1'])
      expect(firstPrompt?.tags).toEqual(['t1'])
      expect(firstPrompt?.content).toBe('Test Content')
    })
  })

  it('should default to XML format', () => {
    const prompts = [createTestPrompt()]

    const result = combinePrompts(prompts)

    expect(result.format).toBe('xml')
    expect(result.combined).toContain('<?xml')
  })

  it('should calculate token estimate', () => {
    const prompts = [createTestPrompt({ content: 'A'.repeat(400) })]

    const result = combinePrompts(prompts, 'xml')

    // Roughly 4 chars per token
    expect(result.totalTokens).toBeGreaterThan(100)
  })

  it('should report correct prompt count', () => {
    const prompts = [
      createTestPrompt({ id: '1' }),
      createTestPrompt({ id: '2' }),
      createTestPrompt({ id: '3' }),
    ]

    const result = combinePrompts(prompts, 'xml')

    expect(result.promptCount).toBe(3)
  })
})

describe('generatePreview', () => {
  it('should return full content if under max length', () => {
    const prompts = [createTestPrompt({ content: 'Short' })]

    const preview = generatePreview(prompts, 'xml', 10000)

    expect(preview).not.toContain('truncated')
  })

  it('should truncate long content', () => {
    const prompts = [createTestPrompt({ content: 'A'.repeat(1000) })]

    const preview = generatePreview(prompts, 'xml', 100)

    expect(preview.length).toBeLessThan(200) // Truncated + indicator
    expect(preview).toContain('truncated')
  })

  it('should use default max length', () => {
    const prompts = [createTestPrompt()]

    const preview = generatePreview(prompts, 'xml')

    expect(typeof preview).toBe('string')
  })
})

describe('validateForCombine', () => {
  it('should error on empty array', () => {
    const result = validateForCombine([])

    expect(result.valid).toBe(false)
    expect(result.errors).toContain('No prompts selected')
  })

  it('should warn on single prompt', () => {
    const result = validateForCombine([createTestPrompt()])

    expect(result.valid).toBe(true)
    expect(result.warnings.length).toBeGreaterThan(0)
    expect(result.warnings.some((w) => w.includes('one prompt'))).toBe(true)
  })

  it('should be valid for multiple prompts', () => {
    const result = validateForCombine([
      createTestPrompt({ id: '1' }),
      createTestPrompt({ id: '2' }),
    ])

    expect(result.valid).toBe(true)
    expect(result.errors).toEqual([])
  })

  it('should warn on empty content', () => {
    const result = validateForCombine([
      createTestPrompt({ id: '1', content: '' }),
      createTestPrompt({ id: '2', content: 'Has content' }),
    ])

    expect(result.warnings.some((w) => w.includes('no content'))).toBe(true)
  })

  it('should warn on draft prompts', () => {
    const result = validateForCombine([
      createTestPrompt({ id: '1', draft: true, name: 'Draft Prompt' }),
      createTestPrompt({ id: '2', draft: false }),
    ])

    expect(result.warnings.some((w) => w.includes('drafts'))).toBe(true)
    expect(result.warnings.some((w) => w.includes('Draft Prompt'))).toBe(true)
  })

  it('should warn on large combined size', () => {
    const largePrompts = Array.from({ length: 10 }, (_, i) =>
      createTestPrompt({ id: `${i}`, content: 'A'.repeat(10000) })
    )

    const result = validateForCombine(largePrompts)

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

  it('should copy combined content to clipboard', async () => {
    mockClipboard.writeText.mockResolvedValue(undefined)
    const prompts = [createTestPrompt()]

    const result = await copyToClipboard(prompts, 'xml')

    expect(result.success).toBe(true)
    expect(mockClipboard.writeText).toHaveBeenCalled()
  })

  it('should return error on clipboard failure', async () => {
    mockClipboard.writeText.mockRejectedValue(new Error('Permission denied'))
    const prompts = [createTestPrompt()]

    const result = await copyToClipboard(prompts, 'xml')

    expect(result.success).toBe(false)
    expect(result.error).toBe('Permission denied')
  })
})
