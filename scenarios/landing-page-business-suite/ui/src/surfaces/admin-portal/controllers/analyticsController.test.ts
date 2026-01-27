import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { AnalyticsSummary, VariantStats, getMetricsSummary, getVariantMetrics } from '../../../shared/api';
import {
  buildDateRange,
  fetchAnalyticsSummary,
  fetchVariantAnalytics,
  type AnalyticsDateRange,
} from './analyticsController';

// Mock the API module
type GetMetricsSummaryFn = typeof getMetricsSummary;
type GetVariantMetricsFn = typeof getVariantMetrics;

const getMetricsSummaryMock = vi.fn<Parameters<GetMetricsSummaryFn>, ReturnType<GetMetricsSummaryFn>>();
const getVariantMetricsMock = vi.fn<Parameters<GetVariantMetricsFn>, ReturnType<GetVariantMetricsFn>>();

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getMetricsSummary: (...args: Parameters<GetMetricsSummaryFn>) => getMetricsSummaryMock(...args),
    getVariantMetrics: (...args: Parameters<GetVariantMetricsFn>) => getVariantMetricsMock(...args),
  };
});

const mockVariantStats: VariantStats[] = [
  {
    variant_id: 1,
    variant_slug: 'control',
    variant_name: 'Control',
    views: 1000,
    cta_clicks: 50,
    conversions: 10,
    downloads: 5,
    conversion_rate: 1.0,
    avg_scroll_depth: 75,
  },
  {
    variant_id: 2,
    variant_slug: 'test-a',
    variant_name: 'Test A',
    views: 800,
    cta_clicks: 60,
    conversions: 15,
    downloads: 8,
    conversion_rate: 1.875,
    avg_scroll_depth: 80,
  },
];

const mockSummary: AnalyticsSummary = {
  total_visitors: 1800,
  total_downloads: 13,
  variant_stats: mockVariantStats,
  top_cta: 'Get Started',
  top_cta_ctr: 0.061,
};

describe('analyticsController', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('buildDateRange', () => {
    it('builds date range with correct start and end dates', () => {
      const range = buildDateRange(7);

      expect(range.startDate).toMatch(/^\d{4}-\d{2}-\d{2}$/);
      expect(range.endDate).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    });

    it('end date is today', () => {
      const range = buildDateRange(7);
      const today = new Date().toISOString().split('T')[0];

      expect(range.endDate).toBe(today);
    });

    it('start date is N days ago', () => {
      const range = buildDateRange(7);

      const expectedStart = new Date();
      expectedStart.setDate(expectedStart.getDate() - 7);
      const expectedStartStr = expectedStart.toISOString().split('T')[0];

      expect(range.startDate).toBe(expectedStartStr);
    });

    it('handles 0 days (same day range)', () => {
      const range = buildDateRange(0);
      const today = new Date().toISOString().split('T')[0];

      expect(range.startDate).toBe(today);
      expect(range.endDate).toBe(today);
    });

    it('handles 30-day range', () => {
      const range = buildDateRange(30);

      const expectedStart = new Date();
      expectedStart.setDate(expectedStart.getDate() - 30);
      const expectedStartStr = expectedStart.toISOString().split('T')[0];

      expect(range.startDate).toBe(expectedStartStr);
    });
  });

  describe('fetchAnalyticsSummary', () => {
    it('calls getMetricsSummary with date range', async () => {
      getMetricsSummaryMock.mockResolvedValue(mockSummary);

      const range: AnalyticsDateRange = {
        startDate: '2025-01-01',
        endDate: '2025-01-07',
      };

      await fetchAnalyticsSummary(range);

      expect(getMetricsSummaryMock).toHaveBeenCalledWith('2025-01-01', '2025-01-07');
    });

    it('returns analytics summary with normalized variant_stats', async () => {
      getMetricsSummaryMock.mockResolvedValue(mockSummary);

      const range: AnalyticsDateRange = {
        startDate: '2025-01-01',
        endDate: '2025-01-07',
      };

      const result = await fetchAnalyticsSummary(range);

      expect(result.total_visitors).toBe(1800);
      expect(result.total_downloads).toBe(13);
      expect(result.variant_stats).toHaveLength(2);
      expect(result.top_cta).toBe('Get Started');
    });

    it('normalizes null variant_stats to empty array', async () => {
      getMetricsSummaryMock.mockResolvedValue({
        ...mockSummary,
        variant_stats: null,
      } as unknown as AnalyticsSummary);

      const range: AnalyticsDateRange = {
        startDate: '2025-01-01',
        endDate: '2025-01-07',
      };

      const result = await fetchAnalyticsSummary(range);

      expect(result.variant_stats).toEqual([]);
    });

    it('normalizes undefined variant_stats to empty array', async () => {
      getMetricsSummaryMock.mockResolvedValue({
        total_visitors: 100,
        variant_stats: undefined,
      } as unknown as AnalyticsSummary);

      const range: AnalyticsDateRange = {
        startDate: '2025-01-01',
        endDate: '2025-01-07',
      };

      const result = await fetchAnalyticsSummary(range);

      expect(result.variant_stats).toEqual([]);
    });

    it('propagates API errors', async () => {
      getMetricsSummaryMock.mockRejectedValue(new Error('API failure'));

      const range: AnalyticsDateRange = {
        startDate: '2025-01-01',
        endDate: '2025-01-07',
      };

      await expect(fetchAnalyticsSummary(range)).rejects.toThrow('API failure');
    });
  });

  describe('fetchVariantAnalytics', () => {
    it('calls getVariantMetrics with variant slug and date range', async () => {
      getVariantMetricsMock.mockResolvedValue({
        start_date: '2025-01-01',
        end_date: '2025-01-07',
        stats: mockVariantStats,
      });

      const range: AnalyticsDateRange = {
        startDate: '2025-01-01',
        endDate: '2025-01-07',
      };

      await fetchVariantAnalytics('control', range);

      expect(getVariantMetricsMock).toHaveBeenCalledWith('control', '2025-01-01', '2025-01-07');
    });

    it('returns normalized variant stats array', async () => {
      getVariantMetricsMock.mockResolvedValue({
        start_date: '2025-01-01',
        end_date: '2025-01-07',
        stats: mockVariantStats,
      });

      const range: AnalyticsDateRange = {
        startDate: '2025-01-01',
        endDate: '2025-01-07',
      };

      const result = await fetchVariantAnalytics('control', range);

      expect(result).toHaveLength(2);
      expect(result[0]?.variant_slug).toBe('control');
    });

    it('normalizes null stats to empty array', async () => {
      getVariantMetricsMock.mockResolvedValue({
        start_date: '2025-01-01',
        end_date: '2025-01-07',
        stats: null,
      } as unknown as Awaited<ReturnType<GetVariantMetricsFn>>);

      const range: AnalyticsDateRange = {
        startDate: '2025-01-01',
        endDate: '2025-01-07',
      };

      const result = await fetchVariantAnalytics('control', range);

      expect(result).toEqual([]);
    });

    it('normalizes undefined stats to empty array', async () => {
      getVariantMetricsMock.mockResolvedValue({
        start_date: '2025-01-01',
        end_date: '2025-01-07',
        stats: undefined,
      } as unknown as Awaited<ReturnType<GetVariantMetricsFn>>);

      const range: AnalyticsDateRange = {
        startDate: '2025-01-01',
        endDate: '2025-01-07',
      };

      const result = await fetchVariantAnalytics('control', range);

      expect(result).toEqual([]);
    });

    it('propagates API errors', async () => {
      getVariantMetricsMock.mockRejectedValue(new Error('Variant not found'));

      const range: AnalyticsDateRange = {
        startDate: '2025-01-01',
        endDate: '2025-01-07',
      };

      await expect(fetchVariantAnalytics('invalid', range)).rejects.toThrow('Variant not found');
    });
  });
});
