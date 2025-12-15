import { createTypedInstruction, createTestInstruction, createMockPage, createTestConfig } from '../../helpers';
import { NavigationHandler } from '../../../src/handlers/navigation';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';
import { FailureKind } from '../../../src/proto';

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
      const instruction = createTypedInstruction('navigate', { url: 'https://example.com' }, { nodeId: 'node-1' });

      const result = await handler.execute(instruction, context);

      // Note: URL is normalized (trailing slash added by URL.href)
      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://example.com/',
        expect.objectContaining({
          waitUntil: 'networkidle',
        })
      );
      expect(result.success).toBe(true);
    });

    it('should return error when url not provided', async () => {
      const instruction = createTypedInstruction('navigate', {}, { nodeId: 'node-1' });

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('MISSING_PARAM');
    });

    it('should use custom timeout when provided', async () => {
      const instruction = createTypedInstruction('navigate', { url: 'https://example.com', timeoutMs: 60000 }, { nodeId: 'node-1' });

      await handler.execute(instruction, context);

      // Note: URL is normalized (trailing slash added by URL.href)
      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://example.com/',
        expect.objectContaining({
          timeout: 60000,
        })
      );
    });

    it('should use custom waitUntil when provided', async () => {
      const instruction = createTypedInstruction('navigate', { url: 'https://example.com', waitUntil: 'load' }, { nodeId: 'node-1' });

      await handler.execute(instruction, context);

      // Note: URL is normalized (trailing slash added by URL.href)
      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://example.com/',
        expect.objectContaining({
          waitUntil: 'load',
        })
      );
    });

    it('should support domcontentloaded waitUntil', async () => {
      const instruction = createTypedInstruction('navigate', { url: 'https://example.com', waitUntil: 'domcontentloaded' }, { nodeId: 'node-1' });

      await handler.execute(instruction, context);

      // Note: URL is normalized (trailing slash added by URL.href)
      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://example.com/',
        expect.objectContaining({
          waitUntil: 'domcontentloaded',
        })
      );
    });

    it('should handle navigation errors', async () => {
      mockPage.goto.mockRejectedValue(new Error('Navigation failed'));

      const instruction = createTypedInstruction('navigate', { url: 'https://example.com' }, { nodeId: 'node-1' });

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
      expect(result.error?.message).toContain('Navigation failed');
    });

    it('should handle timeout errors', async () => {
      mockPage.goto.mockRejectedValue(new Error('Timeout 30000ms exceeded'));

      const instruction = createTypedInstruction('navigate', { url: 'https://example.com' }, { nodeId: 'node-1' });

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error?.kind).toBe(FailureKind.TIMEOUT);
    });

    it('should return error for missing URL with legacy params', async () => {
      // Test edge case with legacy createTestInstruction (no action field)
      const instruction = createTestInstruction({
        type: 'navigate',
        params: { waitUntil: 'load' }, // Missing URL
        nodeId: 'node-1',
      });

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
    });

    it('should use default timeout when not provided', async () => {
      const instruction = createTypedInstruction('navigate', { url: 'https://example.com' }, { nodeId: 'node-1' });

      await handler.execute(instruction, context);

      // Note: URL is normalized (trailing slash added by URL.href)
      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://example.com/',
        expect.objectContaining({
          timeout: expect.any(Number),
        })
      );
    });

    it('should use default waitUntil when not provided', async () => {
      const instruction = createTypedInstruction('navigate', { url: 'https://example.com' }, { nodeId: 'node-1' });

      await handler.execute(instruction, context);

      // Note: URL is normalized (trailing slash added by URL.href)
      expect(mockPage.goto).toHaveBeenCalledWith(
        'https://example.com/',
        expect.objectContaining({
          waitUntil: 'networkidle',
        })
      );
    });
  });
});
