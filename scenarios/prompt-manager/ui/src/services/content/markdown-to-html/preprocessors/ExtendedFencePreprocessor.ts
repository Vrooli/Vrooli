/**
 * ExtendedFencePreprocessor - Detects extended code fences (4+ backticks) and converts
 * them directly to HTML to preserve nesting.
 *
 * Extended code fences are used in markdown to nest code blocks, but marked.js
 * doesn't preserve the backtick count and would interpret inner fences. This preprocessor:
 * 1. Detects fences with 4+ backticks
 * 2. Extracts the full content (including any nested fences)
 * 3. Outputs HTML directly with a data-fence-count attribute
 * 4. The content is preserved exactly as-is
 */

export interface FenceInfo {
  startLine: number
  backtickCount: number
  language: string
}

/**
 * Escape HTML special characters in content.
 */
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

/**
 * Detect extended fences and convert them directly to HTML.
 * This preserves the exact content including any nested code fences.
 *
 * @param markdown - The markdown content to process
 * @returns The processed content with extended fences as HTML
 */
export function preprocessExtendedFences(markdown: string): string {
  if (!markdown) return markdown

  const lines = markdown.split('\n')
  const result: string[] = []
  let i = 0

  while (i < lines.length) {
    const line = lines[i]
    if (line === undefined) {
      i++
      continue
    }

    // Match opening fence with 4+ backticks (with optional language)
    const fenceMatch = line.match(/^(`{4,})(\w*)/)

    if (fenceMatch && fenceMatch[1]) {
      const backtickCount = fenceMatch[1].length
      const language = fenceMatch[2] ?? ''
      const contentLines: string[] = []
      i++

      // Find closing fence (same or more backticks, no language specifier)
      let foundClosing = false
      while (i < lines.length) {
        const closeLine = lines[i]
        if (closeLine === undefined) {
          i++
          continue
        }

        // Match closing fence: must be only backticks and at least as many as opening
        const closeMatch = closeLine.match(/^(`+)$/)
        if (closeMatch && closeMatch[1] && closeMatch[1].length >= backtickCount) {
          foundClosing = true
          break
        }
        contentLines.push(closeLine)
        i++
      }

      // Build HTML output directly for extended fences
      // This bypasses marked's code block handling and preserves content exactly
      const content = escapeHtml(contentLines.join('\n'))
      const langClass = language ? ` class="language-${language}"` : ''
      result.push(`<pre data-fence-count="${backtickCount}"><code${langClass}>${content}</code></pre>`)

      // If we didn't find a closing fence, the content is malformed but we handled it
      if (!foundClosing) {
        // Don't increment i again, we're at the end
      }
    } else {
      result.push(line)
    }
    i++
  }

  return result.join('\n')
}
