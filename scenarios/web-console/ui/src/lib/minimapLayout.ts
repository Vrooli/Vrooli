export type MinimapRowMarker = {
  rowIndex: number;
  topPercent: number;
  heightPercent: number;
};

export type MinimapViewport = {
  topPercent: number;
  heightPercent: number;
  maxScrollable: number;
};

/** Minimum scrollable overflow (px) before the minimap renders. */
export const MINIMAP_MIN_OVERFLOW_PX = 8;

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

/** Map row count to evenly-spaced minimap marker descriptors. */
export function buildMinimapRowMarkers(rowCount: number): MinimapRowMarker[] {
  const safe = Number.isFinite(rowCount) ? Math.floor(rowCount) : 0;
  if (safe <= 0) return [];
  const h = 100 / safe;
  return Array.from({ length: safe }, (_, i) => ({
    rowIndex: i,
    topPercent: i * h,
    heightPercent: h,
  }));
}

/** Calculate viewport indicator position from scroll metrics. */
export function viewportFromScrollMetrics({
  scrollTop,
  scrollHeight,
  clientHeight,
  minViewportPercent = 8,
}: {
  scrollTop: number;
  scrollHeight: number;
  clientHeight: number;
  minViewportPercent?: number;
}): MinimapViewport {
  const safeScrollHeight = Math.max(1, Number.isFinite(scrollHeight) ? scrollHeight : 1);
  const safeClientHeight = Math.max(1, Number.isFinite(clientHeight) ? clientHeight : 1);
  const safeMinPercent = clamp(
    Number.isFinite(minViewportPercent) ? minViewportPercent : 8,
    1,
    100,
  );

  const maxScrollable = Math.max(safeScrollHeight - safeClientHeight, 0);
  const heightPercent = clamp(
    (safeClientHeight / safeScrollHeight) * 100,
    safeMinPercent,
    100,
  );
  const maxTopPercent = Math.max(0, 100 - heightPercent);
  const safeScrollTop = clamp(
    Number.isFinite(scrollTop) ? scrollTop : 0,
    0,
    maxScrollable,
  );
  const topPercent =
    maxScrollable <= 0 ? 0 : (safeScrollTop / maxScrollable) * maxTopPercent;

  return { topPercent, heightPercent, maxScrollable };
}

/** Convert a pointer Y offset on the minimap rail to a scrollTop value. */
export function scrollTopFromMinimapPointer(
  pointerOffsetY: number,
  railHeight: number,
  scrollHeight: number,
  clientHeight: number,
): number {
  if (!Number.isFinite(railHeight) || railHeight <= 0) return 0;
  const ratio = clamp(pointerOffsetY / railHeight, 0, 1);
  const maxScrollable = Math.max(scrollHeight - clientHeight, 0);
  return ratio * maxScrollable;
}
