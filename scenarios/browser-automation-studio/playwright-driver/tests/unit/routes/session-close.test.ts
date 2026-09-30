import { handleSessionClose, handleSessionForceClose } from '../../../src/routes/session-close';
import { SessionManager } from '../../../src/session/manager';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig } from '../../helpers';
import { promises as fs } from 'node:fs';
import path from 'node:path';
import os from 'node:os';

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
    const { sessionId, leaseId } = await sessionManager.startSession({
      execution_id: 'exec-123',
      workflow_id: 'workflow-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    const mockReq = createMockHttpRequest({ method: 'POST', url: `/session/${sessionId}/close`, body: { execution_id: 'exec-123', lease_id: leaseId } });
    const mockRes = createMockHttpResponse();

    await handleSessionClose(mockReq, mockRes, sessionId, sessionManager);

    expect(mockRes.statusCode).toBe(200);
    const json = mockRes.getJSON();
    expect(json.success).toBe(true);
  });

  it('should return 404 for non-existent session', async () => {
    const mockReq = createMockHttpRequest({ method: 'POST', url: '/session/non-existent/close', body: { execution_id: 'exec-123', lease_id: 'lease-123' } });
    const mockRes = createMockHttpResponse();

    await handleSessionClose(mockReq, mockRes, 'non-existent', sessionManager);

    expect(mockRes.statusCode).toBe(404);
  });

  it('should remove session from manager', async () => {
    // Create session first
    const { sessionId, leaseId } = await sessionManager.startSession({
      execution_id: 'exec-123',
      workflow_id: 'workflow-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: {},
    });

    const mockReq = createMockHttpRequest({ method: 'POST', url: `/session/${sessionId}/close`, body: { execution_id: 'exec-123', lease_id: leaseId } });
    const mockRes = createMockHttpResponse();

    await handleSessionClose(mockReq, mockRes, sessionId, sessionManager);

    // Session should no longer exist
    expect(() => sessionManager.getSession(sessionId)).toThrow();
  });

  it('force-closes only for a loopback caller holding the recovery secret', async () => {
    const config = createTestConfig({ server: { adminSecret: 'recovery-secret' } });
    const { sessionId } = await sessionManager.startSession({
      execution_id: 'exec-force-close', workflow_id: 'workflow-force-close', base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 }, reuse_mode: 'fresh', required_capabilities: {},
    });
    const unauthorized = createMockHttpRequest({ method: 'POST', url: `/session/${sessionId}/force-close` });
    Object.assign(unauthorized, { socket: { remoteAddress: '127.0.0.1' } });
    const unauthorizedRes = createMockHttpResponse();
    await handleSessionForceClose(unauthorized, unauthorizedRes, sessionId, sessionManager, config);
    expect(unauthorizedRes.statusCode).toBe(403);
    expect(() => sessionManager.peekSession(sessionId)).not.toThrow();

    const authorized = createMockHttpRequest({ method: 'POST', url: `/session/${sessionId}/force-close`, headers: { 'x-playwright-admin-secret': 'recovery-secret' } });
    Object.assign(authorized, { socket: { remoteAddress: '127.0.0.1' } });
    const authorizedRes = createMockHttpResponse();
    await handleSessionForceClose(authorized, authorizedRes, sessionId, sessionManager, config);
    expect(authorizedRes.statusCode).toBe(200);
    expect(() => sessionManager.peekSession(sessionId)).toThrow();
  });

  it('should return video paths when available', async () => {
    const config = createTestConfig();
    sessionManager = new SessionManager(config);

    const executionId = 'exec-video-123';
    const { sessionId, leaseId } = await sessionManager.startSession({
      execution_id: executionId,
      workflow_id: 'workflow-123',
      base_url: 'https://example.com',
      viewport: { width: 1280, height: 720 },
      reuse_mode: 'fresh',
      required_capabilities: { video: true },
    });

    const session = sessionManager.getSession(sessionId);
    const tempDir = await fs.mkdtemp(path.join(os.tmpdir(), 'bas-video-test-'));
    try {
      const sourcePath = path.join(tempDir, 'source-video.webm');
      await fs.writeFile(sourcePath, 'fake-video');

      session.videoDir = tempDir;
      const pageWithVideo = session.pages[0] as unknown as {
        video: () => { path: () => Promise<string | null> };
      };
      pageWithVideo.video = (): { path: () => Promise<string | null> } => ({
        path: (): Promise<string | null> => Promise.resolve(sourcePath),
      });

      const mockReq = createMockHttpRequest({ method: 'POST', url: `/session/${sessionId}/close`, body: { execution_id: executionId, lease_id: leaseId } });
      const mockRes = createMockHttpResponse();

      await handleSessionClose(mockReq, mockRes, sessionId, sessionManager);

      expect(mockRes.statusCode).toBe(200);
      const json = mockRes.getJSON();
      const expectedPath = path.join(tempDir, `execution-${executionId}-page-1.webm`);
      expect(json.video_paths).toEqual([expectedPath]);
    } finally {
      await fs.rm(tempDir, { recursive: true, force: true });
    }
  });
  it('closes real Chromium and returns readable video, trace and HAR bytes', async () => {
    const root = await fs.mkdtemp(path.join(os.tmpdir(), 'bas-close-captures-'));
    sessionManager = new SessionManager(createTestConfig({ telemetry: { har: { enabled: true } } }));
    try {
      const { sessionId, leaseId } = await sessionManager.startSession({
        execution_id: 'actual-capture-close', workflow_id: 'local-fixture',
        viewport: { width: 640, height: 480 }, reuse_mode: 'fresh',
        required_capabilities: { video: true, tracing: true, har: true },
        artifact_paths: { root },
      });
      await sessionManager.getSession(sessionId).page.goto('data:text/html,<title>Capture fixture</title><h1>Retained evidence</h1>');
      await sessionManager.getSession(sessionId).page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
      const res = createMockHttpResponse();
      await handleSessionClose(createMockHttpRequest({ body: { execution_id: 'actual-capture-close', lease_id: leaseId } }), res, sessionId, sessionManager);
      expect(res.getJSON()).not.toHaveProperty('error');
      expect(res.statusCode).toBe(200);
      const result = res.getJSON() as { success: boolean; video_paths: string[]; trace_path: string; har_path: string };
      expect(result.success).toBe(true);
      expect(result.video_paths).toHaveLength(1);
      const video = await fs.readFile(result.video_paths[0]!);
      const trace = await fs.readFile(result.trace_path);
      const har = JSON.parse(await fs.readFile(result.har_path, 'utf8'));
      expect(video.subarray(0, 4)).toEqual(Buffer.from([0x1a, 0x45, 0xdf, 0xa3]));
      expect(trace.subarray(0, 2).toString()).toBe('PK');
      expect(har.log.version).toBe('1.2');
      expect(sessionManager.getSessionCount()).toBe(0);
    } finally {
      await sessionManager.shutdown();
      await fs.rm(root, { recursive: true, force: true });
    }
  }, 30000);

});
