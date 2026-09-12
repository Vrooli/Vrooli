import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';

const { mockGetEntitlements } = vi.hoisted(() => ({ mockGetEntitlements: vi.fn() }));
vi.mock('../api', () => ({ getEntitlements: mockGetEntitlements }));

import { useEntitlements } from './useEntitlements';

beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.clear();
  mockGetEntitlements.mockResolvedValue({ entitlements: ['pro'] });
});

describe('useEntitlements', () => {
  it('starts empty and does not fetch without an email', async () => {
    const { result } = renderHook(() => useEntitlements());
    expect(result.current.email).toBe('');
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(mockGetEntitlements).not.toHaveBeenCalled();
    expect(result.current.entitlements).toBeNull();
  });

  it('persists the email and fetches entitlements when set', async () => {
    const { result } = renderHook(() => useEntitlements());
    act(() => result.current.setEmail('user@example.com'));

    expect(window.localStorage.getItem('landing_entitlement_email')).toBe('user@example.com');
    await waitFor(() => expect(mockGetEntitlements).toHaveBeenCalled());
    await waitFor(() => expect(result.current.entitlements).toEqual({ entitlements: ['pro'] }));
  });

  it('hydrates the initial email from localStorage', async () => {
    window.localStorage.setItem('landing_entitlement_email', 'stored@example.com');
    const { result } = renderHook(() => useEntitlements());
    expect(result.current.email).toBe('stored@example.com');
    await waitFor(() => expect(mockGetEntitlements).toHaveBeenCalled());
  });

  it('surfaces an error when the fetch fails', async () => {
    mockGetEntitlements.mockRejectedValue(new Error('denied'));
    const { result } = renderHook(() => useEntitlements());
    act(() => result.current.setEmail('user@example.com'));
    await waitFor(() => expect(result.current.error).toBe('denied'));
    expect(result.current.entitlements).toBeNull();
  });

  it('clears the stored email and entitlements when emptied', async () => {
    window.localStorage.setItem('landing_entitlement_email', 'stored@example.com');
    const { result } = renderHook(() => useEntitlements());
    await waitFor(() => expect(mockGetEntitlements).toHaveBeenCalled());

    act(() => result.current.setEmail(''));
    expect(window.localStorage.getItem('landing_entitlement_email')).toBeNull();
    await waitFor(() => expect(result.current.entitlements).toBeNull());
  });
});
