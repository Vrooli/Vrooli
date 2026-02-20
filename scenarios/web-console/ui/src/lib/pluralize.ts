// DOC: docs/internal/UTILS_UNIFICATION_NOTES.md
/**
 * Append "s" to a word when count !== 1.
 *
 * Covers the simple English plural case used throughout the UI
 * (e.g., "1 terminal" vs "2 terminals", "1 session" vs "3 sessions").
 */
export function pluralize(count: number, singular: string): string {
  return count === 1 ? singular : `${singular}s`;
}
