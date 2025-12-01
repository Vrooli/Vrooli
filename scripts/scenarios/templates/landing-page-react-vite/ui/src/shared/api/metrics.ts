import { apiCall } from './common';
import type { AnalyticsSummary, MetricEvent, VariantStats } from './types';

export function trackMetric(event: MetricEvent) {
  return apiCall<{ success: boolean }>('/metrics/track', {
    method: 'POST',
    body: JSON.stringify(event),
  });
}

export function getMetricsSummary(startDate?: string, endDate?: string) {
  const params = new URLSearchParams();
  if (startDate) params.set('start_date', startDate);
  if (endDate) params.set('end_date', endDate);
  const query = params.toString() ? `?${params.toString()}` : '';
  return apiCall<AnalyticsSummary>(`/metrics/summary${query}`);
}

export function getVariantMetrics(variantSlug?: string, startDate?: string, endDate?: string) {
  const params = new URLSearchParams();
  if (variantSlug) params.set('variant', variantSlug);
  if (startDate) params.set('start_date', startDate);
  if (endDate) params.set('end_date', endDate);
  const query = params.toString() ? `?${params.toString()}` : '';
  return apiCall<{ start_date: string; end_date: string; stats: VariantStats[] }>(`/metrics/variants${query}`);
}
