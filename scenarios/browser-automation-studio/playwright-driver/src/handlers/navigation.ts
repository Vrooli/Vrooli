import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { NavigateParamsSchema } from '../types/instruction';
import { DEFAULT_NAVIGATION_TIMEOUT_MS } from '../constants';
import { normalizeError } from '../utils';

/**
 * Navigation handler
 *
 * Handles page navigation operations
 */
export class NavigationHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['navigate'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      // Validate and parse parameters
      const params = NavigateParamsSchema.parse(instruction.params);

      // Get URL from params (support multiple param names for backwards compatibility)
      const url = params.url;
      if (!url) {
        return {
          success: false,
          error: {
            message: 'navigate instruction missing url parameter',
            code: 'MISSING_PARAM',
            kind: 'orchestration',
            retryable: false,
          },
        };
      }

      const timeout = params.timeoutMs || DEFAULT_NAVIGATION_TIMEOUT_MS;
      const waitUntil = params.waitUntil || 'networkidle';

      logger.debug('Navigating to URL', {
        url,
        timeout,
        waitUntil,
      });

      // Navigate
      await page.goto(url, {
        timeout,
        waitUntil,
      });

      logger.info('Navigation successful', {
        url,
        finalUrl: page.url(),
      });

      return {
        success: true,
      };
    } catch (error) {
      logger.error('Navigation failed', {
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
