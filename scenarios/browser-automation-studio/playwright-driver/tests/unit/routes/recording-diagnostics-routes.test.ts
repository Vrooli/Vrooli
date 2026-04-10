import type { BrowserContext } from 'rebrowser-playwright';
import type { SessionManager } from '../../../src/session';
import {
  handleStreamSettings,
  handleRecordDebug,
  handleRecordPipelineTest,
  handleRecordExternalUrlTest,
} from '../../../src/routes/record-mode/recording-diagnostics-routes';
import {
  createMockHttpRequest,
  createMockHttpResponse,
  createTestConfig,
} from '../../helpers';

const updateFrameStreamSettings = jest.fn();
const getFrameStreamSettings = jest.fn();

jest.mock('../../../src/frame-streaming', () => ({
  updateFrameStreamSettings: (...args: unknown[]) => updateFrameStreamSettings(...args),
  getFrameStreamSettings: (...args: unknown[]) => getFrameStreamSettings(...args),
}));

const runRecordingPipelineTest = jest.fn();
const runExternalUrlInjectionTest = jest.fn();

jest.mock('../../../src/recording', () => ({
  runRecordingPipelineTest: (...args: unknown[]) => runRecordingPipelineTest(...args),
  runExternalUrlInjectionTest: (...args: unknown[]) => runExternalUrlInjectionTest(...args),
}));

describe('recording diagnostics routes', () => {
  const config = createTestConfig();

  beforeEach(() => {
    updateFrameStreamSettings.mockClear();
    getFrameStreamSettings.mockClear();
    runRecordingPipelineTest.mockClear();
    runExternalUrlInjectionTest.mockClear();
  });

  it('returns default stream settings when no stream is active', async () => {
    getFrameStreamSettings.mockReturnValueOnce(null);

    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/stream-settings',
      body: { quality: 70, fps: 25 },
    });
    const res = createMockHttpResponse();
    const sessionManager: Pick<SessionManager, 'getSession'> = {
      getSession: () => ({} as ReturnType<SessionManager['getSession']>),
    };

    await handleStreamSettings(req, res, 'test', sessionManager as SessionManager, config);

    expect(res.statusCode).toBe(200);
    const payload = res.getJSON();
    expect(payload.is_streaming).toBe(false);
    expect(payload.updated).toBe(false);
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
      body: { quality: 60, fps: 30, scale: 'device', perfMode: true },
    });
    const res = createMockHttpResponse();
    const sessionManager: Pick<SessionManager, 'getSession'> = {
      getSession: () => ({} as ReturnType<SessionManager['getSession']>),
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

    const sessionManager: Pick<SessionManager, 'getSession'> = {
      getSession: () => session as ReturnType<SessionManager['getSession']>,
    };

    const res = createMockHttpResponse();
    await handleRecordDebug({} as ReturnType<typeof createMockHttpRequest>, res, 'test', sessionManager as SessionManager);

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

    const sessionManager: Pick<SessionManager, 'getSession'> = {
      getSession: () => session as ReturnType<SessionManager['getSession']>,
    };

    const res = createMockHttpResponse();
    await handleRecordDebug({} as ReturnType<typeof createMockHttpRequest>, res, 'test', sessionManager as SessionManager);

    const payload = res.getJSON();
    expect(payload.browser_script).toBeNull();
  });

  it('returns errors when pipeline prerequisites are missing', async () => {
    const sessionManager: Pick<SessionManager, 'getSession'> = {
      getSession: () => ({ page: { url: jest.fn().mockReturnValue('about:blank') } } as ReturnType<SessionManager['getSession']>),
    };

    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/pipeline-test',
      body: {},
    });
    const res = createMockHttpResponse();

    await handleRecordPipelineTest(req, res, 'test', sessionManager as SessionManager, config);

    expect(res.statusCode).toBe(500);
    expect(res.getJSON().error).toBe('MISSING_INITIALIZER');
  });

  it('runs pipeline test and returns response', async () => {
    runRecordingPipelineTest.mockResolvedValue({
      success: true,
      timestamp: '2026-01-01T00:00:00.000Z',
      durationMs: 100,
      steps: [{ name: 'step', passed: true, durationMs: 50 }],
      diagnostics: {
        testPageUrl: 'https://example.com',
        testPageInjected: true,
        scriptStatusBefore: null,
        scriptStatusAfter: null,
        telemetryBefore: null,
        telemetryAfter: null,
        routeStatsBefore: null,
        routeStatsAfter: null,
        eventsCaptured: 2,
        consoleMessages: [],
      },
    });

    const page = {
      url: jest.fn().mockReturnValue('https://example.com'),
      goto: jest.fn().mockResolvedValue(undefined),
    };

    const sessionManager: Pick<SessionManager, 'getSession'> = {
      getSession: () => ({
        page,
        context: {} as BrowserContext,
        recordingInitializer: {},
        pipelineManager: {},
      } as ReturnType<SessionManager['getSession']>),
    };

    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/pipeline-test',
      body: {},
    });
    const res = createMockHttpResponse();

    await handleRecordPipelineTest(req, res, 'test', sessionManager as SessionManager, config);

    expect(res.statusCode).toBe(200);
    expect(res.getJSON().success).toBe(true);
    expect(page.goto).toHaveBeenCalled();
  });

  it('returns errors when external URL test prerequisites are missing', async () => {
    const sessionManager: Pick<SessionManager, 'getSession'> = {
      getSession: () => ({ page: { url: jest.fn().mockReturnValue('about:blank') } } as ReturnType<SessionManager['getSession']>),
    };

    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/external-url-test',
      body: {},
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

    const sessionManager: Pick<SessionManager, 'getSession'> = {
      getSession: () => ({
        page,
        recordingInitializer: {},
      } as ReturnType<SessionManager['getSession']>),
    };

    const req = createMockHttpRequest({
      method: 'POST',
      url: '/session/test/record/external-url-test',
      body: { test_url: 'https://example.com' },
    });
    const res = createMockHttpResponse();

    await handleRecordExternalUrlTest(req, res, 'test', sessionManager as SessionManager, config);

    expect(res.statusCode).toBe(200);
    expect(res.getJSON().success).toBe(true);
    expect(page.goto).toHaveBeenCalled();
  });
});
