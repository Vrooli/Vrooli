/**
 * Highlight utilities for cross-reference navigation.
 *
 * When a user clicks a cross-reference, we navigate to the source entity
 * and highlight the matched skill ID text in the Monaco editor.
 */

import type { ContentSearchMatch } from '@/lib/schemas'

export interface HighlightRequest {
  /** File path within entity (for agents/teams) */
  file?: string
  /** 1-based line number to scroll to */
  line: number
  /** Text to find and highlight on that line */
  text: string
}

/**
 * Create a ContentSearchMatch from a highlight request and file content.
 *
 * Finds all occurrences of `request.text` on the given line and returns
 * a ContentSearchMatch with computed matchRanges for Monaco decorations.
 */
export function createHighlightMatch(
  content: string,
  request: HighlightRequest,
): ContentSearchMatch | null {
  if (!request.text) return null

  const lines = content.split('\n')
  if (request.line < 1 || request.line > lines.length) return null

  const lineContent = lines[request.line - 1]
  if (lineContent === undefined) return null

  const ranges: Array<{ start: number; end: number }> = []
  let idx = 0
  while ((idx = lineContent.indexOf(request.text, idx)) !== -1) {
    ranges.push({ start: idx, end: idx + request.text.length })
    idx += request.text.length
  }
  if (ranges.length === 0) return null

  return {
    skillId: '',
    skillName: '',
    file: request.file ?? '',
    folder: '',
    lineNumber: request.line,
    line: lineContent,
    matchRanges: ranges,
  }
}
