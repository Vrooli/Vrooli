/**
 * Content conversion domain.
 *
 * Provides robust, testable content conversion between Markdown and HTML.
 */

export {
  ContentConverter,
  isHtml,
  markdownToHtml,
  htmlToMarkdown,
  type ConversionResult,
  type ContentConverterConfig,
} from './ContentConverter'

export { MarkedParser, type MarkedParserConfig } from './markdown-to-html'
export { TurndownConverter, type TurndownConverterConfig } from './html-to-markdown'
export {
  validateMarkdown,
  type MarkdownIssue,
  type ValidationResult,
} from './validation'
