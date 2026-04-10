import type { Page } from 'rebrowser-playwright';
import type { FrameStatsReporter, WebSocketProvider, StreamingStrategyConfig } from '../../../src/frame-streaming/strategies';
import { PollingStrategy } from '../../../src/frame-streaming/strategies';

const createConfig = (overrides?: Partial<StreamingStrategyConfig>): StreamingStrategyConfig => ({
  sessionId: 'session-1',
  quality: 80,
  targetFps: 10,
  scale: 'css' as const,
  includePerfHeaders: false,
  ...overrides,
});

describe('PollingStrategy', () => {
  it('skips frames when WebSocket is not ready', async () => {
    const strategy = new PollingStrategy();

    const screenshot = jest.fn().mockResolvedValue(Buffer.from('frame'));
    const page = {
      viewportSize: jest.fn().mockReturnValue(null),
      screenshot,
    } as unknown as Page;

    const ws = { readyState: 1, send: jest.fn() };
    const wsProvider: WebSocketProvider = {
      isReady: () => false,
      getWebSocket: () => ws,
    };

    let resolveSkip: (() => void) | null = null;
    const skipPromise = new Promise<void>((resolve) => {
      resolveSkip = resolve;
    });

    const onFrameSkipped = jest.fn(() => {
      resolveSkip?.();
    });
    const statsReporter: FrameStatsReporter = {
      onFrameSent: jest.fn(),
      onFrameSkipped,
    };

    const handle = await strategy.start(() => page, createConfig(), wsProvider, statsReporter);

    await skipPromise;
    await handle.stop();

    expect(onFrameSkipped).toHaveBeenCalledWith('ws_not_ready');
    expect(ws.send).not.toHaveBeenCalled();
  });

  it('sends frames and skips unchanged buffers', async () => {
    jest.useFakeTimers();

    const strategy = new PollingStrategy();
    const buffers = [Buffer.from('same-frame'), Buffer.from('same-frame')];

    const screenshot = jest.fn().mockImplementation(() => buffers.shift() ?? Buffer.from('fallback'));
    const page = {
      viewportSize: jest.fn().mockReturnValue(null),
      screenshot,
    } as unknown as Page;

    const ws = { readyState: 1, send: jest.fn() };
    const wsProvider: WebSocketProvider = {
      isReady: () => true,
      getWebSocket: () => ws,
    };

    let resolveSent: (() => void) | null = null;
    const sentPromise = new Promise<void>((resolve) => {
      resolveSent = resolve;
    });

    let resolveUnchanged: (() => void) | null = null;
    const unchangedPromise = new Promise<void>((resolve) => {
      resolveUnchanged = resolve;
    });

    const onFrameSent = jest.fn(() => {
      resolveSent?.();
    });
    const onFrameSkipped = jest.fn((reason) => {
      if (reason === 'unchanged') {
        resolveUnchanged?.();
      }
    });
    const statsReporter: FrameStatsReporter = {
      onFrameSent,
      onFrameSkipped,
    };

    const handle = await strategy.start(() => page, createConfig(), wsProvider, statsReporter);

    await sentPromise;
    handle.updateQuality?.(150);
    handle.updateTargetFps?.(120);

    await jest.advanceTimersByTimeAsync(150);
    await unchangedPromise;

    await handle.stop();

    const screenshotCalls = screenshot.mock.calls as Array<[Record<string, unknown>]>;
    expect(screenshotCalls[0]?.[0]?.quality).toBe(80);
    expect(screenshotCalls[1]?.[0]?.quality).toBe(100);
    expect(onFrameSkipped).toHaveBeenCalledWith('unchanged');

    jest.useRealTimers();
  });
});
