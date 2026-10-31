/**
 * Layout arithmetic for a landscape room that never scrolls. Pure so the
 * paging and auto-scroll decisions are testable without a layout engine;
 * the components measure the DOM and hand the boxes in.
 */

/** How long an auto-scroll list shows its first rows, each step and its last rows, and how long it fades back to the top. */
export const AUTOSCROLL_TIMING = { holdStartMs: 4000, stepMs: 1800, holdEndMs: 3000, fadeMs: 450 };

/** How long each page of a paged strip stays up. */
export const STRIP_PAGE_MS = 7000;

export interface RowBox {
  top: number;
  height: number;
}

/** Group laid-out boxes into visual rows by their top edge. */
export function rowsOf(boxes: RowBox[], tolerance = 2): RowBox[] {
  const rows: RowBox[] = [];
  for (const box of [...boxes].sort((a, b) => a.top - b.top)) {
    const row = rows.find((entry) => Math.abs(entry.top - box.top) <= tolerance);
    if (row) row.height = Math.max(row.height, box.height);
    else rows.push({ top: box.top, height: box.height });
  }
  return rows;
}

/**
 * Split rows into pages that each fit `available` pixels and return how many
 * rows each page holds. A row taller than the budget still gets a page of its
 * own rather than being dropped.
 */
export function pageRows(rows: RowBox[], available: number, gap: number): number[] {
  const pages: number[] = [];
  let used = 0;
  let count = 0;
  for (const row of rows) {
    const need = count ? used + gap + row.height : row.height;
    if (count && need > available + 0.5) {
      pages.push(count);
      used = row.height;
      count = 1;
    } else {
      used = need;
      count += 1;
    }
  }
  if (count) pages.push(count);
  return pages;
}

/** Tile index ranges [start, end) for each page, given rows per page and a fixed column count. */
export function pageRanges(rowsPerPage: number[], columns: number, total: number, maxItems = Number.POSITIVE_INFINITY): Array<[number, number]> {
  const ranges: Array<[number, number]> = [];
  let start = 0;
  rowsPerPage.forEach((rows, index) => {
    // The last page takes every remaining tile, so a short final row is never dropped.
    const end = index === rowsPerPage.length - 1 ? total : Math.min(total, start + rows * Math.max(1, columns));
    let pageStart = start;
    while (pageStart < end) {
      const pageEnd = Math.min(end, pageStart + Math.max(1, maxItems));
      ranges.push([pageStart, pageEnd]);
      pageStart = pageEnd;
    }
    start = end;
  });
  return ranges.length ? ranges : [[0, total]];
}

/** Hard desktop safety ceiling; measured height may choose an even smaller page. */
export const MAX_DESKTOP_STRIP_ITEMS = 8;

/**
 * The offsets an auto-scroll viewport rests at. Row by row it stops at each
 * row's top until the last row is in view; by page (reduced motion) it stops at
 * the first row not yet fully shown. The first stop is always 0 and the last is
 * always the end, so every row is seen once per pass.
 */
export function scrollStops(rows: RowBox[], contentHeight: number, viewportHeight: number, byPage = false): number[] {
  const end = Math.max(0, contentHeight - viewportHeight);
  if (end <= 1) return [0];
  const stops = [0];
  const last = () => stops[stops.length - 1] ?? 0;
  for (const row of rows) {
    if (last() >= end) break;
    if (row.top <= last() + 0.5) continue;
    if (byPage && row.top + row.height <= last() + viewportHeight + 0.5) continue;
    stops.push(Math.min(row.top, end));
  }
  if (last() < end) stops.push(end);
  return stops;
}

/** 1-based [first, last] of the rows wholly inside the viewport at `offset`, or null when none is. */
export function visibleRange(rows: RowBox[], offset: number, viewportHeight: number): [number, number] | null {
  let first = -1;
  let lastIndex = -1;
  rows.forEach((row, index) => {
    if (row.top >= offset - 0.5 && row.top + row.height <= offset + viewportHeight + 0.5) {
      if (first < 0) first = index;
      lastIndex = index;
    }
  });
  return first < 0 ? null : [first + 1, lastIndex + 1];
}
