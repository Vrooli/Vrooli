import { handleSessionClose } from '../../../src/routes/session-close';
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
});
