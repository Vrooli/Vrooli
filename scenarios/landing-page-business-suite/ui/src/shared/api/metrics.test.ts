import { beforeEach, describe, expect, it, vi } from 'vitest';

const metricsClient = vi.hoisted(() => ({ trackEvent: vi.fn(), getAnalyticsSummary: vi.fn(), getVariantStats: vi.fn() }));
vi.mock('@connectrpc/connect', () => ({ createClient: vi.fn(() => metricsClient) }));

import { getMetricsSummary, getVariantMetrics, trackMetric } from './metrics';

describe('metrics API', () => {
  beforeEach(() => vi.clearAllMocks());

  it('tracks a metric through the generated contract', async () => {
    metricsClient.trackEvent.mockResolvedValueOnce({ success: true });

    await expect(trackMetric({ event_type: 'click', variant_slug: 'control', session_id: 'session-1', event_data: { element: 'hero', depth: 50 } })).resolves.toEqual({ success: true });
    expect(metricsClient.trackEvent).toHaveBeenCalledWith({
      eventType: 'click', variantSlug: 'control', sessionId: 'session-1', visitorId: '', eventData: { element: 'hero', depth: 50 },
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
        cta_clicks: 10, conversions: 4, downloads: 2, conversion_rate: 4, avg_scroll_depth: 76.5,
      }],
    });
    expect(metricsClient.getVariantStats).toHaveBeenCalledWith({ variant: 'control', startDate: '2026-01-01', endDate: '2026-01-31' });
  });
});
