import type { Page } from 'playwright';
import { BaseHandler, type HandlerContext, type HandlerResult } from './base';
import type { HandlerInstruction, AssertionOutcome } from '../types';
import { getAssertParams } from '../proto';
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
    instruction: HandlerInstruction,
    context: HandlerContext
  ): Promise<HandlerResult> {
    const { page, logger } = context;

    try {
      // Extract typed params from action
      const typedParams = instruction.action ? getAssertParams(instruction.action) : undefined;
      const params = this.requireTypedParams(typedParams, 'assert', instruction.nodeId);

      // Extract values from typed params
      const selector = params.selector;
      const timeoutMs = params.timeoutMs;
      const mode = params.mode || 'contains';
      const attributeName = params.attributeName;
      const expectedValue = params.expected ?? '';

      if (!selector) {
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

      // Prefer config timeout, fallback to param, then constant default
      const timeout = timeoutMs || context.config.execution.assertionTimeoutMs || DEFAULT_ASSERTION_TIMEOUT_MS;

      // Determine assertion mode - already set above
      const normalizedMode = mode.toLowerCase();

      logger.debug('Running assertion', {
        selector,
        mode: normalizedMode,
        timeout,
      });

      let assertion: AssertionOutcome;

      switch (normalizedMode) {
        case 'exists':
          assertion = await this.assertExists(page, selector, timeout);
          break;

        case 'notexists':
        case 'not_exists':
        case 'absent':
          assertion = await this.assertNotExists(page, selector, timeout, logger);
          break;

        case 'visible':
          assertion = await this.assertVisible(page, selector, timeout);
          break;

        case 'hidden':
        case 'notvisible':
        case 'not_visible':
          assertion = await this.assertHidden(page, selector, timeout);
          break;

        case 'attribute':
        case 'attribute_equals':
        case 'attribute_contains':
          if (!attributeName) {
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
          const attrMode = normalizedMode.replace('attribute_', '');
          assertion = await this.assertAttribute(
            page,
            selector,
            String(attributeName),
            String(expectedValue),
            timeout,
            attrMode
          );
          break;

        case 'text_equals':
        case 'equals':
          assertion = await this.assertText(
            page,
            selector,
            'equals',
            String(expectedValue),
            timeout
          );
          break;

        case 'text_contains':
        case 'contains':
          assertion = await this.assertText(
            page,
            selector,
            'contains',
            String(expectedValue),
            timeout
          );
          break;

        case 'matches':
        default:
          assertion = await this.assertText(
            page,
            selector,
            normalizedMode,
            String(expectedValue),
            timeout
          );
          break;
      }

      logger.info('Assertion complete', {
        selector,
        mode: normalizedMode,
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
