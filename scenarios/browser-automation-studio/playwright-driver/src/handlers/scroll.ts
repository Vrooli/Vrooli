import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getScrollParams } from '../types';
import { normalizeError } from '../utils';
import {
  getBehaviorFromContext,
  executeHumanScroll,
  executeSmoothScroll,
  applyPreActionDelay,
} from './behavior-utils';

/**
 * Scroll handler
 *
 * Handles page scroll operations with human-like behavior support.
 *
 * Behavior settings used:
 * - scroll_style: 'smooth' uses native CSS smooth scroll, 'stepped' scrolls in increments
 * - scroll_speed_min/max: Controls step size for stepped scrolling
 * - micro_pause_*: Adds random pauses during scroll for natural appearance
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

      // Get behavior settings from context
      const behavior = getBehaviorFromContext(context);
      const scrollStyle = behavior ? behavior.getScrollStyle() : 'instant';

      logger.debug('Scrolling page', {
        x,
        y,
        humanBehavior: !!behavior,
        scrollStyle,
      });

      // Apply pre-scroll delay if behavior is enabled
      // Use click delay as a reasonable pre-action delay for scroll
      await applyPreActionDelay(behavior, (b) => b.getClickDelay() / 2);

      if (behavior) {
        if (scrollStyle === 'smooth') {
          // Use native smooth scrolling
          await executeSmoothScroll(page, x, y, behavior);
        } else {
          // Use stepped scrolling with human-like timing
          await executeHumanScroll(page, x, y, behavior);
        }
      } else {
        // No behavior configured - instant scroll
        await page.evaluate(
          ([scrollX, scrollY]) => {
            // @ts-expect-error - window is available in browser context
            window.scrollTo(scrollX, scrollY);
          },
          [x, y]
        );
      }

      logger.info('Scroll successful', {
        x,
        y,
        scrollStyle,
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
