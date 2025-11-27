import { handleSessionStart } from '../../../src/routes/session-start';
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

describe('Session Start Route', () => {
  let sessionManager: SessionManager;
  let config: ReturnType<typeof createTestConfig>;

  beforeEach(() => {
    config = createTestConfig();
    sessionManager = new SessionManager(config);
  });

  afterEach(async () => {
    await sessionManager.shutdown();
  });

  it('should create new session', async () => {
    const body = {
      execution_id: 'exec-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    };

    const mockReq = createMockRequest({ method: 'POST', url: '/session/start', body });
    const mockRes = createMockResponse();

    await handleSessionStart(mockReq, mockRes, sessionManager, config);
    await waitForResponse(mockRes);

    expect(mockRes.statusCode).toBe(200);
    const json = (mockRes as any).getJSON();
    expect(json.sessionId).toBeDefined();
    expect(typeof json.sessionId).toBe('string');
  });

  it('should return error for invalid body', async () => {
    const mockReq = createMockRequest({ method: 'POST', url: '/session/start', body: { invalid: 'data' } });
    const mockRes = createMockResponse();

    await handleSessionStart(mockReq, mockRes, sessionManager, config);
    await waitForResponse(mockRes);

    expect(mockRes.statusCode).toBeGreaterThanOrEqual(400);
  });

  it('should handle resource limit errors', async () => {
    const limitedConfig = createTestConfig({
      session: { maxConcurrent: 1, idleTimeoutMs: 300000, cleanupIntervalMs: 60000 },
    });
    const limitedManager = new SessionManager(limitedConfig);

    // Create first session (max reached)
    await limitedManager.startSession({
      execution_id: 'exec-1',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    // Try to create second session
    const body = {
      execution_id: 'exec-2',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    };

    const mockReq = createMockRequest({ method: 'POST', url: '/session/start', body });
    const mockRes = createMockResponse();

    await handleSessionStart(mockReq, mockRes, limitedManager, limitedConfig);
    await waitForResponse(mockRes);

    expect(mockRes.statusCode).toBe(429);

    await limitedManager.shutdown();
  });
});
