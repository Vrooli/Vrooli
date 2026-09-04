import { beforeEach, describe, expect, it, vi } from 'vitest';

const metricsClient = vi.hoisted(() => ({ trackEvent: vi.fn(), getAnalyticsSummary: vi.fn(), getVariantStats: vi.fn(), getTrafficBreakdown: vi.fn(), getTrafficSeries: vi.fn() }));
vi.mock('@connectrpc/connect', () => ({ createClient: vi.fn(() => metricsClient) }));

import { getMetricsSummary, getTrafficBreakdown, getTrafficSeries, getVariantMetrics, trackMetric } from './metrics';

describe('metrics API', () => {
  beforeEach(() => vi.clearAllMocks());

  it('tracks a metric through the generated contract', async () => {
    metricsClient.trackEvent.mockResolvedValueOnce({ success: true });

    await expect(trackMetric({ event_type: 'click', variant_slug: 'control', session_id: 'session-1', event_data: { element: 'hero', depth: 50 } })).resolves.toEqual({ success: true });
    expect(metricsClient.trackEvent).toHaveBeenCalledWith({
      eventType: 'click', variantSlug: 'control', sessionId: 'session-1', visitorId: '', eventId: '', utmSource: '', utmMedium: '', utmCampaign: '', landingPath: '', referrer: '', eventData: { element: 'hero', depth: 50 },
    });
  });

  it('maps generated summary fields to the established UI shape', async () => {
    metricsClient.getAnalyticsSummary.mockResolvedValueOnce({
      totalVisitors: 12n, totalDownloads: 3n, variantStats: [], topCta: 'hero', topCtaCtr: 12.5,
    });

    await expect(getMetricsSummary('2026-01-01', '2026-01-31')).resolves.toEqual({
      total_visitors: 12, total_downloads: 3, variant_stats: [], top_cta: 'hero', top_cta_ctr: 12.5,
    });
    expect(metricsClient.getAnalyticsSummary).toHaveBeenCalledWith({ startDate: '2026-01-01', endDate: '2026-01-31' });
  });

  it('maps generated variant statistics and optional filters', async () => {
    metricsClient.getVariantStats.mockResolvedValueOnce({
      startDate: '2026-01-01', endDate: '2026-01-31', stats: [{
        variantId: 7n, variantSlug: 'control', variantName: 'Control', views: 100n,
        ctaClicks: 10n, conversions: 4n, downloads: 2n, conversionRate: 4, avgScrollDepth: 76.5,
      }],
    });

    await expect(getVariantMetrics('control', '2026-01-01', '2026-01-31')).resolves.toEqual({
      start_date: '2026-01-01', end_date: '2026-01-31', stats: [{
        variant_id: 7, variant_slug: 'control', variant_name: 'Control', views: 100,
        cta_clicks: 10, conversions: 4, downloads: 2, exposures: 0, conversion_rate: 4, avg_scroll_depth: 76.5,
      }],
    });
    expect(metricsClient.getVariantStats).toHaveBeenCalledWith({ variant: 'control', startDate: '2026-01-01', endDate: '2026-01-31' });
  });

  it('maps a dimensional breakdown and uses the generated enum contract', async () => {
    metricsClient.getTrafficBreakdown.mockResolvedValueOnce({
      rows: [{ key: 'US', label: 'United States', sessions: 8n, conversions: 2n, revenueMinor: 1200n, share: 1 }],
      totalSessions: 8n, exhaustive: true, currency: 'usd',
    });

    await expect(getTrafficBreakdown('country', '2026-01-01', '2026-01-31', 10)).resolves.toMatchObject({
      rows: [{ key: 'US', sessions: 8, conversions: 2, revenue_minor: 1200, share: 1 }],
      total_sessions: 8, exhaustive: true, currency: 'usd',
    });
    expect(metricsClient.getTrafficBreakdown).toHaveBeenCalledWith({ dimension: 1, startDate: '2026-01-01', endDate: '2026-01-31', limit: 10 });
  });

  it('maps real series points without inventing client-side samples', async () => {
    metricsClient.getTrafficSeries.mockResolvedValueOnce({
      points: [{ bucketStart: '2026-01-02T00:00:00Z', value: 7 }], unit: 'count',
    });

    await expect(getTrafficSeries('visitors', '2026-01-01', '2026-01-31')).resolves.toEqual({
      points: [{ bucket_start: '2026-01-02T00:00:00Z', value: 7 }], unit: 'count',
    });
  });
});
