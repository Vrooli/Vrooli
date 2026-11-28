import { playwrightProvider } from '../../../src/playwright';
import { handleSessionRelease } from '../../../src/routes/session-release';
import { SessionManager } from '../../../src/session/manager';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig, createMockBrowser } from '../../helpers';

// Exercise the actual configured provider through the existing fixture seam.
let launchSpy: jest.SpiedFunction<typeof playwrightProvider.chromium.launch>;
beforeEach(() => {
  launchSpy = jest.spyOn(playwrightProvider.chromium, 'launch').mockResolvedValue(createMockBrowser());
});
afterEach(() => launchSpy.mockRestore());

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
