import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { ClickParamsSchema, HoverParamsSchema, TypeParamsSchema } from '../types/instruction';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError } from '../utils';

/**
 * Interaction handler
 *
 * Handles user interaction operations: click, hover, type
 */
export class InteractionHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['click', 'hover', 'type'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      switch (instruction.type.toLowerCase()) {
        case 'click':
          return await this.handleClick(instruction, context);

        case 'hover':
          return await this.handleHover(instruction, context);

        case 'type':
          return await this.handleType(instruction, context);

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
      logger.error('Interaction failed', {
        type: instruction.type,
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

    logger.debug('Clicking element', {
      selector: params.selector,
      timeout,
    });

    // Get bounding box before click
    const boundingBox = await this.getBoundingBox(page, params.selector).catch(() => null);

    // Click element
    await page.click(params.selector, { timeout });

    logger.info('Click successful', {
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

    logger.debug('Hovering element', {
      selector: params.selector,
      timeout,
    });

    // Get bounding box before hover
    const boundingBox = await this.getBoundingBox(page, params.selector).catch(() => null);

    // Hover element
    await page.hover(params.selector, { timeout });

    logger.info('Hover successful', {
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

    logger.debug('Typing text', {
      selector: params.selector,
      textLength: text.length,
      timeout,
    });

    // Get bounding box before typing
    const boundingBox = await this.getBoundingBox(page, params.selector).catch(() => null);

    // Fill element with text
    await page.fill(params.selector, text, { timeout });

    logger.info('Type successful', {
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
}
