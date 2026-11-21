import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useComingSoonToggle } from './useComingSoonToggle';

const mocks = vi.hoisted(() => ({ read: vi.fn(), toggle: vi.fn() }));
vi.mock('../services/waitlist.service', () => ({ fetchBranding: mocks.read, toggleComingSoonMode: mocks.toggle }));

describe('private coming soon owner configuration', () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.read.mockResolvedValue({ coming_soon_enabled: true }); mocks.toggle.mockResolvedValue(false); });
  it('reads branding from its existing owner, then changes only the confirmed state', async () => {
    const { result } = renderHook(() => useComingSoonToggle());
    expect(result.current.loading).toBe(true);
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    expect(result.current.comingSoonEnabled).toBe(true);
    await act(async () => { expect(await result.current.handleToggle()).toEqual({ success: true }); });
    expect(mocks.toggle).toHaveBeenCalledWith(true);
    expect(result.current.comingSoonEnabled).toBe(false);
    expect(mocks.read).toHaveBeenCalledTimes(1);
  });
  it('does not turn an unavailable read into an enabled toggle and supports explicit reload', async () => {
    mocks.read.mockRejectedValueOnce(new Error('offline'));
    const { result } = renderHook(() => useComingSoonToggle());
    await waitFor(() => { expect(result.current.error).toContain('unavailable'); });
    await act(async () => { expect((await result.current.handleToggle()).success).toBe(false); });
    expect(mocks.toggle).not.toHaveBeenCalled();
    act(() => { result.current.reload(); });
    await waitFor(() => { expect(result.current.comingSoonEnabled).toBe(true); });
    expect(result.current.error).toBeUndefined();
  });
  it.each([new Error('service unavailable'), 'unknown'])('retains safe failure feedback after an unconfirmed write', async error => {
    mocks.toggle.mockRejectedValueOnce(error);
    const { result } = renderHook(() => useComingSoonToggle());
    await waitFor(() => { expect(result.current.loading).toBe(false); });
    await act(async () => { expect((await result.current.handleToggle()).success).toBe(false); });
    expect(result.current.toggling).toBe(false);
    expect(result.current.error).toBe(error instanceof Error ? error.message : 'Failed to toggle coming soon mode');
    await act(async () => { await result.current.handleToggle(); });
    expect(mocks.toggle).toHaveBeenCalledTimes(1);
  });
  it('ignores late private reads on unmount', async () => {
    let finish!: (value: { coming_soon_enabled: boolean }) => void;
    mocks.read.mockReturnValueOnce(new Promise(resolve => { finish = resolve; }));
    const { result, unmount } = renderHook(() => useComingSoonToggle());
    unmount(); await act(async () => { finish({ coming_soon_enabled: true }); await Promise.resolve(); });
    expect(result.current.loading).toBe(true);
    expect(mocks.toggle).not.toHaveBeenCalled();
  });
});
