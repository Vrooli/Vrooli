import { chromium } from 'rebrowser-playwright';
import { createMockHttpRequest, createMockHttpResponse, createMockPage, createTestConfig } from '../../helpers';

jest.mock('rebrowser-playwright', () => ({
  chromium: {
    launch: jest.fn(),
  },
}));

describe('Session Reset Route', () => {
  let handleSessionReset: typeof import('../../../src/routes/session-reset').handleSessionReset;
  let SessionManager: typeof import('../../../src/session/manager').SessionManager;
  let sessionManager: InstanceType<typeof SessionManager>;

  beforeAll(async () => {
    const chromiumMock = chromium as unknown as { launch: jest.Mock };
    const mockBrowser = {
      newContext: jest.fn().mockResolvedValue({
        newPage: jest.fn().mockResolvedValue(createMockPage()),
        addInitScript: jest.fn().mockResolvedValue(undefined),
        clearCookies: jest.fn().mockResolvedValue(undefined),
        clearPermissions: jest.fn().mockResolvedValue(undefined),
        pages: jest.fn().mockReturnValue([]),
        on: jest.fn(),
        off: jest.fn(),
        newCDPSession: jest.fn().mockResolvedValue({
          send: jest.fn().mockResolvedValue({ result: { type: 'string', value: '{}' } }),
          detach: jest.fn().mockResolvedValue(undefined),
        }),
        close: jest.fn().mockResolvedValue(undefined),
        tracing: {
          start: jest.fn().mockResolvedValue(undefined),
          stop: jest.fn().mockResolvedValue(undefined),
        },
      }),
      close: jest.fn().mockResolvedValue(undefined),
      isConnected: jest.fn().mockReturnValue(true),
      version: jest.fn().mockReturnValue('mock-version'),
    };
    chromiumMock.launch.mockResolvedValue(mockBrowser);

    ({ handleSessionReset } = await import('../../../src/routes/session-reset'));
    ({ SessionManager } = await import('../../../src/session/manager'));
  });

  beforeEach(() => {
    const config = createTestConfig();
    sessionManager = new SessionManager(config);
  });

  afterEach(async () => {
    await sessionManager.shutdown();
  });

  it('should reset existing session', async () => {
    // Create session first
    const { sessionId, leaseId } = await sessionManager.startSession({
      execution_id: 'exec-123',
      workflow_id: 'workflow-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    const mockReq = createMockHttpRequest({ method: 'POST', url: `/session/${sessionId}/reset`, body: { execution_id: 'exec-123', lease_id: leaseId } });
    const mockRes = createMockHttpResponse();

    await handleSessionReset(mockReq, mockRes, sessionId, sessionManager);

    expect(mockRes.statusCode).toBe(200);
    const json = mockRes.getJSON();
    expect(json.success).toBe(true);
  });

  it('should return 404 for non-existent session', async () => {
    const mockReq = createMockHttpRequest({ method: 'POST', url: '/session/non-existent/reset', body: { execution_id: 'owner', lease_id: 'lease' } });
    const mockRes = createMockHttpResponse();

    await handleSessionReset(mockReq, mockRes, 'non-existent', sessionManager);

    expect(mockRes.statusCode).toBe(404);
  });

  it.each([false, true])('joins a current-owner reset and preserves failure=%s', async (fail) => {
    const owner = 'reset-owner';
    const { sessionId, leaseId } = await sessionManager.startSession({
      execution_id: owner, workflow_id: 'fixture', base_url: 'https://example.com',
      viewport: { width: 640, height: 480 }, reuse_mode: 'fresh', required_capabilities: {},
    });
    let finish!: () => void;
    let failReset!: (error: Error) => void;
    const pending = new Promise<void>((resolve, reject) => { finish = resolve; failReset = reject; });
    let admitted!: () => void;
    const admission = new Promise<void>((resolve) => { admitted = resolve; });
    const reset = jest.spyOn(sessionManager.peekSession(sessionId).context, 'clearCookies').mockClear().mockImplementation(() => { admitted(); return pending; });
    const send = (credentials: { execution_id: string; lease_id: string }) => {
      const request = createMockHttpRequest({ method: 'POST' });
      const response = createMockHttpResponse();
      const result = handleSessionReset(request, response, sessionId, sessionManager);
      request.emit('data', Buffer.from(JSON.stringify(credentials)));
      request.emit('end');
      return { response, result };
    };
    const first = send({ execution_id: owner, lease_id: leaseId });
    await admission;
    const second = send({ execution_id: owner, lease_id: leaseId });
    const stale = send({ execution_id: owner, lease_id: 'old-lease' });
    await stale.result;
    expect(stale.response.statusCode).toBe(404);
    expect(first.response.getBody()).toBe('');
    expect(second.response.getBody()).toBe('');
    if (fail) failReset(new Error('reset failed before acknowledgement'));
    else finish();
    await Promise.all([first.result, second.result]);
    expect(reset).toHaveBeenCalledTimes(1);
    expect(first.response.statusCode).toBe(fail ? 500 : 200);
    expect(second.response.statusCode).toBe(fail ? 500 : 200);
    if (!fail) expect(second.response.getJSON().success).toBe(true);
    reset.mockResolvedValue(undefined);
    const later = send({ execution_id: owner, lease_id: leaseId });
    await later.result;
    expect(later.response.statusCode).toBe(200);
    expect(reset).toHaveBeenCalledTimes(2);
    reset.mockRestore();
  });

  it('validates the lease after a delayed body, before joining any reset', async () => {
    const spec = {
      execution_id: 'old-owner', workflow_id: 'fixture', base_url: 'https://example.com',
      viewport: { width: 640, height: 480 }, reuse_mode: 'fresh' as const,
      required_capabilities: {}, labels: { reset: 'delayed-body' },
    };
    const first = await sessionManager.startSession(spec);
    const request = createMockHttpRequest({ method: 'POST' });
    const response = createMockHttpResponse();
    const result = handleSessionReset(request, response, first.sessionId, sessionManager);
    expect(sessionManager.releaseExecutionLease(first.sessionId, spec.execution_id, first.leaseId)).toBe(true);
    const next = await sessionManager.startSession({ ...spec, execution_id: 'new-owner', reuse_mode: 'reuse' });
    expect(next.sessionId).toBe(first.sessionId);
    const reset = jest.spyOn(sessionManager, 'resetSession');
    const lastUsedAt = sessionManager.peekSession(next.sessionId).lastUsedAt;
    request.emit('data', Buffer.from(JSON.stringify({ execution_id: spec.execution_id, lease_id: first.leaseId })));
    request.emit('end');
    await result;
    expect(response.statusCode).toBe(404);
    expect(reset).not.toHaveBeenCalled();
    expect(sessionManager.peekSession(next.sessionId).lastUsedAt).toBe(lastUsedAt);
    reset.mockRestore();
  });

});
