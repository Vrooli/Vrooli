import { BaseHandler, getDocument, type BrowserDocument, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import {
  getClickParams,
  getHoverParams,
  getInputParams,
  getFocusParams,
  getBlurParams,
} from '../types';
import { normalizeError } from '../utils';
import { getActionType } from '../proto';
import { captureElementContext, type ElementContext } from '../telemetry';
import {
  getBehaviorFromContext,
  applyPreActionDelay,
  applyPostActionPause,
  moveMouseNaturally,
  getElementCenter,
  resolveTimeoutFromContext,
  sleep,
} from './behavior-utils';
import type winston from 'winston';

// =============================================================================
// Shared Helpers
// =============================================================================

/** Create a missing selector error result */
function missingSelectorError(instructionType: string): HandlerResult {
  return {
    success: false,
    error: {
      message: `${instructionType} instruction missing selector parameter`,
      code: 'MISSING_PARAM',
      kind: 'orchestration',
      retryable: false,
    },
  };
}

/** Create a success result with element context */
function successWithElementContext(elementContext: ElementContext): HandlerResult {
  return {
    success: true,
    elementContext,
    // Also set focus for backward compatibility
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
}

// =============================================================================
// InteractionHandler
// =============================================================================

/**
 * Interaction handler for user interaction operations: click, hover, type, focus, blur
 */
export class InteractionHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    // `input` is the canonical wire name emitted by typed BAS actions while
    // `type` remains the ergonomic authoring alias. Both reach handleType.
    return ['click', 'hover', 'type', 'input', 'focus', 'blur'];
  }

  async execute(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    try {
		const actionType = getActionType(instruction);
		switch (actionType.toLowerCase()) {
        case 'click': return await this.handleClick(instruction, context);
        case 'hover': return await this.handleHover(instruction, context);
        case 'type':
        case 'input':
          return await this.handleType(instruction, context);
        case 'focus': return await this.handleFocus(instruction, context);
        case 'blur': return await this.handleBlur(instruction, context);
        default:
			return { success: false, error: { message: `Unsupported interaction type: ${actionType}`, code: 'UNSUPPORTED_TYPE', kind: 'orchestration', retryable: false } };
      }
    } catch (error) {
      return this.handleError(error, instruction, context.logger);
    }
  }

  private handleError(error: unknown, instruction: HandlerInstruction, logger: winston.Logger): HandlerResult {
    const driverError = normalizeError(error);
    logger.warn('instruction: interaction failed', {
		type: getActionType(instruction),
      errorCode: driverError.code,
      errorMessage: driverError.message,
      retryable: driverError.retryable,
    });
    return { success: false, error: { message: driverError.message, code: driverError.code, kind: driverError.kind, retryable: driverError.retryable } };
  }

  private async handleClick(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    const { page, logger } = context;
    const target = getDocument(context);

    // Extract typed params from action
    const typedParams = instruction.action ? getClickParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'click', instruction.nodeId);

    if (!params.selector) return missingSelectorError('click');

    // DECISION: Use 'default' category for click operations
    const timeout = resolveTimeoutFromContext(params.timeoutMs, context, 'default');
    const behavior = getBehaviorFromContext(context);

    logger.debug('instruction: click starting', {
      selector: params.selector,
      timeout,
      humanBehavior: !!behavior,
    });

    // Capture element context BEFORE the action (recording-quality telemetry)
    const elementContext = await captureElementContext(target, params.selector, { timeout });

    // Apply human-like behavior if enabled
    if (behavior) {
      // Apply pre-click delay
      await applyPreActionDelay(behavior, (b) => b.getClickDelay());

      // Move mouse naturally to element if using bezier/natural movement
      if (behavior.getMouseMovementStyle() !== 'linear') {
        const center = await getElementCenter(target, params.selector, timeout);
        if (center) {
          await moveMouseNaturally(page, center.x, center.y, behavior);
        }
      }
    }

    await target.click(params.selector, {
      timeout, button: params.button, clickCount: params.clickCount,
      delay: params.delayMs, modifiers: params.modifiers, force: params.force,
    });

    logger.debug('instruction: click completed', { selector: params.selector });
    return successWithElementContext(elementContext);
  }

  private async handleHover(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    const { page, logger } = context;
    const target = getDocument(context);

    // Extract typed params from action
    const typedParams = instruction.action ? getHoverParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'hover', instruction.nodeId);

    if (!params.selector) return missingSelectorError('hover');

    // DECISION: Use 'default' category for hover operations
    const timeout = resolveTimeoutFromContext(params.timeoutMs, context, 'default');
    const behavior = getBehaviorFromContext(context);

    logger.debug('instruction: hover starting', {
      selector: params.selector,
      timeout,
      humanBehavior: !!behavior,
    });

    // Capture element context BEFORE the action (recording-quality telemetry)
    const elementContext = await captureElementContext(target, params.selector, { timeout });

    // Apply human-like behavior if enabled
    if (behavior) {
      // Apply pre-hover delay (use click delay as hover delay)
      await applyPreActionDelay(behavior, (b) => b.getClickDelay());

      // Move mouse naturally to element if using bezier/natural movement
      if (behavior.getMouseMovementStyle() !== 'linear') {
        const center = await getElementCenter(target, params.selector, timeout);
        if (center) {
          await moveMouseNaturally(page, center.x, center.y, behavior);
        }
      }
    }

    await target.hover(params.selector, { timeout });

    // Apply post-hover micro-pause
    await applyPostActionPause(behavior);

    logger.debug('instruction: hover completed', { selector: params.selector });
    return successWithElementContext(elementContext);
  }

  private async handleType(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    const { page, logger } = context;
    const target = getDocument(context);

    // Extract typed params from action
    const typedParams = instruction.action ? getInputParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'type', instruction.nodeId);

    if (!params.selector) return missingSelectorError('type');

    // DECISION: Use 'default' category for type operations
    const timeout = resolveTimeoutFromContext(params.timeoutMs, context, 'default');
    const behavior = getBehaviorFromContext(context);

    logger.debug('instruction: type starting', {
      selector: params.selector,
      textLength: params.value.length,
      timeout,
      humanBehavior: !!behavior,
    });

    // Capture element context BEFORE the action (recording-quality telemetry)
    const elementContext = await captureElementContext(target, params.selector, { timeout });

    const clearFirst = params.clearFirst !== false;
    const typingDelay = params.delayMs ?? behavior?.getTypingDelay() ?? 0;
    if (typingDelay > 0 || !clearFirst) {
      if (behavior && behavior.getMouseMovementStyle() !== 'linear') {
        const center = await getElementCenter(target, params.selector, timeout);
        if (center) await moveMouseNaturally(page, center.x, center.y, behavior);
      }
      if (clearFirst) await target.fill(params.selector, '', { timeout });
      await target.focus(params.selector, { timeout });
      // Explicit append starts after the current value, regardless of selection.
      if (!clearFirst) {
        const positioned = await target.locator(params.selector).evaluate((element) => {
          const input = element as HTMLInputElement | HTMLTextAreaElement;
          if (typeof input.selectionStart !== 'number') return false;
          input.setSelectionRange(input.value.length, input.value.length);
          return true;
        });
        if (!positioned) await target.press(params.selector, 'End', { timeout });
      }
      for (const char of params.value) {
        await page.keyboard.type(char);
        const delay = params.delayMs ?? behavior?.getTypingDelay() ?? 0;
        if (delay > 0) await sleep(delay);
        if (behavior?.shouldMicroPause()) await sleep(behavior.getMicroPauseDuration());
      }
    } else {
      await target.fill(params.selector, params.value, { timeout });
    }
    if (params.submit) await target.press(params.selector, 'Enter', { timeout });

    logger.debug('instruction: type completed', { selector: params.selector, textLength: params.value.length });
    return successWithElementContext(elementContext);
  }

  private async handleFocus(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    const { logger } = context;
    const target = getDocument(context);

    // Extract typed params from action
    const typedParams = instruction.action ? getFocusParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'focus', instruction.nodeId);

    if (!params.selector) return missingSelectorError('focus');

    // DECISION: Use 'default' category for focus operations
    const timeout = resolveTimeoutFromContext(params.timeoutMs, context, 'default');
    logger.debug('instruction: focus starting', { selector: params.selector, timeout });

    // Capture element context BEFORE the action (recording-quality telemetry)
    const elementContext = await captureElementContext(target, params.selector, { timeout });
    await target.focus(params.selector, { timeout });

    logger.debug('instruction: focus completed', { selector: params.selector });
    return successWithElementContext(elementContext);
  }

  private async handleBlur(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    const { logger } = context;
    const target = getDocument(context);

    // Extract typed params from action
    const typedParams = instruction.action ? getBlurParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'blur', instruction.nodeId);

    const selectorForLog = params.selector || '(active element)';
    logger.debug('instruction: blur starting', { selector: selectorForLog });
    await blurElement(target, params.selector);
    logger.debug('instruction: blur completed', { selector: selectorForLog });

    return { success: true };
  }
}

/** Blur an element or the active element if no selector provided */
async function blurElement(target: BrowserDocument, selector?: string): Promise<void> {
  if (selector) {
    // Playwright doesn't have a direct blur method, so we use evaluate
    await target.evaluate((sel) => {
      const element = document.querySelector(sel);
      if (element && element instanceof HTMLElement) element.blur();
    }, selector);
  } else {
    await target.evaluate(() => {
      if (document.activeElement && document.activeElement instanceof HTMLElement) {
        document.activeElement.blur();
      }
    });
  }
}
