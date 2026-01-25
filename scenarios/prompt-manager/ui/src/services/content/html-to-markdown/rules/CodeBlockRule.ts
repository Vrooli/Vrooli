/**
 * Turndown rule for converting code blocks from HTML to Markdown.
 *
 * Handles:
 * - Code blocks with language classes (e.g., language-typescript)
 * - Language detection from both <pre> and <code> elements
 * - Extended fence preservation (4+ backticks via data-fence-count attribute)
 * - Proper fenced code block output (```language or ````language for extended)
 */

import type TurndownService from 'turndown'

export interface CodeBlockRuleOptions {
  /** Fallback language when none is detected */
  defaultLanguage?: string
}

/**
 * Extract language from a class string.
 *
 * @param className - The class attribute value
 * @returns The detected language or null
 */
function extractLanguageFromClass(className: string): string | null {
  if (!className) return null
  const match = className.match(/language-(\w+)/)
  return match?.[1] ?? null
}

/**
 * Create the code block turndown rule.
 *
 * @param options - Configuration options
 * @returns The turndown rule object
 */
export function createCodeBlockRule(
  options: CodeBlockRuleOptions = {}
): TurndownService.Rule {
  const { defaultLanguage = '' } = options

  return {
    filter: (node: HTMLElement): boolean => {
      return (
        node.nodeName === 'PRE' &&
        node.firstChild !== null &&
        node.firstChild.nodeName === 'CODE'
      )
    },
    replacement: (_content: string, node: HTMLElement): string => {
      const codeNode = node.firstChild as HTMLElement
      const text = codeNode.textContent || ''

      // Try to extract language from class - check both <code> and <pre> elements
      // marked puts class on <code>, TipTap may put it on <pre>
      const codeLanguage = extractLanguageFromClass(codeNode.className)
      const preLanguage = extractLanguageFromClass(node.className)
      const language = codeLanguage ?? preLanguage ?? defaultLanguage

      // Get fence count from data attribute (default to 3 for standard fences)
      const fenceCountAttr = node.getAttribute('data-fence-count')
      const fenceCount = fenceCountAttr ? parseInt(fenceCountAttr, 10) : 3
      const fence = '`'.repeat(fenceCount)

      // Strip trailing newlines from content to avoid double newlines before closing fence
      const trimmedText = text.replace(/\n+$/, '')

      // Leading newline ensures code block is on its own line when inside other elements
      return '\n' + fence + language + '\n' + trimmedText + '\n' + fence + '\n'
    },
  }
}
