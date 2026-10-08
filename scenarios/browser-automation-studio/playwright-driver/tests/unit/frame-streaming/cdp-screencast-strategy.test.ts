import type { Page, CDPSession } from 'rebrowser-playwright';
import type {
  FrameStatsReporter,
  WebSocketProvider,
} from '../../../src/frame-streaming/strategies';
import { CdpScreencastStrategy } from '../../../src/frame-streaming/strategies';
import { MAX_QUEUED_FRAME_BYTES } from '../../../src/frame-streaming/types';

type SourceIdentity = {
  stream_kind: 'recording' | 'execution';
  session_id: string;
  execution_id: string;
  lease_id: string;
  page_id: string;
};

const sourceForSession = (sessionId: string): ((page: Page) => SourceIdentity) => {
  const pages = new WeakMap<Page, string>();
  let count = 0;
  return (page: Page): SourceIdentity => {
    if (!pages.has(page)) pages.set(page, `page-${++count}`);
    const pageId = pages.get(page);
    if (!pageId) throw new Error('Page identity was not assigned');
    return {
      stream_kind: 'execution',
      session_id: sessionId,
      execution_id: 'execution-a',
      lease_id: 'lease-a',
      page_id: pageId,
    };
  };
};
const imageBytes = (packet: Buffer): Buffer => packet.subarray(4 + packet.readUInt32BE(0));

const required = <T>(value: T | undefined, label: string): T => {
  if (value === undefined) throw new Error(`${label} was not available`);
  return value;
};

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

type PageFixture = {
  viewport?: { width: number; height: number };
  isClosed?: () => boolean;
  setViewportSize?: jest.Mock;
};

const createCdpPage = (
  acquire: CDPSession | (() => CDPSession | Promise<CDPSession>),
  options: PageFixture = {}
): Page => {
  const newCDPSession = (): Promise<CDPSession> =>
    typeof acquire === 'function' ? Promise.resolve(acquire()) : Promise.resolve(acquire);
  const defaultIsClosed = (): boolean => false;
  return {
    context: () => ({ newCDPSession }),
    viewportSize: () => options.viewport ?? { width: 640, height: 480 },
    setViewportSize: options.setViewportSize ?? jest.fn().mockResolvedValue(undefined),
    isClosed: options.isClosed ?? defaultIsClosed,
  } as unknown as Page;
};

const createViewportFixture = (
  cdpSession: CDPSession,
  newCDPSession: jest.Mock = jest.fn().mockResolvedValue(cdpSession)
): { page: Page; viewportSize: jest.Mock; setViewportSize: jest.Mock } => {
  const viewportSize = jest.fn().mockReturnValue({ width: 1280, height: 720 });
  const setViewportSize = jest.fn().mockResolvedValue(undefined);
  const page = createCdpPage(newCDPSession, { setViewportSize });
  page.viewportSize = viewportSize;
  return { page, viewportSize, setViewportSize };
};

describe('CdpScreencastStrategy', () => {
  it('keeps target cadence when timer delivery is repeatedly rounded up', async () => {
    jest.useFakeTimers();
    const cdp = createCdpSession();
    const page = createCdpPage(cdp.session);
    const socket = { readyState: 1, send: jest.fn() };
    const handle = await new CdpScreencastStrategy().start(
      () => page,
      {
        sessionId: 'cadence',
        sourceForPage: sourceForSession('cadence'),
        quality: 65,
        targetFps: 30,
        scale: 'css',
        includePerfHeaders: false,
      },
      { isReady: () => true, getWebSocket: () => socket },
      { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() }
    );

    try {
      for (let frame = 0; frame < 300; frame++) {
        cdp.session.emit('Page.screencastFrame', {
          sessionId: frame,
          metadata: { timestamp: frame / 30 },
          data: Buffer.from(`frame-${frame}`).toString('base64'),
        });
        await jest.advanceTimersByTimeAsync([32, 33, 35][frame % 3]);
      }

      expect(socket.send.mock.calls.length).toBeGreaterThanOrEqual(299);
    } finally {
      await handle.stop();
      jest.useRealTimers();
    }
  });

  it.each([false, true])(
    'includes immutable source identity independent of performance mode (%s) [REQ:BAS-RH-J22]',
    async (includePerfHeaders) => {
      const cdp = createCdpSession();
      const page = createCdpPage(cdp.session);
      const source = {
        session_id: 'source-owner',
        execution_id: 'execution-a',
        lease_id: 'lease-a',
        page_id: 'page-a',
      };
      const socket = { readyState: 1, send: jest.fn() };
      const config = {
        sessionId: 'source-owner',
        quality: 65,
        targetFps: 30,
        scale: 'css' as const,
        includePerfHeaders,
        sourceForPage: (_page: Page): typeof source => source,
      };
      const handle = await new CdpScreencastStrategy().start(
        () => page,
        config,
        { isReady: () => true, getWebSocket: () => socket },
        { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() }
      );
      try {
        cdp.session.emit('Page.screencastFrame', {
          sessionId: 1,
          metadata: {},
          data: Buffer.from('owned-jpeg').toString('base64'),
        });
        const sentCalls = socket.send.mock.calls as unknown as Array<[Buffer]>;
        const packet = required(sentCalls[0]?.[0], 'first frame packet');
        const length = packet.readUInt32BE(0);
        expect(length).toBeGreaterThan(0);
        expect(length).toBeLessThan(packet.length - 4);
        const header = JSON.parse(packet.subarray(4, 4 + length).toString()) as {
          version: number;
          source: typeof source;
          captured_at: string;
        };
        expect(header).toMatchObject({
          version: 1,
          source,
        });
        expect(typeof header.captured_at).toBe('string');
        expect(packet.subarray(4 + length).toString()).toBe('owned-jpeg');
      } finally {
        await handle.stop();
      }
    }
  );
  it('discards a buffered frame when its lease changes before reconnection', async () => {
    jest.useFakeTimers();
    const cdp = createCdpSession();
    const page = createCdpPage(cdp.session);
    let source = {
      session_id: 'session-1',
      execution_id: 'execution-a',
      lease_id: 'lease-a',
      page_id: 'page-a',
    };
    let ready = false;
    const socket = { readyState: 1, send: jest.fn() };
    const handle = await new CdpScreencastStrategy().start(
      () => page,
      {
        sessionId: 'session-1',
        quality: 80,
        targetFps: 30,
        scale: 'css',
        includePerfHeaders: false,
        sourceForPage: () => source,
      },
      { isReady: () => ready, getWebSocket: () => socket },
      { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() }
    );
    try {
      cdp.session.emit('Page.screencastFrame', {
        sessionId: 1,
        data: Buffer.from('retired').toString('base64'),
        metadata: {},
      });
      await jest.advanceTimersByTimeAsync(0);
      source = { ...source, lease_id: 'lease-b' };
      ready = true;
      await jest.advanceTimersByTimeAsync(50);
      expect(socket.send).not.toHaveBeenCalled();
      cdp.session.emit('Page.screencastFrame', {
        sessionId: 2,
        data: Buffer.from('current').toString('base64'),
        metadata: {},
      });
      await jest.advanceTimersByTimeAsync(0);
      expect(
        socket.send.mock.calls.map(([packet]) => imageBytes(packet as Buffer).toString())
      ).toEqual(['current']);
    } finally {
      await handle.stop();
      jest.useRealTimers();
    }
  });

  it('does not add a frame when the viewer transport queue is at its byte bound', async () => {
    const cdp = createCdpSession();
    const page = createCdpPage(cdp.session);
    const socket = { readyState: 1, bufferedAmount: MAX_QUEUED_FRAME_BYTES, send: jest.fn() };
    const stats: FrameStatsReporter = { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() };
    const handle = await new CdpScreencastStrategy().start(
      () => page,
      {
        sessionId: 'slow-viewer',
        sourceForPage: sourceForSession('slow-viewer'),
        quality: 65,
        targetFps: 30,
        scale: 'css',
        includePerfHeaders: false,
      },
      { isReady: () => true, getWebSocket: () => socket },
      stats
    );
    try {
      cdp.session.emit('Page.screencastFrame', {
        sessionId: 1,
        metadata: {},
        data: Buffer.from('frame').toString('base64'),
      });
      expect(socket.send).not.toHaveBeenCalled();
      expect(stats.onFrameSkipped).toHaveBeenCalledWith('ws_backpressure');
    } finally {
      await handle.stop();
    }
  });

  it('rejects unsupported physical-pixel capture before acquiring resources [REQ:BAS-RH-J23]', async () => {
    const { session: cdpSession } = createCdpSession();
    const newCDPSession = jest.fn().mockResolvedValue(cdpSession);
    const page = createCdpPage(newCDPSession, { viewport: { width: 320, height: 240 } });
    let handle: Awaited<ReturnType<CdpScreencastStrategy['start']>> | undefined;
    try {
      await expect(
        new CdpScreencastStrategy()
          .start(
            () => page,
            {
              sessionId: 'device-scale',
              sourceForPage: sourceForSession('device-scale'),
              quality: 65,
              targetFps: 30,
              scale: 'device',
              includePerfHeaders: false,
            },
            { isReady: () => false, getWebSocket: () => null },
            {
              onFrameSent: jest.fn(),
              onFrameSkipped: jest.fn(),
            }
          )
          .then((started) => {
            handle = started;
            return started;
          })
      ).rejects.toThrow(/device.*scale/i);
      expect(newCDPSession).not.toHaveBeenCalled();
    } finally {
      await handle?.stop();
    }
  });

  it('detects Chromium support via browser type', async () => {
    const strategy = new CdpScreencastStrategy();
    const browserTypeName = (): string => 'chromium';
    const browserType = (): { name: () => string } => ({ name: browserTypeName });
    const browser = (): { browserType: () => { name: () => string } } => ({ browserType });
    const context = (): { browser: () => { browserType: () => { name: () => string } } } => ({
      browser,
    });
    const page = {
      context,
    } as unknown as Page;

    await expect(strategy.isSupported(page)).resolves.toBe(true);
  });

  it('delivers the newest buffered frame when transport reconnects', async () => {
    const strategy = new CdpScreencastStrategy();
    const { session: cdpSession, send } = createCdpSession();
    const { page } = createViewportFixture(cdpSession);

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
        sourceForPage: sourceForSession('session-1'),
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
      const sentCalls = ws.send.mock.calls as unknown as Array<[Buffer]>;
      const packet = required(sentCalls[0]?.[0], 'reconnected frame packet');
      const headerLength = packet.readUInt32BE(0);
      expect(packet.subarray(4 + headerLength).toString()).toBe('frame-2');
      expect(onFrameSent).toHaveBeenCalledTimes(1);
    } finally {
      await handle.stop();
    }
  });

  it('skips viewport updates below threshold', async () => {
    const strategy = new CdpScreencastStrategy();
    const { session: cdpSession } = createCdpSession();
    const { page, viewportSize, setViewportSize } = createViewportFixture(cdpSession);

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
        sourceForPage: sourceForSession('session-1'),
        quality: 80,
        targetFps: 30,
        scale: 'css',
        includePerfHeaders: false,
        cdp: { pageCheckIntervalMs: 100000 },
      },
      wsProvider,
      statsReporter
    );

    viewportSize.mockReturnValue({ width: 1290, height: 735 });
    await handle.updateViewport?.(page);

    expect(setViewportSize).not.toHaveBeenCalled();
    expect(cdpSession.send).toHaveBeenCalledWith(
      'Page.startScreencast',
      expect.objectContaining({ maxWidth: 1290, maxHeight: 735 })
    );

    await handle.stop();
  });

  it('restarts screencast from the applied viewport', async () => {
    const strategy = new CdpScreencastStrategy();
    const firstSession = createCdpSession();
    const secondSession = createCdpSession();

    const newCDPSession = jest
      .fn()
      .mockResolvedValueOnce(firstSession.session)
      .mockResolvedValueOnce(secondSession.session);
    const context = {
      newCDPSession,
    };

    const { page, viewportSize, setViewportSize } = createViewportFixture(
      firstSession.session,
      newCDPSession
    );
    page.context = (): typeof context => context;

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
        sourceForPage: sourceForSession('session-2'),
        quality: 80,
        targetFps: 30,
        scale: 'css',
        includePerfHeaders: false,
        cdp: { pageCheckIntervalMs: 100000 },
      },
      wsProvider,
      statsReporter
    );

    viewportSize.mockReturnValue({ width: 1400, height: 900 });
    await handle.updateViewport?.(page);

    expect(setViewportSize).not.toHaveBeenCalled();
    expect(firstSession.send).toHaveBeenCalledWith('Page.stopScreencast');
    expect(secondSession.send).toHaveBeenCalledWith('Page.startScreencast', expect.any(Object));

    await handle.stop();
  });

  it('stops cleanly when a queued page probe outlives its session', async () => {
    jest.useFakeTimers();
    try {
      const strategy = new CdpScreencastStrategy();
      const { session: cdpSession, send, detach } = createCdpSession();
      const page = createCdpPage(cdpSession, { viewport: { width: 1280, height: 720 } });
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
          sourceForPage: sourceForSession('closed-session'),
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
      send.mockImplementation((method: string) =>
        method === 'Page.startScreencast'
          ? Promise.reject(new Error('start capture failed'))
          : Promise.resolve()
      );
      const page = createCdpPage(session);
      await expect(
        new CdpScreencastStrategy().start(
          () => page,
          {
            sessionId: 'failed-start',
            sourceForPage: sourceForSession('failed-start'),
            quality: 65,
            targetFps: 30,
            scale: 'css',
            includePerfHeaders: false,
          },
          { isReady: () => false, getWebSocket: () => null },
          { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() }
        )
      ).rejects.toThrow('start capture failed');
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
      const page = createCdpPage(session);
      const capture = await new CdpScreencastStrategy().start(
        () => page,
        {
          sessionId: 'ack-owner',
          sourceForPage: sourceForSession('ack-owner'),
          quality: 65,
          targetFps: 30,
          scale: 'css',
          includePerfHeaders: false,
        },
        { isReady: () => false, getWebSocket: () => null },
        { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() }
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
    const deferred = <T>(): { promise: Promise<T>; resolve: (value: T) => void } => {
      let resolve!: (value: T) => void;
      const promise = new Promise<T>((done) => {
        resolve = done;
      });
      return { promise, resolve };
    };
    const options = {
      sessionId: 'generation-owner',
      sourceForPage: sourceForSession('generation-owner'),
      quality: 65,
      targetFps: 30,
      scale: 'css' as const,
      includePerfHeaders: false,
      cdp: { pageCheckIntervalMs: 25 },
    };
    const reporter = (): FrameStatsReporter => ({
      onFrameSent: jest.fn(),
      onFrameSkipped: jest.fn(),
    });
    const emit = (cdp: ReturnType<typeof createCdpSession>, id: number, data: string): void => {
      cdp.session.emit('Page.screencastFrame', {
        sessionId: id,
        metadata: {},
        data: Buffer.from(data).toString('base64'),
      });
    };
    const pageFor = (acquire: () => Promise<CDPSession>, resize = jest.fn()): Page =>
      createCdpPage(acquire, { setViewportSize: resize });

    it.each(['queued', 'protocol'] as const)(
      'cannot restart after stop during %s acquisition',
      async (stage) => {
        const first = createCdpSession();
        const late = createCdpSession();
        const acquired = deferred<CDPSession>();
        const requested = deferred<void>();
        let count = 0;
        const page = pageFor(async () => {
          if (++count === 1) return first.session;
          requested.resolve();
          return acquired.promise;
        });
        const capture = await new CdpScreencastStrategy().start(
          () => page,
          options,
          { isReady: () => false, getWebSocket: () => null },
          reporter()
        );
        jest.spyOn(page, 'viewportSize').mockReturnValue({ width: 900, height: 700 });
        const resize = required(capture.updateViewport, 'updateViewport')(page).then(
          () => 'success',
          () => 'cancelled'
        );
        if (stage === 'protocol') await requested.promise;
        const stopped = capture.stop();
        acquired.resolve(late.session);
        await stopped;
        expect(await resize).toBe('cancelled');
        expect(late.send).not.toHaveBeenCalledWith('Page.startScreencast', expect.anything());
        expect(late.detach).toHaveBeenCalledTimes(stage === 'protocol' ? 1 : 0);
        expect(capture.isActive()).toBe(false);
      }
    );

    it('publishes only current-page pixels after switching a tab', async () => {
      jest.useFakeTimers();
      const first = createCdpSession();
      const second = createCdpSession();
      let current = pageFor(() => Promise.resolve(first.session));
      let ready = false;
      const socket = { readyState: 1, send: jest.fn() };
      const capture = await new CdpScreencastStrategy().start(
        () => current,
        options,
        { isReady: () => ready, getWebSocket: () => socket },
        reporter()
      );
      try {
        emit(first, 1, 'old-buffer');
        await jest.advanceTimersByTimeAsync(0);
        current = pageFor(() => Promise.resolve(second.session));
        await jest.advanceTimersByTimeAsync(25);
        ready = true;
        emit(second, 2, 'new');
        emit(first, 3, 'old-late');
        await jest.advanceTimersByTimeAsync(0);
        expect(
          socket.send.mock.calls.map(([frame]) => imageBytes(frame as Buffer).toString())
        ).toEqual(['new']);
        expect(second.send).not.toHaveBeenCalledWith('Page.screencastFrameAck', { sessionId: 3 });
      } finally {
        await capture.stop();
        jest.useRealTimers();
      }
    });

    it('delivers the buffered current frame when a stable page reconnects without another frame', async () => {
      jest.useFakeTimers();
      const protocol = createCdpSession();
      const page = pageFor(() => Promise.resolve(protocol.session));
      let ready = false;
      const socket = { readyState: 1, send: jest.fn() };
      const capture = await new CdpScreencastStrategy().start(
        () => page,
        options,
        { isReady: () => ready, getWebSocket: () => socket },
        reporter()
      );
      try {
        emit(protocol, 1, 'stable');
        await jest.advanceTimersByTimeAsync(0);
        ready = true;
        await jest.advanceTimersByTimeAsync(50);
        expect(
          socket.send.mock.calls.map(([frame]) => imageBytes(frame as Buffer).toString())
        ).toEqual(['stable']);
      } finally {
        await capture.stop();
        jest.useRealTimers();
      }
    });

    it('acknowledges a frame even when transport send fails', async () => {
      const protocol = createCdpSession();
      const page = pageFor(() => Promise.resolve(protocol.session));
      const capture = await new CdpScreencastStrategy().start(
        () => page,
        options,
        {
          isReady: () => true,
          getWebSocket: () => ({
            readyState: 1,
            send: (): void => {
              throw new Error('socket closed');
            },
          }),
        },
        reporter()
      );
      try {
        emit(protocol, 1, 'paint');
        await new Promise((resolve) => setImmediate(resolve));
        expect(protocol.send).toHaveBeenCalledWith('Page.screencastFrameAck', { sessionId: 1 });
      } finally {
        await capture.stop();
      }
    });
  });
});

describe('effective CDP controls [REQ:BAS-RH-J23]', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => {
    jest.useRealTimers();
    jest.restoreAllMocks();
  });

  type CaptureFixture = {
    handle: Awaited<ReturnType<CdpScreencastStrategy['start']>>;
    protocols: ReturnType<typeof createCdpSession>[];
    socket: { readyState: number; send: jest.Mock };
    emit: (data: string, number?: number) => void;
    page: Page;
  };

  async function capture(
    targetFps = 30,
    configure?: (protocol: ReturnType<typeof createCdpSession>, index: number) => void
  ): Promise<CaptureFixture> {
    const protocols: ReturnType<typeof createCdpSession>[] = [];
    const page = createCdpPage(() => {
      const protocol = createCdpSession();
      configure?.(protocol, protocols.length);
      protocols.push(protocol);
      return Promise.resolve(protocol.session);
    });
    const socket = { readyState: 1, send: jest.fn() };
    const handle = await new CdpScreencastStrategy().start(
      () => page,
      {
        sessionId: 'effective-controls',
        sourceForPage: sourceForSession('effective-controls'),
        quality: 65,
        targetFps,
        scale: 'css',
        includePerfHeaders: false,
        cdp: { pageCheckIntervalMs: 100000 },
      },
      { isReady: () => true, getWebSocket: () => socket },
      { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() }
    );
    const emit = (data: string, number = 1): void =>
      required(protocols.at(-1), 'latest CDP protocol').session.emit('Page.screencastFrame', {
        data: Buffer.from(data).toString('base64'),
        metadata: {},
        sessionId: number,
      });
    return { handle, protocols, socket, emit, page };
  }

  it('limits delivery and flushes the newest stable frame without another paint', async () => {
    const f = await capture(2);
    try {
      for (let i = 0; i < 10; i++) {
        f.emit(`frame-${i}`, i);
        await jest.advanceTimersByTimeAsync(20);
      }
      expect(f.socket.send).toHaveBeenCalledTimes(1);
      await jest.advanceTimersByTimeAsync(299);
      expect(f.socket.send).toHaveBeenCalledTimes(1);
      await jest.advanceTimersByTimeAsync(1);
      expect(f.socket.send).toHaveBeenCalledTimes(2);
      const sentCalls = f.socket.send.mock.calls as unknown as Array<[Buffer]>;
      expect(imageBytes(required(sentCalls[1]?.[0], 'second frame packet')).toString()).toBe(
        'frame-9'
      );
      expect(
        f.protocols[0].send.mock.calls.filter(([method]) => method === 'Page.screencastFrameAck')
      ).toHaveLength(10);
    } finally {
      await f.handle.stop();
    }
    jest.runAllTicks();
    expect(jest.getTimerCount()).toBe(0);
  });

  it('cancels a queued delivery deadline at stop', async () => {
    const f = await capture(1);
    f.emit('first');
    f.emit('pending');
    await jest.advanceTimersByTimeAsync(0);
    await f.handle.stop();
    await jest.advanceTimersByTimeAsync(1000);
    expect(f.socket.send).toHaveBeenCalledTimes(1);
    jest.runAllTicks();
    expect(jest.getTimerCount()).toBe(0);
  });

  it('applies FPS and header changes to subsequent delivery', async () => {
    const f = await capture(1);
    try {
      f.emit('first');
      await jest.advanceTimersByTimeAsync(50);
      f.emit('latest');
      f.handle.updateTargetFps?.(10);
      f.handle.updatePerfMode?.(true);
      await jest.advanceTimersByTimeAsync(49);
      expect(f.socket.send).toHaveBeenCalledTimes(1);
      await jest.advanceTimersByTimeAsync(1);
      expect(f.socket.send).toHaveBeenCalledTimes(2);
      const sentCalls = f.socket.send.mock.calls as unknown as Array<[Buffer]>;
      const packet = required(sentCalls[1]?.[0], 'second frame packet');
      const length = packet.readUInt32BE(0);
      expect(JSON.parse(packet.subarray(4, 4 + length).toString())).toMatchObject({
        timing: { frame_bytes: 6 },
      });
      expect(packet.subarray(4 + length).toString()).toBe('latest');
    } finally {
      await f.handle.stop();
    }
  });

  it('applies an FPS change independently of a failed buffered transport send', async () => {
    const f = await capture(1);
    try {
      f.emit('first');
      await jest.advanceTimersByTimeAsync(500);
      f.emit('pending');
      f.socket.send.mockImplementationOnce(() => {
        throw new Error('viewer disconnected');
      });
      expect(() => f.handle.updateTargetFps?.(60)).not.toThrow();
      await jest.advanceTimersByTimeAsync(0);
      expect(f.handle.isActive()).toBe(true);
    } finally {
      await f.handle.stop();
    }
  });

  it('does not stop a newer resize when a pending quality change is superseded', async () => {
    let release!: () => void;
    const applied = new Promise<void>((resolve) => {
      release = resolve;
    });
    const f = await capture(30, (protocol, index) => {
      if (index === 1)
        protocol.send.mockImplementation((method) =>
          method === 'Page.startScreencast' ? applied : Promise.resolve()
        );
    });
    try {
      const quality = Promise.resolve(f.handle.updateQuality?.(20));
      void quality.catch(() => {});
      await jest.advanceTimersByTimeAsync(0);
      jest.spyOn(f.page, 'viewportSize').mockReturnValue({ width: 900, height: 700 });
      const resize = required(f.handle.updateViewport, 'updateViewport')(f.page);
      void resize.catch(() => {});
      await jest.advanceTimersByTimeAsync(0);
      release();
      const results = await Promise.allSettled([quality, resize]);
      expect(results[0].status).toBe('rejected');
      expect(results[1].status).toBe('fulfilled');
      expect(f.handle.isActive()).toBe(true);
      expect(required(f.protocols.at(-1), 'latest CDP protocol').send).toHaveBeenCalledWith(
        'Page.startScreencast',
        expect.objectContaining({ maxWidth: 900 })
      );
    } finally {
      release();
      await f.handle.stop();
    }
  });
});

describe('capture observes applied viewport [REQ:BAS-RH-J05]', () => {
  it.each([1, 10, 200])(
    'refreshes a %spx resize without changing the browser viewport',
    async (delta) => {
      let viewport = { width: 640, height: 480 };
      const protocols: ReturnType<typeof createCdpSession>[] = [];
      const setViewportSize = jest.fn().mockResolvedValue(undefined);
      const page = createCdpPage(() => {
        const p = createCdpSession();
        protocols.push(p);
        return Promise.resolve(p.session);
      }, { setViewportSize });
      page.viewportSize = (): { width: number; height: number } => viewport;
      const handle = await new CdpScreencastStrategy().start(
        () => page,
        {
          sessionId: 'viewport-observer',
          sourceForPage: sourceForSession('viewport-observer'),
          quality: 65,
          targetFps: 30,
          scale: 'css',
          includePerfHeaders: false,
        },
        { isReady: () => false, getWebSocket: () => null },
        { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() }
      );
      try {
        viewport = { width: 640 + delta, height: 480 + delta };
        await required(handle.updateViewport, 'updateViewport')(page);
        expect(setViewportSize).not.toHaveBeenCalled();
        expect(required(protocols.at(-1), 'latest CDP protocol').send).toHaveBeenCalledWith(
          'Page.startScreencast',
          expect.objectContaining({ maxWidth: viewport.width, maxHeight: viewport.height })
        );
      } finally {
        await handle.stop();
      }
    }
  );
  it('does not mutate a page after the active page has changed', async () => {
    const protocol = createCdpSession();
    const setViewportSize = jest.fn().mockResolvedValue(undefined);
    const page = createCdpPage(protocol.session, { setViewportSize });
    let current = page;
    const handle = await new CdpScreencastStrategy().start(
      () => current,
      {
        sessionId: 'viewport-page',
        sourceForPage: sourceForSession('viewport-page'),
        quality: 65,
        targetFps: 30,
        scale: 'css',
        includePerfHeaders: false,
      },
      { isReady: () => false, getWebSocket: () => null },
      { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() }
    );
    try {
      current = { ...page } as Page;
      await expect(required(handle.updateViewport, 'updateViewport')(page)).rejects.toThrow();
      expect(setViewportSize).not.toHaveBeenCalled();
      expect(
        protocol.send.mock.calls.filter(([method]) => method === 'Page.startScreencast')
      ).toHaveLength(1);
    } finally {
      await handle.stop();
    }
  });
  it.each(['attach', 'start'])(
    'restores the original dimensions when pending %s is superseded',
    async (stage) => {
      let viewport = { width: 640, height: 480 };
      let release!: () => void, acquired!: () => void;
      const gate = new Promise<void>((done) => {
        release = done;
      });
      const waiting = new Promise<void>((done) => {
        acquired = done;
      });
      const protocols: ReturnType<typeof createCdpSession>[] = [];
      const page = {
        viewportSize: () => viewport,
        isClosed: () => false,
        setViewportSize: jest.fn().mockResolvedValue(undefined),
        context: () => ({
          newCDPSession: async (): Promise<CDPSession> => {
            const protocol = createCdpSession();
            protocols.push(protocol);
            if (protocols.length === 2) {
              if (stage === 'attach') {
                acquired();
                await gate;
              } else
                protocol.send.mockImplementation(async (method: string) => {
                  if (method === 'Page.startScreencast') {
                    acquired();
                    await gate;
                  }
                });
            }
            return protocol.session;
          },
        }),
      } as unknown as Page;
      const handle = await new CdpScreencastStrategy().start(
        () => page,
        {
          sessionId: 'latest-viewport',
          sourceForPage: sourceForSession('latest-viewport'),
          quality: 65,
          targetFps: 30,
          scale: 'css',
          includePerfHeaders: false,
        },
        { isReady: () => false, getWebSocket: () => null },
        { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() }
      );
      try {
        viewport = { width: 900, height: 700 };
        const first = required(handle.updateViewport, 'updateViewport')(page);
        void first.catch(() => {});
        await waiting;
        viewport = { width: 640, height: 480 };
        const latest = required(handle.updateViewport, 'updateViewport')(page);
        void latest.catch(() => {});
        release();
        const results = await Promise.allSettled([first, latest]);
        expect(results[0].status).toBe('rejected');
        expect(results[1].status).toBe('fulfilled');
        expect(required(protocols.at(-1), 'latest CDP protocol').send).toHaveBeenCalledWith(
          'Page.startScreencast',
          expect.objectContaining({ maxWidth: 640, maxHeight: 480 })
        );
      } finally {
        release();
        await handle.stop();
      }
    }
  );
});
