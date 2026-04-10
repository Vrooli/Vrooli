/**
 * Tests for backward-compatible contentConverter module.
 *
 * These tests verify that the re-exported functions from the new
 * content module work correctly through the legacy import path.
 *
 * For comprehensive tests, see ./content/ContentConverter.test.ts
 */

import { describe, it, expect } from 'vitest'
import { isHtml, markdownToHtml, htmlToMarkdown } from './contentConverter'

describe('contentConverter (backward compatibility)', () => {
  describe('isHtml', () => {
    it('should return true for HTML content', () => {
      expect(isHtml('<p>Hello</p>')).toBe(true)
      expect(isHtml('<div>Content</div>')).toBe(true)
    })

    it('should return false for markdown', () => {
      expect(isHtml('# Heading')).toBe(false)
      expect(isHtml('**bold**')).toBe(false)
    })
  })

  describe('markdownToHtml', () => {
    it('should convert basic markdown', () => {
      const html = markdownToHtml('**bold**')
      expect(html).toContain('<strong>bold</strong>')
    })

    it('should handle empty string', () => {
      expect(markdownToHtml('')).toBe('')
    })
  })

  describe('htmlToMarkdown', () => {
    it('should convert basic HTML', () => {
      const markdown = htmlToMarkdown('<strong>bold</strong>')
      expect(markdown).toContain('**bold**')
    })

    it('should handle empty string', () => {
      expect(htmlToMarkdown('')).toBe('')
    })
  })

  describe('round-trip conversion', () => {
    it('should preserve bold through round-trip', () => {
      const original = '**bold text**'
      const html = markdownToHtml(original)
      const result = htmlToMarkdown(html)
      expect(result).toContain('**bold text**')
    })

    it('should not escape markdown characters', () => {
      const original = '**bold** and *italic*'
      const html = markdownToHtml(original)
      const result = htmlToMarkdown(html)
      expect(result).not.toContain('\\*')
      expect(result).toContain('**bold**')
      expect(result).toContain('*italic*')
    })

    it('should preserve code blocks with language', () => {
      const original = '```typescript\nconst x = 1;\n```'
      const html = markdownToHtml(original)
      const result = htmlToMarkdown(html)
      expect(result).toContain('```typescript')
      expect(result).toContain('const x = 1;')
    })
  })
})
