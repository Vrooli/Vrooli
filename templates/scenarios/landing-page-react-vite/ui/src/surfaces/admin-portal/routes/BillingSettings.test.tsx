import { describe, it, expect, vi, beforeEach } from 'vitest';
import { BrowserRouter } from 'react-router-dom';
import { screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../../test-utils';
import userEvent from '@testing-library/user-event';
import { create } from '@bufbuild/protobuf';
import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import {
  GetStripeSettingsResponseSchema,
  StripeSettingsSchema,
  StripeConfigSnapshotSchema,
  ConfigSource,
} from '@vrooli/proto-types/landing-page-react-vite/v1/settings_pb';
import { BundleCatalogEntrySchema } from '@vrooli/proto-types/landing-page-react-vite/v1/bundles_pb';
import {
  BundleSchema,
  PlanOptionSchema,
  BillingInterval,
  IntroPricingType,
  PlanKind,
} from '@vrooli/proto-types/landing-page-react-vite/v1/pricing_pb';
import { BillingSettings } from './BillingSettings';
import { AdminAuthProvider } from '../../../app/providers/AdminAuthProvider';
import { getStripeSettings, updateStripeSettings, getBundleCatalog, updateBundlePrice, recordToJsonMap } from '../../../shared/api';
import type { BundleCatalogEntry } from '../../../shared/api';

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
    updateBundlePrice: vi.fn(),
    checkAdminSession: vi.fn().mockResolvedValue({ authenticated: true, email: 'admin@test.com', resetEnabled: false }),
  };
});

const mockedGetStripeSettings = vi.mocked(getStripeSettings);
const mockedUpdateStripeSettings = vi.mocked(updateStripeSettings);
const mockedGetBundleCatalog = vi.mocked(getBundleCatalog);
const mockedUpdateBundlePrice = vi.mocked(updateBundlePrice);

const wrap = (node: React.ReactElement) => (
  <BrowserRouter>
    <AdminAuthProvider>{node}</AdminAuthProvider>
  </BrowserRouter>
);

const stripeSettingsResponse = create(GetStripeSettingsResponseSchema, {
  settings: create(StripeSettingsSchema, {
    dashboardUrl: 'https://dashboard.stripe.com/test',
    updatedAt: timestampFromDate(new Date()),
  }),
  snapshot: create(StripeConfigSnapshotSchema, {
    publishableKeySet: true,
    secretKeySet: false,
    webhookSecretSet: false,
    source: ConfigSource.ENV,
  }),
});

const demoBundle: BundleCatalogEntry = create(BundleCatalogEntrySchema, {
  bundle: create(BundleSchema, {
    bundleKey: 'business_suite',
    name: 'Business Suite',
    stripeProductId: 'prod_demo',
    creditsPerUsd: 1_000_000n,
    displayCreditsMultiplier: 0.001,
    displayCreditsLabel: 'credits',
    environment: 'production',
  }),
  prices: [
    create(PlanOptionSchema, {
      planName: 'Solo Monthly',
      planTier: 'solo',
      billingInterval: BillingInterval.MONTH,
      amountCents: 4900n,
      currency: 'usd',
      introEnabled: false,
      introType: IntroPricingType.FLAT_AMOUNT,
      introPeriods: 0,
      introPriceLookupKey: '',
      monthlyIncludedCredits: 1_000_000n,
      oneTimeBonusCredits: 0n,
      planRank: 1,
      bonusType: 'none',
      kind: PlanKind.SUBSCRIPTION,
      isVariableAmount: false,
      displayEnabled: true,
      stripePriceId: 'price_solo',
      displayWeight: 10,
      bundleKey: 'business_suite',
      metadata: recordToJsonMap({
        features: ['Solo workspace'],
      }),
    }),
  ],
});

describe('BillingSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedGetStripeSettings.mockResolvedValue(stripeSettingsResponse);
    mockedUpdateStripeSettings.mockResolvedValue(
      create(GetStripeSettingsResponseSchema, {
        settings: stripeSettingsResponse.settings,
        snapshot: create(StripeConfigSnapshotSchema, {
          publishableKeySet: true,
          secretKeySet: true,
          webhookSecretSet: false,
          source: ConfigSource.DATABASE,
        }),
      }),
    );
    mockedGetBundleCatalog.mockResolvedValue([demoBundle]);
  });

  it('renders Stripe status and bundle catalog entries', async () => {
    renderWithProviders(wrap(<BillingSettings />), { withoutRouter: true });

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
    renderWithProviders(wrap(<BillingSettings />), { withoutRouter: true });

    const saveButton = await screen.findByRole('button', { name: /save stripe settings/i });
    await user.click(saveButton);

    expect(screen.getByText('Enter at least one field before saving.')).toBeInTheDocument();
    expect(mockedUpdateStripeSettings).not.toHaveBeenCalled();
  });

  it('saves Stripe settings when a key is provided', async () => {
    const user = userEvent.setup();
    renderWithProviders(wrap(<BillingSettings />), { withoutRouter: true });
    const publishable = await screen.findByPlaceholderText('pk_live_...');
    await user.type(publishable, 'pk_live_abc123');
    await user.click(screen.getByRole('button', { name: /save stripe settings/i }));
    await waitFor(() => expect(mockedUpdateStripeSettings).toHaveBeenCalled());
  });

  it('renders when Stripe settings are unavailable', async () => {
    mockedGetStripeSettings.mockResolvedValue(undefined as never);
    renderWithProviders(wrap(<BillingSettings />), { withoutRouter: true });
    expect(await screen.findByText('Stripe Configuration')).toBeInTheDocument();
  });

  it('renders a plan whose metadata carries no feature list', async () => {
    mockedGetBundleCatalog.mockResolvedValue([
      create(BundleCatalogEntrySchema, {
        bundle: demoBundle.bundle,
        prices: [
          create(PlanOptionSchema, {
            planName: 'Bare Plan',
            planTier: 'solo',
            billingInterval: BillingInterval.MONTH,
            amountCents: 4900n,
            currency: 'usd',
            monthlyIncludedCredits: 1_000_000n,
            planRank: 1,
            kind: PlanKind.SUBSCRIPTION,
            displayEnabled: true,
            stripePriceId: 'price_bare',
            displayWeight: 5,
            bundleKey: 'business_suite',
            metadata: recordToJsonMap({}),
          }),
        ],
      }),
    ]);
    renderWithProviders(wrap(<BillingSettings />), { withoutRouter: true });
    expect(await screen.findByDisplayValue('Bare Plan')).toBeInTheDocument();
  });

  it('opens the Stripe dashboard in a new tab', async () => {
    const user = userEvent.setup();
    const openSpy = vi.spyOn(window, 'open').mockReturnValue(null);
    renderWithProviders(wrap(<BillingSettings />), { withoutRouter: true });
    await screen.findByText('Stripe Configuration');
    const dashboardBtn = screen.getByRole('button', { name: /open.*dashboard|stripe dashboard/i });
    await user.click(dashboardBtn);
    expect(openSpy).toHaveBeenCalledWith('https://dashboard.stripe.com/test', '_blank', 'noopener,noreferrer');
    openSpy.mockRestore();
  });

  it('renders an intro-priced plan with bonus credits', async () => {
    mockedGetBundleCatalog.mockResolvedValue([
      create(BundleCatalogEntrySchema, {
        bundle: demoBundle.bundle,
        prices: [
          create(PlanOptionSchema, {
            planName: 'Pro Monthly',
            planTier: 'pro',
            billingInterval: BillingInterval.MONTH,
            amountCents: 9900n,
            currency: 'usd',
            introEnabled: true,
            introType: IntroPricingType.FLAT_AMOUNT,
            introPeriods: 3,
            monthlyIncludedCredits: 10_000_000n,
            oneTimeBonusCredits: 2_000_000n,
            planRank: 2,
            bonusType: 'signup_bonus',
            kind: PlanKind.SUBSCRIPTION,
            displayEnabled: true,
            stripePriceId: 'price_pro',
            displayWeight: 20,
            bundleKey: 'business_suite',
            metadata: recordToJsonMap({ features: ['Everything in Solo'], badge: 'Popular' }),
          }),
        ],
      }),
    ]);
    renderWithProviders(wrap(<BillingSettings />), { withoutRouter: true });
    expect(await screen.findByDisplayValue('Pro Monthly')).toBeInTheDocument();
  });

  it('edits the Stripe secret, webhook, and dashboard fields', async () => {
    const user = userEvent.setup();
    renderWithProviders(wrap(<BillingSettings />), { withoutRouter: true });
    await screen.findByText('Stripe Configuration');
    await user.type(screen.getByPlaceholderText('sk_live_...'), 'sk_live_secret');
    await user.type(screen.getByPlaceholderText('whsec_...'), 'whsec_hook');
    await user.type(screen.getByPlaceholderText('https://dashboard.stripe.com/...'), 'https://dashboard.stripe.com/acct');
    await user.click(screen.getByRole('button', { name: /save stripe settings/i }));
    await waitFor(() => expect(mockedUpdateStripeSettings).toHaveBeenCalled());
  });

  it('surfaces a bundle price save error', async () => {
    const user = userEvent.setup();
    mockedUpdateBundlePrice.mockRejectedValue(new Error('price save failed'));
    renderWithProviders(wrap(<BillingSettings />), { withoutRouter: true });
    const nameInput = await screen.findByDisplayValue('Solo Monthly');
    await user.type(nameInput, ' Plan');
    await user.click(await screen.findByRole('button', { name: 'Save changes' }));
    expect(await screen.findByText('price save failed')).toBeInTheDocument();
  });

  it('saves a plan whose metadata carries subtitle, badge, and cta label', async () => {
    const user = userEvent.setup();
    mockedUpdateBundlePrice.mockResolvedValue(demoBundle.prices[0]);
    mockedGetBundleCatalog.mockResolvedValue([
      create(BundleCatalogEntrySchema, {
        bundle: demoBundle.bundle,
        prices: [
          create(PlanOptionSchema, {
            planName: 'Solo Monthly',
            planTier: 'solo',
            billingInterval: BillingInterval.MONTH,
            amountCents: 4900n,
            currency: 'usd',
            monthlyIncludedCredits: 1_000_000n,
            planRank: 1,
            bonusType: 'none',
            kind: PlanKind.SUBSCRIPTION,
            displayEnabled: true,
            stripePriceId: 'price_solo',
            displayWeight: 10,
            bundleKey: 'business_suite',
            metadata: recordToJsonMap({
              features: ['Solo workspace'],
              subtitle: 'Best for solo builders',
              badge: 'Starter',
              cta_label: 'Pick Solo',
              highlight: true,
            }),
          }),
        ],
      }),
    ]);
    renderWithProviders(wrap(<BillingSettings />), { withoutRouter: true });
    const nameInput = await screen.findByDisplayValue('Solo Monthly');
    await user.type(nameInput, ' Pro');
    await user.click(await screen.findByRole('button', { name: 'Save changes' }));
    await waitFor(() =>
      expect(mockedUpdateBundlePrice).toHaveBeenCalledWith(
        'business_suite',
        'price_solo',
        expect.objectContaining({ subtitle: 'Best for solo builders', badge: 'Starter', ctaLabel: 'Pick Solo' }),
      ),
    );
  });

  it('edits the plan display fields (features, weight, toggles)', async () => {
    const user = userEvent.setup();
    renderWithProviders(wrap(<BillingSettings />), { withoutRouter: true });
    await screen.findByDisplayValue('Solo Monthly');

    // The real Solo plan card is first (demo padding cards follow).
    const features = screen.getAllByPlaceholderText(/One feature per line/i)[0]!;
    await user.type(features, '\nPriority support');
    const weight = screen.getAllByDisplayValue('10')[0]!;
    await user.clear(weight);
    await user.type(weight, '25');
    const checkboxes = screen.getAllByRole('checkbox');
    await user.click(checkboxes[0]!);
    expect((await screen.findAllByRole('button', { name: 'Save changes' }))[0]!).toBeEnabled();
  });

  it('saves a bundle price display update', async () => {
    const user = userEvent.setup();
    mockedUpdateBundlePrice.mockResolvedValue(demoBundle.prices[0]);
    renderWithProviders(wrap(<BillingSettings />), { withoutRouter: true });
    // Editing the plan display name makes the card dirty and enables Save changes.
    const nameInput = await screen.findByDisplayValue('Solo Monthly');
    await user.type(nameInput, ' Plan');
    await user.click(await screen.findByRole('button', { name: 'Save changes' }));
    await waitFor(() => expect(mockedUpdateBundlePrice).toHaveBeenCalled());
  });
});
