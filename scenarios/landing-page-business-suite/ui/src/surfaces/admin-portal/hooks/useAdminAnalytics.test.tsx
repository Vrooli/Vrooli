import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { useAdminAnalytics, DEFAULT_TIME_RANGE, TIME_RANGE_LABELS } from './useAdminAnalytics';
import * as analyticsController from '../controllers/analyticsController';
import * as adminExperience from '../../../shared/lib/adminExperience';
import type { AnalyticsSummary, VariantStats } from '../../../shared/api';
import type { ReactNode } from 'react';

// Mock the controller module
vi.mock('../controllers/analyticsController', async () => {
  const actual = await vi.importActual('../controllers/analyticsController');
  return {
    ...actual,
    fetchAnalyticsSummary: vi.fn(),
    fetchVariantAnalytics: vi.fn(),
  };
});

// Mock admin experience
vi.mock('../../../shared/lib/adminExperience', () => ({
  getAdminExperienceSnapshot: vi.fn().mockReturnValue({ lastAnalytics: null }),
  rememberAnalyticsFilters: vi.fn(),
}));

const mockFetchAnalyticsSummary = vi.mocked(analyticsController.fetchAnalyticsSummary);
const mockFetchVariantAnalytics = vi.mocked(analyticsController.fetchVariantAnalytics);
const mockGetAdminExperienceSnapshot = vi.mocked(adminExperience.getAdminExperienceSnapshot);

const createMockVariantStats = (overrides: Partial<VariantStats> = {}): VariantStats => ({
  variant_id: 1,
  variant_slug: 'test-variant',
  variant_name: 'Test Variant',
  views: 1000,
  cta_clicks: 100,
  conversions: 50,
  downloads: 25,
  conversion_rate: 5.0,
  trend: 'up',
  ...overrides,
});

const createMockSummary = (overrides: Partial<AnalyticsSummary> = {}): AnalyticsSummary => ({
  total_visitors: 5000,
  total_downloads: 500,
  variant_stats: [
    createMockVariantStats({ variant_id: 1, variant_slug: 'variant-a', variant_name: 'Variant A', conversion_rate: 8.0 }),
    createMockVariantStats({ variant_id: 2, variant_slug: 'variant-b', variant_name: 'Variant B', conversion_rate: 3.0 }),
  ],
  top_cta: 'Download Now',
  top_cta_ctr: 12.5,
  ...overrides,
});

function wrapper({ children }: { children: ReactNode }) {
  return <BrowserRouter>{children}</BrowserRouter>;
}

async function settleRouterUpdates(): Promise<void> {
  await act(async () => {
    await Promise.resolve();
  });
}

describe('useAdminAnalytics', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset URL to clear any params from previous tests
    window.history.replaceState({}, '', '/');
    mockGetAdminExperienceSnapshot.mockReturnValue({ version: 1, lastAnalytics: undefined });
    mockFetchAnalyticsSummary.mockResolvedValue(createMockSummary());
    mockFetchVariantAnalytics.mockResolvedValue([]);
  });

  describe('initial state', () => {
    it('starts with loading state', async () => {
      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      expect(result.current.loading).toBe(true);
      await waitFor(() => { expect(result.current.loading).toBe(false); });
    });

    it('uses default time range', async () => {
      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      expect(result.current.timeRange).toBe(DEFAULT_TIME_RANGE);
    });

    it('uses default variant selection (all)', async () => {
      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      expect(result.current.selectedVariant).toBe('all');
    });

    it('uses admin experience for initial values if available', async () => {
      mockGetAdminExperienceSnapshot.mockReturnValue({
        version: 1,
        lastAnalytics: {
          variantSlug: 'saved-variant',
          timeRangeDays: 30,
          savedAt: new Date().toISOString(),
        },
      });

      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.selectedVariant).toBe('saved-variant');
      expect(result.current.timeRange).toBe('30');
    });
  });

  describe('loading data', () => {
    it('fetches analytics on mount', async () => {
      const mockSummary = createMockSummary();
      mockFetchAnalyticsSummary.mockResolvedValue(mockSummary);

      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.summary).toEqual(mockSummary);
      expect(mockFetchAnalyticsSummary).toHaveBeenCalledTimes(1);
    });

    it('handles fetch error', async () => {
      mockFetchAnalyticsSummary.mockRejectedValue(new Error('Network error'));
      const consoleError = vi.spyOn(console, 'error').mockImplementation(() => undefined);

      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.error).toBe('Network error');
      expect(consoleError).toHaveBeenCalledWith('Analytics fetch error:', expect.any(Error));
      consoleError.mockRestore();
    });

    it('can refresh data', async () => {
      mockFetchAnalyticsSummary.mockResolvedValue(createMockSummary());

      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(mockFetchAnalyticsSummary).toHaveBeenCalledTimes(1);

      await act(async () => {
        await result.current.fetchAnalytics();
      });

      expect(mockFetchAnalyticsSummary).toHaveBeenCalledTimes(2);
    });
  });

  describe('filter handling', () => {
    it('changes variant selection', async () => {
      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.handleVariantChange('variant-a');
      });
      await settleRouterUpdates();

      expect(result.current.selectedVariant).toBe('variant-a');
    });

    it('fetches variant details when variant is selected', async () => {
      mockFetchVariantAnalytics.mockResolvedValue([createMockVariantStats()]);

      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.handleVariantChange('variant-a');
      });
      await settleRouterUpdates();

      await waitFor(() => {
        expect(mockFetchVariantAnalytics).toHaveBeenCalledWith('variant-a', expect.any(Object));
      });
    });

    it('clears variant details when selecting "all"', async () => {
      mockFetchVariantAnalytics.mockResolvedValue([createMockVariantStats()]);

      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.handleVariantChange('variant-a');
      });
      await settleRouterUpdates();

      await waitFor(() => {
        expect(result.current.variantDetails.length).toBeGreaterThan(0);
      });

      act(() => {
        result.current.handleVariantChange('all');
      });
      await settleRouterUpdates();

      expect(result.current.variantDetails).toEqual([]);
    });

    it('changes time range', async () => {
      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.handleTimeRangeChange('30');
      });
      await settleRouterUpdates();

      expect(result.current.timeRange).toBe('30');
    });

    it('resets filters', async () => {
      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.handleVariantChange('variant-a');
        result.current.handleTimeRangeChange('30');
      });
      await settleRouterUpdates();

      expect(result.current.selectedVariant).toBe('variant-a');
      expect(result.current.timeRange).toBe('30');

      act(() => {
        result.current.handleResetFilters();
      });
      await settleRouterUpdates();

      expect(result.current.selectedVariant).toBe('all');
      expect(result.current.timeRange).toBe(DEFAULT_TIME_RANGE);
    });

    it('detects when filters changed', async () => {
      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.filtersChanged).toBe(false);

      act(() => {
        result.current.handleVariantChange('variant-a');
      });
      await settleRouterUpdates();

      expect(result.current.filtersChanged).toBe(true);
    });
  });

  describe('computed values', () => {
    it('provides time range label', async () => {
      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.selectedTimeRangeLabel).toBe(TIME_RANGE_LABELS[DEFAULT_TIME_RANGE]);

      act(() => {
        result.current.handleTimeRangeChange('30');
      });
      await settleRouterUpdates();

      expect(result.current.selectedTimeRangeLabel).toBe('Last 30 days');
    });

    it('builds variant name lookup', async () => {
      const mockSummary = createMockSummary();
      mockFetchAnalyticsSummary.mockResolvedValue(mockSummary);

      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.variantNameLookup.get('variant-a')).toBe('Variant A');
      expect(result.current.variantNameLookup.get('variant-b')).toBe('Variant B');
    });

    it('provides selected variant name', async () => {
      mockFetchAnalyticsSummary.mockResolvedValue(createMockSummary());

      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.selectedVariantName).toBeNull();

      act(() => {
        result.current.handleVariantChange('variant-a');
      });
      await settleRouterUpdates();

      expect(result.current.selectedVariantName).toBe('Variant A');
    });

    it('identifies best variant', async () => {
      const mockSummary = createMockSummary({
        variant_stats: [
          createMockVariantStats({ variant_slug: 'low', conversion_rate: 2.0 }),
          createMockVariantStats({ variant_slug: 'high', conversion_rate: 10.0 }),
          createMockVariantStats({ variant_slug: 'mid', conversion_rate: 5.0 }),
        ],
      });
      mockFetchAnalyticsSummary.mockResolvedValue(mockSummary);

      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.bestVariantStat?.variant_slug).toBe('high');
      expect(result.current.bestVariantStat?.conversion_rate).toBe(10.0);
    });

    it('identifies weakest variant', async () => {
      const mockSummary = createMockSummary({
        variant_stats: [
          createMockVariantStats({ variant_slug: 'low', conversion_rate: 2.0 }),
          createMockVariantStats({ variant_slug: 'high', conversion_rate: 10.0 }),
          createMockVariantStats({ variant_slug: 'mid', conversion_rate: 5.0 }),
        ],
      });
      mockFetchAnalyticsSummary.mockResolvedValue(mockSummary);

      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.weakestVariantStat?.variant_slug).toBe('low');
      expect(result.current.weakestVariantStat?.conversion_rate).toBe(2.0);
    });

    it('returns null for best/worst when no variants', async () => {
      mockFetchAnalyticsSummary.mockResolvedValue(createMockSummary({ variant_stats: [] }));

      const { result } = renderHook(() => useAdminAnalytics(), { wrapper });
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.bestVariantStat).toBeNull();
      expect(result.current.weakestVariantStat).toBeNull();
    });
  });
});
