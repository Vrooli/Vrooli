import { createClient } from '@connectrpc/connect';
import type { JsonObject, JsonValue } from '@bufbuild/protobuf';
import { createScenarioConnectTransport } from '@vrooli/api-base';
import {
  MetricsService,
  TrafficDimension,
  type AnalyticsSummary as GeneratedAnalyticsSummary,
  type VariantStats as GeneratedVariantStats,
} from '@vrooli/proto-types/landing-page-business-suite/v1/metrics_pb';
import { apiGet, CONNECT_API_BASE } from './common';
import type { AnalyticsSummary, MetricEvent, VariantStats } from './types';
import { normalizeTimestamp } from '../lib/protobuf-utils';

const metricsClient = createClient(MetricsService, createScenarioConnectTransport({ baseUrl: CONNECT_API_BASE }));

function metricDataToProto(value?: Record<string, unknown>): JsonObject | undefined {
  if (value === undefined) return undefined;
  const encoded = JSON.stringify(value);
  const parsed: unknown = JSON.parse(encoded);
  if (!isJsonObject(parsed)) {
    throw new Error('Metric event data must serialize to a JSON object');
  }
  return parsed;
}

function isJsonValue(value: unknown): value is JsonValue {
  if (value === null || typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') return true;
  if (Array.isArray(value)) return value.every(isJsonValue);
  return isJsonObject(value);
}

function isJsonObject(value: unknown): value is JsonObject {
  return value !== null
    && typeof value === 'object'
    && !Array.isArray(value)
    && Object.values(value).every(isJsonValue);
}

function variantStatsFromProto(value: GeneratedVariantStats): VariantStats {
  // The checked-in generated package is updated with the proto; this narrow
  // intersection keeps an older local file-based install source-compatible
  // until its package link is refreshed.
  const exposures = Number((value as unknown as { exposures?: bigint }).exposures ?? 0n);
  return {
    variant_id: Number(value.variantId),
    variant_slug: value.variantSlug,
    variant_name: value.variantName,
    views: Number(value.views),
    cta_clicks: Number(value.ctaClicks),
    conversions: Number(value.conversions),
    downloads: Number(value.downloads),
    exposures,
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
    eventId: event.event_id ?? '',
    utmSource: event.utm_source ?? '',
    utmMedium: event.utm_medium ?? '',
    utmCampaign: event.utm_campaign ?? '',
    landingPath: event.landing_path ?? '',
    referrer: event.referrer ?? '',
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

export type TrafficDimensionKey = 'country' | 'referrer_kind' | 'utm_source' | 'utm_campaign' | 'device_class' | 'landing_path' | 'variant';

export interface TrafficBreakdownRow {
  key: string;
  label: string;
  sessions: number;
  conversions: number;
  revenue_minor: number;
  share: number;
}

export interface TrafficBreakdown {
  rows: TrafficBreakdownRow[];
  total_sessions: number;
  exhaustive: boolean;
  currency: string;
  observed_at?: string;
}

export interface TrafficSeriesPoint {
  bucket_start: string;
  value: number;
}

export interface TrafficSeries {
  points: TrafficSeriesPoint[];
  unit: string;
  observed_at?: string;
}

const TRAFFIC_DIMENSIONS: Record<TrafficDimensionKey, TrafficDimension> = {
  country: TrafficDimension.COUNTRY,
  referrer_kind: TrafficDimension.REFERRER_KIND,
  utm_source: TrafficDimension.UTM_SOURCE,
  utm_campaign: TrafficDimension.UTM_CAMPAIGN,
  device_class: TrafficDimension.DEVICE_CLASS,
  landing_path: TrafficDimension.LANDING_PATH,
  variant: TrafficDimension.VARIANT,
};

export async function getTrafficBreakdown(
  dimension: TrafficDimensionKey,
  startDate?: string,
  endDate?: string,
  limit = 10,
): Promise<TrafficBreakdown> {
  const response = await metricsClient.getTrafficBreakdown({
    dimension: TRAFFIC_DIMENSIONS[dimension],
    startDate: startDate ?? '',
    endDate: endDate ?? '',
    limit,
  });
  return {
    rows: response.rows.map((row) => ({
      key: row.key,
      label: row.label,
      sessions: Number(row.sessions),
      conversions: Number(row.conversions),
      revenue_minor: Number(row.revenueMinor),
      share: row.share,
    })),
    total_sessions: Number(response.totalSessions),
    exhaustive: response.exhaustive,
    currency: response.currency,
    ...(response.observedAt ? { observed_at: normalizeTimestamp(response.observedAt) } : {}),
  };
}

export async function getTrafficSeries(
  metric: 'visitors' | 'sessions' | 'conversions',
  startDate?: string,
  endDate?: string,
  bucket = 'day',
): Promise<TrafficSeries> {
  const response = await metricsClient.getTrafficSeries({ metric, startDate: startDate ?? '', endDate: endDate ?? '', bucket });
  return {
    points: response.points.map((point) => ({ bucket_start: point.bucketStart, value: point.value })),
    unit: response.unit,
    ...(response.observedAt ? { observed_at: normalizeTimestamp(response.observedAt) } : {}),
  };
}

export interface AdminRevenue {
  mrr: number; mrr_unit: string; today: number; today_unit: string;
  currency: string; sample_size: number; observed_at: string | null;
}

export async function getAdminRevenue(): Promise<AdminRevenue> {
  return apiGet<AdminRevenue>('/admin/dashboard/revenue');
}

export interface AdminRevenueSummary {
  currency: string;
  mrr_unit: string;
  revenue_today_unit: string;
  revenue_window_unit: string;
  credit_unit: string;
  currency_excluded_count: number;
  mrr_minor: number;
  revenue_today_minor: number;
  revenue_window_minor: number;
  active_subscriptions: number;
  subscriptions_churned_window: number;
  churn_rate_percent: number;
  credit_balance_total: number;
  credit_burned_window: number;
  usage_records_window: number;
  sample_size: number;
  trials_without_payment_method: number;
  observed_at: string | null;
}

export async function getAdminRevenueSummary(): Promise<AdminRevenueSummary> {
  return apiGet<AdminRevenueSummary>('/admin/dashboard/revenue/summary');
}
