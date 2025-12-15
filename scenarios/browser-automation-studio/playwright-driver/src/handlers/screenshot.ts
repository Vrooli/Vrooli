import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getScreenshotParams } from '../proto';
import { captureScreenshot, captureCompressedScreenshot } from '../telemetry';
import { normalizeError } from '../utils';

/**
 * Screenshot handler
 *
 * Handles screenshot capture operations
 */
export class ScreenshotHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['screenshot'];
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, config, logger } = context;

    try {
      // Extract typed params from action
      const typedParams = instruction.action ? getScreenshotParams(instruction.action) : undefined;
      const params = this.requireTypedParams(typedParams, 'screenshot', instruction.nodeId);

      logger.debug('Capturing screenshot', {
        fullPage: params.fullPage !== false,
        quality: params.quality,
      });

      // Capture screenshot
      let screenshot;
      if (params.quality && params.quality < 100) {
        // Use compressed JPEG
        screenshot = await captureCompressedScreenshot(
          page,
          params.quality,
          params.fullPage !== false,
          config.telemetry.screenshot.maxSizeBytes
        );
      } else {
        // Use standard PNG
        screenshot = await captureScreenshot(page, config);
      }

      if (!screenshot) {
        return {
          success: false,
          error: {
            message: 'Failed to capture screenshot',
            code: 'SCREENSHOT_FAILED',
            kind: 'engine',
            retryable: true,
          },
        };
      }

      logger.info('Screenshot captured', {
        media_type: screenshot.media_type,
        size: `${screenshot.width}x${screenshot.height}`,
      });

      return {
        success: true,
        screenshot,
      };
    } catch (error) {
      logger.error('Screenshot failed', {
        error: error instanceof Error ? error.message : String(error),
      });

      const driverError = normalizeError(error);

      return {
        success: false,
        error: {
          message: driverError.message,
          code: driverError.code,
          kind: driverError.kind,
          retryable: driverError.retryable,
        },
      };
    }
  }
}
