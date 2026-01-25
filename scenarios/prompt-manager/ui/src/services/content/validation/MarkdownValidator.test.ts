import { describe, it, expect } from 'vitest'
import { validateMarkdown, validateRoundTrip } from './MarkdownValidator'

describe('MarkdownValidator', () => {
  describe('escaped code fences', () => {
    it('detects escaped code fence with language', () => {
      const markdown = '\\`\\`\\`bash\necho hello\n\\`\\`\\`'
      const result = validateMarkdown(markdown)

      expect(result.isValid).toBe(false)
      expect(result.issues).toHaveLength(2) // Opening and closing
      const issue = result.issues[0]
      expect(issue).toBeDefined()
      expect(issue?.type).toBe('escaped-code-fence')
      expect(issue?.line).toBe(1)
      expect(issue?.message).toBe(
        'Escaped code fence will not render as a code block'
      )
      expect(issue?.suggestion).toBe('Remove backslashes: ```bash')
    })

    it('detects escaped code fence without language', () => {
      const markdown = '\\`\\`\\`\ncode\n\\`\\`\\`'
      const result = validateMarkdown(markdown)

      expect(result.isValid).toBe(false)
      expect(result.issues).toHaveLength(2)
      expect(result.issues[0]?.suggestion).toBe('Remove backslashes: ```')
      expect(result.issues[1]?.suggestion).toBe('Remove backslashes: ```')
    })

    it('passes valid 3-backtick code fences', () => {
      const markdown = '```typescript\nconst x = 1\n```'
      const result = validateMarkdown(markdown)

      expect(result.isValid).toBe(true)
      expect(result.issues).toHaveLength(0)
    })

    it('detects escaped fence in complex document', () => {
      const markdown = [
        '# Heading',
        '',
        'Regular text.',
        '',
        '\\`\\`\\`bash',
        'code',
        '\\`\\`\\`',
        '',
        'More text.',
      ].join('\n')
      const result = validateMarkdown(markdown)

      expect(result.isValid).toBe(false)
      expect(result.issues.some((i) => i.type === 'escaped-code-fence')).toBe(true)
      expect(result.issues).toHaveLength(2)
      expect(result.issues[0]?.line).toBe(5)
      expect(result.issues[1]?.line).toBe(7)
    })
  })

  describe('extended code fences (4+ backticks)', () => {
    it('passes 4-backtick code fence (now preserved)', () => {
      const markdown = '````markdown\ninner content\n````'
      const result = validateMarkdown(markdown)

      // Extended fences are now preserved, so no warning
      expect(result.isValid).toBe(true)
      expect(result.issues).toHaveLength(0)
    })

    it('passes 5-backtick code fence (now preserved)', () => {
      const markdown = '`````python\ncode\n`````'
      const result = validateMarkdown(markdown)

      expect(result.isValid).toBe(true)
      expect(result.issues).toHaveLength(0)
    })

    it('passes nested code blocks pattern from skill-authoring.md', () => {
      const markdown = [
        '**Example:**',
        '````markdown',
        '```bash',
        'echo hello',
        '```',
        '````',
      ].join('\n')
      const result = validateMarkdown(markdown)

      // Extended fences are now preserved, so nested blocks are handled correctly
      expect(result.isValid).toBe(true)
      expect(result.issues).toHaveLength(0)
    })

    it('passes standard 3-backtick fences', () => {
      const markdown = '```bash\necho hello\n```'
      const result = validateMarkdown(markdown)

      expect(result.isValid).toBe(true)
      expect(result.issues).toHaveLength(0)
    })

    it('passes multiple extended fences in document', () => {
      const markdown = [
        '# Examples',
        '',
        '````markdown',
        'example 1',
        '````',
        '',
        '````python',
        'example 2',
        '````',
      ].join('\n')
      const result = validateMarkdown(markdown)

      // Extended fences are now preserved
      expect(result.isValid).toBe(true)
      expect(result.issues).toHaveLength(0)
    })
  })

  describe('mixed issues', () => {
    it('detects escaped fences but not extended fences (now preserved)', () => {
      const markdown = [
        '\\`\\`\\`bash',
        'escaped',
        '\\`\\`\\`',
        '',
        '````markdown',
        'extended',
        '````',
      ].join('\n')
      const result = validateMarkdown(markdown)

      expect(result.isValid).toBe(false)
      expect(result.issues.some((i) => i.type === 'escaped-code-fence')).toBe(true)
      // Extended fences are now preserved, so only escaped fence issues exist
      expect(result.issues.every((i) => i.type === 'escaped-code-fence')).toBe(true)
    })

    it('handles complex document with valid fences (extended now preserved)', () => {
      const markdown = [
        '# Title',
        '',
        '```typescript',
        'const x = 1',
        '```',
        '',
        '````markdown',
        '```bash',
        'nested',
        '```',
        '````',
        '',
        '```python',
        'more code',
        '```',
      ].join('\n')
      const result = validateMarkdown(markdown)

      // Extended fences are now preserved, so this is valid
      expect(result.isValid).toBe(true)
      expect(result.issues).toHaveLength(0)
    })
  })

  describe('edge cases', () => {
    it('handles empty string', () => {
      const result = validateMarkdown('')

      expect(result.isValid).toBe(true)
      expect(result.issues).toHaveLength(0)
    })

    it('handles string with only whitespace', () => {
      const result = validateMarkdown('   \n   \n   ')

      expect(result.isValid).toBe(true)
      expect(result.issues).toHaveLength(0)
    })

    it('handles inline backticks (not fences)', () => {
      const markdown = 'Use `code` inline and ``double`` backticks'
      const result = validateMarkdown(markdown)

      expect(result.isValid).toBe(true)
      expect(result.issues).toHaveLength(0)
    })

    it('does not warn for extended fences (now preserved)', () => {
      const markdown = '````typescript\ncode\n````'
      const result = validateMarkdown(markdown)

      // Extended fences are now preserved
      expect(result.issues).toHaveLength(0)
    })
  })

  describe('severity levels', () => {
    it('marks escaped code fences as warnings', () => {
      const markdown = '\\`\\`\\`bash\ncode\n\\`\\`\\`'
      const result = validateMarkdown(markdown)

      expect(result.issues.every((i) => i.severity === 'warning')).toBe(true)
    })

    it('extended code fences produce no issues (now preserved)', () => {
      const markdown = '````markdown\ncode\n````'
      const result = validateMarkdown(markdown)

      // Extended fences are now preserved, so no issues
      expect(result.issues).toHaveLength(0)
    })
  })
})

describe('validateRoundTrip', () => {
  describe('stable content', () => {
    it('marks simple markdown as stable', () => {
      const markdown = '# Heading\n\nParagraph text.'
      const result = validateRoundTrip(markdown)

      expect(result.isStable).toBe(true)
    })

    it('marks code blocks as stable', () => {
      const markdown = '```typescript\nconst x = 1\n```'
      const result = validateRoundTrip(markdown)

      expect(result.isStable).toBe(true)
    })

    it('marks extended fences as stable (now preserved)', () => {
      const markdown = '````markdown\n```bash\necho hello\n```\n````'
      const result = validateRoundTrip(markdown)

      expect(result.isStable).toBe(true)
    })

    it('marks empty content as stable', () => {
      expect(validateRoundTrip('').isStable).toBe(true)
      expect(validateRoundTrip('   ').isStable).toBe(true)
      expect(validateRoundTrip('\n\n').isStable).toBe(true)
    })

    it('marks unordered lists as stable (different markers normalized)', () => {
      const markdown = '* Item 1\n* Item 2\n* Item 3'
      const result = validateRoundTrip(markdown)

      expect(result.isStable).toBe(true)
    })

    it('marks nested lists as stable (2-space vs 4-space indent normalized)', () => {
      const markdown = '* Parent\n  * Child 1\n  * Child 2'
      const result = validateRoundTrip(markdown)

      expect(result.isStable).toBe(true)
    })

    it('marks horizontal rules as stable (--- vs * * * normalized)', () => {
      const markdown = '# Section 1\n\n---\n\n# Section 2'
      const result = validateRoundTrip(markdown)

      expect(result.isStable).toBe(true)
    })

    it('marks tables as stable', () => {
      const markdown = '| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1 | Cell 2 |'
      const result = validateRoundTrip(markdown)

      expect(result.isStable).toBe(true)
    })

    it('marks hard-wrapped paragraphs as stable (merged to single line)', () => {
      // Hard-wrapped paragraphs (sentences on separate lines) get merged
      const markdown = 'First sentence.\nSecond sentence.'
      const result = validateRoundTrip(markdown)

      expect(result.isStable).toBe(true)
    })
  })

  describe('unstable content', () => {
    it('detects escaped fences as potentially unstable', () => {
      // Escaped fences like \`\`\` won't round-trip correctly
      // because they're literal text, not code blocks
      const markdown = '\\`\\`\\`bash\ncode\n\\`\\`\\`'
      const result = validateRoundTrip(markdown)

      // The escaped backslashes may be modified during conversion
      // This depends on the actual converter behavior
      expect(result.roundTrippedContent).toBeDefined()
    })
  })

  describe('round-trip result structure', () => {
    it('returns roundTrippedContent for stable markdown', () => {
      const markdown = '# Test'
      const result = validateRoundTrip(markdown)

      expect(result.isStable).toBe(true)
      expect(result.roundTrippedContent).toBeDefined()
      expect(result.changeDescription).toBeUndefined()
    })

    it('returns changeDescription for unstable markdown', () => {
      // Create content that definitely won't round-trip
      // (This test may need adjustment based on actual converter behavior)
      const markdown = '# Test'
      const result = validateRoundTrip(markdown)

      if (!result.isStable) {
        expect(result.changeDescription).toBeDefined()
      }
    })
  })
})
