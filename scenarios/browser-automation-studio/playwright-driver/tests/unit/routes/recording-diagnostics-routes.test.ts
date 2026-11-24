import type { BrowserContext } from 'rebrowser-playwright';
import type { SessionManager } from '../../../src/session';
import {
  handleRecordDebug,
  handleRecordExternalUrlTest,
} from '../../../src/routes/record-mode/recording-diagnostics-routes';
import { handleStreamSettings } from '../../../src/routes/record-mode/recording-stream-settings-route';
import {
  createMockHttpRequest,
  createMockHttpResponse,
  createTestConfig,
} from '../../helpers';

const updateFrameStreamSettings = jest.fn();
const getFrameStreamSettings = jest.fn();
const ownedSession = { phase: 'ready', page: { url: () => 'https://example.com/' }, createdAt: new Date(), instructionCount: 0 };

jest.mock('../../../src/frame-streaming', () => ({
  updateFrameStreamSettings: (...args: unknown[]) => updateFrameStreamSettings(...args),
  getFrameStreamSettings: (...args: unknown[]) => getFrameStreamSettings(...args),
}));

const runExternalUrlInjectionTest = jest.fn();

jest.mock('../../../src/recording/testing/external-url-injection-test', () => ({
  runExternalUrlInjectionTest: (...args: unknown[]) => runExternalUrlInjectionTest(...args),
}));

describe('recording diagnostics routes', () => {
  const config = createTestConfig();

  beforeEach(() => {
    updateFrameStreamSettings.mockClear();
    getFrameStreamSettings.mockClear();
    runExternalUrlInjectionTest.mockClear();
  });

  it.each([
    { quality: 'bad' }, { quality: 20.5 }, { fps: null }, { perfMode: 'true' }, { scale: 'invalid' },
  ])('rejects malformed stream settings before invoking capture: %j', async (body) => {
    getFrameStreamSettings.mockReset().mockReturnValue({ quality: 65, fps: 30, currentFps: 0, scale: 'css', isStreaming: true, perfMode: false });
    updateFrameStreamSettings.mockResolvedValue(true);
    const req = createMockHttpRequest({ method: 'POST', body: { ...body, execution_id: 'owner', lease_id: 'lease' } });
    const res = createMockHttpResponse();
    await handleStreamSettings(req, res, 'test', { getSessionForLease: () => ownedSession } as unknown as SessionManager, config);
    expect(res.statusCode).toBe(400);
    expect(updateFrameStreamSettings).not.toHaveBeenCalled();
  });

  it('returns default stream settings when no stream is active', async () => {
    getFrameStreamSettings.mockReturnValueOnce(null);

    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/stream-settings',
      body: { execution_id: 'owner', lease_id: 'lease', quality: 70, fps: 25 },
    });
    const res = createMockHttpResponse();
    const sessionManager: Pick<SessionManager, 'getSessionForLease'> = {
      getSessionForLease: () => ownedSession as ReturnType<SessionManager['getSession']>,
    };

    await handleStreamSettings(req, res, 'test', sessionManager as SessionManager, config);

    expect(res.statusCode).toBe(200);
    const payload = res.getJSON();
    expect(payload.is_streaming).toBe(false);
    expect(payload.updated).toBe(false);
    expect(updateFrameStreamSettings).not.toHaveBeenCalled();
  });

  it('denies out-of-scope stream control before updating the stream', async () => {
    const session = { ...ownedSession, spec: { permission_profile: {
      effect: 'act', approval: 'none', view: true, control: true, url_allow: ['https://other.example/*'],
    } } } as ReturnType<SessionManager['getSession']>;
    const manager: Pick<SessionManager, 'getSessionForLease'> = {
      getSessionForLease: () => session,
    };
    const req = createMockHttpRequest({ method: 'POST', body: { execution_id: 'owner', lease_id: 'lease', quality: 70 } });
    const res = createMockHttpResponse();
    await handleStreamSettings(req, res, 'test', manager as SessionManager, config);
    expect(res.statusCode).toBe(403);
    expect(updateFrameStreamSettings).not.toHaveBeenCalled();
  });

  it('updates stream settings and warns on scale changes', async () => {
    getFrameStreamSettings
      .mockReturnValueOnce({
        quality: 50,
        fps: 20,
        currentFps: 15,
        scale: 'css',
        isStreaming: true,
        perfMode: false,
      })
      .mockReturnValueOnce({
        quality: 60,
        fps: 30,
        currentFps: 28,
        scale: 'css',
        isStreaming: true,
        perfMode: true,
      });
    updateFrameStreamSettings.mockReturnValue(true);

    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/stream-settings',
      body: { execution_id: 'owner', lease_id: 'lease', quality: 60, fps: 30, scale: 'device', perfMode: true },
    });
    const res = createMockHttpResponse();
    const sessionManager: Pick<SessionManager, 'getSessionForLease'> = {
      getSessionForLease: () => ownedSession as ReturnType<SessionManager['getSession']>,
    };

    await handleStreamSettings(req, res, 'test', sessionManager as SessionManager, config);

    const payload = res.getJSON();
    expect(payload.updated).toBe(true);
    expect(payload.scale_warning).toContain('Scale cannot be changed');
    expect(payload.quality).toBe(60);
    expect(payload.perf_mode).toBe(true);
  });

  it('returns debug state with browser script telemetry', async () => {
    const cdpSession = {
      send: jest.fn().mockResolvedValue({
        result: {
          type: 'string',
          value: JSON.stringify({
            loaded: true,
            ready: true,
            isActive: true,
            inMainContext: true,
            handlersCount: 2,
            version: '2.0.0',
            eventsDetected: 3,
            eventsCaptured: 3,
            eventsSent: 1,
            eventsSendFailed: 0,
            lastError: null,
            serviceWorkerActive: true,
            serviceWorkerUrl: 'https://example.com/sw.js',
          }),
        },
      }),
      detach: jest.fn().mockResolvedValue(undefined),
    };

    const session = {
      phase: 'recording',
      page: {
        url: jest.fn().mockReturnValue('https://example.com'),
        context: jest.fn().mockReturnValue({
          newCDPSession: jest.fn().mockResolvedValue(cdpSession),
        }),
      },
      pipelineManager: {
        getState: () => ({
          phase: 'capturing',
          recording: { recordingId: 'rec-1', actionCount: 2, generation: 1 },
          verification: {
            scriptLoaded: true,
            scriptReady: true,
            inMainContext: true,
            handlersCount: 2,
            eventRouteActive: true,
            verifiedAt: '2026-01-01T00:00:00.000Z',
          },
          error: null,
        }),
      },
      recordingInitializer: {
        hasEventHandler: () => true,
        getRouteHandlerStats: () => ({
          eventsReceived: 0,
          eventsProcessed: 1,
          eventsDroppedNoHandler: 1,
          eventsWithErrors: 0,
          lastEventAt: '2026-01-01T00:00:00.000Z',
          lastEventType: 'click',
        }),
        getInjectionStats: () => ({
          attempted: 1,
          successful: 1,
          failed: 0,
          avgInjectionTimeMs: 50,
          lastInjectionAt: '2026-01-01T00:00:00.000Z',
        }),
      },
    };

    const sessionManager: Pick<SessionManager, 'getSessionForLease'> = {
      getSessionForLease: () => session as ReturnType<SessionManager['getSessionForLease']>,
    };

    const res = createMockHttpResponse();
    await handleRecordDebug(createMockHttpRequest({ headers: { 'x-bas-execution-id': 'owner', 'x-bas-session-lease-id': 'lease' } }), res, 'test', sessionManager as SessionManager);

    expect(res.statusCode).toBe(200);
    const payload = res.getJSON();
    expect(payload.server.is_recording).toBe(true);
    expect(payload.browser_script?.loaded).toBe(true);
    expect(payload.diagnostics.service_worker_blocking).toBe(true);
  });

  it('handles debug responses when CDP inspection fails', async () => {
    const session = {
      phase: 'recording',
      page: {
        url: jest.fn().mockReturnValue('https://example.com'),
        context: jest.fn().mockReturnValue({
          newCDPSession: jest.fn().mockRejectedValue(new Error('cdp failed')),
        }),
      },
      pipelineManager: {
        getState: () => ({ phase: 'idle', recording: null, verification: null, error: null }),
      },
      recordingInitializer: {
        hasEventHandler: () => false,
        getRouteHandlerStats: () => null,
        getInjectionStats: () => null,
      },
    };

    const sessionManager: Pick<SessionManager, 'getSessionForLease'> = {
      getSessionForLease: () => session as ReturnType<SessionManager['getSessionForLease']>,
    };

    const res = createMockHttpResponse();
    await handleRecordDebug(createMockHttpRequest({ headers: { 'x-bas-execution-id': 'owner', 'x-bas-session-lease-id': 'lease' } }), res, 'test', sessionManager as SessionManager);

    const payload = res.getJSON();
    expect(payload.browser_script).toBeNull();
  });

  it('returns errors when external URL test prerequisites are missing', async () => {
    const session = { phase: 'ready', page: { url: jest.fn().mockReturnValue('about:blank') } } as ReturnType<SessionManager['getSession']>;
    const sessionManager: Pick<SessionManager, 'getSessionForLease'> = {
      getSessionForLease: () => session,
    };

    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/external-url-test',
      body: { execution_id: 'owner', lease_id: 'lease' },
    });
    const res = createMockHttpResponse();

    await handleRecordExternalUrlTest(req, res, 'test', sessionManager as SessionManager, config);

    expect(res.statusCode).toBe(500);
    expect(res.getJSON().error).toBe('MISSING_INITIALIZER');
  });

  it('runs external URL test and returns response', async () => {
    runExternalUrlInjectionTest.mockResolvedValue({
      success: true,
      timestamp: '2026-01-01T00:00:00.000Z',
      durationMs: 200,
      testedUrl: 'https://example.com',
      verification: { handlersCount: 2 },
      injectionStats: { attempted: 1, successful: 1, failed: 0 },
    });

    const page = {
      url: jest.fn().mockReturnValue('https://example.com'),
      goto: jest.fn().mockResolvedValue(undefined),
    };

    const session = {
        phase: 'ready',
        page,
        recordingInitializer: {},
      } as ReturnType<SessionManager['getSession']>;
    const sessionManager: Pick<SessionManager, 'getSessionForLease'> = {
      getSessionForLease: () => session,
    };

    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/external-url-test',
      body: { execution_id: 'owner', lease_id: 'lease', test_url: 'https://example.com' },
    });
    const res = createMockHttpResponse();

    await handleRecordExternalUrlTest(req, res, 'test', sessionManager as SessionManager, config);

    expect(res.statusCode).toBe(200);
    expect(res.getJSON().success).toBe(true);
    expect(page.goto).toHaveBeenCalled();
  });

  it('denies out-of-scope external URL injection before the test runner', async () => {
    const page = { url: jest.fn().mockReturnValue('https://example.com') };
    const session = { phase: 'ready', createdAt: new Date(), instructionCount: 0, page, recordingInitializer: {}, spec: { permission_profile: {
      effect: 'act', approval: 'none', view: true, control: true, url_allow: ['https://example.com/*'],
    } } } as unknown as ReturnType<SessionManager['getSession']>;
    const manager: Pick<SessionManager, 'getSessionForLease'> = { getSessionForLease: () => session };
    const req = createMockHttpRequest({ method: 'POST', body: { execution_id: 'owner', lease_id: 'lease', test_url: 'https://outside.example/' } });
    const res = createMockHttpResponse();
    await handleRecordExternalUrlTest(req, res, 'test', manager as SessionManager, config);
    expect(res.statusCode).toBe(403);
    expect(runExternalUrlInjectionTest).not.toHaveBeenCalled();
  });
});
