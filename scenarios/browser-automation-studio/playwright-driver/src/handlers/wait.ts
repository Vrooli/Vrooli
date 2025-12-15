import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getWaitParams } from '../proto';
import { DEFAULT_WAIT_TIMEOUT_MS } from '../constants';
import { normalizeError } from '../utils';

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

        // Get bounding box of found element
        const boundingBox = await this.getBoundingBox(page, params.selector).catch(() => null);

        return {
          success: true,
          focus: {
            selector: params.selector,
            bounding_box: boundingBox || undefined,
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
