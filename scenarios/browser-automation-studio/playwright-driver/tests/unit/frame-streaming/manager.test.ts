import {
  startFrameStreaming,
  stopFrameStreaming,
  updateFrameStreamSettings,
  getFrameStreamSettings,
  updateFrameStreamViewport,
  isViewportUpdatePending,
} from '../../../src/frame-streaming/manager';
import type { FrameStreamOptions } from '../../../src/frame-streaming/types';

const mockLoadConfig = jest.fn();
const mockPerfCollector = {
  recordFrame: jest.fn(),
  recordSkipped: jest.fn(),
  shouldLogSummary: jest.fn().mockReturnValue(false),
  getAggregatedStats: jest.fn().mockReturnValue({
    frame_count: 0,
    skipped_count: 0,
    capture_p50_ms: 0,
    capture_p90_ms: 0,
    e2e_p50_ms: 0,
    e2e_p90_ms: 0,
    primary_bottleneck: 'none',
  }),
};
const mockPerfCollectorFromConfig = jest.fn(() => mockPerfCollector);

const mockCdpStrategy = {
  name: 'cdp-screencast',
  isSupported: jest.fn(),
  start: jest.fn(),
};
const mockPollingStrategy = {
  name: 'polling',
  isSupported: jest.fn(),
  start: jest.fn(),
};

const mockCreateCdp = jest.fn(() => mockCdpStrategy);
const mockCreatePolling = jest.fn(() => mockPollingStrategy);

const mockWsManager = {
  connect: jest.fn(),
  close: jest.fn(),
  isReady: jest.fn().mockReturnValue(true),
  getWebSocket: jest.fn().mockReturnValue({}),
};
const mockCreateWsManager = jest.fn(() => mockWsManager);
const mockBuildWsUrl = jest.fn(() => 'ws://example.com/stream');

jest.mock('../../../src/config', () => ({
  loadConfig: () => mockLoadConfig(),
}));

jest.mock('../../../src/performance', () => ({
  PerfCollector: {
    fromConfig: (...args: unknown[]) => mockPerfCollectorFromConfig(...args),
  },
}));

jest.mock('../../../src/frame-streaming/strategies', () => ({
  createCdpScreencastStrategy: () => mockCreateCdp(),
  createPollingStrategy: () => mockCreatePolling(),
}));

jest.mock('../../../src/frame-streaming/websocket', () => ({
  createWebSocketConnectionManager: () => mockCreateWsManager(),
  buildWebSocketUrl: (...args: unknown[]) => mockBuildWsUrl(...args),
}));

const flushPromises = async (): Promise<void> => {
  await new Promise((resolve) => setImmediate(resolve));
};

const sessionProvider = {
  getSession: () => ({ page: { name: 'page' } }),
};

const baseOptions: FrameStreamOptions = {
  callbackUrl: 'http://localhost/callback',
  quality: 60,
  fps: 30,
  scale: 'css',
};

describe('frame streaming manager', () => {
  beforeEach(() => {
    mockLoadConfig.mockReturnValue({
      frameStreaming: {
        useScreencast: true,
        fallbackToPolling: true,
      },
      performance: {
        enabled: true,
        includeTimingHeaders: true,
      },
    });
    mockCdpStrategy.isSupported.mockResolvedValue(true);
    mockCdpStrategy.start.mockResolvedValue({
      stop: jest.fn().mockResolvedValue(undefined),
      isActive: jest.fn().mockReturnValue(true),
      updateQuality: jest.fn(),
      updateTargetFps: jest.fn(),
      updatePerfMode: jest.fn(),
      updateViewport: jest.fn().mockResolvedValue(undefined),
      isViewportUpdatePending: jest.fn().mockReturnValue(false),
      getFrameCount: jest.fn().mockReturnValue(2),
    });
    mockPollingStrategy.start.mockResolvedValue({
      stop: jest.fn().mockResolvedValue(undefined),
      isActive: jest.fn().mockReturnValue(true),
      updateQuality: jest.fn(),
      updateTargetFps: jest.fn(),
      updatePerfMode: jest.fn(),
      getFrameCount: jest.fn().mockReturnValue(3),
    });
  });

  afterEach(async () => {
    await stopFrameStreaming('session-1');
    await stopFrameStreaming('session-2');
    jest.clearAllMocks();
  });

  it('starts streaming with screencast when supported', async () => {
    startFrameStreaming('session-1', sessionProvider, baseOptions);
    await flushPromises();

    expect(mockWsManager.connect).toHaveBeenCalled();
    expect(mockCdpStrategy.start).toHaveBeenCalled();

    const settings = getFrameStreamSettings('session-1');
    expect(settings?.quality).toBe(60);
    expect(settings?.fps).toBe(30);
    expect(settings?.isStreaming).toBe(true);
  });

  it('falls back to polling when screencast start fails', async () => {
    mockCdpStrategy.start.mockRejectedValueOnce(new Error('cdp failed'));

    startFrameStreaming('session-2', sessionProvider, baseOptions);
    await flushPromises();

    expect(mockPollingStrategy.start).toHaveBeenCalled();
    const settings = getFrameStreamSettings('session-2');
    expect(settings?.isStreaming).toBe(true);
  });

  it.each([true, false])('selects physical-pixel capture for device scale with failure fallback=%s [REQ:BAS-RH-J23]', async (fallbackToPolling) => {
    mockLoadConfig().frameStreaming.fallbackToPolling = fallbackToPolling;
    startFrameStreaming('session-1', sessionProvider, { ...baseOptions, scale: 'device' });
    await flushPromises();

    expect(mockCdpStrategy.start).not.toHaveBeenCalled();
    expect(mockPollingStrategy.start).toHaveBeenCalledWith(
      expect.any(Function), expect.objectContaining({ scale: 'device' }),
      expect.any(Object), expect.any(Object)
    );
    expect(getFrameStreamSettings('session-1')).toMatchObject({ scale: 'device', isStreaming: true });
  });

  it('updates stream settings when active', async () => {
    startFrameStreaming('session-1', sessionProvider, baseOptions);
    await flushPromises();

    const updated = await updateFrameStreamSettings('session-1', {
      quality: 75,
      fps: 24,
      perfMode: false,
    });

    expect(updated).toBe(true);
    const settings = getFrameStreamSettings('session-1');
    expect(settings?.quality).toBe(75);
    expect(settings?.fps).toBe(24);
    expect(settings?.perfMode).toBe(false);
  });

  describe('effective stream control receipts [REQ:BAS-RH-J23]', () => {
    it.each([0, 7.5])('reports observed frame rate %s separately from its target', async (actualFps) => {
      mockPerfCollector.getAggregatedStats.mockReturnValue({ actual_fps: actualFps } as ReturnType<typeof mockPerfCollector.getAggregatedStats>);
      startFrameStreaming('session-1', sessionProvider, baseOptions);
      await flushPromises();
      expect(getFrameStreamSettings('session-1')).toMatchObject({ fps: 30, currentFps: actualFps });
    });

    it('keeps the prior quality and pending receipt until the capture owner applies the change', async () => {
      startFrameStreaming('session-1', sessionProvider, baseOptions);
      await flushPromises();
      const handle = await mockCdpStrategy.start.mock.results[0].value;
      let applied!: () => void;
      const captureChange = new Promise<void>((resolve) => { applied = resolve; });
      handle.updateQuality.mockReturnValue(captureChange);
      let returned = false;
      const update = Promise.resolve(updateFrameStreamSettings('session-1', { quality: 20 })).then((value) => {
        returned = true; return value;
      });
      await flushPromises();
      const before = { returned, quality: getFrameStreamSettings('session-1')?.quality };
      applied();
      const changed = await update;
      expect(before).toEqual({ returned: false, quality: 60 });
      expect(changed).toBe(true);
      expect(getFrameStreamSettings('session-1')?.quality).toBe(20);
    });

    it('joins pending control application at stop without acknowledging the obsolete update', async () => {
      startFrameStreaming('session-1', sessionProvider, baseOptions);
      await flushPromises();
      const handle = await mockCdpStrategy.start.mock.results[0].value;
      let applied!: () => void;
      handle.updateQuality.mockReturnValue(new Promise<void>((resolve) => { applied = resolve; }));
      const update = updateFrameStreamSettings('session-1', { quality: 20 });
      await flushPromises();
      let stopped = false;
      const stop = stopFrameStreaming('session-1').then(() => { stopped = true; });
      await flushPromises();
      const before = stopped;
      applied();
      const [changed] = await Promise.all([update, stop]);
      expect(before).toBe(false);
      expect(changed).toBe(false);
      expect(getFrameStreamSettings('session-1')).toBeNull();
      expect(handle.stop).toHaveBeenCalledTimes(1);
    });

    it('does not acknowledge quality that the capture owner rejected', async () => {
      startFrameStreaming('session-1', sessionProvider, baseOptions);
      await flushPromises();
      const handle = await mockCdpStrategy.start.mock.results[0].value;
      const failure = Promise.reject(new Error('encoder rejected quality'));
      void failure.catch(() => {});
      handle.updateQuality.mockReturnValue(failure);
      await expect(Promise.resolve(updateFrameStreamSettings('session-1', { quality: 20 }))).rejects.toThrow('encoder rejected quality');
      expect(getFrameStreamSettings('session-1')?.quality).toBe(60);
    });
  });

  it('updates viewport when strategy supports it', async () => {
    startFrameStreaming('session-1', sessionProvider, baseOptions);
    await flushPromises();

    const result = await updateFrameStreamViewport('session-1', { width: 800, height: 600 });
    expect(result.success).toBe(true);
    expect(isViewportUpdatePending('session-1')).toBe(false);
  });
  describe('lifecycle ownership', () => {
    const deferred = <T>() => {
      let resolve!: (value: T) => void;
      const promise = new Promise<T>((done) => { resolve = done; });
      return { promise, resolve };
    };
    const handle = () => ({
      stop: jest.fn().mockResolvedValue(undefined),
      isActive: jest.fn().mockReturnValue(true),
      getFrameCount: jest.fn().mockReturnValue(0),
    });

    it('keeps replacement tracking after the old capture finishes stopping', async () => {
      const gate = deferred<void>();
      const old = handle();
      old.stop.mockReturnValue(gate.promise);
      const next = handle();
      mockCdpStrategy.start.mockResolvedValueOnce(old).mockResolvedValueOnce(next);
      startFrameStreaming('session-1', sessionProvider, baseOptions);
      await flushPromises();
      startFrameStreaming('session-1', sessionProvider, { ...baseOptions, quality: 85 });
      await flushPromises();
      gate.resolve();
      await flushPromises();
      expect(getFrameStreamSettings('session-1')?.quality).toBe(85);
      await stopFrameStreaming('session-1');
      expect(old.stop).toHaveBeenCalledTimes(1);
      expect(next.stop).toHaveBeenCalledTimes(1);
    });

    it('joins pending acquisition and disposes the late capture before stop succeeds', async () => {
      const gate = deferred<ReturnType<typeof handle>>();
      const late = handle();
      mockCdpStrategy.start.mockReturnValueOnce(gate.promise);
      startFrameStreaming('session-1', sessionProvider, baseOptions);
      await flushPromises();
      const stopped = stopFrameStreaming('session-1');
      let settled = false;
      void stopped.then(() => { settled = true; });
      try {
        await flushPromises();
        expect(settled).toBe(false);
        expect(mockWsManager.close).toHaveBeenCalled();
        const transport = mockCdpStrategy.start.mock.calls[0]![2];
        expect(transport.isReady()).toBe(false);
      } finally {
        gate.resolve(late);
        await stopped;
      }
      expect(late.stop).toHaveBeenCalledTimes(1);
      expect(getFrameStreamSettings('session-1')).toBeNull();
    });

    it('does not start capture after a stopped support probe resolves', async () => {
      const probe = deferred<boolean>();
      mockCdpStrategy.isSupported.mockReturnValueOnce(probe.promise);
      startFrameStreaming('session-1', sessionProvider, baseOptions);
      await flushPromises();
      const stopped = stopFrameStreaming('session-1');
      probe.resolve(true);
      await stopped;
      await flushPromises();
      expect(mockCdpStrategy.start).not.toHaveBeenCalled();
      expect(mockWsManager.close).toHaveBeenCalled();
      expect(getFrameStreamSettings('session-1')).toBeNull();
    });

    it('closes transport when every capture strategy fails', async () => {
      mockCdpStrategy.start.mockRejectedValueOnce(new Error('CDP unavailable'));
      mockPollingStrategy.start.mockRejectedValueOnce(new Error('polling unavailable'));
      startFrameStreaming('session-1', sessionProvider, baseOptions);
      await flushPromises();
      expect(mockWsManager.close).toHaveBeenCalled();
      expect(getFrameStreamSettings('session-1')).toBeNull();
    });

    it('coalesces replacement requests while old acquisition is pending', async () => {
      const gate = deferred<ReturnType<typeof handle>>();
      const old = handle();
      const newest = handle();
      mockCdpStrategy.start.mockReturnValueOnce(gate.promise).mockResolvedValueOnce(newest);
      startFrameStreaming('session-1', sessionProvider, baseOptions);
      await flushPromises();
      startFrameStreaming('session-1', sessionProvider, { ...baseOptions, quality: 70 });
      startFrameStreaming('session-1', sessionProvider, { ...baseOptions, quality: 90 });
      await flushPromises();
      expect(mockCdpStrategy.start).toHaveBeenCalledTimes(1);
      gate.resolve(old);
      await flushPromises();
      expect(old.stop).toHaveBeenCalledTimes(1);
      expect(mockCdpStrategy.start).toHaveBeenCalledTimes(2);
      expect(getFrameStreamSettings('session-1')?.quality).toBe(90);
      await Promise.all([stopFrameStreaming('session-1'), stopFrameStreaming('session-1')]);
      expect(newest.stop).toHaveBeenCalledTimes(1);
    });

    it('contains a disappeared page before capture startup and closes its socket', async () => {
      startFrameStreaming('session-1', { getSession: () => { throw new Error('session closed'); } }, baseOptions);
      await flushPromises();
      expect(mockWsManager.close).toHaveBeenCalled();
      expect(mockCdpStrategy.start).not.toHaveBeenCalled();
      expect(getFrameStreamSettings('session-1')).toBeNull();
    });

    it('retains failed disposal for an explicit retry', async () => {
      const capture = handle();
      capture.stop.mockRejectedValueOnce(new Error('detach failed'));
      mockCdpStrategy.start.mockResolvedValueOnce(capture);
      startFrameStreaming('session-1', sessionProvider, baseOptions);
      await flushPromises();
      await expect(stopFrameStreaming('session-1')).rejects.toThrow('detach failed');
      expect(mockWsManager.close).toHaveBeenCalled();
      await stopFrameStreaming('session-1');
      expect(capture.stop).toHaveBeenCalledTimes(2);
      expect(getFrameStreamSettings('session-1')).toBeNull();
    });
  });

});
