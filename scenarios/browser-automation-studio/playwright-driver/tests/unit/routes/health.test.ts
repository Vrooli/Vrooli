import { handleHealth } from '../../../src/routes/health';
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
    const mockReq = createMockHttpRequest({ method: 'GET', url: '/health' });
    const mockRes = createMockHttpResponse();

    await handleHealth(mockReq, mockRes, sessionManager);

    expect(mockRes.statusCode).toBe(200);
  });

  it('should return status and sessions count', async () => {
    const mockReq = createMockHttpRequest({ method: 'GET', url: '/health' });
    const mockRes = createMockHttpResponse();

    await handleHealth(mockReq, mockRes, sessionManager);

    const json = (mockRes as any).getJSON();
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

    await handleHealth(mockReq, mockRes, sessionManager);

    const json = (mockRes as any).getJSON();
    expect(json.sessions).toBe(1);
  });
});
