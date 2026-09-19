import type { JsonObject } from '@bufbuild/protobuf';
import { createClient } from '@connectrpc/connect';
import { MetricsService } from '@vrooli/proto-types/landing-page-react-vite/v1/metrics_pb';
import type {
  AnalyticsSummary,
  VariantStats,
} from '@vrooli/proto-types/landing-page-react-vite/v1/metrics_pb';

import { transport } from './client';

const metricsClient = createClient(MetricsService, transport);

export interface TrackMetricInput {
  eventType: string;
  variantId: bigint;
  sessionId: string;
  visitorId?: string;
  eventId?: string;
  // Dynamic JSON payload (proto google.protobuf.Struct); bridged at the boundary.
  eventData?: Record<string, unknown>;
}

/** Records a visitor analytics event. */
export async function trackMetric(event: TrackMetricInput): Promise<boolean> {
  const resp = await metricsClient.trackEvent({
    eventType: event.eventType,
    variantId: event.variantId,
    sessionId: event.sessionId,
    visitorId: event.visitorId ?? '',
    eventId: event.eventId ?? '',
    eventData: event.eventData as JsonObject | undefined,
  });
  return resp.success;
}

/** Fetches the aggregate analytics summary across variants (admin). */
export function getMetricsSummary(startDate?: string, endDate?: string): Promise<AnalyticsSummary> {
  return metricsClient.getAnalyticsSummary({
    startDate: startDate ?? '',
    endDate: endDate ?? '',
  });
}

/** Fetches per-variant analytics stats (admin). */
export async function getVariantMetrics(
  variant?: string,
  startDate?: string,
  endDate?: string,
): Promise<VariantStats[]> {
  const resp = await metricsClient.getVariantStats({
    variant: variant ?? '',
    startDate: startDate ?? '',
    endDate: endDate ?? '',
  });
  return resp.stats;
}

export type { AnalyticsSummary, VariantStats };
