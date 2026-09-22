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
    off: jest.fn((event: string, handler: FrameHandler) => {
      if (handlers.get(event) === handler) handlers.delete(event);
      return session;
    }),
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
  it('rejects unsupported physical-pixel capture before acquiring resources [REQ:BAS-RH-J23]', async () => {
    const { session: cdpSession } = createCdpSession();
    const newCDPSession = jest.fn().mockResolvedValue(cdpSession);
    const page = {
      context: () => ({ newCDPSession }),
      viewportSize: () => ({ width: 320, height: 240 }),
      isClosed: () => false,
    } as unknown as Page;
    let handle: Awaited<ReturnType<CdpScreencastStrategy['start']>> | undefined;
    try {
      await expect(new CdpScreencastStrategy().start(() => page, {
        sessionId: 'device-scale', quality: 65, targetFps: 30, scale: 'device', includePerfHeaders: false,
      }, { isReady: () => false, getWebSocket: () => null }, {
        onFrameSent: jest.fn(), onFrameSkipped: jest.fn(),
      }).then((started) => { handle = started; return started; })).rejects.toThrow(/device.*scale/i);
      expect(newCDPSession).not.toHaveBeenCalled();
    } finally { await handle?.stop(); }
  });

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

  it('delivers the newest buffered frame when transport reconnects', async () => {
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

    try {
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
  
      expect(ws.send).toHaveBeenCalledTimes(1);
      const packet = ws.send.mock.calls[0][0] as Buffer;
      const headerLength = packet.readUInt32BE(0);
      expect(packet.subarray(4 + headerLength).toString()).toBe('frame-2');
      expect(onFrameSent).toHaveBeenCalledTimes(1);
  
    } finally { await handle.stop(); }
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
      let available = true;
      const pageProvider = jest.fn<() => Page>().mockImplementation(() => {
        if (!available) throw new Error('Session not found: closed-session');
        return page;
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

      available = false;
      await jest.advanceTimersByTimeAsync(25);

      expect(handle.isActive()).toBe(false);
      expect(send).toHaveBeenCalledWith('Page.stopScreencast');
      expect(detach).toHaveBeenCalled();
    } finally {
      jest.useRealTimers();
    }
  });
  it('disposes protocol and polling resources when initial screencast start fails', async () => {
    jest.useFakeTimers();
    try {
      const { session, send, detach } = createCdpSession();
      send.mockImplementation(async (method: string) => {
        if (method === 'Page.startScreencast') throw new Error('start capture failed');
      });
      const page = {
        context: () => ({ newCDPSession: async () => session }),
        viewportSize: () => ({ width: 640, height: 480 }),
      } as unknown as Page;
      await expect(new CdpScreencastStrategy().start(
        () => page,
        { sessionId: 'failed-start', quality: 65, targetFps: 30, scale: 'css', includePerfHeaders: false },
        { isReady: () => false, getWebSocket: () => null },
        { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() },
      )).rejects.toThrow('start capture failed');
      expect(detach).toHaveBeenCalledTimes(1);
      jest.runAllTicks();
      expect(jest.getTimerCount()).toBe(0);
    } finally {
      jest.clearAllTimers();
      jest.useRealTimers();
    }
  });

  it('releases completed frame ACK deadlines before stopping capture', async () => {
    jest.useFakeTimers();
    try {
      const { session } = createCdpSession();
      const page = {
        context: () => ({ newCDPSession: async () => session }),
        viewportSize: () => ({ width: 640, height: 480 }),
      } as unknown as Page;
      const capture = await new CdpScreencastStrategy().start(
        () => page,
        { sessionId: 'ack-owner', quality: 65, targetFps: 30, scale: 'css', includePerfHeaders: false },
        { isReady: () => false, getWebSocket: () => null },
        { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() },
      );
      session.emit('Page.screencastFrame', { data: 'ZmFrZQ==', metadata: {}, sessionId: 1 });
      await jest.advanceTimersByTimeAsync(0);
      expect(jest.getTimerCount()).toBe(1); // Only the live page-change probe.
      await capture.stop();
      jest.runAllTicks();
      expect(jest.getTimerCount()).toBe(0);
    } finally {
      jest.clearAllTimers();
      jest.useRealTimers();
    }
  });

  describe('capture generation ownership', () => {
    const deferred = <T>() => {
      let resolve!: (value: T) => void;
      const promise = new Promise<T>((done) => { resolve = done; });
      return { promise, resolve };
    };
    const options = {
      sessionId: 'generation-owner', quality: 65, targetFps: 30,
      scale: 'css' as const, includePerfHeaders: false,
      cdp: { pageCheckIntervalMs: 25 },
    };
    const reporter = () => ({ onFrameSent: jest.fn(), onFrameSkipped: jest.fn() });
    const emit = (cdp: ReturnType<typeof createCdpSession>, id: number, data: string) => {
      cdp.session.emit('Page.screencastFrame', { sessionId: id, metadata: {}, data: Buffer.from(data).toString('base64') });
    };
    const pageFor = (acquire: () => Promise<CDPSession>, resize = async () => {}) => ({
      context: () => ({ newCDPSession: acquire }),
      viewportSize: () => ({ width: 640, height: 480 }),
      setViewportSize: resize,
      isClosed: () => false,
    } as unknown as Page);

    it.each(['viewport', 'protocol'] as const)('cannot restart after stop during %s acquisition', async (stage) => {
      const first = createCdpSession();
      const late = createCdpSession();
      const viewport = deferred<void>();
      const acquired = deferred<CDPSession>();
      const requested = deferred<void>();
      let count = 0;
      const page = pageFor(async () => {
        if (++count === 1) return first.session;
        requested.resolve();
        return acquired.promise;
      }, () => stage === 'viewport' ? viewport.promise : Promise.resolve());
      const capture = await new CdpScreencastStrategy().start(
        () => page, options, { isReady: () => false, getWebSocket: () => null }, reporter(),
      );
      const resize = capture.updateViewport!(900, 700).then(() => 'success', () => 'cancelled');
      if (stage === 'protocol') await requested.promise;
      const stopped = capture.stop();
      viewport.resolve();
      acquired.resolve(late.session);
      await stopped;
      expect(await resize).toBe('cancelled');
      expect(late.send).not.toHaveBeenCalledWith('Page.startScreencast', expect.anything());
      expect(late.detach).toHaveBeenCalledTimes(stage === 'protocol' ? 1 : 0);
      expect(capture.isActive()).toBe(false);
    });

    it('publishes only current-page pixels after switching a tab', async () => {
      jest.useFakeTimers();
      const first = createCdpSession();
      const second = createCdpSession();
      let current = pageFor(async () => first.session);
      let ready = false;
      const socket = { readyState: 1, send: jest.fn() };
      const capture = await new CdpScreencastStrategy().start(
        () => current, options, { isReady: () => ready, getWebSocket: () => socket }, reporter(),
      );
      try {
        emit(first, 1, 'old-buffer');
        await jest.advanceTimersByTimeAsync(0);
        current = pageFor(async () => second.session);
        await jest.advanceTimersByTimeAsync(25);
        ready = true;
        emit(second, 2, 'new');
        emit(first, 3, 'old-late');
        await jest.advanceTimersByTimeAsync(0);
        expect(socket.send.mock.calls.map(([frame]) => (frame as Buffer).subarray(8).toString())).toEqual(['new']);
        expect(second.send).not.toHaveBeenCalledWith('Page.screencastFrameAck', { sessionId: 3 });
      } finally {
        await capture.stop();
        jest.useRealTimers();
      }
    });

    it('delivers the buffered current frame when a stable page reconnects without another frame', async () => {
      jest.useFakeTimers();
      const protocol = createCdpSession();
      const page = pageFor(async () => protocol.session);
      let ready = false;
      const socket = { readyState: 1, send: jest.fn() };
      const capture = await new CdpScreencastStrategy().start(
        () => page, options, { isReady: () => ready, getWebSocket: () => socket }, reporter(),
      );
      try {
        emit(protocol, 1, 'stable');
        await jest.advanceTimersByTimeAsync(0);
        ready = true;
        await jest.advanceTimersByTimeAsync(50);
        expect(socket.send.mock.calls.map(([frame]) => (frame as Buffer).subarray(8).toString())).toEqual(['stable']);
      } finally {
        await capture.stop();
        jest.useRealTimers();
      }
    });

    it('acknowledges a frame even when transport send fails', async () => {
      const protocol = createCdpSession();
      const page = pageFor(async () => protocol.session);
      const capture = await new CdpScreencastStrategy().start(
        () => page, options,
        { isReady: () => true, getWebSocket: () => ({ readyState: 1, send: () => { throw new Error('socket closed'); } }) },
        reporter(),
      );
      try {
        emit(protocol, 1, 'paint');
        await new Promise((resolve) => setImmediate(resolve));
        expect(protocol.send).toHaveBeenCalledWith('Page.screencastFrameAck', { sessionId: 1 });
      } finally { await capture.stop(); }
    });
  });

});


describe('effective CDP controls [REQ:BAS-RH-J23]', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => { jest.useRealTimers(); jest.restoreAllMocks(); });

  async function capture(targetFps = 30, configure?: (protocol: ReturnType<typeof createCdpSession>, index: number) => void) {
    const protocols: ReturnType<typeof createCdpSession>[] = [];
    const page = {
      viewportSize: () => ({ width: 640, height: 480 }), isClosed: () => false,
      setViewportSize: jest.fn().mockResolvedValue(undefined),
      context: () => ({ newCDPSession: async () => {
        const protocol = createCdpSession(); configure?.(protocol, protocols.length);
        protocols.push(protocol); return protocol.session;
      } }),
    } as unknown as Page;
    const socket = { readyState: 1, send: jest.fn() };
    const handle = await new CdpScreencastStrategy().start(() => page,
      { sessionId: 'effective-controls', quality: 65, targetFps, scale: 'css', includePerfHeaders: false, cdp: { pageCheckIntervalMs: 100000 } },
      { isReady: () => true, getWebSocket: () => socket },
      { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() });
    const emit = (data: string, number = 1) => protocols.at(-1)!.session.emit('Page.screencastFrame', {
      data: Buffer.from(data).toString('base64'), metadata: {}, sessionId: number,
    });
    return { handle, protocols, socket, emit };
  }

  it('limits delivery and flushes the newest stable frame without another paint', async () => {
    const f = await capture(2);
    try {
      for (let i = 0; i < 10; i++) { f.emit(`frame-${i}`, i); await jest.advanceTimersByTimeAsync(20); }
      expect(f.socket.send).toHaveBeenCalledTimes(1);
      await jest.advanceTimersByTimeAsync(299);
      expect(f.socket.send).toHaveBeenCalledTimes(1);
      await jest.advanceTimersByTimeAsync(1);
      expect(f.socket.send).toHaveBeenCalledTimes(2);
      expect((f.socket.send.mock.calls[1][0] as Buffer).subarray(8).toString()).toBe('frame-9');
      expect(f.protocols[0].send.mock.calls.filter(([method]) => method === 'Page.screencastFrameAck')).toHaveLength(10);
    } finally { await f.handle.stop(); }
    jest.runAllTicks();
    expect(jest.getTimerCount()).toBe(0);
  });

  it('cancels a queued delivery deadline at stop', async () => {
    const f = await capture(1);
    f.emit('first'); f.emit('pending');
    await jest.advanceTimersByTimeAsync(0);
    await f.handle.stop();
    await jest.advanceTimersByTimeAsync(1000);
    expect(f.socket.send).toHaveBeenCalledTimes(1);
    jest.runAllTicks(); expect(jest.getTimerCount()).toBe(0);
  });

  it('applies FPS and header changes to subsequent delivery', async () => {
    const f = await capture(1);
    try {
      f.emit('first'); await jest.advanceTimersByTimeAsync(50); f.emit('latest');
      f.handle.updateTargetFps?.(10); f.handle.updatePerfMode?.(true);
      await jest.advanceTimersByTimeAsync(49);
      expect(f.socket.send).toHaveBeenCalledTimes(1);
      await jest.advanceTimersByTimeAsync(1);
      expect(f.socket.send).toHaveBeenCalledTimes(2);
      const packet = f.socket.send.mock.calls[1][0] as Buffer;
      const length = packet.readUInt32BE(0);
      expect(JSON.parse(packet.subarray(4, 4 + length).toString())).toMatchObject({ frame_bytes: 6 });
      expect(packet.subarray(4 + length).toString()).toBe('latest');
    } finally { await f.handle.stop(); }
  });

  it('applies an FPS change independently of a failed buffered transport send', async () => {
    const f = await capture(1);
    try {
      f.emit('first'); await jest.advanceTimersByTimeAsync(500); f.emit('pending');
      f.socket.send.mockImplementationOnce(() => { throw new Error('viewer disconnected'); });
      expect(() => f.handle.updateTargetFps?.(60)).not.toThrow();
      await jest.advanceTimersByTimeAsync(0);
      expect(f.handle.isActive()).toBe(true);
    } finally { await f.handle.stop(); }
  });

  it('does not stop a newer resize when a pending quality change is superseded', async () => {
    let release!: () => void;
    const applied = new Promise<void>((resolve) => { release = resolve; });
    const f = await capture(30, (protocol, index) => {
      if (index === 1) protocol.send.mockImplementation((method) => method === 'Page.startScreencast' ? applied : Promise.resolve());
    });
    try {
      const quality = Promise.resolve(f.handle.updateQuality?.(20));
      void quality.catch(() => {});
      await jest.advanceTimersByTimeAsync(0);
      const resize = f.handle.updateViewport!(900, 700);
      void resize.catch(() => {});
      await jest.advanceTimersByTimeAsync(0);
      release();
      const results = await Promise.allSettled([quality, resize]);
      expect(results[0].status).toBe('rejected');
      expect(results[1].status).toBe('fulfilled');
      expect(f.handle.isActive()).toBe(true);
      expect(f.protocols.at(-1)!.send).toHaveBeenCalledWith('Page.startScreencast', expect.objectContaining({ maxWidth: 900 }));
    } finally { release(); await f.handle.stop(); }
  });
});
