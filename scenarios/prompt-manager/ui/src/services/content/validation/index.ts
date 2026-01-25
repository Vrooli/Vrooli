/**
 * Markdown validation for content that may not survive HTML round-trip.
 */

export {
  validateMarkdown,
  validateRoundTrip,
  normalizeForComparison,
  type IssueType,
  type MarkdownIssue,
  type ValidationResult,
  type RoundTripResult,
} from './MarkdownValidator'
