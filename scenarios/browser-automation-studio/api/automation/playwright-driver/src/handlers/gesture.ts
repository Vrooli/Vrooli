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

    logger.debug('Executing drag-drop', {
      sourceSelector: validated.sourceSelector,
      targetSelector: validated.targetSelector,
      offset: validated.offsetX || validated.offsetY ? { x: validated.offsetX, y: validated.offsetY } : undefined,
      steps: validated.steps,
    });

    // Get source element
    const sourceElement = await page.$(validated.sourceSelector);
    if (!sourceElement) {
      return {
        success: false,
        error: {
          message: `Source element not found: ${validated.sourceSelector}`,
          code: 'ELEMENT_NOT_FOUND',
          kind: 'user',
          retryable: false,
        },
      };
    }

    const sourceBoundingBox = await sourceElement.boundingBox();
    if (!sourceBoundingBox) {
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

    let targetX: number;
    let targetY: number;
    let targetBoundingBox: { x: number; y: number; width: number; height: number } | null = null;

    // Determine target position
    if (validated.targetSelector) {
      // Drag to target element
      const targetElement = await page.$(validated.targetSelector);
      if (!targetElement) {
        return {
          success: false,
          error: {
            message: `Target element not found: ${validated.targetSelector}`,
            code: 'ELEMENT_NOT_FOUND',
            kind: 'user',
            retryable: false,
          },
        };
      }

      targetBoundingBox = await targetElement.boundingBox();
      if (!targetBoundingBox) {
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
    } else if (validated.offsetX !== undefined || validated.offsetY !== undefined) {
      // Drag by offset
      targetX = sourceX + (validated.offsetX || 0);
      targetY = sourceY + (validated.offsetY || 0);
    } else {
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
    await page.mouse.move(sourceX, sourceY);
    await page.mouse.down();

    // Animate drag if steps > 1
    if (steps > 1) {
      const deltaX = (targetX - sourceX) / steps;
      const deltaY = (targetY - sourceY) / steps;
      const delayMs = validated.delayMs || 0;

      for (let i = 1; i <= steps; i++) {
        await page.mouse.move(sourceX + deltaX * i, sourceY + deltaY * i);
        if (delayMs > 0) {
          await page.waitForTimeout(delayMs);
        }
      }
    } else {
      await page.mouse.move(targetX, targetY);
    }

    await page.mouse.up();

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
      },
      elementBoundingBox: sourceBoundingBox,
      clickPosition: { x: targetX, y: targetY },
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
      await page.evaluate((scaleValue) => {
        document.body.style.transform = `scale(${scaleValue})`;
        document.body.style.transformOrigin = 'center center';
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

      await element.evaluate((el, scaleValue) => {
        (el as HTMLElement).style.transform = `scale(${scaleValue})`;
        (el as HTMLElement).style.transformOrigin = 'center center';
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
        elementBoundingBox: boundingBox || undefined,
      };
    }
  }
}
