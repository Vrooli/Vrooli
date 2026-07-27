import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderWithProviders as render } from "../../../test-utils/renderWithProviders";
import type { ReactNode } from 'react';
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { BrowserRouter } from 'react-router-dom';
import { AdminAnalytics } from './AdminAnalytics';
import { AdminAuthProvider } from '../../../app/providers/AdminAuthProvider';
import * as api from '../../../shared/api';
import { isRecord, safeParseJson } from '../../../shared/lib/utils';

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
      variant_id: 1,
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
      variant_id: 2,
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

const renderWithRouter = (component: React.ReactElement) =>
  render(
    <BrowserRouter>
      <AdminAuthProvider>
        {component}
      </AdminAuthProvider>
    </BrowserRouter>
  );

const renderWithAuth = async (component: React.ReactElement) => {
  const utils = renderWithRouter(component);
  await waitFor(() => { expect(vi.mocked(api.checkAdminSession)).toHaveBeenCalled(); });
  return utils;
};

describe('AdminAnalytics [REQ:METRIC-SUMMARY,METRIC-DETAIL,METRIC-FILTER]', () => {
  const originalFetch = globalThis.fetch;
  const originalLocation = window.location;

  beforeEach(() => {
    vi.clearAllMocks();
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: false } as Response);
    window.history.replaceState({}, '', '/admin/analytics');
    window.localStorage.clear();
    vi.stubGlobal('open', vi.fn());

    vi.mocked(api.getMetricsSummary).mockResolvedValue(mockSummary);
    vi.mocked(api.getVariantMetrics).mockResolvedValue({ start_date: '2026-01-01', end_date: '2026-01-07', stats: [mockSummary.variant_stats[0]!] });
    vi.mocked(api.checkAdminSession).mockResolvedValue({ authenticated: true, email: 'ops@vrooli.dev' });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.history.replaceState({}, '', `${originalLocation.pathname}${originalLocation.search}`);
    window.localStorage.clear();
    vi.unstubAllGlobals();
  });

  it('[REQ:METRIC-SUMMARY] should display total visitors metric', async () => {
    await renderWithAuth(<AdminAnalytics />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-total-visitors')).toBeInTheDocument();
      expect(screen.getByText('1,250')).toBeInTheDocument();
    });
  });

  it('[REQ:METRIC-SUMMARY] should display average conversion rate across variants', async () => {
    await renderWithAuth(<AdminAnalytics />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-conversion-rate')).toBeInTheDocument();
      expect(screen.getByText('5.50%')).toBeInTheDocument(); // (5.0 + 6.0) / 2
    });
  });

  it('[REQ:METRIC-SUMMARY] should display top CTA with CTR', async () => {
    await renderWithAuth(<AdminAnalytics />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-top-cta')).toBeInTheDocument();
      expect(screen.getByText('Get Started Free')).toBeInTheDocument();
      expect(screen.getByText('12.5% CTR')).toBeInTheDocument();
    });
  });

  it('[REQ:METRIC-SUMMARY] should display downloads metric', async () => {
    await renderWithAuth(<AdminAnalytics />);

    await waitFor(() => {
      const downloadsCard = screen.getByTestId('analytics-total-downloads');
      expect(downloadsCard).toBeInTheDocument();
      expect(within(downloadsCard).getByText('125')).toBeInTheDocument();
    });
  });

  it('[REQ:METRIC-DETAIL] should display variant performance table', async () => {
    await renderWithAuth(<AdminAnalytics />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-variant-performance')).toBeInTheDocument();
      expect(within(screen.getByTestId('analytics-variant-row-1')).getByText('Control')).toBeInTheDocument();
      expect(within(screen.getByTestId('analytics-variant-row-2')).getByText('Variant A')).toBeInTheDocument();
      expect(within(screen.getByTestId('analytics-variant-row-1')).getByText('500')).toBeInTheDocument();
      expect(within(screen.getByTestId('analytics-variant-row-2')).getByText('750')).toBeInTheDocument();
      expect(screen.getByTestId('analytics-downloads-1')).toHaveTextContent('12');
    });
  });

  it('should handle loading state', async () => {
    vi.mocked(api.getMetricsSummary).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );

    await renderWithAuth(<AdminAnalytics />);
    expect(screen.getByText('Loading analytics...')).toBeInTheDocument();
  });

  it('should render customize shortcut per variant row', async () => {
    await renderWithAuth(<AdminAnalytics />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-edit-1')).toBeInTheDocument();
    });
  });

  it('should persist recent analytics filters to localStorage', async () => {
    await renderWithAuth(<AdminAnalytics />);

    await waitFor(() => {
      const raw = window.localStorage.getItem('landing_admin_experience');
      expect(raw).toBeTruthy();
      const parsed = safeParseJson(raw ?? '{}');
      expect(isRecord(parsed) ? parsed.lastAnalytics : undefined).toBeTruthy();
    });
  });

  it('surfaces focus banner with current view context', async () => {
    await renderWithAuth(<AdminAnalytics />);

    await waitFor(() => {
      const banner = screen.getByTestId('analytics-focus-banner');
      expect(banner).toBeInTheDocument();
      expect(within(banner).getByText(/Analyzing all variants/i)).toBeInTheDocument();
      expect(within(banner).getByText(/Time range: Last 7 days/)).toBeInTheDocument();
    });
  });

  it('provides hero edit actions from analytics table', async () => {
    await renderWithAuth(<AdminAnalytics />);

    await waitFor(() => {
      expect(screen.getByTestId('analytics-edit-hero-1')).toBeInTheDocument();
    });
  });

  it('opens detail analytics and drives the table navigation actions', async () => {
    await renderWithAuth(<AdminAnalytics />);

    const details = await screen.findByTestId('analytics-view-details-1');
    fireEvent.click(details);
    await waitFor(() => {
      expect(screen.getByTestId('analytics-variant-detail')).toBeInTheDocument();
      expect(screen.getByText('Detailed Variant Stats')).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: 'Back to All Variants' }));
    await waitFor(() => { expect(screen.queryByTestId('analytics-variant-detail')).not.toBeInTheDocument(); });
    fireEvent.click(screen.getByTestId('analytics-edit-1'));
    fireEvent.click(screen.getByTestId('analytics-edit-hero-1'));
  });

  it('filters to a variant and exposes focus, preview, shortcut, and detail actions', async () => {
    await renderWithAuth(<AdminAnalytics />);

    fireEvent.click(await screen.findByTestId('analytics-variant-filter'));
    fireEvent.click(await screen.findByRole('option', { name: 'Variant A' }));

    await waitFor(() => {
      expect(screen.getByTestId('analytics-variant-detail')).toBeInTheDocument();
      expect(screen.getByTestId('analytics-reset-filters')).toBeInTheDocument();
      expect(screen.getByTestId('analytics-focus-customize')).toBeInTheDocument();
      expect(screen.getByTestId('analytics-focus-preview')).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId('analytics-focus-preview'));
    fireEvent.click(screen.getByTestId('analytics-reset-filters'));

    await waitFor(() => expect(screen.queryByTestId('analytics-variant-detail')).not.toBeInTheDocument());
    const shortcuts = screen.getByTestId('analytics-shortcuts');
    fireEvent.click(within(shortcuts).getByRole('button', { name: 'Focus analytics' }));
    fireEvent.click(within(shortcuts).getByRole('button', { name: 'View breakdown' }));
    fireEvent.click(within(shortcuts).getByRole('button', { name: 'Inspect metrics' }));
  });

  it('changes time ranges and renders empty analytics safely', async () => {
    vi.mocked(api.getMetricsSummary).mockResolvedValue({ ...mockSummary, variant_stats: [], total_downloads: undefined, top_cta: undefined, top_cta_ctr: undefined });
    await renderWithAuth(<AdminAnalytics />);

    expect(await screen.findByText('No variant data available yet')).toBeInTheDocument();
    expect(screen.getByText('No data yet')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('analytics-time-range'));
    fireEvent.click(await screen.findByRole('option', { name: 'Last 24 hours' }));
    await waitFor(() => expect(vi.mocked(api.getMetricsSummary).mock.calls.length).toBeGreaterThan(1));
  });
});
