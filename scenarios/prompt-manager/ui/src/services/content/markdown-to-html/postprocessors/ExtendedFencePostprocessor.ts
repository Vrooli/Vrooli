/**
 * ExtendedFencePostprocessor - Post-processes extended code fences.
 *
 * Since the preprocessor now outputs HTML directly for extended fences,
 * this postprocessor is a no-op but kept for consistency.
 */

/**
 * Post-process extended fences.
 *
 * Currently a no-op since the preprocessor handles everything.
 * Kept for potential future use.
 *
 * @param html - The HTML content from marked
 * @returns The HTML unchanged
 */
export function postprocessExtendedFences(html: string): string {
  return html
}
