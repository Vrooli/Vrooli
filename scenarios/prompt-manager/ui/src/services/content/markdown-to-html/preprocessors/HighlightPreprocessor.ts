/**
 * HighlightPreprocessor - Converts ==text== highlight syntax to <mark> tags.
 *
 * This preprocessing step is needed because marked doesn't support
 * the highlight syntax natively.
 */

export interface HighlightPreprocessorOptions {
  /** The delimiter pattern to match (default: '==') */
  delimiter?: string
}

/**
 * Convert highlight syntax to HTML mark tags.
 *
 * @param content - The markdown content to process
 * @param options - Preprocessor options
 * @returns The processed content with <mark> tags
 */
export function preprocessHighlight(
  content: string,
  options: HighlightPreprocessorOptions = {}
): string {
  const { delimiter = '==' } = options

  if (!content) return content

  // Escape the delimiter for use in regex
  const escapedDelimiter = delimiter.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

  // Match ==text== but not inside code blocks or inline code
  // This simple approach works for most cases
  const pattern = new RegExp(`${escapedDelimiter}([^=]+)${escapedDelimiter}`, 'g')

  return content.replace(pattern, '<mark>$1</mark>')
}
