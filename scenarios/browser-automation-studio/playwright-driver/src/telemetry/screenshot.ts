import type { Page } from 'rebrowser-playwright';
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
  requested_extent: 'viewport' | 'full_page' | 'element';
  actual_extent: 'viewport' | 'full_page' | 'element';
  degraded: boolean;
}

export class UnsupportedFullPageScreenshotError extends Error {
  constructor() {
    super('Screenshot request rejected: requested_extent=full_page, actual_extent=not_captured during native video recording; use a viewport screenshot or capture the full page in a separate session.');
    this.name = 'UnsupportedFullPageScreenshotError';
  }
}

export class ScreenshotExtentRejectedError extends Error {
  constructor(
    readonly requestedExtent: 'full_page' | 'viewport' | 'element',
    readonly width: number,
    readonly height: number,
    readonly bytes: number,
    readonly maxBytes: number,
  ) {
    super(`Screenshot request rejected: requested_extent=${requestedExtent}, actual_extent=not_captured, raster=${width}x${height}, bytes=${bytes}, max_bytes=${maxBytes}.`);
    this.name = 'ScreenshotExtentRejectedError';
  }
}

function imageDimensions(buffer: Buffer): { width: number; height: number } | undefined {
  const pngSignature = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
  if (buffer.length >= 24 && buffer.subarray(0, 8).equals(pngSignature) && buffer.toString('ascii', 12, 16) === 'IHDR') {
    return { width: buffer.readUInt32BE(16), height: buffer.readUInt32BE(20) };
  }

  if (buffer.length < 4 || buffer[0] !== 0xff || buffer[1] !== 0xd8) return undefined;
  const startOfFrame = new Set([0xc0, 0xc1, 0xc2, 0xc3, 0xc5, 0xc6, 0xc7, 0xc9, 0xca, 0xcb, 0xcd, 0xce, 0xcf]);
  let offset = 2;
  while (offset + 4 <= buffer.length) {
    if (buffer[offset] !== 0xff) {
      offset += 1;
      continue;
    }
    const marker = buffer[offset + 1];
    offset += 2;
    if (marker === undefined || marker === 0xd9 || marker === 0xda) break;
    if (marker === 0x01 || (marker >= 0xd0 && marker <= 0xd7)) continue;
    if (offset + 2 > buffer.length) break;
    const segmentLength = buffer.readUInt16BE(offset);
    if (segmentLength < 2 || offset + segmentLength > buffer.length) break;
    if (startOfFrame.has(marker) && segmentLength >= 7) {
      return { height: buffer.readUInt16BE(offset + 3), width: buffer.readUInt16BE(offset + 5) };
    }
    offset += segmentLength;
  }
  return undefined;
}

function requireImageDimensions(buffer: Buffer): { width: number; height: number } {
  const dimensions = imageDimensions(buffer);
  if (!dimensions || dimensions.width < 1 || dimensions.height < 1) {
    throw new Error('Screenshot image dimensions could not be read; capture evidence was rejected.');
  }
  return dimensions;
}

function assertFullPagePolicy(page: Page, fullPage: boolean): void {
  if (fullPage && page.video?.()) throw new UnsupportedFullPageScreenshotError();
}

export interface ScreenshotCaptureOptions {
  /** Capture a specific visible element instead of the page viewport. */
  selector?: string;
  /** Override the environment default for this action. */
  fullPage?: boolean;
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
  config: Config,
  options?: ScreenshotCaptureOptions
): Promise<ScreenshotCapture | undefined> {
  if (!config.telemetry.screenshot.enabled) {
    return undefined;
  }

  try {
    const startTime = Date.now();
    const viewport = page.viewportSize();
    const fullPage = options?.fullPage ?? config.telemetry.screenshot.fullPage;
    assertFullPagePolicy(page, fullPage);

    // DOM assertions can succeed before the compositor has painted the final
    // state. Give the browser one short render turn so evidence captures the
    // visible story rather than a freshly navigated background.
    await page.waitForTimeout(50);

    // Build screenshot options
    // When fullPage is false, use explicit clip to prevent viewport modification
    // This is critical for execution mode where frame streaming must show consistent size
    const screenshotOptions: Parameters<typeof page.screenshot>[0] = {
      type: 'png',
      quality: undefined, // PNG doesn't support quality
    };

    if (options?.selector) {
      const target = page.locator(options.selector).first();
      await target.waitFor({ state: 'visible', timeout: 5000 });
      const box = await target.boundingBox();
      if (!box) throw new Error(`Screenshot selector is not measurable: ${options.selector}`);
      const buffer = await target.screenshot({ type: 'png' });
      const dimensions = requireImageDimensions(buffer);
      if (buffer.length > config.telemetry.screenshot.maxSizeBytes) {
        logger.warn('Element screenshot exceeds max size', {
          selector: options.selector,
          size: buffer.length,
          max: config.telemetry.screenshot.maxSizeBytes,
        });
        throw new ScreenshotExtentRejectedError('element', dimensions.width, dimensions.height, buffer.length, config.telemetry.screenshot.maxSizeBytes);
      }
      metrics.screenshotSize.observe(buffer.length);
      return {
        base64: buffer.toString('base64'),
        media_type: 'image/png',
        width: dimensions.width,
        height: dimensions.height,
        requested_extent: 'element',
        actual_extent: 'element',
        degraded: false,
      };
    }

    if (fullPage) {
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
    const raster = requireImageDimensions(buffer);

    // Check size limit
    if (buffer.length > config.telemetry.screenshot.maxSizeBytes) {
      logger.warn('Screenshot exceeds max size, truncating', {
        size: buffer.length,
        maxSize: config.telemetry.screenshot.maxSizeBytes,
      });

      // Preserve the requested full-page extent during degradation. A viewport
      // crop would be a different artifact and must not be returned as success.
      const compressed = await captureCompressedScreenshot(
        page,
        config.telemetry.screenshot.quality,
        fullPage,
        config.telemetry.screenshot.maxSizeBytes
      );
      if (compressed) {
        logger.info('Screenshot captured with JPEG fallback', {
          pngSize: buffer.length,
          maxSize: config.telemetry.screenshot.maxSizeBytes,
        });
        return { ...compressed, requested_extent: fullPage ? 'full_page' : 'viewport', degraded: true };
      }

      logger.warn('Screenshot too large after JPEG fallback', {
        size: buffer.length,
        maxSize: config.telemetry.screenshot.maxSizeBytes,
      });
      throw new ScreenshotExtentRejectedError(fullPage ? 'full_page' : 'viewport', raster.width, raster.height, buffer.length, config.telemetry.screenshot.maxSizeBytes);
    }

    const base64 = buffer.toString('base64');

    const duration = Date.now() - startTime;
    logger.debug('Screenshot captured', {
      size: buffer.length,
      duration,
      fullPage,
    });

    metrics.screenshotSize.observe(buffer.length);

    // Use viewport captured at start, or re-fetch if needed
    const dimensions = requireImageDimensions(buffer);
    return {
      base64,
      media_type: 'image/png',
      width: dimensions.width,
      height: dimensions.height,
      requested_extent: fullPage ? 'full_page' : 'viewport',
      actual_extent: fullPage ? 'full_page' : 'viewport',
      degraded: false,
    };
  } catch (error) {
    if (error instanceof UnsupportedFullPageScreenshotError) throw error;
    if (error instanceof ScreenshotExtentRejectedError) throw error;
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
    assertFullPagePolicy(page, fullPage);

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

      const dimensions = requireImageDimensions(buffer);
      throw new ScreenshotExtentRejectedError(fullPage ? 'full_page' : 'viewport', dimensions.width, dimensions.height, buffer.length, maxSizeBytes);
    }

    const base64 = buffer.toString('base64');

    metrics.screenshotSize.observe(buffer.length);

    // Use viewport captured at start, or re-fetch if needed
    const dimensions = requireImageDimensions(buffer);
    return {
      base64,
      media_type: 'image/jpeg',
      width: dimensions.width,
      height: dimensions.height,
      requested_extent: fullPage ? 'full_page' : 'viewport',
      actual_extent: fullPage ? 'full_page' : 'viewport',
      degraded: false,
    };
  } catch (error) {
    if (error instanceof UnsupportedFullPageScreenshotError) throw error;
    if (error instanceof ScreenshotExtentRejectedError) throw error;
    trackScreenshotFailure();
    logger.warn('telemetry: compressed screenshot capture failed', {
      error: error instanceof Error ? error.message : String(error),
      quality,
      fullPage,
    });
    return undefined;
  }
}
