import { SessionCleanup } from '../../../src/session/cleanup';
import { SessionManager } from '../../../src/session/manager';
import { createTestConfig } from '../../helpers';

// Mock playwright
jest.mock('playwright', () => ({
  chromium: {
    launch: jest.fn().mockResolvedValue({
      newContext: jest.fn().mockResolvedValue({
        newPage: jest.fn().mockResolvedValue({
          on: jest.fn(),
          goto: jest.fn(),
          close: jest.fn().mockResolvedValue(undefined),
        }),
        clearCookies: jest.fn().mockResolvedValue(undefined),
        clearPermissions: jest.fn().mockResolvedValue(undefined),
        close: jest.fn().mockResolvedValue(undefined),
      }),
      close: jest.fn().mockResolvedValue(undefined),
      isConnected: jest.fn().mockReturnValue(true),
      version: jest.fn().mockReturnValue('mock-version'),
    }),
  },
}));

describe('SessionCleanup', () => {
  let manager: SessionManager;
  let cleanup: SessionCleanup;
  let config: ReturnType<typeof createTestConfig>;
  let setIntervalSpy: jest.SpyInstance;
  let clearIntervalSpy: jest.SpyInstance;

  beforeEach(() => {
    config = createTestConfig({
      session: {
        maxConcurrent: 10,
        idleTimeoutMs: 100,
        poolSize: 5,
        cleanupIntervalMs: 50,
      },
    });
    manager = new SessionManager(config);
    cleanup = new SessionCleanup(manager, config);

    jest.useFakeTimers();
    setIntervalSpy = jest.spyOn(global, 'setInterval');
    clearIntervalSpy = jest.spyOn(global, 'clearInterval');
  });

  afterEach(async () => {
    cleanup.stop();
    await manager.shutdown();
    setIntervalSpy.mockRestore();
    clearIntervalSpy.mockRestore();
    jest.useRealTimers();
    jest.clearAllMocks();
  });

  describe('start', () => {
    it('should start cleanup interval', () => {
      cleanup.start();

      expect(setIntervalSpy).toHaveBeenCalledWith(expect.any(Function), config.session.cleanupIntervalMs);
    });

    it('should not start multiple intervals', () => {
      cleanup.start();
      cleanup.start();

      expect(setIntervalSpy).toHaveBeenCalledTimes(1);
    });
  });

  describe('stop', () => {
    it('should stop cleanup interval', () => {
      cleanup.start();
      cleanup.stop();

      expect(clearIntervalSpy).toHaveBeenCalled();
    });

    it('should handle stop without start', () => {
      expect(() => cleanup.stop()).not.toThrow();
    });

    it('should handle multiple stops', () => {
      cleanup.start();
      cleanup.stop();
      cleanup.stop();

      expect(() => cleanup.stop()).not.toThrow();
    });
  });

  describe('cleanup execution', () => {
    it('should call cleanupIdleSessions periodically', async () => {
      const cleanupSpy = jest.spyOn(manager, 'cleanupIdleSessions');

      cleanup.start();

      // Fast-forward time by cleanup interval
      jest.advanceTimersByTime(config.session.cleanupIntervalMs);
      await Promise.resolve(); // Let promises settle

      expect(cleanupSpy).toHaveBeenCalled();

      cleanupSpy.mockRestore();
    });

    it('should continue cleanup even if error occurs', async () => {
      const cleanupSpy = jest.spyOn(manager, 'cleanupIdleSessions').mockRejectedValue(new Error('Cleanup failed'));

      cleanup.start();

      // Fast-forward time by cleanup interval
      jest.advanceTimersByTime(config.session.cleanupIntervalMs);
      await Promise.resolve(); // Let promises settle

      // Fast-forward again - should still call cleanup
      jest.advanceTimersByTime(config.session.cleanupIntervalMs);
      await Promise.resolve();

      expect(cleanupSpy).toHaveBeenCalledTimes(2);

      cleanupSpy.mockRestore();
    });

    it('should run multiple cleanup cycles', async () => {
      const cleanupSpy = jest.spyOn(manager, 'cleanupIdleSessions');

      cleanup.start();

      // Fast-forward through 3 cycles
      for (let i = 0; i < 3; i++) {
        jest.advanceTimersByTime(config.session.cleanupIntervalMs);
        await Promise.resolve();
      }

      expect(cleanupSpy).toHaveBeenCalledTimes(3);

      cleanupSpy.mockRestore();
    });
  });

  describe('integration with SessionManager', () => {
    beforeEach(() => {
      // Restore spies before switching to real timers to avoid undefined setInterval
      setIntervalSpy.mockRestore();
      clearIntervalSpy.mockRestore();
      jest.useRealTimers();
    });

    afterEach(() => {
      jest.useFakeTimers();
      // Re-create spies for subsequent tests
      setIntervalSpy = jest.spyOn(global, 'setInterval');
      clearIntervalSpy = jest.spyOn(global, 'clearInterval');
    });

    it('should clean up idle sessions automatically', async () => {
      const realConfig = createTestConfig({
        session: {
          maxConcurrent: 10,
          idleTimeoutMs: 50,
          poolSize: 5,
          cleanupIntervalMs: 25,
        },
      });
      const realManager = new SessionManager(realConfig);
      const realCleanup = new SessionCleanup(realManager, realConfig);

      // Create a session
      const sessionId = await realManager.startSession({
        execution_id: 'exec-123',
        workflow_id: 'workflow-123',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      });

      // Start cleanup
      realCleanup.start();

      // Wait for session to become idle and cleanup to run
      await new Promise((resolve) => setTimeout(resolve, 100));

      // Session should be cleaned up
      const sessions = (realManager as any).sessions;
      expect(sessions.has(sessionId)).toBe(false);

      realCleanup.stop();
      await realManager.shutdown();
    }, 10000);
  });
});
