import { runExternalUrlInjectionTest } from '../../../src/recording/testing/external-url-injection-test';
import type { RecordingContextInitializer } from '../../../src/recording/io/context-initializer';

const mockVerifyScriptInjection = jest.fn();

jest.mock('../../../src/recording/validation/verification', () => ({
  verifyScriptInjection: (...args: unknown[]) => mockVerifyScriptInjection(...args),
  waitForScriptReady: jest.fn(),
}));

describe('external URL injection test', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('checks the initialized page rather than requiring a new injection attempt counter', async () => {
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

    expect(result.success).toBe(true);
    expect(result.failurePoint).toBeUndefined();

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

  it('reports failure when the recording script is not loaded', async () => {
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
    expect(result.failurePoint).toBe('script_load');

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

});
