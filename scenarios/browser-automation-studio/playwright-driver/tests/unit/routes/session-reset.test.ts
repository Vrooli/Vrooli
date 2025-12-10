import { handleSessionReset } from '../../../src/routes/session-reset';
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
          unroute: jest.fn().mockResolvedValue(undefined),
          route: jest.fn().mockResolvedValue(undefined),
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

describe('Session Reset Route', () => {
  let sessionManager: SessionManager;

  beforeEach(() => {
    const config = createTestConfig();
    sessionManager = new SessionManager(config);
  });

  afterEach(async () => {
    await sessionManager.shutdown();
  });

  it('should reset existing session', async () => {
    // Create session first
    const { sessionId } = await sessionManager.startSession({
      execution_id: 'exec-123',
      workflow_id: 'workflow-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    const mockReq = createMockHttpRequest({ method: 'POST', url: `/session/${sessionId}/reset` });
    const mockRes = createMockHttpResponse();

    await handleSessionReset(mockReq, mockRes, sessionId, sessionManager);

    expect(mockRes.statusCode).toBe(200);
    const json = (mockRes as any).getJSON();
    expect(json.success).toBe(true);
  });

  it('should return 404 for non-existent session', async () => {
    const mockReq = createMockHttpRequest({ method: 'POST', url: '/session/non-existent/reset' });
    const mockRes = createMockHttpResponse();

    await handleSessionReset(mockReq, mockRes, 'non-existent', sessionManager);

    expect(mockRes.statusCode).toBe(404);
  });
});
