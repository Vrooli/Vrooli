/**
 * TipTap HTML Output Discovery Tests
 *
 * These tests capture what TipTap actually produces when given various HTML inputs.
 * They help us understand TipTap's transformations so we can adjust our conversion
 * rules accordingly.
 *
 * NOTE: These tests document actual TipTap behavior, not desired behavior.
 * If TipTap's behavior changes, these tests will need to be updated.
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { Editor } from '@tiptap/react'
import {
  createTestEditor,
  destroyTestEditor,
  setContentAndGetHtml,
  compareHtmlTransformation,
} from '@/test/tiptap-test-utils'

describe('TipTap HTML Output Discovery', () => {
  let editor: Editor

  beforeEach(() => {
    editor = createTestEditor()
  })

  afterEach(() => {
    destroyTestEditor(editor)
  })

  describe('Code Blocks', () => {
    it('discovers: language on <code> element (marked style)', () => {
      const input = '<pre><code class="language-typescript">const x = 1;</code></pre>'
      const output = setContentAndGetHtml(editor, input)

      // Document what TipTap produces
      // TipTap should preserve or transform the language attribute
      expect(output).toContain('const x = 1;')

      // Check if language is preserved and where it ends up
      const hasLanguageOnPre = output.includes('<pre class="language-typescript"')
      const hasLanguageOnCode = output.includes('<code class="language-typescript"')

      // Document the actual behavior
      console.log('[Discovery] Code block with language on <code>:')
      console.log('  Input:', input)
      console.log('  Output:', output)
      console.log('  Language on <pre>:', hasLanguageOnPre)
      console.log('  Language on <code>:', hasLanguageOnCode)

      // At least one should have the language
      expect(hasLanguageOnPre || hasLanguageOnCode).toBe(true)
    })

    it('discovers: language on <pre> element (TipTap style)', () => {
      const input = '<pre class="language-typescript"><code>const x = 1;</code></pre>'
      const output = setContentAndGetHtml(editor, input)

      expect(output).toContain('const x = 1;')

      const hasLanguageOnPre = output.includes('<pre class="language-typescript"')
      const hasLanguageOnCode = output.includes('<code class="language-typescript"')

      console.log('[Discovery] Code block with language on <pre>:')
      console.log('  Input:', input)
      console.log('  Output:', output)
      console.log('  Language on <pre>:', hasLanguageOnPre)
      console.log('  Language on <code>:', hasLanguageOnCode)

      expect(hasLanguageOnPre || hasLanguageOnCode).toBe(true)
    })

    it('discovers: code block without language', () => {
      const input = '<pre><code>const x = 1;</code></pre>'
      const output = setContentAndGetHtml(editor, input)

      expect(output).toContain('const x = 1;')

      console.log('[Discovery] Code block without language:')
      console.log('  Input:', input)
      console.log('  Output:', output)
    })

    it('discovers: code block preserves multiline content', () => {
      const input = '<pre><code class="language-typescript">function hello() {\n  console.log("hi");\n}</code></pre>'
      const output = setContentAndGetHtml(editor, input)

      expect(output).toContain('function hello()')
      expect(output).toContain('console.log')

      console.log('[Discovery] Multiline code block:')
      console.log('  Input:', input)
      console.log('  Output:', output)
    })

    it('discovers: code block with special characters', () => {
      const input = '<pre><code>&lt;script&gt;alert("xss")&lt;/script&gt;</code></pre>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Code block with HTML entities:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      // Verify HTML entities are preserved
      expect(output).not.toContain('<script>')
    })
  })

  describe('Text Formatting', () => {
    it('discovers: bold text output', () => {
      const input = '<p><strong>bold text</strong></p>'
      const output = setContentAndGetHtml(editor, input)

      const comparison = compareHtmlTransformation(input, output)
      console.log('[Discovery] Bold text:')
      console.log('  Input:', input)
      console.log('  Output:', output)
      console.log('  Identical:', comparison.isIdentical)

      expect(output).toContain('bold text')
      expect(output).toMatch(/<strong>|<b>/i)
    })

    it('discovers: italic text output', () => {
      const input = '<p><em>italic text</em></p>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Italic text:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('italic text')
      expect(output).toMatch(/<em>|<i>/i)
    })

    it('discovers: strikethrough text output', () => {
      const input = '<p><s>strikethrough</s></p>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Strikethrough with <s>:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      // Check which tag TipTap uses for strikethrough
      const hasS = output.includes('<s>')
      const hasDel = output.includes('<del>')
      const hasStrike = output.includes('<strike>')

      console.log('  Uses <s>:', hasS)
      console.log('  Uses <del>:', hasDel)
      console.log('  Uses <strike>:', hasStrike)

      expect(output).toContain('strikethrough')
    })

    it('discovers: del tag output', () => {
      const input = '<p><del>deleted</del></p>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Strikethrough with <del>:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('deleted')
    })

    it('discovers: highlight/mark output', () => {
      const input = '<p><mark>highlighted</mark></p>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Highlight:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('highlighted')
      // Check if mark tag is preserved
      const hasMark = output.includes('<mark')
      console.log('  Uses <mark>:', hasMark)
    })

    it('discovers: nested formatting', () => {
      const input = '<p><strong><em>bold italic</em></strong></p>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Nested bold+italic:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('bold italic')
    })
  })

  describe('Headings', () => {
    it('discovers: h1 output', () => {
      const input = '<h1>Heading 1</h1>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] H1:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('<h1')
      expect(output).toContain('Heading 1')
    })

    it('discovers: h2 output', () => {
      const input = '<h2>Heading 2</h2>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] H2:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('<h2')
      expect(output).toContain('Heading 2')
    })

    it('discovers: h3 output', () => {
      const input = '<h3>Heading 3</h3>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] H3:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('<h3')
      expect(output).toContain('Heading 3')
    })
  })

  describe('Links', () => {
    it('discovers: link output', () => {
      const input = '<p><a href="https://example.com">Link text</a></p>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Link:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('href="https://example.com"')
      expect(output).toContain('Link text')
    })

    it('discovers: link with special characters in URL', () => {
      const input = '<p><a href="https://example.com/search?q=test&foo=bar">Search</a></p>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Link with query params:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('Search')
      // Check if ampersand is preserved or encoded
      const hasAmpersand = output.includes('&foo=')
      const hasEncodedAmpersand = output.includes('&amp;foo=')
      console.log('  Has raw ampersand:', hasAmpersand)
      console.log('  Has encoded ampersand:', hasEncodedAmpersand)
    })
  })

  describe('Lists', () => {
    it('discovers: unordered list output', () => {
      const input = '<ul><li>Item 1</li><li>Item 2</li></ul>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Unordered list:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('<ul>')
      expect(output).toContain('<li>')
      expect(output).toContain('Item 1')
      expect(output).toContain('Item 2')
    })

    it('discovers: ordered list output', () => {
      const input = '<ol><li>First</li><li>Second</li></ol>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Ordered list:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('<ol>')
      expect(output).toContain('<li>')
      expect(output).toContain('First')
      expect(output).toContain('Second')
    })

    it('discovers: nested list output', () => {
      const input = '<ul><li>Parent<ul><li>Child 1</li><li>Child 2</li></ul></li></ul>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Nested list:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('Parent')
      expect(output).toContain('Child 1')
      expect(output).toContain('Child 2')
    })
  })

  describe('Blockquotes', () => {
    it('discovers: blockquote output', () => {
      const input = '<blockquote><p>Quote text</p></blockquote>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Blockquote:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('<blockquote>')
      expect(output).toContain('Quote text')
    })

    it('discovers: nested blockquote output', () => {
      const input = '<blockquote><p>Outer</p><blockquote><p>Inner</p></blockquote></blockquote>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Nested blockquote:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('Outer')
      expect(output).toContain('Inner')
    })
  })

  describe('Tables', () => {
    it('discovers: simple table output', () => {
      const input = '<table><thead><tr><th>Header 1</th><th>Header 2</th></tr></thead><tbody><tr><td>Cell 1</td><td>Cell 2</td></tr></tbody></table>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Table:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      // TipTap may or may not support tables depending on extensions
      const hasTable = output.includes('<table')
      console.log('  Has <table>:', hasTable)
    })
  })

  describe('Horizontal Rules', () => {
    it('discovers: hr output', () => {
      const input = '<hr>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Horizontal rule:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('<hr')
    })
  })

  describe('Inline Code', () => {
    it('discovers: inline code output', () => {
      const input = '<p>Use <code>const</code> for constants</p>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Inline code:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('<code')
      expect(output).toContain('const')
    })

    it('discovers: inline code with class', () => {
      const input = '<p>Use <code class="inline-code">const</code> for constants</p>'
      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Inline code with class:')
      console.log('  Input:', input)
      console.log('  Output:', output)

      expect(output).toContain('const')
      // Check if class is preserved
      const hasClass = output.includes('class=')
      console.log('  Class preserved:', hasClass)
    })
  })

  describe('Complex Documents', () => {
    it('discovers: complex document structure', () => {
      const input = `
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
`.trim()

      const output = setContentAndGetHtml(editor, input)

      console.log('[Discovery] Complex document:')
      console.log('  Input length:', input.length)
      console.log('  Output length:', output.length)
      console.log('  Output:', output)

      // Verify key elements
      expect(output).toContain('<h2')
      expect(output).toContain('Overview')
      expect(output).toContain('<strong>')
      expect(output).toContain('<em>')
      expect(output).toContain('<h3')
      expect(output).toContain('<pre')
      expect(output).toContain('function test()')
      expect(output).toContain('<ul>')
      expect(output).toContain('<blockquote>')
    })
  })
})
