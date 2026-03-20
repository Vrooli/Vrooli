/**
 * Trigger a browser download of a JSON-serializable value.
 *
 * Extracted from ExportButton to make download logic reusable and testable.
 */
export function downloadJSON(data: unknown, filename: string): void {
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

/**
 * Convert a human-readable name to a filename-safe slug.
 *
 * Example: "My Scheme Name" → "my-scheme-name"
 */
export function slugify(name: string): string {
  return name.replace(/\s+/g, "-").toLowerCase();
}
