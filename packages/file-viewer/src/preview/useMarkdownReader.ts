import { useEffect, useId, useRef, useState } from "react";
import { createSlugger } from "../lib/slugger.js";

export interface ReaderHeading { id: string; label: string; level: number; slug: string }

// The outline is re-derived whenever the article's DOM mutates — which, in a
// document with many lazily-resolved images, happens repeatedly as each image
// swaps in. Returning the previous array when nothing changed keeps those
// mutations from re-rendering the reader chrome on every image load.
function sameHeadings(a: ReaderHeading[], b: ReaderHeading[]): boolean {
  if (a.length !== b.length) return false;
  return a.every((heading, index) => {
    const other = b[index];
    return other !== undefined && heading.id === other.id && heading.label === other.label && heading.level === other.level && heading.slug === other.slug;
  });
}

export function useMarkdownReader(content: string, query: string, source: boolean, preserveIds = false) {
  const articleRef = useRef<HTMLDivElement>(null);
  const [headings, setHeadings] = useState<ReaderHeading[]>([]);
  const [matches, setMatches] = useState<Range[]>([]);
  const [selected, setSelected] = useState(0);
  const prefix = `fv-reader-${useId().replace(/[^a-z0-9]/gi, "")}`;
  const activeName = `${prefix}-active`;
  useEffect(() => {
    const article = articleRef.current;
    if (!article || source) { setHeadings([]); setMatches([]); setSelected(0); return; }
    let frame = 0, firstScan = true;
    const scan = () => {
      frame = 0;
      const slugger = createSlugger();
      const outline = Array.from(article.querySelectorAll<HTMLHeadingElement>("h1,h2,h3,h4,h5,h6")).map((element, index) => {
        const label = element.textContent.trim();
        const id = preserveIds && element.id ? element.id : `${prefix}-section-${String(index)}`;
        element.id = id;
        element.tabIndex = -1;
        return { id, label, level: Number(element.tagName.slice(1)), slug: slugger.slug(label) };
      });
      setHeadings((previous) => sameHeadings(previous, outline) ? previous : outline);
      const ranges: Range[] = [];
      if (query.trim()) {
        const segments: { text: Text; start: number; end: number }[] = [];
        const walker = document.createTreeWalker(article, NodeFilter.SHOW_TEXT);
        let buffer = "", previousBlock: Element | null = null;
        while (walker.nextNode()) {
          const node = walker.currentNode;
          if (!(node instanceof Text) || !node.parentElement || node.parentElement.closest("button,svg,script,style,textarea,[aria-hidden=true],[hidden],[role=status],.sr-only,.rcl-visually-hidden")) continue;
          const block = node.parentElement.closest("p,h1,h2,h3,h4,h5,h6,li,pre,td,th,blockquote");
          if (previousBlock && block !== previousBlock) buffer += "\n";
          previousBlock = block;
          const start = buffer.length; buffer += node.data;
          segments.push({ text: node, start, end: buffer.length });
        }
        const pattern = new RegExp(query.trim().replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), "giu");
        for (const found of buffer.matchAll(pattern)) {
          if (ranges.length >= 500) break;
          const start = found.index, end = start + found[0].length;
          const first = segments.find((segment) => segment.start <= start && segment.end > start);
          const last = segments.find((segment) => segment.start < end && segment.end >= end);
          if (!first || !last) continue;
          const range = document.createRange(); range.setStart(first.text, start - first.start); range.setEnd(last.text, end - last.start); ranges.push(range);
        }
      }
      setMatches(ranges); setSelected((current) => Math.max(0, Math.min(current, ranges.length - 1)));
      if (firstScan && query) ranges[0]?.startContainer.parentElement?.scrollIntoView({ block: "center" });
      firstScan = false;
    };
    const timer = window.setTimeout(scan, 150);
    const observer = new MutationObserver(() => { if (!frame) frame = requestAnimationFrame(scan); });
    observer.observe(article, { childList: true, characterData: true, subtree: true });
    return () => { window.clearTimeout(timer); cancelAnimationFrame(frame); observer.disconnect(); };
  }, [content, query, source, prefix, preserveIds]);
  useEffect(() => { setSelected(0); }, [query]);
  useEffect(() => {
    const supported = typeof CSS !== "undefined" && "highlights" in CSS && typeof Highlight !== "undefined";
    if (!supported) return;
    const highlights = CSS.highlights;
    highlights.set(prefix, new Highlight(...matches));
    const active = matches[selected];
    highlights.set(activeName, new Highlight(...(active ? [active] : [])));
    return () => { highlights.delete(prefix); highlights.delete(activeName); };
  }, [matches, selected, prefix, activeName]);
  const reveal = (id: string) => {
    const element = document.getElementById(id);
    element?.scrollIntoView({ block: "start" }); element?.focus({ preventScroll: true });
  };
  const next = (direction: number) => {
    if (!matches.length) return;
    const index = (selected + direction + matches.length) % matches.length;
    setSelected(index);
    matches[index]?.startContainer.parentElement?.scrollIntoView({ block: "center" });
  };
  return { articleRef, headings, matches: matches.length, selected, next, reveal,
    highlightStyle: `::highlight(${prefix}){background-color:rgb(var(--fv-accent) / .25);color:rgb(var(--fv-text-primary))}::highlight(${activeName}){background-color:rgb(var(--fv-accent) / .55);color:rgb(var(--fv-text-primary))}` };
}

/** The heading a link fragment or an anchor target names: by slug (`plan-1`) or by DOM id. */
export function findHeading(headings: ReaderHeading[], fragment: string): ReaderHeading | undefined {
  let target = fragment.replace(/^#/, "");
  try { target = decodeURIComponent(target); } catch { /* Use the literal fragment. */ }
  return headings.find((heading) => heading.slug === target || heading.id === target);
}

/**
 * Scroll to the heading an anchor names once the reader has found its
 * headings, once per anchor. A later anchor scrolls again; clearing it re-arms.
 */
export function useRevealAnchor(anchor: string | undefined, reader: Pick<ReturnType<typeof useMarkdownReader>, "headings" | "reveal">) {
  const revealed = useRef<string | null>(null), latest = useRef(reader);
  latest.current = reader;
  const { headings } = reader;
  useEffect(() => {
    if (!anchor) { revealed.current = null; return; }
    if (revealed.current === anchor) return;
    const heading = findHeading(headings, anchor);
    if (!heading) return;
    revealed.current = anchor;
    latest.current.reveal(heading.id);
  }, [anchor, headings]);
}
