import { describe, it, expect } from 'vitest';
import { mapClientToViewport, type Rect } from './coordinateMapping';

describe('mapClientToViewport', () => {
  /**
   * Standard container where the image fills the container exactly.
   * No letterboxing, 1:1 scale.
   */
  const exactFitContainer: Rect = {
    left: 0,
    top: 0,
    width: 900,
    height: 700,
  };

  describe('basic coordinate mapping', () => {
    it('maps center click to center of viewport', () => {
      const point = mapClientToViewport(450, 350, exactFitContainer, 900, 700);
      expect(point.x).toBeCloseTo(450);
      expect(point.y).toBeCloseTo(350);
    });

    it('maps top-left corner correctly', () => {
      const point = mapClientToViewport(0, 0, exactFitContainer, 900, 700);
      expect(point.x).toBeCloseTo(0);
      expect(point.y).toBeCloseTo(0);
    });

    it('maps bottom-right corner correctly', () => {
      const point = mapClientToViewport(900, 700, exactFitContainer, 900, 700);
      expect(point.x).toBeCloseTo(900);
      expect(point.y).toBeCloseTo(700);
    });
  });

  describe('coordinate clamping', () => {
    it('clamps negative coordinates to 0', () => {
      const point = mapClientToViewport(-50, -30, exactFitContainer, 900, 700);
      expect(point.x).toBe(0);
      expect(point.y).toBe(0);
    });

    it('clamps coordinates beyond viewport to max bounds', () => {
      const point = mapClientToViewport(1000, 800, exactFitContainer, 900, 700);
      expect(point.x).toBeCloseTo(900);
      expect(point.y).toBeCloseTo(700);
    });
  });

  describe('container offset handling', () => {
    it('accounts for container offset from document origin', () => {
      const offsetContainer: Rect = {
        left: 100,
        top: 50,
        width: 900,
        height: 700,
      };

      // Click at (550, 400) in document space = (450, 350) relative to container
      const point = mapClientToViewport(550, 400, offsetContainer, 900, 700);
      expect(point.x).toBeCloseTo(450);
      expect(point.y).toBeCloseTo(350);
    });
  });

  describe('letterboxing (object-fit: contain)', () => {
    it('handles horizontal letterboxing (container wider than viewport aspect ratio)', () => {
      // Container is 1200x700, viewport is 900x700
      // Image displays at 900x700 (no scaling needed for height)
      // Horizontal offset = (1200 - 900) / 2 = 150px on each side
      const wideContainer: Rect = {
        left: 0,
        top: 0,
        width: 1200,
        height: 700,
      };

      // Click in the center of the container
      const point = mapClientToViewport(600, 350, wideContainer, 900, 700);
      expect(point.x).toBeCloseTo(450); // Should map to center of viewport
      expect(point.y).toBeCloseTo(350);
    });

    it('handles vertical letterboxing (container taller than viewport aspect ratio)', () => {
      // Container is 900x900, viewport is 900x700
      // Image displays at 900x700 (fills width exactly)
      // Vertical offset = (900 - 700) / 2 = 100px on top and bottom
      const tallContainer: Rect = {
        left: 0,
        top: 0,
        width: 900,
        height: 900,
      };

      // Click in the center of the container
      const point = mapClientToViewport(450, 450, tallContainer, 900, 700);
      expect(point.x).toBeCloseTo(450);
      expect(point.y).toBeCloseTo(350); // Should map to center of viewport
    });

    it('clamps clicks in letterbox area to viewport edge', () => {
      // Wide container with 150px letterbox on each side
      const wideContainer: Rect = {
        left: 0,
        top: 0,
        width: 1200,
        height: 700,
      };

      // Click in the left letterbox area (x=50, which is in the empty space)
      const point = mapClientToViewport(50, 350, wideContainer, 900, 700);
      expect(point.x).toBe(0); // Clamped to left edge
      expect(point.y).toBeCloseTo(350);
    });
  });

  describe('scaling (container smaller/larger than viewport)', () => {
    it('handles scaled-down display (container smaller than viewport)', () => {
      // Container is 450x350, viewport is 900x700
      // Display scale is 0.5
      const smallContainer: Rect = {
        left: 0,
        top: 0,
        width: 450,
        height: 350,
      };

      // Click at center of small container (225, 175)
      // Should map to center of viewport (450, 350)
      const point = mapClientToViewport(225, 175, smallContainer, 900, 700);
      expect(point.x).toBeCloseTo(450);
      expect(point.y).toBeCloseTo(350);
    });

    it('handles scaled-up display (container larger than viewport)', () => {
      // Container is 1800x1400, viewport is 900x700
      // Display scale is 2.0
      const largeContainer: Rect = {
        left: 0,
        top: 0,
        width: 1800,
        height: 1400,
      };

      // Click at center of large container (900, 700)
      // Should map to center of viewport (450, 350)
      const point = mapClientToViewport(900, 700, largeContainer, 900, 700);
      expect(point.x).toBeCloseTo(450);
      expect(point.y).toBeCloseTo(350);
    });
  });

  /**
   * CRITICAL TEST: Device Pixel Ratio Bug Prevention
   *
   * This test documents the bug that was fixed and ensures it doesn't regress.
   *
   * The bug: On HiDPI displays (e.g., 2x DPR), Playwright captures screenshots
   * at physical pixel dimensions (1800x1400), but the viewport is logical pixels
   * (900x700). If we mistakenly used bitmap dimensions for coordinate mapping,
   * clicks would be scaled 2x and land outside the viewport.
   *
   * The fix: Always use LOGICAL viewport dimensions for coordinate mapping,
   * never the screenshot bitmap dimensions.
   */
  describe('device pixel ratio bug prevention', () => {
    it('coordinates stay within viewport bounds regardless of bitmap dimensions', () => {
      // Simulates the real scenario:
      // - Container displays the preview at 900x700
      // - Viewport is 900x700 (logical pixels)
      // - Screenshot bitmap is 1800x1400 (2x DPR - but we should NOT use this!)
      const container: Rect = {
        left: 0,
        top: 0,
        width: 900,
        height: 700,
      };

      // User clicks near bottom-right of the visible preview
      const clickX = 800;
      const clickY = 600;

      // CORRECT: Using viewport dimensions (900x700)
      const correctPoint = mapClientToViewport(clickX, clickY, container, 900, 700);
      expect(correctPoint.x).toBeCloseTo(800);
      expect(correctPoint.y).toBeCloseTo(600);
      expect(correctPoint.x).toBeLessThanOrEqual(900); // Within viewport
      expect(correctPoint.y).toBeLessThanOrEqual(700); // Within viewport

      // WRONG: If we had used bitmap dimensions (1800x1400), we'd get:
      // This simulates the bug - DO NOT do this in production!
      const wrongPoint = mapClientToViewport(clickX, clickY, container, 1800, 1400);
      // The wrong calculation would produce coordinates ~2x larger
      expect(wrongPoint.x).toBeCloseTo(1600); // 2x the correct value!
      expect(wrongPoint.y).toBeCloseTo(1200); // 2x the correct value!
      // These coordinates are OUTSIDE the 900x700 viewport!
      expect(wrongPoint.x).toBeGreaterThan(900);
      expect(wrongPoint.y).toBeGreaterThan(700);
    });

    it('documents that viewport prop must be logical pixels, not bitmap pixels', () => {
      // This test serves as documentation
      // The viewport dimensions passed to mapClientToViewport MUST be:
      // - The logical viewport size set in Playwright (e.g., 900x700)
      // NOT:
      // - The screenshot bitmap dimensions (which include device pixel ratio)
      // - The createImageBitmap dimensions (same as screenshot)

      // Example values that would be WRONG to use:
      const bitmapWidth = 1800; // 2x DPR
      const bitmapHeight = 1400;

      // Example values that are CORRECT to use:
      const viewportWidth = 900; // Logical pixels
      const viewportHeight = 700;

      // The ratio should always be 1:1 between container and viewport for accurate mapping
      // (unless intentionally scaling the display)
      expect(viewportWidth).not.toBe(bitmapWidth);
      expect(viewportHeight).not.toBe(bitmapHeight);
    });
  });
});
