/**
 * TurndownConverter - Wrapper around Turndown for HTML to Markdown conversion.
 *
 * Features:
 * - Pre-configured with GFM-compatible settings
 * - Custom rules for code blocks, highlights, and strikethrough
 * - Disabled escaping to prevent markdown corruption on round-trips
 * - Injectable for testing
 */

import TurndownService from 'turndown'
import {
  createCodeBlockRule,
  createHighlightRule,
  createStrikethroughRule,
  createTableRule,
} from './rules'

export interface TurndownConverterConfig {
  /** Use ATX style headers (## style) */
  headingStyle?: 'atx' | 'setext'
  /** Use fenced code blocks (``` style) */
  codeBlockStyle?: 'fenced' | 'indented'
  /** Bullet list marker */
  bulletListMarker?: '-' | '*' | '+'
  /** Emphasis delimiter */
  emDelimiter?: '*' | '_'
  /** Strong delimiter */
  strongDelimiter?: '**' | '__'
  /** Horizontal rule style */
  hr?: '---' | '***' | '* * *' | '- - -'
  /** Whether to disable escaping of markdown characters (important for round-trips) */
  disableEscaping?: boolean
}

const DEFAULT_CONFIG: Required<TurndownConverterConfig> = {
  headingStyle: 'atx',
  codeBlockStyle: 'fenced',
  bulletListMarker: '-',
  emDelimiter: '*',
  strongDelimiter: '**',
  hr: '---',
  disableEscaping: true,
}

/**
 * TurndownConverter provides HTML to Markdown conversion.
 */
export class TurndownConverter {
  private turndown: TurndownService

  constructor(config: TurndownConverterConfig = {}) {
    const finalConfig = { ...DEFAULT_CONFIG, ...config }

    this.turndown = new TurndownService({
      headingStyle: finalConfig.headingStyle,
      codeBlockStyle: finalConfig.codeBlockStyle,
      bulletListMarker: finalConfig.bulletListMarker,
      emDelimiter: finalConfig.emDelimiter,
      strongDelimiter: finalConfig.strongDelimiter,
      hr: finalConfig.hr,
    })

    // CRITICAL: Disable escaping of markdown characters
    // This prevents **bold** from becoming \*\*bold\*\* during round-trips
    if (finalConfig.disableEscaping) {
      this.turndown.escape = (text: string): string => text
    }

    // Add custom rules
    this.addRule('table', createTableRule())
    this.addRule('codeBlock', createCodeBlockRule())
    this.addRule('highlight', createHighlightRule())
    this.addRule('strikethrough', createStrikethroughRule())
  }

  /**
   * Add a custom turndown rule.
   *
   * @param name - Rule name for identification
   * @param rule - The turndown rule object
   */
  addRule(name: string, rule: TurndownService.Rule): this {
    this.turndown.addRule(name, rule)
    return this
  }

  /**
   * Convert HTML to Markdown.
   *
   * @param html - The HTML string to convert
   * @returns The converted Markdown string
   */
  convert(html: string): string {
    if (!html) return ''
    return this.turndown.turndown(html)
  }

  /**
   * Get the underlying TurndownService instance for advanced configuration.
   */
  getTurndownService(): TurndownService {
    return this.turndown
  }
}
