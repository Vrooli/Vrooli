/**
 * Validates markdown content for patterns that won't survive HTML round-trip.
 *
 * This validator detects markdown constructs that will be corrupted when
 * converting through HTML (e.g., in a WYSIWYG editor). Users are warned
 * before switching modes so they can fix issues proactively.
 */

export type IssueType =
  | 'escaped-code-fence'
  | 'extended-code-fence'
  | 'unmatched-fence'
  | 'invalid-nesting'

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
 * 2. Extended code fences (4+ backticks) - will be converted to 3 backticks,
 *    which corrupts nested code blocks
 *
 * @param markdown - The markdown content to validate
 * @returns ValidationResult with any issues found
 */
export function validateMarkdown(markdown: string): ValidationResult {
  const issues: MarkdownIssue[] = []
  const lines = markdown.split('\n')

  // Track code fence state for detecting nesting issues
  let inCodeBlock = false
  let currentFenceBackticks = 0

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

    // Detect code fences (3+ backticks at start of line)
    const fenceMatch = line.match(/^(`{3,})(\w*)/)
    if (fenceMatch && fenceMatch[1]) {
      const backtickCount = fenceMatch[1].length
      const language = fenceMatch[2] ?? ''

      if (!inCodeBlock) {
        // Opening fence
        inCodeBlock = true
        currentFenceBackticks = backtickCount

        // Warn about extended fences (4+ backticks) - they get converted to 3
        if (backtickCount > 3) {
          issues.push({
            type: 'extended-code-fence',
            message: `Extended code fence (${backtickCount} backticks) will be converted to 3 backticks in Rich mode`,
            line: lineNum,
            column: 1,
            endColumn: backtickCount + language.length + 1,
            severity: 'warning',
            suggestion:
              'Nested code blocks will be corrupted. Consider keeping this content in Code mode only, or restructure to avoid nesting.',
          })
        }
      } else {
        // Check if this is a closing fence (same or more backticks, no language)
        if (backtickCount >= currentFenceBackticks && !language) {
          // Closing fence
          inCodeBlock = false
          currentFenceBackticks = 0
        }
        // If backtickCount < currentFenceBackticks or has language, it's nested content
        // The extended fence warning already covers this case
      }
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
