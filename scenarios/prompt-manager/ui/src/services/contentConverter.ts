/**
 * Content Converter - Re-exports from the content module.
 *
 * This file provides backward compatibility for existing imports.
 * New code should import directly from '@/services/content'.
 *
 * Handles:
 * - Markdown to HTML conversion for TipTap rendering (using marked library)
 * - HTML to Markdown conversion for storage (using turndown)
 * - Code block language preservation in both directions
 * - GFM support for tables, strikethrough, etc.
 */

export {
  isHtml,
  markdownToHtml,
  htmlToMarkdown,
  ContentConverter,
  type ConversionResult,
  type ContentConverterConfig,
} from './content'
