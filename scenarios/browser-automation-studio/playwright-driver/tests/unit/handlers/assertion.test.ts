import { createTestInstruction, createMockPage, createTestConfig } from '../../helpers';
import { AssertionHandler } from '../../../src/handlers/assertion';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';

describe('AssertionHandler', () => {
  let handler: AssertionHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new AssertionHandler();
    mockPage = createMockPage();
    context = {
      page: mockPage,
      context: {} as any,
      config: createTestConfig(),
      logger,
      metrics,
      sessionId: 'test-session',
    };
  });

  describe('assert - exists', () => {
    it('should pass when element exists', async () => {
      const instruction = createTestInstruction({
        type: 'assert',
        params: { selector: '#element', mode: 'exists' },
        node_id: 'node-1',
      });

      const mockLocator = {
        count: jest.fn().mockResolvedValue(1),
      };
      mockPage.locator.mockReturnValue(mockLocator as any);

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const assertion = result.extracted_data?.assertion as { success: boolean } | undefined;
      expect(assertion?.success).toBe(true);
    });

    it('should fail when element does not exist', async () => {
      const instruction = createTestInstruction({
        type: 'assert',
        params: { selector: '#missing', mode: 'exists' },
        node_id: 'node-1',
      });

      // The assertion handler uses page.$() not page.locator()
      mockPage.$.mockResolvedValue(null);

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
      const instruction = createTestInstruction({
        type: 'assert',
        params: { selector: '#element', mode: 'visible' },
        node_id: 'node-1',
      });

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
      const instruction = createTestInstruction({
        type: 'assert',
        params: { selector: '#element', mode: 'equals', expected: 'Hello' },
        node_id: 'node-1',
      });

      // The assertion handler uses page.textContent() not page.locator().textContent()
      mockPage.textContent = jest.fn().mockResolvedValue('Hello');

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const assertion = result.extracted_data?.assertion as { success: boolean } | undefined;
      expect(assertion?.success).toBe(true);
    });

    it('should assert text contains', async () => {
      const instruction = createTestInstruction({
        type: 'assert',
        params: { selector: '#element', mode: 'contains', expected: 'World' },
        node_id: 'node-1',
      });

      // The assertion handler uses page.textContent() not page.locator().textContent()
      mockPage.textContent = jest.fn().mockResolvedValue('Hello World');

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      const assertion = result.extracted_data?.assertion as { success: boolean } | undefined;
      expect(assertion?.success).toBe(true);
    });
  });
});
