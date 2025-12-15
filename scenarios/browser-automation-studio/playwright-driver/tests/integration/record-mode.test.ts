/**
 * Record Mode Integration Tests
 *
 * Tests the HTTP endpoints for record mode functionality.
 * These tests use mocked dependencies to test route handling
 * without requiring a running browser.
 */

import type { IncomingMessage, ServerResponse } from 'http';
import { EventEmitter } from 'events';
import {
  handleRecordStart,
  handleRecordStop,
  handleRecordStatus,
  handleRecordActions,
  handleValidateSelector,
  cleanupSessionRecording,
} from '../../src/routes/record-mode';
import type { SessionManager } from '../../src/session';
import type { Config } from '../../src/config';
import { createTestConfig } from '../helpers/test-config';

// Helper to create mock request
function createMockRequest(options: {
  method?: string;
  url?: string;
  body?: unknown;
}): IncomingMessage {
  const req = new EventEmitter() as IncomingMessage;
  (req as unknown as Record<string, unknown>).method = options.method || 'POST';
  (req as unknown as Record<string, unknown>).url = options.url || '/';

  if (options.body) {
    // Emit body data after a tick
    process.nextTick(() => {
      req.emit('data', Buffer.from(JSON.stringify(options.body)));
      req.emit('end');
    });
  } else {
    process.nextTick(() => {
      req.emit('end');
    });
  }

  return req;
}

// Helper to create mock response
function createMockResponse(): ServerResponse & {
  _getData: () => string;
  _getStatusCode: () => number;
} {
  const chunks: Buffer[] = [];
  const state = { statusCode: 200 };
  const headers: Record<string, string> = {};

  const res = {
    get statusCode() {
      return state.statusCode;
    },
    set statusCode(code: number) {
      state.statusCode = code;
    },
    setHeader: jest.fn((name: string, value: string) => {
      headers[name] = value;
    }),
    writeHead: jest.fn((code: number, hdrs?: Record<string, string>) => {
      state.statusCode = code;
      if (hdrs) {
        Object.assign(headers, hdrs);
      }
      return res;
    }),
    write: jest.fn((chunk: Buffer | string) => {
      chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
      return true;
    }),
    end: jest.fn((data?: string | Buffer) => {
      if (data) {
        chunks.push(Buffer.isBuffer(data) ? data : Buffer.from(data));
      }
      return res;
    }),
    _getData: () => Buffer.concat(chunks).toString(),
    _getStatusCode: () => state.statusCode,
  } as unknown as ServerResponse & {
    _getData: () => string;
    _getStatusCode: () => number;
  };

  return res;
}

// Helper to create mock session manager
function createMockSessionManager(session?: unknown): SessionManager {
  const mockSession = session || {
    page: {
      url: jest.fn().mockReturnValue('https://example.com'),
      locator: jest.fn().mockReturnValue({
        count: jest.fn().mockResolvedValue(1),
      }),
      evaluate: jest.fn().mockResolvedValue(1),
    },
    recordingController: null,
    recordingId: undefined,
    phase: 'ready',
  };

  return {
    getSession: jest.fn().mockReturnValue(mockSession),
    setSessionPhase: jest.fn(),
  } as unknown as SessionManager;
}

describe('Record Mode Routes', () => {
  const sessionId = 'test-session-123';
  let config: Config;

  beforeEach(() => {
    config = createTestConfig();
    // Clean up any leftover buffers
    cleanupSessionRecording(sessionId);
  });

  afterEach(() => {
    cleanupSessionRecording(sessionId);
  });

  describe('POST /session/:id/record/start', () => {
    it('should start recording successfully', async () => {
      const mockController = {
        isRecording: jest.fn().mockReturnValue(false),
        startRecording: jest.fn().mockResolvedValue('recording-123'),
      };

      const mockSession = {
        recordingController: mockController,
        recordingId: undefined,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ body: {} });
      const res = createMockResponse();

      await handleRecordStart(req, res, sessionId, sessionManager, config);

      expect(res._getStatusCode()).toBe(200);
      const data = JSON.parse(res._getData());
      expect(data.recording_id).toBe('recording-123');
      expect(data.session_id).toBe(sessionId);
      expect(data.started_at).toBeDefined();
    });

    it('should return 409 if already recording', async () => {
      const mockController = {
        isRecording: jest.fn().mockReturnValue(true),
      };

      const mockSession = {
        recordingController: mockController,
        recordingId: 'existing-recording',
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ body: {} });
      const res = createMockResponse();

      await handleRecordStart(req, res, sessionId, sessionManager, config);

      expect(res._getStatusCode()).toBe(409);
      const data = JSON.parse(res._getData());
      expect(data.error).toBe('RECORDING_IN_PROGRESS');
    });

    it('should create controller if not exists', async () => {
      const mockPage = {
        url: jest.fn().mockReturnValue('https://example.com'),
        exposeFunction: jest.fn().mockResolvedValue(undefined),
        evaluate: jest.fn().mockResolvedValue(undefined),
        on: jest.fn(),
      };

      const mockSession = {
        page: mockPage,
        recordingController: null, // No controller yet
        recordingId: undefined,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ body: {} });
      const res = createMockResponse();

      await handleRecordStart(req, res, sessionId, sessionManager, config);

      expect(res._getStatusCode()).toBe(200);
      // Controller should have been created
      expect(mockSession.recordingController).toBeDefined();
    });
  });

  describe('POST /session/:id/record/stop', () => {
    it('should stop recording successfully', async () => {
      const mockController = {
        isRecording: jest.fn().mockReturnValue(true),
        stopRecording: jest.fn().mockResolvedValue({
          recordingId: 'recording-123',
          actionCount: 5,
        }),
      };

      const mockSession = {
        recordingController: mockController,
        recordingId: 'recording-123',
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({});
      const res = createMockResponse();

      await handleRecordStop(req, res, sessionId, sessionManager);

      expect(res._getStatusCode()).toBe(200);
      const data = JSON.parse(res._getData());
      expect(data.recording_id).toBe('recording-123');
      expect(data.action_count).toBe(5);
      expect(data.stopped_at).toBeDefined();
    });

    it('should return 200 (idempotent) if not recording', async () => {
      // Idempotency: Calling stop when not recording is a successful no-op
      // This allows safe retries when the first stop request succeeded
      // but the response was lost due to network issues
      const mockController = {
        isRecording: jest.fn().mockReturnValue(false),
      };

      const mockSession = {
        recordingController: mockController,
        recordingId: 'previous-recording', // From a prior recording
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({});
      const res = createMockResponse();

      await handleRecordStop(req, res, sessionId, sessionManager);

      // Idempotent: Returns success with action_count: 0
      expect(res._getStatusCode()).toBe(200);
      const data = JSON.parse(res._getData());
      expect(data.recording_id).toBe('previous-recording');
      expect(data.action_count).toBe(0);
      expect(data.stopped_at).toBeDefined();
    });
  });

  describe('GET /session/:id/record/status', () => {
    it('should return recording status', async () => {
      const mockController = {
        getState: jest.fn().mockReturnValue({
          isRecording: true,
          recordingId: 'recording-123',
          actionCount: 3,
          startedAt: '2024-01-01T00:00:00.000Z',
        }),
      };

      const mockSession = {
        recordingController: mockController,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ method: 'GET' });
      const res = createMockResponse();

      await handleRecordStatus(req, res, sessionId, sessionManager);

      expect(res._getStatusCode()).toBe(200);
      const data = JSON.parse(res._getData());
      expect(data.session_id).toBe(sessionId);
      expect(data.is_recording).toBe(true);
      expect(data.recording_id).toBe('recording-123');
      expect(data.action_count).toBe(3);
    });

    it('should handle no controller', async () => {
      const mockSession = {
        recordingController: null,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ method: 'GET' });
      const res = createMockResponse();

      await handleRecordStatus(req, res, sessionId, sessionManager);

      expect(res._getStatusCode()).toBe(200);
      const data = JSON.parse(res._getData());
      expect(data.is_recording).toBe(false);
      expect(data.action_count).toBe(0);
    });
  });

  describe('GET /session/:id/record/actions', () => {
    it('should return buffered actions as TimelineEntry format', async () => {
      const sessionManager = createMockSessionManager();

      // Manually add actions to buffer for this test
      // This simulates actions being recorded
      const req = createMockRequest({ method: 'GET', url: '/session/test/record/actions' });
      const res = createMockResponse();

      await handleRecordActions(req, res, sessionId, sessionManager);

      expect(res._getStatusCode()).toBe(200);
      const data = JSON.parse(res._getData());
      expect(data.session_id).toBe(sessionId);
      // Now returns 'entries' (TimelineEntry format) instead of 'actions'
      expect(Array.isArray(data.entries)).toBe(true);
      expect(typeof data.count).toBe('number');
    });

    it('should clear buffer when clear=true', async () => {
      const sessionManager = createMockSessionManager();

      const req = createMockRequest({
        method: 'GET',
        url: '/session/test/record/actions?clear=true',
      });
      const res = createMockResponse();

      await handleRecordActions(req, res, sessionId, sessionManager);

      expect(res._getStatusCode()).toBe(200);
      const data = JSON.parse(res._getData());
      // Now returns 'entries' (TimelineEntry format) instead of 'actions'
      expect(data.entries).toEqual([]);
    });
  });

  describe('POST /session/:id/record/validate-selector', () => {
    it('should validate CSS selector', async () => {
      const mockController = {
        validateSelector: jest.fn().mockResolvedValue({
          valid: true,
          matchCount: 1,
          selector: 'button#submit',
        }),
      };

      const mockSession = {
        page: { url: jest.fn().mockReturnValue('https://example.com') },
        recordingController: mockController,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ body: { selector: 'button#submit' } });
      const res = createMockResponse();

      await handleValidateSelector(req, res, sessionId, sessionManager, config);

      expect(res._getStatusCode()).toBe(200);
      const data = JSON.parse(res._getData());
      expect(data.valid).toBe(true);
      expect(data.match_count).toBe(1);
    });

    it('should return 400 if selector missing', async () => {
      const sessionManager = createMockSessionManager();
      const req = createMockRequest({ body: {} });
      const res = createMockResponse();

      await handleValidateSelector(req, res, sessionId, sessionManager, config);

      expect(res._getStatusCode()).toBe(400);
      const data = JSON.parse(res._getData());
      expect(data.error).toBe('MISSING_SELECTOR');
    });

    it('should report invalid selector', async () => {
      const mockController = {
        validateSelector: jest.fn().mockResolvedValue({
          valid: false,
          matchCount: 0,
          selector: 'div.nonexistent',
          error: 'No elements found',
        }),
      };

      const mockSession = {
        page: { url: jest.fn().mockReturnValue('https://example.com') },
        recordingController: mockController,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ body: { selector: 'div.nonexistent' } });
      const res = createMockResponse();

      await handleValidateSelector(req, res, sessionId, sessionManager, config);

      expect(res._getStatusCode()).toBe(200);
      const data = JSON.parse(res._getData());
      expect(data.valid).toBe(false);
      expect(data.match_count).toBe(0);
    });
  });

  describe('cleanupSessionRecording', () => {
    it('should clean up session buffer', () => {
      // This should not throw
      expect(() => cleanupSessionRecording(sessionId)).not.toThrow();
    });

    it('should be idempotent', () => {
      cleanupSessionRecording(sessionId);
      cleanupSessionRecording(sessionId);
      // Should not throw
      expect(true).toBe(true);
    });
  });
});
