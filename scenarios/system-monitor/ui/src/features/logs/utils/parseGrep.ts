/**
 * Parse a user-entered grep pattern.
 *
 * The backend hands the string through to `journalctl -g` which uses
 * Perl-compatible regex. We don't translate the syntax — we just sanity-check
 * that the pattern compiles as a JS regex (close enough for early feedback)
 * and trim whitespace. Empty input is `{ pattern: '' }`, never an error.
 */

export interface ParsedGrep {
  pattern: string;
  error?: string;
}

export function parseGrep(input: string): ParsedGrep {
  const trimmed = input.trim();
  if (trimmed === '') return { pattern: '' };
  try {
    // Validate as a regex; we don't actually use the RegExp object — just
    // surface compile errors to the user before they hit the backend.
    new RegExp(trimmed);
    return { pattern: trimmed };
  } catch (err) {
    return {
      pattern: trimmed,
      error: err instanceof Error ? err.message : 'invalid regex',
    };
  }
}
