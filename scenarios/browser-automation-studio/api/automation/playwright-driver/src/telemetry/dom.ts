import type { Page } from 'playwright';
import type { DOMSnapshot } from '../types';
import type { Config } from '../config';
import { logger } from '../utils';

/**
 * Capture DOM snapshot from page
 *
 * Returns HTML content with size limit enforcement
 */
export async function captureDOMSnapshot(
  page: Page,
  config: Config
): Promise<DOMSnapshot | undefined> {
  if (!config.telemetry.dom.enabled) {
    return undefined;
  }

  try {
    const html = await page.content();

    // Check size limit
    if (html.length > config.telemetry.dom.maxSizeBytes) {
      logger.warn('DOM snapshot exceeds max size, truncating', {
        size: html.length,
        maxSize: config.telemetry.dom.maxSizeBytes,
      });

      const truncated = html.substring(0, config.telemetry.dom.maxSizeBytes);
      const preview = truncated.substring(0, 512);

      return {
        html: truncated,
        preview,
        size_bytes: truncated.length,
      };
    }

    const preview = html.substring(0, 512);

    logger.debug('DOM snapshot captured', {
      size: html.length,
    });

    return {
      html,
      preview,
      size_bytes: html.length,
    };
  } catch (error) {
    logger.error('Failed to capture DOM snapshot', {
      error: error instanceof Error ? error.message : String(error),
    });
    return undefined;
  }
}

/**
 * Capture DOM snapshot of specific element
 */
export async function captureElementSnapshot(
  page: Page,
  selector: string,
  config: Config
): Promise<DOMSnapshot | undefined> {
  if (!config.telemetry.dom.enabled) {
    return undefined;
  }

  try {
    const element = await page.locator(selector).first();
    const html = await element.innerHTML();

    if (html.length > config.telemetry.dom.maxSizeBytes) {
      const truncated = html.substring(0, config.telemetry.dom.maxSizeBytes);
      const preview = truncated.substring(0, 512);

      return {
        html: truncated,
        preview,
        size_bytes: truncated.length,
      };
    }

    const preview = html.substring(0, 512);

    return {
      html,
      preview,
      size_bytes: html.length,
    };
  } catch (error) {
    logger.error('Failed to capture element snapshot', {
      selector,
      error: error instanceof Error ? error.message : String(error),
    });
    return undefined;
  }
}
