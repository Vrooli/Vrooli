import { getEventListeners } from 'node:events';
import type { Page } from 'rebrowser-playwright';
import type { FrameStatsReporter, WebSocketProvider, StreamingStrategyConfig } from '../../../src/frame-streaming/strategies';
import { PollingStrategy } from '../../../src/frame-streaming/strategies';
import { MAX_QUEUED_FRAME_BYTES } from '../../../src/frame-streaming/types';
import { createDeferred, createMockPage } from '../../helpers';

const imageBytes = (packet: Buffer): Buffer => packet.subarray(4 + packet.readUInt32BE(0));

const createConfig = (overrides?: Partial<StreamingStrategyConfig>): StreamingStrategyConfig => ({
  sessionId: 'session-1',
  quality: 80,
  targetFps: 10,
  scale: 'css' as const,
  includePerfHeaders: false,
  sourceForPage: () => ({session_id:'session-1',execution_id:'execution-a',lease_id:'lease-a',page_id:'page-a'}),
  ...overrides,
});

type PollingStats = { onFrameSent: jest.MockedFunction<FrameStatsReporter['onFrameSent']>; onFrameSkipped: jest.MockedFunction<FrameStatsReporter['onFrameSkipped']> };
type PollingFixture = { page: Page; screenshot: jest.MockedFunction<() => Promise<Buffer>>; protocol: { send: jest.MockedFunction<() => Promise<{ data: string }>>; detach: jest.MockedFunction<() => Promise<void>> }; socket: { readyState: number; send: jest.MockedFunction<(packet: Buffer) => void> }; stats: PollingStats };

function createPollingFixture(capture: () => Promise<Buffer> = (): Promise<Buffer> => Promise.resolve(Buffer.from('pixels'))): PollingFixture {
  const screenshot = jest.fn(capture) as jest.MockedFunction<() => Promise<Buffer>>;
  const protocol = {
    send: jest.fn(async () => ({ data: (await screenshot()).toString('base64') })) as jest.MockedFunction<() => Promise<{ data: string }>>,
    detach: jest.fn().mockResolvedValue(undefined) as jest.MockedFunction<() => Promise<void>>,
  };
  const page = createMockPage({ viewportSize: jest.fn().mockReturnValue({ width: 640, height: 480 }), screenshot,
    context: jest.fn().mockReturnValue({ newCDPSession: () => Promise.resolve(protocol) }) });
  const socket = { readyState: 1, send: jest.fn() as jest.MockedFunction<(packet: Buffer) => void> };
  const stats: PollingStats = { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() };
  return { page, screenshot, protocol, socket, stats };
}

describe('PollingStrategy', () => {
  it('includes the producing page and lease with performance mode disabled [REQ:BAS-RH-J22]',async()=>{
    const source={session_id:'session-1',execution_id:'execution-a',lease_id:'lease-a',page_id:'page-a'};
    const f = createPollingFixture(() => Promise.resolve(Buffer.from('owned-jpeg')));
    const sent = createDeferred<void>();
    const config={...createConfig(),sourceForPage:(): typeof source => source};
    const handle=await new PollingStrategy().start(()=>f.page,config,
      {isReady:()=>true,getWebSocket:()=>f.socket},{onFrameSent:sent.resolve,onFrameSkipped:jest.fn()});
    try {
      await sent.promise;const packet=f.socket.send.mock.calls[0]?.[0] as Buffer;const length=packet.readUInt32BE(0);
      expect(length).toBeGreaterThan(0);expect(length).toBeLessThan(packet.length-4);
      const metadata = JSON.parse(packet.subarray(4,4+length).toString()) as { version: number; source: typeof source; captured_at: string };
      expect(metadata).toMatchObject({version:1,source});
      expect(metadata.captured_at).toEqual(expect.any(String));
      expect(packet.subarray(4+length).toString()).toBe('owned-jpeg');
    } finally {await handle.stop();}
  });
  it('discards capture completed under a retired lease even when the page is unchanged',async()=>{
    jest.useFakeTimers();
    let source={session_id:'session-1',execution_id:'execution-a',lease_id:'lease-a',page_id:'page-a'};
    const finish = createDeferred<Buffer>();
    const f = createPollingFixture();
    f.screenshot.mockResolvedValue(Buffer.from('new-lease'));
    f.screenshot.mockImplementationOnce(() => finish.promise);
    const handle=await new PollingStrategy().start(()=>f.page,createConfig({sourceForPage:()=>source}),
      {isReady:()=>true,getWebSocket:()=>f.socket},f.stats);
    try {
      source={...source,lease_id:'lease-b'};finish.resolve(Buffer.from('retired-lease'));
      await jest.advanceTimersByTimeAsync(0);expect(f.socket.send).not.toHaveBeenCalled();
      await jest.advanceTimersByTimeAsync(150);
      expect(f.socket.send.mock.calls.map(([packet])=>imageBytes(packet).toString())).toEqual(['new-lease']);
    } finally {await handle.stop();jest.useRealTimers();}
  });

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

  it('skips polling delivery while the viewer transport queue is at its byte bound', async () => {
    jest.useFakeTimers();
    const screenshot = jest.fn().mockResolvedValue(Buffer.from('changing-frame'));
    const page = { viewportSize: () => null, screenshot } as unknown as Page;
    const socket = { readyState: 1, bufferedAmount: MAX_QUEUED_FRAME_BYTES, send: jest.fn() };
    const stats: FrameStatsReporter = { onFrameSent: jest.fn(), onFrameSkipped: jest.fn() };
    const handle = await new PollingStrategy().start(() => page, createConfig(),
      { isReady: () => true, getWebSocket: () => socket }, stats);
    try {
      await jest.advanceTimersByTimeAsync(1);
      expect(socket.send).not.toHaveBeenCalled();
      expect(stats.onFrameSkipped).toHaveBeenCalledWith('ws_backpressure');
    } finally { await handle.stop(); jest.useRealTimers(); }
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
    void handle.updateQuality?.(150);
    void handle.updateTargetFps?.(120);

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


describe('polling capture ownership [REQ:BAS-RH-J22]', () => {
  beforeEach(() => jest.useFakeTimers());
  afterEach(() => {
    jest.restoreAllMocks();
    jest.useRealTimers();
  });

  it('retains only the current sleep listener and releases it at stop', async () => {
    const add = jest.spyOn(AbortSignal.prototype, 'addEventListener');
    const f = createPollingFixture();
    const handle = await new PollingStrategy().start(() => f.page, createConfig(),
      { isReady: () => false, getWebSocket: () => f.socket }, f.stats);
    try {
      await jest.advanceTimersByTimeAsync(800);
      expect(f.stats.onFrameSkipped.mock.calls.length).toBeGreaterThanOrEqual(8);
      const signal = add.mock.contexts[0] as AbortSignal;
      expect(getEventListeners(signal, 'abort').length).toBeLessThanOrEqual(1);
      await handle.stop();
      expect(getEventListeners(signal, 'abort')).toHaveLength(0);
      jest.runAllTicks();
      expect(jest.getTimerCount()).toBe(0);
    } finally { await handle.stop(); }
  });

  it('captures a viewport when the browser does not provide CDP', async () => {
    const f = createPollingFixture();
    f.page.context = ((): ReturnType<Page['context']> => ({ newCDPSession: () => Promise.reject(new Error('CDP unavailable')) })) as Page['context'];
    const handle = await new PollingStrategy().start(() => f.page, createConfig(),
      { isReady: () => true, getWebSocket: () => f.socket }, f.stats);
    try {
      await jest.advanceTimersByTimeAsync(1);
      expect(f.socket.send.mock.calls.map(([packet]) => imageBytes(packet).toString())).toContain('pixels');
    } finally { await handle.stop(); }
  });

  it('leaves no deadline timer after completed capture and stop', async () => {
    const f = createPollingFixture();
    const handle = await new PollingStrategy().start(() => f.page, createConfig(),
      { isReady: () => true, getWebSocket: () => f.socket }, f.stats);
    await jest.advanceTimersByTimeAsync(1);
    expect(f.socket.send).toHaveBeenCalledTimes(1);
    await handle.stop();
    jest.runAllTicks();
    expect(jest.getTimerCount()).toBe(0);
  });

  it('joins in-flight capture for every stop caller and never publishes after stop', async () => {
    const pending = createDeferred<Buffer>();
    const f = createPollingFixture(() => pending.promise);
    const handle = await new PollingStrategy().start(() => f.page, createConfig(),
      { isReady: () => true, getWebSocket: () => f.socket }, f.stats);
    await jest.advanceTimersByTimeAsync(1);
    const firstStop = handle.stop();
    let secondFinished = false;
    const secondStop = handle.stop().then(() => { secondFinished = true; });
    await Promise.resolve();
    const joined = !secondFinished;
    pending.resolve(Buffer.from('late-pixels'));
    await jest.advanceTimersByTimeAsync(300);
    await Promise.all([firstStop, secondStop]);
    expect(joined).toBe(true);
    expect(f.socket.send).not.toHaveBeenCalled();
    expect(f.stats.onFrameSent).not.toHaveBeenCalled();
  });

  it('drops a capture when its page is replaced while the screenshot is pending', async () => {
    const pending = createDeferred<Buffer>();
    const old = createPollingFixture(() => pending.promise);
    const next = createPollingFixture(() => Promise.resolve(Buffer.from('new-page')));
    let currentPage = old.page;
    const handle = await new PollingStrategy().start(() => currentPage, createConfig(),
      { isReady: () => true, getWebSocket: () => old.socket }, old.stats);
    try {
      await jest.advanceTimersByTimeAsync(1);
      currentPage = next.page;
      pending.resolve(Buffer.from('old-page'));
      await jest.advanceTimersByTimeAsync(200);
      const delivered = old.socket.send.mock.calls.map(([buffer]) => imageBytes(buffer).toString());
      expect(delivered).toContain('new-page');
      expect(delivered).not.toContain('old-page');
    } finally { pending.resolve(Buffer.from('old-page')); await handle.stop(); }
  });

  it('sends stable pixels to a replacement viewer', async () => {
    const f = createPollingFixture();
    const replacement = { readyState: 1, send: jest.fn() };
    let viewer = f.socket;
    const handle = await new PollingStrategy().start(() => f.page, createConfig(),
      { isReady: () => true, getWebSocket: () => viewer }, f.stats);
    try {
      await jest.advanceTimersByTimeAsync(1);
      expect(f.socket.send).toHaveBeenCalledTimes(1);
      viewer = replacement;
      await jest.advanceTimersByTimeAsync(200);
      expect(replacement.send).toHaveBeenCalledTimes(1);
      expect(replacement.send.mock.calls.map(([packet]) => imageBytes(packet as Buffer).toString())).toContain('pixels');
    } finally { await handle.stop(); }
  });

  it('does not deliver an in-flight capture to a newly connected viewer', async () => {
    const pending = createDeferred<Buffer>();
    const f = createPollingFixture(() => pending.promise);
    const replacement = { readyState: 1, send: jest.fn() };
    let viewer = f.socket;
    const handle = await new PollingStrategy().start(() => f.page, createConfig(),
      { isReady: () => true, getWebSocket: () => viewer }, f.stats);
    try {
      await jest.advanceTimersByTimeAsync(1);
      viewer = replacement;
      pending.resolve(Buffer.from('old-viewer-capture'));
      f.screenshot.mockResolvedValue(Buffer.from('fresh-capture'));
      await jest.advanceTimersByTimeAsync(200);
      expect(f.socket.send).not.toHaveBeenCalled();
      expect(replacement.send.mock.calls.map(([packet]) => imageBytes(packet as Buffer).toString())).toContain('fresh-capture');
      expect(replacement.send.mock.calls.map(([packet]) => imageBytes(packet as Buffer).toString())).not.toContain('old-viewer-capture');
    } finally { pending.resolve(Buffer.from('old-viewer-capture')); await handle.stop(); }
  });

  it('caps adaptive delivery at the target instead of doubling it', async () => {
    let frame = 0;
    const f = createPollingFixture(() => Promise.resolve(Buffer.from(`frame-${frame++}`)));
    const handle = await new PollingStrategy().start(() => f.page, createConfig({ targetFps: 2 }),
      { isReady: () => true, getWebSocket: () => f.socket }, f.stats);
    try {
      await jest.advanceTimersByTimeAsync(2950);
      expect(f.socket.send).toHaveBeenCalledTimes(6);
    } finally { await handle.stop(); }
  });

  it('applies performance headers even when the next captured pixels are unchanged', async () => {
    const f = createPollingFixture();
    const handle = await new PollingStrategy().start(() => f.page, createConfig(),
      { isReady: () => true, getWebSocket: () => f.socket }, f.stats);
    try {
      await jest.advanceTimersByTimeAsync(1);
      handle.updatePerfMode?.(true);
      await jest.advanceTimersByTimeAsync(100);
      expect(f.socket.send).toHaveBeenCalledTimes(2);
      const packet = f.socket.send.mock.calls[1][0];
      const length = packet.readUInt32BE(0);
      expect(JSON.parse(packet.subarray(4, 4 + length).toString())).toMatchObject({ timing: { frame_bytes: 6 } });
      expect(packet.subarray(4 + length).toString()).toBe('pixels');
    } finally { await handle.stop(); }
  });

  it('retries stable pixels after a failed transport send', async () => {
    const f = createPollingFixture();
    f.socket.send.mockImplementationOnce(() => { throw new Error('transport failure'); });
    const handle = await new PollingStrategy().start(() => f.page, createConfig(),
      { isReady: () => true, getWebSocket: () => f.socket }, f.stats);
    try {
      await jest.advanceTimersByTimeAsync(200);
      expect(f.socket.send).toHaveBeenCalledTimes(2);
      expect(f.stats.onFrameSent).toHaveBeenCalledTimes(1);
    } finally { await handle.stop(); }
  });
});
