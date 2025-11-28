import { Page } from 'playwright';
import { BaseHandler, HandlerContext, HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import {
  DragDropParamsSchema,
  GestureParamsSchema,
  type GestureParams,
} from '../types/instruction';
import { normalizeError } from '../utils/errors';

/**
 * GestureHandler implements complex mouse/touch gestures
 *
 * Supported instruction types:
 * - drag-drop: Drag element from source to target or by offset
 * - swipe: Swipe gesture (mobile/touch emulation)
 * - pinch: Pinch-to-zoom gesture
 * - zoom: Zoom gesture
 *
 * Phase 3 handler - Advanced interactions
 */
export class GestureHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['drag-drop', 'dragdrop', 'drag', 'swipe', 'pinch', 'zoom', 'gesture'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { logger } = context;

    try {
      const instructionType = instruction.type.toLowerCase();

      // Route to appropriate handler based on instruction type
      if (instructionType.includes('drag')) {
        return this.handleDragDrop(instruction, context);
      } else if (instructionType === 'swipe' || instructionType === 'pinch' || instructionType === 'zoom') {
        return this.handleGesture(instruction, context);
      } else if (instructionType === 'gesture') {
        // Generic gesture - check params.type
        const gestureParams = GestureParamsSchema.parse(instruction.params);
        if (gestureParams.type === 'swipe' || gestureParams.type === 'pinch' || gestureParams.type === 'zoom') {
          return this.handleGesture(instruction, context);
        }
        return {
          success: false,
          error: {
            message: `Unknown gesture type: ${gestureParams.type}`,
            code: 'INVALID_GESTURE',
            kind: 'user',
            retryable: false,
          },
        };
      }

      return {
        success: false,
        error: {
          message: `Unsupported instruction type: ${instructionType}`,
          code: 'INVALID_TYPE',
          kind: 'user',
          retryable: false,
        },
      };
    } catch (error) {
      logger.error('Gesture operation failed', {
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
   * Handle drag-and-drop operations
   *
   * Supports:
   * - Drag from source to target element
   * - Drag by offset (x, y)
   * - Animated drag with configurable steps
   */
  private async handleDragDrop(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const validated = DragDropParamsSchema.parse(instruction.params);
    const { page, logger } = context;

    const timeout = validated.timeoutMs || 30000;

    logger.info('🎯 [DRAGDROP] Starting drag-drop operation', {
      sourceSelector: validated.sourceSelector,
      targetSelector: validated.targetSelector,
      offset: validated.offsetX || validated.offsetY ? { x: validated.offsetX, y: validated.offsetY } : undefined,
      steps: validated.steps,
      timeout,
    });

    // Get source element with explicit timeout and wait for visible state
    logger.info('🔍 [DRAGDROP] Waiting for source element...');
    const sourceElement = await page.waitForSelector(validated.sourceSelector, {
      timeout,
      state: 'visible'
    }).catch((error) => {
      logger.error('❌ [DRAGDROP] Source element wait failed', {
        selector: validated.sourceSelector,
        timeout,
        error: error instanceof Error ? error.message : String(error),
      });
      return null;
    });

    if (!sourceElement) {
      logger.error('❌ [DRAGDROP] Source element not found');
      return {
        success: false,
        error: {
          message: `Source element not found or not visible within ${timeout}ms: ${validated.sourceSelector}`,
          code: 'ELEMENT_NOT_FOUND',
          kind: 'user',
          retryable: false,
        },
      };
    }

    logger.info('✅ [DRAGDROP] Source element found, getting bounding box...');
    const sourceBoundingBox = await sourceElement.boundingBox();
    if (!sourceBoundingBox) {
      logger.error('❌ [DRAGDROP] Could not get source bounding box');
      return {
        success: false,
        error: {
          message: `Could not get bounding box for source element: ${validated.sourceSelector}`,
          code: 'NO_BOUNDING_BOX',
          kind: 'engine',
          retryable: true,
        },
      };
    }

    // Calculate source position (center of element)
    const sourceX = sourceBoundingBox.x + sourceBoundingBox.width / 2;
    const sourceY = sourceBoundingBox.y + sourceBoundingBox.height / 2;
    logger.info('✅ [DRAGDROP] Source position calculated', { sourceX, sourceY, box: sourceBoundingBox });

    let targetX: number;
    let targetY: number;
    let targetBoundingBox: { x: number; y: number; width: number; height: number } | null = null;

    // Determine target position
    if (validated.targetSelector) {
      // Drag to target element with explicit timeout
      // Target only needs to be attached (not necessarily visible) for drag operations
      logger.info('🔍 [DRAGDROP] Waiting for target element...');
      const targetElement = await page.waitForSelector(validated.targetSelector, {
        timeout,
        state: 'attached'
      }).catch((error) => {
        logger.error('❌ [DRAGDROP] Target element wait failed', {
          selector: validated.targetSelector,
          timeout,
          error: error instanceof Error ? error.message : String(error),
        });
        return null;
      });

      if (!targetElement) {
        logger.error('❌ [DRAGDROP] Target element not found');
        return {
          success: false,
          error: {
            message: `Target element not found within ${timeout}ms: ${validated.targetSelector}`,
            code: 'ELEMENT_NOT_FOUND',
            kind: 'user',
            retryable: false,
          },
        };
      }

      logger.info('✅ [DRAGDROP] Target element found, getting bounding box...');
      targetBoundingBox = await targetElement.boundingBox();
      if (!targetBoundingBox) {
        logger.error('❌ [DRAGDROP] Could not get target bounding box');
        return {
          success: false,
          error: {
            message: `Could not get bounding box for target element: ${validated.targetSelector}`,
            code: 'NO_BOUNDING_BOX',
            kind: 'engine',
            retryable: true,
          },
        };
      }

      targetX = targetBoundingBox.x + targetBoundingBox.width / 2;
      targetY = targetBoundingBox.y + targetBoundingBox.height / 2;
      logger.info('✅ [DRAGDROP] Target position calculated', { targetX, targetY, box: targetBoundingBox });
    } else if (validated.offsetX !== undefined || validated.offsetY !== undefined) {
      // Drag by offset
      targetX = sourceX + (validated.offsetX || 0);
      targetY = sourceY + (validated.offsetY || 0);
      logger.info('✅ [DRAGDROP] Target position from offset', { targetX, targetY, offset: { x: validated.offsetX, y: validated.offsetY } });
    } else {
      logger.error('❌ [DRAGDROP] No target or offset specified');
      return {
        success: false,
        error: {
          message: 'Either targetSelector or offset (offsetX/offsetY) must be provided',
          code: 'MISSING_PARAMS',
          kind: 'user',
          retryable: false,
        },
      };
    }

    // Perform drag-and-drop
    const steps = validated.steps || 10;
    logger.info('🖱️ [DRAGDROP] Starting mouse operations', { steps, from: { x: sourceX, y: sourceY }, to: { x: targetX, y: targetY } });

    logger.info('🖱️ [DRAGDROP] Moving to source position...');
    await page.mouse.move(sourceX, sourceY);
    logger.info('✅ [DRAGDROP] Moved to source');

    logger.info('🖱️ [DRAGDROP] Mouse down...');
    await page.mouse.down();
    logger.info('✅ [DRAGDROP] Mouse down complete');

    // Animate drag if steps > 1
    if (steps > 1) {
      const deltaX = (targetX - sourceX) / steps;
      const deltaY = (targetY - sourceY) / steps;
      const delayMs = validated.delayMs || 0;
      logger.info('🖱️ [DRAGDROP] Animating drag with steps', { steps, delayMs, delta: { x: deltaX, y: deltaY } });

      for (let i = 1; i <= steps; i++) {
        const currentX = sourceX + deltaX * i;
        const currentY = sourceY + deltaY * i;
        logger.info(`🖱️ [DRAGDROP] Step ${i}/${steps}`, { x: currentX, y: currentY });
        await page.mouse.move(currentX, currentY);
        if (delayMs > 0) {
          await page.waitForTimeout(delayMs);
        }
      }
      logger.info('✅ [DRAGDROP] Animation complete');
    } else {
      logger.info('🖱️ [DRAGDROP] Moving directly to target...');
      await page.mouse.move(targetX, targetY);
      logger.info('✅ [DRAGDROP] Moved to target');
    }

    logger.info('🖱️ [DRAGDROP] Mouse up...');
    await page.mouse.up();
    logger.info('✅ [DRAGDROP] Mouse up complete');

    logger.info('Drag-drop completed', {
      from: { x: sourceX, y: sourceY },
      to: { x: targetX, y: targetY },
      steps,
    });

    return {
      success: true,
      extracted_data: {
        source: {
          selector: validated.sourceSelector,
          position: { x: sourceX, y: sourceY },
          boundingBox: sourceBoundingBox,
        },
        target: {
          selector: validated.targetSelector,
          position: { x: targetX, y: targetY },
          boundingBox: targetBoundingBox,
        },
        clickPosition: { x: targetX, y: targetY },
      },
      focus: {
        selector: validated.sourceSelector,
        bounding_box: sourceBoundingBox,
      },
    };
  }

  /**
   * Handle touch/mobile gestures
   *
   * Supports:
   * - swipe: Swipe in direction (up, down, left, right)
   * - pinch: Pinch-to-zoom (scale < 1.0 = zoom out)
   * - zoom: Zoom gesture (scale > 1.0 = zoom in)
   */
  private async handleGesture(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const validated = GestureParamsSchema.parse(instruction.params);
    const { page, logger } = context;

    logger.debug('Executing gesture', {
      type: validated.type,
      direction: validated.direction,
      distance: validated.distance,
      scale: validated.scale,
    });

    switch (validated.type) {
      case 'swipe':
        return this.handleSwipe(page, validated, logger);
      case 'pinch':
      case 'zoom':
        return this.handlePinchZoom(page, validated, logger);
      default:
        return {
          success: false,
          error: {
            message: `Unknown gesture type: ${validated.type}`,
            code: 'INVALID_GESTURE',
            kind: 'user',
            retryable: false,
          },
        };
    }
  }

  /**
   * Execute swipe gesture
   */
  private async handleSwipe(page: Page, params: GestureParams, logger: any): Promise<HandlerResult> {
    const viewport = page.viewportSize();
    if (!viewport) {
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

    const distance = params.distance || 300;
    const centerX = viewport.width / 2;
    const centerY = viewport.height / 2;

    let startX = centerX;
    let startY = centerY;
    let endX = centerX;
    let endY = centerY;

    switch (params.direction) {
      case 'up':
        startY = centerY + distance / 2;
        endY = centerY - distance / 2;
        break;
      case 'down':
        startY = centerY - distance / 2;
        endY = centerY + distance / 2;
        break;
      case 'left':
        startX = centerX + distance / 2;
        endX = centerX - distance / 2;
        break;
      case 'right':
        startX = centerX - distance / 2;
        endX = centerX + distance / 2;
        break;
      default:
        return {
          success: false,
          error: {
            message: `Invalid swipe direction: ${params.direction}`,
            code: 'INVALID_DIRECTION',
            kind: 'user',
            retryable: false,
          },
        };
    }

    // Execute swipe
    await page.mouse.move(startX, startY);
    await page.mouse.down();
    await page.mouse.move(endX, endY, { steps: 10 });
    await page.mouse.up();

    logger.info('Swipe completed', {
      direction: params.direction,
      from: { x: startX, y: startY },
      to: { x: endX, y: endY },
    });

    return {
      success: true,
      extracted_data: {
        swipe: {
          direction: params.direction,
          distance,
          from: { x: startX, y: startY },
          to: { x: endX, y: endY },
        },
      },
    };
  }

  /**
   * Execute pinch/zoom gesture
   *
   * Note: This is a basic implementation using CSS transform.
   * For true touch events, consider using touch APIs or CDP commands.
   */
  private async handlePinchZoom(page: Page, params: GestureParams, logger: any): Promise<HandlerResult> {
    const scale = params.scale || 1.0;
    const selector = params.selector;

    if (!selector) {
      // Apply zoom to entire page via CSS transform
      await page.evaluate((scaleValue: any) => {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (globalThis as any).document.body.style.transform = `scale(${scaleValue})`;
        (globalThis as any).document.body.style.transformOrigin = 'center center';
      }, scale);

      logger.info('Page zoom applied', { scale });

      return {
        success: true,
        extracted_data: {
          zoom: {
            scale,
            applied: 'page',
          },
        },
      };
    } else {
      // Apply zoom to specific element
      const element = await page.$(selector);
      if (!element) {
        return {
          success: false,
          error: {
            message: `Element not found: ${selector}`,
            code: 'ELEMENT_NOT_FOUND',
            kind: 'user',
            retryable: false,
          },
        };
      }

      await element.evaluate((el: any, scaleValue) => {
        el.style.transform = `scale(${scaleValue})`;
        el.style.transformOrigin = 'center center';
      }, scale);

      const boundingBox = await element.boundingBox();

      logger.info('Element zoom applied', { selector, scale });

      return {
        success: true,
        extracted_data: {
          zoom: {
            scale,
            applied: 'element',
            selector,
          },
        },
        focus: boundingBox ? {
          selector,
          bounding_box: boundingBox,
        } : undefined,
      };
    }
  }
}
