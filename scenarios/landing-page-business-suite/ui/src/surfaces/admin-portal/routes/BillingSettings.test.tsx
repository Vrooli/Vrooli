import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { BrowserRouter } from 'react-router-dom';
import { screen, waitFor } from "@testing-library/react";
import userEvent from '@testing-library/user-event';
import { BillingSettings } from './BillingSettings';
import { AdminAuthProvider } from '../../../app/providers/AdminAuthProvider';
import { getStripeSettings, updateStripeSettings, getBundleCatalog } from '../../../shared/api';
import type { BundleCatalogEntry, StripeSettingsResponse } from '../../../shared/api';

vi.mock('../../../app/providers/LandingVariantProvider', () => ({
  useLandingVariant: () => ({
    variant: null,
    config: null,
    loading: false,
    error: null,
    resolution: 'unknown',
    statusNote: null,
    lastUpdated: null,
    refresh: vi.fn(),
  }),
}));

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getStripeSettings: vi.fn(),
    updateStripeSettings: vi.fn(),
    getBundleCatalog: vi.fn(),
    checkAdminSession: vi.fn().mockResolvedValue({ authenticated: true, email: 'admin@test.com', reset_enabled: false }),
  };
});

const mockedGetStripeSettings = vi.mocked(getStripeSettings);
const mockedUpdateStripeSettings = vi.mocked(updateStripeSettings);
const mockedGetBundleCatalog = vi.mocked(getBundleCatalog);

const wrap = (node: React.ReactElement) => (
  <BrowserRouter>
    <AdminAuthProvider>{node}</AdminAuthProvider>
  </BrowserRouter>
);

const stripeSettingsResponse: StripeSettingsResponse = {
  publishable_key_set: true,
  secret_key_set: false,
  webhook_secret_set: false,
  source: 'env',
  dashboard_url: 'https://dashboard.stripe.com/test',
  updated_at: new Date().toISOString(),
};

const demoBundle: BundleCatalogEntry = {
  bundle: {
    id: 1,
    bundle_key: 'business_suite',
    name: 'Business Suite',
    stripe_product_id: 'prod_demo',
    credits_per_usd: 1_000_000,
    display_credits_multiplier: 0.001,
    display_credits_label: 'credits',
    environment: 'production',
    metadata: {},
  },
  prices: [
    {
      plan_name: 'Solo Monthly',
      plan_tier: 'solo',
      billing_interval: 'month' as const,
      amount_cents: 4900,
      currency: 'usd',
      intro_enabled: false,
      intro_type: 'flat_amount',
      intro_amount_cents: undefined,
      intro_periods: 0,
      intro_price_lookup_key: '',
      monthly_included_credits: 1_000_000,
      one_time_bonus_credits: 0,
      plan_rank: 1,
      bonus_type: 'none',
      kind: 'subscription' as const,
      is_variable_amount: false,
      display_enabled: true,
      stripe_price_id: 'price_solo',
      display_weight: 10,
      bundle_key: 'business_suite',
      metadata: {
        features: ['Solo workspace'],
      },
    },
  ],
};

describe('BillingSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedGetStripeSettings.mockResolvedValue(stripeSettingsResponse);
    mockedUpdateStripeSettings.mockResolvedValue({
      ...stripeSettingsResponse,
      secret_key_set: true,
      source: 'database',
    });
    // Use type assertion because the mock return type is stricter than BundleCatalogEntry
    mockedGetBundleCatalog.mockResolvedValue({ bundles: [demoBundle] });
  });

  it('renders Stripe status and bundle catalog entries', async () => {
    render(wrap(<BillingSettings />));

    expect(await screen.findByText('Stripe Configuration')).toBeInTheDocument();
    expect((await screen.findAllByText('Publishable Key'))[0]).toBeInTheDocument();
    // Badge text reflects initial status flags
    expect(screen.getAllByText('Restricted Key')[0]).toBeInTheDocument();
    expect(screen.getAllByText('Webhook Secret')[0]).toBeInTheDocument();

    await waitFor(() => { expect(mockedGetBundleCatalog).toHaveBeenCalled(); });
    // In preview mode, the section title is "Preview" (not "Plan Display Manager")
    expect(screen.getByText('Preview')).toBeInTheDocument();
    // In preview mode, plan name is displayed as text (not input field)
    // Multiple elements may exist (in the read-only card and pricing preview)
    expect((await screen.findAllByText('Solo Monthly')).length).toBeGreaterThan(0);
  });

  it('blocks empty updates and surfaces errors', async () => {
    const user = userEvent.setup();
    render(wrap(<BillingSettings />));

    const saveButton = await screen.findByRole('button', { name: /save stripe settings/i });
    await user.click(saveButton);

    expect(screen.getByText('Enter at least one field before saving.')).toBeInTheDocument();
    expect(mockedUpdateStripeSettings).not.toHaveBeenCalled();
  });

  it('saves a rotated key, refreshes settings, opens Stripe, and routes to plan management', async () => {
    const user = userEvent.setup();
    const open = vi.fn();
    vi.stubGlobal('open', open);
    render(wrap(<BillingSettings />));

    await user.click(await screen.findByRole('button', { name: 'Replace secret' }));
    const publishableKey = screen.getByPlaceholderText('pk_live_...');
    await user.type(publishableKey, 'pk_live_rotated');
    await user.click(screen.getByRole('button', { name: /save stripe settings/i }));

    await waitFor(() => {
      expect(mockedUpdateStripeSettings).toHaveBeenCalledWith(expect.objectContaining({ publishable_key: 'pk_live_rotated' }));
    });
    await user.click(screen.getByRole('button', { name: 'Refresh' }));
    await user.click(screen.getByRole('button', { name: 'Open Stripe Dashboard' }));
    await user.click(screen.getByRole('button', { name: 'Manage Plans' }));

    expect(mockedGetStripeSettings).toHaveBeenCalledTimes(2);
    expect(open).toHaveBeenCalledWith('https://dashboard.stripe.com/test', '_blank', 'noopener,noreferrer');
    expect(window.location.pathname).toBe('/admin/tiers');
  });
});
