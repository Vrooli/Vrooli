import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { WaitParamsSchema } from '../types/instruction';
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
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      // Validate parameters
      const params = WaitParamsSchema.parse(instruction.params);

      if (params.selector) {
        // Wait for selector
        const timeout = params.timeoutMs || DEFAULT_WAIT_TIMEOUT_MS;

        logger.debug('Waiting for selector', {
          selector: params.selector,
          timeout,
        });

        await page.waitForSelector(params.selector, {
          timeout,
          state: params.state || 'visible',
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
        // Wait for timeout
        const timeout = params.timeoutMs || params.ms || 1000;

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
