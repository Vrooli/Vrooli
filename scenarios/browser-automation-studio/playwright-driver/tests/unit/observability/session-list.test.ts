import { handleSessionList } from '../../../src/observability/route';
import type { SessionManager } from '../../../src/session';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig } from '../../helpers';

describe('recovery session inventory', () => {
  it('requires the loopback administrative secret before exposing lease identities', () => {
    const config = createTestConfig({ server: { adminSecret: 'recovery-secret' } });
    const sessionManager = {
      getSessionList: () => [{ id: 'session-1', lease_id: 'lease-capability' }],
      getSessionSummary: () => ({
        total: 1,
        active: 1,
        idle: 0,
        active_recordings: 0,
        idle_timeout_ms: 60_000,
        capacity: 2,
      }),
    } as unknown as SessionManager;
    const deps = { config, sessionManager, sessionCleanup: {} } as Parameters<
      typeof handleSessionList
    >[2];

    const unauthorized = createMockHttpRequest({ method: 'GET', url: '/observability/sessions' });
    Object.assign(unauthorized, { socket: { remoteAddress: '127.0.0.1' } });
    const unauthorizedRes = createMockHttpResponse();
    handleSessionList(unauthorized, unauthorizedRes, deps);
    expect(unauthorizedRes.statusCode).toBe(403);
    expect(unauthorizedRes.getJSON()).not.toHaveProperty('sessions');

    const authorized = createMockHttpRequest({
      method: 'GET',
      url: '/observability/sessions',
      headers: { 'x-playwright-admin-secret': 'recovery-secret' },
    });
    Object.assign(authorized, { socket: { remoteAddress: '127.0.0.1' } });
    const authorizedRes = createMockHttpResponse();
    handleSessionList(authorized, authorizedRes, deps);
    expect(authorizedRes.statusCode).toBe(200);
    expect(authorizedRes.getJSON().sessions).toEqual([
      { id: 'session-1', lease_id: 'lease-capability' },
    ]);
  });
});
