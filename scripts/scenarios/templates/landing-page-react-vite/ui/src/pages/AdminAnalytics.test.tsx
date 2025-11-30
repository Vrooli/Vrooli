import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { AdminAnalytics } from './AdminAnalytics';
import { AuthProvider } from '../contexts/AuthContext';
import * as api from '../lib/api';

vi.mock('../lib/api');

const mockSummary = {
  total_visitors: 1250,
  total_downloads: 125,
  top_cta: 'Get Started Free',
  top_cta_ctr: 12.5,
  variant_stats: [
    {
      variant_id: 'v1',
      variant_slug: 'control',
      variant_name: 'Control',
      views: 500,
      cta_clicks: 50,
      conversions: 25,
      downloads: 12,
      conversion_rate: 5.0,
      trend: 'up' as const,
    },
    {
      variant_id: 'v2',
      variant_slug: 'variant-a',
      variant_name: 'Variant A',
      views: 750,
      cta_clicks: 90,
      conversions: 45,
      downloads: 8,
      conversion_rate: 6.0,
      trend: 'stable' as const,
    },
  ],
};

const renderWithRouter = (component: React.ReactElement) => {
  return render(
    <BrowserRouter>
      <AuthProvider>
        {component}
      </AuthProvider>
    </BrowserRouter>
  );
};

describe('AdminAnalytics [REQ:METRIC-SUMMARY,METRIC-DETAIL,METRIC-FILTER]', () => {
  const originalFetch = global.fetch;
  const originalLocation = window.location;

  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn().mockResolvedValue({ ok: false } as Response);
    delete (window as { location?: Location }).location;
    window.location = { ...originalLocation, pathname: '/admin/analytics' };

    vi.mocked(api.getMetricsSummary).mockResolvedValue(mockSummary);
  });

  afterEach(() => {
    global.fetch = originalFetch;
    window.location = originalLocation;
  });

  it('[REQ:METRIC-SUMMARY] should display total visitors metric', async () => {
    renderWithRouter(<AdminAnalytics />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-total-visitors')).toBeInTheDocument();
      expect(screen.getByText('1,250')).toBeInTheDocument();
    });
  });

  it('[REQ:METRIC-SUMMARY] should display average conversion rate across variants', async () => {
    renderWithRouter(<AdminAnalytics />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-conversion-rate')).toBeInTheDocument();
      expect(screen.getByText('5.50%')).toBeInTheDocument(); // (5.0 + 6.0) / 2
    });
  });

  it('[REQ:METRIC-SUMMARY] should display top CTA with CTR', async () => {
    renderWithRouter(<AdminAnalytics />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-top-cta')).toBeInTheDocument();
      expect(screen.getByText('Get Started Free')).toBeInTheDocument();
      expect(screen.getByText('12.5% CTR')).toBeInTheDocument();
    });
  });

  it('[REQ:METRIC-SUMMARY] should display downloads metric', async () => {
    renderWithRouter(<AdminAnalytics />);

    await waitFor(() => {
      const downloadsCard = screen.getByTestId('analytics-total-downloads');
      expect(downloadsCard).toBeInTheDocument();
      expect(within(downloadsCard).getByText('125')).toBeInTheDocument();
    });
  });

  it('[REQ:METRIC-DETAIL] should display variant performance table', async () => {
    renderWithRouter(<AdminAnalytics />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-variant-performance')).toBeInTheDocument();
      expect(screen.getByText('Control')).toBeInTheDocument();
      expect(screen.getByText('Variant A')).toBeInTheDocument();
      expect(screen.getByText('500')).toBeInTheDocument();
      expect(screen.getByText('750')).toBeInTheDocument();
      expect(screen.getByTestId('analytics-downloads-v1')).toHaveTextContent('12');
    });
  });

  it('should handle loading state', () => {
    vi.mocked(api.getMetricsSummary).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    renderWithRouter(<AdminAnalytics />);
    expect(screen.getByText('Loading analytics...')).toBeInTheDocument();
  });
});
