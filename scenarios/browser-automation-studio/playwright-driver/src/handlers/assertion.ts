import type { Page } from 'playwright';
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
          assertion = await this.assertNotExists(page, params.selector, timeout, logger);
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
        case 'attribute_equals':
        case 'attribute_contains':
          if (!params.attribute && !params.attr && !params.attributeName) {
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
          // Normalize attribute mode for assertAttribute method
          const attrMode = mode.replace('attribute_', '');
          assertion = await this.assertAttribute(
            page,
            params.selector,
            params.attribute || params.attr || params.attributeName!,
            String(params.expected ?? params.expectedValue ?? params.value ?? ''),
            timeout,
            attrMode
          );
          break;

        case 'text_equals':
        case 'equals':
          assertion = await this.assertText(
            page,
            params.selector,
            'equals',
            String(params.expected ?? params.expectedValue ?? params.text ?? ''),
            timeout
          );
          break;

        case 'text_contains':
        case 'contains':
          assertion = await this.assertText(
            page,
            params.selector,
            'contains',
            String(params.expected ?? params.expectedValue ?? params.text ?? ''),
            timeout
          );
          break;

        case 'matches':
        default:
          assertion = await this.assertText(
            page,
            params.selector,
            mode,
            String(params.expected ?? params.expectedValue ?? params.text ?? ''),
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
    page: Page,
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
    page: Page,
    selector: string,
    timeout: number,
    logger: HandlerContext['logger']
  ): Promise<AssertionOutcome> {
    try {
      logger.debug('assertNotExists: Starting assertion', { selector, timeout });

      // First check if element exists at all
      const initialElement = await page.$(selector);
      logger.debug('assertNotExists: Initial element query', { found: !!initialElement });

      if (!initialElement) {
        // Element already doesn't exist - success
        logger.debug('assertNotExists: Element absent from start - SUCCESS');
        return {
          mode: 'notexists',
          selector,
          expected: 'absent',
          actual: 'absent',
          success: true,
          message: '',
        };
      }

      // Element exists, wait for it to be removed using Playwright's locator API
      // This properly waits for DOM mutations
      logger.debug('assertNotExists: Element present, waiting for detachment');
      const locator = page.locator(selector);
      await locator.waitFor({ state: 'detached', timeout });

      // Element was successfully removed
      logger.debug('assertNotExists: Element successfully detached - SUCCESS');
      return {
        mode: 'notexists',
        selector,
        expected: 'absent',
        actual: 'absent',
        success: true,
        message: '',
      };
    } catch (error) {
      // Timeout - element still present
      logger.debug('assertNotExists: Timeout waiting for detachment - FAILURE', {
        error: error instanceof Error ? error.message : String(error),
      });
      return {
        mode: 'notexists',
        selector,
        expected: 'absent',
        actual: 'present',
        success: false,
        message: 'expected element to be absent',
      };
    }
  }

  private async assertVisible(
    page: Page,
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
    page: Page,
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
    page: Page,
    selector: string,
    attribute: string,
    expected: string,
    timeout: number,
    mode: string = 'equals'
  ): Promise<AssertionOutcome> {
    const actual = (await page.getAttribute(selector, attribute, { timeout })) || '';

    let success = false;
    let message = '';

    if (mode === 'contains') {
      success = actual.includes(expected);
      message = success ? '' : `expected attribute ${attribute} "${actual}" to contain "${expected}"`;
    } else {
      // equals
      success = actual === expected;
      message = success ? '' : `expected attribute ${attribute} "${actual}" to equal "${expected}"`;
    }

    return {
      mode: `attribute_${mode}`,
      selector,
      expected,
      actual,
      success,
      message,
    };
  }

  private async assertText(
    page: Page,
    selector: string,
    mode: string,
    expected: string,
    timeout: number
  ): Promise<AssertionOutcome> {
    const actual = (await page.textContent(selector, { timeout })) || '';

    let success = false;
    let message = '';

    if (mode === 'matches') {
      // Hardened: RegExp construction can throw on invalid patterns
      // Handle gracefully with a clear error message instead of crashing
      let regex: RegExp;
      try {
        regex = new RegExp(expected);
      } catch (regexError) {
        const errorMsg = regexError instanceof Error ? regexError.message : String(regexError);
        return {
          mode,
          selector,
          expected,
          actual,
          success: false,
          message: `Invalid regex pattern "${expected}": ${errorMsg}`,
        };
      }
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
