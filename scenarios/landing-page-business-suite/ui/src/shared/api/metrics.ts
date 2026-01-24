import { z } from 'zod';
import { apiCall } from './common';
import { parseOrNull } from './safeParse';
import { AnalyticsSummarySchema, VariantStatsSchema } from './schemas/variants.schema';
import { SuccessResponseSchema } from './schemas/common.schema';
import type { AnalyticsSummary, MetricEvent, VariantStats } from './types';

// Schema for getVariantMetrics response
const VariantMetricsResponseSchema = z.object({
  start_date: z.string(),
  end_date: z.string(),
  stats: z.array(VariantStatsSchema),
});

export function trackMetric(event: MetricEvent) {
  return apiCall<{ success: boolean }>('/metrics/track', {
    method: 'POST',
    body: JSON.stringify(event),
  }).then((resp) => {
    const validated = parseOrNull(SuccessResponseSchema, resp, 'TrackMetricResponse');
    if (!validated) {
      throw new Error('Invalid track metric response from API');
    }
    return validated;
  });
}

export function getMetricsSummary(startDate?: string, endDate?: string) {
  const params = new URLSearchParams();
  if (startDate) params.set('start_date', startDate);
  if (endDate) params.set('end_date', endDate);
  const query = params.toString() ? `?${params.toString()}` : '';
  return apiCall<AnalyticsSummary>(`/metrics/summary${query}`).then((resp) => {
    const validated = parseOrNull(AnalyticsSummarySchema, resp, 'AnalyticsSummary');
    if (!validated) {
      return { total_visitors: 0, variant_stats: [] };
    }
    return validated;
  });
}

export function getVariantMetrics(variantSlug?: string, startDate?: string, endDate?: string) {
  const params = new URLSearchParams();
  if (variantSlug) params.set('variant', variantSlug);
  if (startDate) params.set('start_date', startDate);
  if (endDate) params.set('end_date', endDate);
  const query = params.toString() ? `?${params.toString()}` : '';
  return apiCall<{ start_date: string; end_date: string; stats: VariantStats[] }>(`/metrics/variants${query}`).then((resp) => {
    const validated = parseOrNull(VariantMetricsResponseSchema, resp, 'VariantMetricsResponse');
    if (!validated) {
      return { start_date: '', end_date: '', stats: [] };
    }
    return validated;
  });
}
