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
});
