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

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const parseResponse = (
  res: ServerResponse & { _getData: () => string }
): Record<string, unknown> => {
  const parsed: unknown = JSON.parse(res._getData());
  return isRecord(parsed) ? parsed : {};
};

const getStringField = (data: Record<string, unknown>, key: string): string | undefined =>
  typeof data[key] === 'string' ? data[key] : undefined;

const getNumberField = (data: Record<string, unknown>, key: string): number | undefined =>
  typeof data[key] === 'number' ? data[key] : undefined;

const getBooleanField = (data: Record<string, unknown>, key: string): boolean | undefined =>
  typeof data[key] === 'boolean' ? data[key] : undefined;

interface MockPipelineManager {
  isRecording: jest.Mock<boolean, []>;
  getRecordingId: jest.Mock<string | undefined, []>;
  getRecordingData: jest.Mock<{ startedAt: string } | undefined, []>;
  getState: jest.Mock<{
    phase: string;
    recording?: { recordingId?: string; actionCount: number; startedAt: string };
  }, []>;
  startRecording: jest.Mock<Promise<string>, []>;
  stopRecording: jest.Mock<Promise<{ recordingId: string; actionCount: number }>, []>;
  validateSelector: jest.Mock<
    Promise<{ valid: boolean; matchCount: number; selector: string; error?: string }>,
    []
  >;
  getVerification: jest.Mock<{
    loaded: boolean;
    ready: boolean;
    handlersCount: number;
    inMainContext: boolean;
    version: string;
  }, []>;
}

// Helper to create mock pipeline manager
function createMockPipelineManager(overrides?: Partial<{
  isRecording: boolean;
  recordingId: string | undefined;
  phase: string;
  actionCount: number;
  startedAt: string;
}>): MockPipelineManager {
  const defaults = {
    isRecording: false,
    recordingId: undefined,
    phase: 'ready',
    actionCount: 0,
    startedAt: '2024-01-01T00:00:00.000Z',
  };
  const config = { ...defaults, ...overrides };

  return {
    isRecording: jest.fn<boolean, []>().mockReturnValue(config.isRecording),
    getRecordingId: jest.fn<string | undefined, []>().mockReturnValue(config.recordingId),
    getRecordingData: jest.fn<{ startedAt: string } | undefined, []>().mockReturnValue(
      config.isRecording ? { startedAt: config.startedAt } : undefined
    ),
    getState: jest.fn<{
      phase: string;
      recording?: { recordingId?: string; actionCount: number; startedAt: string };
    }, []>().mockReturnValue({
      phase: config.isRecording ? 'capturing' : config.phase,
      recording: config.isRecording ? {
        recordingId: config.recordingId,
        actionCount: config.actionCount,
        startedAt: config.startedAt,
      } : undefined,
    }),
    startRecording: jest.fn<Promise<string>, []>()
      .mockResolvedValue(config.recordingId || 'recording-123'),
    stopRecording: jest.fn<Promise<{ recordingId: string; actionCount: number }>, []>()
      .mockResolvedValue({
      recordingId: config.recordingId || 'recording-123',
      actionCount: config.actionCount,
    }),
    validateSelector: jest.fn<Promise<{ valid: boolean; matchCount: number; selector: string; error?: string }>, []>()
      .mockResolvedValue({
      valid: true,
      matchCount: 1,
      selector: 'button#submit',
    }),
    getVerification: jest.fn<{
      loaded: boolean;
      ready: boolean;
      handlersCount: number;
      inMainContext: boolean;
      version: string;
    }, []>().mockReturnValue({
      loaded: true,
      ready: true,
      handlersCount: 10,
      inMainContext: true,
      version: '1.0.0',
    }),
  };
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
    pipelineManager: createMockPipelineManager(),
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
      const mockPipelineManager = createMockPipelineManager({
        isRecording: false,
        recordingId: 'recording-123',
      });

      const mockSession = {
        pipelineManager: mockPipelineManager,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ body: {} });
      const res = createMockResponse();

      await handleRecordStart(req, res, sessionId, sessionManager, config);

      expect(res._getStatusCode()).toBe(200);
      const data = parseResponse(res);
      expect(getStringField(data, 'recording_id')).toBe('recording-123');
      expect(getStringField(data, 'session_id')).toBe(sessionId);
      expect(getStringField(data, 'started_at')).toBeDefined();
    });

    it('should return 409 if already recording', async () => {
      const mockPipelineManager = createMockPipelineManager({
        isRecording: true,
        recordingId: 'existing-recording',
      });

      const mockSession = {
        pipelineManager: mockPipelineManager,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ body: {} });
      const res = createMockResponse();

      await handleRecordStart(req, res, sessionId, sessionManager, config);

      expect(res._getStatusCode()).toBe(409);
      const data = parseResponse(res);
      expect(getStringField(data, 'error')).toBe('RECORDING_IN_PROGRESS');
    });

    it('should use existing pipeline manager from session', async () => {
      const mockPipelineManager = createMockPipelineManager({
        isRecording: false,
        recordingId: 'new-recording-456',
      });

      const mockSession = {
        page: {
          url: jest.fn().mockReturnValue('https://example.com'),
        },
        pipelineManager: mockPipelineManager,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ body: {} });
      const res = createMockResponse();

      await handleRecordStart(req, res, sessionId, sessionManager, config);

      expect(res._getStatusCode()).toBe(200);
      // Pipeline manager should have been used
      expect(mockPipelineManager.startRecording).toHaveBeenCalled();
    });
  });

  describe('POST /session/:id/record/stop', () => {
    it('should stop recording successfully', async () => {
      const mockPipelineManager = createMockPipelineManager({
        isRecording: true,
        recordingId: 'recording-123',
        actionCount: 5,
      });

      const mockSession = {
        pipelineManager: mockPipelineManager,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({});
      const res = createMockResponse();

      await handleRecordStop(req, res, sessionId, sessionManager);

      expect(res._getStatusCode()).toBe(200);
      const data = parseResponse(res);
      expect(getStringField(data, 'recording_id')).toBe('recording-123');
      expect(getNumberField(data, 'action_count')).toBe(5);
      expect(getStringField(data, 'stopped_at')).toBeDefined();
    });

    it('should return 200 (idempotent) if not recording', async () => {
      // Idempotency: Calling stop when not recording is a successful no-op
      // This allows safe retries when the first stop request succeeded
      // but the response was lost due to network issues
      const mockPipelineManager = createMockPipelineManager({
        isRecording: false,
        recordingId: 'previous-recording', // From a prior recording
      });

      const mockSession = {
        pipelineManager: mockPipelineManager,
        phase: 'ready',
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({});
      const res = createMockResponse();

      await handleRecordStop(req, res, sessionId, sessionManager);

      // Idempotent: Returns success with action_count: 0
      expect(res._getStatusCode()).toBe(200);
      const data = parseResponse(res);
      expect(getStringField(data, 'recording_id')).toBe('previous-recording');
      expect(getNumberField(data, 'action_count')).toBe(0);
      expect(getStringField(data, 'stopped_at')).toBeDefined();
    });
  });

  describe('GET /session/:id/record/status', () => {
    it('should return recording status', () => {
      const mockPipelineManager = createMockPipelineManager({
        isRecording: true,
        recordingId: 'recording-123',
        actionCount: 3,
        startedAt: '2024-01-01T00:00:00.000Z',
      });

      const mockSession = {
        pipelineManager: mockPipelineManager,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ method: 'GET' });
      const res = createMockResponse();

      handleRecordStatus(req, res, sessionId, sessionManager);

      expect(res._getStatusCode()).toBe(200);
      const data = parseResponse(res);
      expect(getStringField(data, 'session_id')).toBe(sessionId);
      expect(getBooleanField(data, 'is_recording')).toBe(true);
      expect(getStringField(data, 'recording_id')).toBe('recording-123');
      expect(getNumberField(data, 'action_count')).toBe(3);
    });

    it('should handle no pipeline manager', () => {
      const mockSession = {
        pipelineManager: null,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ method: 'GET' });
      const res = createMockResponse();

      handleRecordStatus(req, res, sessionId, sessionManager);

      expect(res._getStatusCode()).toBe(200);
      const data = parseResponse(res);
      expect(getBooleanField(data, 'is_recording')).toBe(false);
      expect(getNumberField(data, 'action_count')).toBe(0);
    });
  });

  describe('GET /session/:id/record/actions', () => {
    it('should return buffered actions as TimelineEntry format', () => {
      const sessionManager = createMockSessionManager();

      // Manually add actions to buffer for this test
      // This simulates actions being recorded
      const req = createMockRequest({ method: 'GET', url: '/session/test/record/actions' });
      const res = createMockResponse();

      handleRecordActions(req, res, sessionId, sessionManager);

      expect(res._getStatusCode()).toBe(200);
      const data = parseResponse(res);
      expect(getStringField(data, 'session_id')).toBe(sessionId);
      // Now returns 'entries' (TimelineEntry format) instead of 'actions'
      const entries = data.entries;
      const count = getNumberField(data, 'count');
      expect(Array.isArray(entries)).toBe(true);
      expect(typeof count).toBe('number');
    });

    it('should clear buffer when clear=true', () => {
      const sessionManager = createMockSessionManager();

      const req = createMockRequest({
        method: 'GET',
        url: '/session/test/record/actions?clear=true',
      });
      const res = createMockResponse();

      handleRecordActions(req, res, sessionId, sessionManager);

      expect(res._getStatusCode()).toBe(200);
      const data = parseResponse(res);
      // Now returns 'entries' (TimelineEntry format) instead of 'actions'
      expect(data.entries).toEqual([]);
    });
  });

  describe('POST /session/:id/record/validate-selector', () => {
    it('should validate CSS selector', async () => {
      const mockPipelineManager = createMockPipelineManager();
      mockPipelineManager.validateSelector.mockResolvedValue({
        valid: true,
        matchCount: 1,
        selector: 'button#submit',
      });

      const mockSession = {
        page: { url: jest.fn().mockReturnValue('https://example.com') },
        pipelineManager: mockPipelineManager,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ body: { selector: 'button#submit' } });
      const res = createMockResponse();

      await handleValidateSelector(req, res, sessionId, sessionManager, config);

      expect(res._getStatusCode()).toBe(200);
      const data = parseResponse(res);
      expect(getBooleanField(data, 'valid')).toBe(true);
      expect(getNumberField(data, 'match_count')).toBe(1);
    });

    it('should return 400 if selector missing', async () => {
      const sessionManager = createMockSessionManager();
      const req = createMockRequest({ body: {} });
      const res = createMockResponse();

      await handleValidateSelector(req, res, sessionId, sessionManager, config);

      expect(res._getStatusCode()).toBe(400);
      const data = parseResponse(res);
      expect(getStringField(data, 'error')).toBe('MISSING_SELECTOR');
    });

    it('should report invalid selector', async () => {
      const mockPipelineManager = createMockPipelineManager();
      mockPipelineManager.validateSelector.mockResolvedValue({
        valid: false,
        matchCount: 0,
        selector: 'div.nonexistent',
        error: 'No elements found',
      });

      const mockSession = {
        page: { url: jest.fn().mockReturnValue('https://example.com') },
        pipelineManager: mockPipelineManager,
      };

      const sessionManager = createMockSessionManager(mockSession);
      const req = createMockRequest({ body: { selector: 'div.nonexistent' } });
      const res = createMockResponse();

      await handleValidateSelector(req, res, sessionId, sessionManager, config);

      expect(res._getStatusCode()).toBe(200);
      const data = parseResponse(res);
      expect(getBooleanField(data, 'valid')).toBe(false);
      expect(getNumberField(data, 'match_count')).toBe(0);
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
