import type { Page } from 'playwright';
import type { Config } from '../config';
import { MAX_SCREENSHOT_SIZE_BYTES } from '../constants';
import { logger, metrics } from '../utils';

/** Track screenshot capture failures in metrics */
function trackScreenshotFailure(): void {
  metrics.telemetryFailures.inc({ type: 'screenshot' });
}

/**
 * Internal screenshot representation for the driver
 */
export interface ScreenshotCapture {
  base64: string;
  media_type: string;
  width: number;
  height: number;
}

/**
 * Capture screenshot from page
 *
 * Returns base64-encoded screenshot with metadata
 */
export async function captureScreenshot(
  page: Page,
  config: Config
): Promise<ScreenshotCapture | undefined> {
  if (!config.telemetry.screenshot.enabled) {
    return undefined;
  }

  try {
    const startTime = Date.now();

    const buffer = await page.screenshot({
      type: 'png',
      fullPage: config.telemetry.screenshot.fullPage,
      quality: undefined, // PNG doesn't support quality
    });

    // Check size limit
    if (buffer.length > config.telemetry.screenshot.maxSizeBytes) {
      logger.warn('Screenshot exceeds max size, truncating', {
        size: buffer.length,
        maxSize: config.telemetry.screenshot.maxSizeBytes,
      });

      // Try again with viewport-only screenshot if full page was requested
      if (config.telemetry.screenshot.fullPage) {
        const smallerBuffer = await page.screenshot({
          type: 'png',
          fullPage: false,
        });

        if (smallerBuffer.length <= config.telemetry.screenshot.maxSizeBytes) {
          const base64 = smallerBuffer.toString('base64');
          const viewport = page.viewportSize();

          metrics.screenshotSize.observe(smallerBuffer.length);

          return {
            base64,
            media_type: 'image/png',
            width: viewport?.width || 0,
            height: viewport?.height || 0,
          };
        }
      }

      // Still too large, return undefined
      logger.warn('Screenshot too large even with viewport-only capture', {
        size: buffer.length,
      });
      return undefined;
    }

    const base64 = buffer.toString('base64');
    const viewport = page.viewportSize();

    const duration = Date.now() - startTime;
    logger.debug('Screenshot captured', {
      size: buffer.length,
      duration,
      fullPage: config.telemetry.screenshot.fullPage,
    });

    metrics.screenshotSize.observe(buffer.length);

    return {
      base64,
      media_type: 'image/png',
      width: viewport?.width || 0,
      height: viewport?.height || 0,
    };
  } catch (error) {
    // Surface telemetry capture failures with context for debugging
    // This is important signal - telemetry failures can indicate page issues
    trackScreenshotFailure();
    logger.warn('telemetry: screenshot capture failed', {
      error: error instanceof Error ? error.message : String(error),
      hint: 'Page may have navigated, crashed, or become unresponsive',
      fullPage: config.telemetry.screenshot.fullPage,
    });
    return undefined;
  }
}

/**
 * Capture screenshot with JPEG compression for smaller size
 */
export async function captureCompressedScreenshot(
  page: Page,
  quality: number = 80,
  fullPage: boolean = false,
  maxSizeBytes: number = MAX_SCREENSHOT_SIZE_BYTES
): Promise<ScreenshotCapture | undefined> {
  try {
    const buffer = await page.screenshot({
      type: 'jpeg',
      quality,
      fullPage,
    });

    if (buffer.length > maxSizeBytes) {
      logger.warn('Compressed screenshot exceeds max size', {
        size: buffer.length,
        quality,
        maxSize: maxSizeBytes,
      });

      // Try with lower quality if still too large
      if (quality > 50) {
        return captureCompressedScreenshot(page, quality - 20, fullPage, maxSizeBytes);
      }

      return undefined;
    }

    const base64 = buffer.toString('base64');
    const viewport = page.viewportSize();

    metrics.screenshotSize.observe(buffer.length);

    return {
      base64,
      media_type: 'image/jpeg',
      width: viewport?.width || 0,
      height: viewport?.height || 0,
    };
  } catch (error) {
    trackScreenshotFailure();
    logger.warn('telemetry: compressed screenshot capture failed', {
      error: error instanceof Error ? error.message : String(error),
      quality,
      fullPage,
    });
    return undefined;
  }
}
