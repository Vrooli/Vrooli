import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { ReactNode } from 'react';
import { render, screen, waitFor, within } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { AdminAnalytics } from './AdminAnalytics';
import { AdminAuthProvider } from '../../../app/providers/AdminAuthProvider';
import * as api from '../../../shared/api';

vi.mock('../../../shared/api');
vi.mock('../components/RuntimeSignalStrip', () => ({
  RuntimeSignalStrip: () => <div data-testid="runtime-signal-mock" />,
}));
vi.mock('../../../app/providers/LandingVariantProvider', () => ({
  useLandingVariant: () => ({
    variant: { slug: 'control', name: 'Control' },
    config: { sections: [], downloads: [], fallback: false },
    loading: false,
    error: null,
    resolution: 'api_select',
    statusNote: null,
    lastUpdated: Date.now(),
    refresh: vi.fn(),
  }),
  LandingVariantProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

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
      <AdminAuthProvider>
        {component}
      </AdminAuthProvider>
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
    window.localStorage.clear();

    vi.mocked(api.getMetricsSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.checkAdminSession).mockResolvedValue({ authenticated: true, email: 'ops@vrooli.dev' });
  });

  afterEach(() => {
    global.fetch = originalFetch;
    window.location = originalLocation;
    window.localStorage.clear();
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
      expect(within(screen.getByTestId('analytics-variant-row-v1')).getByText('Control')).toBeInTheDocument();
      expect(within(screen.getByTestId('analytics-variant-row-v2')).getByText('Variant A')).toBeInTheDocument();
      expect(within(screen.getByTestId('analytics-variant-row-v1')).getByText('500')).toBeInTheDocument();
      expect(within(screen.getByTestId('analytics-variant-row-v2')).getByText('750')).toBeInTheDocument();
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

  it('should render customize shortcut per variant row', async () => {
    renderWithRouter(<AdminAnalytics />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-edit-v1')).toBeInTheDocument();
    });
  });

  it('should persist recent analytics filters to localStorage', async () => {
    renderWithRouter(<AdminAnalytics />);

    await waitFor(() => {
      const raw = window.localStorage.getItem('landing_admin_experience');
      expect(raw).toBeTruthy();
      const snapshot = JSON.parse(raw ?? '{}');
      expect(snapshot.lastAnalytics).toBeTruthy();
    });
  });

  it('surfaces focus banner with current view context', async () => {
    renderWithRouter(<AdminAnalytics />);

    await waitFor(() => {
      const banner = screen.getByTestId('analytics-focus-banner');
      expect(banner).toBeInTheDocument();
      expect(within(banner).getByText(/Analyzing all variants/i)).toBeInTheDocument();
      expect(within(banner).getByText(/Time range: Last 7 days/)).toBeInTheDocument();
    });
  });

  it('provides hero edit actions from analytics table', async () => {
    renderWithRouter(<AdminAnalytics />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-edit-hero-v1')).toBeInTheDocument();
    });
  });
});
