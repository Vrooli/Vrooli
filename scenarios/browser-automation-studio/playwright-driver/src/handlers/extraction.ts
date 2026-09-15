import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { getExtractParams, getEvaluateParams } from '../types';
import { getActionType } from '../proto';
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
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { logger } = context;

    try {
		const actionType = getActionType(instruction);
		switch (actionType.toLowerCase()) {
        case 'extract':
          return await this.handleExtract(instruction, context);

        case 'evaluate':
          return await this.handleEvaluate(instruction, context);

        default:
          return {
            success: false,
            error: {
				message: `Unsupported extraction type: ${actionType}`,
              code: 'UNSUPPORTED_TYPE',
              kind: 'orchestration',
              retryable: false,
            },
          };
      }
    } catch (error) {
      logger.error('Extraction failed', {
			type: getActionType(instruction),
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
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Get typed params from instruction.action (required after migration)
    const typedParams = instruction.action ? getExtractParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'extract', instruction.nodeId);

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

  /**
   * Run a script, tolerating a navigation that lands mid-evaluate.
   *
   * Playwright tears down the JS execution context when the page navigates, so
   * a script that started just before a navigation commits dies with
   * "Execution context was destroyed". That is transient by definition — the
   * next context can answer the same question — but it surfaced as a hard
   * failure because the executor defaults to MaxAttempts=1, which makes the
   * driver's `retryable` classification inert unless a workflow opts into a
   * resilience block per node.
   *
   * Retrying once here fixes the whole class without loosening retry semantics
   * for genuine failures. A second destruction is not swallowed: the workflow
   * is then navigating continuously and the caller should see it.
   */
  private async evaluateSurvivingNavigation(
    page: HandlerContext['page'],
    script: string,
    logger: HandlerContext['logger']
  ): Promise<unknown> {
    try {
      return await page.evaluate(script);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      if (!/execution context was destroyed/i.test(message)) {
        throw error;
      }
      logger.debug('evaluate: context destroyed by navigation, retrying once', {
        scriptLength: script.length,
      });
      // Let the new document commit before asking it anything.
      await page.waitForLoadState('domcontentloaded').catch(() => undefined);
      return await page.evaluate(script);
    }
  }

  private async handleEvaluate(
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    // Get typed params from instruction.action (required after migration)
    const typedParams = instruction.action ? getEvaluateParams(instruction.action) : undefined;
    const params = this.requireTypedParams(typedParams, 'evaluate', instruction.nodeId);

    const script = params.expression;
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
    // Note: EvaluateParams from proto doesn't support args - evaluate expression directly
    const result = await this.evaluateSurvivingNavigation(page, script, logger);

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
