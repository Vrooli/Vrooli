import { screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AddPlanModal } from './AddPlanModal';
import { createBundlePrice, verifyStripePrice } from '../../../../shared/api/billing';

vi.mock('../../../../shared/api/billing', () => ({
  createBundlePrice: vi.fn(),
  verifyStripePrice: vi.fn(),
}));

const props = () => ({ bundleKey: 'business-suite', isOpen: true, onClose: vi.fn(), onSuccess: vi.fn() });

describe('AddPlanModal', () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it('validates a Stripe price ID before verifying or creating a plan', async () => {
    const user = userEvent.setup();
    render(<AddPlanModal {...props()} />);

    await user.type(screen.getByPlaceholderText('price_...'), 'invalid');
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));

    expect(screen.getByText('Stripe Price IDs must start with "price_".')).toBeInTheDocument();
    expect(createBundlePrice).not.toHaveBeenCalled();
  });

  it('verifies Stripe details, creates a normalized plan, and closes on success', async () => {
    const user = userEvent.setup();
    const modalProps = props();
    vi.mocked(verifyStripePrice).mockResolvedValue({
      id: 'price_pro_monthly', product: 'Pro Monthly', currency: 'USD', amount_cents: 7900, interval: 'month', active: true,
    });
    vi.mocked(createBundlePrice).mockResolvedValue({} as Awaited<ReturnType<typeof createBundlePrice>>);
    render(<AddPlanModal {...modalProps} />);

    await user.type(screen.getByPlaceholderText('price_...'), 'price_pro_monthly');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));
    expect(await screen.findByText('Price verified and fields auto-filled')).toBeInTheDocument();
    await user.type(screen.getByPlaceholderText('For growing teams'), 'Built for teams');
    await user.type(screen.getByPlaceholderText('Popular'), 'Recommended');
    await user.type(screen.getByPlaceholderText('Get Started'), 'Start now');
    await user.type(screen.getByPlaceholderText(/Unlimited projects/), 'Priority support\nAnalytics');
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));

    await waitFor(() => { expect(createBundlePrice).toHaveBeenCalledWith('business-suite', expect.objectContaining({
      stripe_price_id: 'price_pro_monthly', plan_name: 'Pro Monthly', amount_cents: 7900, currency: 'usd',
      subtitle: 'Built for teams', badge: 'Recommended', cta_label: 'Start now', features: ['Priority support', 'Analytics'],
    })); });
    expect(modalProps.onSuccess).toHaveBeenCalledOnce();
    expect(modalProps.onClose).toHaveBeenCalledOnce();
  });

  it('rejects a verified Stripe price when the submitted amount no longer matches it', async () => {
    const user = userEvent.setup();
    vi.mocked(verifyStripePrice).mockResolvedValue({ id: 'price_pro', product: 'Pro', currency: 'usd', amount_cents: 7900, interval: 'month' });
    render(<AddPlanModal {...props()} />);

    await user.type(screen.getByPlaceholderText('price_...'), 'price_pro');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));
    await screen.findByText('Price verified and fields auto-filled');
    await user.clear(screen.getByPlaceholderText('7900'));
    await user.type(screen.getByPlaceholderText('7900'), '8000');
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));

    expect(screen.getByText('Amount (cents) must match the verified Stripe price.')).toBeInTheDocument();
    expect(createBundlePrice).not.toHaveBeenCalled();
  });

  it('reports unsupported Stripe billing intervals instead of treating them as valid plans', async () => {
    const user = userEvent.setup();
    vi.mocked(verifyStripePrice).mockResolvedValue({ id: 'price_usage', interval: 'week' });
    render(<AddPlanModal {...props()} />);

    await user.type(screen.getByPlaceholderText('price_...'), 'price_usage');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));

    expect(await screen.findByText('Stripe interval "week" is not supported. Use month, year, or one_time.')).toBeInTheDocument();
  });

  it('submits selected catalog settings and display switches after Stripe verification', async () => {
    const user = userEvent.setup();
    vi.mocked(verifyStripePrice).mockResolvedValue({
      id: 'price_business', product: 'Business', currency: 'usd', amount_cents: 12000, interval: 'month', active: true,
    });
    vi.mocked(createBundlePrice).mockResolvedValue({} as Awaited<ReturnType<typeof createBundlePrice>>);
    render(<AddPlanModal {...props()} />);

    await user.type(screen.getByPlaceholderText('price_...'), 'price_business');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));
    await screen.findByText('Price verified and fields auto-filled');

    const planTierSelect = screen.getAllByRole('combobox')[0];
    if (!planTierSelect) throw new Error('plan tier selector was not rendered');
    await user.click(planTierSelect);
    await user.click(screen.getByRole('option', { name: 'Business' }));
    await user.clear(screen.getByPlaceholderText('10'));
    await user.type(screen.getByPlaceholderText('10'), '25');
    await user.type(screen.getByPlaceholderText('1000000'), '5000');
    for (const toggle of screen.getAllByRole('switch')) await user.click(toggle);
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));

    await waitFor(() => { expect(createBundlePrice).toHaveBeenCalledWith('business-suite', expect.objectContaining({
      plan_tier: 'business', billing_interval: 'month', currency: 'usd', display_weight: 25,
      monthly_included_credits: 5000, display_enabled: false, highlight: true,
    })); });
  });

  it('rejects invalid credits before sending a catalog mutation', async () => {
    const user = userEvent.setup();
    vi.mocked(verifyStripePrice).mockResolvedValue({ id: 'price_pro', amount_cents: 7900, interval: 'month' });
    render(<AddPlanModal {...props()} />);
    await user.type(screen.getByPlaceholderText('price_...'), 'price_pro');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));
    await screen.findByText('Price verified and fields auto-filled');
    await user.type(screen.getByPlaceholderText('Pro Monthly'), 'Pro');
    await user.type(screen.getByPlaceholderText('1000000'), '-1');
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));
    expect(screen.getByText('Included credits must be a non-negative number.')).toBeInTheDocument();
    expect(createBundlePrice).not.toHaveBeenCalled();
  });

  it('keeps verification failures visible and clears verified state when the price ID changes', async () => {
    const user = userEvent.setup();
    vi.mocked(verifyStripePrice).mockRejectedValue(new Error('Stripe is unavailable'));
    render(<AddPlanModal {...props()} />);

    await user.type(screen.getByPlaceholderText('price_...'), 'price_unavailable');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));
    expect(await screen.findByText('Stripe is unavailable')).toBeInTheDocument();

    await user.clear(screen.getByPlaceholderText('price_...'));
    await user.type(screen.getByPlaceholderText('price_...'), 'price_changed');
    expect(screen.queryByText('Stripe is unavailable')).not.toBeInTheDocument();
  });

  it('rejects free plans with non-zero amounts before mutation', async () => {
    const user = userEvent.setup();
    vi.mocked(verifyStripePrice).mockResolvedValue({ id: 'price_free', product: 'Free', amount_cents: 100, interval: 'month' });
    render(<AddPlanModal {...props()} />);
    await user.type(screen.getByPlaceholderText('price_...'), 'price_free');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));
    await screen.findByText('Price verified and fields auto-filled');
    await user.click(screen.getAllByRole('combobox')[0]!);
    await user.click(screen.getByRole('option', { name: 'Free' }));
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));

    expect(screen.getByText('Free plans must have amount 0. Create a $0 Stripe price for free tiers.')).toBeInTheDocument();
    expect(createBundlePrice).not.toHaveBeenCalled();
  });

  it('validates missing plan details before saving', async () => {
    const user = userEvent.setup();
    render(<AddPlanModal {...props()} />);
    await user.type(screen.getByPlaceholderText('price_...'), 'price_incomplete');
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));
    expect(screen.getByText('Plan Name is required')).toBeInTheDocument();
  });

  it('rejects verified interval and currency changes before creating a plan', async () => {
    const user = userEvent.setup();
    vi.mocked(verifyStripePrice).mockResolvedValue({ id: 'price_pro', product: 'Pro', currency: 'usd', amount_cents: 7900, interval: 'month' });
    render(<AddPlanModal {...props()} />);
    await user.type(screen.getByPlaceholderText('price_...'), 'price_pro');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));
    await screen.findByText('Price verified and fields auto-filled');
    await user.click(screen.getAllByRole('combobox')[1]!);
    await user.click(screen.getByRole('option', { name: 'Yearly' }));
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));
    expect(screen.getByText('Billing interval must match the verified Stripe price.')).toBeInTheDocument();
    await user.click(screen.getAllByRole('combobox')[1]!);
    await user.click(screen.getByRole('option', { name: 'Monthly' }));
    await user.click(screen.getAllByRole('combobox')[2]!);
    await user.click(screen.getByRole('option', { name: 'EUR' }));
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));
    expect(screen.getByText('Currency must match the verified Stripe price.')).toBeInTheDocument();
  });

  it('keeps the dialog open and reports a safe error when catalog creation fails', async () => {
    const user = userEvent.setup();
    vi.mocked(verifyStripePrice).mockResolvedValue({ id: 'price_failure', product: 'Failure plan', currency: 'usd', amount_cents: 500, interval: 'month' });
    vi.mocked(createBundlePrice).mockRejectedValue('offline');
    const modalProps = props();
    render(<AddPlanModal {...modalProps} />);
    await user.type(screen.getByPlaceholderText('price_...'), 'price_failure');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));
    await screen.findByText('Price verified and fields auto-filled');
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));
    expect(await screen.findByText('Failed to create plan')).toBeInTheDocument();
    expect(modalProps.onClose).not.toHaveBeenCalled();
  });

  it('preserves operator-entered fields when Stripe returns only a partial price record', async () => {
    const user = userEvent.setup();
    vi.mocked(verifyStripePrice).mockResolvedValue({ id: '', active: false });
    render(<AddPlanModal {...props()} />);
    await user.type(screen.getByPlaceholderText('price_...'), 'price_partial');
    await user.type(screen.getByPlaceholderText('Pro Monthly'), 'Manual name');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));
    await screen.findByText('Price verified and fields auto-filled');
    expect(screen.getByPlaceholderText('price_...')).toHaveValue('price_partial');
    expect(screen.getByPlaceholderText('Pro Monthly')).toHaveValue('Manual name');
    expect(screen.getAllByRole('switch')[0]).toHaveAttribute('data-state', 'unchecked');
  });

  it('normalizes Stripe one-time interval aliases and retains actionable API error details', async () => {
    const user = userEvent.setup();
    vi.mocked(verifyStripePrice).mockResolvedValue({ id: 'price_once', product: 'One time', currency: 'usd', amount_cents: 500, interval: 'one time' });
    vi.mocked(createBundlePrice).mockRejectedValue(new Error('Catalog write rejected'));
    render(<AddPlanModal {...props()} />);
    await user.type(screen.getByPlaceholderText('price_...'), 'price_once');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));
    await screen.findByText('Price verified and fields auto-filled');
    expect(screen.getAllByRole('combobox')[1]).toHaveTextContent('One-time');
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));
    expect(await screen.findByText('Catalog write rejected')).toBeInTheDocument();
  });

  it('blocks malformed and negative amounts before a price mutation', async () => {
    const user = userEvent.setup();
    vi.mocked(verifyStripePrice).mockResolvedValue({ id: 'price_amount', product: 'Amount plan', currency: 'usd', amount_cents: 500, interval: 'month' });
    render(<AddPlanModal {...props()} />);
    await user.type(screen.getByPlaceholderText('price_...'), 'price_amount');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));
    await screen.findByText('Price verified and fields auto-filled');
    const amount = screen.getByPlaceholderText('7900');
    await user.clear(amount);
    await user.type(amount, '-1');
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));
    expect(screen.getByText('Amount (cents) must be a valid non-negative number.')).toBeInTheDocument();
    expect(createBundlePrice).not.toHaveBeenCalled();
  });

  it('blocks negative display weights before a catalog mutation', async () => {
    const user = userEvent.setup();
    vi.mocked(verifyStripePrice).mockResolvedValue({ id: 'price_weight', product: 'Weight plan', currency: 'usd', amount_cents: 500, interval: 'month' });
    render(<AddPlanModal {...props()} />);
    await user.type(screen.getByPlaceholderText('price_...'), 'price_weight');
    await user.click(screen.getByRole('button', { name: 'Verify Stripe price' }));
    await screen.findByText('Price verified and fields auto-filled');
    const weight = screen.getByPlaceholderText('10');
    await user.clear(weight);
    await user.type(weight, '-1');
    await user.click(screen.getByRole('button', { name: 'Create Plan' }));
    expect(screen.getByText('Display weight must be a non-negative number.')).toBeInTheDocument();
    expect(createBundlePrice).not.toHaveBeenCalled();
  });
});
