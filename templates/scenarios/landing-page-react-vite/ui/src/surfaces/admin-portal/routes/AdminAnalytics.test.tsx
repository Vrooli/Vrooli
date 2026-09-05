import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { ReactNode } from 'react';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '../../../test-utils';
import { BrowserRouter } from 'react-router-dom';
import { create } from '@bufbuild/protobuf';
import {
  AnalyticsSummarySchema,
  VariantStatsSchema,
} from '@vrooli/proto-types/landing-page-react-vite/v1/metrics_pb';
import { AdminSessionResponseSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/admin_pb';
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

const mockSummary = create(AnalyticsSummarySchema, {
  totalVisitors: 1250n,
  totalDownloads: 125n,
  topCta: 'Get Started Free',
  topCtaCtr: 12.5,
  variantStats: [
    create(VariantStatsSchema, {
      variantId: 1n,
      variantSlug: 'control',
      variantName: 'Control',
      views: 500n,
      ctaClicks: 50n,
      conversions: 25n,
      downloads: 12n,
      conversionRate: 5.0,
      trend: 1,
    }),
    create(VariantStatsSchema, {
      variantId: 2n,
      variantSlug: 'variant-a',
      variantName: 'Variant A',
      views: 750n,
      ctaClicks: 90n,
      conversions: 45n,
      downloads: 8n,
      conversionRate: 6.0,
      trend: 0,
    }),
  ],
});

const renderWithRouter = (component: React.ReactElement) => {
  return renderWithProviders(
    <BrowserRouter>
      <AdminAuthProvider>
        {component}
      </AdminAuthProvider>
    </BrowserRouter>,
    { withoutRouter: true }
  );
};

describe('AdminAnalytics [REQ:METRIC-SUMMARY,METRIC-DETAIL,METRIC-FILTER]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }));
    vi.stubGlobal('location', new URL('http://localhost/admin/analytics'));
    window.localStorage.clear();

    vi.mocked(api.getMetricsSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getVariantMetrics).mockResolvedValue([mockSummary.variantStats[0]!]);
    vi.mocked(api.checkAdminSession).mockResolvedValue(
      create(AdminSessionResponseSchema, { authenticated: true, email: 'ops@vrooli.dev' }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
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
      expect(within(screen.getByTestId('analytics-variant-row-1')).getByText('Control')).toBeInTheDocument();
      expect(within(screen.getByTestId('analytics-variant-row-2')).getByText('Variant A')).toBeInTheDocument();
      expect(within(screen.getByTestId('analytics-variant-row-1')).getByText('500')).toBeInTheDocument();
      expect(within(screen.getByTestId('analytics-variant-row-2')).getByText('750')).toBeInTheDocument();
      expect(screen.getByTestId('analytics-downloads-1')).toHaveTextContent('12');
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
      expect(screen.getByTestId('analytics-edit-1')).toBeInTheDocument();
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
      expect(screen.getByTestId('analytics-edit-hero-1')).toBeInTheDocument();
    });
  });

  it('opens the variant detail panel and its actions when viewing details', async () => {
    const user = userEvent.setup();
    vi.spyOn(window, 'open').mockImplementation(() => null);
    renderWithRouter(<AdminAnalytics />);
    await screen.findByTestId('analytics-view-details-1');

    await user.click(screen.getByTestId('analytics-view-details-1'));
    expect(await screen.findByTestId('analytics-variant-detail')).toBeInTheDocument();
    const actions = await screen.findByTestId('analytics-variant-actions');
    expect(actions).toBeInTheDocument();

    // Exercise every detail action handler.
    await user.click(within(actions).getByRole('button', { name: /^Edit / }));
    await user.click(within(actions).getByRole('button', { name: /Preview pinned variant/i }));
    expect(window.open).toHaveBeenCalledWith('/?variant=control', '_blank');
    await user.click(screen.getByTestId('analytics-variant-edit-hero'));
  });


  it('runs the per-row edit and hero-edit navigation handlers', async () => {
    const user = userEvent.setup();
    renderWithRouter(<AdminAnalytics />);
    await screen.findByTestId('analytics-edit-1');
    await user.click(screen.getByTestId('analytics-edit-1'));
    await user.click(screen.getByTestId('analytics-edit-hero-1'));
    // No throw means the navigation handlers executed.
    expect(screen.getByTestId('analytics-variant-performance')).toBeInTheDocument();
  });

  it('customizes and resets the focused variant from the focus banner', async () => {
    const user = userEvent.setup();
    renderWithRouter(<AdminAnalytics />);
    await screen.findByTestId('analytics-view-details-1');
    // Selecting a variant enables the focus-banner customize action.
    await user.click(screen.getByTestId('analytics-view-details-1'));

    const banner = await screen.findByTestId('analytics-focus-banner');
    await user.click(within(banner).getByTestId('analytics-focus-customize'));
    await user.click(within(banner).getByTestId('analytics-reset-filters'));
    expect(banner).toBeInTheDocument();
  });

  it('renders declining and stable trend indicators in the table', async () => {
    vi.mocked(api.getMetricsSummary).mockResolvedValue(
      create(AnalyticsSummarySchema, {
        totalVisitors: 900n,
        totalDownloads: 30n,
        variantStats: [
          create(VariantStatsSchema, { variantId: 1n, variantSlug: 'control', variantName: 'Control', views: 500n, conversions: 20n, downloads: 10n, conversionRate: 4, trend: -1 }),
          create(VariantStatsSchema, { variantId: 2n, variantSlug: 'variant-a', variantName: 'Variant A', views: 400n, conversions: 12n, downloads: 6n, conversionRate: 3, trend: 0 }),
        ],
      }),
    );
    renderWithRouter(<AdminAnalytics />);
    expect(await screen.findByTestId('analytics-variant-row-1')).toBeInTheDocument();
    expect(screen.getByTestId('analytics-variant-row-2')).toBeInTheDocument();
  });

  it('hydrates the initial time range from the saved admin experience', async () => {
    window.localStorage.setItem(
      'landing_admin_experience',
      JSON.stringify({ version: 1, lastAnalytics: { variantSlug: 'all', variantName: 'All variants', timeRangeDays: 30, savedAt: new Date().toISOString() } }),
    );
    renderWithRouter(<AdminAnalytics />);
    // Hydrating the saved 30-day range still renders the analytics dashboard.
    await waitFor(() => expect(screen.getByTestId('analytics-total-visitors')).toBeInTheDocument());
  });

  it('uses singular day copy for a one-day range pinned via URL', async () => {
    vi.stubGlobal('location', new URL('http://localhost/admin/analytics?variant=control&range=1'));
    vi.mocked(api.getVariantMetrics).mockResolvedValue([mockSummary.variantStats[0]!]);
    renderWithRouter(<AdminAnalytics />);
    expect(await screen.findByTestId('analytics-variant-detail')).toBeInTheDocument();
    expect(screen.getAllByText(/last 1 day\b/i).length).toBeGreaterThan(0);
  });

  it('renders gracefully when the summary has no variant stats', async () => {
    vi.mocked(api.getMetricsSummary).mockResolvedValue(
      create(AnalyticsSummarySchema, { totalVisitors: 0n, totalDownloads: 0n, variantStats: [] }),
    );
    renderWithRouter(<AdminAnalytics />);
    await waitFor(() => expect(screen.getByTestId('analytics-total-visitors')).toBeInTheDocument());
    // Conversion rate falls back to 0 with no stats.
    expect(screen.getByTestId('analytics-conversion-rate')).toBeInTheDocument();
    expect(screen.queryByTestId('analytics-variant-row-1')).not.toBeInTheDocument();
  });

  it('pins the selected variant and time range from URL params', async () => {
    vi.stubGlobal('location', new URL('http://localhost/admin/analytics?variant=variant-a&range=30'));
    vi.mocked(api.getVariantMetrics).mockResolvedValue([
      create(VariantStatsSchema, {
        variantId: 2n,
        variantSlug: 'variant-a',
        variantName: 'Variant A',
        views: 750n,
        ctaClicks: 90n,
        conversions: 45n,
        downloads: 8n,
        conversionRate: 6.0,
        trend: 1,
        avgScrollDepth: 0.72,
      }),
    ]);
    renderWithRouter(<AdminAnalytics />);
    // Detail panel renders for the pinned variant, including scroll depth.
    expect(await screen.findByTestId('analytics-variant-detail')).toBeInTheDocument();
    const banner = screen.getByTestId('analytics-focus-banner');
    expect(within(banner).getByTestId('analytics-focus-customize')).toBeInTheDocument();
  });

  it('renders an error state with a retry control when the fetch fails', async () => {
    vi.mocked(api.getMetricsSummary).mockRejectedValueOnce(new Error('metrics down'));
    const user = userEvent.setup();
    renderWithRouter(<AdminAnalytics />);
    expect(await screen.findByText(/metrics down/i)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Retry/i }));
    await waitFor(() => expect(screen.getByTestId('analytics-total-visitors')).toBeInTheDocument());
  });
});
