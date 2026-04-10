/** Extracts a human-readable message from a react-query error, with a fallback. */
export function formatQueryError(error: unknown, fallback: string): string | null {
  if (!error) return null;
  return error instanceof Error ? error.message : fallback;
}
