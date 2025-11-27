import { handleSessionReset } from '../../../src/routes/session-reset';
import { SessionManager } from '../../../src/session/manager';
import { createMockRequest, createMockResponse, waitForResponse, createTestConfig } from '../../helpers';

// Mock playwright
jest.mock('playwright', () => ({
  chromium: {
    launch: jest.fn().mockResolvedValue({
      newContext: jest.fn().mockResolvedValue({
        newPage: jest.fn().mockResolvedValue({ on: jest.fn() }),
        clearCookies: jest.fn(),
        clearPermissions: jest.fn(),
        close: jest.fn(),
      }),
      close: jest.fn(),
      isConnected: jest.fn().mockReturnValue(true),
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
    const sessionId = await sessionManager.startSession({
      execution_id: 'exec-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    const mockReq = createMockRequest({ method: 'POST', url: `/session/${sessionId}/reset` });
    const mockRes = createMockResponse();

    await handleSessionReset(mockReq, mockRes, sessionId, sessionManager);
    await waitForResponse(mockRes);

    expect(mockRes.statusCode).toBe(200);
    const json = (mockRes as any).getJSON();
    expect(json.success).toBe(true);
  });

  it('should return 404 for non-existent session', async () => {
    const mockReq = createMockRequest({ method: 'POST', url: '/session/non-existent/reset' });
    const mockRes = createMockResponse();

    await handleSessionReset(mockReq, mockRes, 'non-existent', sessionManager);
    await waitForResponse(mockRes);

    expect(mockRes.statusCode).toBe(404);
  });
});
