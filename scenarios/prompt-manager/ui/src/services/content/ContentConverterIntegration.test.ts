/**
 * ContentConverter Integration Tests with Real TipTap
 *
 * These tests verify the FULL content conversion flow including TipTap:
 * Markdown → markdownToHtml() → TipTap.setContent() → TipTap.getHTML() → htmlToMarkdown() → Markdown
 *
 * This is the critical path for the WYSIWYG editor mode switch.
 * Content corruption on mode switch indicates a failure in this flow.
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { Editor } from '@tiptap/react'
import {
  createTestEditor,
  destroyTestEditor,
  roundTripThroughTipTap,
  isIdempotentThroughTipTap,
  runTestCases,
  type TestCase,
} from '@/test/tiptap-test-utils'

describe('ContentConverter - TipTap Integration', () => {
  let editor: Editor

  beforeEach(() => {
    editor = createTestEditor()
  })

  afterEach(() => {
    destroyTestEditor(editor)
  })

  describe('Full Round-Trip Through TipTap', () => {
    describe('Code Blocks', () => {
      it('preserves code block with language', () => {
        const markdown = '```typescript\nconst x = 1;\n```'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('```typescript')
        expect(result.outputMarkdown).toContain('const x = 1;')
        expect(result.outputMarkdown).not.toContain('\\`')
      })

      it('preserves code block without language', () => {
        const markdown = '```\nconst x = 1;\n```'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('```')
        expect(result.outputMarkdown).toContain('const x = 1;')
      })

      it('preserves multiline code block', () => {
        const markdown = '```typescript\nfunction hello() {\n  console.log("hi");\n}\n```'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('function hello()')
        expect(result.outputMarkdown).toContain('console.log')
      })

      it('preserves various languages through round-trip', () => {
        const languages = ['javascript', 'python', 'go', 'rust', 'json', 'bash']

        for (const lang of languages) {
          const markdown = `\`\`\`${lang}\ncode_here\n\`\`\``
          const result = roundTripThroughTipTap(markdown)

          expect(result.outputMarkdown).toContain(`\`\`\`${lang}`)
          expect(result.outputMarkdown).toContain('code_here')
        }
      })

      it('preserves special characters in code blocks', () => {
        const markdown = '```\n<script>alert("xss")</script>\n```'
        const result = roundTripThroughTipTap(markdown)

        // The content should be preserved (possibly escaped)
        expect(result.outputMarkdown).toContain('script')
        expect(result.outputMarkdown).toContain('alert')
      })

      it('preserves markdown-like content inside code blocks', () => {
        const markdown = '```\n**not bold** and *not italic* and [not a link](test)\n```'
        const result = roundTripThroughTipTap(markdown)

        // These should NOT be converted to markdown formatting
        expect(result.outputMarkdown).toContain('**not bold**')
        expect(result.outputMarkdown).toContain('*not italic*')
        expect(result.outputMarkdown).toContain('[not a link]')
      })
    })

    describe('Text Formatting', () => {
      it('preserves bold formatting', () => {
        const markdown = '**bold text**'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('**bold text**')
        expect(result.outputMarkdown).not.toContain('\\*')
      })

      it('preserves italic formatting', () => {
        const markdown = '*italic text*'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('*italic text*')
        expect(result.outputMarkdown).not.toContain('\\*')
      })

      it('preserves strikethrough formatting', () => {
        const markdown = '~~deleted text~~'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('~~deleted text~~')
        expect(result.outputMarkdown).not.toContain('\\~')
      })

      it('preserves highlight formatting', () => {
        const markdown = '==highlighted text=='
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('==highlighted text==')
        expect(result.outputMarkdown).not.toContain('\\=')
      })

      it('preserves nested bold and italic', () => {
        const markdown = '***bold italic***'
        const result = roundTripThroughTipTap(markdown)

        // Should contain both bold and italic markers
        expect(result.outputMarkdown).toMatch(/\*+bold italic\*+/)
        expect(result.outputMarkdown).not.toContain('\\*')
      })

      it('preserves inline code', () => {
        const markdown = 'Use `const` for constants'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('`const`')
        expect(result.outputMarkdown).not.toContain('\\`')
      })
    })

    describe('Headings', () => {
      it('preserves h1', () => {
        const markdown = '# Heading 1'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown.trim()).toBe('# Heading 1')
        expect(result.outputMarkdown).not.toContain('\\#')
      })

      it('preserves h2', () => {
        const markdown = '## Heading 2'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown.trim()).toBe('## Heading 2')
      })

      it('preserves h3', () => {
        const markdown = '### Heading 3'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown.trim()).toBe('### Heading 3')
      })
    })

    describe('Links', () => {
      it('preserves basic links', () => {
        const markdown = '[Link text](https://example.com)'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('[Link text](https://example.com)')
        expect(result.outputMarkdown).not.toContain('\\[')
        expect(result.outputMarkdown).not.toContain('\\]')
      })

      it('preserves links with query parameters', () => {
        const markdown = '[Search](https://google.com/search?q=test&foo=bar)'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('[Search]')
        expect(result.outputMarkdown).toContain('https://google.com/search')
      })
    })

    describe('Lists', () => {
      it('preserves unordered lists', () => {
        const markdown = '- Item 1\n- Item 2\n- Item 3'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('Item 1')
        expect(result.outputMarkdown).toContain('Item 2')
        expect(result.outputMarkdown).toContain('Item 3')
        // Should use some list marker
        expect(result.outputMarkdown).toMatch(/[-*]\s+Item/)
      })

      it('preserves ordered lists', () => {
        const markdown = '1. First\n2. Second\n3. Third'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('First')
        expect(result.outputMarkdown).toContain('Second')
        expect(result.outputMarkdown).toContain('Third')
        // Should use numbered list
        expect(result.outputMarkdown).toMatch(/\d+\.\s+First/)
      })
    })

    describe('Blockquotes', () => {
      it('preserves blockquotes', () => {
        const markdown = '> This is a quote'
        const result = roundTripThroughTipTap(markdown)

        expect(result.outputMarkdown).toContain('>')
        expect(result.outputMarkdown).toContain('This is a quote')
      })
    })

    describe('Horizontal Rules', () => {
      it('preserves horizontal rules', () => {
        const markdown = '---'
        const result = roundTripThroughTipTap(markdown)

        // Should contain some form of horizontal rule marker
        // Turndown uses `* * *` format
        expect(result.outputMarkdown).toMatch(/---|\*\s*\*\s*\*|___/)
      })
    })
  })

  describe('Idempotency Through TipTap', () => {
    it('content stabilizes after one round-trip', () => {
      const markdown = '## Heading\n\n**bold** and *italic*\n\n```ts\ncode\n```'

      const first = roundTripThroughTipTap(markdown)
      const second = roundTripThroughTipTap(first.outputMarkdown)
      const third = roundTripThroughTipTap(second.outputMarkdown)

      // After first conversion, content should stabilize
      expect(second.outputMarkdown).toBe(first.outputMarkdown)
      expect(third.outputMarkdown).toBe(second.outputMarkdown)
    })

    it('does not accumulate escape characters', () => {
      const markdown = '**bold** and *italic*'

      let content = markdown
      for (let i = 0; i < 5; i++) {
        const result = roundTripThroughTipTap(content)
        content = result.outputMarkdown
      }

      // Should not have accumulated backslashes
      expect(content).not.toContain('\\*')
      expect(content).not.toContain('\\[')
      expect(content).not.toContain('\\#')
      expect(content).toContain('**bold**')
      expect(content).toContain('*italic*')
    })

    it('passes isIdempotentThroughTipTap check for basic content', () => {
      const markdown = '## Heading\n\n**Bold** text\n\n```typescript\nconst x = 1;\n```'
      expect(isIdempotentThroughTipTap(markdown)).toBe(true)
    })

    it('passes isIdempotentThroughTipTap check for complex content', () => {
      const markdown = `## Overview

This is a **paragraph** with *formatting*.

### Code Example

\`\`\`typescript
function hello(): void {
  console.log("Hello");
}
\`\`\`

### Features

- Bold text: **like this**
- Italic text: *like this*
- Links: [Click here](https://example.com)

> This is a blockquote

---

End of document.`

      expect(isIdempotentThroughTipTap(markdown)).toBe(true)
    })
  })

  describe('Bug Report Scenarios', () => {
    it('handles the mode switch corruption scenario', () => {
      const markdown = `## Overview

**Bold text** should stay bold.

### Code Example

\`\`\`typescript
const example = "test";
\`\`\``

      const result = roundTripThroughTipTap(markdown)

      // These should NOT be escaped
      expect(result.outputMarkdown).not.toContain('\\*\\*')
      expect(result.outputMarkdown).not.toContain('\\#')
      expect(result.outputMarkdown).toContain('**Bold text**')
      expect(result.outputMarkdown).toContain('## Overview')
      expect(result.outputMarkdown).toContain('### Code Example')
    })

    it('handles content with all major features', () => {
      const markdown = `# Main Title

## Section 1

This is **bold**, *italic*, and ~~strikethrough~~ text.

### Subsection

Here's some \`inline code\` and a [link](https://example.com).

\`\`\`typescript
const greeting = "Hello, World!";
console.log(greeting);
\`\`\`

- List item 1
- List item 2
  - Nested item

> A meaningful quote

---

The end.`

      const result = roundTripThroughTipTap(markdown)

      // Verify no escaping corruption
      expect(result.outputMarkdown).not.toContain('\\*')
      expect(result.outputMarkdown).not.toContain('\\#')
      expect(result.outputMarkdown).not.toContain('\\[')
      expect(result.outputMarkdown).not.toContain('\\`')
      expect(result.outputMarkdown).not.toContain('\\~')

      // Verify content preserved
      expect(result.outputMarkdown).toContain('# Main Title')
      expect(result.outputMarkdown).toContain('**bold**')
      expect(result.outputMarkdown).toContain('*italic*')
      expect(result.outputMarkdown).toContain('~~strikethrough~~')
      expect(result.outputMarkdown).toContain('`inline code`')
      expect(result.outputMarkdown).toContain('[link](https://example.com)')
      expect(result.outputMarkdown).toContain('```typescript')
    })
  })

  describe('Batch Test Cases', () => {
    it('runs comprehensive test suite', () => {
      const testCases: TestCase[] = [
        {
          name: 'bold text',
          markdown: '**bold**',
          expectedPreservations: ['**bold**'],
          expectedNotContains: ['\\*'],
        },
        {
          name: 'italic text',
          markdown: '*italic*',
          expectedPreservations: ['*italic*'],
          expectedNotContains: ['\\*'],
        },
        {
          name: 'heading',
          markdown: '## Heading',
          expectedPreservations: ['## Heading'],
          expectedNotContains: ['\\#'],
        },
        {
          name: 'code block',
          markdown: '```ts\ncode\n```',
          expectedPreservations: ['```', 'code'],
          expectedNotContains: ['\\`'],
        },
        {
          name: 'link',
          markdown: '[text](url)',
          expectedPreservations: ['[text](url)'],
          expectedNotContains: ['\\[', '\\]'],
        },
        {
          name: 'inline code',
          markdown: 'Use `const`',
          expectedPreservations: ['`const`'],
          expectedNotContains: ['\\`'],
        },
      ]

      const results = runTestCases(testCases)

      for (const result of results) {
        expect(result.preservationsPassed).toBe(true)
        expect(result.notContainsPassed).toBe(true)
      }
    })
  })

  describe('Newline Preservation Through TipTap', () => {
    it('preserves paragraph structure through TipTap round-trip', () => {
      const markdown = 'Line 1\n\nLine 2\n\nLine 3'
      const result = roundTripThroughTipTap(markdown)

      // Should preserve all three lines
      expect(result.outputMarkdown).toContain('Line 1')
      expect(result.outputMarkdown).toContain('Line 2')
      expect(result.outputMarkdown).toContain('Line 3')

      // Should have paragraph breaks (double newlines) between content
      const doubleNewlineCount = (result.outputMarkdown.match(/\n\n/g) || []).length
      expect(doubleNewlineCount).toBeGreaterThanOrEqual(2)
    })

    it('TipTap preserves paragraph structure in HTML', () => {
      const inputHtml = '<p>Line 1</p><p>Line 2</p><p>Line 3</p>'
      const localEditor = createTestEditor({ content: inputHtml, isHtml: true })
      const outputHtml = localEditor.getHTML()
      destroyTestEditor(localEditor)

      // TipTap should preserve paragraph tags
      expect(outputHtml).toContain('<p>Line 1</p>')
      expect(outputHtml).toContain('<p>Line 2</p>')
      expect(outputHtml).toContain('<p>Line 3</p>')
    })

    it('preserves complex document structure through TipTap', () => {
      const markdown = `# Heading

This is paragraph one.

This is paragraph two.

## Code Example

\`\`\`typescript
const x = 1;
\`\`\`

- List item 1
- List item 2`

      const result = roundTripThroughTipTap(markdown)

      // Verify all content is preserved
      expect(result.outputMarkdown).toContain('# Heading')
      expect(result.outputMarkdown).toContain('This is paragraph one.')
      expect(result.outputMarkdown).toContain('This is paragraph two.')
      expect(result.outputMarkdown).toContain('## Code Example')
      expect(result.outputMarkdown).toContain('```typescript')
      expect(result.outputMarkdown).toContain('List item 1')
      expect(result.outputMarkdown).toContain('List item 2')

      // Verify paragraphs are separated by blank lines
      expect(result.outputMarkdown).toMatch(/paragraph one\.\n\n/i)
      expect(result.outputMarkdown).toMatch(/paragraph two\.\n\n/i)
    })

    it('does not collapse multiple paragraphs into single line', () => {
      const markdown = 'First paragraph.\n\nSecond paragraph.\n\nThird paragraph.'
      const result = roundTripThroughTipTap(markdown)

      // Content should NOT be all on one line
      const lines = result.outputMarkdown.trim().split('\n')
      expect(lines.length).toBeGreaterThan(1)

      // Each paragraph should be present
      expect(result.outputMarkdown).toContain('First paragraph.')
      expect(result.outputMarkdown).toContain('Second paragraph.')
      expect(result.outputMarkdown).toContain('Third paragraph.')
    })

    it('preserves heading-paragraph structure', () => {
      const markdown = '## Title\n\nContent under heading.'
      const result = roundTripThroughTipTap(markdown)

      expect(result.outputMarkdown).toContain('## Title')
      expect(result.outputMarkdown).toContain('Content under heading.')
      // Should have separation between heading and content
      expect(result.outputMarkdown).toMatch(/Title\n+Content/)
    })
  })

  describe('Edge Cases', () => {
    it('handles empty content', () => {
      const markdown = ''
      const result = roundTripThroughTipTap(markdown)

      // Empty content should remain empty or minimal
      expect(result.outputMarkdown.trim()).toBe('')
    })

    it('handles whitespace-only content', () => {
      const markdown = '   \n\n   '
      const result = roundTripThroughTipTap(markdown)

      // Should handle gracefully without errors
      expect(typeof result.outputMarkdown).toBe('string')
    })

    it('handles content with special characters', () => {
      const markdown = 'Special chars: & < > " \''
      const result = roundTripThroughTipTap(markdown)

      // Content should be preserved (possibly entity-encoded in HTML)
      expect(result.outputMarkdown).toContain('Special chars')
    })

    it('handles deeply nested content', () => {
      const markdown = '> > > Deeply nested quote'
      const result = roundTripThroughTipTap(markdown)

      expect(result.outputMarkdown).toContain('Deeply nested quote')
    })
  })
})
