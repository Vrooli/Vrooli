import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { CompiledInstruction, AssertionOutcome } from '../types';
import { AssertParamsSchema } from '../types/instruction';
import { DEFAULT_ASSERTION_TIMEOUT_MS } from '../constants';
import { normalizeError } from '../utils';

/**
 * Assertion handler
 *
 * Handles assertion operations (exists, visible, text contains/equals, attribute)
 */
export class AssertionHandler extends BaseHandler {
  getSupportedTypes(): string[] {
    return ['assert'];
  }

  async execute(
    instruction: CompiledInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      // Validate parameters
      const params = AssertParamsSchema.parse(instruction.params);

      if (!params.selector) {
        return {
          success: false,
          error: {
            message: 'assert instruction missing selector parameter',
            code: 'MISSING_PARAM',
            kind: 'orchestration',
            retryable: false,
          },
        };
      }

      const timeout = params.timeoutMs || DEFAULT_ASSERTION_TIMEOUT_MS;

      // Determine assertion mode
      const mode = (
        params.mode ||
        params.kind ||
        (params.contains === false ? 'equals' : 'contains')
      ).toLowerCase();

      logger.debug('Running assertion', {
        selector: params.selector,
        mode,
        timeout,
      });

      let assertion: AssertionOutcome;

      switch (mode) {
        case 'exists':
          assertion = await this.assertExists(page, params.selector, timeout);
          break;

        case 'notexists':
        case 'not_exists':
        case 'absent':
          assertion = await this.assertNotExists(page, params.selector, timeout);
          break;

        case 'visible':
          assertion = await this.assertVisible(page, params.selector, timeout);
          break;

        case 'hidden':
        case 'notvisible':
        case 'not_visible':
          assertion = await this.assertHidden(page, params.selector, timeout);
          break;

        case 'attribute':
          if (!params.attribute && !params.attr) {
            return {
              success: false,
              error: {
                message: 'assert attribute mode missing attribute parameter',
                code: 'MISSING_PARAM',
                kind: 'orchestration',
                retryable: false,
              },
            };
          }
          assertion = await this.assertAttribute(
            page,
            params.selector,
            params.attribute || params.attr!,
            String(params.expected ?? params.value ?? ''),
            timeout
          );
          break;

        case 'equals':
        case 'contains':
        case 'matches':
        default:
          assertion = await this.assertText(
            page,
            params.selector,
            mode,
            String(params.expected ?? params.text ?? ''),
            timeout
          );
          break;
      }

      logger.info('Assertion complete', {
        selector: params.selector,
        mode,
        success: assertion.success,
      });

      return {
        success: assertion.success,
        extracted_data: {
          assertion,
        },
      };
    } catch (error) {
      logger.error('Assertion failed', {
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

  private async assertExists(
    page: any,
    selector: string,
    _timeout: number
  ): Promise<AssertionOutcome> {
    const element = await page.$(selector);
    const exists = !!element;

    return {
      mode: 'exists',
      selector,
      expected: 'exists',
      actual: exists ? 'exists' : 'missing',
      success: exists,
      message: exists ? '' : 'expected element to exist',
    };
  }

  private async assertNotExists(
    page: any,
    selector: string,
    _timeout: number
  ): Promise<AssertionOutcome> {
    const element = await page.$(selector);
    const absent = !element;

    return {
      mode: 'notexists',
      selector,
      expected: 'absent',
      actual: absent ? 'absent' : 'present',
      success: absent,
      message: absent ? '' : 'expected element to be absent',
    };
  }

  private async assertVisible(
    page: any,
    selector: string,
    timeout: number
  ): Promise<AssertionOutcome> {
    const visible = await page.isVisible(selector, { timeout }).catch(() => false);

    return {
      mode: 'visible',
      selector,
      expected: 'visible',
      actual: visible ? 'visible' : 'not visible',
      success: visible,
      message: visible ? '' : 'expected element to be visible',
    };
  }

  private async assertHidden(
    page: any,
    selector: string,
    timeout: number
  ): Promise<AssertionOutcome> {
    const visible = await page.isVisible(selector, { timeout }).catch(() => false);
    const hidden = !visible;

    return {
      mode: 'hidden',
      selector,
      expected: 'hidden',
      actual: hidden ? 'hidden' : 'visible',
      success: hidden,
      message: hidden ? '' : 'expected element to be hidden',
    };
  }

  private async assertAttribute(
    page: any,
    selector: string,
    attribute: string,
    expected: string,
    timeout: number
  ): Promise<AssertionOutcome> {
    const actual = (await page.getAttribute(selector, attribute, { timeout })) || '';
    const success = actual === expected;

    return {
      mode: 'attribute',
      selector,
      expected,
      actual,
      success,
      message: success ? '' : `expected attribute ${attribute} "${actual}" to equal "${expected}"`,
    };
  }

  private async assertText(
    page: any,
    selector: string,
    mode: string,
    expected: string,
    timeout: number
  ): Promise<AssertionOutcome> {
    const actual = (await page.textContent(selector, { timeout })) || '';

    let success = false;
    let message = '';

    if (mode === 'matches') {
      const regex = new RegExp(expected);
      success = regex.test(actual);
      message = success ? '' : `expected "${actual}" to match /${expected}/`;
    } else if (mode === 'contains') {
      success = actual.includes(expected);
      message = success ? '' : `expected "${actual}" to contain "${expected}"`;
    } else {
      // equals
      success = actual === expected;
      message = success ? '' : `expected "${actual}" to equal "${expected}"`;
    }

    return {
      mode,
      selector,
      expected,
      actual,
      success,
      message,
    };
  }
}
