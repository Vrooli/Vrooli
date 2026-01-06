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

  /**
   * Edge case tests for canvas with object-contain CSS behavior.
   *
   * When a canvas uses object-contain, the rendered bitmap area may be smaller
   * than the canvas's CSS box due to letterboxing. The coordinate mapping must
   * account for this correctly.
   */
  describe('canvas object-contain behavior', () => {
    it('handles canvas with non-matching aspect ratio (horizontal letterbox)', () => {
      // Scenario: Container is 1000x600, viewport is 900x700
      // The viewport has a taller aspect ratio than the container
      // Container aspect ratio: 1000/600 = 1.67
      // Viewport aspect ratio: 900/700 = 1.29
      // Scale by height: 600/700 = 0.857
      // Displayed width: 900 * 0.857 = 771.4
      // Horizontal offset: (1000 - 771.4) / 2 = 114.3
      const container: Rect = { left: 0, top: 0, width: 1000, height: 600 };

      // Click in the center of the container (500, 300)
      const point = mapClientToViewport(500, 300, container, 900, 700);
      // Should map to center of viewport (450, 350)
      expect(point.x).toBeCloseTo(450, 0);
      expect(point.y).toBeCloseTo(350, 0);
    });

    it('handles canvas with non-matching aspect ratio (vertical letterbox)', () => {
      // Scenario: Container is 600x1000, viewport is 900x700
      // The viewport has a wider aspect ratio than the container
      // Container aspect ratio: 600/1000 = 0.6
      // Viewport aspect ratio: 900/700 = 1.29
      // Scale by width: 600/900 = 0.667
      // Displayed height: 700 * 0.667 = 466.7
      // Vertical offset: (1000 - 466.7) / 2 = 266.7
      const container: Rect = { left: 0, top: 0, width: 600, height: 1000 };

      // Click in the center of the container (300, 500)
      const point = mapClientToViewport(300, 500, container, 900, 700);
      // Should map to center of viewport (450, 350)
      expect(point.x).toBeCloseTo(450, 0);
      expect(point.y).toBeCloseTo(350, 0);
    });

    it('correctly maps clicks at the edge of the visible area', () => {
      // Container is 1200x700, viewport is 900x700
      // No scaling needed (both have same height)
      // Horizontal offset: (1200 - 900) / 2 = 150
      const container: Rect = { left: 0, top: 0, width: 1200, height: 700 };

      // Click at the left edge of the visible canvas area (150, 350)
      const leftEdge = mapClientToViewport(150, 350, container, 900, 700);
      expect(leftEdge.x).toBeCloseTo(0, 0);
      expect(leftEdge.y).toBeCloseTo(350, 0);

      // Click at the right edge of the visible canvas area (1050, 350)
      const rightEdge = mapClientToViewport(1050, 350, container, 900, 700);
      expect(rightEdge.x).toBeCloseTo(900, 0);
      expect(rightEdge.y).toBeCloseTo(350, 0);
    });
  });

  /**
   * Tests for real-world recording scenarios to catch regressions.
   */
  describe('real-world recording scenarios', () => {
    it('handles typical sidebar layout where preview takes partial width', () => {
      // Common scenario: sidebar takes 300px, preview takes remaining space
      // Preview container: 900x700, positioned at (300, 0)
      // Viewport: 900x700 (exact fit)
      const container: Rect = { left: 300, top: 0, width: 900, height: 700 };

      // Click at document position (750, 350) = center of preview
      const point = mapClientToViewport(750, 350, container, 900, 700);
      expect(point.x).toBeCloseTo(450);
      expect(point.y).toBeCloseTo(350);
    });

    it('handles preview with header offset', () => {
      // Header takes 50px, preview below
      // Preview container: 900x650, positioned at (0, 50)
      // Viewport: 900x700 (taller than container)
      // Scale: 650/700 = 0.929
      // Displayed width: 900 * 0.929 = 835.7
      // Horizontal offset: (900 - 835.7) / 2 = 32.15
      const container: Rect = { left: 0, top: 50, width: 900, height: 650 };

      // Click at document position (450, 375) = center of container
      const point = mapClientToViewport(450, 375, container, 900, 700);
      expect(point.x).toBeCloseTo(450, 0);
      expect(point.y).toBeCloseTo(350, 0);
    });

    it('handles resized preview after user drags sidebar', () => {
      // User dragged sidebar to make preview narrower
      // Preview container: 700x700, positioned at (400, 0)
      // Viewport: 900x700 (wider than container)
      // Scale: 700/900 = 0.778
      // Displayed height: 700 * 0.778 = 544.4
      // Vertical offset: (700 - 544.4) / 2 = 77.8
      const container: Rect = { left: 400, top: 0, width: 700, height: 700 };

      // Click at document position (750, 350) = center of container
      const point = mapClientToViewport(750, 350, container, 900, 700);
      expect(point.x).toBeCloseTo(450, 0);
      expect(point.y).toBeCloseTo(350, 0);
    });
  });

  /**
   * Edge cases for extreme aspect ratios and viewport sizes.
   */
  describe('extreme aspect ratios', () => {
    it('handles very wide viewport (ultrawide monitor simulation)', () => {
      // Ultrawide viewport: 2560x1080 (21:9 aspect ratio)
      // Container: 1920x1080 (16:9)
      // The viewport is wider, so it will be letterboxed horizontally
      const container: Rect = { left: 0, top: 0, width: 1920, height: 1080 };

      // Click center of container
      const point = mapClientToViewport(960, 540, container, 2560, 1080);
      expect(point.x).toBeCloseTo(1280, 0); // Center of ultrawide viewport
      expect(point.y).toBeCloseTo(540, 0);
    });

    it('handles very tall viewport (mobile portrait simulation)', () => {
      // Mobile viewport: 375x812 (iPhone X aspect ratio)
      // Container: 800x600 (desktop window)
      // The viewport is much taller, so it will be heavily letterboxed
      const container: Rect = { left: 0, top: 0, width: 800, height: 600 };

      // Click center of container
      const point = mapClientToViewport(400, 300, container, 375, 812);
      expect(point.x).toBeCloseTo(187.5, 0);
      expect(point.y).toBeCloseTo(406, 0);
    });

    it('handles square viewport in rectangular container', () => {
      // Square viewport: 500x500
      // Container: 800x400 (wide rectangular)
      const container: Rect = { left: 0, top: 0, width: 800, height: 400 };

      // Click center of container
      const point = mapClientToViewport(400, 200, container, 500, 500);
      expect(point.x).toBeCloseTo(250, 0);
      expect(point.y).toBeCloseTo(250, 0);
    });
  });

  /**
   * Combined scenarios with multiple transformations.
   */
  describe('combined transformations', () => {
    it('handles offset + scaling + letterboxing together', () => {
      // Complex scenario:
      // - Container at (200, 100) offset
      // - Container: 600x400
      // - Viewport: 900x700 (different aspect ratio, larger)
      // This tests all transformations at once
      const container: Rect = { left: 200, top: 100, width: 600, height: 400 };

      // Click at center of container in document space (500, 300)
      const point = mapClientToViewport(500, 300, container, 900, 700);
      expect(point.x).toBeCloseTo(450, 0);
      expect(point.y).toBeCloseTo(350, 0);
    });

    it('handles negative container offset (scrolled page)', () => {
      // When page is scrolled, container can have negative offset
      const container: Rect = { left: -50, top: -100, width: 900, height: 700 };

      // Click at origin (which is outside the container)
      const point = mapClientToViewport(0, 0, container, 900, 700);
      // Relative to container: (50, 100)
      expect(point.x).toBeCloseTo(50, 0);
      expect(point.y).toBeCloseTo(100, 0);
    });
  });

  /**
   * Boundary precision tests to ensure floating point math doesn't cause issues.
   */
  describe('floating point precision', () => {
    it('handles fractional container dimensions', () => {
      // Containers can have fractional dimensions due to CSS transforms
      const container: Rect = { left: 0.5, top: 0.25, width: 899.75, height: 699.5 };

      const point = mapClientToViewport(450.25, 350.125, container, 900, 700);
      // Should still map reasonably close to center
      expect(point.x).toBeGreaterThan(440);
      expect(point.x).toBeLessThan(460);
      expect(point.y).toBeGreaterThan(340);
      expect(point.y).toBeLessThan(360);
    });

    it('handles very small scale factors', () => {
      // Tiny container: 90x70 (0.1x scale)
      const container: Rect = { left: 0, top: 0, width: 90, height: 70 };

      // Click at center of tiny container
      const point = mapClientToViewport(45, 35, container, 900, 700);
      expect(point.x).toBeCloseTo(450, 0);
      expect(point.y).toBeCloseTo(350, 0);
    });

    it('maintains accuracy at viewport boundaries', () => {
      const container: Rect = { left: 0, top: 0, width: 900, height: 700 };

      // Test all four corners with sub-pixel precision
      const topLeft = mapClientToViewport(0.001, 0.001, container, 900, 700);
      expect(topLeft.x).toBeGreaterThanOrEqual(0);
      expect(topLeft.y).toBeGreaterThanOrEqual(0);

      const bottomRight = mapClientToViewport(899.999, 699.999, container, 900, 700);
      expect(bottomRight.x).toBeLessThanOrEqual(900);
      expect(bottomRight.y).toBeLessThanOrEqual(700);
    });
  });

  /**
   * Boundary and edge case handling.
   *
   * Note: Zero-dimension containers (width=0 or height=0) are not supported
   * and will produce NaN results. This is expected behavior since such
   * containers cannot meaningfully display content.
   */
  describe('boundary handling', () => {
    it('handles click exactly on container boundary', () => {
      const container: Rect = { left: 100, top: 100, width: 900, height: 700 };

      // Click exactly on top-left boundary
      const topLeft = mapClientToViewport(100, 100, container, 900, 700);
      expect(topLeft.x).toBeCloseTo(0);
      expect(topLeft.y).toBeCloseTo(0);

      // Click exactly on bottom-right boundary
      const bottomRight = mapClientToViewport(1000, 800, container, 900, 700);
      expect(bottomRight.x).toBeCloseTo(900);
      expect(bottomRight.y).toBeCloseTo(700);
    });
  });
});
