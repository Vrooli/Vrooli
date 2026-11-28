/**
 * Pure geometry for the crop drag-box. The canvas shows the image with
 * `object-contain`, so the rendered content occupies a letterboxed rectangle
 * inside the element. These helpers map between *image pixels* (what the op
 * receives) and *display pixels* (what the pointer touches), clamp the crop to
 * the image, and snap to an aspect ratio — all without touching the DOM, so
 * the logic is unit-testable in isolation.
 */

export interface Size {
  width: number;
  height: number;
}

export interface Point {
  x: number;
  y: number;
}

export interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

/** The letterboxed content rectangle of an object-contain image. */
export interface ContentRect extends Rect {
  /** Image-pixels per display-pixel (uniform for object-contain). */
  scale: number;
}

/**
 * Where the image content actually sits inside an object-contain element, and
 * the scale factor from image px → display px. Returns a zero-scale rect if any
 * dimension is non-positive (caller should treat that as "not ready").
 */
export function contentRect(natural: Size, client: Size): ContentRect {
  if (
    natural.width <= 0 ||
    natural.height <= 0 ||
    client.width <= 0 ||
    client.height <= 0
  ) {
    return { x: 0, y: 0, width: 0, height: 0, scale: 0 };
  }
  const displayScale = Math.min(
    client.width / natural.width,
    client.height / natural.height,
  );
  const width = natural.width * displayScale;
  const height = natural.height * displayScale;
  return {
    x: (client.width - width) / 2,
    y: (client.height - height) / 2,
    width,
    height,
    scale: displayScale,
  };
}

/** Convert an image-pixel rect to a display-pixel rect within the element. */
export function imageRectToDisplay(rect: Rect, content: ContentRect): Rect {
  return {
    x: content.x + rect.x * content.scale,
    y: content.y + rect.y * content.scale,
    width: rect.width * content.scale,
    height: rect.height * content.scale,
  };
}

/** Convert a display-pixel point (element-relative) to an image-pixel point. */
export function displayPointToImage(point: Point, content: ContentRect): Point {
  if (content.scale <= 0) {
    return { x: 0, y: 0 };
  }
  return {
    x: (point.x - content.x) / content.scale,
    y: (point.y - content.y) / content.scale,
  };
}

/** Round every field of a rect to integer image pixels. */
export function roundRect(rect: Rect): Rect {
  return {
    x: Math.round(rect.x),
    y: Math.round(rect.y),
    width: Math.round(rect.width),
    height: Math.round(rect.height),
  };
}

/**
 * Clamp a crop rect to the image bounds. Keeps width/height ≥ 1 and shifts the
 * origin so the box stays fully inside `[0, natural]`.
 */
export function clampRect(rect: Rect, natural: Size): Rect {
  const width = Math.max(1, Math.min(rect.width, natural.width));
  const height = Math.max(1, Math.min(rect.height, natural.height));
  const x = Math.max(0, Math.min(rect.x, natural.width - width));
  const y = Math.max(0, Math.min(rect.y, natural.height - height));
  return { x, y, width, height };
}

/**
 * Snap a rect to a width:height aspect ratio, anchored at its top-left, fitting
 * inside the image. A non-positive ratio leaves the rect untouched (Free).
 */
export function applyAspect(rect: Rect, ratio: number, natural: Size): Rect {
  if (ratio <= 0) {
    return clampRect(rect, natural);
  }
  // Derive a height from the current width, then cap both to the image.
  let width = rect.width;
  let height = width / ratio;
  if (rect.x + width > natural.width) {
    width = natural.width - rect.x;
    height = width / ratio;
  }
  if (rect.y + height > natural.height) {
    height = natural.height - rect.y;
    width = height * ratio;
  }
  return clampRect({ x: rect.x, y: rect.y, width, height }, natural);
}

/** The full-image crop rect for a freshly loaded image. */
export function fullImageRect(natural: Size): Rect {
  return { x: 0, y: 0, width: natural.width, height: natural.height };
}
