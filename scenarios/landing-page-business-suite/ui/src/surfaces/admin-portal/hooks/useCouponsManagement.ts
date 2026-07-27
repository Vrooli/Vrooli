import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  listCoupons,
  createCoupon,
  deleteCoupon,
  getCouponUsage,
  type StripeCoupon,
  type CreateCouponPayload,
  type CouponUsageStats,
} from '../../../shared/api/billing';

export type CouponFilter = 'all' | 'active' | 'expired';

export interface UseCouponsManagementReturn {
  // Data state
  coupons: StripeCoupon[];
  filteredCoupons: StripeCoupon[];
  introCouponMap: Record<string, string> | null;
  usageStats: CouponUsageStats[];

  // Filter state
  filter: CouponFilter;
  setFilter: (filter: CouponFilter) => void;

  // Stats
  totalCount: number;
  activeCount: number;
  introConfiguredCount: number;

  // UI state
  loading: boolean;
  error: string | null;
  createModalOpen: boolean;
  creating: boolean;
  createError: string | null;
  deletingId: string | null;

  // Actions
  loadCoupons: () => Promise<void>;
  openCreateModal: () => void;
  closeCreateModal: () => void;
  handleCreate: (payload: CreateCouponPayload) => Promise<{ success: boolean; error?: string }>;
  handleDelete: (couponId: string) => Promise<{ success: boolean; error?: string }>;
  clearError: () => void;
  clearCreateError: () => void;
}

/**
 * Hook for managing Stripe coupons in the admin portal.
 */
export function useCouponsManagement(): UseCouponsManagementReturn {
  // Data state
  const [coupons, setCoupons] = useState<StripeCoupon[]>([]);
  const [introCouponMap, setIntroCouponMap] = useState<Record<string, string> | null>(null);
  const [usageStats, setUsageStats] = useState<CouponUsageStats[]>([]);

  // Filter state
  const [filter, setFilter] = useState<CouponFilter>('all');

  // UI state
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  /**
   * Load coupons from API
   */
  const loadCoupons = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [couponsResponse, usageResponse] = await Promise.all([
        listCoupons(),
        getCouponUsage().catch(() => [] as CouponUsageStats[]),
      ]);
      setCoupons(couponsResponse.coupons);
      setIntroCouponMap(couponsResponse.intro_coupon_map ?? null);
      setUsageStats(usageResponse);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load coupons');
    } finally {
      setLoading(false);
    }
  }, []);

  // Initial load
  useEffect(() => {
    void loadCoupons();
  }, [loadCoupons]);

  /**
   * Filtered coupons based on current filter
   */
  const filteredCoupons = useMemo(() => {
    switch (filter) {
      case 'active':
        return coupons.filter((c) => c.valid);
      case 'expired':
        return coupons.filter((c) => !c.valid);
      default:
        return coupons;
    }
  }, [coupons, filter]);

  /**
   * Computed stats
   */
  const totalCount = coupons.length;
  const activeCount = useMemo(() => coupons.filter((c) => c.valid).length, [coupons]);
  const introConfiguredCount = useMemo(
    () => (introCouponMap ? Object.keys(introCouponMap).length : 0),
    [introCouponMap]
  );

  /**
   * Open create modal
   */
  const openCreateModal = useCallback(() => {
    setCreateError(null);
    setCreateModalOpen(true);
  }, []);

  /**
   * Close create modal
   */
  const closeCreateModal = useCallback(() => {
    setCreateModalOpen(false);
    setCreateError(null);
  }, []);

  /**
   * Create a new coupon
   */
  const handleCreate = useCallback(
    async (payload: CreateCouponPayload): Promise<{ success: boolean; error?: string }> => {
      setCreating(true);
      setCreateError(null);
      try {
        const newCoupon = await createCoupon(payload);
        setCoupons((prev) => [newCoupon, ...prev]);
        setCreateModalOpen(false);
        return { success: true };
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to create coupon';
        setCreateError(message);
        return { success: false, error: message };
      } finally {
        setCreating(false);
      }
    },
    []
  );

  /**
   * Delete a coupon
   */
  const handleDelete = useCallback(
    async (couponId: string): Promise<{ success: boolean; error?: string }> => {
      setDeletingId(couponId);
      try {
        await deleteCoupon(couponId);
        setCoupons((prev) => prev.filter((c) => c.id !== couponId));
        return { success: true };
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to delete coupon';
        return { success: false, error: message };
      } finally {
        setDeletingId(null);
      }
    },
    []
  );

  /**
   * Clear main error
   */
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  /**
   * Clear create error
   */
  const clearCreateError = useCallback(() => {
    setCreateError(null);
  }, []);

  return {
    // Data state
    coupons,
    filteredCoupons,
    introCouponMap,
    usageStats,

    // Filter state
    filter,
    setFilter,

    // Stats
    totalCount,
    activeCount,
    introConfiguredCount,

    // UI state
    loading,
    error,
    createModalOpen,
    creating,
    createError,
    deletingId,

    // Actions
    loadCoupons,
    openCreateModal,
    closeCreateModal,
    handleCreate,
    handleDelete,
    clearError,
    clearCreateError,
  };
}
