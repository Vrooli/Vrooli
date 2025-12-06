import { describe, it, expect, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { render, screen, waitFor } from '@testing-library/react';
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
      billing_interval: 'month',
      amount_cents: 4900,
      currency: 'usd',
      intro_enabled: false,
      intro_type: '',
      intro_amount_cents: undefined,
      intro_periods: 0,
      intro_price_lookup_key: '',
      monthly_included_credits: 1_000_000,
      one_time_bonus_credits: 0,
      plan_rank: 1,
      bonus_type: 'none',
      kind: 'subscription',
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
    mockedGetBundleCatalog.mockResolvedValue({ bundles: [demoBundle] });
  });

  it('renders Stripe status and bundle catalog entries', async () => {
    render(wrap(<BillingSettings />));

    expect(await screen.findByText('Stripe Configuration')).toBeInTheDocument();
    expect(screen.getAllByText('Publishable Key')[0]).toBeInTheDocument();
    // Badge text reflects initial status flags
    expect(screen.getAllByText('Secret Key')[0]).toBeInTheDocument();
    expect(screen.getAllByText('Webhook Secret')[0]).toBeInTheDocument();

    await waitFor(() => expect(mockedGetBundleCatalog).toHaveBeenCalled());
    expect(screen.getByText('Plan Display Manager')).toBeInTheDocument();
    expect(await screen.findByDisplayValue('Solo Monthly')).toBeInTheDocument();
  });

  it('blocks empty updates and surfaces errors', async () => {
    const user = userEvent.setup();
    render(wrap(<BillingSettings />));

    const saveButton = await screen.findByRole('button', { name: /save stripe settings/i });
    await user.click(saveButton);

    expect(screen.getByText('Enter at least one field before saving.')).toBeInTheDocument();
    expect(mockedUpdateStripeSettings).not.toHaveBeenCalled();
  });
});
