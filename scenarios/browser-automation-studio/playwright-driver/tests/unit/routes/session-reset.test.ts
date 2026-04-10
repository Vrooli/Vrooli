import { chromium } from 'rebrowser-playwright';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig } from '../../helpers';

jest.mock('rebrowser-playwright', () => ({
  chromium: {
    launch: jest.fn(),
  },
}));

describe('Session Reset Route', () => {
  let handleSessionReset: typeof import('../../../src/routes/session-reset').handleSessionReset;
  let SessionManager: typeof import('../../../src/session/manager').SessionManager;
  let sessionManager: InstanceType<typeof SessionManager>;

  beforeAll(async () => {
    const chromiumMock = chromium as unknown as { launch: jest.Mock };
    const mockBrowser = {
      newContext: jest.fn().mockResolvedValue({
        newPage: jest.fn().mockResolvedValue({
          on: jest.fn(),
          goto: jest.fn().mockResolvedValue(null),
          close: jest.fn().mockResolvedValue(undefined),
          evaluate: jest.fn().mockResolvedValue(undefined),
          waitForLoadState: jest.fn().mockResolvedValue(undefined),
          viewportSize: jest.fn().mockReturnValue({ width: 1280, height: 720 }),
          unroute: jest.fn().mockResolvedValue(undefined),
          route: jest.fn().mockResolvedValue(undefined),
          context: jest.fn(),
        }),
        addInitScript: jest.fn().mockResolvedValue(undefined),
        clearCookies: jest.fn().mockResolvedValue(undefined),
        clearPermissions: jest.fn().mockResolvedValue(undefined),
        pages: jest.fn().mockReturnValue([]),
        on: jest.fn(),
        off: jest.fn(),
        newCDPSession: jest.fn().mockResolvedValue({
          send: jest.fn().mockResolvedValue({ result: { type: 'string', value: '{}' } }),
          detach: jest.fn().mockResolvedValue(undefined),
        }),
        close: jest.fn().mockResolvedValue(undefined),
        tracing: {
          start: jest.fn().mockResolvedValue(undefined),
          stop: jest.fn().mockResolvedValue(undefined),
        },
      }),
      close: jest.fn().mockResolvedValue(undefined),
      isConnected: jest.fn().mockReturnValue(true),
      version: jest.fn().mockReturnValue('mock-version'),
    };
    chromiumMock.launch.mockResolvedValue(mockBrowser);

    ({ handleSessionReset } = await import('../../../src/routes/session-reset'));
    ({ SessionManager } = await import('../../../src/session/manager'));
  });

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
    const json = mockRes.getJSON();
    expect(json.success).toBe(true);
  });

  it('should return 404 for non-existent session', async () => {
    const mockReq = createMockHttpRequest({ method: 'POST', url: '/session/non-existent/reset' });
    const mockRes = createMockHttpResponse();

    await handleSessionReset(mockReq, mockRes, 'non-existent', sessionManager);

    expect(mockRes.statusCode).toBe(404);
  });
});
