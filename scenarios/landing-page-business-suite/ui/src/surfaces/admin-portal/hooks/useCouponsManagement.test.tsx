import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useCouponsManagement } from './useCouponsManagement';
import * as billing from '../../../shared/api/billing';

vi.mock('../../../shared/api/billing', async () => ({
  ...(await vi.importActual('../../../shared/api/billing')),
  listCoupons: vi.fn(),
  createCoupon: vi.fn(),
  deleteCoupon: vi.fn(),
  getCouponUsage: vi.fn(),
}));

const activeCoupon = {
  id: 'launch-20', duration: 'once' as const, times_redeemed: 2, valid: true,
  created: 1_704_067_200, is_intro_coupon: false, percent_off: 20,
};
const expiredCoupon = { ...activeCoupon, id: 'expired', valid: false };

describe('useCouponsManagement', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(billing.listCoupons).mockResolvedValue({
      coupons: [activeCoupon, expiredCoupon], intro_coupon_map: { starter: 'launch-20' },
    });
    vi.mocked(billing.getCouponUsage).mockResolvedValue([{ coupon_id: 'launch-20', total_uses: 3 }]);
  });

  it('loads coupons, usage, and computes the active and intro-pricing views', async () => {
    const { result } = renderHook(() => useCouponsManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });

    expect(result.current.totalCount).toBe(2);
    expect(result.current.activeCount).toBe(1);
    expect(result.current.introConfiguredCount).toBe(1);
    expect(result.current.usageStats).toEqual([{ coupon_id: 'launch-20', total_uses: 3 }]);

    act(() => { result.current.setFilter('expired'); });
    expect(result.current.filteredCoupons).toEqual([expiredCoupon]);
    act(() => { result.current.setFilter('active'); });
    expect(result.current.filteredCoupons).toEqual([activeCoupon]);
    act(() => { result.current.setFilter('all'); });
    expect(result.current.filteredCoupons).toEqual([activeCoupon, expiredCoupon]);
  });

  it('treats absent intro mappings as unconfigured and exposes a safe list failure', async () => {
    vi.mocked(billing.listCoupons).mockResolvedValue({ coupons: [activeCoupon] });
    const available = renderHook(() => useCouponsManagement());
    await waitFor(() => { expect(available.result.current.loading).toBe(false); });
    expect(available.result.current.introCouponMap).toBeNull();
    expect(available.result.current.introConfiguredCount).toBe(0);
    available.unmount();

    vi.mocked(billing.listCoupons).mockRejectedValue('offline');
    const failed = renderHook(() => useCouponsManagement());
    await waitFor(() => { expect(failed.result.current.loading).toBe(false); });
    expect(failed.result.current.error).toBe('Failed to load coupons');
  });

  it('keeps coupon listing available when usage reporting is temporarily unavailable', async () => {
    vi.mocked(billing.getCouponUsage).mockRejectedValue(new Error('Usage unavailable'));
    const { result } = renderHook(() => useCouponsManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });

    expect(result.current.coupons).toEqual([activeCoupon, expiredCoupon]);
    expect(result.current.usageStats).toEqual([]);
    expect(result.current.error).toBeNull();
  });

  it('creates and deletes coupons optimistically only after their respective API calls succeed', async () => {
    vi.mocked(billing.createCoupon).mockResolvedValue({ ...activeCoupon, id: 'new-coupon' });
    vi.mocked(billing.deleteCoupon).mockResolvedValue(undefined);
    const { result } = renderHook(() => useCouponsManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });

    act(() => { result.current.openCreateModal(); });
    await act(async () => {
      expect(await result.current.handleCreate({ percent_off: 10, duration: 'once' })).toEqual({ success: true });
    });
    expect(result.current.createModalOpen).toBe(false);
    expect(result.current.coupons[0]?.id).toBe('new-coupon');

    await act(async () => {
      expect(await result.current.handleDelete('launch-20')).toEqual({ success: true });
    });
    expect(result.current.coupons.map((coupon) => coupon.id)).not.toContain('launch-20');
  });

  it('returns actionable create and delete errors without corrupting the displayed coupons', async () => {
    vi.mocked(billing.createCoupon).mockRejectedValue(new Error('Stripe unavailable'));
    vi.mocked(billing.deleteCoupon).mockRejectedValue(new Error('Coupon is already in use'));
    const { result } = renderHook(() => useCouponsManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });

    await act(async () => {
      expect(await result.current.handleCreate({ percent_off: 10, duration: 'once' })).toEqual({ success: false, error: 'Stripe unavailable' });
    });
    expect(result.current.createError).toBe('Stripe unavailable');

    await act(async () => {
      expect(await result.current.handleDelete('launch-20')).toEqual({ success: false, error: 'Coupon is already in use' });
    });
    expect(result.current.coupons).toEqual([activeCoupon, expiredCoupon]);
    expect(result.current.deletingId).toBeNull();
  });

  it('keeps modal and operation state predictable for non-Error failures', async () => {
    vi.mocked(billing.createCoupon).mockRejectedValue('offline');
    vi.mocked(billing.deleteCoupon).mockRejectedValue('offline');
    const { result } = renderHook(() => useCouponsManagement());
    await waitFor(() => { expect(result.current.loading).toBe(false); });

    act(() => { result.current.openCreateModal(); });
    expect(result.current.createModalOpen).toBe(true);
    await act(async () => {
      expect(await result.current.handleCreate({ percent_off: 10, duration: 'once' })).toEqual({ success: false, error: 'Failed to create coupon' });
    });
    expect(result.current.creating).toBe(false);
    expect(result.current.createModalOpen).toBe(true);
    act(() => { result.current.clearCreateError(); result.current.closeCreateModal(); });
    expect(result.current.createError).toBeNull();
    expect(result.current.createModalOpen).toBe(false);

    await act(async () => {
      expect(await result.current.handleDelete('launch-20')).toEqual({ success: false, error: 'Failed to delete coupon' });
    });
    expect(result.current.deletingId).toBeNull();
  });
});
