/** [start, end) offsets of every case-insensitive occurrence of `query` in `text`. */
export function findRanges(text: string, query: string): Array<[number, number]> {
  const needle = query.trim().toLowerCase();
  if (!needle) return [];
  const haystack = text.toLowerCase();
  const out: Array<[number, number]> = [];
  for (let at = haystack.indexOf(needle); at !== -1; at = haystack.indexOf(needle, at + needle.length)) {
    out.push([at, at + needle.length]);
  }
  return out;
}

/** DOM ranges for every match of `query` in the rendered text under `root`, in document order. */
export function rangesInElement(root: HTMLElement, query: string): Range[] {
  if (!query.trim()) return [];
  const doc = root.ownerDocument;
  const walker = doc.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  const ranges: Range[] = [];
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    const text = node.nodeValue ?? "";
    for (const [start, end] of findRanges(text, query)) {
      const range = doc.createRange();
      range.setStart(node, start);
      range.setEnd(node, end);
      ranges.push(range);
    }
  }
  return ranges;
}

const HIGHLIGHT = "wc-find";
const HIGHLIGHT_ACTIVE = "wc-find-active";

interface HighlightRegistry {
  set: (name: string, highlight: unknown) => void;
  delete: (name: string) => void;
}

function highlightApi(): { registry: HighlightRegistry; Highlight: new (...ranges: Range[]) => unknown } | null {
  const css = (globalThis as { CSS?: { highlights?: HighlightRegistry } }).CSS;
  const Ctor = (globalThis as { Highlight?: new (...ranges: Range[]) => unknown }).Highlight;
  return css?.highlights && Ctor ? { registry: css.highlights, Highlight: Ctor } : null;
}

/**
 * Paints find matches without touching the rendered markdown: the CSS Custom
 * Highlight API where the browser has it, and `<mark>` wrappers otherwise.
 * Returns the element to scroll to for the active match, and a cleanup that
 * removes every highlight.
 */
export function paintMatches(ranges: Range[], activeIndex: number): { activeElement: HTMLElement | null; clear: () => void } {
  const api = highlightApi();
  if (api) {
    api.registry.set(HIGHLIGHT, new api.Highlight(...ranges.filter((_, index) => index !== activeIndex)));
    const active = ranges[activeIndex];
    if (active) api.registry.set(HIGHLIGHT_ACTIVE, new api.Highlight(active));
    const anchor = active?.startContainer.parentElement ?? null;
    return {
      activeElement: anchor,
      clear: () => { api.registry.delete(HIGHLIGHT); api.registry.delete(HIGHLIGHT_ACTIVE); },
    };
  }
  // Fallback: wrap from the last match backwards so earlier offsets stay valid.
  const marks: HTMLElement[] = [];
  for (let index = ranges.length - 1; index >= 0; index -= 1) {
    const range = ranges[index];
    if (!range) continue;
    const mark = range.startContainer.ownerDocument?.createElement("mark");
    if (!mark) continue;
    mark.dataset.findMatch = index === activeIndex ? "active" : "match";
    range.surroundContents(mark);
    marks.unshift(mark);
  }
  return {
    activeElement: marks[activeIndex] ?? null,
    clear: () => {
      for (const mark of marks) {
        const parent = mark.parentNode;
        if (!parent) continue;
        while (mark.firstChild) parent.insertBefore(mark.firstChild, mark);
        parent.removeChild(mark);
        parent.normalize();
      }
    },
  };
}

/** Styles for the Highlight API names; rendered once by the reader. */
export const FIND_HIGHLIGHT_CSS = `::highlight(${HIGHLIGHT}){background-color:rgba(250,204,21,.3);color:inherit}::highlight(${HIGHLIGHT_ACTIVE}){background-color:rgba(250,204,21,.85);color:#111}`;
