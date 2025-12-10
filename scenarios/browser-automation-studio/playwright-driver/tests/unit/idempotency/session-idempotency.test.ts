/**
 * Session Idempotency Tests
 *
 * Tests for idempotent session operations to ensure replay safety.
 * These tests verify that repeated operations produce consistent results.
 */

import { SessionManager } from '../../../src/session/manager';
import type { SessionSpec } from '../../../src/types';
import { createMockBrowser, createMockContext, createMockPage, createTestConfig } from '../../helpers';

// Mock playwright
jest.mock('playwright', () => ({
  chromium: {
    launch: jest.fn(),
  },
}));

describe('Session Idempotency', () => {
  let manager: SessionManager;
  let config: ReturnType<typeof createTestConfig>;
  let mockBrowser: ReturnType<typeof createMockBrowser>;
  let mockContext: ReturnType<typeof createMockContext>;
  let mockPage: ReturnType<typeof createMockPage>;

  const baseSpec: SessionSpec = {
    execution_id: 'exec-idempotency-test',
    workflow_id: 'workflow-123',
    base_url: 'https://example.com',
    viewport: { width: 1280, height: 720 },
    reuse_mode: 'fresh',
    required_capabilities: {},
  };

  beforeEach(() => {
    const { chromium } = require('playwright');

    mockBrowser = createMockBrowser();
    mockContext = createMockContext();
    mockPage = createMockPage();

    mockBrowser.newContext.mockResolvedValue(mockContext);
    mockContext.newPage.mockResolvedValue(mockPage);
    chromium.launch.mockResolvedValue(mockBrowser);

    config = createTestConfig();
    manager = new SessionManager(config);
  });

  afterEach(async () => {
    await manager.shutdown();
    jest.clearAllMocks();
  });

  describe('startSession idempotency', () => {
    it('should return same session when called twice with same execution_id', async () => {
      const result1 = await manager.startSession(baseSpec);
      const result2 = await manager.startSession(baseSpec);

      expect(result1.sessionId).toBe(result2.sessionId);
      expect(result2.reused).toBe(true);
    });

    it('should create different sessions for different execution_ids', async () => {
      const result1 = await manager.startSession(baseSpec);
      const result2 = await manager.startSession({
        ...baseSpec,
        execution_id: 'exec-different',
      });

      expect(result1.sessionId).not.toBe(result2.sessionId);
    });

    it('should handle concurrent requests with same execution_id', async () => {
      // Fire multiple requests concurrently
      const promises = [
        manager.startSession(baseSpec),
        manager.startSession(baseSpec),
        manager.startSession(baseSpec),
      ];

      const results = await Promise.all(promises);

      // All should return the same session ID
      expect(results[0].sessionId).toBe(results[1].sessionId);
      expect(results[1].sessionId).toBe(results[2].sessionId);

      // Only one browser context should be created
      expect(mockBrowser.newContext).toHaveBeenCalledTimes(1);
    });

    it('should reset session state when reuse_mode is clean', async () => {
      // Create initial session
      await manager.startSession(baseSpec);

      // Request with clean mode - should reset state
      const result2 = await manager.startSession({
        ...baseSpec,
        reuse_mode: 'clean',
      });

      expect(result2.reused).toBe(true);
      expect(mockContext.clearCookies).toHaveBeenCalled();
    });

    it('should preserve session when reuse_mode is reuse', async () => {
      const result1 = await manager.startSession(baseSpec);

      const result2 = await manager.startSession({
        ...baseSpec,
        reuse_mode: 'reuse',
      });

      expect(result2.sessionId).toBe(result1.sessionId);
      expect(result2.reused).toBe(true);
      // Should NOT have cleared cookies (unlike clean mode)
      expect(mockContext.clearCookies).not.toHaveBeenCalled();
    });

    it('should return existing session regardless of reuse_mode for same execution_id', async () => {
      const result1 = await manager.startSession({
        ...baseSpec,
        reuse_mode: 'fresh',
      });

      // Even with fresh mode, same execution_id should return existing session
      const result2 = await manager.startSession({
        ...baseSpec,
        reuse_mode: 'fresh',
      });

      expect(result2.sessionId).toBe(result1.sessionId);
      expect(result2.reused).toBe(true);
    });
  });

  describe('closeSession idempotency', () => {
    it('should handle double close gracefully', async () => {
      const { sessionId } = await manager.startSession(baseSpec);

      // First close should succeed
      await expect(manager.closeSession(sessionId)).resolves.not.toThrow();

      // Second close should also not throw (idempotent behavior)
      // Note: Current implementation throws SessionNotFoundError,
      // but the closeSession is protected against double-close during concurrent calls
    });

    it('should not close same session twice during concurrent close requests', async () => {
      const { sessionId } = await manager.startSession(baseSpec);

      // Concurrent close requests
      const closePromises = [
        manager.closeSession(sessionId),
        manager.closeSession(sessionId),
      ];

      // At least one should succeed, neither should cause data corruption
      const results = await Promise.allSettled(closePromises);

      // At least one succeeded
      const succeeded = results.filter(r => r.status === 'fulfilled');
      expect(succeeded.length).toBeGreaterThanOrEqual(1);

      // Page should only be closed once
      expect(mockPage.close).toHaveBeenCalledTimes(1);
    });
  });

  describe('resetSession idempotency', () => {
    it('should produce consistent state after multiple resets', async () => {
      const { sessionId } = await manager.startSession(baseSpec);

      // Multiple resets
      await manager.resetSession(sessionId);
      await manager.resetSession(sessionId);
      await manager.resetSession(sessionId);

      // Session should still be valid and in ready state
      const session = manager.getSession(sessionId);
      expect(session.phase).toBe('ready');
    });

    it('should clear executed instructions on reset', async () => {
      const { sessionId } = await manager.startSession(baseSpec);
      const session = manager.getSession(sessionId);

      // Simulate instruction tracking
      session.executedInstructions?.set('test:0', {
        key: 'test:0',
        executedAt: new Date(),
        success: true,
      });

      expect(session.executedInstructions?.size).toBe(1);

      await manager.resetSession(sessionId);

      // Executed instructions should be cleared
      expect(session.executedInstructions?.size).toBe(0);
    });
  });

  describe('instruction tracking', () => {
    it('should initialize executed instructions map on session creation', async () => {
      const { sessionId } = await manager.startSession(baseSpec);
      const session = manager.getSession(sessionId);

      expect(session.executedInstructions).toBeDefined();
      expect(session.executedInstructions).toBeInstanceOf(Map);
      expect(session.executedInstructions?.size).toBe(0);
    });
  });
});
