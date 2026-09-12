import { useState, useEffect, useCallback } from 'react';
import {
  getCouponMappings,
  setCouponForPlan,
  removeCouponFromPlan,
  listCoupons,
  type StripeCoupon,
} from '../../../shared/api/billing';

export interface UseCouponMappingsReturn {
  /** Map of priceID -> couponID */
  mappings: Record<string, string>;
  /** Available coupons from Stripe */
  availableCoupons: StripeCoupon[];
  /** Loading state */
  loading: boolean;
  /** Error message */
  error: string | null;
  /** Whether a save operation is in progress */
  saving: boolean;
  /** Load mappings and coupons from API */
  refresh: () => Promise<void>;
  /** Assign a coupon to a plan */
  assignCoupon: (priceId: string, couponId: string) => Promise<{ success: boolean; error?: string }>;
  /** Remove coupon assignment from a plan */
  unassignCoupon: (priceId: string) => Promise<{ success: boolean; error?: string }>;
  /** Get the coupon assigned to a price (or undefined) */
  getCouponForPrice: (priceId: string) => StripeCoupon | undefined;
  /** Clear error */
  clearError: () => void;
}

/**
 * Hook for managing coupon-to-plan mappings.
 * Provides the mappings and available coupons for use in plan editing.
 */
export function useCouponMappings(): UseCouponMappingsReturn {
  const [mappings, setMappings] = useState<Record<string, string>>({});
  const [availableCoupons, setAvailableCoupons] = useState<StripeCoupon[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  /**
   * Load mappings and coupons from API
   */
  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [mappingsResponse, couponsResponse] = await Promise.all([
        getCouponMappings(),
        listCoupons(),
      ]);
      setMappings(mappingsResponse.mappings);
      // Only show valid coupons as options
      setAvailableCoupons(couponsResponse.coupons.filter((c) => c.valid));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load coupon mappings');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial load
  useEffect(() => {
    void refresh();
  }, [refresh]);

  /**
   * Assign a coupon to a plan
   */
  const assignCoupon = useCallback(
    async (priceId: string, couponId: string): Promise<{ success: boolean; error?: string }> => {
      setSaving(true);
      try {
        await setCouponForPlan(priceId, couponId);
        setMappings((prev) => ({ ...prev, [priceId]: couponId }));
        return { success: true };
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to assign coupon';
        return { success: false, error: message };
      } finally {
        setSaving(false);
      }
    },
    []
  );

  /**
   * Remove coupon assignment from a plan
   */
  const unassignCoupon = useCallback(
    async (priceId: string): Promise<{ success: boolean; error?: string }> => {
      setSaving(true);
      try {
        await removeCouponFromPlan(priceId);
        setMappings((prev) => {
          const next = { ...prev };
          Reflect.deleteProperty(next, priceId);
          return next;
        });
        return { success: true };
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to remove coupon';
        return { success: false, error: message };
      } finally {
        setSaving(false);
      }
    },
    []
  );

  /**
   * Get the coupon assigned to a price (or undefined)
   */
  const getCouponForPrice = useCallback(
    (priceId: string): StripeCoupon | undefined => {
      const couponId = mappings[priceId];
      if (!couponId) return undefined;
      return availableCoupons.find((c) => c.id === couponId);
    },
    [mappings, availableCoupons]
  );

  /**
   * Clear error
   */
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  return {
    mappings,
    availableCoupons,
    loading,
    error,
    saving,
    refresh,
    assignCoupon,
    unassignCoupon,
    getCouponForPrice,
    clearError,
  };
}
