import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useCouponMappings } from './useCouponMappings';
import * as billing from '../../../shared/api/billing';

vi.mock('../../../shared/api/billing', async () => ({
  ...(await vi.importActual('../../../shared/api/billing')),
  getCouponMappings: vi.fn(), listCoupons: vi.fn(), setCouponForPlan: vi.fn(), removeCouponFromPlan: vi.fn(),
}));

const coupon = { id: 'launch-20', duration: 'forever' as const, times_redeemed: 0, valid: true, created: 1_704_067_200, is_intro_coupon: false, percent_off: 20 };

describe('useCouponMappings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(billing.getCouponMappings).mockResolvedValue({ mappings: { price_pro: 'launch-20' } });
    vi.mocked(billing.listCoupons).mockResolvedValue({ coupons: [coupon, { ...coupon, id: 'expired', valid: false }] });
    vi.mocked(billing.setCouponForPlan).mockResolvedValue(undefined);
    vi.mocked(billing.removeCouponFromPlan).mockResolvedValue(undefined);
  });

  it('loads mappings and presents only valid coupons for subscription pricing', async () => {
    const { result } = renderHook(() => useCouponMappings());
    await waitFor(() => { expect(result.current.loading).toBe(false); });

    expect(result.current.mappings).toEqual({ price_pro: 'launch-20' });
    expect(result.current.availableCoupons).toEqual([coupon]);
    expect(result.current.getCouponForPrice('price_pro')).toEqual(coupon);
  });

  it('assigns and removes a coupon only after persistence succeeds', async () => {
    const { result } = renderHook(() => useCouponMappings());
    await waitFor(() => { expect(result.current.loading).toBe(false); });

    await act(async () => { expect(await result.current.assignCoupon('price_team', 'launch-20')).toEqual({ success: true }); });
    expect(billing.setCouponForPlan).toHaveBeenCalledWith('price_team', 'launch-20');
    expect(result.current.mappings.price_team).toBe('launch-20');

    await act(async () => { expect(await result.current.unassignCoupon('price_pro')).toEqual({ success: true }); });
    expect(result.current.mappings.price_pro).toBeUndefined();
  });

  it('retains existing mappings when a billing mutation fails', async () => {
    vi.mocked(billing.removeCouponFromPlan).mockRejectedValue(new Error('Access denied'));
    const { result } = renderHook(() => useCouponMappings());
    await waitFor(() => { expect(result.current.loading).toBe(false); });

    await act(async () => { expect(await result.current.unassignCoupon('price_pro')).toEqual({ success: false, error: 'Access denied' }); });
    expect(result.current.mappings).toEqual({ price_pro: 'launch-20' });
    expect(result.current.saving).toBe(false);
  });
});
