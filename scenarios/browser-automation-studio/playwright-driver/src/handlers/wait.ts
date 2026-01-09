import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getWaitParams } from '../types';
import { DEFAULT_WAIT_TIMEOUT_MS } from '../constants';
import { normalizeError } from '../utils';
import { captureElementContext } from '../telemetry';

/**
 * Wait handler
 *
 * Handles wait operations (selector or timeout)
 */
export class WaitHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['wait'];
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      // Extract typed params from action
      const typedParams = instruction.action ? getWaitParams(instruction.action) : undefined;
      const params = this.requireTypedParams(typedParams, 'wait', instruction.nodeId);

      if (params.selector) {
        // Wait for selector
        // Prefer config timeout, fallback to param, then constant default
        const timeout = params.timeoutMs || context.config.execution.waitTimeoutMs || DEFAULT_WAIT_TIMEOUT_MS;

        logger.debug('Waiting for selector', {
          selector: params.selector,
          timeout,
        });

        await page.waitForSelector(params.selector, {
          timeout,
          state: (params.state || 'visible') as 'attached' | 'detached' | 'visible' | 'hidden',
        });

        logger.info('Wait for selector successful', {
          selector: params.selector,
        });

        // Capture element context AFTER the wait completes (element now exists)
        const elementContext = await captureElementContext(page, params.selector);

        return {
          success: true,
          elementContext,
          focus: {
            selector: elementContext.selector,
            bounding_box: elementContext.boundingBox ? {
              x: elementContext.boundingBox.x,
              y: elementContext.boundingBox.y,
              width: elementContext.boundingBox.width,
              height: elementContext.boundingBox.height,
            } : undefined,
          },
        };
      } else {
        // Wait for timeout using durationMs from typed params
        const timeout = params.durationMs || 1000;

        logger.debug('Waiting for timeout', {
          timeout,
        });

        await page.waitForTimeout(timeout);

        logger.info('Wait for timeout successful', {
          timeout,
        });

        return {
          success: true,
        };
      }
    } catch (error) {
      logger.error('Wait failed', {
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
