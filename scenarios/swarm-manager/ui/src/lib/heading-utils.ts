/**
 * Heading extraction and slug utilities for markdown TOC generation.
 *
 * Used by both renderMarkdown (to generate anchor IDs) and PlanPanel TOC popover
 * (to build the navigation list). Both consumers MUST produce identical IDs, which
 * is guaranteed by sharing slugify() and the same dedup counter logic.
 */

export interface HeadingEntry {
  level: 1 | 2 | 3;
  text: string;
  /** Slugified, deduplicated anchor ID */
  id: string;
  /** 1-based line number in the source markdown */
  line: number;
}

/** Convert heading text to a URL-safe anchor slug. */
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "");
}

/**
 * Deduplicate a slug against a map of seen slugs.
 * Mutates `seen` by incrementing the counter. Returns the unique slug.
 */
export function dedupeSlug(slug: string, seen: Map<string, number>): string {
  const count = seen.get(slug) ?? 0;
  seen.set(slug, count + 1);
  return count === 0 ? slug : `${slug}-${count}`;
}

const HEADING_RE = /^(#{1,3})\s+(.+)$/;
const CODE_FENCE_RE = /^```/;

/**
 * Extract headings (h1–h3) from raw markdown, skipping code fences.
 * Returns entries with deduplicated anchor IDs matching renderMarkdown output.
 */
export function extractHeadings(markdown: string): HeadingEntry[] {
  const lines = markdown.split("\n");
  const headings: HeadingEntry[] = [];
  const seen = new Map<string, number>();
  let inCodeFence = false;

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    if (CODE_FENCE_RE.test(line)) {
      inCodeFence = !inCodeFence;
      continue;
    }

    if (inCodeFence) continue;

    const match = HEADING_RE.exec(line);
    if (match) {
      const level = match[1].length as 1 | 2 | 3;
      const text = match[2].trim();
      const slug = slugify(text);
      const id = dedupeSlug(slug, seen);

      headings.push({ level, text, id, line: i + 1 });
    }
  }

  return headings;
}
