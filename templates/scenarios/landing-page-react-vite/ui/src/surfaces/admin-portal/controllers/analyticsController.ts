import { getMetricsSummary, getVariantMetrics, type AnalyticsSummary, type VariantStats } from '../../../shared/api';

export interface AnalyticsDateRange {
  startDate: string;
  endDate: string;
}

function formatDate(daysAgo: number): string {
  const date = new Date();
  date.setDate(date.getDate() - daysAgo);
  return date.toISOString().split('T')[0];
}

export function buildDateRange(days: number): AnalyticsDateRange {
  return {
    startDate: formatDate(days),
    endDate: formatDate(0),
  };
}

function normalizeVariantStats(stats: VariantStats[] | null | undefined): VariantStats[] {
  return Array.isArray(stats) ? stats : [];
}

export async function fetchAnalyticsSummary(range: AnalyticsDateRange): Promise<AnalyticsSummary> {
  const data = await getMetricsSummary(range.startDate, range.endDate);
  return {
    ...data,
    variant_stats: normalizeVariantStats(data.variant_stats),
  };
}

export async function fetchVariantAnalytics(variantSlug: string, range: AnalyticsDateRange): Promise<VariantStats[]> {
  const data = await getVariantMetrics(variantSlug, range.startDate, range.endDate);
  return normalizeVariantStats(data.stats);
}
