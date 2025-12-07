import { handleSessionStart } from '../../../src/routes/session-start';
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
      workflow_id: 'workflow-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    };

    const mockReq = createMockHttpRequest({ method: 'POST', url: '/session/start', body });
    const mockRes = createMockHttpResponse();

    await handleSessionStart(mockReq, mockRes, sessionManager, config);

    expect(mockRes.statusCode).toBe(200);
    const json = (mockRes as any).getJSON();
    expect(json.session_id).toBeDefined();
    expect(typeof json.session_id).toBe('string');
  });

  it('should return error for invalid body', async () => {
    const mockReq = createMockHttpRequest({ method: 'POST', url: '/session/start', body: { invalid: 'data' } });
    const mockRes = createMockHttpResponse();

    await handleSessionStart(mockReq, mockRes, sessionManager, config);

    expect(mockRes.statusCode).toBeGreaterThanOrEqual(400);
  });

  it('should handle resource limit errors', async () => {
    const limitedConfig = createTestConfig({
      session: { maxConcurrent: 1, idleTimeoutMs: 300000, poolSize: 5, cleanupIntervalMs: 60000 },
    });
    const limitedManager = new SessionManager(limitedConfig);

    // Create first session (max reached)
    await limitedManager.startSession({
      execution_id: 'exec-1',
      workflow_id: 'workflow-1',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    // Try to create second session
    const body = {
      execution_id: 'exec-2',
      workflow_id: 'workflow-2',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    };

    const mockReq = createMockHttpRequest({ method: 'POST', url: '/session/start', body });
    const mockRes = createMockHttpResponse();

    await handleSessionStart(mockReq, mockRes, limitedManager, limitedConfig);

    expect(mockRes.statusCode).toBe(429);

    await limitedManager.shutdown();
  });
});
