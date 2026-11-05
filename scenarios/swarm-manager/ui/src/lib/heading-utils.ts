/**
 * Heading extraction and slug utilities for markdown TOC generation.
 *
 * Used by the PlanPanel TOC popover to build a stable navigation list. Heading IDs
 * are derived with the shared slugify() and dedup counter logic.
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
 * Returns entries with deterministic, deduplicated anchor IDs.
 */
export function extractHeadings(markdown: string): HeadingEntry[] {
  const lines = markdown.split("\n");
  const headings: HeadingEntry[] = [];
  const seen = new Map<string, number>();
  let inCodeFence = false;

  for (const [index, line] of lines.entries()) {
    const lineNumber = index + 1;

    if (CODE_FENCE_RE.test(line)) {
      inCodeFence = !inCodeFence;
      continue;
    }

    if (inCodeFence) continue;

    const match = HEADING_RE.exec(line);
    if (match) {
      const [, levelText, headingText] = match;
      if (!levelText || !headingText) {
        continue;
      }

      const level = levelText.length as 1 | 2 | 3;
      const text = headingText.trim();
      const slug = slugify(text);
      const id = dedupeSlug(slug, seen);

      headings.push({ level, text, id, line: lineNumber });
    }
  }

  return headings;
}
