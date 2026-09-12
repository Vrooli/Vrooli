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

    // Extract typed params from action up front so the error path can preserve
    // the known selector even if validation below fails.
    const typedParams = instruction.action ? getWaitParams(instruction.action) : undefined;

    try {
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
        const attributes = elementContext.elementMeta?.attributes ?? {};
        const extracted_data: Record<string, string> = {};
        if (attributes['data-experience-surface']) {
          extracted_data.experience_surface_id = attributes['data-experience-surface'];
        }
        if (attributes['data-experience-state']) {
          extracted_data.experience_surface_state = attributes['data-experience-state'];
        }

        return {
          success: true,
          elementContext,
          extracted_data: Object.keys(extracted_data).length > 0 ? extracted_data : undefined,
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

      const driverError = normalizeError(error, { selector: typedParams?.selector });

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
