import { describe, it, expect } from 'vitest'
import { validateMarkdown } from './MarkdownValidator'

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
    it('detects 4-backtick code fence', () => {
      const markdown = '````markdown\ninner content\n````'
      const result = validateMarkdown(markdown)

      expect(result.isValid).toBe(false)
      expect(result.issues).toHaveLength(1)
      const issue = result.issues[0]
      expect(issue).toBeDefined()
      expect(issue?.type).toBe('extended-code-fence')
      expect(issue?.line).toBe(1)
      expect(issue?.message).toContain('4 backticks')
      expect(issue?.message).toContain('converted to 3')
    })

    it('detects 5-backtick code fence', () => {
      const markdown = '`````python\ncode\n`````'
      const result = validateMarkdown(markdown)

      expect(result.isValid).toBe(false)
      expect(result.issues).toHaveLength(1)
      const issue = result.issues[0]
      expect(issue).toBeDefined()
      expect(issue?.type).toBe('extended-code-fence')
      expect(issue?.message).toContain('5 backticks')
    })

    it('detects nested code blocks pattern from skill-authoring.md', () => {
      const markdown = [
        '**Example:**',
        '````markdown',
        '```bash',
        'echo hello',
        '```',
        '````',
      ].join('\n')
      const result = validateMarkdown(markdown)

      expect(result.isValid).toBe(false)
      expect(result.issues.some((i) => i.type === 'extended-code-fence')).toBe(true)
    })

    it('passes standard 3-backtick fences', () => {
      const markdown = '```bash\necho hello\n```'
      const result = validateMarkdown(markdown)

      expect(result.isValid).toBe(true)
      expect(result.issues).toHaveLength(0)
    })

    it('handles multiple extended fences in document', () => {
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

      expect(result.isValid).toBe(false)
      expect(result.issues).toHaveLength(2)
      expect(result.issues[0]?.line).toBe(3)
      expect(result.issues[1]?.line).toBe(7)
    })
  })

  describe('mixed issues', () => {
    it('detects both escaped and extended fences', () => {
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
      expect(result.issues.some((i) => i.type === 'extended-code-fence')).toBe(true)
    })

    it('handles complex document with valid and invalid fences', () => {
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

      expect(result.isValid).toBe(false)
      expect(result.issues).toHaveLength(1) // Only the 4-backtick fence
      expect(result.issues[0]?.line).toBe(7)
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

    it('correctly identifies column positions for extended fence', () => {
      const markdown = '````typescript'
      const result = validateMarkdown(markdown)

      expect(result.issues).toHaveLength(1)
      const issue = result.issues[0]
      expect(issue).toBeDefined()
      expect(issue?.column).toBe(1)
      expect(issue?.endColumn).toBe(15) // 4 backticks + 'typescript'
    })
  })

  describe('severity levels', () => {
    it('marks escaped code fences as warnings', () => {
      const markdown = '\\`\\`\\`bash\ncode\n\\`\\`\\`'
      const result = validateMarkdown(markdown)

      expect(result.issues.every((i) => i.severity === 'warning')).toBe(true)
    })

    it('marks extended code fences as warnings', () => {
      const markdown = '````markdown\ncode\n````'
      const result = validateMarkdown(markdown)

      expect(result.issues.every((i) => i.severity === 'warning')).toBe(true)
    })
  })
})
