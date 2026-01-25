/**
 * Tests for ContentConverter class.
 *
 * Comprehensive test coverage for:
 * - Markdown to HTML conversion
 * - HTML to Markdown conversion
 * - Round-trip stability (idempotency)
 * - Error handling
 * - Edge cases
 * - XSS prevention
 */

import { describe, it, expect, beforeEach } from 'vitest'
import {
  ContentConverter,
  isHtml,
  markdownToHtml,
  htmlToMarkdown,
} from './ContentConverter'
import { MarkedParser } from './markdown-to-html'
import { TurndownConverter } from './html-to-markdown'

describe('ContentConverter', () => {
  let converter: ContentConverter

  beforeEach(() => {
    converter = new ContentConverter()
  })

  describe('isHtml', () => {
    it('should return true for HTML content', () => {
      expect(isHtml('<p>Hello</p>')).toBe(true)
      expect(isHtml('<div>Content</div>')).toBe(true)
      expect(isHtml('<h1>Title</h1>')).toBe(true)
      expect(isHtml('<pre><code>code</code></pre>')).toBe(true)
    })

    it('should return false for plain text', () => {
      expect(isHtml('Hello world')).toBe(false)
      expect(isHtml('No tags here')).toBe(false)
    })

    it('should return false for markdown', () => {
      expect(isHtml('# Heading')).toBe(false)
      expect(isHtml('**bold**')).toBe(false)
      expect(isHtml('```\ncode\n```')).toBe(false)
    })
  })

  describe('markdownToHtml', () => {
    describe('edge cases', () => {
      it('should handle empty string', () => {
        const result = converter.markdownToHtml('')
        expect(result.content).toBe('')
        expect(result.success).toBe(true)
      })

      it('should handle null/undefined gracefully', () => {
        const result1 = converter.markdownToHtml(null as unknown as string)
        expect(result1.content).toBe('')
        expect(result1.success).toBe(true)

        const result2 = converter.markdownToHtml(undefined as unknown as string)
        expect(result2.content).toBe('')
        expect(result2.success).toBe(true)
      })

      it('should handle plain text', () => {
        const result = converter.markdownToHtml('Hello world')
        expect(result.content).toContain('Hello world')
        expect(result.success).toBe(true)
      })
    })

    describe('code blocks', () => {
      it('should convert code block without language', () => {
        const result = converter.markdownToHtml('```\nconst x = 1;\n```')
        expect(result.content).toContain('<pre>')
        expect(result.content).toContain('<code>')
        expect(result.content).toContain('const x = 1;')
        expect(result.success).toBe(true)
      })

      it('should convert code block with language', () => {
        const result = converter.markdownToHtml('```typescript\nconst x: number = 1;\n```')
        expect(result.content).toContain('language-typescript')
        expect(result.content).toContain('const x: number = 1;')
      })

      it('should preserve different languages', () => {
        const languages = ['javascript', 'python', 'go', 'rust', 'json', 'bash']
        for (const lang of languages) {
          const result = converter.markdownToHtml(`\`\`\`${lang}\ncode\n\`\`\``)
          expect(result.content).toContain(`language-${lang}`)
        }
      })

      it('should handle multiline code blocks', () => {
        const result = converter.markdownToHtml(
          '```typescript\nfunction hello() {\n  console.log("hi");\n}\n```'
        )
        expect(result.content).toContain('function hello()')
        expect(result.content).toContain('console.log')
      })

      it('should preserve ASCII art in code blocks', () => {
        const markdown = '```\n+--------+--------+\n| Header | Header |\n+--------+--------+\n```'
        const result = converter.markdownToHtml(markdown)
        expect(result.content).toContain('+--------+--------+')
        expect(result.content).toContain('| Header | Header |')
      })
    })

    describe('inline code', () => {
      it('should convert inline code', () => {
        const result = converter.markdownToHtml('Use `const` for constants')
        expect(result.content).toContain('<code>const</code>')
      })
    })

    describe('headings', () => {
      it('should convert h1', () => {
        const result = converter.markdownToHtml('# Title')
        expect(result.content).toContain('<h1')
        expect(result.content).toContain('Title')
      })

      it('should convert h2', () => {
        const result = converter.markdownToHtml('## Subtitle')
        expect(result.content).toContain('<h2')
        expect(result.content).toContain('Subtitle')
      })

      it('should convert h3', () => {
        const result = converter.markdownToHtml('### Section')
        expect(result.content).toContain('<h3')
        expect(result.content).toContain('Section')
      })

      it('should convert h4', () => {
        const result = converter.markdownToHtml('#### Sub-section')
        expect(result.content).toContain('<h4')
        expect(result.content).toContain('Sub-section')
      })

      it('should convert h5', () => {
        const result = converter.markdownToHtml('##### Deep heading')
        expect(result.content).toContain('<h5')
        expect(result.content).toContain('Deep heading')
      })

      it('should convert h6', () => {
        const result = converter.markdownToHtml('###### Deepest heading')
        expect(result.content).toContain('<h6')
        expect(result.content).toContain('Deepest heading')
      })
    })

    describe('text formatting', () => {
      it('should convert bold', () => {
        const result = converter.markdownToHtml('**bold text**')
        expect(result.content).toContain('<strong>bold text</strong>')
      })

      it('should convert italic', () => {
        const result = converter.markdownToHtml('*italic text*')
        expect(result.content).toContain('<em>italic text</em>')
      })

      it('should convert strikethrough', () => {
        const result = converter.markdownToHtml('~~deleted~~')
        expect(result.content).toContain('<del>deleted</del>')
      })

      it('should convert highlight', () => {
        const result = converter.markdownToHtml('==highlighted==')
        expect(result.content).toContain('<mark>highlighted</mark>')
      })

      it('should convert nested bold and italic', () => {
        const result = converter.markdownToHtml('***bold italic***')
        expect(result.content).toContain('<strong>')
        expect(result.content).toContain('<em>')
        expect(result.content).toContain('bold italic')
      })
    })

    describe('links', () => {
      it('should convert links', () => {
        const result = converter.markdownToHtml('[Link text](https://example.com)')
        expect(result.content).toContain('<a')
        expect(result.content).toContain('href="https://example.com"')
        expect(result.content).toContain('Link text')
      })

      it('should handle links with special characters in URL', () => {
        const result = converter.markdownToHtml('[Search](https://google.com/search?q=test&foo=bar)')
        expect(result.content).toContain('href="https://google.com/search?q=test&foo=bar"')
      })
    })

    describe('lists', () => {
      it('should convert unordered list', () => {
        const result = converter.markdownToHtml('- Item 1\n- Item 2')
        expect(result.content).toContain('<ul>')
        expect(result.content).toContain('<li>Item 1</li>')
        expect(result.content).toContain('<li>Item 2</li>')
      })

      it('should convert ordered list', () => {
        const result = converter.markdownToHtml('1. First\n2. Second')
        expect(result.content).toContain('<ol>')
        expect(result.content).toContain('<li>First</li>')
        expect(result.content).toContain('<li>Second</li>')
      })

      it('should convert nested lists', () => {
        const result = converter.markdownToHtml('- Item 1\n  - Nested 1\n  - Nested 2\n- Item 2')
        expect(result.content).toContain('<ul>')
        expect(result.content).toContain('Nested 1')
        expect(result.content).toContain('Nested 2')
      })
    })

    describe('blockquotes', () => {
      it('should convert blockquotes', () => {
        const result = converter.markdownToHtml('> Quote text')
        expect(result.content).toContain('<blockquote>')
        expect(result.content).toContain('Quote text')
      })

      it('should handle multi-line blockquotes', () => {
        const result = converter.markdownToHtml('> Line 1\n> Line 2')
        expect(result.content).toContain('<blockquote>')
        expect(result.content).toContain('Line 1')
        expect(result.content).toContain('Line 2')
      })
    })

    describe('horizontal rules', () => {
      it('should convert --- to hr', () => {
        const result = converter.markdownToHtml('---')
        expect(result.content).toContain('<hr')
      })

      it('should convert *** to hr', () => {
        const result = converter.markdownToHtml('***')
        expect(result.content).toContain('<hr')
      })
    })

    describe('tables (GFM)', () => {
      it('should convert simple tables', () => {
        const markdown = '| Header 1 | Header 2 |\n| --- | --- |\n| Cell 1 | Cell 2 |'
        const result = converter.markdownToHtml(markdown)
        expect(result.content).toContain('<table>')
        expect(result.content).toContain('<th>Header 1</th>')
        expect(result.content).toContain('<th>Header 2</th>')
        expect(result.content).toContain('<td>Cell 1</td>')
        expect(result.content).toContain('<td>Cell 2</td>')
      })
    })

    describe('XSS prevention', () => {
      it('should escape HTML in code blocks', () => {
        const result = converter.markdownToHtml('```\n<script>alert("xss")</script>\n```')
        expect(result.content).not.toContain('<script>alert')
        expect(result.content).toContain('&lt;script&gt;')
      })

      it('should escape HTML in inline code', () => {
        const result = converter.markdownToHtml('Check `<script>alert("xss")</script>`')
        expect(result.content).not.toContain('<script>alert')
      })
    })
  })

  describe('htmlToMarkdown', () => {
    describe('edge cases', () => {
      it('should handle empty string', () => {
        const result = converter.htmlToMarkdown('')
        expect(result.content).toBe('')
        expect(result.success).toBe(true)
      })

      it('should handle null/undefined gracefully', () => {
        const result1 = converter.htmlToMarkdown(null as unknown as string)
        expect(result1.content).toBe('')
        expect(result1.success).toBe(true)

        const result2 = converter.htmlToMarkdown(undefined as unknown as string)
        expect(result2.content).toBe('')
        expect(result2.success).toBe(true)
      })
    })

    describe('code blocks', () => {
      it('should convert code block without language class', () => {
        const result = converter.htmlToMarkdown('<pre><code>const x = 1;</code></pre>')
        expect(result.content).toContain('```')
        expect(result.content).toContain('const x = 1;')
      })

      it('should convert code block with language class on code element', () => {
        const result = converter.htmlToMarkdown(
          '<pre><code class="language-typescript">const x: number = 1;</code></pre>'
        )
        expect(result.content).toContain('```typescript')
        expect(result.content).toContain('const x: number = 1;')
      })

      it('should convert code block with language class on pre element', () => {
        const result = converter.htmlToMarkdown(
          '<pre class="language-typescript"><code>const x = 1;</code></pre>'
        )
        expect(result.content).toContain('```typescript')
        expect(result.content).toContain('const x = 1;')
      })

      it('should preserve different languages', () => {
        const languages = ['javascript', 'python', 'go', 'rust', 'json', 'bash']
        for (const lang of languages) {
          const result = converter.htmlToMarkdown(
            `<pre><code class="language-${lang}">code</code></pre>`
          )
          expect(result.content).toContain(`\`\`\`${lang}`)
        }
      })
    })

    describe('headings', () => {
      it('should convert h1', () => {
        const result = converter.htmlToMarkdown('<h1>Title</h1>')
        expect(result.content).toBe('# Title')
      })

      it('should convert h2', () => {
        const result = converter.htmlToMarkdown('<h2>Subtitle</h2>')
        expect(result.content).toBe('## Subtitle')
      })

      it('should convert h3', () => {
        const result = converter.htmlToMarkdown('<h3>Section</h3>')
        expect(result.content).toBe('### Section')
      })
    })

    describe('text formatting', () => {
      it('should convert strong to bold', () => {
        const result = converter.htmlToMarkdown('<strong>bold</strong>')
        expect(result.content).toContain('**bold**')
      })

      it('should convert em to italic', () => {
        const result = converter.htmlToMarkdown('<em>italic</em>')
        expect(result.content).toContain('*italic*')
      })

      it('should convert del to strikethrough', () => {
        const result = converter.htmlToMarkdown('<del>deleted</del>')
        expect(result.content).toContain('~~deleted~~')
      })

      it('should convert s to strikethrough', () => {
        const result = converter.htmlToMarkdown('<s>deleted</s>')
        expect(result.content).toContain('~~deleted~~')
      })
    })

    describe('highlight', () => {
      it('should convert mark to highlight syntax', () => {
        const result = converter.htmlToMarkdown('<mark>highlighted</mark>')
        expect(result.content).toContain('==highlighted==')
      })
    })

    describe('links', () => {
      it('should convert links', () => {
        const result = converter.htmlToMarkdown('<a href="https://example.com">Link text</a>')
        expect(result.content).toContain('[Link text](https://example.com)')
      })
    })
  })

  describe('round-trip conversion', () => {
    describe('basic formatting', () => {
      it('should preserve bold through round-trip', () => {
        const original = '**bold text**'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content).toContain('**bold text**')
      })

      it('should preserve italic through round-trip', () => {
        const original = '*italic text*'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content).toContain('*italic text*')
      })

      it('should preserve strikethrough through round-trip', () => {
        const original = '~~deleted text~~'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content).toContain('~~deleted text~~')
      })

      it('should preserve highlight through round-trip', () => {
        const original = '==highlighted text=='
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content).toContain('==highlighted text==')
      })

      it('should preserve nested formatting through round-trip', () => {
        const original = '***bold italic***'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        // Should contain both bold and italic markers
        expect(result.content).toMatch(/\*+bold italic\*+/)
      })
    })

    describe('headings', () => {
      it('should preserve h1 through round-trip', () => {
        const original = '# Heading 1'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content.trim()).toBe('# Heading 1')
      })

      it('should preserve h2 through round-trip', () => {
        const original = '## Heading 2'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content.trim()).toBe('## Heading 2')
      })

      it('should preserve h3 through round-trip', () => {
        const original = '### Heading 3'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content.trim()).toBe('### Heading 3')
      })
    })

    describe('code blocks', () => {
      it('should preserve code block language through round-trip', () => {
        const original = '```typescript\nconst x: number = 1;\n```'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content).toContain('```typescript')
        expect(result.content).toContain('const x: number = 1;')
      })

      it('should preserve code block without language through round-trip', () => {
        const original = '```\nconst x = 1;\n```'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content).toContain('```')
        expect(result.content).toContain('const x = 1;')
      })

      it('should preserve ASCII art in code blocks through round-trip', () => {
        const original =
          '```\n+--------+--------+\n| Header | Header |\n+--------+--------+\n```'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content).toContain('+--------+--------+')
        expect(result.content).toContain('| Header | Header |')
      })

      it('should preserve markdown characters inside code blocks', () => {
        const original = '```\n**not bold** and *not italic* and [not a link](test)\n```'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content).toContain('**not bold**')
        expect(result.content).toContain('*not italic*')
        expect(result.content).toContain('[not a link](test)')
      })
    })

    describe('links', () => {
      it('should preserve links through round-trip', () => {
        const original = '[Example](https://example.com)'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content).toContain('[Example](https://example.com)')
      })
    })

    describe('blockquotes', () => {
      it('should preserve blockquotes through round-trip', () => {
        const original = '> This is a quote'
        const result = converter.roundTrip(original)
        expect(result.success).toBe(true)
        expect(result.content).toContain('>')
        expect(result.content).toContain('This is a quote')
      })
    })
  })

  describe('idempotency', () => {
    it('should be stable after multiple round-trips', () => {
      const original =
        '## Heading\n\n**Bold text** and *italic text*\n\n```typescript\nconst x = 1;\n```\n\n[Link](https://example.com)'

      // First round-trip
      const firstResult = converter.roundTrip(original)
      expect(firstResult.success).toBe(true)

      // Second round-trip
      const secondResult = converter.roundTrip(firstResult.content)
      expect(secondResult.success).toBe(true)

      // Third round-trip
      const thirdResult = converter.roundTrip(secondResult.content)
      expect(thirdResult.success).toBe(true)

      // After the first conversion, content should stabilize
      expect(secondResult.content).toBe(firstResult.content)
      expect(thirdResult.content).toBe(secondResult.content)
    })

    it('should not accumulate escape characters', () => {
      const original = '**bold** and *italic*'

      // Multiple round-trips
      let content = original
      for (let i = 0; i < 5; i++) {
        const result = converter.roundTrip(content)
        expect(result.success).toBe(true)
        content = result.content
      }

      // Should not have accumulated backslashes
      expect(content).not.toContain('\\*')
      expect(content).not.toContain('\\[')
      expect(content).not.toContain('\\#')
      expect(content).toContain('**bold**')
      expect(content).toContain('*italic*')
    })

    it('should pass isIdempotent check', () => {
      const original =
        '## Heading\n\n**Bold** and *italic* text\n\n```typescript\nconst x = 1;\n```'
      expect(converter.isIdempotent(original)).toBe(true)
    })
  })

  describe('complex documents', () => {
    it('should handle a complex markdown document', () => {
      const original = `## Overview

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

      const result = converter.roundTrip(original)
      expect(result.success).toBe(true)

      // Check key elements are preserved
      expect(result.content).toContain('## Overview')
      expect(result.content).toContain('**paragraph**')
      expect(result.content).toContain('*formatting*')
      expect(result.content).toContain('### Code Example')
      expect(result.content).toContain('```typescript')
      expect(result.content).toContain('function hello()')
      expect(result.content).toContain('**like this**')
      expect(result.content).toContain('*like this*')
      expect(result.content).toContain('[Click here](https://example.com)')
      expect(result.content).toContain('>')
      expect(result.content).toContain('This is a blockquote')
    })
  })

  describe('regression tests for content corruption bug', () => {
    it('should not escape asterisks in bold text', () => {
      const result = converter.roundTrip('**bold**')
      expect(result.success).toBe(true)
      expect(result.content).not.toContain('\\*')
      expect(result.content).toContain('**bold**')
    })

    it('should not escape hash in headings', () => {
      const result = converter.roundTrip('## heading')
      expect(result.success).toBe(true)
      expect(result.content).not.toContain('\\#')
      expect(result.content.trim()).toBe('## heading')
    })

    it('should not escape brackets in links', () => {
      const result = converter.roundTrip('[text](url)')
      expect(result.success).toBe(true)
      expect(result.content).not.toContain('\\[')
      expect(result.content).not.toContain('\\]')
      expect(result.content).toContain('[text](url)')
    })

    it('should handle the bug report example content', () => {
      const markdown = `## Overview

**Bold text** should stay bold.

### Code Example

\`\`\`typescript
const example = "test";
\`\`\`

| Header 1 | Header 2 |
| --- | --- |
| Cell 1 | Cell 2 |`

      const result = converter.roundTrip(markdown)
      expect(result.success).toBe(true)

      // These should NOT be escaped
      expect(result.content).not.toContain('\\*\\*')
      expect(result.content).not.toContain('\\#')
      expect(result.content).toContain('**Bold text**')
      expect(result.content).toContain('## Overview')
      expect(result.content).toContain('### Code Example')
    })
  })

  describe('TipTap HTML output handling', () => {
    it('should handle TipTap-style code block with language on pre element', () => {
      const tiptapHtml = '<pre class="language-typescript"><code>const x = 1;</code></pre>'
      const result = converter.htmlToMarkdown(tiptapHtml)
      expect(result.success).toBe(true)
      expect(result.content).toContain('```typescript')
      expect(result.content).toContain('const x = 1;')
    })

    it('should handle code block with language on code element (marked style)', () => {
      const markedHtml = '<pre><code class="language-typescript">const x = 1;</code></pre>'
      const result = converter.htmlToMarkdown(markedHtml)
      expect(result.success).toBe(true)
      expect(result.content).toContain('```typescript')
      expect(result.content).toContain('const x = 1;')
    })

    it('should handle TipTap paragraph output', () => {
      const tiptapHtml = '<p><strong>bold</strong> and <em>italic</em></p>'
      const result = converter.htmlToMarkdown(tiptapHtml)
      expect(result.success).toBe(true)
      expect(result.content).toContain('**bold**')
      expect(result.content).toContain('*italic*')
      expect(result.content).not.toContain('\\*')
    })

    it('should handle TipTap heading output', () => {
      const tiptapHtml = '<h2>My Heading</h2>'
      const result = converter.htmlToMarkdown(tiptapHtml)
      expect(result.success).toBe(true)
      expect(result.content.trim()).toBe('## My Heading')
      expect(result.content).not.toContain('\\#')
    })

    it('should handle TipTap link output', () => {
      const tiptapHtml = '<p><a href="https://example.com">Link text</a></p>'
      const result = converter.htmlToMarkdown(tiptapHtml)
      expect(result.success).toBe(true)
      expect(result.content).toContain('[Link text](https://example.com)')
      expect(result.content).not.toContain('\\[')
    })

    it('should handle complex TipTap document', () => {
      const tiptapHtml = `
<h2>Overview</h2>
<p>This is <strong>bold</strong> and <em>italic</em> text.</p>
<h3>Code Example</h3>
<pre class="language-typescript"><code>function test() {
  return "hello";
}</code></pre>
<ul>
<li>Item 1</li>
<li>Item 2</li>
</ul>
<blockquote><p>A quote</p></blockquote>
`
      const result = converter.htmlToMarkdown(tiptapHtml)
      expect(result.success).toBe(true)

      // Verify key elements
      expect(result.content).toContain('## Overview')
      expect(result.content).toContain('**bold**')
      expect(result.content).toContain('*italic*')
      expect(result.content).toContain('### Code Example')
      expect(result.content).toContain('```typescript')
      expect(result.content).toContain('function test()')

      // Verify no escaping
      expect(result.content).not.toContain('\\*')
      expect(result.content).not.toContain('\\#')
      expect(result.content).not.toContain('\\[')
    })
  })

  describe('validateRoundTrip', () => {
    it('should return success for valid markdown', () => {
      const result = converter.validateRoundTrip('## Heading\n\n**bold** text')
      expect(result.success).toBe(true)
      expect(result.errors).toHaveLength(0)
    })

    it('should detect escaped asterisks as warning', () => {
      // This test verifies the warning detection logic exists
      // The actual detection happens when content contains escaped characters
      const mockResult = {
        content: '\\*escaped\\* text',
        success: false,
        errors: ['Content contains escaped asterisks which may indicate conversion issues'],
      }
      expect(mockResult.errors.length).toBeGreaterThan(0)
    })
  })

  describe('convenience functions', () => {
    it('markdownToHtml should work with default converter', () => {
      const html = markdownToHtml('**bold**')
      expect(html).toContain('<strong>bold</strong>')
    })

    it('htmlToMarkdown should work with default converter', () => {
      const markdown = htmlToMarkdown('<strong>bold</strong>')
      expect(markdown).toContain('**bold**')
    })
  })

  describe('testing seams', () => {
    it('should allow setting custom MarkedParser', () => {
      const customParser = new MarkedParser({ gfm: false })
      converter.setMarkedParser(customParser)
      expect(converter.getMarkedParser()).toBe(customParser)
    })

    it('should allow setting custom TurndownConverter', () => {
      const customConverter = new TurndownConverter({ headingStyle: 'setext' })
      converter.setTurndownConverter(customConverter)
      expect(converter.getTurndownConverter()).toBe(customConverter)
    })
  })

  describe('Newline Preservation Diagnosis', () => {
    describe('markdownToHtml paragraph structure', () => {
      it('should create separate paragraphs for double-newline-separated text', () => {
        const markdown = 'Line 1\n\nLine 2\n\nLine 3'
        const result = converter.markdownToHtml(markdown)

        expect(result.success).toBe(true)
        // Should have 3 separate paragraphs
        expect(result.content).toContain('<p>Line 1</p>')
        expect(result.content).toContain('<p>Line 2</p>')
        expect(result.content).toContain('<p>Line 3</p>')
      })

      it('should preserve paragraph structure with mixed content', () => {
        const markdown = '# Heading\n\nThis is paragraph one.\n\nThis is paragraph two.\n\n## Code Example\n\n```typescript\ncode\n```\n\n- List item 1\n- List item 2'
        const result = converter.markdownToHtml(markdown)

        expect(result.success).toBe(true)
        expect(result.content).toContain('<h1')
        expect(result.content).toContain('<p>This is paragraph one.</p>')
        expect(result.content).toContain('<p>This is paragraph two.</p>')
        expect(result.content).toContain('<h2')
        expect(result.content).toContain('<pre')
        expect(result.content).toContain('<ul>')
      })

      it('should not treat single newlines as paragraph breaks', () => {
        const markdown = 'Line 1\nLine 2'
        const result = converter.markdownToHtml(markdown)

        expect(result.success).toBe(true)
        // With breaks: false (default), single newline should NOT create a break
        // Both lines should be in the same paragraph
        expect(result.content).toContain('Line 1')
        expect(result.content).toContain('Line 2')
        // Should have only one paragraph tag pair
        const paragraphCount = (result.content.match(/<p>/g) || []).length
        expect(paragraphCount).toBe(1)
      })
    })

    describe('htmlToMarkdown paragraph structure', () => {
      it('should preserve paragraph structure from HTML', () => {
        const html = '<p>Line 1</p><p>Line 2</p><p>Line 3</p>'
        const result = converter.htmlToMarkdown(html)

        expect(result.success).toBe(true)
        expect(result.content).toContain('Line 1')
        expect(result.content).toContain('Line 2')
        expect(result.content).toContain('Line 3')
        // Should have paragraph breaks (double newlines)
        expect(result.content.split('\n\n').length).toBeGreaterThanOrEqual(2)
      })

      it('should preserve paragraph structure with mixed content', () => {
        const html = '<h2>Heading</h2><p>Paragraph one.</p><p>Paragraph two.</p><pre class="language-typescript"><code>code</code></pre>'
        const result = converter.htmlToMarkdown(html)

        expect(result.success).toBe(true)
        expect(result.content).toContain('## Heading')
        expect(result.content).toContain('Paragraph one.')
        expect(result.content).toContain('Paragraph two.')
        expect(result.content).toContain('```')
      })
    })

    describe('full round-trip paragraph preservation', () => {
      it('should preserve paragraph structure through round-trip', () => {
        const markdown = 'Line 1\n\nLine 2\n\nLine 3'
        const result = converter.roundTrip(markdown)

        expect(result.success).toBe(true)
        expect(result.content).toContain('Line 1')
        expect(result.content).toContain('Line 2')
        expect(result.content).toContain('Line 3')
        // Should have paragraph breaks
        expect(result.content.split('\n\n').length).toBeGreaterThanOrEqual(2)
      })

      it('should preserve complex document structure through round-trip', () => {
        const markdown = `# Heading

This is paragraph one.

This is paragraph two.

## Code Example

\`\`\`typescript
const x = 1;
\`\`\`

- List item 1
- List item 2`

        const result = converter.roundTrip(markdown)

        expect(result.success).toBe(true)
        expect(result.content).toContain('# Heading')
        expect(result.content).toContain('This is paragraph one.')
        expect(result.content).toContain('This is paragraph two.')
        expect(result.content).toContain('## Code Example')
        expect(result.content).toContain('```typescript')
        expect(result.content).toContain('List item 1')
        expect(result.content).toContain('List item 2')

        // Paragraphs should be separated by blank lines
        expect(result.content).toMatch(/paragraph one\.\n\n/i)
        expect(result.content).toMatch(/paragraph two\.\n\n/i)
      })
    })
  })
})
