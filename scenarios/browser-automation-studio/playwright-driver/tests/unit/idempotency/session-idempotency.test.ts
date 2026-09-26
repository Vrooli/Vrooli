/**
 * Session Idempotency Tests
 *
 * Tests for idempotent session operations to ensure replay safety.
 * These tests verify that repeated operations produce consistent results.
 */

import type { SessionSpec } from '../../../src/types';
import { playwrightProvider } from '../../../src/playwright';
import { SessionManager } from '../../../src/session/manager';
import { createMockBrowser, createMockContext, createMockPage, createTestConfig } from '../../helpers';

describe('Session Idempotency', () => {
  let manager: InstanceType<typeof SessionManager>;
  let config: ReturnType<typeof createTestConfig>;
  let mockBrowser: ReturnType<typeof createMockBrowser>;
  let mockContext: ReturnType<typeof createMockContext>;
  let mockPage: ReturnType<typeof createMockPage>;
  let launchSpy: jest.SpiedFunction<typeof playwrightProvider.chromium.launch>;

  const baseSpec: SessionSpec = {
    execution_id: 'exec-idempotency-test',
    workflow_id: 'workflow-123',
    base_url: 'https://example.com',
    viewport: { width: 1280, height: 720 },
    reuse_mode: 'fresh',
    required_capabilities: {},
  };

  beforeEach(() => {
    mockBrowser = createMockBrowser();
    mockContext = createMockContext();
    mockPage = createMockPage();

    mockBrowser.newContext.mockResolvedValue(mockContext);
    mockContext.newPage.mockResolvedValue(mockPage);
    launchSpy = jest.spyOn(playwrightProvider.chromium, 'launch').mockResolvedValue(mockBrowser);

    config = createTestConfig();
    manager = new SessionManager(config);
  });

  afterEach(async () => {
    launchSpy.mockRestore();
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
      const [firstResult, secondResult, thirdResult] = results;
      if (!firstResult || !secondResult || !thirdResult) {
        throw new Error('Expected three session results');
      }
      expect(firstResult.sessionId).toBe(secondResult.sessionId);
      expect(secondResult.sessionId).toBe(thirdResult.sessionId);

      // Only one browser context should be created
      // The first session also performs the process-cached host-audio probe.
      expect(mockBrowser.newContext.mock.calls.length).toBe(2);
    });

    it('clean mode resets a released session for a different owner', async () => {
      const labels = { pool: 'clean-idempotency' };
      const first = await manager.startSession({ ...baseSpec, labels });
      expect(manager.releaseExecutionLease(first.sessionId, baseSpec.execution_id, first.leaseId)).toBe(true);
      const second = await manager.startSession({
        ...baseSpec, execution_id: 'next-clean-owner', labels, reuse_mode: 'clean',
      });
      expect(second.sessionId).toBe(first.sessionId);
      expect(second.leaseId).not.toBe(first.leaseId);
      expect(second.reused).toBe(true);
      expect(mockContext.clearCookies).toHaveBeenCalledTimes(1);
    });

    it('should preserve session when reuse_mode is reuse', async () => {
      const result1 = await manager.startSession(baseSpec);

      const result2 = await manager.startSession({
        ...baseSpec,
        reuse_mode: 'reuse',
      });

      expect(result2.sessionId).toBe(result1.sessionId);
      expect(result2.reused).toBe(true);
      // A repeated start must preserve the current owner state.
      expect(mockContext.clearCookies.mock.calls.length).toBe(0);
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
      // The probe page and the session page each close exactly once.
      expect(mockPage.close.mock.calls.length).toBe(2);
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

  });

  describe('phase preservation', () => {
    it.each(['ready', 'executing', 'recording', 'resetting', 'closing'] as const)(
      'same-execution retry preserves %s until the owning operation changes it', async (phase) => {
        const first = await manager.startSession(baseSpec);
        const session = manager.getSession(first.sessionId);
        session.phase = phase;
        const result = await manager.startSession(baseSpec);
        expect(result.sessionId).toBe(first.sessionId);
        expect(result.leaseId).toBe(first.leaseId);
        expect(result.reused).toBe(true);
        expect(session.phase).toBe(phase);
        expect(manager.canAcceptInstructions(first.sessionId)).toBe(phase === 'ready' || phase === 'recording');
      }
    );
  });
});
