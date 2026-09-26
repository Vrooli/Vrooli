import { errors } from 'rebrowser-playwright';
import { BaseHandler, getDocument, type BrowserDocument, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction, AssertionOutcome } from '../types';
import { getAssertParams } from '../types';
import { DEFAULT_ASSERTION_TIMEOUT_MS } from '../constants';
import { InvalidInstructionError, normalizeError } from '../utils';
import { captureElementContext } from '../telemetry';

type ElementState = 'attached' | 'detached' | 'visible' | 'hidden';
const stateAssertions: Record<string, { state: ElementState; inverse: ElementState; expected: string; opposite: string }> = {
  exists: { state: 'attached', inverse: 'detached', expected: 'present', opposite: 'absent' },
  notexists: { state: 'detached', inverse: 'attached', expected: 'absent', opposite: 'present' },
  visible: { state: 'visible', inverse: 'hidden', expected: 'visible', opposite: 'hidden' },
  hidden: { state: 'hidden', inverse: 'visible', expected: 'hidden', opposite: 'visible' },
};
const comparisonModes = new Set(['text_equals', 'text_contains', 'attribute_equals', 'attribute_contains']);

/** Interprets the eight DOM predicates in the typed AssertParams contract. */
export class AssertionHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['assert'];
  }

  async execute(instruction: HandlerInstruction, context: HandlerContext): Promise<HandlerResult> {
    try {
      const page = getDocument(context);
      const params = this.requireTypedParams(
        instruction.action ? getAssertParams(instruction.action) : undefined, 'assert', instruction.nodeId
      );
      const { selector, mode, attributeName } = params;
      const attribute = mode.startsWith('attribute_');
      if (!selector || (attribute && !attributeName)) {
        return {
          success: false,
          error: {
            message: !selector ? 'assert instruction missing selector parameter' : 'assert attribute mode missing attribute parameter',
            code: 'MISSING_PARAM', kind: 'orchestration', retryable: false,
          },
        };
      }
      if (!stateAssertions[mode] && !comparisonModes.has(mode)) {
        throw new InvalidInstructionError(`Unsupported assertion mode: ${mode}`, { nodeId: instruction.nodeId });
      }
      const timeout = params.timeoutMs ?? context.config.execution.assertionTimeoutMs ?? DEFAULT_ASSERTION_TIMEOUT_MS;
      const negated = params.negated ?? false;
      const caseSensitive = params.caseSensitive ?? true;
      // Browser timeout zero disables its deadline. Our zero means observe now.
      const elementContext = await captureElementContext(page, selector, { timeout: timeout || 1 });
      const state = stateAssertions[mode];
      let assertion: AssertionOutcome;
      if (state) {
        const success = await this.waitForState(page, selector, negated ? state.inverse : state.state, timeout);
        assertion = {
          expected: state.expected,
          actual: success !== negated ? state.expected : state.opposite,
          success,
          message: `expected element to be ${negated ? 'not ' : ''}${state.expected}`,
        };
      } else {
        assertion = await this.compareValue(page, selector, mode, String(params.expected ?? ''),
          attributeName, timeout, negated, caseSensitive);
      }
      assertion = {
        ...assertion, selector, mode, negated, caseSensitive,
        message: assertion.success ? '' : (params.failureMessage || assertion.message),
      };
      context.logger.info('Assertion complete', { selector, mode, success: assertion.success });
      return {
        success: assertion.success,
        elementContext,
        extracted_data: { assertion },
        focus: elementContext.boundingBox ? {
          selector: elementContext.selector,
          bounding_box: {
            x: elementContext.boundingBox.x, y: elementContext.boundingBox.y,
            width: elementContext.boundingBox.width, height: elementContext.boundingBox.height,
          },
        } : undefined,
      };
    } catch (error) {
      const driverError = normalizeError(error);
      context.logger.error('Assertion evaluation failed', { error: driverError.message });
      return {
        success: false,
        error: { message: driverError.message, code: driverError.code, kind: driverError.kind, retryable: driverError.retryable },
      };
    }
  }

  private async waitForState(page: BrowserDocument, selector: string, state: ElementState, timeout: number): Promise<boolean> {
    const locator = page.locator(selector).first();
    if (timeout === 0) {
      if (state === 'attached' || state === 'detached') {
        const exists = await locator.count() > 0;
        return state === 'attached' ? exists : !exists;
      }
      const visible = await locator.isVisible();
      return state === 'visible' ? visible : !visible;
    }
    try {
      await locator.waitFor({ state, timeout });
      return true;
    } catch (error) {
      if (error instanceof errors.TimeoutError) return false;
      throw error;
    }
  }

  private async compareValue(page: BrowserDocument, selector: string, mode: string, expected: string,
    attributeName: string | undefined, timeout: number, negated: boolean, caseSensitive: boolean): Promise<AssertionOutcome> {
    const attribute = mode.startsWith('attribute_');
    const actual = attribute
      ? await page.getAttribute(selector, attributeName!, { timeout: timeout || 1 })
      : (await page.textContent(selector, { timeout: timeout || 1 })) ?? '';
    const comparableActual = caseSensitive ? actual : actual?.toLowerCase();
    const comparableExpected = caseSensitive ? expected : expected.toLowerCase();
    const contains = mode.endsWith('_contains');
    const matches = comparableActual != null && (contains
      ? comparableActual.includes(comparableExpected) : comparableActual === comparableExpected);
    return {
      expected, actual: actual ?? '(missing attribute)', success: matches !== negated,
      message: `expected "${actual ?? '(missing attribute)'}" ${negated ? 'not ' : ''}to ${contains ? 'contain' : 'equal'} "${expected}"`,
    };
  }
}
