import { AssertionHandler } from '../../../src/handlers/assertion';
import type { CompiledInstruction, HandlerContext } from '../../../src/types';
import { createMockPage, createTestConfig } from '../../helpers';
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
      const instruction: CompiledInstruction = {
        type: 'assert',
        params: { selector: '#element', mode: 'exists' },
        node_id: 'node-1',
      };

      const mockLocator = {
        count: jest.fn().mockResolvedValue(1),
      };
      mockPage.locator.mockReturnValue(mockLocator as any);

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      expect(result.extracted_data?.assertion_result?.passed).toBe(true);
    });

    it('should fail when element does not exist', async () => {
      const instruction: CompiledInstruction = {
        type: 'assert',
        params: { selector: '#missing', mode: 'exists' },
        node_id: 'node-1',
      };

      const mockLocator = {
        count: jest.fn().mockResolvedValue(0),
      };
      mockPage.locator.mockReturnValue(mockLocator as any);

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error?.kind).toBe('user');
      expect(result.extracted_data?.assertion_result?.passed).toBe(false);
    });
  });

  describe('assert - visible', () => {
    it('should pass when element is visible', async () => {
      const instruction: CompiledInstruction = {
        type: 'assert',
        params: { selector: '#element', mode: 'visible' },
        node_id: 'node-1',
      };

      const mockLocator = {
        isVisible: jest.fn().mockResolvedValue(true),
      };
      mockPage.locator.mockReturnValue(mockLocator as any);

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      expect(result.extracted_data?.assertion_result?.passed).toBe(true);
    });
  });

  describe('assert - text', () => {
    it('should assert text equals', async () => {
      const instruction: CompiledInstruction = {
        type: 'assert',
        params: { selector: '#element', mode: 'equals', expectedText: 'Hello' },
        node_id: 'node-1',
      };

      const mockLocator = {
        textContent: jest.fn().mockResolvedValue('Hello'),
      };
      mockPage.locator.mockReturnValue(mockLocator as any);

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      expect(result.extracted_data?.assertion_result?.passed).toBe(true);
    });

    it('should assert text contains', async () => {
      const instruction: CompiledInstruction = {
        type: 'assert',
        params: { selector: '#element', mode: 'contains', expectedText: 'World' },
        node_id: 'node-1',
      };

      const mockLocator = {
        textContent: jest.fn().mockResolvedValue('Hello World'),
      };
      mockPage.locator.mockReturnValue(mockLocator as any);

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      expect(result.extracted_data?.assertion_result?.passed).toBe(true);
    });
  });
});
