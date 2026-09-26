import { act, renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useCouponImport } from './useCouponImport';
import * as billing from '../../../shared/api/billing';

vi.mock('../../../shared/api/billing', async () => ({ ...(await vi.importActual('../../../shared/api/billing')), getStripeCouponPreview: vi.fn() }));

const preview = { coupons: [{ id: 'launch-20', duration: 'once' as const, times_redeemed: 0, valid: true, exists_locally: false }], total_coupons: 1, existing_count: 0, new_count: 1 };

describe('useCouponImport', () => {
  beforeEach(() => { vi.clearAllMocks(); vi.mocked(billing.getStripeCouponPreview).mockResolvedValue(preview); });

  it('opens with a Stripe preview, refreshes it, and clears sensitive stale state on close', async () => {
    const { result } = renderHook(() => useCouponImport());
    await act(async () => { await result.current.openModal(); });
    expect(result.current).toMatchObject({ isModalOpen: true, preview, loading: false, error: null });

    await act(async () => { await result.current.refreshPreview(); });
    expect(billing.getStripeCouponPreview).toHaveBeenCalledTimes(2);
    act(() => { result.current.closeModal(); });
    expect(result.current).toMatchObject({ isModalOpen: false, preview: null, error: null });
  });

  it('keeps the modal usable and exposes the preview failure', async () => {
    vi.mocked(billing.getStripeCouponPreview).mockRejectedValue(new Error('Stripe is unavailable'));
    const { result } = renderHook(() => useCouponImport());
    await act(async () => { await result.current.openModal(); });
    expect(result.current).toMatchObject({ isModalOpen: true, preview: null, loading: false, error: 'Stripe is unavailable' });
  });
});
