import { handleHealth } from '../../../src/routes/health';
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

describe('Health Route', () => {
  let sessionManager: SessionManager;

  beforeEach(() => {
    const config = createTestConfig();
    sessionManager = new SessionManager(config);
  });

  afterEach(async () => {
    await sessionManager.shutdown();
  });

  it('should return 200 OK', async () => {
    const mockReq = createMockRequest({ method: 'GET', url: '/health' });
    const mockRes = createMockResponse();

    await handleHealth(mockReq, mockRes, sessionManager);
    await waitForResponse(mockRes);

    expect(mockRes.statusCode).toBe(200);
  });

  it('should return status and active sessions count', async () => {
    const mockReq = createMockRequest({ method: 'GET', url: '/health' });
    const mockRes = createMockResponse();

    await handleHealth(mockReq, mockRes, sessionManager);
    await waitForResponse(mockRes);

    const json = (mockRes as any).getJSON();
    expect(json.status).toBe('ok');
    expect(json.activeSessions).toBe(0);
  });

  it('should count active sessions', async () => {
    // Create a session
    await sessionManager.startSession({
      execution_id: 'exec-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    const mockReq = createMockRequest({ method: 'GET', url: '/health' });
    const mockRes = createMockResponse();

    await handleHealth(mockReq, mockRes, sessionManager);
    await waitForResponse(mockRes);

    const json = (mockRes as any).getJSON();
    expect(json.activeSessions).toBe(1);
  });
});
