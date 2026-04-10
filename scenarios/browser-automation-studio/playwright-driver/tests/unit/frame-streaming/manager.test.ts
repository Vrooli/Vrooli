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
      updateViewport: jest.fn().mockResolvedValue(undefined),
      isViewportUpdatePending: jest.fn().mockReturnValue(false),
      getFrameCount: jest.fn().mockReturnValue(2),
    });
    mockPollingStrategy.start.mockResolvedValue({
      stop: jest.fn().mockResolvedValue(undefined),
      isActive: jest.fn().mockReturnValue(true),
      updateQuality: jest.fn(),
      updateTargetFps: jest.fn(),
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

  it('updates stream settings when active', async () => {
    startFrameStreaming('session-1', sessionProvider, baseOptions);
    await flushPromises();

    const updated = updateFrameStreamSettings('session-1', {
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

  it('updates viewport when strategy supports it', async () => {
    startFrameStreaming('session-1', sessionProvider, baseOptions);
    await flushPromises();

    const result = await updateFrameStreamViewport('session-1', { width: 800, height: 600 });
    expect(result.success).toBe(true);
    expect(isViewportUpdatePending('session-1')).toBe(false);
  });
});
