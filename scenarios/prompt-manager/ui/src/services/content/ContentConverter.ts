/**
 * ContentConverter - Main class for bidirectional Markdown/HTML conversion.
 *
 * Features:
 * - Markdown to HTML conversion (using marked library)
 * - HTML to Markdown conversion (using turndown)
 * - Round-trip testing capabilities
 * - Error handling and validation
 * - Configurable parsers for testing seams
 *
 * This class serves as the main entry point for all content conversion
 * operations in the WYSIWYG editor.
 */

import { MarkedParser, type MarkedParserConfig } from './markdown-to-html'
import { TurndownConverter, type TurndownConverterConfig } from './html-to-markdown'

export interface ConversionResult {
  /** The converted content */
  content: string
  /** Whether the conversion was successful */
  success: boolean
  /** Any errors that occurred during conversion */
  errors: string[]
}

export interface ContentConverterConfig {
  /** Configuration for the markdown to HTML parser */
  markedConfig?: MarkedParserConfig
  /** Configuration for the HTML to markdown converter */
  turndownConfig?: TurndownConverterConfig
}

/**
 * Check if content looks like HTML (has HTML tags).
 *
 * @param content - The content to check
 * @returns True if content contains HTML tags
 */
export function isHtml(content: string): boolean {
  // Check for common HTML tags
  return /<[a-z][\s\S]*>/i.test(content)
}

/**
 * ContentConverter handles bidirectional conversion between Markdown and HTML.
 */
export class ContentConverter {
  private markedParser: MarkedParser
  private turndownConverter: TurndownConverter

  constructor(config: ContentConverterConfig = {}) {
    this.markedParser = new MarkedParser(config.markedConfig)
    this.turndownConverter = new TurndownConverter(config.turndownConfig)
  }

  /**
   * Convert Markdown to HTML.
   *
   * @param markdown - The markdown string to convert
   * @returns ConversionResult with the HTML content
   */
  markdownToHtml(markdown: string): ConversionResult {
    const errors: string[] = []

    try {
      // Handle empty/falsy input defensively
      if (!markdown) {
        return { content: '', success: true, errors: [] }
      }

      const content = this.markedParser.parse(markdown)
      return { content, success: true, errors }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error)
      errors.push(`Markdown to HTML conversion failed: ${errorMessage}`)
      return { content: markdown, success: false, errors }
    }
  }

  /**
   * Convert HTML to Markdown.
   *
   * @param html - The HTML string to convert
   * @returns ConversionResult with the Markdown content
   */
  htmlToMarkdown(html: string): ConversionResult {
    const errors: string[] = []

    try {
      // Handle empty/falsy input defensively
      if (!html) {
        return { content: '', success: true, errors: [] }
      }

      const content = this.turndownConverter.convert(html)
      return { content, success: true, errors }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error)
      errors.push(`HTML to Markdown conversion failed: ${errorMessage}`)
      return { content: html, success: false, errors }
    }
  }

  /**
   * Perform a round-trip conversion: Markdown → HTML → Markdown.
   *
   * This is useful for testing conversion stability and detecting
   * content corruption issues.
   *
   * @param markdown - The markdown string to round-trip
   * @returns ConversionResult with the round-tripped content
   */
  roundTrip(markdown: string): ConversionResult {
    const errors: string[] = []

    try {
      // Markdown → HTML
      const htmlResult = this.markdownToHtml(markdown)
      if (!htmlResult.success) {
        return {
          content: markdown,
          success: false,
          errors: [...htmlResult.errors, 'Round-trip failed at Markdown to HTML step'],
        }
      }

      // HTML → Markdown
      const markdownResult = this.htmlToMarkdown(htmlResult.content)
      if (!markdownResult.success) {
        return {
          content: markdown,
          success: false,
          errors: [...markdownResult.errors, 'Round-trip failed at HTML to Markdown step'],
        }
      }

      return { content: markdownResult.content, success: true, errors }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error)
      errors.push(`Round-trip conversion failed: ${errorMessage}`)
      return { content: markdown, success: false, errors }
    }
  }

  /**
   * Test if content is idempotent after multiple round-trips.
   *
   * @param markdown - The markdown to test
   * @param iterations - Number of round-trips to perform (default: 3)
   * @returns True if content stabilizes, false if it keeps changing
   */
  isIdempotent(markdown: string, iterations: number = 3): boolean {
    let content = markdown

    for (let i = 0; i < iterations; i++) {
      const result = this.roundTrip(content)
      if (!result.success) {
        return false
      }

      // After first iteration, content should stabilize
      if (i > 0 && result.content !== content) {
        return false
      }

      content = result.content
    }

    return true
  }

  /**
   * Validate that markdown content survives round-trip conversion.
   *
   * This method is useful before saving to warn users if content
   * may not display correctly in the rich editor.
   *
   * @param markdown - The markdown to validate
   * @returns ConversionResult with validation status and any warnings
   */
  validateRoundTrip(markdown: string): ConversionResult {
    const result = this.roundTrip(markdown)

    if (!result.success) {
      return result
    }

    // Check for potential issues that don't cause conversion failure
    // but may indicate content problems
    const warnings: string[] = []

    // Check for escaped characters that shouldn't be escaped
    if (result.content.includes('\\*')) {
      warnings.push('Content contains escaped asterisks which may indicate conversion issues')
    }
    if (result.content.includes('\\#')) {
      warnings.push('Content contains escaped hashes which may indicate conversion issues')
    }
    if (result.content.includes('\\[')) {
      warnings.push('Content contains escaped brackets which may indicate conversion issues')
    }

    return {
      content: result.content,
      success: warnings.length === 0,
      errors: warnings,
    }
  }

  // Testing seams - allow injecting custom parsers for testing

  /**
   * Set a custom MarkedParser instance.
   * Primarily for testing purposes.
   *
   * @param parser - The parser instance to use
   */
  setMarkedParser(parser: MarkedParser): void {
    this.markedParser = parser
  }

  /**
   * Set a custom TurndownConverter instance.
   * Primarily for testing purposes.
   *
   * @param converter - The converter instance to use
   */
  setTurndownConverter(converter: TurndownConverter): void {
    this.turndownConverter = converter
  }

  /**
   * Get the current MarkedParser instance.
   */
  getMarkedParser(): MarkedParser {
    return this.markedParser
  }

  /**
   * Get the current TurndownConverter instance.
   */
  getTurndownConverter(): TurndownConverter {
    return this.turndownConverter
  }
}

// Create a singleton instance for convenience
const defaultConverter = new ContentConverter()

/**
 * Convert Markdown to HTML using the default converter.
 *
 * @param markdown - The markdown string to convert
 * @returns The converted HTML string
 */
export function markdownToHtml(markdown: string): string {
  return defaultConverter.markdownToHtml(markdown).content
}

/**
 * Convert HTML to Markdown using the default converter.
 *
 * @param html - The HTML string to convert
 * @returns The converted Markdown string
 */
export function htmlToMarkdown(html: string): string {
  return defaultConverter.htmlToMarkdown(html).content
}
