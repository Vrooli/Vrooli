import { handleRecordStatus, handleRecordStop } from '../../../src/routes/record-mode/recording-lifecycle';
import type { SessionManager } from '../../../src/session';
import { createMockHttpRequest, createMockHttpResponse } from '../../helpers';

type PipelineManagerStub = {
  getRecordingId: jest.Mock;
  getState: jest.Mock;
  isRecording: jest.Mock;
  stopRecording: jest.Mock;
};

function createPipelineManager(overrides?: Partial<PipelineManagerStub>): PipelineManagerStub {
  return {
    getRecordingId: jest.fn().mockReturnValue('recording-123'),
    getState: jest.fn().mockReturnValue(undefined),
    isRecording: jest.fn().mockReturnValue(false),
    stopRecording: jest.fn(),
    ...overrides,
  };
}

function createSessionManager(session: Record<string, unknown>): SessionManager {
  return {
    getSession: jest.fn(() => session as ReturnType<SessionManager['getSession']>),
    setSessionPhase: jest.fn(),
  } as unknown as SessionManager;
}

describe('recording lifecycle routes', () => {
  afterEach(() => {
    jest.clearAllMocks();
  });

  describe('handleRecordStatus', () => {
    it('reports pipeline recording state using the public response shape', () => {
      const pipelineManager = createPipelineManager({
        getState: jest.fn().mockReturnValue({
          phase: 'capturing',
          recording: {
            recordingId: 'recording-abc',
            actionCount: 7,
            startedAt: '2026-05-01T12:00:00.000Z',
          },
        }),
      });
      const sessionManager = createSessionManager({ pipelineManager });
      const req = createMockHttpRequest({
        method: 'GET',
        url: '/session/session-123/record/status',
      });
      const res = createMockHttpResponse();

      handleRecordStatus(req, res, 'session-123', sessionManager);

      expect(res.statusCode).toBe(200);
      expect(res.getJSON()).toEqual({
        session_id: 'session-123',
        is_recording: true,
        recording_id: 'recording-abc',
        action_count: 7,
        started_at: '2026-05-01T12:00:00.000Z',
      });
    });

    it('falls back to an idle response when no pipeline manager is attached', () => {
      const sessionManager = createSessionManager({});
      const req = createMockHttpRequest({
        method: 'GET',
        url: '/session/session-idle/record/status',
      });
      const res = createMockHttpResponse();

      handleRecordStatus(req, res, 'session-idle', sessionManager);

      expect(res.statusCode).toBe(200);
      expect(res.getJSON()).toEqual({
        session_id: 'session-idle',
        is_recording: false,
        action_count: 0,
      });
    });
  });

  describe('handleRecordStop', () => {
    it('treats repeated stop requests as successful no-ops', async () => {
      const pipelineManager = createPipelineManager({
        getRecordingId: jest.fn().mockReturnValue('recording-previous'),
        isRecording: jest.fn().mockReturnValue(false),
      });
      const sessionManager = createSessionManager({
        phase: 'ready',
        pipelineManager,
      });
      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/session-123/record/stop',
      });
      const res = createMockHttpResponse();

      await handleRecordStop(req, res, 'session-123', sessionManager);

      expect(pipelineManager.stopRecording).not.toHaveBeenCalled();
      expect(sessionManager.setSessionPhase).not.toHaveBeenCalled();
      expect(res.statusCode).toBe(200);
      expect(res.getJSON()).toMatchObject({
        recording_id: 'recording-previous',
        session_id: 'session-123',
        action_count: 0,
      });
      expect(res.getJSON().stopped_at).toEqual(expect.any(String));
    });

    it('stops active recordings, cleans page listeners, and resets the session phase', async () => {
      const pageLifecycleCleanup = jest.fn();
      const pipelineManager = createPipelineManager({
        getRecordingId: jest.fn().mockReturnValue('recording-active'),
        isRecording: jest.fn().mockReturnValue(true),
        stopRecording: jest.fn().mockResolvedValue({
          recordingId: 'recording-active',
          actionCount: 3,
        }),
      });
      const session = {
        phase: 'recording',
        page: {
          url: jest.fn().mockReturnValue('https://example.com'),
        },
        pageLifecycleCleanup,
        pipelineManager,
      };
      const sessionManager = createSessionManager(session);
      const req = createMockHttpRequest({
        method: 'POST',
        url: '/session/session-123/record/stop',
      });
      const res = createMockHttpResponse();

      await handleRecordStop(req, res, 'session-123', sessionManager);

      expect(pipelineManager.stopRecording).toHaveBeenCalledTimes(1);
      expect(pageLifecycleCleanup).toHaveBeenCalledTimes(1);
      expect(session.pageLifecycleCleanup).toBeUndefined();
      expect(sessionManager.setSessionPhase).toHaveBeenCalledWith('session-123', 'ready');
      expect(res.statusCode).toBe(200);
      expect(res.getJSON()).toMatchObject({
        recording_id: 'recording-active',
        session_id: 'session-123',
        action_count: 3,
      });
      expect(res.getJSON().stopped_at).toEqual(expect.any(String));
    });
  });
});
