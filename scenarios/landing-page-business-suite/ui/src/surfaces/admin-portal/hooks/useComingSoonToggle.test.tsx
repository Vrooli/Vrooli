import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useComingSoonToggle } from './useComingSoonToggle';

const { mockRefresh, mockToggleComingSoonMode, mockUseLandingVariant } = vi.hoisted(() => ({
  mockRefresh: vi.fn(),
  mockToggleComingSoonMode: vi.fn(),
  mockUseLandingVariant: vi.fn(),
}));

vi.mock('../../../app/providers/useLandingVariant', () => ({
  useLandingVariant: mockUseLandingVariant,
}));

vi.mock('../services/waitlist.service', () => ({
  toggleComingSoonMode: mockToggleComingSoonMode,
}));

describe('useComingSoonToggle', () => {
  beforeEach(() => {
    mockRefresh.mockResolvedValue(undefined);
    mockToggleComingSoonMode.mockResolvedValue(undefined);
    mockUseLandingVariant.mockReturnValue({
      config: { branding: { coming_soon_enabled: true } },
      refresh: mockRefresh,
    });
  });

  it('toggles the configured state and refreshes the landing configuration', async () => {
    const { result } = renderHook(() => useComingSoonToggle());

    let response: Awaited<ReturnType<typeof result.current.handleToggle>> | undefined;
    await act(async () => {
      response = await result.current.handleToggle();
    });

    expect(response).toEqual({ success: true });
    expect(mockToggleComingSoonMode).toHaveBeenCalledWith(true);
    expect(mockRefresh).toHaveBeenCalledTimes(1);
    expect(result.current.toggling).toBe(false);
  });

  it('returns a safe failure result when toggling fails', async () => {
    mockToggleComingSoonMode.mockRejectedValueOnce(new Error('service unavailable'));
    const { result } = renderHook(() => useComingSoonToggle());

    let response: Awaited<ReturnType<typeof result.current.handleToggle>> | undefined;
    await act(async () => {
      response = await result.current.handleToggle();
    });

    await waitFor(() => {
      expect(result.current.toggling).toBe(false);
    });
    expect(response).toEqual({ success: false, message: 'service unavailable' });
  });

  it('returns the generic failure message when a non-Error value is thrown', async () => {
    mockToggleComingSoonMode.mockRejectedValueOnce('service unavailable');
    const { result } = renderHook(() => useComingSoonToggle());

    let response: Awaited<ReturnType<typeof result.current.handleToggle>> | undefined;
    await act(async () => {
      response = await result.current.handleToggle();
    });

    expect(response).toEqual({ success: false, message: 'Failed to toggle coming soon mode' });
  });

  it('treats an absent branding configuration as coming soon disabled', () => {
    mockUseLandingVariant.mockReturnValue({ config: null, refresh: mockRefresh });

    const { result } = renderHook(() => useComingSoonToggle());

    expect(result.current.comingSoonEnabled).toBe(false);
  });
});
