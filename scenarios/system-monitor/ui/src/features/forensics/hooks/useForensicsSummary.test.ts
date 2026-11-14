import { renderHook, act, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useForensicsSummary } from './useForensicsSummary';

const mocks = vi.hoisted(() => ({ fetchForensicsSummary: vi.fn(), usePolling: vi.fn() }));
vi.mock('../api', () => ({ fetchForensicsSummary: mocks.fetchForensicsSummary }));
vi.mock('../../../shared/hooks/usePolling', () => ({ usePolling: mocks.usePolling }));
vi.mock('../../../shared/api/apiFetch', () => ({
  extractErrorMessage: (error: unknown, fallback: string) => error instanceof Error ? error.message : fallback,
}));

const summary = { generatedAt: 'now', pstore: {}, bootHistory: {}, mce: {}, autoheal: {} };

describe('useForensicsSummary', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.fetchForensicsSummary.mockResolvedValue(summary);
  });

  it('loads and refreshes a summary, and wires the polling callback', async () => {
    const { result } = renderHook(() => useForensicsSummary(1000));
    await waitFor(() => { expect(result.current.summary).toEqual(summary); });
    expect(result.current.isLoading).toBe(false);
    expect(mocks.usePolling).toHaveBeenCalledWith(expect.any(Function), 1000, true);
    await act(async () => { await result.current.refresh(); });
    expect(mocks.fetchForensicsSummary).toHaveBeenCalledTimes(2);
  });

  it('surfaces ordinary failures and ignores abort failures', async () => {
    mocks.fetchForensicsSummary.mockRejectedValueOnce(new Error('forensics unavailable'));
    const { result } = renderHook(() => useForensicsSummary());
    await waitFor(() => { expect(result.current.error).toBe('forensics unavailable'); });

    mocks.fetchForensicsSummary.mockRejectedValueOnce(new DOMException('cancelled', 'AbortError'));
    await act(async () => { await result.current.refresh(); });
    expect(result.current.error).toBe('forensics unavailable');
  });

  it('does not publish a result after the request has been aborted', async () => {
    let resolveRequest: ((value: typeof summary) => void) | undefined;
    mocks.fetchForensicsSummary.mockImplementationOnce(() => new Promise(resolve => { resolveRequest = resolve; }));
    const { result, unmount } = renderHook(() => useForensicsSummary());
    unmount();
    resolveRequest?.(summary);
    await act(async () => { await Promise.resolve(); });
    expect(result.current.summary).toBeNull();
  });
});
