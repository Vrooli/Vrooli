import { handleSessionStart } from '../../../src/routes/session-start';
import * as frameStreaming from '../../../src/frame-streaming';
import { SessionManager } from '../../../src/session/manager';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig } from '../../helpers';

// Mock playwright - must be inline to avoid hoisting issues
jest.mock('playwright', () => ({
  chromium: {
    launch: jest.fn().mockResolvedValue({
      newContext: jest.fn().mockResolvedValue({
        newPage: jest.fn().mockResolvedValue({
          on: jest.fn(),
          goto: jest.fn().mockResolvedValue(null),
          close: jest.fn().mockResolvedValue(undefined),
          evaluate: jest.fn().mockResolvedValue(undefined),
          viewportSize: jest.fn().mockReturnValue({ width: 1280, height: 720 }),
        }),
        clearCookies: jest.fn().mockResolvedValue(undefined),
        clearPermissions: jest.fn().mockResolvedValue(undefined),
        close: jest.fn().mockResolvedValue(undefined),
        tracing: {
          start: jest.fn().mockResolvedValue(undefined),
          stop: jest.fn().mockResolvedValue(undefined),
        },
      }),
      close: jest.fn().mockResolvedValue(undefined),
      isConnected: jest.fn().mockReturnValue(true),
      version: jest.fn().mockReturnValue('mock-version'),
    }),
  },
}));

describe('Session Start Route', () => {
  let sessionManager: SessionManager;
  let config: ReturnType<typeof createTestConfig>;

  beforeEach(() => {
    config = createTestConfig();
    sessionManager = new SessionManager(config);
  });

  afterEach(async () => {
    await sessionManager.shutdown();
  });

  it('should create new session', async () => {
    const body = {
      execution_id: 'exec-123',
      workflow_id: 'workflow-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    };

    const mockReq = createMockHttpRequest({ method: 'POST', url: '/session/start', body });
    const mockRes = createMockHttpResponse();

    await handleSessionStart(mockReq, mockRes, sessionManager, config);

    expect(mockRes.statusCode).toBe(200);
    const json = mockRes.getJSON();
    expect(json.session_id).toBeDefined();
    expect(typeof json.session_id).toBe('string');
    expect(typeof json.lease_id).toBe('string');
    expect(json.last_instruction_sequence).toBe(0);
    // New response fields from signal improvements
    expect(json.phase).toBe('ready');
    expect(json.created_at).toBeDefined();
    // reused should be undefined for fresh sessions
    expect(json.reused).toBeUndefined();
  });

  it('returns the current operation highwater on a repeated start', async () => {
    const body = { execution_id: 'same-owner', workflow_id: 'workflow', reuse_mode: 'fresh', viewport: { width: 1280, height: 720 } };
    const first = createMockHttpResponse();
    await handleSessionStart(createMockHttpRequest({ body }), first, sessionManager, config);
    const session = sessionManager.peekSession(first.getJSON().session_id);
    session.lastInstructionSequence = 27;
    const repeated = createMockHttpResponse();
    await handleSessionStart(createMockHttpRequest({ body }), repeated, sessionManager, config);
    expect(repeated.statusCode).toBe(200);
    expect(repeated.getJSON().last_instruction_sequence).toBe(27);
    expect(repeated.getJSON().lease_id).toBe(first.getJSON().lease_id);
  });

  describe('deferred preview ownership', () => {
    let startPreview: jest.SpiedFunction<typeof frameStreaming.startFrameStreaming>;
    let completeReadiness: (ready: boolean) => void;
    let readiness: jest.SpiedFunction<SessionManager['waitForPipelineReady']>;

    beforeEach(() => {
      startPreview = jest.spyOn(frameStreaming, 'startFrameStreaming').mockImplementation(() => {});
      readiness = jest.spyOn(sessionManager, 'waitForPipelineReady').mockReturnValue(
        new Promise<boolean>((resolve) => { completeReadiness = resolve; }),
      );
    });

    afterEach(() => {
      startPreview.mockRestore();
      readiness.mockRestore();
    });

    it.each(['not-ready', 'closed', 'released', 'reassigned', 'closing', 'ready'] as const)(
      'starts a deferred preview only for ready current ownership: %s', async (state) => {
        const body = {
          execution_id: 'preview-owner', workflow_id: 'preview',
          viewport: { width: 640, height: 480 }, reuse_mode: 'fresh',
          labels: { pool: 'preview' },
          frame_streaming: { callback_url: 'http://127.0.0.1:65534/frames' },
        };
        const response = createMockHttpResponse();
        await handleSessionStart(createMockHttpRequest({ body }), response, sessionManager, config);
        expect(response.statusCode).toBe(200);
        expect(startPreview).not.toHaveBeenCalled();
        const { session_id: id, lease_id: lease } = response.getJSON();
        if (state === 'closed') await sessionManager.closeSession(id);
        if (state === 'closing') sessionManager.peekSession(id).phase = 'closing';
        if (state === 'released' || state === 'reassigned') {
          expect(sessionManager.releaseExecutionLease(id, body.execution_id, lease)).toBe(true);
        }
        if (state === 'reassigned') {
          const reused = await sessionManager.startSession({ ...body, execution_id: 'replacement', reuse_mode: 'reuse' });
          expect(reused.sessionId).toBe(id);
          expect(reused.leaseId).not.toBe(lease);
        }
        completeReadiness(state !== 'not-ready');
        await new Promise((resolve) => setImmediate(resolve));
        expect(startPreview).toHaveBeenCalledTimes(state === 'ready' ? 1 : 0);
        if (state === 'ready') {
          const provider = startPreview.mock.calls[0]![1];
          expect(provider.getSession(id).page).toBe(sessionManager.peekSession(id).page);
          sessionManager.releaseExecutionLease(id, body.execution_id, lease);
          expect(() => provider.getSession(id)).toThrow();
        }
      },
    );
  });

  it('should return error for invalid body', async () => {
    const mockReq = createMockHttpRequest({ method: 'POST', url: '/session/start', body: { invalid: 'data' } });
    const mockRes = createMockHttpResponse();

    await handleSessionStart(mockReq, mockRes, sessionManager, config);

    expect(mockRes.statusCode).toBeGreaterThanOrEqual(400);
  });

  it('should handle resource limit errors', async () => {
    const limitedConfig = createTestConfig({
      session: { maxConcurrent: 1, idleTimeoutMs: 300000, poolSize: 5, cleanupIntervalMs: 60000 },
    });
    const limitedManager = new SessionManager(limitedConfig);

    // Create first session (max reached)
    await limitedManager.startSession({
      execution_id: 'exec-1',
      workflow_id: 'workflow-1',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    // Try to create second session
    const body = {
      execution_id: 'exec-2',
      workflow_id: 'workflow-2',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    };

    const mockReq = createMockHttpRequest({ method: 'POST', url: '/session/start', body });
    const mockRes = createMockHttpResponse();

    await handleSessionStart(mockReq, mockRes, limitedManager, limitedConfig);

    expect(mockRes.statusCode).toBe(429);

    await limitedManager.shutdown();
  });
});
