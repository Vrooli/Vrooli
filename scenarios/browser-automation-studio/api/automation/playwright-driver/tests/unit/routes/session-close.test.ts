import { handleSessionClose } from '../../../src/routes/session-close';
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
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    const mockReq = createMockRequest({ method: 'POST', url: `/session/${sessionId}/close` });
    const mockRes = createMockResponse();

    await handleSessionClose(mockReq, mockRes, sessionId, sessionManager);
    await waitForResponse(mockRes);

    expect(mockRes.statusCode).toBe(200);
    const json = (mockRes as any).getJSON();
    expect(json.success).toBe(true);
  });

  it('should return 404 for non-existent session', async () => {
    const mockReq = createMockRequest({ method: 'POST', url: '/session/non-existent/close' });
    const mockRes = createMockResponse();

    await handleSessionClose(mockReq, mockRes, 'non-existent', sessionManager);
    await waitForResponse(mockRes);

    expect(mockRes.statusCode).toBe(404);
  });

  it('should remove session from manager', async () => {
    // Create session first
    const sessionId = await sessionManager.startSession({
      execution_id: 'exec-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    const mockReq = createMockRequest({ method: 'POST', url: `/session/${sessionId}/close` });
    const mockRes = createMockResponse();

    await handleSessionClose(mockReq, mockRes, sessionId, sessionManager);
    await waitForResponse(mockRes);

    // Session should no longer exist
    expect(() => sessionManager.getSession(sessionId)).toThrow();
  });
});
