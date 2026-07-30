import { beforeEach, describe, it, expect, vi } from 'vitest';
import { renderWithProviders as render } from "../../../test-utils/renderWithProviders";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { PricingSection } from './PricingSection';
import { createCheckoutSession, type PricingOverview } from '../../../shared/api';

vi.mock('../../../shared/hooks/useMetricsHook', () => ({
  useMetrics: () => ({
    trackCTAClick: vi.fn(),
  }),
}));

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return { ...actual, createCheckoutSession: vi.fn() };
});

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
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders demo placeholder tiers when pricing data is missing', () => {
    const pricingOverview: PricingOverview = {
      bundle,
      monthly: null as unknown as PricingOverview['monthly'],
      yearly: null as unknown as PricingOverview['yearly'],
      updated_at: '2025-01-01T00:00:00Z',
    };

    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);

    expect(screen.getByRole('heading', { name: 'Solo' })).toBeDefined();
    expect(screen.getByRole('heading', { name: 'Studio' })).toBeDefined();
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
          display_enabled: true,
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

    expect(screen.getAllByText('Solo Monthly').length).toBeGreaterThan(0);
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
          display_enabled: true,
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

  it('switches to yearly plans, shows calculated savings, and renders coupon-aware plan copy', async () => {
    const pricingOverview: PricingOverview = {
      bundle,
      monthly: [{
        plan_name: 'Growth Monthly', plan_tier: 'growth', billing_interval: 'month', amount_cents: 2000,
        currency: 'usd', intro_enabled: false, stripe_price_id: 'price-month', monthly_included_credits: 2_000_000,
        one_time_bonus_credits: 1000, plan_rank: 2, bonus_type: 'launch_bonus', display_enabled: true, display_weight: 2,
        metadata: { features: ['Monthly feature'], subtitle: 'For teams' },
      }],
      yearly: [{
        plan_name: 'Growth Yearly', plan_tier: 'growth', billing_interval: 'year', amount_cents: 19200,
        currency: 'usd', intro_enabled: false, stripe_price_id: 'price-year', monthly_included_credits: 2_000_000,
        one_time_bonus_credits: 0, plan_rank: 2, bonus_type: 'none', display_enabled: true, display_weight: 2,
        metadata: { features: ['Yearly feature'] },
      }],
      updated_at: '2025-01-01T00:00:00Z',
    };

    render(
      <PricingSection
        content={{ title: 'Pricing' }}
        pricingOverview={pricingOverview}
        couponMappings={{ 'price-year': 'coupon-year' }}
        availableCoupons={[{ id: 'coupon-year', percent_off: 20, duration: 'once' } as never]}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Yearly billing' }));
    expect(await screen.findByRole('heading', { name: 'Growth Yearly' })).toBeInTheDocument();
    expect(screen.getByText('Save 20% yearly')).toBeInTheDocument();
    expect(screen.getAllByText(/Start with 20% off/i)).toHaveLength(2);
    expect(screen.getByText('Yearly feature')).toBeInTheDocument();
  });

  it('handles checkout failures and lets customers dismiss the mobile featured-plan prompt', async () => {
    vi.mocked(createCheckoutSession).mockRejectedValue(new Error('Checkout is unavailable'));
    const pricingOverview: PricingOverview = {
      bundle,
      monthly: [{
        plan_name: 'Growth', plan_tier: 'growth', billing_interval: 'month', amount_cents: 2000,
        currency: 'usd', intro_enabled: false, stripe_price_id: 'price-growth', monthly_included_credits: 0,
        one_time_bonus_credits: 0, plan_rank: 1, bonus_type: 'none', display_enabled: true, display_weight: 1,
        metadata: { highlight: true },
      }],
      yearly: [], updated_at: '2025-01-01T00:00:00Z',
    };
    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);

    fireEvent.click(screen.getByTestId('pricing-cta-growth'));
    await waitFor(() => {
      expect(createCheckoutSession).toHaveBeenCalledWith(expect.objectContaining({ price_id: 'price-growth' }));
      expect(screen.getByText('Checkout is unavailable')).toBeInTheDocument();
    });
    const dismissButton = screen.getByRole('button', { name: 'Dismiss pricing sticky' });
    expect(within(dismissButton.parentElement!).getByRole('button', { name: 'Choose plan' })).toHaveClass('min-h-11');
    expect(dismissButton).toHaveClass('min-h-11');
    fireEvent.click(dismissButton);
    expect(screen.queryByRole('button', { name: 'Dismiss pricing sticky' })).not.toBeInTheDocument();
  });

  it('communicates native introductory pricing, bonus credits, and custom-sales plans accurately', () => {
    const pricingOverview: PricingOverview = {
      bundle,
      monthly: [
        {
          plan_name: 'Intro Pro', plan_tier: 'pro', billing_interval: 'month', amount_cents: 5000,
          currency: 'usd', intro_enabled: true, intro_amount_cents: 2500, intro_periods: 2,
          stripe_price_id: 'price_intro', monthly_included_credits: 1_500_000, one_time_bonus_credits: 500_000,
          plan_rank: 3, bonus_type: 'launch_bonus', display_enabled: true, display_weight: 3, metadata: {},
        },
        {
          plan_name: 'Enterprise', plan_tier: 'enterprise', billing_interval: 'month', amount_cents: -1,
          currency: 'usd', intro_enabled: false, stripe_price_id: '', monthly_included_credits: 0,
          one_time_bonus_credits: 0, plan_rank: 4, bonus_type: 'none', display_enabled: true, display_weight: 4,
          metadata: {},
        },
      ],
      yearly: [], updated_at: '2025-01-01T00:00:00Z',
    };
    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);

    expect(screen.getByText('$25 intro for 2 months')).toBeInTheDocument();
    expect(screen.getAllByText('Start $25 intro')).toHaveLength(2);
    expect(screen.getByText('1.5k credits included')).toBeInTheDocument();
    expect(screen.getByText('Bonus 500 credits')).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Enterprise' })).not.toBeInTheDocument();
  });

  it('fails safely when Stripe accepts a checkout request but returns no redirect URL', async () => {
    vi.mocked(createCheckoutSession).mockResolvedValue({ id: 'cs_missing_url' } as never);
    const pricingOverview: PricingOverview = {
      bundle,
      monthly: [{
        plan_name: 'Pro', plan_tier: 'pro', billing_interval: 'month', amount_cents: 7900, currency: 'usd', intro_enabled: false,
        stripe_price_id: 'price_pro', monthly_included_credits: 0, one_time_bonus_credits: 0, plan_rank: 1, bonus_type: 'none', display_enabled: true, display_weight: 1, metadata: {},
      }],
      yearly: [], updated_at: '2025-01-01T00:00:00Z',
    };
    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);

    fireEvent.click(screen.getByTestId('pricing-cta-pro'));
    expect(await screen.findByText('Stripe did not return a checkout URL. Try again.')).toBeInTheDocument();
  });

  it('reverts to monthly pricing when yearly plans disappear after a catalog refresh', () => {
    const yearlyOverview: PricingOverview = {
      bundle,
      monthly: [{
        plan_name: 'Monthly Pro', plan_tier: 'pro', billing_interval: 'month', amount_cents: 1000, currency: 'usd', intro_enabled: false,
        stripe_price_id: 'price_month', monthly_included_credits: 0, one_time_bonus_credits: 0, plan_rank: 1, bonus_type: 'none', display_enabled: true, display_weight: 1, metadata: {},
      }],
      yearly: [{
        plan_name: 'Yearly Pro', plan_tier: 'pro', billing_interval: 'year', amount_cents: 10000, currency: 'usd', intro_enabled: false,
        stripe_price_id: 'price_year', monthly_included_credits: 0, one_time_bonus_credits: 0, plan_rank: 1, bonus_type: 'none', display_enabled: true, display_weight: 1, metadata: {},
      }], updated_at: '2025-01-01T00:00:00Z',
    };
    const { rerender } = render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={yearlyOverview} />);
    fireEvent.click(screen.getByRole('button', { name: 'Yearly billing' }));
    expect(screen.getByRole('heading', { name: 'Yearly Pro' })).toBeInTheDocument();

    rerender(<PricingSection content={{ title: 'Pricing' }} pricingOverview={{ ...yearlyOverview, yearly: [] }} />);
    expect(screen.getByRole('heading', { name: 'Monthly Pro' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Yearly billing' })).not.toBeInTheDocument();
  });

  it('filters demo and invalid catalog entries while preserving free and supporter download paths', () => {
    const pricingOverview: PricingOverview = {
      bundle,
      monthly: [
        {
          plan_name: 'Free Access', plan_tier: 'free', billing_interval: 'one_time', amount_cents: 0,
          currency: 'usd', intro_enabled: false, stripe_price_id: '', monthly_included_credits: 0,
          one_time_bonus_credits: 0, display_enabled: true, display_weight: 1, metadata: {},
        },
        {
          plan_name: 'Support the project', plan_tier: 'supporter', billing_interval: 'month', amount_cents: 1200,
          currency: 'usd', intro_enabled: false, stripe_price_id: 'price_support', monthly_included_credits: 0,
          one_time_bonus_credits: 0, kind: 'supporter_contribution', display_enabled: true, display_weight: 2,
          metadata: { badge: 'Community', cta_label: 'Contribute', highlight: true, features: 'not-an-array' } as never,
        },
        {
          plan_name: 'Demo placeholder', plan_tier: 'pro', billing_interval: 'month', amount_cents: 1000,
          currency: 'usd', intro_enabled: false, stripe_price_id: 'demo', monthly_included_credits: 0,
          one_time_bonus_credits: 0, display_enabled: true, display_weight: 3,
          metadata: { __demo_placeholder: true },
        },
        {
          plan_name: 'Broken amount', plan_tier: 'pro', billing_interval: 'month', amount_cents: -1,
          currency: 'usd', intro_enabled: false, stripe_price_id: 'broken', monthly_included_credits: 0,
          one_time_bonus_credits: 0, display_enabled: true, display_weight: 4, metadata: {},
        },
      ],
      yearly: [], updated_at: '2025-01-01T00:00:00Z',
    };

    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);

    expect(screen.getByRole('heading', { name: 'Free Access' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Support the project' })).toBeInTheDocument();
    expect(screen.getByText('Community')).toBeInTheDocument();
    expect(screen.getByTestId('pricing-cta-support-the-project')).toHaveTextContent('Download');
    expect(screen.queryByRole('heading', { name: 'Demo placeholder' })).not.toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Broken amount' })).not.toBeInTheDocument();
  });

  it('renders fallback plan copy for partial catalog records and formats large credit allowances', () => {
    const pricingOverview: PricingOverview = {
      bundle: { ...bundle, display_credits_multiplier: 1, display_credits_label: 'tokens' },
      monthly: [{
        plan_name: 'Scale', plan_tier: 'scale', billing_interval: 1 as never, amount_cents: 1250,
        currency: 'eur', intro_enabled: true, intro_amount_cents: undefined, intro_periods: 1,
        stripe_price_id: '', monthly_included_credits: 1_250_000, one_time_bonus_credits: 1_000,
        plan_rank: 8, bonus_type: 'welcome_bonus', display_enabled: true, display_weight: 1,
      }],
      yearly: [], updated_at: '2025-01-01T00:00:00Z',
    };

    render(<PricingSection content={{}} pricingOverview={pricingOverview} />);

    expect(screen.getByRole('heading', { name: 'Scale' })).toBeInTheDocument();
    expect(screen.getAllByText('€13 / month')).toHaveLength(2);
    expect(screen.getByText('1.3M tokens included')).toBeInTheDocument();
    expect(screen.getByText('Bonus 1k tokens')).toBeInTheDocument();
    expect(screen.getByText('welcome bonus')).toBeInTheDocument();
    expect(screen.getByText('Plan rank #8')).toBeInTheDocument();
    expect(screen.getByTestId('pricing-cta-scale')).toHaveTextContent('Choose plan');
    expect(screen.getByTestId('pricing-cta-scale')).toHaveClass('min-h-11');
  });

  it('keeps paid checkout controls loading only for the selected price and recovers after a successful session response', async () => {
    let resolveCheckout: (value: { url: string }) => void = () => undefined;
    const onCheckoutRedirect = vi.fn();
    vi.mocked(createCheckoutSession).mockImplementationOnce(() => new Promise((resolve) => { resolveCheckout = resolve; }) as never);
    const pricingOverview: PricingOverview = {
      bundle,
      monthly: [
        { plan_name: 'Basic', plan_tier: 'basic', billing_interval: 'month', amount_cents: 1000, currency: 'usd', intro_enabled: false, stripe_price_id: 'price_basic', monthly_included_credits: 0, one_time_bonus_credits: 0, display_enabled: true, display_weight: 1, metadata: {} },
        { plan_name: 'Pro', plan_tier: 'pro', billing_interval: 'month', amount_cents: 2000, currency: 'usd', intro_enabled: false, stripe_price_id: 'price_pro', monthly_included_credits: 0, one_time_bonus_credits: 0, display_enabled: true, display_weight: 2, metadata: {} },
      ],
      yearly: [], updated_at: '2025-01-01T00:00:00Z',
    };
    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} onCheckoutRedirect={onCheckoutRedirect} />);
    fireEvent.click(screen.getByTestId('pricing-cta-basic'));
    expect(screen.getByTestId('pricing-cta-basic')).toHaveTextContent('Redirecting...');
    expect(screen.getByTestId('pricing-cta-pro')).not.toBeDisabled();
    resolveCheckout({ url: 'https://checkout.example.test/session' });
    await waitFor(() => { expect(screen.getByTestId('pricing-cta-basic')).not.toBeDisabled(); });
    expect(onCheckoutRedirect).toHaveBeenCalledWith('https://checkout.example.test/session');
  });

  it('includes a canonical free tier when the remote catalog only contains paid plans', () => {
    const pricingOverview: PricingOverview = {
      bundle,
      monthly: [{ plan_name: 'Paid only', plan_tier: 'pro', billing_interval: 'month', amount_cents: 1000, currency: 'usd', intro_enabled: false, stripe_price_id: 'price_paid', monthly_included_credits: 0, one_time_bonus_credits: 0, display_enabled: true, display_weight: 1, metadata: {} }],
      yearly: [], updated_at: '2025-01-01T00:00:00Z',
    };
    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);
    expect(screen.getByRole('heading', { name: /Free/i })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Paid only' })).toBeInTheDocument();
  });

  it('retains custom catalog records with a zero price only when they are valid free plans', () => {
    const pricingOverview: PricingOverview = {
      bundle,
      monthly: [
        { plan_name: 'Free access', plan_tier: 'free', billing_interval: 'unexpected' as never, amount_cents: 0, currency: 'usd', intro_enabled: false, stripe_price_id: '', monthly_included_credits: 0, one_time_bonus_credits: 0, display_enabled: true, display_weight: 1, metadata: {} },
        { plan_name: 'Invalid interval paid', plan_tier: 'pro', billing_interval: 'unexpected' as never, amount_cents: 1000, currency: 'usd', intro_enabled: false, stripe_price_id: 'price_invalid', monthly_included_credits: 0, one_time_bonus_credits: 0, display_enabled: true, display_weight: 2, metadata: {} },
      ],
      yearly: [{ plan_name: 'Invalid yearly interval', plan_tier: 'pro', billing_interval: 'month', amount_cents: 10000, currency: 'usd', intro_enabled: false, stripe_price_id: 'price_invalid_year', monthly_included_credits: 0, one_time_bonus_credits: 0, display_enabled: true, display_weight: 1, metadata: {} }],
      updated_at: '2025-01-01T00:00:00Z',
    };

    render(<PricingSection content={{ title: 'Pricing' }} pricingOverview={pricingOverview} />);
    expect(screen.getByRole('heading', { name: 'Free access' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Invalid interval paid' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Yearly billing' })).not.toBeInTheDocument();
  });

});
