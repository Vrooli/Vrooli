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

  it('loads logs, applies range/filter changes, paginates, and resets', async () => {
    mocks.fetchLogs.mockResolvedValueOnce({ entries: [{ id: '1' }], nextCursor: 'cursor-1', available: true });
    const { result } = renderHook(() => useLogQuery({ since: '1h ago', until: 'now' }));
    await waitFor(() => { expect(result.current.entries).toHaveLength(1); });
    expect(result.current.filters.since).toBe('1h ago');
    act(() => { result.current.nextPage(); });
    expect(result.current.currentCursor).toBe('cursor-1');
    act(() => { result.current.prevPage(); });
    expect(result.current.currentCursor).toBeUndefined();
    act(() => { result.current.setFilter({ grep: 'oom', limit: 99999 }); });
    expect(result.current.filters.limit).toBeLessThan(99999);
    act(() => { result.current.resetFilters(); });
    expect(result.current.filters.since).toBe('1h ago');
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
