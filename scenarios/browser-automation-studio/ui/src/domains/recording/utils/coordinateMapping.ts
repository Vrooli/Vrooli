/**
 * Coordinate Mapping Utilities for PlaywrightView
 *
 * Maps screen coordinates to Playwright viewport coordinates, accounting for:
 * - CSS object-fit: contain (image is centered with letterboxing)
 * - Device pixel ratio differences between screenshots and viewport
 *
 * ## Why This Matters
 *
 * Playwright captures screenshots at the device's pixel ratio (e.g., 2x on HiDPI).
 * A 900x700 viewport produces an 1800x1400 screenshot on a 2x display.
 *
 * When mapping user clicks to Playwright coordinates, we must use the LOGICAL
 * viewport dimensions (900x700), not the screenshot bitmap dimensions (1800x1400).
 *
 * Using bitmap dimensions would scale coordinates by the device pixel ratio,
 * causing clicks to land outside the visible viewport.
 *
 * @example
 * // Correct: Use viewport dimensions
 * const point = mapClientToViewport(clickX, clickY, containerRect, 900, 700);
 * // point might be { x: 450, y: 350 } - within viewport bounds
 *
 * // Wrong: Using bitmap dimensions (1800x1400)
 * // Would produce { x: 900, y: 700 } - outside viewport!
 */

export interface Rect {
  left: number;
  top: number;
  width: number;
  height: number;
}

export interface Point {
  x: number;
  y: number;
}

/**
 * Maps client (screen) coordinates to Playwright viewport coordinates.
 *
 * Assumes the image is displayed with CSS `object-fit: contain`, which:
 * - Scales the image to fit within the container while preserving aspect ratio
 * - Centers the image, potentially with letterboxing (empty space) on sides
 *
 * @param clientX - X coordinate relative to the document (e.g., from MouseEvent.clientX)
 * @param clientY - Y coordinate relative to the document (e.g., from MouseEvent.clientY)
 * @param containerRect - Bounding rect of the container element
 * @param viewportWidth - Logical viewport width (NOT screenshot bitmap width)
 * @param viewportHeight - Logical viewport height (NOT screenshot bitmap height)
 * @returns Point in viewport coordinates, clamped to viewport bounds
 */
export function mapClientToViewport(
  clientX: number,
  clientY: number,
  containerRect: Rect,
  viewportWidth: number,
  viewportHeight: number
): Point {
  // Calculate how the image is displayed (object-contain centers the image)
  const scale = Math.min(
    containerRect.width / viewportWidth,
    containerRect.height / viewportHeight
  );
  const displayWidth = viewportWidth * scale;
  const displayHeight = viewportHeight * scale;

  // Offset due to centering (letterboxing)
  const offsetX = (containerRect.width - displayWidth) / 2;
  const offsetY = (containerRect.height - displayHeight) / 2;

  // Get position relative to the displayed image area
  const relativeX = clientX - containerRect.left - offsetX;
  const relativeY = clientY - containerRect.top - offsetY;

  // Clamp to image bounds
  const clampedX = Math.max(0, Math.min(displayWidth, relativeX));
  const clampedY = Math.max(0, Math.min(displayHeight, relativeY));

  // Map from display coordinates to viewport coordinates
  return {
    x: (clampedX / displayWidth) * viewportWidth,
    y: (clampedY / displayHeight) * viewportHeight,
  };
}
