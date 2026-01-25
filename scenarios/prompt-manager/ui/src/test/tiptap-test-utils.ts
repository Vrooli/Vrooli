/**
 * TipTap Test Utilities
 *
 * Provides utilities for testing with real TipTap editor instances.
 * This enables integration testing of the full content flow:
 * Markdown → markdownToHtml → TipTap.setContent → TipTap.getHTML → htmlToMarkdown → Markdown
 *
 * IMPORTANT: Uses the SAME extensions as production to ensure test accuracy.
 * ReactNodeViewRenderer components won't render in headless mode, but the
 * HTML parsing/serialization logic will be identical.
 */

import { Editor } from '@tiptap/react'
import { createTipTapExtensions } from '@/components/editor/tiptap'
import { markdownToHtml, htmlToMarkdown } from '@/services/content'

/**
 * Result of a round-trip through TipTap.
 */
export interface TipTapRoundTripResult {
  /** The HTML produced by markdownToHtml() */
  inputHtml: string
  /** The HTML produced by TipTap's getHTML() after setContent */
  tiptapHtml: string
  /** The markdown produced by htmlToMarkdown() from TipTap's HTML */
  outputMarkdown: string
}

/**
 * Options for creating a test editor.
 */
export interface TestEditorOptions {
  /** Initial content for the editor */
  content?: string
  /** Whether the content is already HTML */
  isHtml?: boolean
}

/**
 * Create a TipTap editor instance for testing.
 *
 * Uses EXACTLY the same extensions as production to ensure test accuracy.
 * ReactNodeViewRenderer components won't render in headless mode, but the
 * HTML parsing/serialization logic (which is what we're testing) will be identical.
 *
 * @param options - Configuration options
 * @returns A configured TipTap Editor instance
 */
export function createTestEditor(options: TestEditorOptions = {}): Editor {
  const { content = '', isHtml = false } = options

  // Convert markdown to HTML if needed
  const htmlContent = isHtml ? content : (content ? markdownToHtml(content) : '')

  // Use EXACTLY the same extensions as production
  // This ensures tests catch any configuration mismatches
  const editor = new Editor({
    extensions: createTipTapExtensions({ placeholder: '' }),
    content: htmlContent,
    enableInputRules: true,
    enablePasteRules: true,
  })

  return editor
}

/**
 * Destroy a test editor instance.
 *
 * @param editor - The editor to destroy
 */
export function destroyTestEditor(editor: Editor): void {
  editor.destroy()
}

/**
 * Perform a round-trip conversion through TipTap.
 *
 * This simulates the full content flow:
 * 1. Markdown → markdownToHtml() → HTML
 * 2. HTML → TipTap.setContent() → TipTap normalizes/transforms HTML
 * 3. TipTap.getHTML() → HTML (may differ from input)
 * 4. HTML → htmlToMarkdown() → Markdown
 *
 * @param markdown - The markdown content to round-trip
 * @returns Object containing all intermediate and final results
 */
export function roundTripThroughTipTap(markdown: string): TipTapRoundTripResult {
  // Step 1: Convert markdown to HTML
  const inputHtml = markdownToHtml(markdown)

  // Step 2 & 3: Create editor, set content, get HTML
  const editor = createTestEditor({ content: inputHtml, isHtml: true })
  const tiptapHtml = editor.getHTML()
  destroyTestEditor(editor)

  // Step 4: Convert TipTap's HTML back to markdown
  const outputMarkdown = htmlToMarkdown(tiptapHtml)

  return {
    inputHtml,
    tiptapHtml,
    outputMarkdown,
  }
}

/**
 * Set HTML content in an editor and get the resulting HTML.
 *
 * This is useful for testing how TipTap normalizes/transforms HTML.
 *
 * @param editor - The TipTap editor instance
 * @param html - The HTML content to set
 * @returns The HTML from TipTap after setting the content
 */
export function setContentAndGetHtml(editor: Editor, html: string): string {
  editor.commands.setContent(html)
  return editor.getHTML()
}

/**
 * Test if content is idempotent through TipTap round-trips.
 *
 * Content is idempotent if after the first round-trip, subsequent
 * round-trips produce the same result.
 *
 * @param markdown - The markdown to test
 * @param iterations - Number of round-trips to perform (default: 3)
 * @returns True if content stabilizes, false if it keeps changing
 */
export function isIdempotentThroughTipTap(
  markdown: string,
  iterations: number = 3
): boolean {
  let content = markdown

  for (let i = 0; i < iterations; i++) {
    const result = roundTripThroughTipTap(content)

    // After first iteration, content should stabilize
    if (i > 0 && result.outputMarkdown !== content) {
      return false
    }

    content = result.outputMarkdown
  }

  return true
}

/**
 * Compare HTML output to understand TipTap transformations.
 *
 * @param inputHtml - The HTML before TipTap processing
 * @param tiptapHtml - The HTML after TipTap processing
 * @returns Object describing differences
 */
export function compareHtmlTransformation(
  inputHtml: string,
  tiptapHtml: string
): {
  isIdentical: boolean
  inputNormalized: string
  outputNormalized: string
} {
  // Normalize whitespace for comparison
  const normalize = (html: string) =>
    html.replace(/\s+/g, ' ').replace(/>\s+</g, '><').trim()

  const inputNormalized = normalize(inputHtml)
  const outputNormalized = normalize(tiptapHtml)

  return {
    isIdentical: inputNormalized === outputNormalized,
    inputNormalized,
    outputNormalized,
  }
}

/**
 * Create multiple test cases for batch testing.
 */
export interface TestCase {
  name: string
  markdown: string
  expectedPreservations?: string[]
  expectedNotContains?: string[]
}

/**
 * Run multiple test cases through TipTap round-trip.
 *
 * @param testCases - Array of test cases to run
 * @returns Results for each test case
 */
export function runTestCases(testCases: TestCase[]): Array<{
  testCase: TestCase
  result: TipTapRoundTripResult
  preservationsPassed: boolean
  notContainsPassed: boolean
}> {
  return testCases.map(testCase => {
    const result = roundTripThroughTipTap(testCase.markdown)

    const preservationsPassed = testCase.expectedPreservations?.every(
      expected => result.outputMarkdown.includes(expected)
    ) ?? true

    const notContainsPassed = testCase.expectedNotContains?.every(
      notExpected => !result.outputMarkdown.includes(notExpected)
    ) ?? true

    return {
      testCase,
      result,
      preservationsPassed,
      notContainsPassed,
    }
  })
}
