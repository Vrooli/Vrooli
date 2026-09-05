import { useCallback, useEffect, useState } from 'react';
import { getAdminRevenueSummary, type AdminRevenueSummary } from '../../../shared/api';

export interface UseAdminRevenueReturn {
  summary: AdminRevenueSummary | null;
  loading: boolean;
  error: string | null;
  fetchRevenue: () => Promise<void>;
}

/** Loads the finance-owned rollup and preserves the distinction between empty and failed data. */
export function useAdminRevenue(): UseAdminRevenueReturn {
  const [summary, setSummary] = useState<AdminRevenueSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchRevenue = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setSummary(await getAdminRevenueSummary());
    } catch (cause) {
      setSummary(null);
      setError(cause instanceof Error ? cause.message : 'Failed to load revenue');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void fetchRevenue(); }, [fetchRevenue]);

  return { summary, loading, error, fetchRevenue };
}
