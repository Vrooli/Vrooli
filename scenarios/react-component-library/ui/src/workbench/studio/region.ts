/**
 * Region geometry for the Assets pane's image capture.
 *
 * Every function here is pure and works in whole pixels. The studio keeps the
 * marquee in the image's **natural** pixel space, so the same region survives
 * preview scaling: pointer coordinates are converted once (see
 * `regionToPixels`), and cropping can then use the rectangle verbatim.
 *
 * Rules the callers rely on:
 *   - a region always stays inside `0..bounds.width` / `0..bounds.height`;
 *   - it is never smaller than `MIN_REGION` on either axis (unless the image
 *     itself is smaller than that);
 *   - resize handles move one edge at a time and never flip the rectangle.
 */

export interface Region {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface RegionBounds {
  width: number;
  height: number;
}

export type ResizeHandle = "nw" | "ne" | "sw" | "se";

/** Smallest marquee the studio accepts, in natural image pixels. */
export const MIN_REGION = 16;

function round(value: number): number {
  return Math.round(Number.isFinite(value) ? value : 0);
}

function clamp(value: number, low: number, high: number): number {
  if (high < low) return low;
  return Math.min(high, Math.max(low, value));
}

function dimensions(bounds: RegionBounds): { width: number; height: number } {
  return {
    width: Math.max(0, round(bounds.width)),
    height: Math.max(0, round(bounds.height)),
  };
}

/**
 * Normalise a region into whole pixels, inside the image, at or above the
 * minimum size. When a minimum would push the rectangle past an edge, the
 * rectangle is nudged inwards instead of overflowing.
 */
export function clampRegion(
  region: Region,
  bounds: RegionBounds,
  min: number = MIN_REGION,
): Region {
  const { width, height } = dimensions(bounds);
  const minWidth = Math.min(Math.max(0, round(min)), width);
  const minHeight = Math.min(Math.max(0, round(min)), height);
  const w = clamp(round(region.w), minWidth, width);
  const h = clamp(round(region.h), minHeight, height);
  const x = clamp(round(region.x), 0, width - w);
  const y = clamp(round(region.y), 0, height - h);
  return { x, y, w, h };
}

/** Translate a region without changing its size, keeping it in bounds. */
export function moveRegion(
  region: Region,
  dx: number,
  dy: number,
  bounds: RegionBounds,
): Region {
  const base = clampRegion(region, bounds);
  return clampRegion({ ...base, x: base.x + dx, y: base.y + dy }, bounds);
}

/**
 * Drag one corner by (`dx`, `dy`). The opposite corner stays put; the moving
 * edge stops at the image bound or at the minimum size, whichever comes first.
 */
export function resizeRegion(
  region: Region,
  handle: ResizeHandle,
  dx: number,
  dy: number,
  bounds: RegionBounds,
  min: number = MIN_REGION,
): Region {
  const base = clampRegion(region, bounds, min);
  const { width, height } = dimensions(bounds);
  const minWidth = Math.min(Math.max(0, round(min)), width);
  const minHeight = Math.min(Math.max(0, round(min)), height);
  const right = base.x + base.w;
  const bottom = base.y + base.h;
  const stepX = round(dx);
  const stepY = round(dy);

  switch (handle) {
    case "nw": {
      const x = clamp(base.x + stepX, 0, right - minWidth);
      const y = clamp(base.y + stepY, 0, bottom - minHeight);
      return { x, y, w: right - x, h: bottom - y };
    }
    case "ne": {
      const w = clamp(base.w + stepX, minWidth, width - base.x);
      const y = clamp(base.y + stepY, 0, bottom - minHeight);
      return { x: base.x, y, w, h: bottom - y };
    }
    case "sw": {
      const x = clamp(base.x + stepX, 0, right - minWidth);
      const h = clamp(base.h + stepY, minHeight, height - base.y);
      return { x, y: base.y, w: right - x, h };
    }
    case "se":
    default: {
      const w = clamp(base.w + stepX, minWidth, width - base.x);
      const h = clamp(base.h + stepY, minHeight, height - base.y);
      return { x: base.x, y: base.y, w, h };
    }
  }
}

/**
 * Fit `region` to `ratio` (width ÷ height). The longer edge of the current
 * rectangle is kept when it fits, then the rectangle is repositioned by
 * `anchor` (`center` keeps the middle, `nw` keeps the top-left corner).
 */
export function applyAspect(
  region: Region,
  ratio: number | null,
  bounds: RegionBounds,
  anchor: "center" | "nw" = "center",
  min: number = MIN_REGION,
): Region {
  const base = clampRegion(region, bounds, min);
  if (ratio === null || !Number.isFinite(ratio) || ratio <= 0) return base;

  const { width, height } = dimensions(bounds);
  const maxWidth = Math.min(width, height * ratio);
  const minWidth = Math.min(Math.max(0, round(min)), maxWidth);
  let w = clamp(base.w, minWidth, maxWidth);
  let h = w / ratio;

  const minimum = round(min);
  if (h < minimum) {
    h = Math.min(minimum, height);
    w = h * ratio;
  }
  if (w > maxWidth) {
    w = maxWidth;
    h = w / ratio;
  }

  const x = anchor === "nw" ? base.x : base.x + (base.w - w) / 2;
  const y = anchor === "nw" ? base.y : base.y + (base.h - h) / 2;
  return clampRegion({ x, y, w, h }, bounds, min);
}

/** Rectangle formed by a pointer drag, regardless of drag direction. */
export function regionFromDrag(
  start: { x: number; y: number },
  end: { x: number; y: number },
  bounds: RegionBounds,
  min: number = MIN_REGION,
): Region {
  return clampRegion(
    {
      x: Math.min(start.x, end.x),
      y: Math.min(start.y, end.y),
      w: Math.abs(end.x - start.x),
      h: Math.abs(end.y - start.y),
    },
    bounds,
    min,
  );
}

/**
 * Convert a region measured against the rendered preview into natural image
 * pixels. `displayed` is the preview's CSS box, which may be scaled.
 */
export function regionToPixels(
  region: Region,
  natural: RegionBounds,
  displayed: RegionBounds,
): Region {
  const scaleX = displayed.width > 0 ? natural.width / displayed.width : 0;
  const scaleY = displayed.height > 0 ? natural.height / displayed.height : 0;
  return clampRegion(
    {
      x: region.x * scaleX,
      y: region.y * scaleY,
      w: region.w * scaleX,
      h: region.h * scaleY,
    },
    natural,
    1,
  );
}
