import type { Page } from 'playwright';
import type { Screenshot } from '../types';
import type { Config } from '../config';
import { MAX_SCREENSHOT_SIZE_BYTES } from '../constants';
import { logger, metrics } from '../utils';

/**
 * Capture screenshot from page
 *
 * Returns base64-encoded screenshot with metadata
 */
export async function captureScreenshot(
  page: Page,
  config: Config
): Promise<Screenshot | undefined> {
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
          const data = smallerBuffer.toString('base64');
          const viewport = page.viewportSize();

          metrics.screenshotSize.observe(smallerBuffer.length);

          return {
            data,
            format: 'png',
            viewport_width: viewport?.width || 0,
            viewport_height: viewport?.height || 0,
          };
        }
      }

      // Still too large, return undefined
      logger.warn('Screenshot too large even with viewport-only capture', {
        size: buffer.length,
      });
      return undefined;
    }

    const data = buffer.toString('base64');
    const viewport = page.viewportSize();

    const duration = Date.now() - startTime;
    logger.debug('Screenshot captured', {
      size: buffer.length,
      duration,
      fullPage: config.telemetry.screenshot.fullPage,
    });

    metrics.screenshotSize.observe(buffer.length);

    return {
      data,
      format: 'png',
      viewport_width: viewport?.width || 0,
      viewport_height: viewport?.height || 0,
    };
  } catch (error) {
    logger.error('Failed to capture screenshot', {
      error: error instanceof Error ? error.message : String(error),
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
): Promise<Screenshot | undefined> {
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

    const data = buffer.toString('base64');
    const viewport = page.viewportSize();

    metrics.screenshotSize.observe(buffer.length);

    return {
      data,
      format: 'jpeg',
      viewport_width: viewport?.width || 0,
      viewport_height: viewport?.height || 0,
    };
  } catch (error) {
    logger.error('Failed to capture compressed screenshot', {
      error: error instanceof Error ? error.message : String(error),
    });
    return undefined;
  }
}
