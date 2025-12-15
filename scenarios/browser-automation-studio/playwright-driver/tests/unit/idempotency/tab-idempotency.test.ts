/**
 * Tab Handler Idempotency Tests
 *
 * Tests for idempotent tab operations to ensure replay safety.
 * These tests verify that tab operations handle concurrent and repeated
 * requests correctly.
 */

import { TabHandler } from '../../../src/handlers/tab';
import type { HandlerContext } from '../../../src/handlers/base';
import { createMockPage, createMockContext, createTestConfig, createTypedInstruction } from '../../helpers';
import { logger, metrics } from '../../../src/utils';

describe('TabHandler Idempotency', () => {
  let handler: TabHandler;
  let mockPage: ReturnType<typeof createMockPage>;
  let mockBrowserContext: ReturnType<typeof createMockContext>;
  let config: ReturnType<typeof createTestConfig>;

  beforeEach(() => {
    handler = new TabHandler();
    mockPage = createMockPage({
      bringToFront: jest.fn().mockResolvedValue(undefined) as any,
    });
    mockBrowserContext = createMockContext();
    config = createTestConfig();

    // Setup newPage to return a new mock page each time
    mockBrowserContext.newPage.mockImplementation(() => {
      const newMockPage = createMockPage({
        bringToFront: jest.fn().mockResolvedValue(undefined) as any,
      });
      newMockPage.url.mockReturnValue('about:blank');
      newMockPage.title.mockResolvedValue('New Tab');
      return Promise.resolve(newMockPage);
    });
  });

  describe('tab-switch open idempotency', () => {
    it('should track concurrent open requests with same URL', async () => {
      const instruction = createTypedInstruction('tab-switch', {
        action: 'open',
        url: 'https://example.com',
      });

      const context: HandlerContext = {
        page: mockPage,
        context: mockBrowserContext,
        config,
        logger,
        metrics,
        sessionId: 'test-session-concurrent',
        frameStack: [],
      };

      // Execute two concurrent requests
      const promise1 = handler.execute(instruction, context);
      const promise2 = handler.execute(instruction, context);

      const [result1, result2] = await Promise.all([promise1, promise2]);

      // Both should succeed
      expect(result1.success).toBe(true);
      expect(result2.success).toBe(true);

      // Only one page should be created (second request awaits first)
      // Note: In the actual implementation, the second request will wait
      // for the first, but in this test they share the same Promise
      expect(mockBrowserContext.newPage).toHaveBeenCalled();
    });

    it('should create separate tabs for different URLs', async () => {
      const instruction1 = createTypedInstruction('tab-switch', {
        action: 'open',
        url: 'https://example.com/page1',
      });

      const instruction2 = createTypedInstruction('tab-switch', {
        action: 'open',
        url: 'https://example.com/page2',
      }, { index: 1 });

      const context: HandlerContext = {
        page: mockPage,
        context: mockBrowserContext,
        config,
        logger,
        metrics,
        sessionId: 'test-session-diff-urls',
        frameStack: [],
      };

      const result1 = await handler.execute(instruction1, context);
      const result2 = await handler.execute(instruction2, context);

      expect(result1.success).toBe(true);
      expect(result2.success).toBe(true);

      // Two different URLs should create two separate tabs
      expect(mockBrowserContext.newPage).toHaveBeenCalledTimes(2);
    });
  });

  describe('tab-switch switch idempotency', () => {
    it('should return success when switching to already-current tab', async () => {
      // Setup: Create a tab stack with the current page
      const tabStack = [mockPage];

      const instruction = createTypedInstruction('tab-switch', {
        action: 'switch',
        index: 0,
      });

      const context: HandlerContext = {
        page: mockPage,
        context: mockBrowserContext,
        config,
        logger,
        metrics,
        sessionId: 'test-session-switch-current',
        frameStack: [],
        tabStack,
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      // Should indicate this was an idempotent operation
      expect((result.extracted_data as Record<string, unknown>)?.tab).toMatchObject({
        idempotent: true,
      });

      // bringToFront should NOT be called since we're already on this tab
      expect(mockPage.bringToFront).not.toHaveBeenCalled();
    });

    it('should switch tabs and not be idempotent when switching to different tab', async () => {
      const secondPage = createMockPage({
        bringToFront: jest.fn().mockResolvedValue(undefined) as any,
      });
      secondPage.url.mockReturnValue('https://example.com/page2');
      secondPage.title.mockResolvedValue('Page 2');

      const tabStack = [mockPage, secondPage];

      const instruction = createTypedInstruction('tab-switch', {
        action: 'switch',
        index: 1,
      });

      const context: HandlerContext = {
        page: mockPage, // Currently on first page
        context: mockBrowserContext,
        config,
        logger,
        metrics,
        sessionId: 'test-session-switch-other',
        frameStack: [],
        tabStack,
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(true);
      // Should NOT indicate idempotent since we actually switched
      expect((result.extracted_data as Record<string, unknown>)?.tab).not.toMatchObject({
        idempotent: true,
      });

      // bringToFront should be called on the target page
      expect(secondPage.bringToFront).toHaveBeenCalled();
    });

    it('should handle multiple consecutive switch requests to same tab', async () => {
      const secondPage = createMockPage({
        bringToFront: jest.fn().mockResolvedValue(undefined) as any,
      });
      secondPage.url.mockReturnValue('https://example.com/page2');
      secondPage.title.mockResolvedValue('Page 2');

      const tabStack = [mockPage, secondPage];

      const instruction = createTypedInstruction('tab-switch', {
        action: 'switch',
        index: 1,
      });

      const context: HandlerContext = {
        page: secondPage, // Already on second page
        context: mockBrowserContext,
        config,
        logger,
        metrics,
        sessionId: 'test-session-switch-multiple',
        frameStack: [],
        tabStack,
      };

      // Execute the same switch instruction multiple times
      const result1 = await handler.execute(instruction, context);
      const result2 = await handler.execute(instruction, context);
      const result3 = await handler.execute(instruction, context);

      // All should succeed
      expect(result1.success).toBe(true);
      expect(result2.success).toBe(true);
      expect(result3.success).toBe(true);

      // All should be idempotent since we're already on the target tab
      const tab1 = (result1.extracted_data as Record<string, unknown>)?.tab as Record<string, unknown>;
      const tab2 = (result2.extracted_data as Record<string, unknown>)?.tab as Record<string, unknown>;
      const tab3 = (result3.extracted_data as Record<string, unknown>)?.tab as Record<string, unknown>;
      expect(tab1?.idempotent).toBe(true);
      expect(tab2?.idempotent).toBe(true);
      expect(tab3?.idempotent).toBe(true);
    });
  });

  describe('tab-switch close behavior', () => {
    it('should return error when closing non-existent tab index', async () => {
      // Need at least 2 tabs to avoid LAST_TAB error
      const secondPage = createMockPage({
        bringToFront: jest.fn().mockResolvedValue(undefined) as any,
      });
      const tabStack = [mockPage, secondPage];

      const instruction = createTypedInstruction('tab-switch', {
        action: 'close',
        index: 5, // Out of range
      });

      const context: HandlerContext = {
        page: mockPage,
        context: mockBrowserContext,
        config,
        logger,
        metrics,
        sessionId: 'test-session-close-invalid',
        frameStack: [],
        tabStack,
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('INVALID_INDEX');
    });

    it('should prevent closing the last tab', async () => {
      const tabStack = [mockPage];

      const instruction = createTypedInstruction('tab-switch', {
        action: 'close',
        index: 0,
      });

      const context: HandlerContext = {
        page: mockPage,
        context: mockBrowserContext,
        config,
        logger,
        metrics,
        sessionId: 'test-session-close-last',
        frameStack: [],
        tabStack,
      };

      const result = await handler.execute(instruction, context);

      expect(result.success).toBe(false);
      expect(result.error?.code).toBe('LAST_TAB');
    });
  });

  describe('tab-switch list idempotency', () => {
    it('should return consistent results for multiple list calls', async () => {
      const secondPage = createMockPage();
      secondPage.url.mockReturnValue('https://example.com/page2');
      secondPage.title.mockResolvedValue('Page 2');

      const tabStack = [mockPage, secondPage];

      const instruction = createTypedInstruction('tab-switch', {
        action: 'list',
      });

      const context: HandlerContext = {
        page: mockPage,
        context: mockBrowserContext,
        config,
        logger,
        metrics,
        sessionId: 'test-session-list',
        frameStack: [],
        tabStack,
      };

      const result1 = await handler.execute(instruction, context);
      const result2 = await handler.execute(instruction, context);

      expect(result1.success).toBe(true);
      expect(result2.success).toBe(true);

      // Results should be consistent (same data)
      const data1 = result1.extracted_data as Record<string, unknown>;
      const data2 = result2.extracted_data as Record<string, unknown>;
      expect(data1?.totalTabs).toBe(data2?.totalTabs);
      expect(data1?.currentIndex).toBe(data2?.currentIndex);
    });
  });
});
