import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useEntitlements } from './useEntitlements';
import * as api from '../api';

vi.mock('../api', async () => {
  const actual = await vi.importActual<typeof import('../api')>('../api');
  return { ...actual, getEntitlements: vi.fn() };
});

const getEntitlements = vi.mocked(api.getEntitlements);

describe('useEntitlements', () => {
  beforeEach(() => {
    getEntitlements.mockReset();
    window.localStorage.clear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('starts empty without making an entitlement request', () => {
    const { result } = renderHook(() => useEntitlements());

    expect(result.current.email).toBe('');
    expect(result.current.entitlements).toBeNull();
    expect(result.current.loading).toBe(false);
    expect(result.current.error).toBeNull();
    expect(getEntitlements).not.toHaveBeenCalled();
  });

  it('persists an email and refreshes successful entitlement data', async () => {
    const payload = { status: 'active', plan_tier: 'pro' };
    getEntitlements.mockResolvedValue(payload);
    const { result } = renderHook(() => useEntitlements());

    act(() => result.current.setEmail(' buyer@example.com '));
    await act(async () => { await result.current.refresh(); });

    await waitFor(() => expect(result.current.entitlements).toEqual(payload));
    expect(result.current.email).toBe(' buyer@example.com ');
    expect(window.localStorage.getItem('landing_entitlement_email')).toBe(' buyer@example.com ');
    expect(result.current.error).toBeNull();
  });

  it('clears persisted access when the email is cleared', async () => {
    const { result } = renderHook(() => useEntitlements());

    act(() => result.current.setEmail('buyer@example.com'));
    await waitFor(() => expect(result.current.email).toBe('buyer@example.com'));
    act(() => result.current.setEmail(''));
    await waitFor(() => expect(result.current.email).toBe(''));

    expect(result.current.entitlements).toBeNull();
    expect(result.current.error).toBeNull();
    expect(window.localStorage.getItem('landing_entitlement_email')).toBeNull();
    expect(getEntitlements).toHaveBeenCalledTimes(1);
  });

  it('exposes safe messages for Error and non-Error request failures', async () => {
    const { result } = renderHook(() => useEntitlements());
    act(() => result.current.setEmail('buyer@example.com'));

    getEntitlements.mockRejectedValueOnce(new Error('Entitlements unavailable'));
    await act(async () => { await result.current.refresh(); });
    expect(result.current.error).toBe('Entitlements unavailable');
    expect(result.current.entitlements).toBeNull();

    getEntitlements.mockRejectedValueOnce('unexpected failure');
    await act(async () => { await result.current.refresh(); });
    expect(result.current.error).toBe('Failed to load entitlements');
  });

  it('continues safely when localStorage is unavailable', async () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => { throw new Error('storage blocked'); });
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('storage blocked'); });
    vi.spyOn(Storage.prototype, 'removeItem').mockImplementation(() => { throw new Error('storage blocked'); });
    getEntitlements.mockResolvedValue({ status: 'active' });

    const { result } = renderHook(() => useEntitlements());
    expect(result.current.email).toBe('');
    act(() => result.current.setEmail('buyer@example.com'));
    await act(async () => { await result.current.refresh(); });

    expect(result.current.entitlements).toEqual({ status: 'active' });
  });
});
