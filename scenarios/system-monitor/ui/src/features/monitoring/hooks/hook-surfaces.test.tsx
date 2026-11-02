import { renderHook, waitFor, act } from '@testing-library/react';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useMetricHistory } from './useMetricHistory';
import { useHealthCheck } from './useHealthCheck';
import { useApiCall } from '../../../shared/hooks/useApiCall';

const mocks = vi.hoisted(() => ({
  apiFetch: vi.fn(),
  protoFetch: vi.fn(),
}));

vi.mock('../../../shared/api/apiFetch', () => ({
  apiFetch: mocks.apiFetch,
  protoFetch: mocks.protoFetch,
  toApiError: (error: unknown) => ({ error: error instanceof Error ? error.message : 'request failed' }),
}));

const pointTime = timestampFromDate(new Date('2026-02-02T00:00:00Z'));

describe('monitoring hook surfaces', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.apiFetch.mockResolvedValue({ status: 'healthy', processor_active: true, maintenance_state: 'active' });
    mocks.protoFetch.mockResolvedValue({ success: true, maintenanceState: 'inactive' });
  });

  it('builds and appends metric history while filtering unsupported GPU samples', async () => {
    mocks.protoFetch.mockResolvedValue({
      windowSeconds: 120,
      sampleIntervalSeconds: 5,
      samples: [
        {
          timestamp: pointTime,
          cpu: { state: { case: 'measured', value: 1 } },
          memory: { state: { case: 'measured', value: 2 } },
          connections: { state: { case: 'measured', value: 3 } },
          gpu: { observedAt: pointTime, state: { case: 'measured', value: 4 } }
        },
        {
          timestamp: undefined,
          cpu: { state: { case: 'measured', value: 5 } },
          memory: { state: { case: 'measured', value: 6 } },
          connections: { state: { case: 'measured', value: 7 } },
          gpu: { state: { case: 'unsupported', value: 'no GPU' } }
        },
      ],
    });
    const setError = vi.fn();
    const { result } = renderHook(() => useMetricHistory(setError));

    await act(async () => { await result.current.fetchMetricsTimeline(120); });
    expect(result.current.metricHistory?.cpu).toHaveLength(1);
    expect(result.current.metricHistory?.gpu).toHaveLength(1);

    act(() => {
      result.current.appendGpuPoint('now', 9);
      result.current.appendDiskPoints('now', 2, Number.NaN);
      result.current.appendDiskUsagePoint('now', 80);
      result.current.appendDiskPoints('shared-observation', 4, 5);
      result.current.appendDiskPoints('shared-observation', 4, 5);
    });
    expect(result.current.metricHistory?.gpu?.at(-1)?.value).toBe(9);
    expect(result.current.metricHistory?.diskRead?.at(-1)?.value).toBe(4);
    expect(result.current.metricHistory?.diskRead?.filter(point => point.timestamp === 'shared-observation')).toHaveLength(1);
    expect(result.current.metricHistory?.diskWrite?.some(point => point.timestamp === 'now')).toBe(false);
    expect(result.current.metricHistory?.diskWrite?.filter(point => point.timestamp === 'shared-observation')).toHaveLength(1);
    expect(result.current.metricHistory?.diskUsage?.at(-1)?.value).toBe(80);
  });

  it('commits only the latest requested timeline and discards a cleared subject', async () => {
    const pending: Array<(value: unknown) => void> = [];
    mocks.protoFetch.mockImplementation(() => new Promise(resolve => { pending.push(resolve); }));
    const { result } = renderHook(() => useMetricHistory(vi.fn()));
    let first!: Promise<void>;
    let second!: Promise<void>;
    await act(async () => {
      first = result.current.fetchMetricsTimeline(3600);
      second = result.current.fetchMetricsTimeline(900);
      await vi.waitFor(() => { expect(pending).toHaveLength(2); });
      pending[1]!({ windowSeconds: 900, sampleIntervalSeconds: 5, samples: [] });
      await second;
      pending[0]!({ windowSeconds: 3600, sampleIntervalSeconds: 5, samples: [] });
      await first;
    });
    expect(result.current.metricHistory?.windowSeconds).toBe(900);
    let third!: Promise<void>;
    await act(async () => {
      third = result.current.fetchMetricsTimeline(3600);
      await vi.waitFor(() => { expect(pending).toHaveLength(3); });
      result.current.clearHistory();
      pending[2]!({ windowSeconds: 3600, sampleIntervalSeconds: 5, samples: [] });
      await third;
    });
    expect(result.current.metricHistory).toBeNull();
  });

  it('retains exact server window bounds independently of observation coverage', async () => {
    const start = timestampFromDate(new Date('2026-01-01T00:00:00Z'));
    const end = timestampFromDate(new Date('2026-01-01T01:00:00Z'));
    mocks.protoFetch.mockResolvedValue({ windowSeconds: 3600, sampleIntervalSeconds: 5, windowStart: start, windowEnd: end, samples: [] });
    const { result } = renderHook(() => useMetricHistory(vi.fn()));
    await act(async () => { await result.current.fetchMetricsTimeline(3600); });
    expect(result.current.metricHistory?.windowStart).toBe('2026-01-01T00:00:00.000Z');
    expect(result.current.metricHistory?.windowEnd).toBe('2026-01-01T01:00:00.000Z');
  });

  it('uses cycle cadence and marks unavailable samples as gaps', async () => {
    const at = (seconds: number) => timestampFromDate(new Date(Date.UTC(2026, 0, 1, 0, 0, seconds)));
    mocks.protoFetch.mockResolvedValue({ windowSeconds: 60, sampleIntervalSeconds: 5, samples: [
      { timestamp: at(0), cpu: { state: { case: 'measured', value: 1 } } },
      { timestamp: at(10), cpu: { state: { case: 'measured', value: 2 } } },
      { timestamp: at(20), cpu: { state: { case: 'unsupportedReason', value: 'unavailable' } } },
      { timestamp: at(30), cpu: { state: { case: 'measured', value: 3 } } },
    ] });
    const { result } = renderHook(() => useMetricHistory(vi.fn()));
    await act(async () => { await result.current.fetchMetricsTimeline(); });
    expect(result.current.metricHistory?.sampleIntervalSeconds).toBe(10);
    expect(result.current.metricHistory?.cpu.map(point => point.gapBefore ?? false)).toEqual([false, false, true]);
  });

  it('uses GPU observation times and never invents missing or repeated readings', async () => {
    const at = (seconds: number) => timestampFromDate(new Date(Date.UTC(2026, 0, 1, 0, 0, seconds)));
    mocks.protoFetch.mockResolvedValue({ windowSeconds: 60, samples: [
      { timestamp: at(10), gpu: { observedAt: at(5), state: { case: 'measured', value: 4 } } },
      { timestamp: at(20), gpu: { observedAt: at(5), state: { case: 'measured', value: 4 } } },
      { timestamp: at(30), gpu: { state: { case: 'measured', value: 0 } } },
      { timestamp: at(40), gpu: { observedAt: at(35), state: { case: 'measured', value: 0 } } },
    ] });
    const { result } = renderHook(() => useMetricHistory(vi.fn()));
    await act(async () => { await result.current.fetchMetricsTimeline(); });
    expect(result.current.metricHistory?.gpu).toEqual([
      { timestamp: '2026-01-01T00:00:05.000Z', value: 4 },
      { timestamp: '2026-01-01T00:00:35.000Z', value: 0, gapBefore: true },
    ]);
  });

  it('does not treat cycles without a GPU observation as failed probes', async () => {
    const at = (seconds: number) => timestampFromDate(new Date(Date.UTC(2026, 0, 1, 0, 0, seconds)));
    const measured = (seconds: number, value: number) => ({ timestamp: at(seconds), gpu: { observedAt: at(seconds), state: { case: 'measured', value } } });
    mocks.protoFetch.mockResolvedValue({ windowSeconds: 120, samples: [
      measured(0, 1), { timestamp: at(10) }, measured(20, 2), { timestamp: at(30) }, measured(40, 3),
      { timestamp: at(50), gpu: { state: { case: 'failedError', value: 'probe failed' } } }, measured(60, 0),
      { timestamp: at(70) }, { timestamp: at(80) }, { timestamp: at(90) }, measured(100, 4),
    ] });
    const { result } = renderHook(() => useMetricHistory(vi.fn()));
    await act(async () => { await result.current.fetchMetricsTimeline(); });
    expect(result.current.metricHistory?.gpu?.map(point => point.gapBefore ?? false)).toEqual([false, false, false, true, true]);
    expect(result.current.metricHistory?.gpu?.map(point => point.value)).toEqual([1, 2, 3, 0, 4]);
  });

  it('reports timeline failures and ignores empty responses', async () => {
    const setError = vi.fn();
    const { result } = renderHook(() => useMetricHistory(setError));
    mocks.protoFetch.mockResolvedValueOnce({ samples: undefined });
    await act(async () => { await result.current.fetchMetricsTimeline(); });
    expect(result.current.metricHistory).toBeNull();
    mocks.protoFetch.mockRejectedValueOnce(new Error('timeline failed'));
    await act(async () => { await result.current.fetchMetricsTimeline(); });
    expect(setError).toHaveBeenCalledWith({ error: 'timeline failed' });
    mocks.protoFetch.mockResolvedValueOnce({ windowSeconds: 120, samples: [] });
    await act(async () => { await result.current.fetchMetricsTimeline(); });
    expect(setError).toHaveBeenLastCalledWith(null);
  });

  it('preserves measured GPU session gaps and ignores repeated timestamps', () => {
    const { result } = renderHook(() => useMetricHistory(vi.fn()));
    act(() => {
      result.current.appendGpuPoint('2026-02-02T00:00:00.000Z', 12);
      result.current.appendGpuPoint('2026-02-02T00:00:00.000Z', 12);
      result.current.appendGpuPoint('2026-02-01T23:59:55.000Z', 99);
      result.current.appendGpuPoint('2026-02-02T00:00:10.000Z', 0, true);
    });
    expect(result.current.metricHistory?.gpu).toEqual([
      { timestamp: '2026-02-02T00:00:00.000Z', value: 12 },
      { timestamp: '2026-02-02T00:00:10.000Z', value: 0, gapBefore: true },
    ]);
  });

  it('invalidates a pending timeline read without erasing the last history', async () => {
    let finish!: (value: unknown) => void;
    mocks.protoFetch.mockResolvedValueOnce({ windowSeconds: 60, samples: [{ timestamp: pointTime, cpu: { state: { case: 'measured', value: 42 } } }] });
    mocks.protoFetch.mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
    const { result } = renderHook(() => useMetricHistory(vi.fn()));
    await act(async () => { await result.current.fetchMetricsTimeline(); });
    act(() => { void result.current.fetchMetricsTimeline(); });
    await waitFor(() => { expect(finish).toBeDefined(); });
    act(() => { result.current.invalidatePending(); });
    await act(async () => { finish({ windowSeconds: 60, samples: [{ timestamp: pointTime, cpu: { state: { case: 'measured', value: 99 } } }] }); await Promise.resolve(); });
    expect(result.current.metricHistory?.cpu.map(point => point.value)).toEqual([42]);
  });

  it('counts untimed cycles without plotting their measurements', async () => {
    mocks.protoFetch.mockResolvedValue({ windowSeconds: 60, sampleIntervalSeconds: 5, samples: [
      { cpu: { state: { case: 'measured', value: 91 } } },
      { timestamp: pointTime, cpu: { state: { case: 'measured', value: 42 } } },
    ] });
    const { result } = renderHook(() => useMetricHistory(vi.fn()));
    await act(async () => { await result.current.fetchMetricsTimeline(); });
    expect(result.current.metricHistory?.missingTimestampCount).toBe(1);
    expect(result.current.metricHistory?.cpu.map(point => point.value)).toEqual([42]);
    act(() => { result.current.appendGpuPoint('2026-01-01T00:00:00Z', 3); });
    expect(result.current.metricHistory?.missingTimestampCount).toBe(1);
  });

  it('retains measured kernel history without inventing zero for unsupported or old samples', async () => {
    mocks.protoFetch.mockResolvedValue({ windowSeconds: 120, sampleIntervalSeconds: 5, samples: [
      { timestamp: pointTime, kernelMemory: { slabBytes: { state: { case: 'measured', value: 18000 } }, unreclaimableBytes: { state: { case: 'measured', value: 13000 } }, reclaimableBytes: { state: { case: 'measured', value: 0 } } } },
      { timestamp: pointTime, kernelMemory: { slabBytes: { state: { case: 'unsupportedReason', value: 'not Linux' } } } },
      { timestamp: pointTime },
    ] });
    const { result } = renderHook(() => useMetricHistory(vi.fn()));
    await act(async () => { await result.current.fetchMetricsTimeline(); });
    expect(result.current.metricHistory?.kernelSlab?.map(point => point.value)).toEqual([18000]);
    expect(result.current.metricHistory?.kernelUnreclaimable?.map(point => point.value)).toEqual([13000]);
    expect(result.current.metricHistory?.kernelReclaimable?.map(point => point.value)).toEqual([0]);
    act(() => { result.current.appendGpuPoint('now', 9); });
    expect(result.current.metricHistory?.kernelUnreclaimable?.map(point => point.value)).toEqual([13000]);
  });

  it('checks health, toggles maintenance, and surfaces server errors', async () => {
    const { result } = renderHook(() => useHealthCheck());
    await act(async () => { expect(await result.current.checkHealth()).toBe(true); });
    expect(result.current.healthStatus?.status).toBe('healthy');
    await act(async () => { await result.current.toggleMonitoring(); });
    expect(mocks.protoFetch).toHaveBeenCalledWith('/maintenance/state', expect.anything(), expect.objectContaining({ method: 'POST' }));

    mocks.protoFetch.mockResolvedValueOnce({ success: false, error: 'cannot change' });
    await act(async () => { await result.current.toggleMonitoring(); });
    await waitFor(() => { expect(result.current.healthError).toBeNull(); });
  });

  it('loads health when toggled before the first health response exists', async () => {
    const { result } = renderHook(() => useHealthCheck());
    await act(async () => { await result.current.toggleMonitoring(); });
    expect(mocks.apiFetch).toHaveBeenCalledWith('/health', expect.anything());
  });

  it('reports health and maintenance failures, including unknown error values', async () => {
    const { result } = renderHook(() => useHealthCheck());
    mocks.apiFetch.mockRejectedValueOnce('offline');
    await act(async () => { expect(await result.current.checkHealth()).toBe(false); });
    expect(result.current.healthError).toBe('Unknown error');

    mocks.apiFetch.mockResolvedValueOnce({ status: 'healthy', maintenance_state: 'inactive' });
    await act(async () => { expect(await result.current.checkHealth()).toBe(true); });
    mocks.protoFetch.mockResolvedValueOnce({ success: false });
    mocks.apiFetch.mockRejectedValueOnce(new Error('refresh failed'));
    await act(async () => { await result.current.toggleMonitoring(); });
    await waitFor(() => { expect(result.current.healthError).toBe('refresh failed'); });
  });

  it('does not publish a health response after unmount', async () => {
    let resolveHealth: ((value: unknown) => void) | undefined;
    mocks.apiFetch.mockImplementationOnce(() => new Promise(resolve => { resolveHealth = resolve; }));
    const { result, unmount } = renderHook(() => useHealthCheck());
    const pending = result.current.checkHealth();
    unmount();
    resolveHealth?.({ status: 'healthy' });
    await pending;
  });

  it('handles successful, failed, and aborted generic API calls', async () => {
    const { result } = renderHook(() => useApiCall<{ value: number }>());
    mocks.apiFetch.mockResolvedValueOnce({ value: 1 });
    await act(async () => { await result.current.execute('/ok'); });
    expect(result.current.data).toEqual({ value: 1 });
    mocks.apiFetch.mockRejectedValueOnce(new Error('bad request'));
    await act(async () => { await result.current.execute('/bad'); });
    expect(result.current.error?.error).toBe('bad request');
    mocks.apiFetch.mockRejectedValueOnce(new DOMException('cancelled', 'AbortError'));
    await act(async () => { await result.current.execute('/cancel'); });
    expect(result.current.data).toEqual({ value: 1 });
  });

  it('handles API-shaped errors and unmounted requests without publishing state', async () => {
    const { result, unmount } = renderHook(() => useApiCall<{ value: number }>());
    mocks.apiFetch.mockRejectedValueOnce({ error: 'api-shaped' });
    await act(async () => { await result.current.execute('/api-error'); });
    expect(result.current.error?.error).toBe('request failed');

    let resolveRequest: ((value: { value: number }) => void) | undefined;
    mocks.apiFetch.mockImplementationOnce(() => new Promise(resolve => { resolveRequest = resolve; }));
    const pending = result.current.execute('/pending');
    unmount();
    resolveRequest?.({ value: 2 });
    await pending;
  });
});
