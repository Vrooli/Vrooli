/**
 * Client-side glob pattern validation utilities.
 *
 * Mirrors the server-side validateGlobs() logic in api/internal/backlog/types.go
 * so the UI can give instant feedback before hitting the API.
 */

export interface GlobLineError {
  line: number;
  error: string;
}

export interface GlobValidationResult {
  valid: boolean;
  errors: GlobLineError[];
}

/**
 * Validate a single glob line. Returns an error string if invalid, undefined if ok.
 */
export function validateGlobLine(line: string): string | undefined {
  const trimmed = line.trim();
  if (trimmed === "") {
    return "empty pattern not allowed";
  }
  if (trimmed.startsWith("/")) {
    return "absolute paths not allowed";
  }
  // Check for unbalanced braces (simple check matching Go's filepath.Match behavior)
  let braceDepth = 0;
  for (let i = 0; i < trimmed.length; i++) {
    const ch = trimmed[i];
    if (ch === "\\") {
      i++; // skip escaped char
      continue;
    }
    if (ch === "{") braceDepth++;
    if (ch === "}") braceDepth--;
    if (braceDepth < 0) return "unbalanced closing brace '}'";
    // Check for invalid bracket pattern (unclosed [)
    if (ch === "[") {
      const closingIdx = trimmed.indexOf("]", i + 1);
      if (closingIdx === -1) return "unclosed bracket '['";
      i = closingIdx;
    }
  }
  if (braceDepth > 0) return "unclosed brace '{'";
  return undefined;
}

/**
 * Validate all lines from a textarea. Returns errors with 1-based line numbers.
 * Empty/whitespace-only lines are skipped (not errors).
 */
export function validateGlobLines(text: string): GlobValidationResult {
  const lines = text.split("\n");
  const errors: GlobLineError[] = [];
  for (let i = 0; i < lines.length; i++) {
    const raw = lines[i] ?? "";
    // Skip blank lines — they are filtered out by parseGlobTextarea
    if (raw.trim() === "") continue;
    const err = validateGlobLine(raw);
    if (err) {
      errors.push({ line: i + 1, error: err });
    }
  }
  return { valid: errors.length === 0, errors };
}

/**
 * Parse a textarea value into an array of glob patterns.
 * Trims each line and filters out empty lines.
 */
export function parseGlobTextarea(text: string): string[] {
  return text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);
}
