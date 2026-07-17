import {
  createTypedInstruction,
  createTestInstruction,
  createMockPage,
  createMockContext,
  createTestConfig,
} from '../../helpers';
import { NavigationHandler } from '../../../src/handlers/navigation';
import type { HandlerContext } from '../../../src/handlers/base';
import { logger, metrics } from '../../../src/utils';
import { FailureKind } from '../../../src/proto';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const isUnknownArray = (value: unknown): value is unknown[] => Array.isArray(value);

const getGotoCall = (
  page: ReturnType<typeof createMockPage>
): { url: string; options: Record<string, unknown> } => {
  const call: unknown = page.goto.mock.calls[0];
  if (!isUnknownArray(call)) {
    throw new Error('Expected page.goto to be called');
  }

  const urlValue = call[0];
  const optionsValue = call[1];
  if (typeof urlValue !== 'string') {
    throw new Error('Expected page.goto to be called with a URL string');
  }

  return {
    url: urlValue,
    options: isRecord(optionsValue) ? optionsValue : {},
  };
};

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
      browserContext: createMockContext(),
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
      const { url, options } = getGotoCall(mockPage);
      expect(url).toBe('https://example.com/');
      expect(options).toEqual(expect.objectContaining({ waitUntil: 'domcontentloaded' }));
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
      const { url, options } = getGotoCall(mockPage);
      expect(url).toBe('https://example.com/');
      expect(options).toEqual(expect.objectContaining({ timeout: 60000 }));
    });

    it('should use custom waitUntil when provided', async () => {
      const instruction = createTypedInstruction('navigate', { url: 'https://example.com', waitUntil: 'load' }, { nodeId: 'node-1' });

      await handler.execute(instruction, context);

      // Note: URL is normalized (trailing slash added by URL.href)
      const { url, options } = getGotoCall(mockPage);
      expect(url).toBe('https://example.com/');
      expect(options).toEqual(expect.objectContaining({ waitUntil: 'domcontentloaded' }));
      expect(mockPage.waitForLoadState).toHaveBeenCalledWith('load', expect.objectContaining({ timeout: expect.any(Number) }));
    });

    it('should support domcontentloaded waitUntil', async () => {
      const instruction = createTypedInstruction('navigate', { url: 'https://example.com', waitUntil: 'domcontentloaded' }, { nodeId: 'node-1' });

      await handler.execute(instruction, context);

      // Note: URL is normalized (trailing slash added by URL.href)
      const { url, options } = getGotoCall(mockPage);
      expect(url).toBe('https://example.com/');
      expect(options).toEqual(expect.objectContaining({ waitUntil: 'domcontentloaded' }));
    });

    it('waits for a selector only after navigation completes', async () => {
      const sequence: string[] = [];
      mockPage.goto.mockImplementation(async () => { sequence.push('goto'); return null; });
      mockPage.waitForSelector.mockImplementation(async () => { sequence.push('selector'); return null; });
      const instruction = createTypedInstruction('navigate', { url: 'https://example.com', waitForSelector: '[data-testid=ready]' }, { nodeId: 'node-1' });

      await handler.execute(instruction, context);

      expect(sequence).toEqual(['goto', 'selector']);
      expect(mockPage.waitForSelector).toHaveBeenCalledWith('[data-testid=ready]', expect.objectContaining({ state: 'visible' }));
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
      const { url, options } = getGotoCall(mockPage);
      expect(url).toBe('https://example.com/');
      const timeout = options.timeout;
      expect(typeof timeout).toBe('number');
    });

    it('should use default waitUntil when not provided', async () => {
      const instruction = createTypedInstruction('navigate', { url: 'https://example.com' }, { nodeId: 'node-1' });

      await handler.execute(instruction, context);

      // Note: URL is normalized (trailing slash added by URL.href)
      // Default is 'domcontentloaded' - 'networkidle' times out on ad-heavy sites
      const { url, options } = getGotoCall(mockPage);
      expect(url).toBe('https://example.com/');
      expect(options).toEqual(expect.objectContaining({ waitUntil: 'domcontentloaded' }));
    });
  });
});
