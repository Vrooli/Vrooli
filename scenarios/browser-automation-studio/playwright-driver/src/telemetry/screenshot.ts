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
 * Returns base64-encoded screenshot with metadata.
 *
 * IMPORTANT: When fullPage is false, we use explicit clipping to the viewport
 * dimensions. This prevents Playwright from modifying the viewport during
 * capture, which would cause screen size oscillation during execution mode.
 * See: https://github.com/anthropics/vrooli/issues/XXX
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
    const viewport = page.viewportSize();

    // Build screenshot options
    // When fullPage is false, use explicit clip to prevent viewport modification
    // This is critical for execution mode where frame streaming must show consistent size
    const screenshotOptions: Parameters<typeof page.screenshot>[0] = {
      type: 'png',
      quality: undefined, // PNG doesn't support quality
    };

    if (config.telemetry.screenshot.fullPage) {
      screenshotOptions.fullPage = true;
    } else if (viewport) {
      // Use explicit clipping to viewport - this does NOT modify the viewport
      // during capture, unlike fullPage which can cause temporary viewport changes
      screenshotOptions.clip = {
        x: 0,
        y: 0,
        width: viewport.width,
        height: viewport.height,
      };
    }
    // If no viewport available and not fullPage, Playwright will use current viewport

    const buffer = await page.screenshot(screenshotOptions);

    // Check size limit
    if (buffer.length > config.telemetry.screenshot.maxSizeBytes) {
      logger.warn('Screenshot exceeds max size, truncating', {
        size: buffer.length,
        maxSize: config.telemetry.screenshot.maxSizeBytes,
      });

      // Try again with viewport-only screenshot if full page was requested
      if (config.telemetry.screenshot.fullPage && viewport) {
        // Use explicit clip to prevent any viewport modification
        const smallerBuffer = await page.screenshot({
          type: 'png',
          clip: {
            x: 0,
            y: 0,
            width: viewport.width,
            height: viewport.height,
          },
        });

        if (smallerBuffer.length <= config.telemetry.screenshot.maxSizeBytes) {
          const base64 = smallerBuffer.toString('base64');

          metrics.screenshotSize.observe(smallerBuffer.length);

          return {
            base64,
            media_type: 'image/png',
            width: viewport.width,
            height: viewport.height,
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

    const duration = Date.now() - startTime;
    logger.debug('Screenshot captured', {
      size: buffer.length,
      duration,
      fullPage: config.telemetry.screenshot.fullPage,
    });

    metrics.screenshotSize.observe(buffer.length);

    // Use viewport captured at start, or re-fetch if needed
    const finalViewport = viewport ?? page.viewportSize();
    return {
      base64,
      media_type: 'image/png',
      width: finalViewport?.width || 0,
      height: finalViewport?.height || 0,
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
 * Capture screenshot with JPEG compression for smaller size.
 *
 * When fullPage is false, uses explicit clipping to prevent viewport modification.
 */
export async function captureCompressedScreenshot(
  page: Page,
  quality: number = 80,
  fullPage: boolean = false,
  maxSizeBytes: number = MAX_SCREENSHOT_SIZE_BYTES
): Promise<ScreenshotCapture | undefined> {
  try {
    const viewport = page.viewportSize();

    // Build screenshot options with explicit clipping when not fullPage
    const screenshotOptions: Parameters<typeof page.screenshot>[0] = {
      type: 'jpeg',
      quality,
    };

    if (fullPage) {
      screenshotOptions.fullPage = true;
    } else if (viewport) {
      // Use explicit clip to prevent viewport modification during capture
      screenshotOptions.clip = {
        x: 0,
        y: 0,
        width: viewport.width,
        height: viewport.height,
      };
    }

    const buffer = await page.screenshot(screenshotOptions);

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

    metrics.screenshotSize.observe(buffer.length);

    // Use viewport captured at start, or re-fetch if needed
    const finalViewport = viewport ?? page.viewportSize();
    return {
      base64,
      media_type: 'image/jpeg',
      width: finalViewport?.width || 0,
      height: finalViewport?.height || 0,
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
