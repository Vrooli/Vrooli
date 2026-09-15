import { forwardRef, useCallback, useEffect, useId, useLayoutEffect, useRef, useState, type CSSProperties, type ReactElement } from "react";
import { useBeatHold } from "../lib/boardContext";
import { STRIP_PAGE_MS, pageRanges, pageRows, rowsOf } from "../lib/fit";
import { useLandscapeRoom } from "../lib/media";

type Density = "normal" | "compact";

interface StripLayout {
  density: Density;
  columns: number;
  ranges: Array<[number, number]>;
  /** The tallest page's content height: every page reserves it, so a page flip never resizes the hero. */
  height: number;
}

const px = (value: string): number => {
  const parsed = Number.parseFloat(value);
  return Number.isFinite(parsed) ? parsed : 0;
};

/**
 * The supporting strip of a landscape room. It lays every tile out once to
 * measure, tightens to a compact density if the tiles overrun the strip's
 * height budget, and pages what still does not fit, holding the column count
 * across pages. Portrait shows every tile; the page scrolls there.
 */
export const SupportingStrip = forwardRef<HTMLUListElement, { children: ReactElement[] }>(function SupportingStrip({ children }, forwardedRef) {
  const id = useId();
  const hold = useBeatHold();
  const landscape = useLandscapeRoom();
  const listRef = useRef<HTMLUListElement | null>(null);
  const [trial, setTrial] = useState<Density>("normal");
  const [layout, setLayout] = useState<StripLayout | null>(null);
  const [page, setPage] = useState(0);
  const [readAll, setReadAll] = useState(false);
  const overflowChecks = useRef(0);
  const signature = children.map((child) => child.key).join("|");

  const setRefs = useCallback((node: HTMLUListElement | null) => {
    listRef.current = node;
    if (typeof forwardedRef === "function") forwardedRef(node);
    else if (forwardedRef) forwardedRef.current = node;
  }, [forwardedRef]);

  const remeasure = useCallback(() => {
    setTrial("normal");
    setLayout(null);
  }, []);

  // A new set of readings or a new orientation starts from a fresh measurement.
  useLayoutEffect(() => {
    overflowChecks.current = 0;
    remeasure();
    setPage(0);
  }, [landscape, remeasure, signature]);

  // Measure pass: every tile is laid out at the trial density, then split into pages.
  useLayoutEffect(() => {
    const list = listRef.current;
    if (layout || !list || !landscape) return;
    const style = getComputedStyle(list);
    const budget = px(style.maxHeight) - px(style.paddingTop) - px(style.borderTopWidth);
    const origin = list.getBoundingClientRect().top + px(style.paddingTop) + px(style.borderTopWidth);
    const boxes = Array.from(list.children, (tile) => {
      const box = tile.getBoundingClientRect();
      return { top: box.top - origin, height: box.height };
    });
    const rows = rowsOf(boxes);
    const columns = Math.max(1, boxes.filter((box) => Math.abs(box.top - (rows[0]?.top ?? 0)) <= 2).length);
    const gap = px(style.rowGap);
    const pages = budget > 0 ? pageRows(rows, budget, gap) : [rows.length];
    if (pages.length > 1 && trial === "normal") {
      setTrial("compact");
      return;
    }
    let first = 0;
    const heights = pages.map((count) => {
      const page = rows.slice(first, first + count);
      first += count;
      return page.reduce((sum, row) => sum + row.height, 0) + gap * Math.max(0, page.length - 1);
    });
    setLayout({ density: trial, columns, ranges: pageRanges(pages, columns, boxes.length), height: Math.max(0, ...heights) });
  }, [landscape, layout, trial]);

  // A tile that grows after the measurement (a qualifier line appearing) re-measures, twice at most per reading set.
  useLayoutEffect(() => {
    const list = listRef.current;
    if (!layout || !list || !landscape || overflowChecks.current >= 2) return;
    if (list.scrollHeight > list.clientHeight + 1 && px(getComputedStyle(list).maxHeight) > 0) {
      overflowChecks.current += 1;
      remeasure();
    }
  });

  useEffect(() => {
    if (!landscape || typeof ResizeObserver === "undefined") return;
    const region = listRef.current?.parentElement;
    if (!region) return;
    let width = region.clientWidth;
    let height = window.innerHeight;
    const observer = new ResizeObserver(() => {
      if (region.clientWidth === width && window.innerHeight === height) return;
      width = region.clientWidth;
      height = window.innerHeight;
      overflowChecks.current = 0;
      remeasure();
    });
    observer.observe(region);
    return () => observer.disconnect();
  }, [landscape, remeasure]);

  const pages = layout?.ranges.length ?? 1;
  useEffect(() => {
    setReadAll(pages <= 1);
    if (pages <= 1) return;
    const flip = window.setInterval(() => setPage((current) => (current + 1) % pages), STRIP_PAGE_MS);
    const done = window.setTimeout(() => setReadAll(true), pages * STRIP_PAGE_MS);
    return () => {
      window.clearInterval(flip);
      window.clearTimeout(done);
    };
  }, [pages]);

  useEffect(() => {
    hold(id, pages > 1 && !readAll);
    return () => hold(id, false);
  }, [hold, id, pages, readAll]);

  const paged = landscape && layout !== null && pages > 1;
  const range = layout?.ranges[Math.min(page, pages - 1)] ?? [0, children.length];
  const shown = paged ? children.slice(range[0], range[1]) : children;
  const density = landscape ? layout?.density ?? trial : "normal";
  return (
    <>
      {paged ? <span className="cc-strip-pages" data-testid="strip-pages" aria-label={`Page ${page + 1} of ${pages}`}>{page + 1} / {pages}</span> : null}
      <ul
        ref={setRefs}
        className="cc-readings"
        data-testid="metric-list"
        data-density={density}
        data-paged={paged || undefined}
        data-measuring={(landscape && layout === null) || undefined}
        style={paged ? ({ "--strip-columns": layout.columns, minHeight: `${Math.ceil(layout.height)}px` } as CSSProperties) : undefined}
      >
        {shown}
      </ul>
    </>
  );
});
