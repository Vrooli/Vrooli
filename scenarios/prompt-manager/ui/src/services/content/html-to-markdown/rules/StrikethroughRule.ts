/**
 * Turndown rule for converting strikethrough elements to Markdown.
 *
 * Handles:
 * - <del> tags (standard HTML)
 * - <s> tags (obsolete but still used)
 *
 * Outputs GFM-style ~~text~~ syntax.
 */

import type TurndownService from 'turndown'

export interface StrikethroughRuleOptions {
  /** The delimiter to use for strikethrough (default: '~~') */
  delimiter?: string
}

/**
 * Create the strikethrough turndown rule.
 *
 * @param options - Configuration options
 * @returns The turndown rule object
 */
export function createStrikethroughRule(
  options: StrikethroughRuleOptions = {}
): TurndownService.Rule {
  const { delimiter = '~~' } = options

  return {
    // 'strike' is obsolete but still used in some HTML, so we cast to include it
    filter: ['del', 's'] as (keyof HTMLElementTagNameMap)[],
    replacement: (content: string): string => {
      return `${delimiter}${content}${delimiter}`
    },
  }
}
