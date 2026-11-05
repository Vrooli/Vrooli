import { ConditionalType } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { errors } from 'rebrowser-playwright';
import { BaseHandler, getDocument, type BrowserDocument, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction } from '../types';
import { InvalidInstructionError, normalizeError } from '../utils';

/** Browser predicates; workflow variables remain with the executor's store. */
export class ConditionalHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['conditional'];
  }

  async execute(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    try {
      const action = instruction.action?.params;
      const params = this.requireTypedParams(action?.case === 'conditional' ? action.value : undefined,
        'conditional', instruction.nodeId);
      const page = getDocument(context);
      let actual: unknown;
      let outcome: boolean;
      let type: string;
      switch (params.conditionType) {
        case ConditionalType.EXPRESSION: {
          const expression = params.expression?.trim();
          if (!expression) throw new InvalidInstructionError('conditional expression is required');
          const result = await page.evaluate(async source => {
            // Only syntax compilation can select the body form. Never catch a
            // runtime exception and execute the expression a second time.
            let evaluate: () => unknown;
            try {
              evaluate = new Function(`return (${source}\n)`) as () => unknown;
            } catch (error) {
              if (!(error instanceof SyntaxError)) throw error;
              evaluate = new Function(source) as () => unknown;
            }
            const value = await evaluate();
            return { outcome: Boolean(value), actual: value };
          }, expression);
          ({ actual, outcome } = result);
          type = 'expression';
          break;
        }
        case ConditionalType.ELEMENT: {
          if (!params.selector?.trim()) throw new InvalidInstructionError('conditional selector is required');
          const timeout = params.timeoutMs ?? 10000;
          const polling = params.pollIntervalMs ?? 250;
          if (!Number.isInteger(timeout) || timeout < 0 || timeout > 120000 || !Number.isInteger(polling) || polling < 1 || polling > 5000) {
            throw new InvalidInstructionError('conditional timeout must be 0..120000 ms and poll interval 1..5000 ms');
          }
          actual = outcome = await this.elementExists(page, params.selector, timeout, polling);
          type = 'element_exists';
          break;
        }
        default:
          throw new InvalidInstructionError('conditional requires a browser expression or element predicate; variables belong to the workflow executor');
      }
      const negated = params.negate ?? false;
      return { success: true, condition: {
        type, outcome: outcome !== negated, negated, actual, expected: true,
        expression: params.expression, selector: params.selector,
      } };
    } catch (error) {
      const failure = normalizeError(error);
      return { success: false, error: {
        message: failure.message, code: failure.code, kind: failure.kind,
        // A page expression may commit an effect before throwing.
        retryable: false,
      } };
    }
  }

  private async elementExists(page: BrowserDocument, selector: string, timeout: number, polling: number): Promise<boolean> {
    if (timeout === 0) return await page.locator(selector).count() > 0;
    try {
      const result = await page.waitForFunction(value => document.querySelector(value) !== null, selector, { timeout, polling });
      await result.dispose();
      return true;
    } catch (error) {
      if (error instanceof errors.TimeoutError) return false;
      throw error;
    }
  }
}
