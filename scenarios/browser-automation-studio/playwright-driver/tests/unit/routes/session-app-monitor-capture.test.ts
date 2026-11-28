import { Readable } from 'stream';
import { handleSessionAppMonitorCapture } from '../../../src/routes/session-app-monitor-capture';
import type { SessionManager } from '../../../src/session';
import type { SessionState } from '../../../src/types/session';

function fixture(options: { url?: string; appMonitorTrace?: boolean } = {}) {
  const session = {
    ownerExecutionId: 'owner-1',
    leaseId: 'lease-1',
    createdAt: new Date(),
    instructionCount: 0,
    phase: 'ready',
    instructionInFlight: false,
    spec: {
      browser_backend: 'managed',
      permission_profile: {
        effect: 'observe',
        approval: 'none',
        view: true,
        app_monitor_trace: options.appMonitorTrace ?? true,
        app_monitor_trace_max_duration_ms: 10_000,
        app_monitor_trace_max_bytes: 1 << 20,
        url_allow: ['http://localhost/apps/workspace*'],
      },
    },
    page: { url: () => options.url ?? 'http://localhost:20000/apps/workspace' },
    pageBindings: { getId: () => 'page-1' },
  } as unknown as SessionState;
  const manager = {
    getSessionForLease: (sessionId: string, executionId: string, leaseId: string) => {
      if (sessionId !== 'session-1' || executionId !== 'owner-1' || leaseId !== 'lease-1')
        throw new Error('not found');
      return session;
    },
  } as unknown as SessionManager;
  const makeRequest = (body: unknown) => {
    const req = Readable.from([Buffer.from(JSON.stringify(body))]);
    Object.assign(req, {
      url: '/session/session-1/app-monitor-capture/start',
      headers: {
        'x-bas-execution-id': 'owner-1',
        'x-bas-session-lease-id': 'lease-1',
      },
    });
    return req;
  };
  const makeResponse = () => ({
    statusCode: 200,
    headers: {} as Record<string, string>,
    body: '',
    setHeader(name: string, value: string) {
      this.headers[name] = value;
    },
    end(body: string) {
      this.body = body;
    },
  });
  return { session, manager, makeRequest, makeResponse };
}

describe('handleSessionAppMonitorCapture', () => {
  test.each([
    ['missing explicit trace grant', { appMonitorTrace: false }],
    ['redirected current URL', { url: 'http://localhost:20000/login' }],
  ])('denies %s before starting CDP', async (_name, options) => {
    const f = fixture(options);
    const req = f.makeRequest({ capture_id: '550e8400-e29b-41d4-a716-446655440000' });
    const res = f.makeResponse();
    await handleSessionAppMonitorCapture(
      req as never,
      res as never,
      'session-1',
      'begin',
      f.manager
    );
    expect(res.statusCode).toBe(403);
    expect(f.session.appMonitorTraceCapture).toBeUndefined();
  });

  test('denies an invalid lease before reading or starting capture', async () => {
    const f = fixture();
    const req = f.makeRequest({ capture_id: '550e8400-e29b-41d4-a716-446655440000' });
    (req as unknown as { headers: Record<string, string> }).headers['x-bas-session-lease-id'] =
      'wrong-lease';
    const res = f.makeResponse();
    await handleSessionAppMonitorCapture(
      req as never,
      res as never,
      'session-1',
      'begin',
      f.manager
    );
    expect(res.statusCode).toBe(404);
    expect(res.body).toContain('session not found');
    expect(f.session.appMonitorTraceCapture).toBeUndefined();
  });
});
