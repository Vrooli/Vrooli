import { renderHook, waitFor, act } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { useSystemMonitor } from './useSystemMonitor';

const mocks = vi.hoisted(() => ({
  protoFetch: vi.fn(),
  checkHealth: vi.fn(),
  refreshHealth: vi.fn(),
  toggleMonitoring: vi.fn(),
  usePolling: vi.fn(),
  appendGpuPoint: vi.fn(),
  appendDiskPoints: vi.fn(),
  appendDiskUsagePoint: vi.fn(),
  fetchMetricsTimeline: vi.fn(),
  showApiError: vi.fn(),
}));

vi.mock('../../../shared/api/apiFetch', () => ({
  protoFetch: mocks.protoFetch,
  toApiError: (error: unknown) => ({ error: error instanceof Error ? error.message : 'request failed' }),
}));
vi.mock('../../../shared/hooks/usePolling', () => ({ usePolling: mocks.usePolling }));
vi.mock('./useHealthCheck', () => ({
  useHealthCheck: () => ({
    healthStatus: { status: 'healthy' },
    healthError: null,
    checkHealth: mocks.checkHealth,
    refreshHealth: mocks.refreshHealth,
    toggleMonitoring: mocks.toggleMonitoring,
  }),
}));
vi.mock('../../../shared/components/ToastProvider', () => ({
  useToast: () => ({ showApiError: mocks.showApiError }),
}));
vi.mock('./useMetricHistory', () => ({
  useMetricHistory: () => ({
    metricHistory: { windowSeconds: 60, sampleIntervalSeconds: 5, cpu: [], memory: [], network: [] },
    fetchMetricsTimeline: mocks.fetchMetricsTimeline,
    appendGpuPoint: mocks.appendGpuPoint,
    appendDiskPoints: mocks.appendDiskPoints,
    appendDiskUsagePoint: mocks.appendDiskUsagePoint,
  }),
}));

const ts = timestampFromDate(new Date('2026-02-02T00:00:00Z'));

const successPayloads = () => ({
  metrics: { timestamp: ts, gpuUsage: 12 },
  detailed: { timestamp: ts, memoryDetails: { diskUsage: { percent: 70 } } },
  processes: { processHealth: { totalProcesses: 1 } },
  infrastructure: { timestamp: ts, storageIo: { readMbPerSec: 1, writeMbPerSec: 2 } },
  investigations: [
    { id: 'new', startTime: ts },
    { id: 'unknown', startTime: undefined },
  ],
});

describe('useSystemMonitor', () => {
  beforeEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
    mocks.checkHealth.mockResolvedValue(true);
    mocks.refreshHealth.mockResolvedValue(undefined);
    mocks.toggleMonitoring.mockResolvedValue(undefined);
    mocks.fetchMetricsTimeline.mockResolvedValue(undefined);
    const payloads = successPayloads();
    mocks.protoFetch.mockImplementation((url: string) => {
      if (url.includes('/metrics/current')) return Promise.resolve(payloads.metrics);
      if (url.includes('/metrics/detailed')) return Promise.resolve(payloads.detailed);
      if (url.includes('/metrics/processes')) return Promise.resolve(payloads.processes);
      if (url.includes('/metrics/infrastructure')) return Promise.resolve(payloads.infrastructure);
      if (url.includes('/investigations')) return Promise.resolve(payloads.investigations);
      if (url.includes('/maintenance/state')) return Promise.resolve({ maintenanceState: 'inactive', success: true });
      return Promise.resolve({});
    });
  });

  it('loads primary and deferred telemetry, updates history, and refreshes', async () => {
    const { result } = renderHook(() => useSystemMonitor(300));

    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    expect(result.current.metrics).toBeTruthy();
    expect(result.current.detailedMetrics).toBeNull();

    act(() => {
      result.current.refresh();
    });

    await waitFor(() => { expect(result.current.detailedMetrics).toBeTruthy(); }, { timeout: 5000 });
    expect(result.current.processMonitorData).toBeTruthy();
    expect(result.current.infrastructureData).toBeTruthy();
    expect(result.current.investigations).toHaveLength(2);
    expect(mocks.appendGpuPoint).toHaveBeenCalled();
    expect(mocks.appendDiskUsagePoint).toHaveBeenCalled();
    expect(mocks.appendDiskPoints).toHaveBeenCalledWith(expect.any(String), 1, 2);
    expect(mocks.fetchMetricsTimeline).toHaveBeenCalledWith(300);
  });

  it('reports primary metric failures and avoids deferred work when health is down', async () => {
    mocks.checkHealth.mockResolvedValue(false);
    const { result } = renderHook(() => useSystemMonitor());

    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    expect(result.current.error?.error).toBe('API server is not responding');
    expect(result.current.metrics).toBeNull();

    mocks.checkHealth.mockResolvedValue(true);
    mocks.protoFetch.mockImplementation((url: string) => {
      if (url.includes('/metrics/current')) return Promise.reject(new Error('metrics unavailable'));
      return Promise.resolve({});
    });
    act(() => { result.current.refresh(); });
    await waitFor(() => { expect(result.current.error?.error).toBe('metrics unavailable'); });
    expect(mocks.showApiError).toHaveBeenCalled();
  });

  it('activates monitoring after the delayed maintenance gate and restores it on unmount', async () => {
    vi.useFakeTimers();
    const { unmount } = renderHook(() => useSystemMonitor());
    await act(async () => {
      vi.advanceTimersByTime(2200);
      await vi.runAllTimersAsync();
    });
    expect(mocks.protoFetch.mock.calls.some(([url]) => url === '/metrics/detailed')).toBe(true);
    unmount();
    expect(globalThis.fetch).toBeDefined();
  });

  it('tracks independent deferred failures, stale metrics, and fresh-mode polling', async () => {
    let metricsCalls = 0;
    mocks.protoFetch.mockImplementation((url: string) => {
      if (url.includes('/metrics/current')) {
        metricsCalls += 1;
        return metricsCalls <= 1 ? Promise.resolve({ metrics: { timestamp: ts } }) : Promise.reject(new Error('metrics down'));
      }
      if (url.includes('/maintenance/state')) return Promise.resolve({ maintenanceState: 'active', success: true });
      if (url.includes('/metrics/detailed')) return Promise.reject(new Error('detailed down'));
      if (url.includes('/metrics/processes')) return Promise.reject(new Error('processes down'));
      if (url.includes('/metrics/infrastructure')) return Promise.resolve({ timestamp: ts, storageIo: { readMbPerSec: Number.NaN, writeMbPerSec: Number.NaN } });
      if (url.includes('/investigations')) return Promise.resolve([{ id: 'no-time' }, { id: 'newer', startTime: ts }]);
      return Promise.resolve({});
    });
    const { result } = renderHook(() => useSystemMonitor(60, { enabled: false }));
    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    act(() => { result.current.refresh(); });
    await waitFor(() => { expect(result.current.subsystemErrors.detailedMetrics).toBeTruthy(); });
    expect(result.current.subsystemErrors.processes).toBeTruthy();
    expect(result.current.subsystemErrors.investigations).toBeUndefined();

    await act(async () => { result.current.refreshMetrics(); await Promise.resolve(); });
    await act(async () => { result.current.refreshMetrics(); await Promise.resolve(); });
    await act(async () => { result.current.refreshMetrics(); await Promise.resolve(); });
    await waitFor(() => { expect(result.current.isStale).toBe(true); });
    expect(mocks.usePolling).toHaveBeenCalledWith(expect.any(Function), 5000, false, expect.anything());
  });

  it('clears subsystem errors and preserves honest empty optional telemetry', async () => {
    mocks.protoFetch.mockImplementation((url: string, _parser: unknown, options?: RequestInit) => {
      if (url.includes('/metrics/current')) return Promise.resolve({ timestamp: ts, gpuUsage: 12 });
      if (url === '/maintenance/state' && options?.method === 'POST') return Promise.resolve({ success: false });
      if (url === '/maintenance/state') return Promise.resolve({ maintenanceState: 'inactive' });
      if (url.includes('/metrics/detailed')) return Promise.resolve({ timestamp: ts });
      if (url.includes('/metrics/processes')) return Promise.resolve({});
      if (url.includes('/metrics/infrastructure')) return Promise.resolve({ timestamp: ts });
      if (url.includes('/investigations')) return Promise.resolve([{ id: 'a' }, { id: 'b' }]);
      return Promise.resolve({});
    });
    const { result } = renderHook(() => useSystemMonitor(60));
    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    act(() => { result.current.refresh(); });
    await waitFor(() => { expect(result.current.detailedMetrics).toBeTruthy(); });
    await waitFor(() => { expect(result.current.error).toBeNull(); });
    expect(result.current.metrics).toBeTruthy();
    expect(result.current.subsystemErrors).toEqual({});
    expect(mocks.appendDiskPoints).not.toHaveBeenCalled();
  });
});
