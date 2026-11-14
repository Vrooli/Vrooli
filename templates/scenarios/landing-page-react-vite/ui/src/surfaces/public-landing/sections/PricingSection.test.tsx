import { describe, it, expect, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../test-utils';
import { create } from '@bufbuild/protobuf';
import {
  BundleSchema,
  PlanOptionSchema,
  PricingOverviewSchema,
  BillingInterval,
} from '@vrooli/proto-types/landing-page-react-vite/v1/pricing_pb';
import { PricingSection } from './PricingSection';
import { recordToJsonMap } from '../../../shared/api';

vi.mock('../../../shared/hooks/useMetrics', () => ({
  useMetrics: () => ({
    trackCTAClick: vi.fn(),
  }),
}));

const bundle = create(BundleSchema, {
  bundleKey: 'business_suite',
  name: 'Business Suite',
  stripeProductId: 'prod_123',
  creditsPerUsd: 1_000_000n,
  displayCreditsMultiplier: 0.001,
  displayCreditsLabel: 'credits',
  environment: 'production',
});

describe('PricingSection', () => {
  it('renders demo placeholder tiers when pricing data is missing', () => {
    const pricingOverview = create(PricingOverviewSchema, { bundle });

    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);

    expect(screen.getByText('Launch (Demo)')).toBeDefined();
    expect(screen.getByText('Pro (Demo)')).toBeDefined();
  });

  it('renders remote pricing tiers when arrays contain plans', () => {
    const pricingOverview = create(PricingOverviewSchema, {
      bundle,
      monthly: [
        create(PlanOptionSchema, {
          planName: 'Solo Monthly',
          planTier: 'solo',
          billingInterval: BillingInterval.MONTH,
          amountCents: 4900n,
          currency: 'usd',
          introEnabled: false,
          stripePriceId: 'price_solo',
          monthlyIncludedCredits: 5_000_000n,
          oneTimeBonusCredits: 0n,
          planRank: 1,
          bonusType: 'none',
          displayWeight: 10,
          metadata: recordToJsonMap({
            features: ['Solo workspace'],
          }),
        }),
      ],
    });

    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);

    expect(screen.getByText('Solo Monthly')).toBeDefined();
    expect(screen.queryByText('Starter')).toBeNull();
  });

  it('pads monthly pricing tiers up to three cards when less data is available', () => {
    const pricingOverview = create(PricingOverviewSchema, {
      bundle,
      monthly: [
        create(PlanOptionSchema, {
          planName: 'Only Plan',
          planTier: 'solo',
          billingInterval: BillingInterval.MONTH,
          amountCents: 4900n,
          currency: 'usd',
          introEnabled: false,
          stripePriceId: 'price_only',
          monthlyIncludedCredits: 5_000_000n,
          oneTimeBonusCredits: 0n,
          planRank: 1,
          bonusType: 'none',
          displayWeight: 10,
          metadata: recordToJsonMap({
            features: ['Solo workspace'],
          }),
        }),
      ],
      yearly: [],
    });

    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);

    expect(screen.getAllByTestId(/pricing-tier-/i)).toHaveLength(3);
  });

  it('renders static content tiers with a featured plan and handles CTA clicks', async () => {
    const { default: userEvent } = await import('@testing-library/user-event');
    const user = userEvent.setup();
    render(
      <PricingSection
        content={{
          title: 'Plans',
          tiers: [
            { name: 'Starter', price: '$0', description: 'Free tier', features: ['Basic'], cta_text: 'Start', cta_url: '/start' },
            { name: 'Growth', price: '$49', description: 'For teams', highlighted: true, features: ['Everything'], cta_text: 'Upgrade', cta_url: '/upgrade' },
          ] as never,
        }}
      />,
    );
    expect(screen.getByText('Starter')).toBeInTheDocument();
    expect(screen.getByText('Growth')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Upgrade' }));
  });

  it('exposes intro pricing and bonus credits from a remote plan', () => {
    const pricingOverview = create(PricingOverviewSchema, {
      bundle,
      monthly: [
        create(PlanOptionSchema, {
          planName: 'Pro Monthly',
          planTier: 'pro',
          billingInterval: BillingInterval.MONTH,
          amountCents: 9900n,
          currency: 'usd',
          introEnabled: true,
          introAmountCents: 4900n,
          introPeriods: 3,
          stripePriceId: 'price_pro',
          monthlyIncludedCredits: 10_000_000n,
          oneTimeBonusCredits: 2_000_000n,
          bonusType: 'signup_bonus',
          planRank: 2,
          displayWeight: 20,
          metadata: recordToJsonMap({ features: ['Everything in Solo'], badge: 'Most popular', subtitle: 'For teams' }),
        }),
      ],
      yearly: [
        create(PlanOptionSchema, {
          planName: 'Pro Yearly',
          planTier: 'pro',
          billingInterval: BillingInterval.YEAR,
          amountCents: 99000n,
          currency: 'usd',
          stripePriceId: 'price_pro_year',
          monthlyIncludedCredits: 10_000_000n,
          planRank: 2,
          displayWeight: 20,
          metadata: recordToJsonMap({ features: ['Everything in Solo'] }),
        }),
      ],
    });
    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);
    expect(screen.getByText('Pro Monthly')).toBeInTheDocument();
    expect(screen.getAllByText(/Most popular/).length).toBeGreaterThan(0);
  });
});
