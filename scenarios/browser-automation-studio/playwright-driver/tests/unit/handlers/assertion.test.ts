import {
  createTypedInstruction,
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

const buildAssertInstruction = (params: {
  selector: string;
  mode: AssertionMode;
  expected?: string;
  attributeName?: string;
}) => {
  const action = create(ActionDefinitionSchema, {
    type: ActionType.ASSERT,
    params: {
      case: 'assert',
      value: create(AssertParamsSchema, {
        selector: params.selector,
        mode: params.mode,
        expected: params.expected
          ? create(JsonValueSchema, { kind: { case: 'stringValue', value: params.expected } })
          : undefined,
        attributeName: params.attributeName,
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

  describe('assert - exists', () => {
    // exists WAITS for the element up to the assertion timeout rather than
    // sampling the DOM once. Sampling made every exists assertion a race
    // against whatever the previous step set in motion.
    it('should pass when the element attaches', async () => {
      const instruction = createTypedInstruction('assert', { selector: '#element', mode: 'exists' }, { nodeId: 'node-1' });

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const assertion = result.extracted_data?.assertion as { success: boolean } | undefined;
      expect(assertion?.success).toBe(true);
    });

    it('waits for the element instead of sampling the DOM once', async () => {
      const instruction = createTypedInstruction('assert', { selector: '#element', mode: 'exists' }, { nodeId: 'node-1' });

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
      const instruction = createTypedInstruction('assert', { selector: '#missing', mode: 'exists' }, { nodeId: 'node-1' });

      // waitFor rejects on timeout, which is how absence now surfaces.
      const firstLocator = mockPage.locator('#missing').first() as unknown as {
        waitFor: jest.Mock;
      };
      firstLocator.waitFor.mockRejectedValue(new Error('Timeout 5000ms exceeded'));

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
      const instruction = createTypedInstruction('assert', { selector: '#element', mode: 'visible' }, { nodeId: 'node-1' });

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
      const instruction = createTypedInstruction('assert', { selector: '#element', mode: 'equals', expected: 'Hello' }, { nodeId: 'node-1' });

      // The assertion handler uses page.textContent() not page.locator().textContent()
      mockPage.textContent = jest.fn().mockResolvedValue('Hello');

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const assertion = result.extracted_data?.assertion as { success: boolean } | undefined;
      expect(assertion?.success).toBe(true);
    });

    it('should assert text contains', async () => {
      const instruction = createTypedInstruction('assert', { selector: '#element', mode: 'contains', expected: 'World' }, { nodeId: 'node-1' });

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
      mockPage.locator.mockReturnValue({
        waitFor: jest.fn().mockRejectedValue(new Error('timeout')),
      } as never);

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
