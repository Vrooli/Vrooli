import { playwrightProvider } from '../../../src/playwright';
import { handleHealth } from '../../../src/routes/health';
import { SessionManager } from '../../../src/session/manager';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig, createMockBrowser } from '../../helpers';

// Exercise the actual configured provider through the existing fixture seam.
let launchSpy: jest.SpiedFunction<typeof playwrightProvider.chromium.launch>;
beforeEach(() => {
  launchSpy = jest.spyOn(playwrightProvider.chromium, 'launch').mockResolvedValue(createMockBrowser());
});
afterEach(() => launchSpy.mockRestore());

describe('Health Route', () => {
  let sessionManager: SessionManager;

  beforeEach(() => {
    const config = createTestConfig();
    sessionManager = new SessionManager(config);
  });

  afterEach(async () => {
    await sessionManager.shutdown();
  });

  it('should return 200 OK', () => {
    const mockReq = createMockHttpRequest({ method: 'GET', url: '/health' });
    const mockRes = createMockHttpResponse();

    handleHealth(mockReq, mockRes, sessionManager);

    expect(mockRes.statusCode).toBe(200);
  });

  it('should return status and sessions count', () => {
    const mockReq = createMockHttpRequest({ method: 'GET', url: '/health' });
    const mockRes = createMockHttpResponse();

    handleHealth(mockReq, mockRes, sessionManager);

    const json = mockRes.getJSON();
    // Status is 'degraded' when browser hasn't been verified yet
    expect(['ok', 'degraded']).toContain(json.status);
    expect(json.sessions).toBe(0);
  });

  it('should count sessions', async () => {
    // Create a session
    await sessionManager.startSession({
      execution_id: 'exec-123',
      workflow_id: 'workflow-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    const mockReq = createMockHttpRequest({ method: 'GET', url: '/health' });
    const mockRes = createMockHttpResponse();

    handleHealth(mockReq, mockRes, sessionManager);

    const json = mockRes.getJSON();
    expect(json.sessions).toBe(1);
  });

  it('does not refresh idle activity while health is polled', async () => {
    const { sessionId } = await sessionManager.startSession({
      execution_id: 'exec-idle', workflow_id: 'workflow-idle', base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 }, reuse_mode: 'fresh', required_capabilities: {},
    });
    const session = sessionManager.peekSession(sessionId);
    session.lastUsedAt = new Date(Date.now() - 10 * 60 * 1000);
    for (let index = 0; index < 3; index++) {
      handleHealth(createMockHttpRequest({ method: 'GET', url: '/health' }), createMockHttpResponse(), sessionManager);
    }
    await sessionManager.cleanupIdleSessions();
    expect(() => sessionManager.peekSession(sessionId)).toThrow();
  });
});
