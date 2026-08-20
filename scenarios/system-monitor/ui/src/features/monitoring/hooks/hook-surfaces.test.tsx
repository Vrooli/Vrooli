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
        { timestamp: pointTime, cpuUsage: 1, memoryUsage: 2, tcpConnections: 3, gpuUsage: 4 },
        { timestamp: undefined, cpuUsage: 5, memoryUsage: 6, tcpConnections: 7, gpuUsage: Number.NaN },
      ],
    });
    const setError = vi.fn();
    const { result } = renderHook(() => useMetricHistory(setError));

    await act(async () => { await result.current.fetchMetricsTimeline(120); });
    expect(result.current.metricHistory?.cpu).toHaveLength(2);
    expect(result.current.metricHistory?.gpu).toHaveLength(1);

    act(() => {
      result.current.appendGpuPoint('now', 9);
      result.current.appendDiskPoints('now', 2, Number.NaN);
      result.current.appendDiskUsagePoint('now', 80);
    });
    expect(result.current.metricHistory?.gpu?.at(-1)?.value).toBe(9);
    expect(result.current.metricHistory?.diskRead?.at(-1)?.value).toBe(2);
    expect(result.current.metricHistory?.diskWrite).toBeUndefined();
    expect(result.current.metricHistory?.diskUsage?.at(-1)?.value).toBe(80);
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
