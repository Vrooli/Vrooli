/**
 * Coordinate Mapping Utilities for PlaywrightView
 *
 * Maps screen coordinates to Playwright viewport coordinates, accounting for:
 * - CSS object-fit: contain (image is centered with letterboxing)
 * - Device pixel ratio differences between screenshots and viewport
 * - Frame dimensions vs viewport dimensions mismatch
 *
 * ## Why This Matters
 *
 * Playwright captures screenshots at the device's pixel ratio (e.g., 2x on HiDPI).
 * A 900x700 viewport produces an 1800x1400 screenshot on a 2x display.
 *
 * When mapping user clicks to Playwright coordinates, we need TWO dimensions:
 * 1. FRAME dimensions (1800x1400) - for calculating how the image is displayed
 * 2. VIEWPORT dimensions (900x700) - for the final coordinates to send to Playwright
 *
 * The CSS object-contain scales the FRAME to fit the container, but Playwright
 * operates in VIEWPORT coordinate space.
 *
 * @example
 * // Frame is 1800x1400, viewport is 900x700, container is 800x600
 * const point = mapClientToViewportWithFrame(
 *   clickX, clickY, containerRect,
 *   1800, 1400,  // frame dimensions (for display calculation)
 *   900, 700     // viewport dimensions (for output coordinates)
 * );
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
 * IMPORTANT: This function now accepts both frame AND viewport dimensions.
 * Use `mapClientToViewportWithFrame` for accurate mapping when frame dimensions
 * differ from viewport (e.g., HiDPI displays).
 *
 * @deprecated Use `mapClientToViewportWithFrame` for accurate HiDPI support.
 * This function assumes frame dimensions equal viewport dimensions.
 *
 * @param clientX - X coordinate relative to the document
 * @param clientY - Y coordinate relative to the document
 * @param containerRect - Bounding rect of the container element
 * @param viewportWidth - Logical viewport width
 * @param viewportHeight - Logical viewport height
 * @returns Point in viewport coordinates, clamped to viewport bounds
 */
export function mapClientToViewport(
  clientX: number,
  clientY: number,
  containerRect: Rect,
  viewportWidth: number,
  viewportHeight: number
): Point {
  // Fallback: assume frame dimensions equal viewport dimensions
  return mapClientToViewportWithFrame(
    clientX,
    clientY,
    containerRect,
    viewportWidth,
    viewportHeight,
    viewportWidth,
    viewportHeight
  );
}

/**
 * Maps client (screen) coordinates to Playwright viewport coordinates,
 * correctly handling HiDPI displays where frame dimensions differ from viewport.
 *
 * Assumes the image is displayed with CSS `object-fit: contain`, which:
 * - Scales the image to fit within the container while preserving aspect ratio
 * - Centers the image, potentially with letterboxing (empty space) on sides
 *
 * @param clientX - X coordinate relative to the document (e.g., from MouseEvent.clientX)
 * @param clientY - Y coordinate relative to the document (e.g., from MouseEvent.clientY)
 * @param containerRect - Bounding rect of the canvas element (CSS box)
 * @param frameWidth - Width of the frame bitmap (what's actually displayed)
 * @param frameHeight - Height of the frame bitmap (what's actually displayed)
 * @param viewportWidth - Logical viewport width (what Playwright uses)
 * @param viewportHeight - Logical viewport height (what Playwright uses)
 * @returns Point in viewport coordinates, clamped to viewport bounds
 */
export function mapClientToViewportWithFrame(
  clientX: number,
  clientY: number,
  containerRect: Rect,
  frameWidth: number,
  frameHeight: number,
  viewportWidth: number,
  viewportHeight: number
): Point {
  // Calculate how the FRAME is displayed (object-contain centers the image)
  // This uses FRAME dimensions, not viewport dimensions
  const scale = Math.min(
    containerRect.width / frameWidth,
    containerRect.height / frameHeight
  );
  const displayWidth = frameWidth * scale;
  const displayHeight = frameHeight * scale;

  // Offset due to centering (letterboxing)
  const offsetX = (containerRect.width - displayWidth) / 2;
  const offsetY = (containerRect.height - displayHeight) / 2;

  // Get position relative to the displayed image area
  const relativeX = clientX - containerRect.left - offsetX;
  const relativeY = clientY - containerRect.top - offsetY;

  // Clamp to image bounds (in display coordinates)
  const clampedX = Math.max(0, Math.min(displayWidth, relativeX));
  const clampedY = Math.max(0, Math.min(displayHeight, relativeY));

  // Map from display coordinates to VIEWPORT coordinates
  // The ratio within the frame maps to the same ratio in the viewport
  // This correctly handles HiDPI: click at 50% of frame = 50% of viewport
  return {
    x: (clampedX / displayWidth) * viewportWidth,
    y: (clampedY / displayHeight) * viewportHeight,
  };
}
