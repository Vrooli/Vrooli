import { handleSessionRelease } from '../../../src/routes/session-release';
import { SessionManager } from '../../../src/session/manager';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig } from '../../helpers';

jest.mock('playwright', () => ({
  chromium: {
    launch: jest.fn().mockResolvedValue({
      newContext: jest.fn().mockResolvedValue({
        newPage: jest.fn().mockResolvedValue({ on: jest.fn(), goto: jest.fn().mockResolvedValue(null), viewportSize: jest.fn().mockReturnValue({ width: 1280, height: 720 }) }),
        clearCookies: jest.fn().mockResolvedValue(undefined),
        clearPermissions: jest.fn().mockResolvedValue(undefined),
        close: jest.fn().mockResolvedValue(undefined),
      }),
      close: jest.fn().mockResolvedValue(undefined),
      isConnected: jest.fn().mockReturnValue(true),
    }),
  },
}));

describe('Session Release Route', () => {
  let sessionManager: SessionManager;

  beforeEach(() => {
    sessionManager = new SessionManager(createTestConfig());
  });

  afterEach(async () => {
    await sessionManager.shutdown();
  });

  it('releases only the current execution lease', async () => {
    const { sessionId, leaseId } = await sessionManager.startSession({
      execution_id: 'exec-owner', workflow_id: 'workflow-123', viewport: { width: 1280, height: 720 }, reuse_mode: 'fresh', required_capabilities: {},
    });
    const response = createMockHttpResponse();
    await handleSessionRelease(createMockHttpRequest({ method: 'POST', url: `/session/${sessionId}/release`, body: { execution_id: 'exec-owner', lease_id: leaseId } }), response, sessionId, sessionManager);
    expect(response.statusCode).toBe(200);
    expect(response.getJSON()).toEqual({ success: true });

    const staleResponse = createMockHttpResponse();
    await handleSessionRelease(createMockHttpRequest({ method: 'POST', url: `/session/${sessionId}/release`, body: { execution_id: 'exec-owner', lease_id: 'stale' } }), staleResponse, sessionId, sessionManager);
    expect(staleResponse.statusCode).toBe(404);
  });
});
