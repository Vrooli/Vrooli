/**
 * Validates markdown content for patterns that won't survive HTML round-trip.
 *
 * This validator detects markdown constructs that will be corrupted when
 * converting through HTML (e.g., in a WYSIWYG editor). Users are warned
 * before switching modes so they can fix issues proactively.
 */

import { ContentConverter } from '../ContentConverter'

export type IssueType =
  | 'escaped-code-fence'
  | 'unmatched-fence'
  | 'invalid-nesting'
  | 'round-trip-unstable'

export interface MarkdownIssue {
  type: IssueType
  message: string
  line: number
  column: number
  endLine?: number
  endColumn?: number
  severity: 'warning' | 'error'
  suggestion?: string
}

export interface ValidationResult {
  isValid: boolean
  issues: MarkdownIssue[]
}


/**
 * Validates markdown content for patterns that won't round-trip correctly
 * through HTML conversion.
 *
 * Detected issues:
 * 1. Escaped code fences (\`\`\`) - literal backslash-backticks won't render as code
 *
 * Note: Extended code fences (4+ backticks) are now preserved through the
 * conversion pipeline and no longer need warnings.
 *
 * @param markdown - The markdown content to validate
 * @returns ValidationResult with any issues found
 */
export function validateMarkdown(markdown: string): ValidationResult {
  const issues: MarkdownIssue[] = []
  const lines = markdown.split('\n')

  lines.forEach((line, index) => {
    const lineNum = index + 1

    // Detect escaped code fences: \`\`\` at start of line (with optional language)
    const escapedFenceMatch = line.match(/^(\\`\\`\\`)(\w*)/)
    if (escapedFenceMatch) {
      const language = escapedFenceMatch[2] || ''
      const suggestion = language
        ? `Remove backslashes: \`\`\`${language}`
        : 'Remove backslashes: ```'

      issues.push({
        type: 'escaped-code-fence',
        message: 'Escaped code fence will not render as a code block',
        line: lineNum,
        column: 1,
        endColumn: escapedFenceMatch[0].length + 1,
        severity: 'warning',
        suggestion,
      })
      return // Skip further processing for this line
    }

    // Also detect escaped fences in the middle of lines
    const midLineMatch = line.match(/(.+)(\\`\\`\\`)(\w*)/)
    if (midLineMatch && !escapedFenceMatch) {
      const prefix = midLineMatch[1] ?? ''
      const fence = midLineMatch[2] ?? ''
      const language = midLineMatch[3] ?? ''
      const suggestion = language
        ? `Remove backslashes: \`\`\`${language}`
        : 'Remove backslashes: ```'

      issues.push({
        type: 'escaped-code-fence',
        message: 'Escaped code fence will not render as a code block',
        line: lineNum,
        column: prefix.length + 1,
        endColumn: prefix.length + fence.length + language.length + 1,
        severity: 'warning',
        suggestion,
      })
    }
  })

  return {
    isValid: issues.length === 0,
    issues,
  }
}

export interface RoundTripResult {
  /** Whether the markdown survives round-trip unchanged */
  isStable: boolean
  /** The round-tripped content */
  roundTrippedContent: string
  /** Description of what changed, if unstable */
  changeDescription?: string
}

/**
 * Normalize markdown content for comparison.
 *
 * Applies normalizations for differences that are semantically equivalent
 * in WYSIWYG editing. These are cosmetic differences that don't affect
 * the rendered output or content meaning.
 *
 * Normalizations applied:
 * - Trim leading/trailing whitespace
 * - Normalize line endings to \n
 * - Normalize trailing whitespace on each line
 * - Normalize multiple consecutive blank lines to single blank lines
 * - Normalize whitespace around code fences (any length: ```, ````, etc.)
 * - Normalize horizontal rules (* * * → ---)
 * - Normalize list markers (* → -)
 * - Normalize spacing after list markers (- item, -   item → - item)
 * - Normalize nested list indentation (2-space → 4-space)
 * - Join hard-wrapped paragraph lines (same paragraph, different source lines)
 * - Normalize table separator format
 *
 * @param content - The content to normalize
 * @returns Normalized content
 */
export function normalizeForComparison(content: string): string {
  const lines = content.split('\n')
  const result: string[] = []
  let inCodeBlock = false
  let currentParagraph: string[] = []

  const flushParagraph = (): void => {
    if (currentParagraph.length > 0) {
      // Join hard-wrapped paragraph lines with a space
      result.push(currentParagraph.join(' '))
      currentParagraph = []
    }
  }

  const normalizeLine = (line: string): string => {
    // Trim trailing whitespace
    let normalized = line.trimEnd()

    // Unescape common markdown characters (turndown may unescape these)
    // \. → .  \* → *  \# → #  \[ → [  \] → ]  \( → (  \) → )
    normalized = normalized.replace(/\\([.*#\[\]()])/g, '$1')

    // Normalize horizontal rules: * * * or - - - → ---
    if (/^[\s]*(\*\s*\*\s*\*|-\s*-\s*-)[\s]*$/.test(normalized)) {
      return '---'
    }

    // Normalize table separator rows: |---|---| or | --- | --- | → |---|---|
    if (/^\|[\s\-:]+\|/.test(normalized)) {
      // Count columns by splitting on |
      const cols = normalized.split('|').filter((s) => s.trim().length > 0)
      return '|' + cols.map(() => '---').join('|') + '|'
    }

    // Normalize list items with optional task list checkbox
    // Match: optional leading whitespace + bullet (* or -) + spaces + optional checkbox + content
    const listMatch = normalized.match(/^(\s*)([\*\-])\s+(\[[ xX]\]\s+)?(.*)$/)
    if (listMatch) {
      const indent = listMatch[1] ?? ''
      const itemContent = listMatch[4] ?? ''
      // Normalize indent: any 1-4 spaces = level 1, 5-8 = level 2, etc.
      // This handles both 2-space and 4-space conventions
      const indentLevel = indent.length > 0 ? Math.ceil(indent.length / 4) : 0
      const normalizedIndent = '    '.repeat(indentLevel)
      // Drop task list checkbox - it doesn't survive HTML round-trip
      return `${normalizedIndent}- ${itemContent}`
    }

    // Normalize numbered list items
    const numberedMatch = normalized.match(/^(\s*)(\d+)\.\s+(.*)$/)
    if (numberedMatch) {
      const indent = numberedMatch[1] ?? ''
      const number = numberedMatch[2] ?? '1'
      const itemContent = numberedMatch[3] ?? ''
      const indentLevel = indent.length > 0 ? Math.ceil(indent.length / 4) : 0
      const normalizedIndent = '    '.repeat(indentLevel)
      return `${normalizedIndent}${number}. ${itemContent}`
    }

    // Normalize block quotes (may have varying whitespace after >)
    const blockQuoteMatch = normalized.match(/^(\s*)>\s*(.*)$/)
    if (blockQuoteMatch) {
      const indent = blockQuoteMatch[1] ?? ''
      const quoteContent = blockQuoteMatch[2] ?? ''
      return `${indent}> ${quoteContent}`
    }

    // Normalize indented code fences (inside list items)
    const codeFenceMatch = normalized.match(/^(\s*)(`{3,})(.*)$/)
    if (codeFenceMatch) {
      const indent = codeFenceMatch[1] ?? ''
      const fence = codeFenceMatch[2] ?? '```'
      const rest = codeFenceMatch[3] ?? ''
      const indentLevel = indent.length > 0 ? Math.ceil(indent.length / 4) : 0
      const normalizedIndent = '    '.repeat(indentLevel)
      return `${normalizedIndent}${fence}${rest}`
    }

    // Normalize any indented line (continuation content inside lists)
    // This catches code block content and other indented text
    if (/^\s+/.test(normalized) && !/^(\s*)([-*]|\d+\.)\s/.test(normalized)) {
      const match = normalized.match(/^(\s+)(.*)$/)
      if (match) {
        const indent = match[1] ?? ''
        const content = match[2] ?? ''
        const indentLevel = indent.length > 0 ? Math.ceil(indent.length / 4) : 0
        const normalizedIndent = '    '.repeat(indentLevel)
        return `${normalizedIndent}${content}`
      }
    }

    return normalized
  }

  // Helper to check if a line is a block element (shouldn't be joined)
  const isBlockElement = (line: string): boolean => {
    const trimmed = line.trim()
    return (
      trimmed === '' || // Blank line
      /^#{1,6}\s/.test(trimmed) || // Heading
      /^[\*\-]\s/.test(trimmed) || // Unordered list (with or without checkbox)
      /^\d+\.\s/.test(trimmed) || // Ordered list
      /^>/.test(trimmed) || // Block quote (including empty >)
      /^```/.test(trimmed) || // Code fence
      /^---$/.test(trimmed) || // HR
      /^\|/.test(trimmed) || // Table row
      /^\s+[\*\-]\s/.test(line) || // Indented list item
      /^\s+\d+\.\s/.test(line) // Indented numbered list item
    )
  }

  for (const line of lines) {
    const trimmed = line.trim()

    // Track code blocks
    if (/^`{3,}/.test(trimmed)) {
      flushParagraph()
      inCodeBlock = !inCodeBlock
      result.push(normalizeLine(line))
      continue
    }

    // Inside code blocks, strip common leading indentation (list indentation)
    // but preserve relative indentation within the code
    if (inCodeBlock) {
      result.push(line.trimEnd())
      continue
    }

    // Blank line ends paragraph (including lines that are only whitespace)
    if (trimmed === '') {
      flushParagraph()
      result.push('')
      continue
    }

    // Block elements get their own line
    if (isBlockElement(line)) {
      flushParagraph()
      result.push(normalizeLine(line))
      continue
    }

    // Regular text - accumulate for paragraph joining
    currentParagraph.push(trimmed)
  }

  flushParagraph()

  return (
    result
      .join('\n')
      .trim()
      // Normalize line endings
      .replace(/\r\n/g, '\n')
      // Normalize multiple blank lines to single blank line
      .replace(/\n{3,}/g, '\n\n')
      // Remove blank lines before block elements (lists, headings, HRs)
      // These are optional in markdown and HTML conversion adds them
      .replace(/\n\n([-*] )/g, '\n$1')
      .replace(/\n\n(\d+\. )/g, '\n$1')
      .replace(/\n\n(#{1,6} )/g, '\n$1')
      .replace(/\n\n(---)/g, '\n$1')
      .replace(/\n\n(\|)/g, '\n$1')
      // Remove blank lines after headings (HTML conversion adds these)
      .replace(/(#{1,6} [^\n]+)\n\n/g, '$1\n')
      // Remove blank lines after HRs
      .replace(/(---)\n\n/g, '$1\n')
      // Normalize whitespace before closing code fences
      .replace(/\n+(`{3,})/g, '\n$1')
      // Remove blank lines before/after block quotes
      .replace(/\n\n(>)/g, '\n$1')
      .replace(/(>[^\n]*)\n\n/g, '$1\n')
      // Normalize block quote empty lines (> with nothing after)
      .replace(/^>\s*$/gm, '>')
      // Remove blank lines between any list items (bullet, numbered, or mixed)
      // This handles turndown adding blank lines inside nested lists
      // Pattern: any list item followed by blank line followed by any list item (with indentation)
      .replace(/([-*] [^\n]+)\n\n(\s*[-*] )/g, '$1\n$2')
      .replace(/([-*] [^\n]+)\n\n(\s*\d+\. )/g, '$1\n$2')
      .replace(/(\d+\.\s+[^\n]+)\n\n(\s*[-*] )/g, '$1\n$2')
      .replace(/(\d+\.\s+[^\n]+)\n\n(\s*\d+\. )/g, '$1\n$2')
      // Apply multiple times for deeply nested lists
      .replace(/([-*] [^\n]+)\n\n(\s*[-*] )/g, '$1\n$2')
      .replace(/([-*] [^\n]+)\n\n(\s*\d+\. )/g, '$1\n$2')
      .replace(/(\d+\.\s+[^\n]+)\n\n(\s*[-*] )/g, '$1\n$2')
      .replace(/(\d+\.\s+[^\n]+)\n\n(\s*\d+\. )/g, '$1\n$2')
  )
}

/**
 * Checks if markdown survives HTML round-trip unchanged.
 *
 * This is a catch-all protection for unknown edge cases. If markdown
 * changes after conversion to HTML and back, the content may be corrupted
 * in Rich mode.
 *
 * Note: Minor whitespace differences (like extra newlines around code fences)
 * are normalized before comparison since they don't affect content semantics.
 *
 * @param markdown - The markdown content to validate
 * @returns RoundTripResult indicating stability
 */
export function validateRoundTrip(markdown: string): RoundTripResult {
  // Handle empty content
  if (!markdown || !markdown.trim()) {
    return {
      isStable: true,
      roundTrippedContent: markdown,
    }
  }

  const converter = new ContentConverter()
  const result = converter.roundTrip(markdown)

  if (!result.success) {
    return {
      isStable: false,
      roundTrippedContent: result.content,
      changeDescription: 'Conversion failed: ' + result.errors.join(', '),
    }
  }

  // Normalize for comparison
  const normalizedOriginal = normalizeForComparison(markdown)
  const normalizedResult = normalizeForComparison(result.content)

  if (normalizedOriginal !== normalizedResult) {
    return {
      isStable: false,
      roundTrippedContent: result.content,
      changeDescription: 'Content will be modified during conversion',
    }
  }

  return {
    isStable: true,
    roundTrippedContent: result.content,
  }
}
