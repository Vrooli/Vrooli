import { runExternalUrlInjectionTest, runRecordingPipelineTest } from '../../../src/recording/testing/self-test';
import type { RecordingContextInitializer } from '../../../src/recording/io/context-initializer';
import type { RecordingPipelineManager } from '../../../src/recording/orchestration/pipeline-manager';

const mockVerifyScriptInjection = jest.fn();

jest.mock('../../../src/recording/validation/verification', () => ({
  verifyScriptInjection: (...args: unknown[]) => mockVerifyScriptInjection(...args),
  waitForScriptReady: jest.fn(),
}));

describe('recording self-test helpers', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('reports failure when external injection is not attempted', async () => {
    jest.useFakeTimers();
    const page = {
      goto: jest.fn().mockResolvedValue(undefined),
    };
    const contextInitializer = {
      getInjectionStats: jest
        .fn()
        .mockReturnValueOnce({ attempted: 0, successful: 0, failed: 0 })
        .mockReturnValueOnce({ attempted: 0, successful: 0, failed: 0 }),
    } as unknown as RecordingContextInitializer;

    mockVerifyScriptInjection.mockResolvedValue({
      loaded: true,
      ready: true,
      inMainContext: true,
      handlersCount: 1,
      version: '1.0.0',
    });

    const promise = runExternalUrlInjectionTest(page as never, contextInitializer);
    await jest.runAllTimersAsync();
    const result = await promise;

    expect(result.success).toBe(false);
    expect(result.failurePoint).toBe('fetch');

    jest.useRealTimers();
  });

  it('reports failure when script fails to load after injection', async () => {
    jest.useFakeTimers();
    const page = {
      goto: jest.fn().mockResolvedValue(undefined),
    };
    const contextInitializer = {
      getInjectionStats: jest
        .fn()
        .mockReturnValueOnce({ attempted: 0, successful: 0, failed: 0 })
        .mockReturnValueOnce({ attempted: 1, successful: 1, failed: 0 }),
    } as unknown as RecordingContextInitializer;

    mockVerifyScriptInjection.mockResolvedValue({
      loaded: false,
      ready: false,
      inMainContext: true,
      handlersCount: 0,
      version: null,
      error: 'not loaded',
    });

    const promise = runExternalUrlInjectionTest(page as never, contextInitializer, { testUrl: 'https://example.com' });
    await jest.runAllTimersAsync();
    const result = await promise;

    expect(result.success).toBe(false);
    expect(result.failurePoint).toBe('script_load');

    jest.useRealTimers();
  });

  it('reports failure when injection attempt fails', async () => {
    jest.useFakeTimers();
    const page = {
      goto: jest.fn().mockResolvedValue(undefined),
    };
    const contextInitializer = {
      getInjectionStats: jest
        .fn()
        .mockReturnValueOnce({ attempted: 0, successful: 0, failed: 0 })
        .mockReturnValueOnce({ attempted: 1, successful: 0, failed: 1 }),
    } as unknown as RecordingContextInitializer;

    mockVerifyScriptInjection.mockResolvedValue({
      loaded: false,
      ready: false,
      inMainContext: true,
      handlersCount: 0,
      version: null,
      error: 'not loaded',
    });

    const promise = runExternalUrlInjectionTest(page as never, contextInitializer, { testUrl: 'https://example.com' });
    await jest.runAllTimersAsync();
    const result = await promise;

    expect(result.success).toBe(false);
    expect(result.failurePoint).toBe('fetch');

    jest.useRealTimers();
  });

  it('reports network failures during external URL injection', async () => {
    jest.useFakeTimers();
    const page = {
      goto: jest.fn().mockRejectedValue(new Error('net::ERR_CONNECTION_REFUSED')),
    };
    const contextInitializer = {
      getInjectionStats: jest.fn().mockReturnValue({ attempted: 0, successful: 0, failed: 0 }),
    } as unknown as RecordingContextInitializer;

    const promise = runExternalUrlInjectionTest(page as never, contextInitializer, { testUrl: 'https://example.com' });
    await jest.runAllTimersAsync();
    const result = await promise;

    expect(result.success).toBe(false);
    expect(result.failurePoint).toBe('network');

    jest.useRealTimers();
  });

  it('reports failure when script is not ready after injection', async () => {
    jest.useFakeTimers();
    const page = {
      goto: jest.fn().mockResolvedValue(undefined),
    };
    const contextInitializer = {
      getInjectionStats: jest
        .fn()
        .mockReturnValueOnce({ attempted: 0, successful: 0, failed: 0 })
        .mockReturnValueOnce({ attempted: 1, successful: 1, failed: 0 }),
    } as unknown as RecordingContextInitializer;

    mockVerifyScriptInjection.mockResolvedValue({
      loaded: true,
      ready: false,
      inMainContext: true,
      handlersCount: 0,
      version: '1.0.0',
      initError: 'init failed',
    });

    const promise = runExternalUrlInjectionTest(page as never, contextInitializer, { testUrl: 'https://example.com' });
    await jest.runAllTimersAsync();
    const result = await promise;

    expect(result.success).toBe(false);
    expect(result.failurePoint).toBe('script_ready');

    jest.useRealTimers();
  });

  it('reports failure when script is in isolated context', async () => {
    jest.useFakeTimers();
    const page = {
      goto: jest.fn().mockResolvedValue(undefined),
    };
    const contextInitializer = {
      getInjectionStats: jest
        .fn()
        .mockReturnValueOnce({ attempted: 0, successful: 0, failed: 0 })
        .mockReturnValueOnce({ attempted: 1, successful: 1, failed: 0 }),
    } as unknown as RecordingContextInitializer;

    mockVerifyScriptInjection.mockResolvedValue({
      loaded: true,
      ready: true,
      inMainContext: false,
      handlersCount: 1,
      version: '1.0.0',
    });

    const promise = runExternalUrlInjectionTest(page as never, contextInitializer, { testUrl: 'https://example.com' });
    await jest.runAllTimersAsync();
    const result = await promise;

    expect(result.success).toBe(false);
    expect(result.failurePoint).toBe('context_wrong');

    jest.useRealTimers();
  });

  it('reports success when injection and script checks pass', async () => {
    jest.useFakeTimers();
    const page = {
      goto: jest.fn().mockResolvedValue(undefined),
    };
    const contextInitializer = {
      getInjectionStats: jest
        .fn()
        .mockReturnValueOnce({ attempted: 0, successful: 0, failed: 0 })
        .mockReturnValueOnce({ attempted: 1, successful: 1, failed: 0 }),
    } as unknown as RecordingContextInitializer;

    mockVerifyScriptInjection.mockResolvedValue({
      loaded: true,
      ready: true,
      inMainContext: true,
      handlersCount: 5,
      version: '1.0.0',
    });

    const promise = runExternalUrlInjectionTest(page as never, contextInitializer, { testUrl: 'https://example.com' });
    await jest.runAllTimersAsync();
    const result = await promise;

    expect(result.success).toBe(true);
    expect(result.failurePoint).toBeUndefined();

    jest.useRealTimers();
  });

  it('returns early when navigation fails in pipeline test', async () => {
    const page = {
      goto: jest.fn().mockRejectedValue(new Error('Network error')),
      on: jest.fn(),
    };

    const pipelineManager = {
      isRecording: jest.fn().mockReturnValue(false),
      getRecordingId: jest.fn().mockReturnValue(null),
    } as unknown as RecordingPipelineManager;

    const contextInitializer = {
      getInjectionStats: jest.fn().mockReturnValue({ attempted: 0, successful: 0, failed: 0 }),
      resetStats: jest.fn(),
      getRouteHandlerStats: jest.fn().mockReturnValue({ eventsReceived: 0, eventsProcessed: 0, eventsDroppedNoHandler: 0, eventsWithErrors: 0 }),
    } as unknown as RecordingContextInitializer;

    const result = await runRecordingPipelineTest(
      page as never,
      {} as never,
      pipelineManager,
      contextInitializer,
      { captureConsole: false, timeoutMs: 1000 }
    );

    expect(result.success).toBe(false);
    expect(result.failurePoint).toBe('page_load');
  });

  it('returns script_injection when injection attempt fails in pipeline test', async () => {
    jest.useFakeTimers();
    const page = {
      goto: jest.fn().mockResolvedValue(undefined),
      on: jest.fn(),
    };

    const pipelineManager = {
      isRecording: jest.fn().mockReturnValue(false),
      getRecordingId: jest.fn().mockReturnValue(null),
    } as unknown as RecordingPipelineManager;

    const contextInitializer = {
      getInjectionStats: jest
        .fn()
        .mockReturnValueOnce({ attempted: 0, successful: 0, failed: 0 })
        .mockReturnValueOnce({ attempted: 1, successful: 0, failed: 1 }),
      resetStats: jest.fn(),
      getRouteHandlerStats: jest.fn().mockReturnValue({ eventsReceived: 0, eventsProcessed: 0, eventsDroppedNoHandler: 0, eventsWithErrors: 0 }),
    } as unknown as RecordingContextInitializer;

    const promise = runRecordingPipelineTest(
      page as never,
      {} as never,
      pipelineManager,
      contextInitializer,
      { captureConsole: false, timeoutMs: 1000 }
    );
    await jest.runAllTimersAsync();
    const result = await promise;

    expect(result.success).toBe(false);
    expect(result.failurePoint).toBe('script_injection');

    jest.useRealTimers();
  });

  it('returns script_initialization when script not ready in pipeline test', async () => {
    jest.useFakeTimers();
    const page = {
      goto: jest.fn().mockResolvedValue(undefined),
      on: jest.fn(),
    };

    const pipelineManager = {
      isRecording: jest.fn().mockReturnValue(false),
      getRecordingId: jest.fn().mockReturnValue(null),
    } as unknown as RecordingPipelineManager;

    const contextInitializer = {
      getInjectionStats: jest
        .fn()
        .mockReturnValueOnce({ attempted: 0, successful: 0, failed: 0 })
        .mockReturnValueOnce({ attempted: 1, successful: 1, failed: 0 }),
      resetStats: jest.fn(),
      getRouteHandlerStats: jest.fn().mockReturnValue({ eventsReceived: 0, eventsProcessed: 0, eventsDroppedNoHandler: 0, eventsWithErrors: 0 }),
      setupPageEventRoute: jest.fn().mockResolvedValue(undefined),
    } as unknown as RecordingContextInitializer;

    mockVerifyScriptInjection.mockResolvedValue({
      loaded: true,
      ready: false,
      inMainContext: true,
      handlersCount: 1,
      version: '1.0.0',
      initError: 'init failed',
    });

    const promise = runRecordingPipelineTest(
      page as never,
      {} as never,
      pipelineManager,
      contextInitializer,
      { captureConsole: false, timeoutMs: 1000 }
    );
    await jest.runAllTimersAsync();
    const result = await promise;

    expect(result.success).toBe(false);
    expect(result.failurePoint).toBe('script_initialization');

    jest.useRealTimers();
  });

  it('returns handler_process when recording fails to start', async () => {
    jest.useFakeTimers();
    const page = {
      goto: jest.fn().mockResolvedValue(undefined),
    };

    const pipelineManager = {
      isRecording: jest.fn().mockReturnValue(true),
      getRecordingId: jest.fn().mockReturnValue('recording-1'),
      stopRecording: jest.fn().mockResolvedValue(undefined),
      startRecording: jest.fn().mockRejectedValue(new Error('start failed')),
    } as unknown as RecordingPipelineManager;

    const contextInitializer = {
      getInjectionStats: jest
        .fn()
        .mockReturnValueOnce({ attempted: 0, successful: 0, failed: 0 })
        .mockReturnValueOnce({ attempted: 1, successful: 1, failed: 0 }),
      resetStats: jest.fn(),
      getRouteHandlerStats: jest.fn().mockReturnValue({ eventsReceived: 0, eventsProcessed: 0, eventsDroppedNoHandler: 0, eventsWithErrors: 0 }),
      setupPageEventRoute: jest.fn().mockResolvedValue(undefined),
    } as unknown as RecordingContextInitializer;

    mockVerifyScriptInjection.mockResolvedValue({
      loaded: true,
      ready: true,
      inMainContext: true,
      handlersCount: 1,
      version: '1.0.0',
    });

    const promise = runRecordingPipelineTest(
      page as never,
      {} as never,
      pipelineManager,
      contextInitializer,
      { captureConsole: false, timeoutMs: 1000 }
    );
    await jest.runAllTimersAsync();
    const result = await promise;

    expect(result.success).toBe(false);
    expect(result.failurePoint).toBe('handler_process');

    jest.useRealTimers();
  });
});
