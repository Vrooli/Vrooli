import { useState, useCallback } from 'react';
import { getStripeCouponPreview, type CouponImportPreview } from '../../../shared/api/billing';

export interface UseCouponImportReturn {
  /** Whether the modal is open */
  isModalOpen: boolean;
  /** Open the import modal and fetch preview */
  openModal: () => Promise<void>;
  /** Close the modal */
  closeModal: () => void;
  /** Preview data from Stripe */
  preview: CouponImportPreview | null;
  /** Loading state */
  loading: boolean;
  /** Error message */
  error: string | null;
  /** Refresh the preview */
  refreshPreview: () => Promise<void>;
}

/**
 * Hook for managing the coupon import modal.
 * Simpler than plan import since we just display coupons -
 * users assign them to plans via the plan cards.
 */
export function useCouponImport(): UseCouponImportReturn {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [preview, setPreview] = useState<CouponImportPreview | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchPreview = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getStripeCouponPreview();
      setPreview(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch coupon preview');
    } finally {
      setLoading(false);
    }
  }, []);

  const openModal = useCallback(async () => {
    setIsModalOpen(true);
    await fetchPreview();
  }, [fetchPreview]);

  const closeModal = useCallback(() => {
    setIsModalOpen(false);
    setPreview(null);
    setError(null);
  }, []);

  const refreshPreview = useCallback(async () => {
    await fetchPreview();
  }, [fetchPreview]);

  return {
    isModalOpen,
    openModal,
    closeModal,
    preview,
    loading,
    error,
    refreshPreview,
  };
}
