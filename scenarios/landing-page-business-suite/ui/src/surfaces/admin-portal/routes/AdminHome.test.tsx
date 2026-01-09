import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { ReactNode } from 'react';
import { render, screen, waitFor } from '@testing-library/react';
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

const mockedListVariants = vi.mocked(listVariants);
const mockedCheckAdminSession = vi.mocked(checkAdminSession);
const mockedGetStripeSettings = vi.mocked(getStripeSettings);
const mockedResetDemoData = vi.mocked(resetDemoData);
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

const renderWithRouter = (component: React.ReactElement) => {
  return render(
    <BrowserRouter>
      <AdminAuthProvider>
        {component}
      </AdminAuthProvider>
    </BrowserRouter>
  );
};

describe('AdminHome [REQ:ADMIN-MODES]', () => {
  const originalFetch = global.fetch;
  const originalLocation = window.location;
  let fetchAnalyticsSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn().mockResolvedValue({ ok: false } as Response);
    delete (window as { location?: Location }).location;
    window.location = { ...originalLocation, pathname: '/admin' };
    window.localStorage.clear();
    mockedListVariants.mockResolvedValue(mockVariantsResponse);
    mockedCheckAdminSession.mockResolvedValue({ authenticated: true, email: 'ops@vrooli.dev', reset_enabled: false });
    fetchAnalyticsSpy = vi.spyOn(analyticsController, 'fetchAnalyticsSummary').mockResolvedValue(mockAnalyticsSummary);
    mockedGetStripeSettings.mockResolvedValue(mockStripeSettings);
    mockedResetDemoData.mockResolvedValue({ reset: true, timestamp: new Date().toISOString() });
  });

  afterEach(() => {
    global.fetch = originalFetch;
    window.location = originalLocation;
    window.localStorage.clear();
    fetchAnalyticsSpy.mockRestore();
  });

  it('[REQ:ADMIN-MODES] should display exactly two modes: Analytics and Customization', () => {
    renderWithRouter(<AdminHome />);

    expect(screen.getByText('Analytics / Metrics')).toBeInTheDocument();
    // Use role query to avoid ambiguity - there's a heading and button text with "Customization"
    expect(screen.getByRole('heading', { name: 'Customization' })).toBeInTheDocument();

    const modeButtons = screen.getAllByRole('button').filter(btn =>
      btn.getAttribute('data-testid')?.startsWith('admin-mode-')
    );
    expect(modeButtons).toHaveLength(2);
  });

  it('[REQ:ADMIN-NAV] should navigate to analytics when Analytics mode is clicked', async () => {
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);

    const analyticsButton = screen.getByTestId('admin-mode-analytics');
    await user.click(analyticsButton);

    expect(mockNavigate).toHaveBeenCalledWith('/admin/analytics');
  });

  it('[REQ:ADMIN-NAV] should navigate to customization when Customization mode is clicked', async () => {
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);

    const customizationButton = screen.getByTestId('admin-mode-customization');
    await user.click(customizationButton);

    expect(mockNavigate).toHaveBeenCalledWith('/admin/customization');
  });

  it('should display mode descriptions', () => {
    renderWithRouter(<AdminHome />);

    expect(screen.getByText(/View conversion rates, A\/B test results/)).toBeInTheDocument();
    expect(screen.getByText(/Customize landing page content, trigger agent-based/)).toBeInTheDocument();
  });

  it('renders the experience guide with preview affordance', async () => {
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null);
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);

    expect(screen.getByTestId('admin-experience-guide')).toBeInTheDocument();
    await user.click(screen.getByTestId('admin-guide-preview'));

    expect(openSpy).toHaveBeenCalledWith('/', '_blank', 'noopener,noreferrer');
    openSpy.mockRestore();
  });

  it('surfaces demo data reset control with confirmation dialog', async () => {
    const user = userEvent.setup();

    renderWithRouter(<AdminHome />);

    // Reset card should always be visible
    const resetCard = await screen.findByTestId('admin-reset-demo-card');
    expect(resetCard).toBeInTheDocument();

    // Click the reset button to show confirmation dialog
    await user.click(screen.getByTestId('admin-reset-demo-btn'));

    // Confirmation dialog should appear
    const confirmDialog = await screen.findByTestId('admin-reset-confirm-dialog');
    expect(confirmDialog).toBeInTheDocument();
    expect(confirmDialog).toHaveTextContent('Are you sure you want to reset?');

    // Click confirm to execute reset
    await user.click(screen.getByTestId('admin-reset-confirm-btn'));

    expect(mockedResetDemoData).toHaveBeenCalled();
  });

  it('should surface quick resume panel when recents exist', async () => {
    window.localStorage.setItem(
      'landing_admin_experience',
      JSON.stringify({
        version: 1,
        lastVariant: {
          slug: 'alpha',
          name: 'Variant Alpha',
          surface: 'variant',
          lastVisitedAt: new Date().toISOString(),
        },
        lastAnalytics: {
          variantSlug: 'beta',
          variantName: 'Variant Beta',
          timeRangeDays: 30,
          savedAt: new Date().toISOString(),
        },
      })
    );

    renderWithRouter(<AdminHome />);

    expect(await screen.findByTestId('admin-resume-panel')).toBeInTheDocument();
    expect(screen.getByTestId('admin-resume-customization')).toBeInTheDocument();
    expect(screen.getByTestId('admin-resume-analytics')).toBeInTheDocument();
  });

  it('renders experience health digest with attention summary', async () => {
    renderWithRouter(<AdminHome />);

    await waitFor(() => {
      expect(screen.getByTestId('admin-health-digest')).toBeInTheDocument();
    });
    expect(screen.getByTestId('admin-health-attention-card')).toHaveTextContent('Beta Variant');
  });

  it('navigates to focused customization from health digest', async () => {
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);

    const reviewButton = await screen.findByTestId('admin-health-review');
    await user.click(reviewButton);

    expect(mockNavigate).toHaveBeenCalledWith('/admin/customization?focus=beta&focusSectionType=hero');
  });
});
const mockStripeSettings = {
  publishable_key_set: true,
  secret_key_set: true,
  webhook_secret_set: false,
  source: 'env' as const,
  updated_at: new Date().toISOString(),
  dashboard_url: 'https://dashboard.stripe.com/',
};

  it('surfaces monetization guardrails and billing flow', async () => {
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);

    const monetizationCard = await screen.findByTestId('admin-monetization-card');
    expect(monetizationCard).toHaveTextContent('Monetization guardrail');
    expect(monetizationCard).toHaveTextContent('Publishable key');
    expect(monetizationCard).toHaveTextContent('Webhook secret');

    const billingFlowButton = await screen.findByTestId('admin-guide-billing');
    await user.click(billingFlowButton);
    expect(mockNavigate).toHaveBeenCalledWith('/admin/billing');
  });
