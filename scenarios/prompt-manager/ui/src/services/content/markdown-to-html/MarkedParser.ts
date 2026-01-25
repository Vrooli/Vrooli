/**
 * MarkedParser - Wrapper around the marked library for Markdown to HTML conversion.
 *
 * Features:
 * - Pre-configured for GitHub Flavored Markdown (GFM)
 * - Preprocessor pipeline for custom syntax extensions
 * - Injectable for testing
 */

import { marked, type MarkedOptions } from 'marked'
import { preprocessHighlight } from './preprocessors'

export interface MarkedParserConfig {
  /** Enable GitHub Flavored Markdown (tables, strikethrough, etc.) */
  gfm?: boolean
  /** Convert \n to <br> */
  breaks?: boolean
  /** Enable highlight syntax preprocessing (==text==) */
  highlightSyntax?: boolean
}

const DEFAULT_CONFIG: Required<MarkedParserConfig> = {
  gfm: true,
  breaks: false,
  highlightSyntax: true,
}

/**
 * MarkedParser provides Markdown to HTML conversion.
 */
export class MarkedParser {
  private config: Required<MarkedParserConfig>
  private markedOptions: MarkedOptions

  constructor(config: MarkedParserConfig = {}) {
    this.config = { ...DEFAULT_CONFIG, ...config }

    this.markedOptions = {
      gfm: this.config.gfm,
      breaks: this.config.breaks,
    }
  }

  /**
   * Convert Markdown to HTML.
   *
   * @param markdown - The markdown string to convert
   * @returns The converted HTML string
   */
  parse(markdown: string): string {
    if (!markdown) return ''

    // Apply preprocessors
    let processed = markdown

    // Highlight syntax preprocessing (==text== -> <mark>text</mark>)
    if (this.config.highlightSyntax) {
      processed = preprocessHighlight(processed)
    }

    // Use marked for proper markdown parsing
    // Note: When async: false, marked.parse returns string synchronously
    return marked.parse(processed, {
      ...this.markedOptions,
      async: false,
    })
  }

  /**
   * Get the current configuration.
   */
  getConfig(): Required<MarkedParserConfig> {
    return { ...this.config }
  }

  /**
   * Update the configuration.
   *
   * @param config - New configuration options
   */
  setConfig(config: Partial<MarkedParserConfig>): void {
    this.config = { ...this.config, ...config }
    this.markedOptions = {
      gfm: this.config.gfm,
      breaks: this.config.breaks,
    }
  }
}
