/**
 * Screenshot Capture
 *
 * STABILITY: STABLE CORE
 *
 * This module provides screenshot capture functionality for the vision agent.
 * It wraps Playwright's screenshot capabilities with size management.
 */

import type { Page } from 'playwright';
import type { ScreenshotCaptureInterface, ScreenshotOptions } from '../vision-agent/types';
import type { ScreenshotResult, CaptureOptions } from './types';

/**
 * Default maximum screenshot size (5MB).
 */
const DEFAULT_MAX_SIZE_BYTES = 5 * 1024 * 1024;

/**
 * Default JPEG quality.
 */
const DEFAULT_QUALITY = 80;

/**
 * Create a screenshot capture service.
 *
 * TESTING SEAM: This returns an interface that can be mocked.
 */
export function createScreenshotCapture(): ScreenshotCaptureInterface {
  return {
    async capture(page: Page, options?: ScreenshotOptions): Promise<Buffer> {
      const result = await captureScreenshot(page, {
        format: options?.format,
        quality: options?.quality,
        fullPage: options?.fullPage,
      });
      return result.buffer;
    },
  };
}

/**
 * Capture a screenshot from the page.
 *
 * @param page - Playwright page
 * @param options - Capture options
 * @returns Screenshot result with buffer and metadata
 */
export async function captureScreenshot(
  page: Page,
  options: CaptureOptions = {}
): Promise<ScreenshotResult> {
  const format = options.format ?? 'jpeg';
  const quality = format === 'jpeg' ? (options.quality ?? DEFAULT_QUALITY) : undefined;
  const fullPage = options.fullPage ?? false;
  const maxSizeBytes = options.maxSizeBytes ?? DEFAULT_MAX_SIZE_BYTES;

  let buffer = await page.screenshot({
    type: format,
    quality,
    fullPage,
  });

  // If too large, try reducing quality (for JPEG) or switching to JPEG
  if (buffer.length > maxSizeBytes) {
    if (format === 'jpeg' && quality && quality > 50) {
      // Try lower quality
      buffer = await page.screenshot({
        type: 'jpeg',
        quality: quality - 20,
        fullPage,
      });
    } else if (format === 'png') {
      // Try JPEG instead
      buffer = await page.screenshot({
        type: 'jpeg',
        quality: 60,
        fullPage,
      });
    }

    // If still too large and fullPage, try viewport only
    if (buffer.length > maxSizeBytes && fullPage) {
      buffer = await page.screenshot({
        type: 'jpeg',
        quality: 60,
        fullPage: false,
      });
    }
  }

  const viewport = page.viewportSize();

  return {
    buffer,
    mediaType: format === 'png' ? 'image/png' : 'image/jpeg',
    width: viewport?.width ?? 0,
    height: viewport?.height ?? 0,
  };
}

/**
 * Create a mock screenshot capture for testing.
 */
export function createMockScreenshotCapture(
  mockBuffer?: Buffer
): ScreenshotCaptureInterface {
  const defaultBuffer = mockBuffer ?? Buffer.from('mock-screenshot-data');

  return {
    async capture(_page: Page, _options?: ScreenshotOptions): Promise<Buffer> {
      return defaultBuffer;
    },
  };
}
