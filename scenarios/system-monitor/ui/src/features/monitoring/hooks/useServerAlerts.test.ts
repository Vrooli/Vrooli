import { create } from '@bufbuild/protobuf';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { renderHook, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { AlertSchema, ListAlertsResponseSchema } from '@vrooli/proto-types/system-monitor/v1/alerts/alerts_pb';
import { useServerAlerts } from './useServerAlerts';

const fetchServerAlerts = vi.fn();

vi.mock('../serverAlerts', async importOriginal => ({
  ...(await importOriginal<typeof import('../serverAlerts')>()),
  fetchServerAlerts: (signal?: AbortSignal) => fetchServerAlerts(signal),
}));

// Polling cadence is usePolling's concern; these tests cover the hook's own
// load, error and cancellation behavior.
vi.mock('../../../shared/hooks/usePolling', () => ({ usePolling: () => undefined }));

const activeResponse = create(ListAlertsResponseSchema, {
  monitoringActive: true,
  evaluatedAt: timestampFromDate(new Date('2026-10-08T02:30:00Z')),
  sustainSeconds: 300,
  alerts: [create(AlertSchema, {
    id: 'cond_orphaned_agents_host',
    type: 'condition',
    severity: 'warning',
    ruleId: 'orphaned_agents',
    message: '5 orphaned session(s) older than 1h0m0s.',
    openedAt: timestampFromDate(new Date('2026-10-08T02:31:25Z')),
  })],
});

afterEach(() => {
  fetchServerAlerts.mockReset();
});

describe('useServerAlerts', () => {
  it('does not read alerts while disabled', () => {
    const { result } = renderHook(() => useServerAlerts(false));

    expect(fetchServerAlerts).not.toHaveBeenCalled();
    expect(result.current.feed.status).toBe('loading');
    expect(result.current.alerts).toEqual([]);
  });

  it('shows the server alert feed once it loads', async () => {
    fetchServerAlerts.mockResolvedValue(activeResponse);

    const { result } = renderHook(() => useServerAlerts(true));

    await waitFor(() => expect(result.current.feed.status).toBe('ready'));
    expect(result.current.alerts.map(alert => alert.id)).toEqual(['cond_orphaned_agents_host']);
  });

  it('keeps the last alerts on screen and marks the feed errored when a read fails', async () => {
    fetchServerAlerts.mockResolvedValueOnce(activeResponse);
    const { result, rerender } = renderHook(({ enabled }) => useServerAlerts(enabled), { initialProps: { enabled: true } });
    await waitFor(() => expect(result.current.feed.status).toBe('ready'));

    fetchServerAlerts.mockRejectedValueOnce(new Error('API unreachable'));
    rerender({ enabled: false });
    rerender({ enabled: true });

    await waitFor(() => expect(result.current.feed.status).toBe('error'));
    expect(result.current.feed.error).toBe('API unreachable');
    expect(result.current.alerts.map(alert => alert.id)).toEqual(['cond_orphaned_agents_host']);
  });

  it('reports a generic reason when the failure is not an Error', async () => {
    fetchServerAlerts.mockRejectedValue('connection reset');

    const { result } = renderHook(() => useServerAlerts(true));

    await waitFor(() => expect(result.current.feed.status).toBe('error'));
    expect(result.current.feed.error).toBe('Alert state is unavailable');
  });

  it('ignores a read that completes after the hook unmounts', async () => {
    let resolveRead: (value: typeof activeResponse) => void = () => undefined;
    fetchServerAlerts.mockImplementation((signal?: AbortSignal) => new Promise(resolve => {
      resolveRead = resolve;
      expect(signal).toBeDefined();
    }));

    const { result, unmount } = renderHook(() => useServerAlerts(true));
    await waitFor(() => expect(fetchServerAlerts).toHaveBeenCalledTimes(1));
    const lastSignal = fetchServerAlerts.mock.calls[0][0] as AbortSignal;
    unmount();
    resolveRead(activeResponse);

    expect(lastSignal.aborted).toBe(true);
    expect(result.current.feed.status).toBe('loading');
  });

  it('ignores a failure from a read that was cancelled', async () => {
    let rejectRead: (reason: unknown) => void = () => undefined;
    fetchServerAlerts.mockImplementation(() => new Promise((_resolve, reject) => {
      rejectRead = reject;
    }));

    const { result, unmount } = renderHook(() => useServerAlerts(true));
    await waitFor(() => expect(fetchServerAlerts).toHaveBeenCalledTimes(1));
    unmount();
    rejectRead(new Error('aborted'));

    await Promise.resolve();
    expect(result.current.feed.status).toBe('loading');
  });
});
