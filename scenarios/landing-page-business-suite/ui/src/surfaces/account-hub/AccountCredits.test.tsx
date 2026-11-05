import { screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import AccountCredits from './AccountCredits';

const mocks = vi.hoisted(() => ({ get: vi.fn(), plans: vi.fn() }));
vi.mock('../../app/providers/useUserAuth', () => ({ useUserAuth: () => ({ isAuthenticated: true, isSessionLoading: false }) }));
vi.mock('../../shared/api/common', () => ({ apiGet: mocks.get }));
vi.mock('../../shared/api/landing', () => ({ getPlans: mocks.plans }));
vi.mock('../../shared/api/billing', () => ({ createCreditsCheckoutSession: vi.fn() }));
vi.mock('../public-landing/site/SiteShell', () => ({ SiteShell: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));

describe('AccountCredits', () => {
  it('loads the authenticated account through the shared API suffix once', async () => {
    vi.stubGlobal('matchMedia', (media: string) => ({ matches: false, media, addEventListener() {}, removeEventListener() {}, addListener() {}, removeListener() {} }));
    mocks.get.mockImplementation(async (path: string) => path === '/business-accounts'
      ? { accounts: [{ id: 'account', display_name: 'My account', billing_email: 'buyer@example.invalid' }] }
      : { limits: { ai_credits: 100 }, remaining: { ai_credits: 100 }, wallet: { available: 10, on_hold: 0, holds: [], history: [], livemode: false } });
    mocks.plans.mockResolvedValue({ bundle: { display_credits_label: 'credits', display_credits_multiplier: 1 }, monthly: [], yearly: [], credit_topups: [] });
    render(<MemoryRouter><AccountCredits /></MemoryRouter>, { withoutRouter: true });
    await waitFor(() => { expect(mocks.get).toHaveBeenCalledWith('/business-accounts'); });
    expect(await screen.findByRole('button', { name: 'Billing account' })).toHaveTextContent('My account');
  });
});
