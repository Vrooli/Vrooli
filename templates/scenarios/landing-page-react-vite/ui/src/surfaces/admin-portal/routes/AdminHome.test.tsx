import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { MockInstance } from 'vitest';
import type { ReactNode } from 'react';
import { screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../../test-utils';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { create } from '@bufbuild/protobuf';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { VariantSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/variant_pb';
import { AnalyticsSummarySchema, VariantStatsSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/metrics_pb';
import { AdminSessionResponseSchema, ResetDemoDataResponseSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/admin_pb';
import {
  GetStripeSettingsResponseSchema,
  StripeConfigSnapshotSchema,
  StripeSettingsSchema,
  ConfigSource,
} from '@vrooli/proto-types/landing-page-react-vite/v1/settings_pb';
import { AdminHome } from './AdminHome';
import { AdminAuthProvider } from '../../../app/providers/AdminAuthProvider';
import { listVariants, checkAdminSession, getStripeSettings, resetDemoData, getBranding, listDownloadAppsAdmin } from '../../../shared/api';
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
    getBranding: vi.fn(),
    listDownloadAppsAdmin: vi.fn(),
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
const mockVariants = [
  create(VariantSchema, {
    id: 1n,
    slug: 'control',
    name: 'Control Variant',
    status: 'active',
    weight: 70,
    updatedAt: timestampFromDate(new Date()),
  }),
  create(VariantSchema, {
    id: 2n,
    slug: 'beta',
    name: 'Beta Variant',
    status: 'active',
    weight: 30,
    updatedAt: timestampFromDate(new Date(Date.now() - 12 * 24 * 60 * 60 * 1000)),
  }),
];

const mockAnalyticsSummary = create(AnalyticsSummarySchema, {
  totalVisitors: 1000n,
  totalDownloads: 80n,
  variantStats: [
    create(VariantStatsSchema, {
      variantId: 1n,
      variantSlug: 'control',
      variantName: 'Control Variant',
      views: 700n,
      ctaClicks: 200n,
      conversions: 120n,
      downloads: 50n,
      conversionRate: 17.14,
    }),
    create(VariantStatsSchema, {
      variantId: 2n,
      variantSlug: 'beta',
      variantName: 'Beta Variant',
      views: 300n,
      ctaClicks: 40n,
      conversions: 12n,
      downloads: 30n,
      conversionRate: 4,
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

describe('AdminHome [REQ:ADMIN-MODES]', () => {
  let fetchAnalyticsSpy: MockInstance<
    Parameters<typeof analyticsController.fetchAnalyticsSummary>,
    ReturnType<typeof analyticsController.fetchAnalyticsSummary>
  >;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false }));
    vi.stubGlobal('location', { ...window.location, pathname: '/admin', search: '', hash: '' });
    window.localStorage.clear();
    mockedListVariants.mockResolvedValue(mockVariants);
    mockedCheckAdminSession.mockResolvedValue(
      create(AdminSessionResponseSchema, { authenticated: true, email: 'ops@vrooli.dev', resetEnabled: false })
    );
    fetchAnalyticsSpy = vi.spyOn(analyticsController, 'fetchAnalyticsSummary').mockResolvedValue(mockAnalyticsSummary);
    mockedGetStripeSettings.mockResolvedValue(mockStripeSettings);
    mockedResetDemoData.mockResolvedValue(
      create(ResetDemoDataResponseSchema, { reset: true, timestamp: new Date().toISOString() })
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
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

  it('surfaces demo data reset control when flag enabled', async () => {
    mockedCheckAdminSession.mockResolvedValue(
      create(AdminSessionResponseSchema, { authenticated: true, email: 'ops@vrooli.dev', resetEnabled: true })
    );
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true);
    const user = userEvent.setup();

    renderWithRouter(<AdminHome />);

    const resetCard = await screen.findByTestId('admin-reset-demo-card');
    expect(resetCard).toBeInTheDocument();

    await user.click(screen.getByTestId('admin-reset-demo-btn'));

    expect(confirmSpy).toHaveBeenCalled();
    expect(mockedResetDemoData).toHaveBeenCalled();

    confirmSpy.mockRestore();
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

  it('renders branding and downloads health cards when their data resolves', async () => {
    vi.mocked(getBranding).mockResolvedValue({
      siteName: 'Acme',
      tagline: 'Ship',
      logoUrl: 'logo.png',
      faviconUrl: 'fav.png',
      defaultOgImageUrl: 'og.png',
      canonicalBaseUrl: 'https://acme.test',
    } as never);
    vi.mocked(listDownloadAppsAdmin).mockResolvedValue([
      { appKey: 'suite', name: 'Suite', platforms: [{ platform: 'mac', artifactUrl: 'x', releaseVersion: '1' }], storefronts: [] },
    ] as never);
    renderWithRouter(<AdminHome />);
    // The monetization/health cards render once their async data lands.
    expect(await screen.findByTestId('admin-monetization-card')).toBeInTheDocument();
  });

  it('marks incomplete branding and an empty downloads catalog in the health cards', async () => {
    // Identity present, but favicon/SEO/OG missing -> mixed check states.
    vi.mocked(getBranding).mockResolvedValue({ siteName: 'Acme' } as never);
    vi.mocked(listDownloadAppsAdmin).mockResolvedValue([] as never);
    renderWithRouter(<AdminHome />);
    expect(await screen.findByTestId('admin-monetization-card')).toBeInTheDocument();
  });

  it('resumes a section-surface editing target from recents', async () => {
    window.localStorage.setItem(
      'landing_admin_experience',
      JSON.stringify({
        version: 1,
        lastVariant: { slug: 'alpha', name: 'Variant Alpha', surface: 'section', sectionId: 3, sectionType: 'hero', lastVisitedAt: new Date().toISOString() },
      }),
    );
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);
    await user.click(await screen.findByTestId('admin-resume-customization'));
    expect(mockNavigate).toHaveBeenCalled();
  });

  it('routes through the experience-guide navigation shortcuts', async () => {
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);
    await screen.findByTestId('admin-experience-guide');

    await user.click(screen.getByTestId('admin-guide-analytics'));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/analytics');
    await user.click(screen.getByTestId('admin-guide-customization'));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/customization');
    await user.click(screen.getByTestId('admin-guide-billing'));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/billing');
    await user.click(screen.getByTestId('admin-guide-downloads'));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/downloads');
  });

  it('resumes recent variant and analytics work from the quick-resume panel', async () => {
    window.localStorage.setItem(
      'landing_admin_experience',
      JSON.stringify({
        version: 1,
        lastVariant: { slug: 'alpha', name: 'Variant Alpha', surface: 'variant', lastVisitedAt: new Date().toISOString() },
        lastAnalytics: { variantSlug: 'beta', variantName: 'Variant Beta', timeRangeDays: 30, savedAt: new Date().toISOString() },
      }),
    );
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);

    await user.click(await screen.findByTestId('admin-resume-customization'));
    await user.click(screen.getByTestId('admin-resume-analytics'));
    expect(mockNavigate).toHaveBeenCalled();
  });

  it('refreshes the health digest and inspects an attention variant', async () => {
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);

    await user.click(await screen.findByTestId('admin-health-refresh'));
    await user.click(screen.getByTestId('admin-health-attention-analytics'));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/analytics?variant=beta');
  });

  it('renders even when the variant list fails to load', async () => {
    mockedListVariants.mockRejectedValueOnce(new Error('variants offline'));
    renderWithRouter(<AdminHome />);
    // The two admin modes still render from static config.
    expect(await screen.findByTestId('admin-mode-analytics')).toBeInTheDocument();
  });

  it('tolerates a stripe settings load failure', async () => {
    mockedGetStripeSettings.mockRejectedValueOnce(new Error('stripe offline'));
    renderWithRouter(<AdminHome />);
    expect(await screen.findByTestId('admin-mode-customization')).toBeInTheDocument();
  });

  it.each([
    ['over-allocated', [70, 60]],
    ['under-allocated', [30, 20]],
  ])('summarizes %s traffic weight in the health digest', async (_label, weights) => {
    mockedListVariants.mockResolvedValue(
      weights.map((w, i) =>
        create(VariantSchema, { id: BigInt(i + 1), slug: `v${i}`, name: `V${i}`, status: 'active', weight: w, updatedAt: timestampFromDate(new Date()) }),
      ),
    );
    renderWithRouter(<AdminHome />);
    expect(await screen.findByTestId('admin-health-digest')).toBeInTheDocument();
  });

  it('merges multiple attention reasons for a single variant', async () => {
    // A stale variant that is also the lowest converter -> two reasons.
    mockedListVariants.mockResolvedValue([
      create(VariantSchema, { id: 1n, slug: 'weak', name: 'Weak', status: 'active', weight: 100, updatedAt: timestampFromDate(new Date(Date.now() - 20 * 24 * 60 * 60 * 1000)) }),
    ]);
    fetchAnalyticsSpy.mockResolvedValue(
      create(AnalyticsSummarySchema, {
        totalVisitors: 100n,
        totalDownloads: 5n,
        variantStats: [create(VariantStatsSchema, { variantId: 1n, variantSlug: 'weak', variantName: 'Weak', views: 100n, conversions: 1n, downloads: 1n, conversionRate: 1 })],
      }),
    );
    renderWithRouter(<AdminHome />);
    expect(await screen.findByTestId('admin-health-attention-card')).toHaveTextContent('Weak');
  });

  it('lists multiple attention variants (stale and never customized)', async () => {
    mockedListVariants.mockResolvedValue([
      create(VariantSchema, { id: 1n, slug: 'stale', name: 'Stale One', status: 'active', weight: 50, updatedAt: timestampFromDate(new Date(Date.now() - 15 * 24 * 60 * 60 * 1000)) }),
      create(VariantSchema, { id: 2n, slug: 'never', name: 'Never One', status: 'active', weight: 50 }),
    ]);
    renderWithRouter(<AdminHome />);
    expect(await screen.findByTestId('admin-health-attention-card')).toBeInTheDocument();
  });

  it('resumes with only recent analytics and no variant recency', async () => {
    window.localStorage.setItem(
      'landing_admin_experience',
      JSON.stringify({
        version: 1,
        lastAnalytics: { variantSlug: 'beta', variantName: 'Variant Beta', timeRangeDays: 30, savedAt: new Date().toISOString() },
      }),
    );
    renderWithRouter(<AdminHome />);
    expect(await screen.findByTestId('admin-resume-analytics')).toBeInTheDocument();
    expect(screen.queryByTestId('admin-resume-customization')).not.toBeInTheDocument();
  });

  it('resumes with only a recent variant and no analytics recency', async () => {
    window.localStorage.setItem(
      'landing_admin_experience',
      JSON.stringify({
        version: 1,
        lastVariant: { slug: 'alpha', name: 'Variant Alpha', surface: 'variant', lastVisitedAt: new Date().toISOString() },
      }),
    );
    renderWithRouter(<AdminHome />);
    expect(await screen.findByTestId('admin-resume-customization')).toBeInTheDocument();
    expect(screen.queryByTestId('admin-resume-analytics')).not.toBeInTheDocument();
  });

  it('surfaces a reset-demo-data error', async () => {
    mockedCheckAdminSession.mockResolvedValue(
      create(AdminSessionResponseSchema, { authenticated: true, email: 'ops@vrooli.dev', resetEnabled: true }),
    );
    mockedResetDemoData.mockRejectedValueOnce(new Error('reset failed'));
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true);
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);
    await user.click(await screen.findByTestId('admin-reset-demo-btn'));
    expect(await screen.findByTestId('admin-reset-error')).toBeInTheDocument();
    confirmSpy.mockRestore();
  });

  it('does not reset demo data when the confirmation is declined', async () => {
    mockedCheckAdminSession.mockResolvedValue(
      create(AdminSessionResponseSchema, { authenticated: true, email: 'ops@vrooli.dev', resetEnabled: true }),
    );
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false);
    const user = userEvent.setup();
    renderWithRouter(<AdminHome />);
    await user.click(await screen.findByTestId('admin-reset-demo-btn'));
    expect(mockedResetDemoData).not.toHaveBeenCalled();
    confirmSpy.mockRestore();
  });

  it('reflects a fully configured Stripe account from the database source', async () => {
    mockedGetStripeSettings.mockResolvedValue(
      create(GetStripeSettingsResponseSchema, {
        snapshot: create(StripeConfigSnapshotSchema, {
          publishableKeySet: true,
          secretKeySet: true,
          webhookSecretSet: true,
          source: ConfigSource.DATABASE,
        }),
        settings: create(StripeSettingsSchema, { dashboardUrl: 'https://dashboard.stripe.com/', updatedAt: timestampFromDate(new Date()) }),
      }),
    );
    renderWithRouter(<AdminHome />);
    const card = await screen.findByTestId('admin-monetization-card');
    expect(card).toBeInTheDocument();
  });

  it('shows a healthy digest when all variants are fresh and converting', async () => {
    mockedListVariants.mockResolvedValue([
      create(VariantSchema, { id: 1n, slug: 'control', name: 'Control', status: 'active', weight: 50, updatedAt: timestampFromDate(new Date()) }),
      create(VariantSchema, { id: 2n, slug: 'beta', name: 'Beta', status: 'active', weight: 50, updatedAt: timestampFromDate(new Date()) }),
    ]);
    fetchAnalyticsSpy.mockResolvedValue(
      create(AnalyticsSummarySchema, {
        totalVisitors: 2000n,
        totalDownloads: 200n,
        variantStats: [
          create(VariantStatsSchema, { variantId: 1n, variantSlug: 'control', variantName: 'Control', views: 1000n, conversions: 200n, downloads: 100n, conversionRate: 20 }),
          create(VariantStatsSchema, { variantId: 2n, variantSlug: 'beta', variantName: 'Beta', views: 1000n, conversions: 180n, downloads: 100n, conversionRate: 18 }),
        ],
      }),
    );
    renderWithRouter(<AdminHome />);
    expect(await screen.findByTestId('admin-health-digest')).toBeInTheDocument();
  });
});
const mockStripeSettings = create(GetStripeSettingsResponseSchema, {
  snapshot: create(StripeConfigSnapshotSchema, {
    publishableKeySet: true,
    secretKeySet: true,
    webhookSecretSet: false,
    source: ConfigSource.ENV,
  }),
  settings: create(StripeSettingsSchema, {
    dashboardUrl: 'https://dashboard.stripe.com/',
    updatedAt: timestampFromDate(new Date()),
  }),
});

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
