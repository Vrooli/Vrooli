import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { ScrollParamsSchema } from '../types/instruction';
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
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      // Validate parameters
      const params = ScrollParamsSchema.parse(instruction.params);

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
