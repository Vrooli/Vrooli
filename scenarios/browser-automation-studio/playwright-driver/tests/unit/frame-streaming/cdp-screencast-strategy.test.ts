import type { Page, CDPSession } from 'rebrowser-playwright';
import type { FrameStatsReporter, WebSocketProvider } from '../../../src/frame-streaming/strategies';
import { CdpScreencastStrategy } from '../../../src/frame-streaming/strategies';

type FrameHandler = (event: {
  data: string;
  metadata: Record<string, number>;
  sessionId: number;
}) => void;

const createCdpSession = (): {
  session: jest.Mocked<CDPSession> & { emit: (event: string, payload: unknown) => void };
  send: jest.Mock;
  detach: jest.Mock;
} => {
  const handlers = new Map<string, FrameHandler>();
  const send = jest.fn().mockResolvedValue(undefined);
  const detach = jest.fn().mockResolvedValue(undefined);
  const session: jest.Mocked<CDPSession> & { emit: (event: string, payload: unknown) => void } = {
    send,
    on: jest.fn((event: string, handler: FrameHandler) => {
      handlers.set(event, handler);
      return session;
    }),
    detach,
    emit: (event: string, payload: unknown) => {
      const handler = handlers.get(event);
      if (handler) {
        handler(payload as { data: string; metadata: Record<string, number>; sessionId: number });
      }
    },
  } as unknown as jest.Mocked<CDPSession> & { emit: (event: string, payload: unknown) => void };

  return { session, send, detach };
};

describe('CdpScreencastStrategy', () => {
  it('detects Chromium support via browser type', async () => {
    const strategy = new CdpScreencastStrategy();
    const browserTypeName = (): string => 'chromium';
    const browserType = (): { name: () => string } => ({ name: browserTypeName });
    const browser = (): { browserType: () => { name: () => string } } => ({ browserType });
    const context = (): { browser: () => { browserType: () => { name: () => string } } } => ({ browser });
    const page = {
      context,
    } as unknown as Page;

    await expect(strategy.isSupported(page)).resolves.toBe(true);
  });

  it('buffers frames while WebSocket is not ready and flushes on reconnect', async () => {
    const strategy = new CdpScreencastStrategy();
    const { session: cdpSession, send } = createCdpSession();

    const newCDPSession = jest.fn().mockResolvedValue(cdpSession);
    const viewportSize = jest.fn().mockReturnValue({ width: 1280, height: 720 });
    const setViewportSize = jest.fn().mockResolvedValue(undefined);
    const isClosed = jest.fn().mockReturnValue(false);
    const page = {
      context: () => ({ newCDPSession }),
      viewportSize,
      setViewportSize,
      isClosed,
    } as unknown as Page;

    let isReady = false;
    const ws = { readyState: 1, send: jest.fn() };
    const wsProvider: WebSocketProvider = {
      isReady: () => isReady,
      getWebSocket: () => ws,
    };

    const onFrameSent = jest.fn();
    const onFrameSkipped = jest.fn();
    const statsReporter: FrameStatsReporter = {
      onFrameSent,
      onFrameSkipped,
    };

    const handle = await strategy.start(
      () => page,
      {
        sessionId: 'session-1',
        quality: 80,
        targetFps: 30,
        scale: 'css',
        includePerfHeaders: true,
        cdp: { pageCheckIntervalMs: 100000 },
      },
      wsProvider,
      statsReporter
    );

    cdpSession.emit('Page.screencastFrame', {
      data: Buffer.from('frame-1').toString('base64'),
      metadata: {},
      sessionId: 1,
    });

    expect(onFrameSkipped).toHaveBeenCalledWith('ws_not_ready');
    expect(send).toHaveBeenCalledWith('Page.screencastFrameAck', { sessionId: 1 });

    isReady = true;
    cdpSession.emit('Page.screencastFrame', {
      data: Buffer.from('frame-2').toString('base64'),
      metadata: {},
      sessionId: 2,
    });

    expect(ws.send).toHaveBeenCalledTimes(2);
    expect(onFrameSent).toHaveBeenCalledTimes(2);

    await handle.stop();
  });

  it('skips viewport updates below threshold', async () => {
    const strategy = new CdpScreencastStrategy();
    const { session: cdpSession } = createCdpSession();

    const newCDPSession = jest.fn().mockResolvedValue(cdpSession);
    const viewportSize = jest.fn().mockReturnValue({ width: 1280, height: 720 });
    const setViewportSize = jest.fn().mockResolvedValue(undefined);
    const isClosed = jest.fn().mockReturnValue(false);
    const page = {
      context: () => ({ newCDPSession }),
      viewportSize,
      setViewportSize,
      isClosed,
    } as unknown as Page;

    const wsProvider: WebSocketProvider = {
      isReady: () => true,
      getWebSocket: () => ({ readyState: 1, send: jest.fn() }),
    };

    const onFrameSent = jest.fn();
    const onFrameSkipped = jest.fn();
    const statsReporter: FrameStatsReporter = {
      onFrameSent,
      onFrameSkipped,
    };

    const handle = await strategy.start(
      () => page,
      {
        sessionId: 'session-1',
        quality: 80,
        targetFps: 30,
        scale: 'css',
        includePerfHeaders: false,
        cdp: { pageCheckIntervalMs: 100000 },
      },
      wsProvider,
      statsReporter
    );

    await handle.updateViewport?.(1290, 735);

    expect(setViewportSize).not.toHaveBeenCalled();
    expect(handle.isViewportUpdatePending?.()).toBe(false);

    await handle.stop();
  });

  it('restarts screencast when viewport updates exceed threshold', async () => {
    const strategy = new CdpScreencastStrategy();
    const firstSession = createCdpSession();
    const secondSession = createCdpSession();

    const newCDPSession = jest.fn()
      .mockResolvedValueOnce(firstSession.session)
      .mockResolvedValueOnce(secondSession.session);
    const context = {
      newCDPSession,
    };

    const viewportSize = jest.fn().mockReturnValue({ width: 1280, height: 720 });
    const setViewportSize = jest.fn().mockResolvedValue(undefined);
    const isClosed = jest.fn().mockReturnValue(false);
    const page = {
      context: () => context,
      viewportSize,
      setViewportSize,
      isClosed,
    } as unknown as Page;

    const wsProvider: WebSocketProvider = {
      isReady: () => true,
      getWebSocket: () => ({ readyState: 1, send: jest.fn() }),
    };

    const onFrameSent = jest.fn();
    const onFrameSkipped = jest.fn();
    const statsReporter: FrameStatsReporter = {
      onFrameSent,
      onFrameSkipped,
    };

    const handle = await strategy.start(
      () => page,
      {
        sessionId: 'session-2',
        quality: 80,
        targetFps: 30,
        scale: 'css',
        includePerfHeaders: false,
        cdp: { pageCheckIntervalMs: 100000 },
      },
      wsProvider,
      statsReporter
    );

    await handle.updateViewport?.(1400, 900);

    expect(setViewportSize).toHaveBeenCalledWith({ width: 1400, height: 900 });
    expect(firstSession.send).toHaveBeenCalledWith('Page.stopScreencast');
    expect(secondSession.send).toHaveBeenCalledWith('Page.startScreencast', expect.any(Object));

    await handle.stop();
  });

  it('stops cleanly when a queued page probe outlives its session', async () => {
    jest.useFakeTimers();
    try {
      const strategy = new CdpScreencastStrategy();
      const { session: cdpSession, send, detach } = createCdpSession();
      const page = {
        context: () => ({ newCDPSession: jest.fn().mockResolvedValue(cdpSession) }),
        viewportSize: () => ({ width: 1280, height: 720 }),
      } as unknown as Page;
      const pageProvider = jest.fn<() => Page>()
        .mockReturnValueOnce(page)
        .mockImplementation(() => {
          throw new Error('Session not found: closed-session');
        });
      const wsProvider: WebSocketProvider = {
        isReady: () => true,
        getWebSocket: () => ({ readyState: 1, send: jest.fn() }),
      };
      const statsReporter: FrameStatsReporter = {
        onFrameSent: jest.fn(),
        onFrameSkipped: jest.fn(),
      };

      const handle = await strategy.start(
        pageProvider,
        {
          sessionId: 'closed-session',
          quality: 80,
          targetFps: 30,
          scale: 'css',
          includePerfHeaders: false,
          cdp: { pageCheckIntervalMs: 25 },
        },
        wsProvider,
        statsReporter
      );

      await jest.advanceTimersByTimeAsync(25);

      expect(handle.isActive()).toBe(false);
      expect(send).toHaveBeenCalledWith('Page.stopScreencast');
      expect(detach).toHaveBeenCalled();
    } finally {
      jest.useRealTimers();
    }
  });
});
