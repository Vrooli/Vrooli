import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getRotateParams } from '../types';
import { normalizeError } from '../utils';

/**
 * DeviceHandler implements device emulation and orientation changes
 *
 * Supported instruction types:
 * - rotate: Change device orientation (portrait/landscape)
 *
 * Operations:
 * - Rotate device to portrait or landscape
 * - Set specific rotation angle (0, 90, 180, 270)
 * - Update viewport dimensions based on orientation
 *
 * Phase 3 handler - Device operations
 */
export class DeviceHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['rotate', 'orientation', 'device'];
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      // Get typed params from instruction.action (required after migration)
      const typedParams = instruction.action ? getRotateParams(instruction.action) : undefined;
      const params = this.requireTypedParams(typedParams, 'rotate', instruction.nodeId);

      logger.debug('Executing device rotation', {
        orientation: params.orientation,
        angle: params.angle,
      });

      return await this.handleRotate(params, page, logger);
    } catch (error) {
      logger.error('Device rotation failed', {
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

  /**
   * Handle device rotation
   *
   * Changes viewport dimensions to match orientation
   */
  private async handleRotate(
    params: Record<string, unknown>,
    page: any,
    logger: any
  ): Promise<HandlerResult> {
    const currentViewport = page.viewportSize();

    if (!currentViewport) {
      return {
        success: false,
        error: {
          message: 'Viewport size not available',
          code: 'NO_VIEWPORT',
          kind: 'engine',
          retryable: false,
        },
      };
    }

    // Determine target dimensions based on orientation or angle
    let targetWidth: number;
    let targetHeight: number;
    let actualAngle: number;

    const orientation = (params as any).orientation;
    const angle = (params as any).angle;

    if (orientation) {
      // Use orientation to determine dimensions
      const currentIsPortrait = currentViewport.height > currentViewport.width;
      const targetIsPortrait = orientation === 'portrait';

      if (currentIsPortrait === targetIsPortrait) {
        // Already in correct orientation
        logger.info('Already in target orientation', {
          orientation,
        });
        targetWidth = currentViewport.width;
        targetHeight = currentViewport.height;
        actualAngle = currentIsPortrait ? 0 : 90;
      } else {
        // Swap dimensions
        targetWidth = currentViewport.height;
        targetHeight = currentViewport.width;
        actualAngle = targetIsPortrait ? 0 : 90;
      }
    } else if (angle !== undefined) {
      // Use angle to determine dimensions
      actualAngle = angle;

      switch (angle) {
        case 0:
        case 180:
          // Portrait or upside-down portrait
          if (currentViewport.height < currentViewport.width) {
            // Need to swap
            targetWidth = currentViewport.height;
            targetHeight = currentViewport.width;
          } else {
            targetWidth = currentViewport.width;
            targetHeight = currentViewport.height;
          }
          break;
        case 90:
        case 270:
          // Landscape
          if (currentViewport.width < currentViewport.height) {
            // Need to swap
            targetWidth = currentViewport.height;
            targetHeight = currentViewport.width;
          } else {
            targetWidth = currentViewport.width;
            targetHeight = currentViewport.height;
          }
          break;
        default:
          return {
            success: false,
            error: {
              message: `Invalid rotation angle: ${angle}. Must be 0, 90, 180, or 270`,
              code: 'INVALID_ANGLE',
              kind: 'orchestration',
              retryable: false,
            },
          };
      }
    } else {
      return {
        success: false,
        error: {
          message: 'Must provide either orientation or angle',
          code: 'MISSING_PARAM',
          kind: 'orchestration',
          retryable: false,
        },
      };
    }

    // Set new viewport size
    await page.setViewportSize({
      width: targetWidth,
      height: targetHeight,
    });

    // Apply CSS transform for visual rotation if angle is 180 or 270
    if (actualAngle === 180 || actualAngle === 270) {
      await page.evaluate((rotationAngle: number) => {
        // @ts-expect-error - document is available in browser context
        document.body.style.transform = `rotate(${rotationAngle}deg)`;
        // @ts-expect-error - document is available in browser context
        document.body.style.transformOrigin = 'center center';
        // @ts-expect-error - document is available in browser context
        document.body.style.width = '100vw';
        // @ts-expect-error - document is available in browser context
        document.body.style.height = '100vh';
      }, actualAngle);
    } else {
      // Remove any existing rotation
      await page.evaluate(() => {
        // @ts-expect-error - document is available in browser context
        document.body.style.transform = '';
        // @ts-expect-error - document is available in browser context
        document.body.style.transformOrigin = '';
        // @ts-expect-error - document is available in browser context
        document.body.style.width = '';
        // @ts-expect-error - document is available in browser context
        document.body.style.height = '';
      });
    }

    const finalOrientation = targetHeight > targetWidth ? 'portrait' : 'landscape';

    logger.info('Device rotated', {
      fromViewport: currentViewport,
      toViewport: { width: targetWidth, height: targetHeight },
      orientation: finalOrientation,
      angle: actualAngle,
    });

    return {
      success: true,
      extracted_data: {
        device: {
          operation: 'rotate',
          orientation: finalOrientation,
          angle: actualAngle,
          viewport: {
            width: targetWidth,
            height: targetHeight,
          },
          previousViewport: currentViewport,
        },
      },
    };
  }
}
