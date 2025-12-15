import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getScrollParams } from '../proto';
import { normalizeError } from '../utils';

/**
 * Scroll handler
 *
 * Handles page scroll operations
 */
export class ScrollHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['scroll'];
  }

  async execute(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      // Extract typed params from action
      const typedParams = instruction.action ? getScrollParams(instruction.action) : undefined;
      const params = this.requireTypedParams(typedParams, 'scroll', instruction.nodeId);

      const x = Number(params.x || 0);
      const y = Number(params.y || 0);

      logger.debug('Scrolling page', {
        x,
        y,
      });

      // Scroll window to coordinates
      await page.evaluate(
        ([scrollX, scrollY]) => {
          // @ts-expect-error - window is available in browser context
          window.scrollTo(scrollX, scrollY);
        },
        [x, y]
      );

      logger.info('Scroll successful', {
        x,
        y,
      });

      return {
        success: true,
      };
    } catch (error) {
      logger.error('Scroll failed', {
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
