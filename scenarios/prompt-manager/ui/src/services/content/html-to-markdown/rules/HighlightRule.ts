/**
 * Turndown rule for converting highlight/mark elements to Markdown.
 *
 * Converts <mark> tags to ==text== syntax (non-standard but commonly supported).
 */

import type TurndownService from 'turndown'

export interface HighlightRuleOptions {
  /** The delimiter to use for highlights (default: '==') */
  delimiter?: string
}

/**
 * Create the highlight turndown rule.
 *
 * @param options - Configuration options
 * @returns The turndown rule object
 */
export function createHighlightRule(
  options: HighlightRuleOptions = {}
): TurndownService.Rule {
  const { delimiter = '==' } = options

  return {
    filter: 'mark',
    replacement: (content: string): string => {
      return `${delimiter}${content}${delimiter}`
    },
  }
}
