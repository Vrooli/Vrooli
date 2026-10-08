import type { ElementHandle } from 'rebrowser-playwright';
import { BaseHandler, getDocument, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getScrollParams } from '../types';
import { normalizeError } from '../utils';
import {
  getBehaviorFromContext,
  executeHumanScroll,
  executeSmoothScroll,
  applyPreActionDelay,
  scrollTarget,
  resolveTimeoutFromContext,
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

  async execute(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    const { logger } = context;
    try {
      const page = getDocument(context);
      // Extract typed params from action
      const typedParams = instruction.action ? getScrollParams(instruction.action) : undefined;
      const params = this.requireTypedParams(typedParams, 'scroll', instruction.nodeId);

      const timeout = resolveTimeoutFromContext(undefined, context, 'default');
      const target: ElementHandle<Element> | null = params.selector
        ? await page.waitForSelector(params.selector, { state: 'attached', timeout })
        : (
            await page.evaluateHandle(() => document.scrollingElement ?? document.documentElement)
          ).asElement();
      if (!target) throw new Error('Scroll target is unavailable');
      try {
        const position = await target.evaluate((element) => ({
          x: element.scrollLeft,
          y: element.scrollTop,
          maxX: Math.max(0, element.scrollWidth - element.clientWidth),
          maxY: Math.max(0, element.scrollHeight - element.clientHeight),
        }));
        // Absolute coordinates take precedence on their axis. Missing axes stay put.
        const x = Math.max(
          0,
          Math.min(position.maxX, params.x ?? position.x + (params.deltaX ?? 0))
        );
        const y = Math.max(
          0,
          Math.min(position.maxY, params.y ?? position.y + (params.deltaY ?? 0))
        );
        const behavior = getBehaviorFromContext(context);
        const style = params.behavior ?? behavior?.getScrollStyle() ?? 'instant';
        await applyPreActionDelay(behavior, (b) => b.getClickDelay() / 2);
        if (style === 'smooth') await executeSmoothScroll(target, x, y, behavior, timeout);
        else if (behavior && params.behavior === undefined)
          await executeHumanScroll(target, x, y, behavior);
        else await scrollTarget(target, x, y);
      } finally {
        await target.dispose();
      }

      logger.info('Scroll successful', {
        selector: params.selector,
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
