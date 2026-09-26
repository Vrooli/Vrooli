import {
  createMockPage,
  createMockContext,
  createTestConfig,
  createTestInstruction,
} from '../../helpers';
import { AssertionHandler } from '../../../src/handlers/assertion';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';
import { create } from '@bufbuild/protobuf';
import {
  ActionDefinitionSchema,
  AssertParamsSchema,
  ActionType,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { AssertionMode } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { JsonValueSchema } from '@vrooli/proto-types/common/v1/types_pb';
import { errors } from 'rebrowser-playwright';

const buildAssertInstruction = (params: {
  selector: string;
  mode: AssertionMode;
  expected?: string;
  attributeName?: string;
  negated?: boolean;
  caseSensitive?: boolean;
  failureMessage?: string;
  timeoutMs?: number;
}) => {
  const action = create(ActionDefinitionSchema, {
    type: ActionType.ASSERT,
    params: {
      case: 'assert',
      value: create(AssertParamsSchema, {
        ...params,
        expected: params.expected !== undefined
          ? create(JsonValueSchema, { kind: { case: 'stringValue', value: params.expected } })
          : undefined,
      }),
    },
  });

  return createTestInstruction({
    type: 'assert',
    action,
    params: {},
  });
};

describe('AssertionHandler', () => {
  let handler: AssertionHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new AssertionHandler();
    mockPage = createMockPage();
    context = {
      page: mockPage,
      browserContext: createMockContext(),
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'test-session',
    };
  });

  // [REQ:BAS-RH-J24] Negation changes a successfully evaluated predicate only.
  describe('typed assertion semantics', () => {
    const cases: { mode: AssertionMode; state?: string; actual: string; expected?: string; matches: boolean }[] = [
      { mode: AssertionMode.EXISTS, state: 'attached', actual: 'present', matches: true },
      { mode: AssertionMode.NOT_EXISTS, state: 'detached', actual: 'absent', matches: true },
      { mode: AssertionMode.VISIBLE, state: 'visible', actual: 'visible', matches: true },
      { mode: AssertionMode.HIDDEN, state: 'hidden', actual: 'hidden', matches: true },
      { mode: AssertionMode.TEXT_EQUALS, actual: 'Ready', expected: 'Ready', matches: true },
      { mode: AssertionMode.TEXT_CONTAINS, actual: 'Ready now', expected: 'Ready', matches: true },
      { mode: AssertionMode.ATTRIBUTE_EQUALS, actual: 'Ready', expected: 'Ready', matches: true },
      { mode: AssertionMode.ATTRIBUTE_CONTAINS, actual: 'Ready now', expected: 'Ready', matches: true },
    ];

    it.each(cases.flatMap(c => [false, true].flatMap(matches => [false, true].map(negated => ({ ...c, matches, negated })))))
    ('mode $mode matching=$matches negated=$negated', async ({ mode, state, actual, expected, matches, negated }) => {
      const wait = mockPage.locator('#subject').first().waitFor as jest.Mock;
      wait.mockImplementation(async ({ state: wanted }) => {
        if ((wanted === state) !== matches) throw new errors.TimeoutError('Timed out waiting for state');
      });
      (mockPage.$ as jest.Mock).mockResolvedValue(matches === (mode === AssertionMode.EXISTS) ? {} : null);
      mockPage.isVisible.mockResolvedValue(matches === (mode === AssertionMode.VISIBLE));
      mockPage.textContent.mockResolvedValue(matches ? actual : 'different');
      mockPage.getAttribute.mockResolvedValue(matches ? actual : 'different');
      const result = await handler.execute(buildAssertInstruction({
        selector: '#subject', mode, expected, negated, attributeName: 'data-state', timeoutMs: 12,
      }), context);
      expect(result.success).toBe(matches !== negated);
      expect(result.error).toBeUndefined();
      expect(result.extracted_data?.assertion).toMatchObject({ success: matches !== negated, negated, caseSensitive: true });
    });

    it.each([AssertionMode.TEXT_EQUALS, AssertionMode.TEXT_CONTAINS, AssertionMode.ATTRIBUTE_EQUALS, AssertionMode.ATTRIBUTE_CONTAINS]
      .flatMap(mode => [true, false].map(caseSensitive => ({ mode, caseSensitive }))))
    ('honors case sensitivity for mode $mode: $caseSensitive', async ({ mode, caseSensitive }) => {
      mockPage.textContent.mockResolvedValue('READY');
      mockPage.getAttribute.mockResolvedValue('READY');
      const result = await handler.execute(buildAssertInstruction({
        selector: '#subject', mode, expected: 'ready', attributeName: 'data-state', caseSensitive,
      }), context);
      expect(result.success).toBe(!caseSensitive);
      expect(result.extracted_data?.assertion).toMatchObject({ actual: 'READY', expected: 'ready', caseSensitive });
    });

    it.each([false, true])('preserves custom mismatch message, negated=%s', async negated => {
      mockPage.textContent.mockResolvedValue(negated ? 'expected' : 'different');
      const result = await handler.execute(buildAssertInstruction({ selector: '#subject', mode: AssertionMode.TEXT_EQUALS,
        expected: 'expected', negated, failureMessage: 'Checkout is not ready' }), context);
      expect(result.success).toBe(false);
      expect(result.extracted_data?.assertion).toMatchObject({ message: 'Checkout is not ready', negated });
    });

    it.each([false, true])('distinguishes missing from empty attributes, negated=%s', async negated => {
      mockPage.getAttribute.mockResolvedValue(null);
      const result = await handler.execute(buildAssertInstruction({ selector: '#subject', mode: AssertionMode.ATTRIBUTE_EQUALS,
        expected: '', attributeName: 'data-missing', negated }), context);
      expect(result.success).toBe(negated);
      expect(result.extracted_data?.assertion).toMatchObject({ actual: '(missing attribute)' });
    });

    it.each(cases.flatMap(c => [false, true].map(negated => ({ ...c, negated }))))
    ('keeps browser failure distinct for mode $mode, negated=$negated', async ({ mode, negated }) => {
      const failure = new Error('Target page, context or browser has been closed');
      (mockPage.locator('#subject').first().waitFor as jest.Mock).mockRejectedValue(failure);
      mockPage.$.mockRejectedValue(failure);
      mockPage.isVisible.mockRejectedValue(failure);
      mockPage.textContent.mockRejectedValue(failure);
      mockPage.getAttribute.mockRejectedValue(failure);
      const result = await handler.execute(buildAssertInstruction({ selector: '#subject', mode, negated, attributeName: 'data-state' }), context);
      expect(result.success).toBe(false);
      expect(result.error?.message).toContain('closed');
      expect(result.extracted_data?.assertion).toBeUndefined();
    });

    it.each([AssertionMode.UNSPECIFIED, 999 as AssertionMode])('rejects invalid typed mode %s', async mode => {
      const result = await handler.execute(buildAssertInstruction({ selector: '#subject', mode }), context);
      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('INVALID_INSTRUCTION');
    });

    it.each([false, true])('does not negate an invalid selector, negated=%s', async negated => {
      (mockPage.locator('[').first().waitFor as jest.Mock).mockRejectedValue(new Error('Unexpected token in selector'));
      const result = await handler.execute(buildAssertInstruction({ selector: '[', mode: AssertionMode.EXISTS, negated }), context);
      expect(result.success).toBe(false);
      expect(result.error?.message).toContain('selector');
      expect(result.extracted_data?.assertion).toBeUndefined();
    });

    it.each([AssertionMode.EXISTS, AssertionMode.NOT_EXISTS, AssertionMode.VISIBLE, AssertionMode.HIDDEN]
      .flatMap(mode => [false, true].flatMap(present => [false, true].map(negated => ({ mode, present, negated })))))
    ('observes immediately at timeout zero: $mode present=$present negated=$negated', async ({ mode, present, negated }) => {
      const locator = mockPage.locator('#subject').first();
      Object.assign(locator, { count: jest.fn().mockResolvedValue(present ? 1 : 0), isVisible: jest.fn().mockResolvedValue(present) });
      const result = await handler.execute(buildAssertInstruction({ selector: '#subject', mode, negated, timeoutMs: 0 }), context);
      const positive = mode === AssertionMode.EXISTS || mode === AssertionMode.VISIBLE;
      expect(result.success).toBe((present === positive) !== negated);
      expect(locator.waitFor).not.toHaveBeenCalled();
    });
  });

  describe('assert - exists', () => {
    // exists WAITS for the element up to the assertion timeout rather than
    // sampling the DOM once. Sampling made every exists assertion a race
    // against whatever the previous step set in motion.
    it('should pass when the element attaches', async () => {
      const instruction = buildAssertInstruction({ selector: '#element', mode: AssertionMode.EXISTS });

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const assertion = result.extracted_data?.assertion as { success: boolean } | undefined;
      expect(assertion?.success).toBe(true);
    });

    it('waits for the element instead of sampling the DOM once', async () => {
      const instruction = buildAssertInstruction({ selector: '#element', mode: AssertionMode.EXISTS });

      await handler.execute(instruction, context);

      // The regression this guards: reverting to page.$() would still report
      // the right answer for an element that is already present, so asserting
      // the outcome alone cannot catch it. Assert the wait actually happened.
      const firstLocator = mockPage.locator('#element').first() as unknown as {
        waitFor: jest.Mock;
      };
      expect(firstLocator.waitFor).toHaveBeenCalledWith(
        expect.objectContaining({ state: 'attached' })
      );
    });

    it('should fail when the element never attaches', async () => {
      const instruction = buildAssertInstruction({ selector: '#missing', mode: AssertionMode.EXISTS });

      // waitFor rejects on timeout, which is how absence now surfaces.
      const firstLocator = mockPage.locator('#missing').first() as unknown as {
        waitFor: jest.Mock;
      };
      firstLocator.waitFor.mockRejectedValue(new errors.TimeoutError('Timeout 5000ms exceeded'));

      const result = await handler.execute(instruction, context);

      // Assertion failures return success=false with the assertion result,
      // but no error object (error is only set for exceptions)
      expect(result.success).toBe(false);
      const assertion = result.extracted_data?.assertion as { success: boolean } | undefined;
      expect(assertion?.success).toBe(false);
    });
  });

  describe('assert - visible', () => {
    it('should pass when element is visible', async () => {
      const instruction = buildAssertInstruction({ selector: '#element', mode: AssertionMode.VISIBLE });

      // The assertion handler uses page.isVisible() not page.locator().isVisible()
      mockPage.isVisible = jest.fn().mockResolvedValue(true);

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const assertion = result.extracted_data?.assertion as { success: boolean } | undefined;
      expect(assertion?.success).toBe(true);
    });
  });

  describe('assert - text', () => {
    it('should assert text equals', async () => {
      const instruction = buildAssertInstruction({ selector: '#element', mode: AssertionMode.TEXT_EQUALS, expected: 'Hello' });

      // The assertion handler uses page.textContent() not page.locator().textContent()
      mockPage.textContent = jest.fn().mockResolvedValue('Hello');

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const assertion = result.extracted_data?.assertion as { success: boolean } | undefined;
      expect(assertion?.success).toBe(true);
    });

    it('should assert text contains', async () => {
      const instruction = buildAssertInstruction({ selector: '#element', mode: AssertionMode.TEXT_CONTAINS, expected: 'World' });

      // The assertion handler uses page.textContent() not page.locator().textContent()
      mockPage.textContent = jest.fn().mockResolvedValue('Hello World');

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const assertion = result.extracted_data?.assertion as { success: boolean } | undefined;
      expect(assertion?.success).toBe(true);
    });
  });

  describe('assert - additional branches', () => {
    it('returns error when selector is missing', async () => {
      const instruction = buildAssertInstruction({
        selector: '',
        mode: AssertionMode.EXISTS,
      });

      const result = await handler.execute(instruction, context);
      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('MISSING_PARAM');
    });

    it('handles not-exists when element is already absent', async () => {
      const instruction = buildAssertInstruction({
        selector: '#missing',
        mode: AssertionMode.NOT_EXISTS,
      });

      mockPage.$.mockResolvedValue(null);

      const result = await handler.execute(instruction, context);
      const assertion = result.extracted_data?.assertion as { success: boolean } | undefined;
      expect(assertion?.success).toBe(true);
    });

    it('handles not-exists timeout when element stays present', async () => {
      const instruction = buildAssertInstruction({
        selector: '#stays',
        mode: AssertionMode.NOT_EXISTS,
      });

      mockPage.$.mockResolvedValue({}); // element exists
      (mockPage.locator('#stays').first().waitFor as jest.Mock).mockRejectedValue(new errors.TimeoutError('timeout'));

      const result = await handler.execute(instruction, context);
      const assertion = result.extracted_data?.assertion as { success: boolean; message?: string } | undefined;
      expect(assertion?.success).toBe(false);
      expect(assertion?.message).toContain('expected element to be absent');
    });

    it('returns error when attribute assertion missing attributeName', async () => {
      const instruction = buildAssertInstruction({
        selector: '#attr',
        mode: AssertionMode.ATTRIBUTE_EQUALS,
        expected: 'value',
      });

      const result = await handler.execute(instruction, context);
      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('MISSING_PARAM');
    });

    it('asserts attribute contains', async () => {
      const instruction = buildAssertInstruction({
        selector: '#attr',
        mode: AssertionMode.ATTRIBUTE_CONTAINS,
        expected: 'test',
        attributeName: 'data-test',
      });

      mockPage.getAttribute = jest.fn().mockResolvedValue('test-value');

      const result = await handler.execute(instruction, context);
      expect(result.success).toBe(true);
      const assertion = result.extracted_data?.assertion as { success: boolean; mode?: string } | undefined;
      expect(assertion?.mode).toBe('attribute_contains');
    });

    it('asserts attribute equals', async () => {
      const instruction = buildAssertInstruction({
        selector: '#attr',
        mode: AssertionMode.ATTRIBUTE_EQUALS,
        expected: 'exact',
        attributeName: 'data-test',
      });

      mockPage.getAttribute = jest.fn().mockResolvedValue('exact');

      const result = await handler.execute(instruction, context);
      expect(result.success).toBe(true);
      const assertion = result.extracted_data?.assertion as { success: boolean; mode?: string } | undefined;
      expect(assertion?.mode).toBe('attribute_equals');
    });

    it('asserts hidden when element not visible', async () => {
      const instruction = buildAssertInstruction({
        selector: '#hidden',
        mode: AssertionMode.HIDDEN,
      });

      mockPage.isVisible = jest.fn().mockResolvedValue(false);

      const result = await handler.execute(instruction, context);
      expect(result.success).toBe(true);
    });
  });
});
