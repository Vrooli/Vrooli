import { beforeEach, describe, expect, it, vi } from 'vitest';
import { getMetricsSummary, getVariantMetrics, trackMetric } from './metrics';
import { apiCall } from './common';

vi.mock('./common', () => ({ apiCall: vi.fn() }));

const mockApiCall = vi.mocked(apiCall);

describe('metrics API', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('tracks a valid metric and accepts the API acknowledgement shape', async () => {
    mockApiCall.mockResolvedValueOnce({ success: true });
    await expect(trackMetric({ event_type: 'click', variant_slug: 'control', session_id: 'session-1' })).resolves.toEqual({ success: true });
    expect(mockApiCall).toHaveBeenCalledWith('/metrics/track', expect.objectContaining({ method: 'POST' }));

    mockApiCall.mockResolvedValueOnce({});
    await expect(trackMetric({ event_type: 'click', variant_slug: 'control', session_id: 'session-1' })).resolves.toEqual({});
  });

  it('serializes summary date filters and preserves a valid analytics response', async () => {
    const response = { total_visitors: 12, variant_stats: [] };
    mockApiCall.mockResolvedValueOnce(response);
    await expect(getMetricsSummary('2026-01-01', '2026-01-31')).resolves.toEqual(response);
    expect(mockApiCall).toHaveBeenCalledWith('/metrics/summary?start_date=2026-01-01&end_date=2026-01-31');
  });

  it('uses a safe empty summary when the API payload fails validation', async () => {
    mockApiCall.mockResolvedValueOnce({ total_visitors: 'invalid' });
    await expect(getMetricsSummary()).resolves.toEqual({ total_visitors: 0, variant_stats: [] });
    expect(mockApiCall).toHaveBeenCalledWith('/metrics/summary');
  });

  it('serializes variant filters and uses an empty result for malformed payloads', async () => {
    mockApiCall.mockResolvedValueOnce({ start_date: '2026-01-01', end_date: '2026-01-31', stats: [] });
    await expect(getVariantMetrics('control', '2026-01-01', '2026-01-31')).resolves.toEqual({ start_date: '2026-01-01', end_date: '2026-01-31', stats: [] });
    expect(mockApiCall).toHaveBeenCalledWith('/metrics/variants?variant=control&start_date=2026-01-01&end_date=2026-01-31');

    mockApiCall.mockResolvedValueOnce({});
    await expect(getVariantMetrics()).resolves.toEqual({ start_date: '', end_date: '', stats: [] });
  });
});
