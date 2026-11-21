import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useClipboard } from './useClipboard';

const pendingWrite = () => {
  let resolve!: () => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<void>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
};

describe('clipboard feedback', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); });

  it('keeps the latest copy visible for its full feedback interval', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal('navigator', { clipboard: { writeText } });
    const { result, unmount } = renderHook(() => useClipboard(2000));
    await act(async () => { expect(await result.current.copy('first')).toBe(true); });
    act(() => vi.advanceTimersByTime(1500));
    await act(async () => { await result.current.copy('second'); });
    act(() => vi.advanceTimersByTime(500));
    expect(result.current.isCopied).toBe(true);
    act(() => vi.advanceTimersByTime(1500));
    expect(result.current.isCopied).toBe(false);
    expect(writeText.mock.calls).toEqual([['first'], ['second']]);
    unmount();
    expect(vi.getTimerCount()).toBe(0);
  });

  it('does not let an older failure replace a newer successful copy', async () => {
    const first = pendingWrite();
    const writeText = vi.fn().mockReturnValueOnce(first.promise).mockResolvedValueOnce(undefined);
    vi.stubGlobal('navigator', { clipboard: { writeText } });
    const { result, unmount } = renderHook(() => useClipboard());
    let older!: Promise<boolean>;
    act(() => { older = result.current.copy('older'); });
    await act(async () => { await result.current.copy('newer'); });
    await act(async () => { first.reject(new Error('permission denied')); await older; });
    expect(result.current.isCopied).toBe(true);
    expect(result.current.error).toBeNull();
    unmount();
  });

  it('does not schedule feedback after a pending write outlives its component', async () => {
    const write = pendingWrite();
    vi.stubGlobal('navigator', { clipboard: { writeText: () => write.promise } });
    const { result, unmount } = renderHook(() => useClipboard());
    let copying!: Promise<boolean>;
    act(() => { copying = result.current.copy('path'); });
    unmount();
    await act(async () => { write.resolve(); await copying; });
    expect(vi.getTimerCount()).toBe(0);
  });

  it.each(['unavailable', 'denied'])('reports %s clipboard access without claiming success', async (mode) => {
    vi.stubGlobal('navigator', mode === 'unavailable' ? {} : {
      clipboard: { writeText: vi.fn().mockRejectedValue(new Error('denied')) },
    });
    const { result, unmount } = renderHook(() => useClipboard());
    await act(async () => { expect(await result.current.copy('path')).toBe(false); });
    expect(result.current.isCopied).toBe(false);
    expect(result.current.error).toMatch(/manually/);
    expect(vi.getTimerCount()).toBe(0);
    unmount();
  });
});
