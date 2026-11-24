import { handleSessionAppMonitorMetrics } from '../../../src/routes/session-app-monitor-metrics';
import type { SessionManager } from '../../../src/session';
import type { SessionState } from '../../../src/types/session';

function fixture(
  options: { url?: string; appMonitorMetrics?: boolean; backend?: 'managed' | 'installed' } = {}
) {
  const session = {
    ownerExecutionId: 'owner-1',
    leaseId: 'lease-1',
    createdAt: new Date(),
    instructionCount: 0,
    phase: 'ready',
    instructionInFlight: false,
    spec: {
      browser_backend: options.backend ?? 'managed',
      permission_profile: {
        effect: 'observe',
        approval: 'none',
        view: true,
        app_monitor_metrics: options.appMonitorMetrics ?? true,
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
  const req = {
    url: '/session/session-1/app-monitor-metrics',
    headers: { 'x-bas-execution-id': 'owner-1', 'x-bas-session-lease-id': 'lease-1' },
  };
  const res = {
    statusCode: 200,
    headers: {} as Record<string, string>,
    body: '',
    setHeader(name: string, value: string) {
      this.headers[name] = value;
    },
    end(body: string) {
      this.body = body;
    },
  };
  return { req, res, manager, session };
}

const counters = {
  iframeCount: 8,
  paneCount: 8,
  tabCardCount: 8,
  profilerMarkCount: 3,
  workspaceStorageBytes: 4096,
};

describe('handleSessionAppMonitorMetrics', () => {
  test.each([
    ['missing opt-in', { appMonitorMetrics: false }],
    ['installed backend', { backend: 'installed' as const }],
    ['redirected path', { url: 'http://localhost:20000/login' }],
    ['query route', { url: 'http://localhost:20000/apps/workspace?workspaceMode=add-pane' }],
  ])('denies %s before the fixed read', async (_name, options) => {
    const f = fixture(options);
    let reads = 0;
    await handleSessionAppMonitorMetrics(
      f.req as never,
      f.res as never,
      'session-1',
      f.manager,
      async () => {
        reads += 1;
        return counters;
      }
    );
    expect([403, 409]).toContain(f.res.statusCode);
    expect(reads).toBe(0);
  });

  test('returns only fixed metric scalars for the authorized active page', async () => {
    const f = fixture();
    await handleSessionAppMonitorMetrics(
      f.req as never,
      f.res as never,
      'session-1',
      f.manager,
      async () => counters
    );
    expect(f.res.statusCode).toBe(200);
    expect(JSON.parse(f.res.body)).toEqual({
      contract: 'app-monitor.preview-workspace.metrics.v1',
      sessionId: 'session-1',
      pageId: 'page-1',
      url: 'http://localhost:20000/apps/workspace',
      observedAt: expect.any(String),
      ...counters,
    });
  });

  test('rejects untyped values and never reflects raw content', async () => {
    const f = fixture();
    await handleSessionAppMonitorMetrics(
      f.req as never,
      f.res as never,
      'session-1',
      f.manager,
      async () =>
        ({
          ...counters,
          errorText: 'private page content',
        }) as never
    );
    expect(f.res.statusCode).toBe(409);
    expect(f.res.body).not.toContain('private page content');
  });
});
