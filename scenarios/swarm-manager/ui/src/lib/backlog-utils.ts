/**
 * Backlog utilities shared across UI components.
 */

export function sanitizeBacklogName(value: string): string {
  return value
    .toLowerCase()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9-]/g, "");
}

export function parseTagsInput(value: string): string[] {
  const rawTags = value
    .split(",")
    .map((tag) => tag.trim())
    .filter((tag) => tag.length > 0);

  const seen = new Set<string>();
  const result: string[] = [];
  for (const tag of rawTags) {
    const key = tag.toLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    result.push(tag);
  }
  return result;
}

export function tagsToInput(tags: string[]): string {
  return tags.join(", ");
}
