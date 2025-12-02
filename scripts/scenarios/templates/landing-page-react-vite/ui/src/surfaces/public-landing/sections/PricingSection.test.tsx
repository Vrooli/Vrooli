import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PricingSection } from './PricingSection';
import type { PricingOverview } from '../../../shared/api';

vi.mock('../../../shared/hooks/useMetrics', () => ({
  useMetrics: () => ({
    trackCTAClick: vi.fn(),
  }),
}));

const bundle = {
  bundle_key: 'business_suite',
  name: 'Business Suite',
  stripe_product_id: 'prod_123',
  credits_per_usd: 1_000_000,
  display_credits_multiplier: 0.001,
  display_credits_label: 'credits',
  environment: 'production',
};

describe('PricingSection', () => {
  it('renders demo placeholder tiers when pricing data is missing', () => {
    const pricingOverview: PricingOverview = {
      bundle,
      monthly: null as unknown as PricingOverview['monthly'],
      yearly: null as unknown as PricingOverview['yearly'],
      updated_at: '2025-01-01T00:00:00Z',
    };

    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);

    expect(screen.getByText('Launch (Demo)')).toBeDefined();
    expect(screen.getByText('Pro (Demo)')).toBeDefined();
  });

  it('renders remote pricing tiers when arrays contain plans', () => {
    const pricingOverview: PricingOverview = {
      bundle,
      monthly: [
        {
          plan_name: 'Solo Monthly',
          plan_tier: 'solo',
          billing_interval: 'month',
          amount_cents: 4900,
          currency: 'usd',
          intro_enabled: false,
          stripe_price_id: 'price_solo',
          monthly_included_credits: 5_000_000,
          one_time_bonus_credits: 0,
          plan_rank: 1,
          bonus_type: 'none',
          display_weight: 10,
          metadata: {
            features: ['Solo workspace'],
          },
        },
      ],
      yearly: null as unknown as PricingOverview['yearly'],
      updated_at: '2025-01-01T00:00:00Z',
    };

    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);

    expect(screen.getByText('Solo Monthly')).toBeDefined();
    expect(screen.queryByText('Starter')).toBeNull();
  });

  it('pads monthly pricing tiers up to three cards when less data is available', () => {
    const pricingOverview: PricingOverview = {
      bundle,
      monthly: [
        {
          plan_name: 'Only Plan',
          plan_tier: 'solo',
          billing_interval: 'month',
          amount_cents: 4900,
          currency: 'usd',
          intro_enabled: false,
          stripe_price_id: 'price_only',
          monthly_included_credits: 5_000_000,
          one_time_bonus_credits: 0,
          plan_rank: 1,
          bonus_type: 'none',
          display_weight: 10,
          metadata: {
            features: ['Solo workspace'],
          },
        },
      ],
      yearly: [],
      updated_at: '2025-01-01T00:00:00Z',
    };

    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);

    expect(screen.getAllByTestId(/pricing-tier-/i)).toHaveLength(3);
  });
});
