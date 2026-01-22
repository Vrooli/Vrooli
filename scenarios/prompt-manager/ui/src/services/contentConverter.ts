/**
 * Content Converter - Pure functions for HTML/Markdown conversion.
 *
 * Handles:
 * - Markdown to HTML conversion for TipTap rendering
 * - HTML to Markdown conversion for storage
 * - Code block language preservation in both directions
 */

import TurndownService from 'turndown'

// Configure Turndown for HTML→Markdown conversion
const turndown = new TurndownService({
  headingStyle: 'atx',      // Use ## style headers
  codeBlockStyle: 'fenced', // Use ``` style code blocks
  bulletListMarker: '-',    // Use - for bullet lists
  emDelimiter: '*',         // Use * for emphasis
  strongDelimiter: '**',    // Use ** for bold
})

// Add custom rule for code blocks with language classes
turndown.addRule('codeBlock', {
  filter: (node: HTMLElement) => {
    return (
      node.nodeName === 'PRE' &&
      node.firstChild !== null &&
      node.firstChild.nodeName === 'CODE'
    )
  },
  replacement: (_content: string, node: HTMLElement) => {
    const codeNode = node.firstChild as HTMLElement
    const text = codeNode.textContent || ''
    // Extract language from class (e.g., "language-typescript" -> "typescript")
    const language = codeNode.className.match(/language-(\w+)/)?.[1] ?? ''
    return '\n```' + language + '\n' + text + '\n```\n'
  },
})

// Add custom rule for highlight/mark
turndown.addRule('highlight', {
  filter: 'mark',
  replacement: (content: string) => `==${content}==`,
})

/**
 * Check if content looks like HTML (has HTML tags).
 *
 * @param content - The content to check
 * @returns True if content contains HTML tags
 */
export function isHtml(content: string): boolean {
  // Check for common HTML tags
  return /<[a-z][\s\S]*>/i.test(content)
}

/**
 * Convert Markdown to HTML for TipTap rendering.
 *
 * Supports:
 * - Code blocks with language specifiers (```language)
 * - Inline code
 * - Headers (h1-h3)
 * - Bold, italic, strikethrough
 * - Highlight (==text==)
 * - Blockquotes
 * - Lists (unordered and ordered)
 * - Horizontal rules
 *
 * @param markdown - Markdown string to convert
 * @returns HTML string
 */
export function markdownToHtml(markdown: string): string {
  let html = markdown

  // Escape HTML entities first to prevent XSS
  html = html
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  // Code blocks (must be before other replacements)
  html = html.replace(/```(\w*)\n([\s\S]*?)```/g, (_match: string, lang: string, code: string) => {
    // Include language class so TipTap can parse it back
    const langClass = lang ? ` class="language-${lang}"` : ''
    return `<pre><code${langClass}>${code.trim()}</code></pre>`
  })

  // Inline code
  html = html.replace(/`([^`]+)`/g, '<code>$1</code>')

  // Headers (must be before bold since # can be at line start)
  html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>')
  html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>')
  html = html.replace(/^# (.+)$/gm, '<h1>$1</h1>')

  // Bold and italic
  html = html.replace(/\*\*\*(.+?)\*\*\*/g, '<strong><em>$1</em></strong>')
  html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
  html = html.replace(/\*(.+?)\*/g, '<em>$1</em>')
  html = html.replace(/~~(.+?)~~/g, '<s>$1</s>')

  // Highlight (==text==)
  html = html.replace(/==(.+?)==/g, '<mark>$1</mark>')

  // Blockquotes
  html = html.replace(/^> (.+)$/gm, '<blockquote>$1</blockquote>')

  // Horizontal rules
  html = html.replace(/^---$/gm, '<hr>')
  html = html.replace(/^\*\*\*$/gm, '<hr>')

  // Unordered lists
  html = html.replace(/^- (.+)$/gm, '<li>$1</li>')
  html = html.replace(/(<li>.*<\/li>\n?)+/g, (match) => `<ul>${match}</ul>`)

  // Ordered lists (basic support)
  html = html.replace(/^\d+\. (.+)$/gm, '<li>$1</li>')

  // Paragraphs - wrap remaining text blocks
  // Split by double newlines and wrap non-tag content
  const blocks = html.split(/\n\n+/)
  html = blocks
    .map((block) => {
      const trimmed = block.trim()
      if (!trimmed) return ''
      // Don't wrap if already wrapped in a block element
      if (/^<(h[1-6]|p|ul|ol|li|blockquote|pre|hr)/i.test(trimmed)) {
        return trimmed
      }
      // Replace single newlines with <br> within paragraphs
      return `<p>${trimmed.replace(/\n/g, '<br>')}</p>`
    })
    .join('\n')

  return html
}

/**
 * Convert HTML to Markdown for storage.
 *
 * @param html - HTML string to convert
 * @returns Markdown string
 */
export function htmlToMarkdown(html: string): string {
  return turndown.turndown(html)
}
