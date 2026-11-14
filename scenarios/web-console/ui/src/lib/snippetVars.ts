// Snippet variable substitution.
//
// This module deliberately has zero dependencies. It only recognizes names
// and supplied string values; it cannot learn what a value means or grow a
// branch for a workflow, file shape, session type, or destination.
// DOC: docs/internal/SNIPPETS-AND-MESSAGE-ACTIONS-UX.md

/** Lowercase named tokens are the complete snippet language. */
export const VARIABLE_PATTERN = /\{\{([a-z][a-z0-9_]*)\}\}/g;

/** The closed set the composer may populate without asking the operator. */
export const AUTO_VARIABLES = Object.freeze([
  "payload",
  "cwd",
  "session",
  "selection",
] as const);

/** Returns valid names once, in the order they first appear. */
export function distinctVariables(body: string): string[] {
  const seen = new Set<string>();
  const names: string[] = [];
  for (const match of body.matchAll(VARIABLE_PATTERN)) {
    const name = match[1];
    if (!name) continue;
    if (!seen.has(name)) {
      seen.add(name);
      names.push(name);
    }
  }
  return names;
}

/**
 * Substitutes only explicitly supplied names. An absent name stays visible so
 * unfinished text remains honest; an explicitly supplied empty string is a
 * deliberate value and therefore replaces the token.
 */
export function renderSnippet(body: string, values: Record<string, string>): string {
  return body.replace(VARIABLE_PATTERN, (token, name: string) =>
    Object.prototype.hasOwnProperty.call(values, name) ? values[name] as string : token,
  );
}
