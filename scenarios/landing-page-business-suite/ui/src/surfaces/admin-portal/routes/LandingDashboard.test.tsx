/* eslint-disable @typescript-eslint/unbound-method -- assertions exercise Vitest/browser mocks, not detached production methods. */
import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils/renderWithProviders';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { LandingDashboard } from './LandingDashboard';
import * as adminHome from '../hooks/useAdminHome';

const navigate = vi.fn();
vi.mock('react-router-dom', () => ({ useNavigate: () => navigate }));
vi.mock('../hooks/useAdminHome');
vi.mock('../../../app/providers/useLandingVariant', () => ({ useLandingVariant: vi.fn() }));
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title }: { title: string }) => <h1>{title}</h1> }));

const landingVariant = await import('../../../app/providers/useLandingVariant');

function homeState(overrides: Record<string, unknown> = {}) {
  return { experience: null, healthSnapshot: null, healthLoading: false, healthError: null, healthMetricsDegraded: false, refreshHealthSnapshot: vi.fn(), brandingHealth: null, brandingLoading: false, ...overrides } as unknown as ReturnType<typeof adminHome.useAdminHome>;
}

describe('LandingDashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('open', vi.fn());
    vi.mocked(landingVariant.useLandingVariant).mockReturnValue({ variant: null, resolution: 'fallback' } as never);
  });

  it('routes every landing-management quick flow and refreshes/opens public preview', () => {
    const state = homeState({ healthLoading: true, brandingLoading: true });
    vi.mocked(adminHome.useAdminHome).mockReturnValue(state);
    render(<LandingDashboard />);
    fireEvent.click(screen.getByTestId('flow-customization'));
    fireEvent.click(screen.getByTestId('flow-analytics'));
    fireEvent.click(screen.getByTestId('flow-branding'));
    fireEvent.click(screen.getByTestId('flow-agent'));
    fireEvent.click(screen.getByTestId('landing-health-refresh'));
    fireEvent.click(screen.getByRole('button', { name: 'Preview' }));
    expect(navigate).toHaveBeenNthCalledWith(1, '/admin/customization');
    expect(navigate).toHaveBeenNthCalledWith(2, '/admin/analytics');
    expect(navigate).toHaveBeenNthCalledWith(3, '/admin/branding');
    expect(navigate).toHaveBeenNthCalledWith(4, '/admin/customization/agent');
    expect(state.refreshHealthSnapshot).toHaveBeenCalledOnce();
    expect(window.open).toHaveBeenCalledWith('/', '_blank', 'noopener,noreferrer');
  });

  it('shows degraded allocation and routes resumed section customization and scoped analytics', () => {
    const state = homeState({
      experience: { lastVariant: { slug: 'experiment-b', name: 'Experiment B', surface: 'section', sectionId: 42, sectionType: 'hero' }, lastAnalytics: { variantSlug: 'experiment-b', variantName: 'Experiment B', timeRangeDays: 30 } },
      healthSnapshot: { activeCount: 2, attentionCount: 1, totalWeight: 80, weightStatus: 'under' }, healthMetricsDegraded: true,
      brandingHealth: { hasIdentity: true, hasFavicon: false, hasSeo: true, hasOgImage: false },
    });
    vi.mocked(adminHome.useAdminHome).mockReturnValue(state);
    vi.mocked(landingVariant.useLandingVariant).mockReturnValue({ variant: { name: 'Experiment B', slug: 'experiment-b' }, resolution: 'cookie' } as never);
    render(<LandingDashboard />);
    expect(screen.getByText('Analytics data partially unavailable. Variant status is still accurate.')).toBeInTheDocument();
    expect(screen.getByTestId('landing-health-stats')).toHaveTextContent('Active variants2');
    expect(screen.getByText('20% of visitors are idle because weights total less than 100%.')).toBeInTheDocument();
    expect(screen.getAllByText('Experiment B')).toHaveLength(3);
    fireEvent.click(screen.getByTestId('landing-resume-customization'));
    fireEvent.click(screen.getByTestId('landing-resume-analytics'));
    expect(navigate).toHaveBeenCalledWith('/admin/customization/variants/experiment-b/sections/42');
    expect(navigate).toHaveBeenCalledWith('/admin/analytics?variant=experiment-b&range=30');
    expect(screen.getAllByText('Configured')).toHaveLength(2);
    expect(screen.getAllByText('Missing')).toHaveLength(2);
  });

  it('surfaces health errors and retries without hiding branding configuration access', () => {
    const state = homeState({ healthError: 'Health service unavailable' });
    vi.mocked(adminHome.useAdminHome).mockReturnValue(state);
    render(<LandingDashboard />);
    expect(screen.getByText('Health service unavailable')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    fireEvent.click(screen.getByRole('button', { name: 'Configure branding' }));
    expect(state.refreshHealthSnapshot).toHaveBeenCalledOnce();
    expect(navigate).toHaveBeenCalledWith('/admin/branding');
  });

  it('renders a balanced healthy experience and resumes unscoped variant and default analytics views', () => {
    const state = homeState({
      experience: {
        lastVariant: { slug: 'control', surface: 'variant' },
        lastAnalytics: { variantSlug: null, timeRangeDays: 7 },
      },
      healthSnapshot: { activeCount: 1, attentionCount: 0, totalWeight: 125, weightStatus: 'balanced' },
      brandingHealth: null,
    });
    vi.mocked(adminHome.useAdminHome).mockReturnValue(state);
    vi.mocked(landingVariant.useLandingVariant).mockReturnValue({ variant: { slug: 'control' }, resolution: 'url_param' } as never);
    render(<LandingDashboard />);

    expect(screen.getByTestId('landing-health-stats')).toHaveTextContent('Traffic assigned125%');
    expect(screen.getByText('Traffic is fully allocated across variants.')).toBeInTheDocument();
    expect(screen.getByText('Unable to load branding status')).toBeInTheDocument();
    expect(screen.getByText('All variants')).toBeInTheDocument();
    expect(screen.getByText('Showing last 7 days window.')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('landing-resume-customization'));
    fireEvent.click(screen.getByTestId('landing-resume-analytics'));
    expect(navigate).toHaveBeenCalledWith('/admin/customization/variants/control');
    expect(navigate).toHaveBeenCalledWith('/admin/analytics');
  });
});
