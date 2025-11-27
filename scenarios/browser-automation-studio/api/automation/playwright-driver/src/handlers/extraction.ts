import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction } from '../types';
import { ExtractParamsSchema, EvaluateParamsSchema } from '../types/instruction';
import { DEFAULT_TIMEOUT_MS } from '../constants';
import { normalizeError } from '../utils';

/**
 * Extraction handler
 *
 * Handles data extraction operations: extract text, evaluate script
 */
export class ExtractionHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['extract', 'evaluate'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      switch (instruction.type.toLowerCase()) {
        case 'extract':
          return await this.handleExtract(instruction, context);

        case 'evaluate':
          return await this.handleEvaluate(instruction, context);

        default:
          return {
            success: false,
            error: {
              message: `Unsupported extraction type: ${instruction.type}`,
              code: 'UNSUPPORTED_TYPE',
              kind: 'orchestration',
              retryable: false,
            },
          };
      }
    } catch (error) {
      logger.error('Extraction failed', {
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

  private async handleExtract(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Validate parameters
    const params = ExtractParamsSchema.parse(instruction.params);

    if (!params.selector) {
      return {
        success: false,
        error: {
          message: 'extract instruction missing selector parameter',
          code: 'MISSING_PARAM',
          kind: 'orchestration',
          retryable: false,
        },
      };
    }

    const timeout = params.timeoutMs || DEFAULT_TIMEOUT_MS;

    logger.debug('Extracting text', {
      selector: params.selector,
      timeout,
    });

    // Extract text content
    const text = (await page.textContent(params.selector, { timeout })) || '';

    logger.info('Text extraction successful', {
      selector: params.selector,
      textLength: text.length,
    });

    return {
      success: true,
      extracted_data: {
        [params.selector]: text,
      },
    };
  }

  private async handleEvaluate(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Validate parameters
    const params = EvaluateParamsSchema.parse(instruction.params);

    const script = params.script || params.expression;
    if (!script) {
      return {
        success: false,
        error: {
          message: 'evaluate instruction missing script/expression parameter',
          code: 'MISSING_PARAM',
          kind: 'orchestration',
          retryable: false,
        },
      };
    }

    logger.debug('Evaluating script', {
      scriptLength: script.length,
    });

    // Evaluate script in browser context
    const result = await page.evaluate(script, params.args || {});

    logger.info('Script evaluation successful', {
      resultType: typeof result,
    });

    return {
      success: true,
      extracted_data: {
        result,
      },
    };
  }
}
