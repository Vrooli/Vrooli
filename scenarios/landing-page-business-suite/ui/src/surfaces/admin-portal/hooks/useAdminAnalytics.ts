import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import type { AnalyticsSummary, VariantStats } from '../../../shared/api';
import { buildDateRange, fetchAnalyticsSummary, fetchVariantAnalytics } from '../controllers/analyticsController';
import { getAdminExperienceSnapshot, rememberAnalyticsFilters } from '../../../shared/lib/adminExperience';

export const VALID_TIME_RANGES = new Set(['1', '7', '30', '90']);
export const DEFAULT_TIME_RANGE = '7';
export const TIME_RANGE_LABELS: Record<string, string> = {
  '1': 'Last 24 hours',
  '7': 'Last 7 days',
  '30': 'Last 30 days',
  '90': 'Last 90 days',
};

export interface UseAdminAnalyticsReturn {
  // Data state
  summary: AnalyticsSummary | null;
  variantDetails: VariantStats[];

  // Filter state
  selectedVariant: string;
  timeRange: string;
  selectedTimeRangeLabel: string;
  filtersChanged: boolean;

  // Computed values
  variantNameLookup: Map<string, string>;
  selectedVariantName: string | null;
  bestVariantStat: VariantStats | null;
  weakestVariantStat: VariantStats | null;

  // UI state
  loading: boolean;
  error: string | null;

  // Actions
  fetchAnalytics: () => Promise<void>;
  handleVariantChange: (value: string) => void;
  handleTimeRangeChange: (value: string) => void;
  handleResetFilters: () => void;

  // Navigation helpers
  navigateToHeroSection: (slug: string) => void;
  navigateToVariantEditor: (slug: string) => void;
  openVariantPreview: (slug: string) => void;
}

/**
 * Reactive hook for admin analytics dashboard
 *
 * Provides state and handlers for:
 * - Loading analytics summary and variant details
 * - Filtering by time range and variant
 * - Syncing filters to URL and admin experience
 * - Computing best/worst performers
 */
export function useAdminAnalytics(): UseAdminAnalyticsReturn {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  // Compute initial values from URL and admin experience
  const adminExperience = useMemo(() => getAdminExperienceSnapshot(), []);
  const initialVariant = searchParams.get('variant') ?? adminExperience.lastAnalytics?.variantSlug ?? 'all';
  const initialTimeRangeFromUrl = searchParams.get('range');
  const initialRangeFromExperience = adminExperience.lastAnalytics?.timeRangeDays
    ? String(adminExperience.lastAnalytics.timeRangeDays)
    : undefined;
  const initialRange =
    initialTimeRangeFromUrl && VALID_TIME_RANGES.has(initialTimeRangeFromUrl)
      ? initialTimeRangeFromUrl
      : initialRangeFromExperience && VALID_TIME_RANGES.has(initialRangeFromExperience)
        ? initialRangeFromExperience
        : DEFAULT_TIME_RANGE;

  // Data state
  const [summary, setSummary] = useState<AnalyticsSummary | null>(null);
  const [variantDetails, setVariantDetails] = useState<VariantStats[]>([]);

  // Filter state
  const [selectedVariant, setSelectedVariant] = useState<string>(initialVariant);
  const [timeRange, setTimeRange] = useState<string>(initialRange);

  // UI state
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Track initial mount to avoid overwriting admin experience values
  const isInitialMount = useRef(true);
  // Track latest state values to avoid stale closures in sync effect
  const selectedVariantRef = useRef(selectedVariant);
  const timeRangeRef = useRef(timeRange);
  const searchSignature = searchParams.toString();

  // Keep refs in sync with state
  useEffect(() => {
    selectedVariantRef.current = selectedVariant;
    timeRangeRef.current = timeRange;
  });

  /**
   * Fetch analytics summary
   */
  const fetchAnalytics = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const days = parseInt(timeRange, 10);
      const range = buildDateRange(days);
      const data = await fetchAnalyticsSummary(range);
      setSummary(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load analytics');
      console.error('Analytics fetch error:', err);
    } finally {
      setLoading(false);
    }
  }, [timeRange]);

  /**
   * Fetch variant details
   */
  const fetchVariantDetails = useCallback(
    async (variantSlug: string) => {
      try {
        const range = buildDateRange(parseInt(timeRange, 10));
        const stats = await fetchVariantAnalytics(variantSlug, range);
        setVariantDetails(stats);
      } catch (err) {
        console.error('Failed to load variant details:', err);
        setVariantDetails([]);
      }
    },
    [timeRange]
  );

  // Initial load and time range changes
  useEffect(() => {
    fetchAnalytics();
  }, [fetchAnalytics]);

  // Fetch variant details when selection changes
  useEffect(() => {
    if (selectedVariant !== 'all') {
      fetchVariantDetails(selectedVariant);
    } else {
      setVariantDetails([]);
    }
  }, [selectedVariant, timeRange, fetchVariantDetails]);

  // Remember analytics filters in admin experience
  useEffect(() => {
    if (!summary) return;
    const variantSlug = selectedVariant === 'all' ? null : selectedVariant;
    const variantName = variantSlug
      ? summary.variant_stats.find((stat) => stat.variant_slug === variantSlug)?.variant_name
      : undefined;
    rememberAnalyticsFilters({
      variantSlug,
      variantName,
      timeRangeDays: parseInt(timeRange, 10),
    });
  }, [summary, selectedVariant, timeRange]);

  // Sync URL params to state (skip initial mount to respect admin experience values)
  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      return;
    }

    const params = new URLSearchParams(searchSignature);
    const urlVariant = params.get('variant') ?? 'all';
    const urlRange = params.get('range');
    const normalizedRange = urlRange && VALID_TIME_RANGES.has(urlRange) ? urlRange : DEFAULT_TIME_RANGE;

    // Use refs to get latest state values (avoids stale closure issues)
    if (urlVariant !== selectedVariantRef.current) {
      setSelectedVariant(urlVariant);
    }
    if (normalizedRange !== timeRangeRef.current) {
      setTimeRange(normalizedRange);
    }
  }, [searchSignature]);

  /**
   * Sync filters to URL
   */
  const syncFiltersToUrl = useCallback(
    (nextVariant: string, nextRange: string) => {
      const params = new URLSearchParams();
      if (nextVariant !== 'all') {
        params.set('variant', nextVariant);
      }
      if (nextRange !== DEFAULT_TIME_RANGE) {
        params.set('range', nextRange);
      }

      const nextQuery = params.toString();
      if (nextQuery === searchSignature) {
        return;
      }

      setSearchParams(params, { replace: true });
    },
    [searchSignature, setSearchParams]
  );

  /**
   * Handle variant selection change
   */
  const handleVariantChange = useCallback(
    (value: string) => {
      selectedVariantRef.current = value; // Update ref immediately for other handlers
      setSelectedVariant(value);
      syncFiltersToUrl(value, timeRangeRef.current);
    },
    [syncFiltersToUrl]
  );

  /**
   * Handle time range change
   */
  const handleTimeRangeChange = useCallback(
    (value: string) => {
      timeRangeRef.current = value; // Update ref immediately for other handlers
      setTimeRange(value);
      syncFiltersToUrl(selectedVariantRef.current, value);
    },
    [syncFiltersToUrl]
  );

  /**
   * Reset filters to defaults
   */
  const handleResetFilters = useCallback(() => {
    selectedVariantRef.current = 'all';
    timeRangeRef.current = DEFAULT_TIME_RANGE;
    setSelectedVariant('all');
    setTimeRange(DEFAULT_TIME_RANGE);
    syncFiltersToUrl('all', DEFAULT_TIME_RANGE);
  }, [syncFiltersToUrl]);

  /**
   * Navigate to hero section for a variant
   */
  const navigateToHeroSection = useCallback(
    (slug: string) => {
      const params = new URLSearchParams({ focus: slug, focusSectionType: 'hero' });
      navigate(`/admin/customization?${params.toString()}`);
    },
    [navigate]
  );

  /**
   * Navigate to variant editor
   */
  const navigateToVariantEditor = useCallback(
    (slug: string) => {
      navigate(`/admin/customization/variants/${slug}`);
    },
    [navigate]
  );

  /**
   * Open variant preview in new tab
   */
  const openVariantPreview = useCallback((slug: string) => {
    window.open(`/?variant=${slug}`, '_blank');
  }, []);

  // Computed values
  const filtersChanged = selectedVariant !== 'all' || timeRange !== DEFAULT_TIME_RANGE;
  const selectedTimeRangeLabel = TIME_RANGE_LABELS[timeRange] ?? TIME_RANGE_LABELS[DEFAULT_TIME_RANGE] ?? 'Last 7 days';

  const variantNameLookup = useMemo(() => {
    const map = new Map<string, string>();
    summary?.variant_stats.forEach((stat) => map.set(stat.variant_slug, stat.variant_name));
    return map;
  }, [summary]);

  const selectedVariantName =
    selectedVariant !== 'all' ? variantNameLookup.get(selectedVariant) ?? selectedVariant : null;

  const bestVariantStat = useMemo(() => {
    if (!summary?.variant_stats?.length) {
      return null;
    }
    return summary.variant_stats.reduce<VariantStats | null>((best, stat) => {
      if (!best) return stat;
      return stat.conversion_rate > best.conversion_rate ? stat : best;
    }, null);
  }, [summary]);

  const weakestVariantStat = useMemo(() => {
    if (!summary?.variant_stats?.length) {
      return null;
    }
    return summary.variant_stats.reduce<VariantStats | null>((worst, stat) => {
      if (!worst) return stat;
      return stat.conversion_rate < worst.conversion_rate ? stat : worst;
    }, null);
  }, [summary]);

  return {
    // Data state
    summary,
    variantDetails,

    // Filter state
    selectedVariant,
    timeRange,
    selectedTimeRangeLabel,
    filtersChanged,

    // Computed values
    variantNameLookup,
    selectedVariantName,
    bestVariantStat,
    weakestVariantStat,

    // UI state
    loading,
    error,

    // Actions
    fetchAnalytics,
    handleVariantChange,
    handleTimeRangeChange,
    handleResetFilters,

    // Navigation helpers
    navigateToHeroSection,
    navigateToVariantEditor,
    openVariantPreview,
  };
}
