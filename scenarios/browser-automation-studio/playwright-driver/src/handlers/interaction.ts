import type { Page } from 'playwright';
import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import {
  getClickParams,
  getHoverParams,
  getInputParams,
  getFocusParams,
  getBlurParams,
} from '../types';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError } from '../utils';
import { captureElementContext, type ElementContext } from '../telemetry';
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

/** Resolve timeout from params or config */
function resolveTimeout(paramsTimeout: number | undefined, configTimeout: number | undefined): number {
  return paramsTimeout || configTimeout || DEFAULT_TIMEOUT_MS;
}

// =============================================================================
// InteractionHandler
// =============================================================================

/**
 * Interaction handler for user interaction operations: click, hover, type, focus, blur
 */
export class InteractionHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['click', 'hover', 'type', 'focus', 'blur'];
  }

  async execute(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    try {
      switch (instruction.type.toLowerCase()) {
        case 'click': return await this.handleClick(instruction, context);
        case 'hover': return await this.handleHover(instruction, context);
        case 'type': return await this.handleType(instruction, context);
        case 'focus': return await this.handleFocus(instruction, context);
        case 'blur': return await this.handleBlur(instruction, context);
        default:
          return { success: false, error: { message: `Unsupported interaction type: ${instruction.type}`, code: 'UNSUPPORTED_TYPE', kind: 'orchestration', retryable: false } };
      }
    } catch (error) {
      return this.handleError(error, instruction, context.logger);
    }
  }

  private handleError(error: unknown, instruction: HandlerInstruction, logger: winston.Logger): HandlerResult {
    const driverError = normalizeError(error);
    logger.warn('instruction: interaction failed', {
      type: instruction.type,
      selector: (instruction.params as Record<string, unknown>).selector,
      errorCode: driverError.code,
      errorMessage: driverError.message,
      retryable: driverError.retryable,
    });
    return { success: false, error: { message: driverError.message, code: driverError.code, kind: driverError.kind, retryable: driverError.retryable } };
  }

  private async handleClick(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    const { page, logger } = context;

    // Extract typed params from action
    const typedParams = instruction.action ? getClickParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'click', instruction.nodeId);

    if (!params.selector) return missingSelectorError('click');

    const timeout = resolveTimeout(params.timeoutMs, context.config.execution.defaultTimeoutMs);
    logger.debug('instruction: click starting', { selector: params.selector, timeout });

    // Capture element context BEFORE the action (recording-quality telemetry)
    const elementContext = await captureElementContext(page, params.selector, { timeout });
    await page.click(params.selector, { timeout });

    logger.debug('instruction: click completed', { selector: params.selector });
    return successWithElementContext(elementContext);
  }

  private async handleHover(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    const { page, logger } = context;

    // Extract typed params from action
    const typedParams = instruction.action ? getHoverParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'hover', instruction.nodeId);

    if (!params.selector) return missingSelectorError('hover');

    const timeout = resolveTimeout(params.timeoutMs, context.config.execution.defaultTimeoutMs);
    logger.debug('instruction: hover starting', { selector: params.selector, timeout });

    // Capture element context BEFORE the action (recording-quality telemetry)
    const elementContext = await captureElementContext(page, params.selector, { timeout });
    await page.hover(params.selector, { timeout });

    logger.debug('instruction: hover completed', { selector: params.selector });
    return successWithElementContext(elementContext);
  }

  private async handleType(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    const { page, logger } = context;

    // Extract typed params from action
    const typedParams = instruction.action ? getInputParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'type', instruction.nodeId);

    if (!params.selector) return missingSelectorError('type');

    const timeout = resolveTimeout(params.timeoutMs, context.config.execution.defaultTimeoutMs);
    logger.debug('instruction: type starting', { selector: params.selector, textLength: params.value.length, timeout });

    // Capture element context BEFORE the action (recording-quality telemetry)
    const elementContext = await captureElementContext(page, params.selector, { timeout });
    await page.fill(params.selector, params.value, { timeout });

    logger.debug('instruction: type completed', { selector: params.selector, textLength: params.value.length });
    return successWithElementContext(elementContext);
  }

  private async handleFocus(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    const { page, logger } = context;

    // Extract typed params from action
    const typedParams = instruction.action ? getFocusParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'focus', instruction.nodeId);

    if (!params.selector) return missingSelectorError('focus');

    const timeout = resolveTimeout(params.timeoutMs, context.config.execution.defaultTimeoutMs);
    logger.debug('instruction: focus starting', { selector: params.selector, timeout });

    // Capture element context BEFORE the action (recording-quality telemetry)
    const elementContext = await captureElementContext(page, params.selector, { timeout });
    await page.focus(params.selector, { timeout });

    logger.debug('instruction: focus completed', { selector: params.selector });
    return successWithElementContext(elementContext);
  }

  private async handleBlur(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    const { page, logger } = context;

    // Extract typed params from action
    const typedParams = instruction.action ? getBlurParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'blur', instruction.nodeId);

    const selectorForLog = params.selector || '(active element)';
    logger.debug('instruction: blur starting', { selector: selectorForLog });
    await blurElement(page, params.selector);
    logger.debug('instruction: blur completed', { selector: selectorForLog });

    return { success: true };
  }
}

/** Blur an element or the active element if no selector provided */
async function blurElement(page: Page, selector?: string): Promise<void> {
  if (selector) {
    // Playwright doesn't have a direct blur method, so we use evaluate
    await page.evaluate((sel) => {
      // @ts-expect-error - document is available in browser context
      const element = document.querySelector(sel);
      // @ts-expect-error - HTMLElement is available in browser context
      if (element && element instanceof HTMLElement) element.blur();
    }, selector);
  } else {
    await page.evaluate(() => {
      // @ts-expect-error - document/HTMLElement available in browser context
      if (document.activeElement && document.activeElement instanceof HTMLElement) {
        // @ts-expect-error - document available in browser context
        document.activeElement.blur();
      }
    });
  }
}
