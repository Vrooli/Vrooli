import { SessionManager } from '../../../src/session/manager';
import type { SessionSpec } from '../../../src/types';
import { SessionNotFoundError, ResourceLimitError } from '../../../src/utils/errors';
import { createMockBrowser, createMockContext, createMockPage, createTestConfig } from '../../helpers';

// Mock playwright
jest.mock('playwright', () => ({
  chromium: {
    launch: jest.fn(),
  },
}));

describe('SessionManager', () => {
  let manager: SessionManager;
  let config: ReturnType<typeof createTestConfig>;
  let mockBrowser: ReturnType<typeof createMockBrowser>;
  let mockContext: ReturnType<typeof createMockContext>;
  let mockPage: ReturnType<typeof createMockPage>;

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

  describe('startSession', () => {
    const sessionSpec: SessionSpec = {
      execution_id: 'exec-123',
      workflow_id: 'workflow-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    };

    it('should create a new session', async () => {
      const result = await manager.startSession(sessionSpec);

      expect(result.sessionId).toBeDefined();
      expect(typeof result.sessionId).toBe('string');
      expect(result.sessionId.length).toBeGreaterThan(0);
      expect(result.reused).toBe(false);
      expect(result.createdAt).toBeInstanceOf(Date);
    });

    it('should launch browser on first session', async () => {
      const { chromium } = require('playwright');

      await manager.startSession(sessionSpec);

      expect(chromium.launch).toHaveBeenCalledWith(
        expect.objectContaining({
          headless: config.browser.headless,
        })
      );
    });

    it('should create browser context', async () => {
      await manager.startSession(sessionSpec);

      expect(mockBrowser.newContext).toHaveBeenCalled();
    });

    it('should create page', async () => {
      await manager.startSession(sessionSpec);

      expect(mockContext.newPage).toHaveBeenCalled();
    });

    it('should throw error when max sessions reached', async () => {
      const configLimited = createTestConfig({
        session: { maxConcurrent: 2, idleTimeoutMs: 300000, poolSize: 5, cleanupIntervalMs: 60000 },
      });
      const limitedManager = new SessionManager(configLimited);

      // Create 2 sessions (max)
      await limitedManager.startSession({ ...sessionSpec, execution_id: 'exec-1' });
      await limitedManager.startSession({ ...sessionSpec, execution_id: 'exec-2' });

      // Try to create 3rd session
      await expect(limitedManager.startSession({ ...sessionSpec, execution_id: 'exec-3' })).rejects.toThrow(
        ResourceLimitError
      );

      await limitedManager.shutdown();
    });

    it('should reuse session with reuse mode', async () => {
      const result1 = await manager.startSession(sessionSpec);

      const reuseSpec: SessionSpec = {
        ...sessionSpec,
        reuse_mode: 'reuse',
      };
      const result2 = await manager.startSession(reuseSpec);

      expect(result2.sessionId).toBe(result1.sessionId);
      expect(result2.reused).toBe(true);
    });

    it('should create new session with fresh mode even if reusable exists', async () => {
      const result1 = await manager.startSession(sessionSpec);

      const freshSpec: SessionSpec = {
        ...sessionSpec,
        reuse_mode: 'fresh',
      };
      const result2 = await manager.startSession(freshSpec);

      expect(result2.sessionId).not.toBe(result1.sessionId);
      expect(result2.reused).toBe(false);
    });

    it('should reset session with clean mode', async () => {
      const result1 = await manager.startSession(sessionSpec);

      const cleanSpec: SessionSpec = {
        ...sessionSpec,
        reuse_mode: 'clean',
      };
      const result2 = await manager.startSession(cleanSpec);

      expect(result2.sessionId).toBe(result1.sessionId);
      expect(result2.reused).toBe(true);
      expect(mockContext.clearCookies).toHaveBeenCalled();
    });

    it('should set creation time on session', async () => {
      const before = Date.now();
      const result = await manager.startSession(sessionSpec);
      const after = Date.now();

      const session = manager.getSession(result.sessionId);
      expect(session.createdAt.getTime()).toBeGreaterThanOrEqual(before);
      expect(session.createdAt.getTime()).toBeLessThanOrEqual(after);
    });
  });

  describe('getSession', () => {
    it('should return session by ID', async () => {
      const spec: SessionSpec = {
        execution_id: 'exec-123',
        workflow_id: 'workflow-123',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };
      const { sessionId } = await manager.startSession(spec);

      const session = manager.getSession(sessionId);

      expect(session).toBeDefined();
      expect(session.id).toBe(sessionId);
    });

    it('should throw error for non-existent session', () => {
      expect(() => manager.getSession('non-existent')).toThrow(SessionNotFoundError);
    });
  });

  describe('resetSession', () => {
    it('should clear cookies and permissions', async () => {
      const spec: SessionSpec = {
        execution_id: 'exec-123',
        workflow_id: 'workflow-123',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };
      const { sessionId } = await manager.startSession(spec);

      await manager.resetSession(sessionId);

      expect(mockContext.clearCookies).toHaveBeenCalled();
      expect(mockContext.clearPermissions).toHaveBeenCalled();
    });

    it('should update last used time', async () => {
      const spec: SessionSpec = {
        execution_id: 'exec-123',
        workflow_id: 'workflow-123',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };
      const { sessionId } = await manager.startSession(spec);

      // Wait a bit
      await new Promise((resolve) => setTimeout(resolve, 10));

      const before = Date.now();
      await manager.resetSession(sessionId);
      const after = Date.now();

      const session = manager.getSession(sessionId);
      expect(session.lastUsedAt.getTime()).toBeGreaterThanOrEqual(before);
      // Allow small scheduler drift to avoid flakiness
      expect(session.lastUsedAt.getTime()).toBeLessThanOrEqual(after + 5);
    });

    it('should throw error for non-existent session', async () => {
      await expect(manager.resetSession('non-existent')).rejects.toThrow(SessionNotFoundError);
    });
  });

  describe('closeSession', () => {
    it('should close session and remove from map', async () => {
      const spec: SessionSpec = {
        execution_id: 'exec-123',
        workflow_id: 'workflow-123',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };
      const { sessionId } = await manager.startSession(spec);

      await manager.closeSession(sessionId);

      expect(() => manager.getSession(sessionId)).toThrow(SessionNotFoundError);
    });

    it('should close page', async () => {
      const spec: SessionSpec = {
        execution_id: 'exec-123',
        workflow_id: 'workflow-123',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };
      const { sessionId } = await manager.startSession(spec);

      await manager.closeSession(sessionId);

      expect(mockPage.close).toHaveBeenCalled();
    });

    it('should close context', async () => {
      const spec: SessionSpec = {
        execution_id: 'exec-123',
        workflow_id: 'workflow-123',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };
      const { sessionId } = await manager.startSession(spec);

      await manager.closeSession(sessionId);

      expect(mockContext.close).toHaveBeenCalled();
    });

    it('should throw error for non-existent session', async () => {
      await expect(manager.closeSession('non-existent')).rejects.toThrow(SessionNotFoundError);
    });

    it('should handle already closed page gracefully', async () => {
      const spec: SessionSpec = {
        execution_id: 'exec-123',
        workflow_id: 'workflow-123',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };
      const { sessionId } = await manager.startSession(spec);

      mockPage.close.mockRejectedValueOnce(new Error('Page already closed'));

      await expect(manager.closeSession(sessionId)).resolves.not.toThrow();
    });
  });

  describe('cleanupIdleSessions', () => {
    it('should close idle sessions', async () => {
      const configShortIdle = createTestConfig({
        session: { maxConcurrent: 10, idleTimeoutMs: 100, poolSize: 5, cleanupIntervalMs: 50 },
      });
      const managerShortIdle = new SessionManager(configShortIdle);

      const spec: SessionSpec = {
        execution_id: 'exec-123',
        workflow_id: 'workflow-123',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };
      const { sessionId } = await managerShortIdle.startSession(spec);

      // Wait for session to become idle
      await new Promise((resolve) => setTimeout(resolve, 150));

      await managerShortIdle.cleanupIdleSessions();

      expect(() => managerShortIdle.getSession(sessionId)).toThrow(SessionNotFoundError);

      await managerShortIdle.shutdown();
    });

    it('should not close active sessions', async () => {
      const spec: SessionSpec = {
        execution_id: 'exec-123',
        workflow_id: 'workflow-123',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };
      const { sessionId } = await manager.startSession(spec);

      await manager.cleanupIdleSessions();

      expect(() => manager.getSession(sessionId)).not.toThrow();
    });
  });

  describe('shutdown', () => {
    it('should close all sessions', async () => {
      const spec1: SessionSpec = {
        execution_id: 'exec-1',
        workflow_id: 'workflow-1',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };
      const spec2: SessionSpec = {
        execution_id: 'exec-2',
        workflow_id: 'workflow-2',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };

      const { sessionId: sessionId1 } = await manager.startSession(spec1);
      const { sessionId: sessionId2 } = await manager.startSession(spec2);

      await manager.shutdown();

      expect(() => manager.getSession(sessionId1)).toThrow(SessionNotFoundError);
      expect(() => manager.getSession(sessionId2)).toThrow(SessionNotFoundError);
    });

    it('should close browser', async () => {
      const spec: SessionSpec = {
        execution_id: 'exec-123',
        workflow_id: 'workflow-123',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };
      await manager.startSession(spec);

      await manager.shutdown();

      expect(mockBrowser.close).toHaveBeenCalled();
    });

    it('should handle multiple shutdowns gracefully', async () => {
      await manager.shutdown();
      await expect(manager.shutdown()).resolves.not.toThrow();
    });
  });

  describe('updateActivity', () => {
    it('should update last used time', async () => {
      const spec: SessionSpec = {
        execution_id: 'exec-123',
        workflow_id: 'workflow-123',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'fresh',
        required_capabilities: {},
      };
      const { sessionId } = await manager.startSession(spec);

      // Wait a bit
      await new Promise((resolve) => setTimeout(resolve, 10));

      const before = Date.now();
      manager.updateActivity(sessionId);
      const after = Date.now();

      const session = manager.getSession(sessionId);
      expect(session.lastUsedAt.getTime()).toBeGreaterThanOrEqual(before);
      expect(session.lastUsedAt.getTime()).toBeLessThanOrEqual(after);
    });

    it('should not throw for non-existent session', () => {
      expect(() => manager.updateActivity('non-existent')).not.toThrow();
    });
  });
});
