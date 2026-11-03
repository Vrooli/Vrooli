import { renderHook, act, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useLogQuery } from './useLogQuery';

const mocks = vi.hoisted(() => ({ fetchLogs: vi.fn() }));
vi.mock('../api', () => ({ fetchLogs: mocks.fetchLogs }));
vi.mock('../../../shared/api/apiFetch', () => ({
  extractErrorMessage: (error: unknown, fallback: string) => error instanceof Error ? error.message : fallback,
}));

describe('useLogQuery', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.fetchLogs.mockResolvedValue({ entries: [], available: true });
  });

  it('keeps free-form time edits until the shared range itself changes', async () => {
    const { result, rerender } = renderHook(
      ({ since, until }) => useLogQuery({ since, until }),
      { initialProps: { since: '1h ago', until: 'now' } },
    );
    await waitFor(() => { expect(result.current.isLoading).toBe(false); });
    act(() => { result.current.setFilter({ since: 'yesterday', until: '10m ago' }); });
    expect(result.current.filters.since).toBe('yesterday');
    expect(result.current.filters.until).toBe('10m ago');
    rerender({ since: '2h ago', until: 'now' });
    await waitFor(() => { expect(result.current.filters.since).toBe('2h ago'); });
    expect(result.current.filters.until).toBe('now');
  });

  it('does not commit a pending response after the reader pauses', async () => {
    let finish!: (value: unknown) => void;
    mocks.fetchLogs.mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
    const { result } = renderHook(() => useLogQuery());
    await waitFor(() => { expect(result.current.isLoading).toBe(true); });
    act(() => { result.current.cancel(); });
    await act(async () => {
      finish({ entries: [{ message: 'late result' }], available: true });
      await Promise.resolve();
    });
    expect(result.current.entries).toEqual([]);
    expect(result.current.isLoading).toBe(false);
  });

  it('does not replace a newer filter result with an old response', async () => {
    let finishOld!: (value: unknown) => void;
    mocks.fetchLogs.mockImplementationOnce(() => new Promise(resolve => { finishOld = resolve; }));
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ message: 'new query' }], available: true });
    const { result } = renderHook(() => useLogQuery());
    act(() => { result.current.setFilter({ grep: 'new' }); });
    await waitFor(() => { expect(result.current.entries[0]?.message).toBe('new query'); });
    await act(async () => { finishOld({ entries: [{ message: 'stale' }], available: true }); await Promise.resolve(); });
    expect(result.current.entries[0].message).toBe('new query');
    expect(result.current.filters.grep).toBe('new');
  });

  it('loads logs, applies range/filter changes, paginates, and resets', async () => {
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ id: '1' }], nextCursor: 'cursor-1', available: true });
    const { result } = renderHook(() => useLogQuery({ since: '1h ago', until: 'now' }));
    await waitFor(() => { expect(result.current.entries).toHaveLength(1); });
    expect(result.current.filters.since).toBe('1h ago');
    act(() => { result.current.nextPage(); });
    await waitFor(() => { expect(result.current.currentCursor).toBe('cursor-1'); });
    expect(mocks.fetchLogs.mock.calls.at(-1)?.[0].direction).toBe('reverse');
    act(() => { result.current.prevPage(); });
    await waitFor(() => { expect(result.current.currentCursor).toBeUndefined(); });
    act(() => { result.current.setFilter({ grep: 'oom', limit: 99999 }); });
    expect(result.current.filters.limit).toBeLessThan(99999);
    act(() => { result.current.resetFilters(); });
    expect(result.current.filters.since).toBe('1h ago');
  });

  it('retains the loaded page and cursor when an older page fails', async () => {
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ message: 'retained' }], nextCursor: 'older', available: true });
    const { result } = renderHook(() => useLogQuery());
    await waitFor(() => { expect(result.current.entries).toHaveLength(1); });
    mocks.fetchLogs.mockResolvedValueOnce({ available: false, reason: 'cursor expired' });
    act(() => { result.current.nextPage(); });
    await waitFor(() => { expect(result.current.error).toBe('cursor expired'); });
    expect(result.current.entries[0].message).toBe('retained');
    expect(result.current.currentCursor).toBeUndefined();
    expect(result.current.nextCursor).toBe('older');
    expect(result.current.cursorStack).toEqual([]);
  });

  it('retries the failed older cursor while retaining the current page until success', async () => {
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ message: 'retained' }], nextCursor: 'older', available: true });
    const { result } = renderHook(() => useLogQuery());
    await waitFor(() => { expect(result.current.nextCursor).toBe('older'); });
    mocks.fetchLogs.mockResolvedValueOnce({ available: false, error: 'older failed' });
    act(() => { result.current.nextPage(); });
    await waitFor(() => { expect(result.current.error).toBe('older failed'); });
    expect(result.current.entries[0].message).toBe('retained');
    expect(result.current.retryCursorStack).toEqual(['older']);
    await act(async () => { await result.current.refreshBackground(); });
    expect(mocks.fetchLogs).toHaveBeenCalledTimes(2);
    expect(result.current.error).toBe('older failed');
    let finish!: (value: unknown) => void;
    mocks.fetchLogs.mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
    act(() => { result.current.retryFailed(); });
    expect(mocks.fetchLogs.mock.calls.at(-1)?.[0].cursor).toBe('older');
    expect(result.current.entries[0].message).toBe('retained');
    await act(async () => { finish({ entries: [{ message: 'older result' }], available: true }); await Promise.resolve(); });
    await waitFor(() => { expect(result.current.currentCursor).toBe('older'); });
    expect(result.current.entries[0].message).toBe('older result');
    expect(result.current.retryCursorStack).toBeUndefined();
  });

  it('keeps an unavailable state until an explicit refresh recovers it', async () => {
    mocks.fetchLogs.mockResolvedValueOnce({ available: false, reason: 'unavailable' });
    const { result } = renderHook(() => useLogQuery());
    await waitFor(() => { expect(result.current.available).toBe(false); });
    await act(async () => { await result.current.refreshBackground(); });
    expect(mocks.fetchLogs).toHaveBeenCalledTimes(1);
    expect(result.current.reason).toBe('unavailable');
    await act(async () => { await result.current.refresh(); });
    expect(result.current.available).toBe(true);
  });

  it('discards a failed page cursor when the query filters change', async () => {
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ message: 'head' }], nextCursor: 'older', available: true });
    const { result } = renderHook(() => useLogQuery());
    await waitFor(() => { expect(result.current.nextCursor).toBe('older'); });
    mocks.fetchLogs.mockResolvedValueOnce({ available: false, error: 'older failed' });
    act(() => { result.current.nextPage(); });
    await waitFor(() => { expect(result.current.retryCursorStack).toEqual(['older']); });
    act(() => { result.current.setFilter({ grep: 'new query' }); });
    expect(result.current.retryCursorStack).toBeUndefined();
    await waitFor(() => { expect(mocks.fetchLogs.mock.calls.at(-1)?.[0].filters.grep).toBe('new query'); });
    expect(mocks.fetchLogs.mock.calls.at(-1)?.[0].cursor).toBeUndefined();
  });

  it('reports a journal query error without replacing the last page', async () => {
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ message: 'retained' }], available: true });
    const { result } = renderHook(() => useLogQuery());
    await waitFor(() => { expect(result.current.entries).toHaveLength(1); });
    mocks.fetchLogs.mockResolvedValueOnce({ available: false, error: 'invalid expression' });
    await act(async () => { await result.current.refresh(); });
    expect(result.current.error).toBe('invalid expression');
    expect(result.current.entries[0].message).toBe('retained');
  });

  it('retains the prior page identity after a changed query fails and can retry the draft', async () => {
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ message: 'head' }], nextCursor: 'older', available: true });
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ message: 'old page' }], available: true });
    const { result } = renderHook(() => useLogQuery());
    await waitFor(() => { expect(result.current.nextCursor).toBe('older'); });
    act(() => { result.current.nextPage(); });
    await waitFor(() => { expect(result.current.currentCursor).toBe('older'); });
    mocks.fetchLogs.mockResolvedValueOnce({ available: false, error: 'query rejected' });
    act(() => { result.current.setFilter({ grep: 'new filter' }); });
    await waitFor(() => { expect(result.current.error).toBe('query rejected'); });
    expect(result.current.filters.grep).toBe('new filter');
    expect(result.current.loadedPage?.currentCursor).toBe('older');
    expect(result.current.entries[0].message).toBe('old page');
    act(() => { result.current.restoreLoaded(); });
    expect(result.current.currentCursor).toBe('older');
    expect(result.current.filters.grep).toBe('');
    expect(result.current.retryFilters?.grep).toBe('new filter');
    act(() => { result.current.retryPreviousQuery(); });
    expect(result.current.filters.grep).toBe('new filter');
  });

  it('keeps the current page when return to latest fails', async () => {
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ message: 'head' }], nextCursor: 'older', available: true });
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ message: 'older page' }], available: true });
    const { result } = renderHook(() => useLogQuery());
    await waitFor(() => { expect(result.current.nextCursor).toBe('older'); });
    act(() => { result.current.nextPage(); });
    await waitFor(() => { expect(result.current.currentCursor).toBe('older'); });
    mocks.fetchLogs.mockResolvedValueOnce({ available: false, reason: 'unavailable' });
    let success = true;
    await act(async () => { success = await result.current.returnToLatest(); });
    expect(success).toBe(false);
    expect(result.current.currentCursor).toBe('older');
    expect(result.current.entries[0].message).toBe('older page');
  });

  it('does not cancel an explicit page request when scrolling pauses background updates', async () => {
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ message: 'head' }], nextCursor: 'older', available: true });
    let finish!: (value: unknown) => void;
    mocks.fetchLogs.mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
    const { result } = renderHook(() => useLogQuery());
    await waitFor(() => { expect(result.current.nextCursor).toBe('older'); });
    act(() => { result.current.nextPage(); result.current.cancelBackground(); });
    await act(async () => { finish({ entries: [{ message: 'older' }], available: true }); await Promise.resolve(); });
    await waitFor(() => { expect(result.current.currentCursor).toBe('older'); });
    expect(result.current.entries[0].message).toBe('older');
  });

  it('does not let a polling tick supersede an in-flight page request', async () => {
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ message: 'head' }], nextCursor: 'older', available: true });
    let finish!: (value: unknown) => void;
    mocks.fetchLogs.mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
    const { result } = renderHook(() => useLogQuery());
    await waitFor(() => { expect(result.current.nextCursor).toBe('older'); });
    act(() => { result.current.nextPage(); });
    await act(async () => { await result.current.refreshBackground(); });
    expect(mocks.fetchLogs).toHaveBeenCalledTimes(2);
    await act(async () => { finish({ entries: [{ message: 'older' }], available: true }); await Promise.resolve(); });
    await waitFor(() => { expect(result.current.currentCursor).toBe('older'); });
  });

  it('handles unavailable responses, invalid limits, errors, and an empty page stack', async () => {
    mocks.fetchLogs.mockResolvedValueOnce({ entries: undefined, available: false, reason: 'journald unavailable' });
    const { result } = renderHook(() => useLogQuery());
    await waitFor(() => { expect(result.current.available).toBe(false); });
    expect(result.current.entries).toEqual([]);
    expect(result.current.reason).toBe('journald unavailable');
    act(() => { result.current.prevPage(); result.current.nextPage(); result.current.setFilter({ limit: 0 }); });
    expect(result.current.filters.limit).toBeGreaterThan(0);
    mocks.fetchLogs.mockRejectedValueOnce(new Error('logs failed'));
    await act(async () => { await result.current.refresh(); });
    expect(result.current.error).toBe('logs failed');
  });

  it('ignores abort errors', async () => {
    mocks.fetchLogs.mockRejectedValueOnce(new DOMException('cancelled', 'AbortError'));
    const { result } = renderHook(() => useLogQuery());
    await act(async () => { await result.current.refresh(); });
    expect(result.current.error).toBeNull();
  });
});
