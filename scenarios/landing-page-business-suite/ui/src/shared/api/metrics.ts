import { createClient } from '@connectrpc/connect';
import type { JsonObject } from '@bufbuild/protobuf';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import {
  MetricsService,
  type AnalyticsSummary as GeneratedAnalyticsSummary,
  type VariantStats as GeneratedVariantStats,
} from '@vrooli/proto-types/landing-page-business-suite/metrics_pb';
import { CONNECT_API_BASE } from './common';
import type { AnalyticsSummary, MetricEvent, VariantStats } from './types';

const metricsClient = createClient(MetricsService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

function metricDataToProto(value?: Record<string, unknown>): JsonObject | undefined {
  if (value === undefined) return undefined;
  const encoded = JSON.stringify(value);
  return encoded === undefined ? undefined : JSON.parse(encoded) as JsonObject;
}

function variantStatsFromProto(value: GeneratedVariantStats): VariantStats {
  return {
    variant_id: Number(value.variantId),
    variant_slug: value.variantSlug,
    variant_name: value.variantName,
    views: Number(value.views),
    cta_clicks: Number(value.ctaClicks),
    conversions: Number(value.conversions),
    downloads: Number(value.downloads),
    conversion_rate: value.conversionRate,
    ...(value.avgScrollDepth !== undefined ? { avg_scroll_depth: value.avgScrollDepth } : {}),
  };
}

function analyticsSummaryFromProto(value: GeneratedAnalyticsSummary): AnalyticsSummary {
  return {
    total_visitors: Number(value.totalVisitors),
    total_downloads: Number(value.totalDownloads),
    variant_stats: value.variantStats.map(variantStatsFromProto),
    ...(value.topCta !== undefined ? { top_cta: value.topCta } : {}),
    ...(value.topCtaCtr !== undefined ? { top_cta_ctr: value.topCtaCtr } : {}),
  };
}

export async function trackMetric(event: MetricEvent): Promise<{ success: boolean }> {
  const response = await metricsClient.trackEvent({
    eventType: event.event_type,
    variantSlug: event.variant_slug,
    sessionId: event.session_id,
    visitorId: event.visitor_id ?? '',
    eventData: metricDataToProto(event.event_data),
  });
  return { success: response.success };
}

export async function getMetricsSummary(startDate?: string, endDate?: string): Promise<AnalyticsSummary> {
  const response = await metricsClient.getAnalyticsSummary({ startDate: startDate ?? '', endDate: endDate ?? '' });
  return analyticsSummaryFromProto(response);
}

export async function getVariantMetrics(variantSlug?: string, startDate?: string, endDate?: string): Promise<{ start_date: string; end_date: string; stats: VariantStats[] }> {
  const response = await metricsClient.getVariantStats({ variant: variantSlug ?? '', startDate: startDate ?? '', endDate: endDate ?? '' });
  return { start_date: response.startDate, end_date: response.endDate, stats: response.stats.map(variantStatsFromProto) };
}
