import { Page } from 'playwright';
import { BaseHandler, HandlerContext, HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getDragDropParams, getGestureParams } from '../types';
import { normalizeError } from '../utils/errors';
import { DEFAULT_DRAG_ANIMATION_STEPS } from '../constants';
import { captureElementContext } from '../telemetry';

/** Internal gesture params type for handler use */
interface GestureParams {
  type?: string;
  selector?: string;
  direction?: string;
  distance?: number;
  scale?: number;
  durationMs?: number;
}

/**
 * Canonical gesture types supported by this handler.
 */
type GestureType = 'drag' | 'swipe' | 'pinch' | 'zoom' | 'unknown';

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

  /**
   * DECISION: Gesture Type Resolution
   *
   * Resolves the canonical gesture type from an instruction.
   * This centralizes the logic for determining what kind of gesture to execute.
   *
   * Resolution rules:
   * - 'drag-drop', 'dragdrop', 'drag' → 'drag'
   * - 'swipe', 'pinch', 'zoom' → as-is (direct match)
   * - 'gesture' with gestureType param → param value
   * - Otherwise → 'unknown'
   */
  private resolveGestureType(instruction: HandlerInstruction): GestureType {
    const type = instruction.type.toLowerCase();

    // Direct drag types
    if (type.includes('drag')) {
      return 'drag';
    }

    // Direct gesture types
    if (type === 'swipe' || type === 'pinch' || type === 'zoom') {
      return type;
    }

    // Generic 'gesture' instruction - resolve from params
    if (type === 'gesture') {
      const typedParams = instruction.action ? getGestureParams(instruction.action) : undefined;
      if (typedParams?.gestureType) {
        const gestureType = typedParams.gestureType;
        if (gestureType === 'swipe' || gestureType === 'pinch' || gestureType === 'zoom') {
          return gestureType;
        }
      }
    }

    return 'unknown';
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { logger } = context;

    try {
      const gestureType = this.resolveGestureType(instruction);

      // Dispatch to appropriate handler based on resolved type
      switch (gestureType) {
        case 'drag':
          return this.handleDragDrop(instruction, context);
        case 'swipe':
        case 'pinch':
        case 'zoom':
          return this.handleGesture(instruction, context);
        case 'unknown':
          return {
            success: false,
            error: {
              message: `Unknown or unsupported gesture type: ${instruction.type}`,
              code: 'INVALID_GESTURE',
              kind: 'user',
              retryable: false,
            },
          };
      }
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
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    // Get typed params from instruction.action (required after migration)
    const typedParams = instruction.action ? getDragDropParams(instruction.action) : undefined;
    const validated = this.requireTypedParams(typedParams, 'drag-drop', instruction.nodeId);
    const { page, logger } = context;

    // Prefer param timeout, fallback to config, then hard-coded default
    const timeout = validated.timeoutMs || context.config.execution.defaultTimeoutMs || 30000;

    logger.debug('drag-drop: starting operation', {
      sourceSelector: validated.sourceSelector,
      targetSelector: validated.targetSelector,
      offset: validated.offsetX || validated.offsetY ? { x: validated.offsetX, y: validated.offsetY } : undefined,
      steps: validated.steps,
      timeout,
    });

    // Capture element context for source element BEFORE the drag (recording-quality telemetry)
    const sourceElementContext = await captureElementContext(page, validated.sourceSelector, { timeout });

    // Get source element with explicit timeout and wait for visible state
    const sourceElement = await page.waitForSelector(validated.sourceSelector, {
      timeout,
      state: 'visible'
    }).catch((error) => {
      logger.debug('drag-drop: source element wait failed', {
        selector: validated.sourceSelector,
        timeout,
        error: error instanceof Error ? error.message : String(error),
      });
      return null;
    });

    if (!sourceElement) {
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
      // Drag to target element with explicit timeout
      // Target only needs to be attached (not necessarily visible) for drag operations
      const targetElement = await page.waitForSelector(validated.targetSelector, {
        timeout,
        state: 'attached'
      }).catch((error) => {
        logger.debug('drag-drop: target element wait failed', {
          selector: validated.targetSelector,
          timeout,
          error: error instanceof Error ? error.message : String(error),
        });
        return null;
      });

      if (!targetElement) {
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
    const steps = validated.steps || DEFAULT_DRAG_ANIMATION_STEPS;

    await page.mouse.move(sourceX, sourceY);
    await page.mouse.down();

    // Animate drag if steps > 1
    if (steps > 1) {
      const deltaX = (targetX - sourceX) / steps;
      const deltaY = (targetY - sourceY) / steps;
      const delayMs = validated.delayMs || 0;

      for (let i = 1; i <= steps; i++) {
        const currentX = sourceX + deltaX * i;
        const currentY = sourceY + deltaY * i;
        await page.mouse.move(currentX, currentY);
        if (delayMs > 0) {
          await page.waitForTimeout(delayMs);
        }
      }
    } else {
      await page.mouse.move(targetX, targetY);
    }

    await page.mouse.up();

    logger.info('drag-drop: completed', {
      from: { x: sourceX, y: sourceY },
      to: { x: targetX, y: targetY },
      steps,
    });

    return {
      success: true,
      elementContext: sourceElementContext,
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
        selector: sourceElementContext.selector,
        bounding_box: sourceElementContext.boundingBox ? {
          x: sourceElementContext.boundingBox.x,
          y: sourceElementContext.boundingBox.y,
          width: sourceElementContext.boundingBox.width,
          height: sourceElementContext.boundingBox.height,
        } : undefined,
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
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    // Get typed params from instruction.action (required after migration)
    const typedParams = instruction.action ? getGestureParams(instruction.action) : undefined;
    const gestureParams = this.requireTypedParams(typedParams, 'gesture', instruction.nodeId);
    // Map typed params to expected format
    const validated: GestureParams = {
      type: gestureParams.gestureType,
      selector: gestureParams.selector,
      direction: gestureParams.direction,
      distance: gestureParams.distance,
      scale: gestureParams.scale,
      durationMs: gestureParams.durationMs,
    };
    const { page, logger } = context;

    logger.debug('Executing gesture', {
      type: validated.type,
      direction: validated.direction,
      distance: validated.distance,
      scale: validated.scale,
    });

    switch (validated.type) {
      case 'swipe':
        return this.handleSwipe(page, validated, logger, context.config.recording.defaultSwipeDistance);
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
  private async handleSwipe(page: Page, params: GestureParams, logger: any, defaultDistance: number = 300): Promise<HandlerResult> {
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

    // Prefer param distance, fallback to config default
    const distance = params.distance || defaultDistance;
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
      // Capture element context BEFORE the zoom (recording-quality telemetry)
      const elementContext = await captureElementContext(page, selector);

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

      logger.info('Element zoom applied', { selector, scale });

      return {
        success: true,
        elementContext,
        extracted_data: {
          zoom: {
            scale,
            applied: 'element',
            selector,
          },
        },
        focus: elementContext.boundingBox ? {
          selector: elementContext.selector,
          bounding_box: {
            x: elementContext.boundingBox.x,
            y: elementContext.boundingBox.y,
            width: elementContext.boundingBox.width,
            height: elementContext.boundingBox.height,
          },
        } : undefined,
      };
    }
  }
}
