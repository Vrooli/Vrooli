import { moveVideo } from '../../../src/session/session-teardown';
import * as verification from '../../../src/recording/validation/verification';
import { initRecordingBuffer, bufferTimelineEntry, getTimelineEntries, acknowledgeTimelineEntries } from '../../../src/recording';
import { promises as fs } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { handleSessionClose } from '../../../src/routes/session-close';
import type { SessionSpec } from '../../../src/types';
import { playwrightProvider } from '../../../src/playwright';
import { SessionManager } from '../../../src/session/manager';
import { ServiceWorkerController } from '../../../src/service-worker';
import { SessionNotFoundError, ResourceLimitError } from '../../../src/utils/errors';
import { createMockBrowser, createMockContext, createMockPage, createTestConfig, createMockHttpRequest, createMockHttpResponse } from '../../helpers';

describe('SessionManager', () => {
  let manager: InstanceType<typeof SessionManager>;
  let config: ReturnType<typeof createTestConfig>;
  let mockBrowser: ReturnType<typeof createMockBrowser>;
  let mockContext: ReturnType<typeof createMockContext>;
  let mockPage: ReturnType<typeof createMockPage>;
  let launchSpy: jest.SpiedFunction<typeof playwrightProvider.chromium.launch>;
  let readinessSpy: jest.SpiedFunction<typeof verification.waitForScriptReady>;

  beforeEach(() => {
    // These manager tests use Page doubles without a recording script. Native
    // readiness and close behavior are covered by pipeline-e2e.test.ts.
    readinessSpy = jest.spyOn(verification, 'waitForScriptReady').mockResolvedValue({
      loaded: true, ready: true, inMainContext: true, handlersCount: 1,
      loadTime: 1, version: 'fixture',
    });
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
    if (manager) {
      await manager.shutdown();
    }
    launchSpy.mockRestore();
    readinessSpy.mockRestore();
    jest.clearAllMocks();
  });

  describe('page callback disposal', () => {
    it.each(['reset', 'close'] as const)('disposes page callbacks before session %s browser effects', async (operation) => {
      const { sessionId } = await manager.startSession({
        execution_id: `callback-${operation}`, viewport: { width: 800, height: 600 },
        reuse_mode: 'fresh', required_capabilities: {},
      });
      const session = manager.getSession(sessionId);
      let active = true;
      const cleanup = jest.fn(() => { active = false; });
      session.pageLifecycleCleanup = cleanup;
      const browserEffects: boolean[] = [];
      mockPage.goto.mockImplementation(async () => { browserEffects.push(active); return null; });
      mockPage.close.mockImplementation(async () => { browserEffects.push(active); });
      if (operation === 'reset') await manager.resetSession(sessionId);
      else await manager.closeSession(sessionId);
      expect(cleanup).toHaveBeenCalledTimes(1);
      expect(session.pageLifecycleCleanup).toBeUndefined();
      expect(browserEffects.length).toBeGreaterThan(0);
      expect(browserEffects.every((callbacksActive) => !callbacksActive)).toBe(true);
    });

    it('retains failed callback disposal for explicit close retry', async () => {
      const { sessionId } = await manager.startSession({
        execution_id: 'callback-retry', viewport: { width: 800, height: 600 },
        reuse_mode: 'fresh', required_capabilities: {},
      });
      const session = manager.getSession(sessionId);
      const cleanup = jest.fn().mockImplementationOnce(() => { throw new Error('callback disposal fault'); });
      session.pageLifecycleCleanup = cleanup;
      const closesBefore = mockContext.close.mock.calls.length;
      await expect(manager.closeSession(sessionId)).rejects.toThrow('callback disposal fault');
      expect(session.pageLifecycleCleanup).toBe(cleanup);
      expect(mockContext.close).toHaveBeenCalledTimes(closesBefore);
      await manager.closeSession(sessionId);
      expect(cleanup).toHaveBeenCalledTimes(2);
      expect(session.pageLifecycleCleanup).toBeUndefined();
    });
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

    describe('released-session context compatibility', () => {
      const original: SessionSpec = {
        execution_id: 'profile-owner',
        workflow_id: 'profile-workflow',
        base_url: 'https://example.com',
        viewport: { width: 1280, height: 720 },
        reuse_mode: 'reuse',
        session_profile_version: 'profile-a@1',
        storage_state: { cookies: [{ name: 'identity', value: 'first' }], origins: [] },
        labels: { pool: 'shared' },
        required_capabilities: {},
      };

      it.each([
        ['profile identity or revision', (spec: SessionSpec) => ({ ...spec, session_profile_version: 'profile-b@1' })],
        ['clean mode with another profile', (spec: SessionSpec) => ({ ...spec, reuse_mode: 'clean' as const, session_profile_version: 'profile-b@1' })],
        ['storage snapshot', (spec: SessionSpec) => ({ ...spec, storage_state: { cookies: [{ name: 'identity', value: 'second' }], origins: [] } })],
        ['viewport', (spec: SessionSpec) => ({ ...spec, viewport: { width: 390, height: 844 } })],
      ])('does not pool a released session across a changed %s', async (_change, change) => {
        const first = await manager.startSession(original);
        expect(manager.releaseExecutionLease(first.sessionId, original.execution_id, first.leaseId)).toBe(true);

        const requested = change({
          ...original,
          execution_id: 'different-owner',
          workflow_id: 'different-workflow',
        });
        const second = await manager.startSession(requested);

        expect(second.reused).toBe(false);
        expect(second.sessionId).not.toBe(first.sessionId);
        expect(mockBrowser.newContext).toHaveBeenCalled();
        const lastContextOptions = mockBrowser.newContext.mock.calls.at(-1)?.[0];
        expect(lastContextOptions).toEqual(expect.objectContaining({
          viewport: requested.viewport,
          storageState: requested.storage_state,
        }));
        if (requested.viewport.width <= 480) {
          expect(lastContextOptions).toEqual(expect.objectContaining({ isMobile: true, hasTouch: true }));
        }
      });

      it('pools a released session when profile identity and context inputs match', async () => {
        const first = await manager.startSession(original);
        expect(manager.releaseExecutionLease(first.sessionId, original.execution_id, first.leaseId)).toBe(true);

        const second = await manager.startSession({
          ...original,
          execution_id: 'same-profile-owner',
          workflow_id: 'same-profile-workflow',
        });

        expect(second.reused).toBe(true);
        expect(second.sessionId).toBe(first.sessionId);
      });
    });

    it('should launch browser on first session', async () => {
      await manager.startSession(sessionSpec);

      const launchCalls = launchSpy.mock.calls as Array<[Record<string, unknown>]>;
      expect(launchCalls[0]?.[0]).toEqual(
        expect.objectContaining({
          headless: config.browser.headless,
        })
      );
    });

    it('should create browser context', async () => {
      await manager.startSession(sessionSpec);

      expect(mockBrowser.newContext.mock.calls.length).toBeGreaterThan(0);
    });

    it('should create page', async () => {
      await manager.startSession(sessionSpec);

      expect(mockContext.newPage.mock.calls.length).toBeGreaterThan(0);
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

    it('reserves capacity while a distinct session is still being created', async () => {
      const limitedManager = new SessionManager(createTestConfig({
        session: { maxConcurrent: 1, idleTimeoutMs: 300000, poolSize: 5, cleanupIntervalMs: 60000 },
      }));
      let enteredCreate!: () => void;
      let releaseCreate!: () => void;
      const entered = new Promise<void>((resolve) => { enteredCreate = resolve; });
      const creationGate = new Promise<void>((resolve) => { releaseCreate = resolve; });
      mockBrowser.newContext.mockImplementation(async () => {
        enteredCreate();
        await creationGate;
        return mockContext;
      });

      const firstStart = limitedManager.startSession({ ...sessionSpec, execution_id: 'capacity-first' });
      await entered;
      const secondStart = limitedManager.startSession({ ...sessionSpec, execution_id: 'capacity-second' });
      releaseCreate();

      const [first, second] = await Promise.allSettled([firstStart, secondStart]);
      try {
        expect(first.status).toBe('fulfilled');
        expect(second.status).toBe('rejected');
        if (second.status === 'rejected') expect(second.reason).toBeInstanceOf(ResourceLimitError);
        expect(limitedManager.getSessionCount()).toBe(1);
      } finally {
        await limitedManager.shutdown();
      }
    });

    it('releases a capacity reservation when browser context creation fails', async () => {
      const limitedManager = new SessionManager(createTestConfig({
        session: { maxConcurrent: 1, idleTimeoutMs: 300000, poolSize: 5, cleanupIntervalMs: 60000 },
      }));
      // The first context belongs to BrowserManager's audio-capability probe;
      // fail the subsequent context created for the session itself.
      mockBrowser.newContext
        .mockReturnValueOnce(Promise.resolve(mockContext))
        .mockRejectedValueOnce(new Error('context creation failed'));

      await expect(limitedManager.startSession({ ...sessionSpec, execution_id: 'capacity-failed' }))
        .rejects.toThrow('context creation failed');
      const next = await limitedManager.startSession({ ...sessionSpec, execution_id: 'capacity-retry' });

      expect(next.sessionId).toBeTruthy();
      expect(limitedManager.getSessionCount()).toBe(1);
      await limitedManager.shutdown();
    });

    it.each(['ready', 'executing', 'recording', 'resetting', 'closing'] as const)(
      'same-execution clean retry preserves %s state and browser data', async (phase) => {
        const first = await manager.startSession(sessionSpec);
        const session = manager.getSession(first.sessionId);
        session.phase = phase;
        session.instructionReceipts?.set(1, { fingerprint: 'prior-effect', response: '{"success":true}' });
        session.lastInstructionSequence = 1;
        const navigationCount = mockPage.goto.mock.calls.length;
        const retry = await manager.startSession({ ...sessionSpec, reuse_mode: 'clean' });
        expect(retry.sessionId).toBe(first.sessionId);
        expect(retry.leaseId).toBe(first.leaseId);
        expect(session.phase).toBe(phase);
        expect(mockPage.goto).toHaveBeenCalledTimes(navigationCount);
        expect(mockContext.clearCookies).not.toHaveBeenCalled();
        expect(session.instructionReceipts?.has(1)).toBe(true);
        expect(session.lastInstructionSequence).toBe(1);
      }
    );

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

    it('should return existing session with fresh mode when execution_id matches (idempotent)', async () => {
      // Idempotency: Same execution_id always returns existing session,
      // regardless of reuse_mode, to ensure replay safety
      const result1 = await manager.startSession(sessionSpec);

      const freshSpec: SessionSpec = {
        ...sessionSpec,
        reuse_mode: 'fresh',
      };
      const result2 = await manager.startSession(freshSpec);

      // Same execution_id => same session (idempotent behavior)
      expect(result2.sessionId).toBe(result1.sessionId);
      expect(result2.reused).toBe(true);
    });

    it('should create new session with fresh mode when execution_id differs', async () => {
      const result1 = await manager.startSession(sessionSpec);

      const freshSpec: SessionSpec = {
        ...sessionSpec,
        execution_id: 'different-execution-id',
        reuse_mode: 'fresh',
      };
      const result2 = await manager.startSession(freshSpec);

      // Different execution_id => new session
      expect(result2.sessionId).not.toBe(result1.sessionId);
      expect(result2.reused).toBe(false);
    });

    it('resets an explicitly released session for a new clean owner', async () => {
      const labels = { pool: 'clean-session' };
      const first = await manager.startSession({ ...sessionSpec, labels });
      expect(manager.releaseExecutionLease(first.sessionId, sessionSpec.execution_id, first.leaseId)).toBe(true);
      const second = await manager.startSession({ ...sessionSpec, execution_id: 'new-owner', labels, reuse_mode: 'clean' });
      expect(second.sessionId).toBe(first.sessionId);
      expect(second.leaseId).not.toBe(first.leaseId);
      expect(second.reused).toBe(true);
      expect(mockContext.clearCookies).toHaveBeenCalledTimes(1);
    });

    it('does not hand an active lease to a different execution by labels', async () => {
      const first = await manager.startSession({
        ...sessionSpec,
        execution_id: 'exec-original',
        labels: { suite: 'playbook', scenario: 'web-console' },
      });
      const existingSession = manager.getSession(first.sessionId);
      existingSession.instructionReceipts?.set(1, { fingerprint: 'prior-effect', response: '{"success":true}' });
      existingSession.lastInstructionSequence = 1;
      existingSession.instructionCount = 5;

      const second = await manager.startSession({
        ...sessionSpec,
        execution_id: 'exec-new',
        workflow_id: 'workflow-new',
        reuse_mode: 'reuse',
        labels: { suite: 'playbook', scenario: 'web-console' },
      });

      expect(second.sessionId).not.toBe(first.sessionId);
      expect(second.reused).toBe(false);
      expect(manager.getSession(first.sessionId).ownerExecutionId).toBe('exec-original');
    });

    it('reuses a label only after the owner releases its exact lease', async () => {
      const first = await manager.startSession({ ...sessionSpec, execution_id: 'exec-original', labels: { suite: 'playbook' } });
      const original = manager.getSession(first.sessionId);
      const originalLeaseID = original.leaseId;
      original.lastInstructionSequence = 9;
      original.instructionReceipts?.set(9, { fingerprint: 'old-effect', response: '{"success":true}' });
      expect(manager.releaseExecutionLease(first.sessionId, 'wrong-owner', originalLeaseID)).toBe(false);
      expect(manager.releaseExecutionLease(first.sessionId, 'exec-original', originalLeaseID)).toBe(true);

      const second = await manager.startSession({ ...sessionSpec, execution_id: 'exec-new', reuse_mode: 'reuse', labels: { suite: 'playbook' } });
      expect(second.sessionId).toBe(first.sessionId);
      const reassigned = manager.getSession(second.sessionId);
      expect(reassigned.ownerExecutionId).toBe('exec-new');
      expect(reassigned.lastInstructionSequence).toBe(0);
      expect(reassigned.instructionReceipts?.size).toBe(0);
      expect(reassigned.leaseId).not.toBe(originalLeaseID);
      await expect(manager.closeSessionForLease(first.sessionId, 'exec-original', originalLeaseID)).rejects.toThrow(SessionNotFoundError);
    });

    it('should set creation time on session', async () => {
      const before = Date.now();
      const result = await manager.startSession(sessionSpec);
      const after = Date.now();

      const session = manager.getSession(result.sessionId);
      expect(session.createdAt.getTime()).toBeGreaterThanOrEqual(before);
      expect(session.createdAt.getTime()).toBeLessThanOrEqual(after);
    });

    it('cleans up a session already inserted into the map when service-worker initialization fails', async () => {
      const enable = jest
        .spyOn(ServiceWorkerController.prototype, 'enable')
        .mockRejectedValueOnce(new Error('service-worker initialization failed'));

      await expect(manager.startSession(sessionSpec)).rejects.toThrow('service-worker initialization failed');

      expect(enable).toHaveBeenCalledTimes(1);
      expect(manager.getSessionCount()).toBe(0);
      // The cached host-audio probe owns a separate context. The failed
      // session also closes its page before closing its browser context.
      expect(mockContext.close).toHaveBeenCalledTimes(3);
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

  describe('readiness deadline ownership', () => {
    it.each(['ready', 'not-ready', 'rejected', 'timeout'] as const)(
      'releases the deadline after %s', async (outcome) => {
        const { sessionId } = await manager.startSession({
          execution_id: 'readiness-deadline', workflow_id: 'readiness',
          base_url: 'https://example.com', viewport: { width: 640, height: 480 },
          reuse_mode: 'fresh', required_capabilities: {},
        });
        const session = manager.peekSession(sessionId);
        await session.pipelineReadyPromise;
        jest.spyOn(session.pipelineManager!, 'isReady').mockReturnValue(false);
        let resolve!: (value: boolean) => void;
        let reject!: (error: Error) => void;
        session.pipelineReadyPromise = new Promise((yes, no) => { resolve = yes; reject = no; });
        jest.useFakeTimers();
        try {
          const waiting = manager.waitForPipelineReady(sessionId, 25);
          expect(jest.getTimerCount()).toBe(1);
          if (outcome === 'timeout') await jest.advanceTimersByTimeAsync(25);
          else if (outcome === 'rejected') reject(new Error('injection failed'));
          else resolve(outcome === 'ready');
          expect(await waiting).toBe(outcome === 'ready');
          expect(jest.getTimerCount()).toBe(0);
        } finally {
          resolve(false);
          jest.clearAllTimers();
          jest.useRealTimers();
        }
      },
    );
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

      expect(mockContext.clearCookies.mock.calls.length).toBeGreaterThan(0);
      expect(mockContext.clearPermissions.mock.calls.length).toBeGreaterThan(0);
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


    async function resetFixture() {
      const { sessionId } = await manager.startSession({
        execution_id: 'reset-fault-owner', workflow_id: 'reset-fixture',
        viewport: { width: 640, height: 480 }, reuse_mode: 'fresh', required_capabilities: {},
      });
      return { sessionId, session: manager.peekSession(sessionId) };
    }

    it('reserves reset before an asynchronous recording flush and joins all callers', async () => {
      const { sessionId, session } = await resetFixture();
      let finish!: () => void;
      const pending = new Promise<void>((resolve) => { finish = resolve; });
      const stop = jest.fn().mockReturnValue(pending);
      session.pipelineManager = { isRecording: () => true, stopRecording: stop } as never;
      const first = manager.resetSession(sessionId);
      const second = manager.resetSession(sessionId);
      expect(session.phase).toBe('resetting');
      expect(manager.canAcceptInstructions(sessionId)).toBe(false);
      expect(stop).toHaveBeenCalledTimes(1);
      expect(mockContext.clearCookies).not.toHaveBeenCalled();
      finish();
      await Promise.all([first, second]);
      expect(session.phase).toBe('ready');
      expect(mockContext.clearCookies).toHaveBeenCalledTimes(1);
    });

    it.each([false, true])('joins a pending reset before close even when reset fails=%s', async (fail) => {
      const { sessionId, session } = await resetFixture();
      let finish!: () => void;
      let began!: () => void;
      const started = new Promise<void>((resolve) => { began = resolve; });
      mockPage.goto.mockImplementationOnce(async () => {
        began(); await new Promise<void>((resolve) => { finish = resolve; });
        if (fail) throw new Error('reset navigation rejected');
        return null;
      });
      mockContext.close.mockClear(); // Exclude the browser capability probe.
      const reset = manager.resetSession(sessionId);
      await started;
      const close = manager.closeSession(sessionId);
      try {
        expect(session.phase).toBe('closing');
        expect(mockContext.close).not.toHaveBeenCalled();
      } finally {
        finish();
        const settled = await Promise.allSettled([reset, close]);
        expect(settled[0]!.status).toBe(fail ? 'rejected' : 'fulfilled');
        expect(settled[1]!.status).toBe('fulfilled');
      }
      expect(session.phase).toBe('closing');
      expect(mockContext.close).toHaveBeenCalledTimes(1);
      expect(manager.getSessionCount()).toBe(0);
    });

    it('retains the origin inventory after a partial clear and permits explicit retry', async () => {
      const { sessionId, session } = await resetFixture();
      const cdp = { send: jest.fn().mockResolvedValue({}), detach: jest.fn().mockResolvedValue(undefined) };
      mockContext.newCDPSession = jest.fn().mockResolvedValue(cdp);
      session.storageOrigins.add('https://owned.test');
      session.lastInstructionSequence = 11;
      mockContext.clearCookies.mockRejectedValueOnce(new Error('cookie clear rejected'));
      await expect(manager.resetSession(sessionId)).rejects.toThrow('cookie clear rejected');
      expect(session.phase).toBe('resetting');
      expect(session.storageOrigins).toEqual(new Set(['https://owned.test']));
      expect(session.lastInstructionSequence).toBe(11);
      await manager.resetSession(sessionId);
      expect(session.phase).toBe('ready');
      expect(session.storageOrigins.size).toBe(0);
      expect(session.lastInstructionSequence).toBe(11);
      expect(cdp.send).toHaveBeenCalledTimes(2);
      expect(cdp.detach).toHaveBeenCalledTimes(2);
    });

    it('retains recovery ownership when an extra page refuses to close', async () => {
      const { sessionId, session } = await resetFixture();
      const other = createMockPage();
      other.close.mockRejectedValueOnce(new Error('page close rejected'));
      mockContext.pages.mockReturnValue([mockPage, other]);
      session.pages.push(other);
      await expect(manager.resetSession(sessionId)).rejects.toThrow('page close rejected');
      expect(session.phase).toBe('resetting');
      expect(session.pages).toContain(other);
      expect(mockContext.clearCookies).not.toHaveBeenCalled();
      await manager.resetSession(sessionId);
      expect(session.phase).toBe('ready');
      expect(session.pages).toEqual([mockPage]);
      expect(other.close).toHaveBeenCalledTimes(2);
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

      expect(mockPage.close.mock.calls.length).toBeGreaterThan(0);
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

      expect(mockContext.close.mock.calls.length).toBeGreaterThan(0);
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

      mockPage.isClosed.mockReturnValue(true);

      await expect(manager.closeSession(sessionId)).resolves.not.toThrow();
    });
  });

  describe('close recovery ownership', () => {
    const create = async () => {
      const created = await manager.startSession({
        execution_id: 'close-recovery', workflow_id: 'fixture', base_url: 'about:blank',
        viewport: { width: 800, height: 600 }, reuse_mode: 'fresh', required_capabilities: {},
      });
      // The session start performs a host-audio capability probe using these
      // same fake handles. Measure teardown calls only after that completes.
      mockPage.close.mockClear();
      mockContext.close.mockClear();
      return created;
    };
    const close = async (sessionId: string, leaseId: string) => {
      const res = createMockHttpResponse();
      await handleSessionClose(createMockHttpRequest({ body: { execution_id: 'close-recovery', lease_id: leaseId } }), res, sessionId, manager);
      return res;
    };

    it('reports a trace flush failure and retains the context for an explicit retry', async () => {
      const { sessionId, leaseId } = await create();
      const session = manager.getSession(sessionId);
      const dir = await fs.mkdtemp(path.join(os.tmpdir(), 'bas-close-trace-'));
      try {
        session.tracing = true;
        session.tracePath = path.join(dir, 'trace.zip');
        mockContext.tracing.stop.mockRejectedValueOnce(new Error('trace flush fault'));
        mockContext.tracing.stop.mockImplementationOnce(async () => { await fs.writeFile(session.tracePath!, 'retained trace'); });
        const failed = await close(sessionId, leaseId);
        expect(failed.statusCode).toBe(500);
        expect(manager.peekSession(sessionId).phase).toBe('closing');
        expect(manager.canAcceptInstructions(sessionId)).toBe(false);
        expect(mockContext.close).not.toHaveBeenCalled();
        const retried = await close(sessionId, leaseId);
        expect(retried.statusCode).toBe(200);
        expect(retried.getJSON().trace_path).toBe(session.tracePath);
        expect(await fs.readFile(session.tracePath!, 'utf8')).toBe('retained trace');
        expect(() => manager.peekSession(sessionId)).toThrow(SessionNotFoundError);
      } finally {
        await fs.rm(dir, { recursive: true, force: true });
      }
    });

    it('retains context failure and does not repeat completed audio/page cleanup on retry', async () => {
      const { sessionId, leaseId } = await create();
      const session = manager.getSession(sessionId);
      const stopAudio = jest.fn().mockResolvedValue(undefined);
      session.audioPlaybackStop = stopAudio;
      mockContext.close.mockRejectedValueOnce(new Error('context close fault'));
      const failed = await close(sessionId, leaseId);
      expect(failed.statusCode).toBe(500);
      expect(manager.peekSession(sessionId).phase).toBe('closing');
      expect(stopAudio).toHaveBeenCalledTimes(1);
      expect(mockPage.close).toHaveBeenCalledTimes(1);
      const retried = await close(sessionId, leaseId);
      expect(retried.statusCode).toBe(200);
      expect(stopAudio).toHaveBeenCalledTimes(1);
      expect(mockPage.close).toHaveBeenCalledTimes(1);
      expect(mockContext.close).toHaveBeenCalledTimes(2);
    });

    it.each(['audio', 'page', 'service-worker', 'cdp'] as const)(
      '%s cleanup failure reaches HTTP and retains the exact lease for retry', async (stage) => {
        const { sessionId, leaseId } = await create();
        const session = manager.getSession(sessionId);
        const failing = jest.fn().mockRejectedValueOnce(new Error(`${stage} fault`)).mockResolvedValue(undefined);
        if (stage === 'audio') session.audioPlaybackStop = failing;
        if (stage === 'page') mockPage.close.mockImplementation(failing);
        if (stage === 'service-worker') session.serviceWorkerController = { disable: failing } as never;
        if (stage === 'cdp') { session.externalTarget = true; mockBrowser.close.mockImplementation(failing); }
        expect((await close(sessionId, leaseId)).statusCode).toBe(500);
        expect(manager.peekSession(sessionId)).toBe(session);
        expect(session.phase).toBe('closing');
        expect(mockContext.close).not.toHaveBeenCalled();
        expect((await close(sessionId, 'stale-lease')).statusCode).toBe(404);
        expect(failing).toHaveBeenCalledTimes(1);
        expect((await close(sessionId, leaseId)).statusCode).toBe(200);
        expect(failing).toHaveBeenCalledTimes(2);
        if (stage === 'cdp') {
          expect(mockPage.close).not.toHaveBeenCalled();
          expect(mockContext.close).not.toHaveBeenCalled();
        }
      }
    );

    it('video relocation fallback requires readable original bytes', async () => {
      const { sessionId } = await create();
      const session = manager.getSession(sessionId);
      const root = await fs.mkdtemp(path.join(os.tmpdir(), 'bas-video-fallback-'));
      const source = path.join(root, 'source.webm');
      const blockedDirectory = path.join(root, 'file-instead-of-directory');
      try {
        await fs.writeFile(source, 'original video');
        await fs.writeFile(blockedDirectory, 'blocks relocation');
        session.videoDir = blockedDirectory;
        const video = { path: async () => source } as never;
        expect(await moveVideo(video, session, 0)).toBe(source);
        expect(await fs.readFile(source, 'utf8')).toBe('original video');
        await fs.unlink(source);
        await expect(moveVideo(video, session, 0)).rejects.toThrow();
        await fs.mkdir(source);
        await expect(moveVideo(video, session, 0)).rejects.toThrow('not a regular file');
      } finally {
        await fs.rm(root, { recursive: true, force: true });
      }
    });

    it('retains the recording buffer when recording stop fails', async () => {
      const { sessionId, leaseId } = await create();
      const session = manager.getSession(sessionId);
      const stop = jest.fn().mockRejectedValueOnce(new Error('recording stop fault')).mockResolvedValue(undefined);
      session.pipelineManager = { isRecording: () => true, stopRecording: stop } as never;
      initRecordingBuffer(sessionId);
      bufferTimelineEntry(sessionId, { id: 'pending-entry', sequenceNum: 0 } as never);
      expect((await close(sessionId, leaseId)).statusCode).toBe(500);
      expect(getTimelineEntries(sessionId).map((entry) => entry.id)).toEqual(['pending-entry']);
      expect(mockContext.close).not.toHaveBeenCalled();
      await expect(manager.resetSession(sessionId)).rejects.toThrow('closing');
      expect(manager.peekSession(sessionId).phase).toBe('closing');
      expect((await close(sessionId, leaseId)).statusCode).toBe(500);
      expect(mockContext.close).not.toHaveBeenCalled();
      acknowledgeTimelineEntries(sessionId, ['pending-entry'], true);
      expect((await close(sessionId, leaseId)).statusCode).toBe(200);
      expect(stop).toHaveBeenCalledTimes(2);
      expect(getTimelineEntries(sessionId)).toEqual([]);
    });

    it('refuses reset before navigation or storage clearing while pull entries are pending', async () => {
      const { sessionId } = await create();
      initRecordingBuffer(sessionId);
      bufferTimelineEntry(sessionId, { id: 'pending-reset', sequenceNum: 0 } as never);
      mockPage.goto.mockClear();
      mockContext.clearCookies.mockClear();
      try {
        await expect(manager.resetSession(sessionId)).rejects.toThrow('pending');
        expect(mockPage.goto).not.toHaveBeenCalled();
        expect(mockContext.clearCookies).not.toHaveBeenCalled();
        expect(getTimelineEntries(sessionId)).toHaveLength(1);
      } finally {
        acknowledgeTimelineEntries(sessionId, ['pending-reset'], true);
      }
      await manager.resetSession(sessionId);
      expect(mockPage.goto).toHaveBeenCalledWith('about:blank');
    });

    it('refuses a missing trace reference after flush without destroying the context', async () => {
      const { sessionId, leaseId } = await create();
      const session = manager.getSession(sessionId);
      const dir = await fs.mkdtemp(path.join(os.tmpdir(), 'bas-close-missing-'));
      session.tracing = true;
      session.tracePath = path.join(dir, 'trace.zip');
      try {
        expect((await close(sessionId, leaseId)).statusCode).toBe(500);
        expect(mockContext.close).not.toHaveBeenCalled();
        await fs.writeFile(session.tracePath, 'late trace');
        expect((await close(sessionId, leaseId)).statusCode).toBe(200);
        expect(mockContext.tracing.stop).toHaveBeenCalledTimes(1);
      } finally {
        // Keep cleanup independent of the assertion so a failed check cannot
        // hide the original result behind a missing fixture file on shutdown.
        if (manager.getAllSessionIds().includes(sessionId)) {
          await fs.writeFile(session.tracePath, 'cleanup trace');
          await manager.closeSession(sessionId);
        }
        await fs.rm(dir, { recursive: true, force: true });
      }
    });

    it('shutdown attempts other sessions and reports a retained cleanup failure', async () => {
      const first = await create();
      const otherContext = createMockContext();
      const otherPage = createMockPage();
      otherContext.newPage.mockResolvedValue(otherPage);
      mockBrowser.newContext.mockResolvedValueOnce(otherContext);
      await manager.startSession({
        execution_id: 'other-close-owner', viewport: { width: 800, height: 600 },
        reuse_mode: 'fresh', required_capabilities: {},
      });
      mockContext.close.mockRejectedValueOnce(new Error('first context fault'));
      await expect(manager.shutdown()).rejects.toThrow('shutdown incomplete');
      expect(otherContext.close).toHaveBeenCalledTimes(1);
      expect(mockBrowser.close).toHaveBeenCalled();
      expect(manager.getAllSessionIds()).toEqual([first.sessionId]);
      expect(manager.peekSession(first.sessionId).phase).toBe('closing');
    });

    it('waits for context finalization before moving video bytes', async () => {
      const { sessionId, leaseId } = await create();
      const session = manager.getSession(sessionId);
      const dir = await fs.mkdtemp(path.join(os.tmpdir(), 'bas-close-video-flush-'));
      const source = path.join(dir, 'video.webm');
      session.videoDir = dir;
      mockPage.video.mockReturnValue({ path: async () => source } as never);
      mockContext.close.mockImplementationOnce(async () => { await fs.writeFile(source, 'final video bytes'); });
      try {
        const response = await close(sessionId, leaseId);
        expect(response.getJSON()).not.toHaveProperty('error');
        expect(response.statusCode).toBe(200);
        const videoPaths = response.getJSON().video_paths as string[];
        expect(videoPaths).toHaveLength(1);
        expect(await fs.readFile(videoPaths[0]!, 'utf8')).toBe('final video bytes');
      } finally {
        if (manager.getAllSessionIds().includes(sessionId)) {
          await fs.writeFile(source, 'cleanup bytes');
          await manager.closeSession(sessionId);
        }
        await fs.rm(dir, { recursive: true, force: true });
      }
    });

    it('joins the authorized close result while shared-device cleanup is still pending', async () => {
      const { sessionId, leaseId } = await create();
      let release!: () => void;
      let entered!: () => void;
      const started = new Promise<void>((resolve) => { entered = resolve; });
      Reflect.set(manager, 'qualificationDevice', Promise.resolve({ close: async () => {
        entered();
        await new Promise<void>((resolve) => { release = resolve; });
      } }));
      const first = close(sessionId, leaseId);
      await started;
      let returned = false;
      const joined = close(sessionId, leaseId).then((res) => { returned = true; return res; });
      try {
        // The wrong lease must still fail while the resource result is pending.
        expect((await close(sessionId, 'wrong-lease')).statusCode).toBe(404);
        expect(returned).toBe(false);
      } finally {
        release();
        const [a, b] = await Promise.all([first, joined]);
        expect(a.statusCode).toBe(200);
        expect(b.getJSON()).toEqual(a.getJSON());
      }
    });

    it('joins concurrent HTTP close callers until the same cleanup completes', async () => {
      const { sessionId, leaseId } = await create();
      let release!: () => void;
      let entered!: () => void;
      const started = new Promise<void>((resolve) => { entered = resolve; });
      mockPage.close.mockImplementationOnce(async () => {
        entered();
        await new Promise<void>((resolve) => { release = resolve; });
      });
      let secondEntered!: () => void;
      const joined = new Promise<void>((resolve) => { secondEntered = resolve; });
      const actualClose = manager.closeSessionForLease.bind(manager);
      let callers = 0;
      jest.spyOn(manager, 'closeSessionForLease').mockImplementation((...args) => {
        if (++callers === 2) secondEntered();
        return actualClose(...args);
      });
      const first = close(sessionId, leaseId);
      await started;
      let secondCompleted = false;
      const second = close(sessionId, leaseId).then((res) => { secondCompleted = true; return res; });
      try {
        await joined;
        await new Promise<void>((resolve) => setImmediate(resolve));
        expect(secondCompleted).toBe(false);
      } finally {
        release();
        const [a, b] = await Promise.all([first, second]);
        expect(a.statusCode).toBe(200);
        expect(b.getJSON()).toEqual(a.getJSON());
        expect(mockPage.close).toHaveBeenCalledTimes(1);
      }
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

      expect(mockBrowser.close.mock.calls.length).toBeGreaterThan(0);
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

      jest.useFakeTimers();
      try {
        const before = Date.now();
        jest.advanceTimersByTime(1);
        manager.updateActivity(sessionId);
        const after = Date.now();

        const session = manager.getSession(sessionId);
        expect(session.lastUsedAt.getTime()).toBeGreaterThanOrEqual(before);
        expect(session.lastUsedAt.getTime()).toBeLessThanOrEqual(after);
      } finally {
        jest.useRealTimers();
      }
    });

    it('should not throw for non-existent session', () => {
      expect(() => manager.updateActivity('non-existent')).not.toThrow();
    });
  });
});
