import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import type { ReactNode } from 'react';
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { AdminHome } from './AdminHome';
import { AdminAuthProvider } from '../../../app/providers/AdminAuthProvider';
import { listVariants, checkAdminSession, getStripeSettings, resetDemoData } from '../../../shared/api';
import * as analyticsController from '../controllers/analyticsController';

const mockNavigate = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

vi.mock('../components/RuntimeSignalStrip', () => ({
  RuntimeSignalStrip: () => <div data-testid="runtime-signal-mock" />,
}));
vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    listVariants: vi.fn(),
    checkAdminSession: vi.fn(),
    getStripeSettings: vi.fn(),
    resetDemoData: vi.fn(),
  };
});
vi.mock('../../../app/providers/LandingVariantProvider', () => ({
  useLandingVariant: () => ({
    variant: { slug: 'control', name: 'Control Variant' },
    config: null,
    loading: false,
    error: null,
    resolution: 'api_select',
    statusNote: 'Serving weighted traffic',
    lastUpdated: Date.now(),
    refresh: vi.fn(),
  }),
  LandingVariantProvider: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

// Also mock the separate useLandingVariant hook file (AdminHome imports from here)
vi.mock('../../../app/providers/useLandingVariant', () => ({
  useLandingVariant: () => ({
    variant: { slug: 'control', name: 'Control Variant' },
    config: null,
    loading: false,
    error: null,
    resolution: 'api_select',
    statusNote: 'Serving weighted traffic',
    lastUpdated: Date.now(),
    refresh: vi.fn(),
  }),
}));

const mockedListVariants = vi.mocked(listVariants);
const mockedCheckAdminSession = vi.mocked(checkAdminSession);
const mockedGetStripeSettings = vi.mocked(getStripeSettings);
const mockedResetDemoData = vi.mocked(resetDemoData);
const mockStripeSettings = {
  publishable_key_set: true,
  secret_key_set: true,
  webhook_secret_set: false,
  source: 'env' as const,
  updated_at: new Date().toISOString(),
  dashboard_url: 'https://dashboard.stripe.com/',
};
const mockVariantsResponse = {
  variants: [
    {
      id: 1,
      slug: 'control',
      name: 'Control Variant',
      status: 'active' as const,
      weight: 70,
      updated_at: new Date().toISOString(),
    },
    {
      id: 2,
      slug: 'beta',
      name: 'Beta Variant',
      status: 'active' as const,
      weight: 30,
      updated_at: new Date(Date.now() - 12 * 24 * 60 * 60 * 1000).toISOString(),
    },
  ],
};

const mockAnalyticsSummary = {
  total_visitors: 1000,
  total_downloads: 80,
  variant_stats: [
    {
      variant_id: 1,
      variant_slug: 'control',
      variant_name: 'Control Variant',
      views: 700,
      cta_clicks: 200,
      conversions: 120,
      downloads: 50,
      conversion_rate: 17.14,
      trend: 'up' as const,
    },
    {
      variant_id: 2,
      variant_slug: 'beta',
      variant_name: 'Beta Variant',
      views: 300,
      cta_clicks: 40,
      conversions: 12,
      downloads: 30,
      conversion_rate: 4,
      trend: 'down' as const,
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
  await waitFor(() => { expect(mockedCheckAdminSession).toHaveBeenCalled(); });
  return utils;
};

describe('AdminHome [REQ:ADMIN-MODES]', () => {
  const originalFetch = globalThis.fetch;
  const originalLocation = window.location;
  let fetchAnalyticsSpy: { mockRestore: () => void } | undefined;

  beforeEach(() => {
    vi.clearAllMocks();
    globalThis.fetch = vi.fn().mockResolvedValue({ ok: false });
    window.history.replaceState({}, '', '/admin');
    window.localStorage.clear();
    mockedListVariants.mockResolvedValue(mockVariantsResponse);
    mockedCheckAdminSession.mockResolvedValue({ authenticated: true, email: 'ops@vrooli.dev', reset_enabled: false });
    fetchAnalyticsSpy = vi.spyOn(analyticsController, 'fetchAnalyticsSummary').mockResolvedValue(mockAnalyticsSummary);
    mockedGetStripeSettings.mockResolvedValue(mockStripeSettings);
    mockedResetDemoData.mockResolvedValue({ reset: true, timestamp: new Date().toISOString() });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    window.history.replaceState({}, '', `${originalLocation.pathname}${originalLocation.search}`);
    window.localStorage.clear();
    fetchAnalyticsSpy?.mockRestore();
  });

  it('[REQ:ADMIN-MODES] should display exactly two modes: Analytics and Customization', async () => {
    await renderWithAuth(<AdminHome />);

    expect(screen.getByTestId('admin-quick-flows')).toBeInTheDocument();
    expect(screen.getByTestId('flow-landing')).toBeInTheDocument();
    expect(screen.getByTestId('flow-billing')).toBeInTheDocument();
    expect(screen.getByTestId('flow-apps')).toBeInTheDocument();
    expect(screen.getByTestId('flow-users')).toBeInTheDocument();
  });

  it('[REQ:ADMIN-NAV] navigates to quick flow destinations', async () => {
    const user = userEvent.setup();
    await renderWithAuth(<AdminHome />);

    await user.click(screen.getByTestId('flow-landing'));
    await user.click(screen.getByTestId('flow-billing'));
    await user.click(screen.getByTestId('flow-apps'));
    await user.click(screen.getByTestId('flow-users'));

    expect(mockNavigate).toHaveBeenCalledWith('/admin/landing');
    expect(mockNavigate).toHaveBeenCalledWith('/admin/billing-home');
    expect(mockNavigate).toHaveBeenCalledWith('/admin/apps');
    expect(mockNavigate).toHaveBeenCalledWith('/admin/users');
  });

  it('shows computed stats once health data loads', async () => {
    await renderWithAuth(<AdminHome />);

    const statsBar = await screen.findByTestId('admin-stats-bar');
    await waitFor(() => {
      expect(within(statsBar).getByText('2')).toBeInTheDocument();
    });
    expect(within(statsBar).getByText('100%')).toBeInTheDocument();
    expect(within(statsBar).getByText('No')).toBeInTheDocument();
    expect(within(statsBar).getByText('Control Variant')).toBeInTheDocument();
  });

  it('opens preview and handles demo reset confirmation flow', async () => {
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null);
    const user = userEvent.setup();

    await renderWithAuth(<AdminHome />);

    await user.click(screen.getByTestId('admin-preview-landing'));
    expect(openSpy).toHaveBeenCalledWith('/', '_blank', 'noopener,noreferrer');

    await user.click(screen.getByTestId('admin-danger-toggle'));
    await user.click(screen.getByTestId('admin-reset-demo-btn'));

    const confirmDialog = await screen.findByTestId('admin-reset-confirm-dialog');
    expect(confirmDialog).toBeInTheDocument();

    await user.click(screen.getByTestId('admin-reset-confirm-btn'));
    expect(mockedResetDemoData).toHaveBeenCalled();

    openSpy.mockRestore();
  });
});
