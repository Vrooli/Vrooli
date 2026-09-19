import { describe, expect, it, beforeEach, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { renderWithProviders } from "@vrooli/api-base/testing";
import { AppsManagement } from './AppsManagement';

const mockNavigate = vi.fn();
const mockRefreshDownloadsHealth = vi.fn();

const downloadsHealth = {
  appCount: 2,
  platformsConfigured: 3,
  storefrontsConfigured: 1,
  hasApps: true,
};

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');
  return { ...actual, useNavigate: () => mockNavigate };
});

vi.mock('../components/AdminLayout', () => ({
  AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main>,
}));

vi.mock('../hooks/useAdminHome', () => ({ useAdminHome: vi.fn() }));

import { useAdminHome } from '../hooks/useAdminHome';

const mockedUseAdminHome = vi.mocked(useAdminHome);

function createAdminHomeState(
  overrides: Partial<ReturnType<typeof useAdminHome>> = {}
): ReturnType<typeof useAdminHome> {
  return {
    experience: null,
    healthSnapshot: null,
    healthLoading: false,
    healthError: null,
    healthMetricsDegraded: false,
    refreshHealthSnapshot: vi.fn(),
    stripeSettings: null,
    stripeLoading: false,
    stripeError: null,
    refreshStripeStatus: vi.fn(),
    brandingHealth: null,
    brandingLoading: false,
    refreshBrandingHealth: vi.fn(),
    downloadsHealth,
    downloadsLoading: false,
    refreshDownloadsHealth: mockRefreshDownloadsHealth,
    resettingDemoData: false,
    resetMessage: null,
    resetError: null,
    showResetConfirm: false,
    setShowResetConfirm: vi.fn(),
    handleResetDemoData: vi.fn(),
    buildResumeVariantPath: vi.fn(),
    buildResumeAnalyticsPath: vi.fn(),
    ...overrides,
  };
}

function renderPage() {
  return renderWithProviders(<MemoryRouter initialEntries={['/admin/apps']}><AppsManagement /></MemoryRouter>);
}

describe('AppsManagement', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedUseAdminHome.mockReturnValue(createAdminHomeState());
  });

  it('shows configured download coverage and the product quick flows', () => {
    renderPage();

    expect(screen.getByTestId('apps-dashboard-header')).toBeInTheDocument();
    expect(screen.getByText('2 apps in registry')).toBeInTheDocument();
    expect(screen.getByText('3 platforms with installers')).toBeInTheDocument();
    expect(screen.getByText('1 storefront linked')).toBeInTheDocument();
    expect(screen.getByTestId('flow-downloads')).toBeInTheDocument();
    expect(screen.getByTestId('flow-usage')).toBeInTheDocument();
    expect(screen.getByTestId('flow-tier-limits')).toBeInTheDocument();
    expect(screen.getByTestId('flow-app-limits')).toBeInTheDocument();
  });

  it('navigates each quick flow to its owned management surface', async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByTestId('flow-downloads'));
    await user.click(screen.getByTestId('flow-usage'));
    await user.click(screen.getByTestId('flow-tier-limits'));
    await user.click(screen.getByTestId('flow-app-limits'));

    expect(mockNavigate).toHaveBeenNthCalledWith(1, '/admin/downloads');
    expect(mockNavigate).toHaveBeenNthCalledWith(2, '/admin/usage');
    expect(mockNavigate).toHaveBeenNthCalledWith(3, '/admin/tier-limits');
    expect(mockNavigate).toHaveBeenNthCalledWith(4, '/admin/app-limits');
  });

  it('offers refresh and retry controls when downloads health is unavailable', async () => {
    const user = userEvent.setup();
    mockedUseAdminHome.mockReturnValue(createAdminHomeState({
      downloadsHealth: null,
    }));

    renderPage();

    expect(screen.getByText('Unable to load downloads status')).toBeInTheDocument();
    await user.click(screen.getByTestId('apps-downloads-refresh'));
    await user.click(screen.getByRole('button', { name: 'Retry' }));
    expect(mockRefreshDownloadsHealth).toHaveBeenCalledTimes(2);
  });

  it('guides an empty app registry to download configuration', async () => {
    const user = userEvent.setup();
    mockedUseAdminHome.mockReturnValue(createAdminHomeState({
      downloadsHealth: { ...downloadsHealth, appCount: 0, platformsConfigured: 0, storefrontsConfigured: 0, hasApps: false },
    }));

    renderPage();

    expect(screen.getByTestId('apps-guidance')).toBeInTheDocument();
    await user.click(screen.getByTestId('apps-add-first'));
    expect(mockNavigate).toHaveBeenCalledWith('/admin/downloads');
  });
});
