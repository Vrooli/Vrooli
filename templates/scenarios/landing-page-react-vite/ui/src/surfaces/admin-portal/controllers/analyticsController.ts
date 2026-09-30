import { getMetricsSummary, getVariantMetrics, type AnalyticsSummary, type VariantStats } from '../../../shared/api';

export interface AnalyticsDateRange {
  startDate: string;
  endDate: string;
}

function formatDate(daysAgo: number): string {
  const date = new Date();
  date.setDate(date.getDate() - daysAgo);
  return date.toISOString().split('T')[0] ?? '';
}

export function buildDateRange(days: number): AnalyticsDateRange {
  return {
    startDate: formatDate(days),
    endDate: formatDate(0),
  };
}

export async function fetchAnalyticsSummary(range: AnalyticsDateRange): Promise<AnalyticsSummary> {
  return getMetricsSummary(range.startDate, range.endDate);
}

export async function fetchVariantAnalytics(variantSlug: string, range: AnalyticsDateRange): Promise<VariantStats[]> {
  return getVariantMetrics(variantSlug, range.startDate, range.endDate);
}
