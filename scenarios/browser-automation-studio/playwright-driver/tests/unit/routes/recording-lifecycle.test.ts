import * as frameStreaming from '../../../src/frame-streaming';
import * as pageEvents from '../../../src/routes/record-mode/page-events';
import { SessionNotFoundError } from '../../../src/utils';
import { handleRecordStart, handleRecordStatus, handleRecordStop } from '../../../src/routes/record-mode/recording-lifecycle';
import type { SessionManager } from '../../../src/session';
import { createMockHttpRequest, createMockHttpResponse, createTestConfig } from '../../helpers';

type PipelineManagerStub = {
  getRecordingId: jest.Mock;
  getRecordingData: jest.Mock;
  getState: jest.Mock;
  isRecording: jest.Mock;
  stopRecording: jest.Mock;
};

function createPipelineManager(overrides?: Partial<PipelineManagerStub>): PipelineManagerStub {
  return {
    getRecordingId: jest.fn().mockReturnValue('recording-123'),
    getRecordingData: jest.fn().mockReturnValue(undefined),
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
    it('retains the committed result when stop is retried', async () => {
      const pipelineManager = createPipelineManager({
        getRecordingId: jest.fn().mockReturnValue('recording-previous'),
        getRecordingData: jest.fn().mockReturnValue({ actionCount: 7, stoppedAt: '2026-09-22T00:00:00.000Z' }),
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
      expect(sessionManager.setSessionPhase).toHaveBeenCalledWith('session-123', 'ready');
      expect(res.statusCode).toBe(200);
      expect(res.getJSON()).toMatchObject({
        recording_id: 'recording-previous',
        session_id: 'session-123',
        action_count: 7,
        stopped_at: '2026-09-22T00:00:00.000Z',
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

describe('recording start continuation ownership [REQ:BAS-RH-J17]', () => {
  const config = createTestConfig();
  function deferred() {
    let resolve!: () => void;
    const promise = new Promise<void>((done) => { resolve = done; });
    return { promise, resolve };
  }
  function fixture() {
    const state = { phase: 'ready', generation: 0 };
    const pipeline = {
      isRecording: jest.fn(() => state.phase === 'capturing'),
      getState: () => ({ phase: state.phase }),
      getGeneration: () => state.generation,
      getRecordingId: () => 'same-public-id',
      getRecordingData: () => ({ recordingId: 'same-public-id', generation: state.generation, actionCount: 0, startedAt: 'fixture-start' }),
      getVerification: () => undefined,
      startRecording: jest.fn(async () => { state.phase = 'capturing'; state.generation++; return 'same-public-id'; }),
      stopRecording: jest.fn(async () => { state.phase = 'ready'; return { recordingId: 'same-public-id', actionCount: 0 }; }),
    };
    const page = { url: () => 'https://fixture.invalid', waitForLoadState: jest.fn().mockResolvedValue(undefined) };
    const session = { phase: 'ready', ownerExecutionId: 'owner-1', leaseId: 'lease-1', page, pipelineManager: pipeline,
      pageLifecycleCleanup: undefined as (() => void) | undefined };
    const manager = {
      getSession: () => session,
      getSessionForLease: (_id: string, executionId: string, leaseId: string) => {
        if (executionId !== session.ownerExecutionId || leaseId !== session.leaseId) throw new SessionNotFoundError('recording-session');
        return session;
      },
      setSessionPhase: jest.fn((_id: string, phase: string) => { session.phase = phase; return true; }),
    } as unknown as SessionManager;
    const start = (body = { frame_callback_url: 'http://fixture.invalid/frames' }) => {
      const response = createMockHttpResponse();
      const finished = handleRecordStart(createMockHttpRequest({ method: 'POST', body }), response, 'recording-session', manager, config);
      return { response, finished };
    };
    return { state, pipeline, page, session, manager, start };
  }
  beforeEach(() => {
    jest.spyOn(frameStreaming, 'startFrameStreaming').mockImplementation(() => {});
    jest.spyOn(frameStreaming, 'stopFrameStreaming').mockResolvedValue(undefined);
  });
  afterEach(() => jest.restoreAllMocks());

  it('starts preview after pipeline readiness without waiting again for DOM load', async () => {
    const f = fixture(); const dom = deferred();
    f.page.waitForLoadState.mockReturnValue(dom.promise);
    const { finished, response } = f.start();
    await new Promise<void>((resolve) => setImmediate(resolve));
    dom.resolve(); await finished;
    expect(f.page.waitForLoadState).not.toHaveBeenCalled();
    expect(response.statusCode).toBe(200);
    expect(frameStreaming.startFrameStreaming).toHaveBeenCalledTimes(1);
  });

  it.each(['ready', 'stopping'])('rejects a start whose pipeline became %s before its completion', async (phase) => {
    const f = fixture();
    f.pipeline.startRecording.mockImplementation(async () => { f.state.generation++; f.state.phase = phase; return 'same-public-id'; });
    const { finished, response } = f.start(); await finished;
    expect(response.statusCode).toBe(409);
    expect(frameStreaming.startFrameStreaming).not.toHaveBeenCalled();
    expect(f.manager.setSessionPhase).not.toHaveBeenCalled();
  });

  it('rejects ownership transferred while the body was being read', async () => {
    const f = fixture();
    const { finished, response } = f.start();
    f.session.ownerExecutionId = 'owner-2'; f.session.leaseId = 'lease-2';
    await finished;
    expect(response.statusCode).toBe(404);
    expect(f.pipeline.startRecording).not.toHaveBeenCalled();
    expect(frameStreaming.startFrameStreaming).not.toHaveBeenCalled();
  });

  it('rejects ownership transferred during pipeline startup', async () => {
    const f = fixture(); const entered = deferred(); const proceed = deferred();
    f.pipeline.startRecording.mockImplementation(async () => {
      entered.resolve(); await proceed.promise;
      f.state.generation++; f.state.phase = 'capturing'; return 'same-public-id';
    });
    const { finished, response } = f.start(); await entered.promise;
    f.session.ownerExecutionId = 'owner-2'; f.session.leaseId = 'lease-2'; proceed.resolve(); await finished;
    expect(response.statusCode).toBe(404);
    expect(frameStreaming.startFrameStreaming).not.toHaveBeenCalled();
    expect(f.manager.setSessionPhase).not.toHaveBeenCalled();
  });

  it.each(['generation', 'lease', 'phase'] as const)('binds future frame page lookups to the admitted recording %s', async (changed) => {
    const f = fixture(); const { finished, response } = f.start(); await finished;
    expect(response.statusCode).toBe(200);
    const provider = jest.mocked(frameStreaming.startFrameStreaming).mock.calls[0][1];
    expect(provider.getSession('recording-session').page).toBe(f.page);
    if (changed === 'generation') f.state.generation++; // Same public recording ID, new capture.
    if (changed === 'lease') f.session.leaseId = 'lease-2';
    if (changed === 'phase') f.session.phase = 'resetting';
    expect(() => provider.getSession('recording-session')).toThrow();
  });

  it('does not accept a newer generation with the same public ID as completion of the admitted start', async () => {
    const f = fixture();
    f.pipeline.startRecording.mockImplementation(async () => {
      f.state.generation += 2; f.state.phase = 'capturing'; return 'same-public-id';
    });
    const { finished, response } = f.start(); await finished;
    expect(response.statusCode).toBe(409);
    expect(frameStreaming.startFrameStreaming).not.toHaveBeenCalled();
    expect(frameStreaming.stopFrameStreaming).not.toHaveBeenCalled();
  });

  it('preserves idempotent retry of the active recording without starting another generation', async () => {
    const f = fixture(); f.state.phase = 'capturing'; f.state.generation = 5;
    const response = createMockHttpResponse();
    await handleRecordStart(createMockHttpRequest({ method: 'POST', body: { recording_id: 'same-public-id' } }),
      response, 'recording-session', f.manager, config);
    expect(response.statusCode).toBe(200);
    expect(response.getJSON()).toMatchObject({ recording_id: 'same-public-id', started_at: 'fixture-start' });
    expect(f.pipeline.startRecording).not.toHaveBeenCalled();
    expect(frameStreaming.startFrameStreaming).not.toHaveBeenCalled();
    expect(f.state.generation).toBe(5);
  });

  it('does not report current success after stop wins during initial page callback readiness', async () => {
    const f = fixture(); const ready = deferred(); const attached = deferred(); const cleanup = jest.fn();
    jest.spyOn(pageEvents, 'setupPageLifecycleListeners').mockImplementation(() => {
      attached.resolve(); return { cleanup, ready: ready.promise };
    });
    const response = createMockHttpResponse();
    const start = handleRecordStart(createMockHttpRequest({ method: 'POST', body: { page_callback_url: 'http://fixture.invalid/pages' } }),
      response, 'recording-session', f.manager, config);
    await attached.promise;
    const stopResponse = createMockHttpResponse();
    await handleRecordStop(createMockHttpRequest(), stopResponse, 'recording-session', f.manager);
    ready.resolve(); await start;
    expect(stopResponse.statusCode).toBe(200);
    expect(cleanup).toHaveBeenCalledTimes(1);
    expect(response.statusCode).toBe(409);
    expect(f.session.phase).toBe('ready');
  });
});
