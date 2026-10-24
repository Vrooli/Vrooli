import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { useRecordedVideoStatus } from './useRecordedVideoStatus';
import type { ExportApiClient, RecordedVideoStatus } from '../api/exportClient';
function deferred() {
  let resolve!: (value: RecordedVideoStatus) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<RecordedVideoStatus>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
const available = (id: string): RecordedVideoStatus => ({ available: true, count: 1, videos: [{ id, url: `/${id}.mp4` }] });
const client = (fetchRecordedVideoStatus: ExportApiClient['fetchRecordedVideoStatus']): ExportApiClient => ({ fetchRecordedVideoStatus, executeExport: vi.fn() });
describe('recorded artifacts belong to the current execution and request', () => {
  it('clears the old execution URL while the replacement execution is loading', async () => {
    const b = deferred(); const fetch = vi.fn().mockResolvedValueOnce(available('a')).mockReturnValueOnce(b.promise);
    const apiClient = client(fetch);
    const { result, rerender } = renderHook(({ id }) => useRecordedVideoStatus({ executionId: id, apiClient }), { initialProps: { id: 'a' } });
    await act(async () => {}); expect(result.current.url).toBe('/a.mp4');
    rerender({ id: 'b' });
    expect(result.current.url).toBeNull(); expect(result.current.available).toBe(false); expect(result.current.count).toBe(0); expect(result.current.loading).toBe(true);
    await act(async () => b.resolve(available('b')));
    expect(result.current.url).toBe('/b.mp4');
  });
  it.each(['resolve', 'reject'] as const)('ignores an old manual refresh that later %ss and leaves current loading intact', async outcome => {
    const refresh = deferred(); const b = deferred();
    const fetch = vi.fn().mockResolvedValueOnce(available('a')).mockReturnValueOnce(refresh.promise).mockReturnValueOnce(b.promise);
    const apiClient = client(fetch);
    const { result, rerender } = renderHook(({ id }) => useRecordedVideoStatus({ executionId: id, apiClient }), { initialProps: { id: 'a' } });
    await act(async () => {});
    act(() => result.current.refresh()); rerender({ id: 'b' });
    await act(async () => outcome === 'resolve' ? refresh.resolve(available('old-a')) : refresh.reject(Error('old failure')));
    expect(result.current.loading).toBe(true); expect(result.current.url).toBeNull(); expect(result.current.error).toBeNull();
    await act(async () => b.resolve(available('b'))); expect(result.current.url).toBe('/b.mp4');
  });
  it('ignores an aborted prior effect finalizer even when the transport rejects after B starts', async () => {
    const a = deferred(); const b = deferred(); const fetch = vi.fn().mockReturnValueOnce(a.promise).mockReturnValueOnce(b.promise);
    const apiClient = client(fetch);
    const { result, rerender } = renderHook(({ id }) => useRecordedVideoStatus({ executionId: id, apiClient }), { initialProps: { id: 'a' } });
    rerender({ id: 'b' });
    await act(async () => a.reject(new DOMException('aborted', 'AbortError')));
    expect(result.current.loading).toBe(true);
    await act(async () => b.resolve(available('b'))); expect(result.current.loading).toBe(false);
  });
  it('lets only the latest refresh commit, aborts retired requests, and exposes current errors', async () => {
    const first = deferred(); const latest = deferred();
    const fetch = vi.fn().mockResolvedValueOnce(available('a')).mockReturnValueOnce(first.promise).mockReturnValueOnce(latest.promise);
    const apiClient = client(fetch); const { result, unmount } = renderHook(() => useRecordedVideoStatus({ executionId: 'a', apiClient }));
    await act(async () => {});
    act(() => result.current.refresh()); act(() => result.current.refresh());
    expect(fetch.mock.calls[1]?.[1]?.aborted).toBe(true);
    await act(async () => latest.reject(Error('current failure')));
    expect(result.current.error).toBe('current failure'); expect(result.current.url).toBeNull();
    await act(async () => first.resolve(available('old-a')));
    expect(result.current.error).toBe('current failure'); expect(result.current.url).toBeNull();
    unmount(); expect(fetch.mock.calls[2]?.[1]?.aborted).toBe(true);
  });
  it('does not let a retained old refresh callback cancel the new execution request', async () => {
    const a = deferred(); const b = deferred(); const fetch = vi.fn().mockReturnValueOnce(a.promise).mockReturnValueOnce(b.promise);
    const apiClient = client(fetch);
    const { result, rerender } = renderHook(({ id }) => useRecordedVideoStatus({ executionId: id, apiClient }), { initialProps: { id: 'a' } });
    const oldRefresh = result.current.refresh;
    rerender({ id: 'b' }); act(() => oldRefresh());
    expect(fetch).toHaveBeenCalledTimes(2); expect(fetch.mock.calls[1]?.[1]?.aborted).toBe(false);
    await act(async () => b.resolve(available('b'))); expect(result.current.url).toBe('/b.mp4');
    await act(async () => a.resolve(available('a'))); expect(result.current.url).toBe('/b.mp4');
  });
  it('skips empty identities and suppresses late unmounted responses', async () => {
    const pending = deferred(); const fetch = vi.fn().mockReturnValue(pending.promise); const apiClient = client(fetch);
    const { result, rerender, unmount } = renderHook(({ id }) => useRecordedVideoStatus({ executionId: id, apiClient }), { initialProps: { id: '' } });
    expect(fetch).not.toHaveBeenCalled(); expect(result.current.loading).toBe(false);
    rerender({ id: 'a' }); unmount();
    expect(fetch.mock.calls[0]?.[1]?.aborted).toBe(true);
    await act(async () => pending.resolve(available('a')));
  });
});
