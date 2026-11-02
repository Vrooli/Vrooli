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
  clearHistory: vi.fn(),
  invalidatePending: vi.fn(),
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
    clearHistory: mocks.clearHistory,
    invalidatePending: mocks.invalidatePending,
    appendGpuPoint: mocks.appendGpuPoint,
    appendDiskPoints: mocks.appendDiskPoints,
    appendDiskUsagePoint: mocks.appendDiskUsagePoint,
  }),
}));

const ts = timestampFromDate(new Date('2026-02-02T00:00:00Z'));

const successPayloads = () => ({
  metrics: { timestamp: ts, gpuUsage: 12, gpu: { observedAt: ts, state: { case: 'measured', value: 12 } } },
  detailed: { timestamp: ts, memoryDetails: { diskUsage: { percent: 70 } } },
  processes: { processHealth: { totalProcesses: 1 } },
  infrastructure: { timestamp: ts, storageIo: { readMbPerSec: 1, writeMbPerSec: 2, readRate: { observedAt: ts, state: { case: 'measured', value: 1 } }, writeRate: { observedAt: ts, state: { case: 'measured', value: 2 } } } },
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
    expect(mocks.appendDiskPoints).toHaveBeenCalledWith(expect.any(String), 1, 2, false);
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

  it('does not change collection state on entry or exit', async () => {
    vi.useFakeTimers();
    const fetchSpy = vi.spyOn(globalThis, 'fetch');
    const { unmount } = renderHook(() => useSystemMonitor());
    await act(async () => { await vi.advanceTimersByTimeAsync(3000); });
    unmount();
    expect(mocks.protoFetch.mock.calls.some(([url, , options]) =>
      url === '/maintenance/state' && options?.method === 'POST')).toBe(false);
    expect(fetchSpy).not.toHaveBeenCalled();
    expect(mocks.toggleMonitoring).not.toHaveBeenCalled();
    fetchSpy.mockRestore();
  });

  it('tracks independent deferred failures, stale metrics, and polling', async () => {
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

  it('does not append legacy placeholder disk scalars without measured rate states', async () => {
    mocks.protoFetch.mockImplementation((url: string) => url.includes('/metrics/infrastructure')
      ? Promise.resolve({ timestamp: ts, storageIo: { readMbPerSec: 15, writeMbPerSec: 8 } })
      : Promise.resolve({ timestamp: ts }));
    const { result } = renderHook(() => useSystemMonitor());
    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    act(() => { result.current.refresh(); });
    await waitFor(() => { expect(result.current.infrastructureData).toBeTruthy(); });
    expect(mocks.appendDiskPoints).not.toHaveBeenCalled();
  });

  it('does not invent a timestamp for measured disk rates', async () => {
    mocks.protoFetch.mockImplementation((url: string) => url.includes('/metrics/infrastructure')
      ? Promise.resolve({ storageIo: { readRate: { state: { case: 'measured', value: 1 } }, writeRate: { state: { case: 'measured', value: 2 } } } })
      : Promise.resolve({ timestamp: ts }));
    const { result } = renderHook(() => useSystemMonitor());
    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    act(() => { result.current.refresh(); });
    await waitFor(() => { expect(result.current.infrastructureData).toBeTruthy(); });
    expect(mocks.appendDiskPoints).not.toHaveBeenCalled();
  });

  it('does not combine disk rates whose observation timestamps differ within a millisecond', async () => {
    mocks.protoFetch.mockImplementation((url: string) => url.includes('/metrics/infrastructure')
      ? Promise.resolve({ timestamp: ts, storageIo: { readRate: { observedAt: ts, state: { case: 'measured', value: 1 } }, writeRate: { observedAt: { ...ts, nanos: ts.nanos + 1 }, state: { case: 'measured', value: 2 } } } })
      : Promise.resolve({ timestamp: ts }));
    const { result } = renderHook(() => useSystemMonitor());
    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    await act(async () => { await result.current.refreshMetrics(); });
    expect(mocks.appendDiskPoints).not.toHaveBeenCalled();
  });

  it('retains the disk gap through a repeated old observation after a failed refresh', async () => {
    let diskCalls = 0;
    const later = timestampFromDate(new Date('2026-02-02T00:00:10Z'));
    const older = timestampFromDate(new Date('2026-02-01T23:59:55Z'));
    mocks.protoFetch.mockImplementation((url: string) => {
      if (!url.includes('/metrics/infrastructure')) return Promise.resolve({ timestamp: ts });
      diskCalls += 1;
      const observedAt = diskCalls >= 5 ? later : diskCalls === 3 ? older : ts;
      return diskCalls === 2 ? Promise.reject(new Error('disk refresh failed')) : Promise.resolve({ timestamp: ts, storageIo: { readRate: { observedAt, state: { case: 'measured', value: 1 } }, writeRate: { observedAt, state: { case: 'measured', value: 2 } } } });
    });
    const { result } = renderHook(() => useSystemMonitor());
    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    for (let n = 1; n <= 5; n += 1) {
      act(() => { result.current.refreshMetrics(); });
      await waitFor(() => { expect(diskCalls).toBe(n); });
    }
    expect(mocks.appendDiskPoints).toHaveBeenCalledTimes(2);
    expect(mocks.appendDiskPoints).toHaveBeenLastCalledWith('2026-02-02T00:00:10.000Z', 1, 2, true);
  });

  it('allows a disk observation again after the subject changes away and back', async () => {
    const { result, rerender } = renderHook(({ node }: { node?: string }) => useSystemMonitor(120, { node }), { initialProps: { node: undefined } });
    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    act(() => { result.current.refreshMetrics(); });
    await waitFor(() => { expect(mocks.appendDiskPoints).toHaveBeenCalledTimes(1); });
    rerender({ node: 'remote' });
    rerender({ node: undefined });
    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    act(() => { result.current.refreshMetrics(); });
    await waitFor(() => { expect(mocks.appendDiskPoints).toHaveBeenCalledTimes(2); });
  });

  it('uses GPU observation time, deduplicates repeated readings and breaks history after a failed sample', async () => {
    const later = timestampFromDate(new Date('2026-02-02T00:00:10Z'));
    const older = timestampFromDate(new Date('2026-02-01T23:59:55Z'));
    let reads = 0;
    mocks.protoFetch.mockImplementation((url: string) => {
      if (!url.includes('/metrics/current')) return Promise.resolve({});
      reads += 1;
      if (reads === 3) return Promise.reject(new Error('metrics unavailable'));
      const observedAt = reads >= 6 ? later : reads === 4 ? older : ts;
      return Promise.resolve({ timestamp: later, gpu: { observedAt, state: { case: 'measured', value: reads >= 6 ? 0 : 12 } } });
    });
    const { result } = renderHook(() => useSystemMonitor());
    await waitFor(() => { expect(mocks.appendGpuPoint).toHaveBeenCalledTimes(1); });
    expect(mocks.appendGpuPoint).toHaveBeenNthCalledWith(1, '2026-02-02T00:00:00.000Z', 12, false);
    for (let n = 2; n <= 6; n += 1) {
      act(() => { result.current.refreshMetrics(); });
      await waitFor(() => { expect(reads).toBeGreaterThanOrEqual(n); });
    }
    await waitFor(() => { expect(mocks.appendGpuPoint).toHaveBeenCalledTimes(2); });
    expect(mocks.appendGpuPoint).toHaveBeenNthCalledWith(2, '2026-02-02T00:00:10.000Z', 0, true);
  });

  it('does not invent a GPU sample time when observation time is missing', async () => {
    mocks.protoFetch.mockImplementation((url: string) => url.includes('/metrics/current')
      ? Promise.resolve({ timestamp: ts, gpu: { state: { case: 'measured', value: 12 } } })
      : Promise.resolve({}));
    const { result } = renderHook(() => useSystemMonitor());
    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    expect(mocks.appendGpuPoint).not.toHaveBeenCalled();
  });

  it('does not append a legacy GPU scalar when the measurement is unsupported', async () => {
    mocks.protoFetch.mockImplementation((url: string) => url.includes('/metrics/current')
      ? Promise.resolve({ timestamp: ts, gpuUsage: 0, gpu: { state: { case: 'unsupportedReason', value: 'no GPU' } } })
      : Promise.resolve({}));
    const { result } = renderHook(() => useSystemMonitor());
    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    expect(mocks.appendGpuPoint).not.toHaveBeenCalled();
  });

  it('never renders the prior subject evidence under a newly selected subject', async () => {
    const localGraph = { available: true, devices: [{ id: 'local-device' }], subsystems: [] };
    mocks.protoFetch.mockImplementation((url: string) => {
      if (url.includes('?node=remote')) return new Promise(() => {});
      if (url === '/metrics/devices') return Promise.resolve(localGraph);
      if (url === '/metrics/current') return Promise.resolve(successPayloads().metrics);
      return Promise.resolve({});
    });
    const renders: Array<{ node?: string; graph: unknown; metrics: unknown; history: unknown }> = [];
    const { result, rerender } = renderHook(({ node }: { node?: string }) => {
      const value = useSystemMonitor(120, { node });
      renders.push({ node, graph: value.deviceGraph, metrics: value.metrics, history: value.metricHistory });
      return value;
    }, { initialProps: { node: undefined } });
    await waitFor(() => { expect(result.current.deviceGraph).toBe(localGraph); });
    rerender({ node: 'remote' });
    const firstRemote = renders.find(render => render.node === 'remote');
    expect(firstRemote).toEqual({ node: 'remote', graph: null, metrics: null, history: null });
    expect(renders.filter(render => render.node === 'remote').every(render => render.graph !== localGraph)).toBe(true);
  });

  it('hides a remote graph immediately when returning to the local subject', async () => {
    const remoteGraph = { available: true, devices: [{ id: 'remote-device' }], subsystems: [] };
    mocks.protoFetch.mockImplementation((url: string) => {
      if (url === '/metrics/devices?node=remote') return Promise.resolve(remoteGraph);
      if (url === '/metrics/current?node=remote') return Promise.resolve(successPayloads().metrics);
      return new Promise(() => {});
    });
    const renders: Array<{ node?: string; graph: unknown; metrics: unknown; history: unknown }> = [];
    const { result, rerender } = renderHook(({ node }: { node?: string }) => {
      const value = useSystemMonitor(120, { node });
      renders.push({ node, graph: value.deviceGraph, metrics: value.metrics, history: value.metricHistory });
      return value;
    }, { initialProps: { node: 'remote' as string | undefined } });
    await waitFor(() => { expect(result.current.deviceGraph).toBe(remoteGraph); });
    rerender({ node: undefined });
    expect(renders.find(render => render.node === undefined)).toEqual({ node: undefined, graph: null, metrics: null, history: null });
    expect(renders.filter(render => render.node === undefined).every(render => render.graph !== remoteGraph)).toBe(true);
  });

  it('does not apply an in-flight metric response after view pause and resumes with a new read', async () => {
    let finish!: (value: unknown) => void;
    let reads = 0;
    mocks.protoFetch.mockImplementation((url: string) => {
      if (!url.includes('/metrics/current')) return Promise.resolve({});
      reads += 1;
      if (reads === 2) return new Promise(resolve => { finish = resolve; });
      return Promise.resolve({ timestamp: ts, gpuUsage: reads === 1 ? 12 : 33 });
    });
    const { result, rerender } = renderHook(({ enabled }) => useSystemMonitor(120, { enabled }), { initialProps: { enabled: true } });
    await waitFor(() => { expect(result.current.metrics?.gpuUsage).toBe(12); });
    act(() => { result.current.refreshMetrics(); });
    await waitFor(() => { expect(finish).toBeDefined(); });
    rerender({ enabled: false });
    await act(async () => { finish({ timestamp: ts, gpuUsage: 99 }); await Promise.resolve(); });
    expect(result.current.metrics?.gpuUsage).toBe(12);
    expect(mocks.invalidatePending).toHaveBeenCalled();
    rerender({ enabled: true });
    act(() => { result.current.refreshMetrics(); });
    await waitFor(() => { expect(result.current.metrics?.gpuUsage).toBe(33); });
  });

  it('discards a late local metric response after changing subjects', async () => {
    let resolveLocal!: (value: unknown) => void;
    mocks.protoFetch.mockImplementation((url: string) => {
      if (url === '/metrics/current') return new Promise(resolve => { resolveLocal = resolve; });
      if (url === '/metrics/current?node=remote') return Promise.resolve({ timestamp: ts, gpuUsage: 7 });
      return Promise.resolve({});
    });
    const { result, rerender } = renderHook(({ node }: { node?: string }) => useSystemMonitor(120, { node }), { initialProps: { node: undefined } });
    await waitFor(() => { expect(resolveLocal).toBeDefined(); });
    rerender({ node: 'remote' });
    await waitFor(() => { expect(result.current.metrics?.gpuUsage).toBe(7); });
    await act(async () => { resolveLocal({ timestamp: ts, gpuUsage: 99 }); await Promise.resolve(); });
    expect(result.current.metrics?.gpuUsage).toBe(7);
  });

  it('clears remote history when deliberately returning to the local subject', async () => {
    const { rerender } = renderHook(({ node }: { node?: string }) => useSystemMonitor(120, { node }), { initialProps: { node: 'remote' } });
    await waitFor(() => { expect(mocks.clearHistory).toHaveBeenCalled(); });
    mocks.clearHistory.mockClear();
    rerender({ node: undefined });
    await waitFor(() => { expect(mocks.clearHistory).toHaveBeenCalled(); });
  });

  it('reads a remote node through the relay and skips host-local readings', async () => {
    mocks.protoFetch.mockImplementation((url: string) => {
      if (url.includes('/metrics/current')) return Promise.reject(new Error('relay unavailable'));
      return Promise.resolve({});
    });
    const { result } = renderHook(() => useSystemMonitor(120, { node: 'minimouse' }));

    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    act(() => { result.current.refresh(); });

    await waitFor(() => {
      expect(mocks.protoFetch.mock.calls.some(([url]) => url === '/metrics/current?node=minimouse')).toBe(true);
    });
    const requested = mocks.protoFetch.mock.calls.map(([url]) => String(url));
    for (const localOnly of ['/metrics/detailed', '/metrics/processes', '/metrics/infrastructure', '/investigations']) {
      expect(requested.some(url => url.startsWith(localOnly))).toBe(false);
    }
    // The remote identity strip reports the failure persistently; no toast repeats it.
    await waitFor(() => { expect(result.current.error?.error).toBe('relay unavailable'); });
    expect(mocks.showApiError).not.toHaveBeenCalled();
    expect(mocks.clearHistory).toHaveBeenCalled();
  });
});

