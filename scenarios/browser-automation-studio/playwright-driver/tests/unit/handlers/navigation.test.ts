import { createTestInstruction } from '../../helpers';
import { NavigationHandler } from '../../../src/handlers/navigation';
import type { CompiledInstruction, HandlerContext } from '../../../src/types';
import { createMockPage, createTestConfig } from '../../helpers';
import { logger, metrics } from '../../../src/utils';

describe('NavigationHandler', () => {
  let handler: NavigationHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let context: HandlerContext;

  beforeEach(() => {
    handler = new NavigationHandler();
    mockPage = createMockPage();

    const config = createTestConfig();
    context = {
      page: mockPage,
      context: {} as any,
      config,
      logger,
      metrics,
      sessionId: 'test-session',
    };
  });

  describe('getSupportedTypes', () => {
    it('should support navigate instruction', () => {
      const types = handler.getSupportedTypes();

      expect(types).toEqual(['navigate']);
    });
  });

  describe('execute', () => {
    it('should navigate to URL from params', async () => {
      const instruction = createTestInstruction({
        type: 'navigate',
        params: { url: 'https://example.com' },
        node_id: 'node-1',
      });

      const result = await handler.execute(instruction, context);

      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://example.com',
        expect.objectContaining({
          waitUntil: 'networkidle',
        })
      );
      expect(result.success).toBe(true);
    });

    it('should use current URL when url not provided', async () => {
      mockPage.url.mockReturnValue('https://current-page.com');

      const instruction = createTestInstruction({
        type: 'navigate',
        params: {},
        node_id: 'node-1',
      });

      const result = await handler.execute(instruction, context);

      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://current-page.com',
        expect.any(Object)
      );
      expect(result.success).toBe(true);
    });

    it('should use custom timeout when provided', async () => {
      const instruction = createTestInstruction({
        type: 'navigate',
        params: { url: 'https://example.com', timeoutMs: 60000 },
        node_id: 'node-1',
      });

      await handler.execute(instruction, context);

      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://example.com',
        expect.objectContaining({
          timeout: 60000,
        })
      );
    });

    it('should use custom waitUntil when provided', async () => {
      const instruction = createTestInstruction({
        type: 'navigate',
        params: { url: 'https://example.com', waitUntil: 'load' },
        node_id: 'node-1',
      });

      await handler.execute(instruction, context);

      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://example.com',
        expect.objectContaining({
          waitUntil: 'load',
        })
      );
    });

    it('should support domcontentloaded waitUntil', async () => {
      const instruction = createTestInstruction({
        type: 'navigate',
        params: { url: 'https://example.com', waitUntil: 'domcontentloaded' },
        node_id: 'node-1',
      });

      await handler.execute(instruction, context);

      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://example.com',
        expect.objectContaining({
          waitUntil: 'domcontentloaded',
        })
      );
    });

    it('should handle navigation errors', async () => {
      mockPage.goto.mockRejectedValue(new Error('Navigation failed'));

      const instruction = createTestInstruction({
        type: 'navigate',
        params: { url: 'https://example.com' },
        node_id: 'node-1',
      });

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
      expect(result.error?.message).toContain('Navigation failed');
    });

    it('should handle timeout errors', async () => {
      mockPage.goto.mockRejectedValue(new Error('Timeout 30000ms exceeded'));

      const instruction = createTestInstruction({
        type: 'navigate',
        params: { url: 'https://example.com' },
        node_id: 'node-1',
      });

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error?.kind).toBe('timeout');
    });

    it('should validate params with Zod schema', async () => {
      const instruction = createTestInstruction({
        type: 'navigate',
        params: { waitUntil: 'invalid-value' }, // Invalid waitUntil
        node_id: 'node-1',
      });

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
    });

    it('should use default timeout when not provided', async () => {
      const instruction = createTestInstruction({
        type: 'navigate',
        params: { url: 'https://example.com' },
        node_id: 'node-1',
      });

      await handler.execute(instruction, context);

      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://example.com',
        expect.objectContaining({
          timeout: expect.any(Number),
        })
      );
    });

    it('should use default waitUntil when not provided', async () => {
      const instruction = createTestInstruction({
        type: 'navigate',
        params: { url: 'https://example.com' },
        node_id: 'node-1',
      });

      await handler.execute(instruction, context);

      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://example.com',
        expect.objectContaining({
          waitUntil: 'networkidle',
        })
      );
    });
  });
});
