/**
 * Heading slugs the way GitHub numbers them, so `#plan` and `#plan-1` in a
 * document resolve to the first and second "Plan" heading. Lower-case, drop
 * everything but letters, marks, numbers, `_` and `-`, turn each space into
 * `-` (runs are not collapsed: "a  b" is "a--b"), then make duplicates unique
 * with a counter. Matches the `github-slugger` algorithm for ordinary text.
 */
const REMOVED = /[^\p{L}\p{M}\p{N}\p{Pc}\- ]/gu;

export function slugify(value: string): string {
  return value.toLowerCase().replace(REMOVED, "").replace(/ /g, "-");
}

export interface Slugger {
  /** The next unique slug for a heading's text; call once per heading in document order. */
  slug(value: string): string;
}

export function createSlugger(): Slugger {
  const occurrences = new Map<string, number>();
  return {
    slug(value) {
      const original = slugify(value);
      let slug = original;
      while (occurrences.has(slug)) {
        const next = (occurrences.get(original) ?? 0) + 1;
        occurrences.set(original, next);
        slug = `${original}-${String(next)}`;
      }
      occurrences.set(slug, 0);
      return slug;
    },
  };
}
