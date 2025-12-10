import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import {
  ClickParamsSchema,
  HoverParamsSchema,
  TypeParamsSchema,
  FocusParamsSchema,
  BlurParamsSchema,
} from '../types/instruction';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError } from '../utils';

/**
 * Interaction handler
 *
 * Handles user interaction operations: click, hover, type, focus, blur
 */
export class InteractionHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['click', 'hover', 'type', 'focus', 'blur'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { logger } = context;

    try {
      switch (instruction.type.toLowerCase()) {
        case 'click':
          return await this.handleClick(instruction, context);

        case 'hover':
          return await this.handleHover(instruction, context);

        case 'type':
          return await this.handleType(instruction, context);

        case 'focus':
          return await this.handleFocus(instruction, context);

        case 'blur':
          return await this.handleBlur(instruction, context);

        default:
          return {
            success: false,
            error: {
              message: `Unsupported interaction type: ${instruction.type}`,
              code: 'UNSUPPORTED_TYPE',
              kind: 'orchestration',
              retryable: false,
            },
          };
      }
    } catch (error) {
      const driverError = normalizeError(error);
      logger.warn('instruction: interaction failed', {
        type: instruction.type,
        selector: (instruction.params as Record<string, unknown>).selector,
        errorCode: driverError.code,
        errorMessage: driverError.message,
        retryable: driverError.retryable,
      });

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

  private async handleClick(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Validate parameters
    const params = ClickParamsSchema.parse(instruction.params);

    if (!params.selector) {
      return {
        success: false,
        error: {
          message: 'click instruction missing selector parameter',
          code: 'MISSING_PARAM',
          kind: 'orchestration',
          retryable: false,
        },
      };
    }

    const timeout = params.timeoutMs || DEFAULT_TIMEOUT_MS;

    logger.debug('instruction: click starting', {
      selector: params.selector,
      timeout,
    });

    // Get bounding box before click
    const boundingBox = await this.getBoundingBox(page, params.selector).catch(() => null);

    // Click element
    await page.click(params.selector, { timeout });

    logger.debug('instruction: click completed', {
      selector: params.selector,
      hasBoundingBox: !!boundingBox,
    });

    return {
      success: true,
      focus: {
        selector: params.selector,
        bounding_box: boundingBox || undefined,
      },
    };
  }

  private async handleHover(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Validate parameters
    const params = HoverParamsSchema.parse(instruction.params);

    if (!params.selector) {
      return {
        success: false,
        error: {
          message: 'hover instruction missing selector parameter',
          code: 'MISSING_PARAM',
          kind: 'orchestration',
          retryable: false,
        },
      };
    }

    const timeout = params.timeoutMs || DEFAULT_TIMEOUT_MS;

    logger.debug('instruction: hover starting', {
      selector: params.selector,
      timeout,
    });

    // Get bounding box before hover
    const boundingBox = await this.getBoundingBox(page, params.selector).catch(() => null);

    // Hover element
    await page.hover(params.selector, { timeout });

    logger.debug('instruction: hover completed', {
      selector: params.selector,
    });

    return {
      success: true,
      focus: {
        selector: params.selector,
        bounding_box: boundingBox || undefined,
      },
    };
  }

  private async handleType(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Validate parameters
    const params = TypeParamsSchema.parse(instruction.params);

    if (!params.selector) {
      return {
        success: false,
        error: {
          message: 'type instruction missing selector parameter',
          code: 'MISSING_PARAM',
          kind: 'orchestration',
          retryable: false,
        },
      };
    }

    const text = params.text || params.value || '';
    const timeout = params.timeoutMs || DEFAULT_TIMEOUT_MS;

    logger.debug('instruction: type starting', {
      selector: params.selector,
      textLength: text.length,
      timeout,
    });

    // Get bounding box before typing
    const boundingBox = await this.getBoundingBox(page, params.selector).catch(() => null);

    // Fill element with text
    await page.fill(params.selector, text, { timeout });

    logger.debug('instruction: type completed', {
      selector: params.selector,
      textLength: text.length,
    });

    return {
      success: true,
      focus: {
        selector: params.selector,
        bounding_box: boundingBox || undefined,
      },
    };
  }

  private async handleFocus(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Validate parameters
    const params = FocusParamsSchema.parse(instruction.params);

    if (!params.selector) {
      return {
        success: false,
        error: {
          message: 'focus instruction missing selector parameter',
          code: 'MISSING_PARAM',
          kind: 'orchestration',
          retryable: false,
        },
      };
    }

    const timeout = params.timeoutMs || DEFAULT_TIMEOUT_MS;

    logger.debug('instruction: focus starting', {
      selector: params.selector,
      timeout,
    });

    // Get bounding box before focus
    const boundingBox = await this.getBoundingBox(page, params.selector).catch(() => null);

    // Focus element
    await page.focus(params.selector, { timeout });

    logger.debug('instruction: focus completed', {
      selector: params.selector,
    });

    return {
      success: true,
      focus: {
        selector: params.selector,
        bounding_box: boundingBox || undefined,
      },
    };
  }

  private async handleBlur(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Validate parameters
    const params = BlurParamsSchema.parse(instruction.params);

    logger.debug('instruction: blur starting', {
      selector: params.selector || '(active element)',
    });

    if (params.selector) {
      // Blur specific element by focusing on body
      // (Playwright doesn't have a direct blur method)
      await page.evaluate((sel) => {
        // @ts-expect-error - document is available in browser context
        const element = document.querySelector(sel);
        // @ts-expect-error - HTMLElement is available in browser context
        if (element && element instanceof HTMLElement) {
          element.blur();
        }
      }, params.selector);

      logger.debug('instruction: blur completed', {
        selector: params.selector,
      });
    } else {
      // Blur active element
      await page.evaluate(() => {
        // @ts-expect-error - activeElement is available
        if (document.activeElement && document.activeElement instanceof HTMLElement) {
          // @ts-expect-error - activeElement is available
          document.activeElement.blur();
        }
      });

      logger.debug('instruction: blur completed', {
        selector: '(active element)',
      });
    }

    return {
      success: true,
    };
  }
}
