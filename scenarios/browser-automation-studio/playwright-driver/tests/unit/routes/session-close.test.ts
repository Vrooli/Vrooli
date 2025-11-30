import { handleSessionClose } from '../../../src/routes/session-close';
import { SessionManager } from '../../../src/session/manager';
import { createMockHttpRequest, createMockHttpResponse, waitForResponse, createTestConfig } from '../../helpers';

// Mock playwright
jest.mock('playwright', () => ({
  chromium: {
    launch: jest.fn().mockResolvedValue({
      newContext: jest.fn().mockResolvedValue({
        newPage: jest.fn().mockResolvedValue({ on: jest.fn(), close: jest.fn().mockResolvedValue(undefined) }),
        clearCookies: jest.fn(),
        clearPermissions: jest.fn(),
        close: jest.fn().mockResolvedValue(undefined),
      }),
      close: jest.fn().mockResolvedValue(undefined),
      isConnected: jest.fn().mockReturnValue(true),
      version: jest.fn().mockReturnValue('mock-version'),
    }),
  },
}));

describe('Session Close Route', () => {
  let sessionManager: SessionManager;

  beforeEach(() => {
    const config = createTestConfig();
    sessionManager = new SessionManager(config);
  });

  afterEach(async () => {
    await sessionManager.shutdown();
  });

  it('should close existing session', async () => {
    // Create session first
    const sessionId = await sessionManager.startSession({
      execution_id: 'exec-123',
      workflow_id: 'workflow-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    const mockReq = createMockHttpRequest({ method: 'POST', url: `/session/${sessionId}/close` });
    const mockRes = createMockHttpResponse();

    await handleSessionClose(mockReq, mockRes, sessionId, sessionManager);
    await waitForResponse(mockRes);

    expect(mockRes.statusCode).toBe(200);
    const json = (mockRes as any).getJSON();
    expect(json.success).toBe(true);
  });

  it('should return 404 for non-existent session', async () => {
    const mockReq = createMockHttpRequest({ method: 'POST', url: '/session/non-existent/close' });
    const mockRes = createMockHttpResponse();

    await handleSessionClose(mockReq, mockRes, 'non-existent', sessionManager);
    await waitForResponse(mockRes);

    expect(mockRes.statusCode).toBe(404);
  });

  it('should remove session from manager', async () => {
    // Create session first
    const sessionId = await sessionManager.startSession({
      execution_id: 'exec-123',
      workflow_id: 'workflow-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    const mockReq = createMockHttpRequest({ method: 'POST', url: `/session/${sessionId}/close` });
    const mockRes = createMockHttpResponse();

    await handleSessionClose(mockReq, mockRes, sessionId, sessionManager);
    await waitForResponse(mockRes);

    // Session should no longer exist
    expect(() => sessionManager.getSession(sessionId)).toThrow();
  });
});
