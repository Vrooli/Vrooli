import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AccountHub } from './AccountHub';
import { publicConfig } from '../public-landing/presentation/publicTestFixtures';
import { LandingVariantContext, type LandingVariantContextType } from '../../app/providers/LandingVariantContext';
import { UserAuthContext, type UserAuthContextValue } from '../../app/providers/UserAuthContext';

const mocks = vi.hoisted(() => ({
  getEntitlements: vi.fn<typeof import('../../shared/api/account').getEntitlements>(),
  getCreditInfo: vi.fn<typeof import('../../shared/api/account').getCreditInfo>(),
  portal: vi.fn<typeof import('../../shared/api/billing').createBillingPortalSession>(),
}));
vi.mock('../../shared/api/account', () => ({ getEntitlements: mocks.getEntitlements, getCreditInfo: mocks.getCreditInfo }));
vi.mock('../../shared/api/landing', () => ({ getLandingConfig: vi.fn(() => Promise.resolve(publicConfig('/'))) }));
vi.mock('../../shared/api/billing', () => ({ createBillingPortalSession: mocks.portal }));
vi.mock('../public-landing/site/useSiteIdentity', () => ({
  useSiteIdentity: () => ({ brandName: 'Configured suite', brandLogo: '', tagline: '', contactEmail: '', addressLines: [], copyright: '© Configured', businessName: '', headerAction: undefined }),
}));
vi.unmock('../../app/providers/useLandingVariant');
vi.unmock('../../app/providers/useUserAuth');

const user = { id: 'user-1', email: 'customer@example.test', email_verified: true };
const auth: UserAuthContextValue = { user, isAuthenticated: true, isSessionLoading: false, logout: vi.fn().mockResolvedValue(undefined), refreshSession: vi.fn().mockResolvedValue(undefined) };

function mount(session = auth) {
  const config = publicConfig('/');
  const value: LandingVariantContextType = { config, variant: { slug: 'control' }, loading: false, error: null, resolution: 'api_select', statusNote: null, lastUpdated: null, refresh: vi.fn(), request: { route: '/', locale: '', variant: '' } };
  return render(
    <MemoryRouter initialEntries={['/account']}>
      <LandingVariantContext.Provider value={value}>
        <UserAuthContext.Provider value={session}>
          <Routes>
            <Route path="/account" element={<AccountHub />} />
            <Route path="/auth/login" element={<p>Configured sign-in page</p>} />
          </Routes>
        </UserAuthContext.Provider>
      </LandingVariantContext.Provider>
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

beforeEach(() => {
  mocks.getEntitlements.mockReset();
  mocks.getCreditInfo.mockReset();
  mocks.portal.mockReset();
  mocks.getEntitlements.mockResolvedValue({ status: 'active', plan_tier: 'pro', features: [], subscription: { status: 'active', plan_tier: 'pro' } });
  mocks.getCreditInfo.mockResolvedValue({ customer_email: user.email, balance_credits: 12_000_000, bonus_credits: 0, display_credits_label: 'credits', display_credits_multiplier: 0.001 });
});
afterEach(() => { cleanup(); vi.restoreAllMocks(); });

describe('account hub', () => {
  it('shows the plan, converted credit balance, billing entry, security and profile for a subscriber', async () => {
    mount();
    expect(await screen.findByText('Pro')).toBeInTheDocument();
    expect(screen.getByText('Active')).toBeInTheDocument();
    expect(screen.getByText('12,000')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Manage billing' })).toBeEnabled();
    expect(screen.getByRole('link', { name: 'Manage security' })).toHaveAttribute('href', '/account/security');
    expect(screen.getByText(user.email)).toBeInTheDocument();
    expect(screen.getByText('Verified')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Sign out' })).toBeEnabled();
  });
  it('invites a visitor without a subscription to see plans instead of inventing one', async () => {
    mocks.getEntitlements.mockResolvedValue({ status: 'inactive', features: [], subscription: { status: 'inactive' } });
    mount();
    expect(await screen.findByText('No subscription yet')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Manage billing' })).not.toBeInTheDocument();
    expect(screen.getAllByRole('link', { name: /plans|credits/i }).length).toBeGreaterThan(0);
  });
  it('reports an entitlement outage without breaking the page', async () => {
    mocks.getEntitlements.mockRejectedValue(new Error('offline'));
    mount();
    await waitFor(() => { expect(screen.getAllByText('We could not load your plan right now. Try again in a moment.').length).toBeGreaterThan(0); });
    expect(screen.getByRole('button', { name: 'Sign out' })).toBeEnabled();
  });
  it('sends an anonymous visitor to sign in with a return path to the hub', () => {
    mount({ ...auth, user: null, isAuthenticated: false });
    expect(screen.getByText('Configured sign-in page')).toBeInTheDocument();
  });
});
