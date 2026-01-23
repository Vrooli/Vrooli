import { useState, useEffect, useCallback, useMemo } from 'react';
import type { AdminUsageSummary } from '../../../shared/api';
import {
  getCurrentPeriod,
  navigatePeriod,
  isCurrentPeriod,
  formatMonthYear,
} from '../../../shared/lib/dateFormatters';
import {
  calculateTotalUsage,
  getSortedAppTotals,
  getTopUsers,
  getLimitedRecords,
  fetchUsageSummary,
} from '../services/usage.service';

export interface UseUsageDashboardReturn {
  // Data state
  summary: AdminUsageSummary | null;
  totalUsage: number;
  sortedAppTotals: ReturnType<typeof getSortedAppTotals>;
  topUsers: ReturnType<typeof getTopUsers>;
  recentRecords: AdminUsageSummary['records'];

  // Period state
  billingPeriod: string;
  formattedPeriod: string;
  isCurrentPeriod: boolean;

  // UI state
  loading: boolean;
  error: string | null;

  // Actions
  fetchSummary: () => Promise<void>;
  navigateMonth: (delta: number) => void;
  clearError: () => void;
}

/**
 * Reactive hook for usage dashboard
 *
 * Provides state and handlers for:
 * - Loading usage summary data
 * - Navigating between billing periods
 * - Computing derived values (totals, sorted lists)
 */
export function useUsageDashboard(): UseUsageDashboardReturn {
  // Data state
  const [summary, setSummary] = useState<AdminUsageSummary | null>(null);

  // Period state
  const [billingPeriod, setBillingPeriod] = useState(() => getCurrentPeriod());

  // UI state
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  /**
   * Fetch usage summary for current billing period
   */
  const fetchSummary = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchUsageSummary(billingPeriod);
      setSummary(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load usage data');
    } finally {
      setLoading(false);
    }
  }, [billingPeriod]);

  // Fetch on mount and when billing period changes
  useEffect(() => {
    fetchSummary();
  }, [fetchSummary]);

  /**
   * Navigate to previous or next month
   */
  const navigateMonth = useCallback((delta: number) => {
    setBillingPeriod((current) => navigatePeriod(current, delta));
  }, []);

  /**
   * Clear error state
   */
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  // Computed values
  const totalUsage = useMemo(
    () => calculateTotalUsage(summary?.user_totals),
    [summary]
  );

  const sortedAppTotals = useMemo(
    () => (summary?.app_totals ? getSortedAppTotals(summary.app_totals) : []),
    [summary]
  );

  const topUsers = useMemo(
    () => (summary?.user_totals ? getTopUsers(summary.user_totals, 10) : []),
    [summary]
  );

  const recentRecords = useMemo(
    () => (summary?.records ? getLimitedRecords(summary.records, 20) : []),
    [summary]
  );

  const formattedPeriod = useMemo(
    () => formatMonthYear(billingPeriod),
    [billingPeriod]
  );

  const isCurrentBillingPeriod = useMemo(
    () => isCurrentPeriod(billingPeriod),
    [billingPeriod]
  );

  return {
    // Data state
    summary,
    totalUsage,
    sortedAppTotals,
    topUsers,
    recentRecords,

    // Period state
    billingPeriod,
    formattedPeriod,
    isCurrentPeriod: isCurrentBillingPeriod,

    // UI state
    loading,
    error,

    // Actions
    fetchSummary,
    navigateMonth,
    clearError,
  };
}
