import { describe, expect, it, vi, beforeEach } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders } from '../../../test-utils/renderWithProviders';
import { CheckoutPage } from './CheckoutPage';
import * as api from '../../../shared/api';
import type { PricingOverview } from '../../../shared/api';
import { ApiError } from '../../../shared/api/common';

vi.mock('../../../shared/api', async () => {
  const actual = await vi.importActual<typeof import('../../../shared/api')>('../../../shared/api');
  return {
    ...actual,
    getPlans: vi.fn(),
    createCheckoutSession: vi.fn(),
  };
});

const getPlans = vi.mocked(api.getPlans);
const createCheckoutSession = vi.mocked(api.createCheckoutSession);

const pricing: PricingOverview = {
  bundle: {
    bundle_key: 'business-suite',
    name: 'Business Suite',
    stripe_product_id: 'prod_test',
    credits_per_usd: 1,
    display_credits_multiplier: 1,
    display_credits_label: 'credits',
    environment: 'test',
  },
  monthly: [{
    plan_name: 'Solo Monthly',
    plan_tier: 'solo',
    billing_interval: 'month',
    amount_cents: 3900,
    currency: 'usd',
    intro_enabled: false,
    stripe_price_id: 'price_solo',
    monthly_included_credits: 100,
    one_time_bonus_credits: 0,
    plan_rank: 1,
    bonus_type: 'none',
    display_enabled: true,
    display_weight: 1,
    metadata: {},
  }],
  yearly: [],
  updated_at: '2026-01-01T00:00:00Z',
};

function renderCheckout(path = '/checkout?price_id=price_solo') {
  return renderWithProviders(<CheckoutPage />, { route: path });
}

describe('CheckoutPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getPlans.mockResolvedValue(pricing);
    createCheckoutSession.mockImplementation(() => new Promise(() => undefined));
  });

  it('selects the requested displayed plan and starts a Stripe checkout session', async () => {
    renderCheckout();

    expect(await screen.findByRole('heading', { name: 'Solo Monthly', level: 2 })).toBeInTheDocument();
    await waitFor(() => {
      expect(createCheckoutSession).toHaveBeenCalledWith(expect.objectContaining({
        price_id: 'price_solo',
      }));
    });
  });

  it('shows a retryable load error when the plan catalog cannot be fetched', async () => {
    getPlans.mockRejectedValue(new Error('Catalog unavailable'));
    renderCheckout();

    expect(await screen.findByText('Unable to Load Checkout')).toBeInTheDocument();
    expect(screen.getByText('Catalog unavailable')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Try Again' })).toBeInTheDocument();
  });

  it('distinguishes network failures and retries loading the catalog', async () => {
    getPlans
      .mockRejectedValueOnce(new ApiError('Offline', 'network'))
      .mockResolvedValueOnce(pricing);
    renderCheckout();

    expect(await screen.findByRole('heading', { name: 'Connection Issue' })).toBeInTheDocument();
    expect(screen.getByText('Unable to connect. Please check your internet connection.')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Try Again' }));
    expect(await screen.findByRole('heading', { name: 'Solo Monthly', level: 2 })).toBeInTheDocument();
  });

  it.each([
    [new ApiError('Maintenance window', 'server_error'), 'Something went wrong on our end. Please try again later.', true],
    [new ApiError('Slow down', 'rate_limited'), 'Too many requests. Please wait a moment and try again.', true],
    [new ApiError('Invalid plan', 'validation'), 'The request was invalid. Please check your input and try again.', false],
    [new ApiError('Plan removed', 'not_found'), 'The requested resource was not found.', false],
  ])('classifies %s catalog errors with the correct retry affordance', async (error, message, retryable) => {
    getPlans.mockRejectedValue(error);
    renderCheckout();

    expect(await screen.findByText(message)).toBeInTheDocument();
    if (retryable) {
      expect(screen.getByRole('button', { name: 'Try Again' })).toBeInTheDocument();
    } else {
      expect(screen.queryByRole('button', { name: 'Try Again' })).toBeNull();
    }
  });

  it('explains when no displayed plans can be checked out', async () => {
    getPlans.mockResolvedValue({ ...pricing, monthly: [{ ...pricing.monthly[0]!, display_enabled: false }] });
    renderCheckout();
    expect(await screen.findByText('No active plans are available right now.')).toBeInTheDocument();
    expect(createCheckoutSession).not.toHaveBeenCalled();
  });

  it('surfaces a missing Stripe redirect URL and retries session creation safely', async () => {
    createCheckoutSession.mockResolvedValue({} as Awaited<ReturnType<typeof api.createCheckoutSession>>);
    renderCheckout();

    expect(await screen.findByText('Checkout Failed')).toBeInTheDocument();
    expect(screen.getByText('Stripe did not return a checkout URL. Try again or contact support.')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Retry Checkout' }));
    await waitFor(() => expect(createCheckoutSession).toHaveBeenCalledTimes(2));
  });

  it('renders selected plan features and falls back to the first active plan', async () => {
    getPlans.mockResolvedValue({
      ...pricing,
      monthly: [{ ...pricing.monthly[0]!, metadata: { features: ['Priority support', 42, 'Analytics'] } as unknown as typeof pricing.monthly[number]['metadata'] }],
    });
    renderCheckout('/checkout?price_id=missing');

    expect(await screen.findByText('Priority support')).toBeInTheDocument();
    expect(screen.getByText('Analytics')).toBeInTheDocument();
    expect(screen.queryByText('42')).not.toBeInTheDocument();
    expect(screen.getByText('$39 / month')).toBeInTheDocument();
  });

  it('shows introductory pricing details and retries a retryable session error', async () => {
    createCheckoutSession.mockRejectedValueOnce(new ApiError('Network interrupted', 'timeout'));
    getPlans.mockResolvedValue({
      ...pricing,
      monthly: [{
        ...pricing.monthly[0]!, intro_enabled: true, intro_amount_cents: 500, intro_periods: 1,
        billing_interval: 'year', metadata: { features: 'invalid' } as never,
      }],
    });
    renderCheckout();

    expect(await screen.findByText('Intro $5 for 1 month')).toBeInTheDocument();
    expect(await screen.findByText('Connection Issue')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Retry Checkout' }));
    await waitFor(() => expect(createCheckoutSession).toHaveBeenCalledTimes(2));
  });
});
