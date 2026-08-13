/* eslint-disable @typescript-eslint/unbound-method -- assertions exercise Vitest/browser mocks, not detached production methods. */
import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { BillingDashboard } from './BillingDashboard';
import * as adminHome from '../hooks/useAdminHome';

const navigate = vi.fn();
vi.mock('react-router-dom', () => ({ useNavigate: () => navigate }));
vi.mock('../hooks/useAdminHome');
vi.mock('../components/AdminLayout', () => ({ AdminLayout: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('../components/PageHeader', () => ({ PageHeader: ({ title }: { title: string }) => <h1>{title}</h1> }));

function homeState(overrides: Record<string, unknown> = {}) {
  return {
    stripeSettings: null, stripeLoading: false, stripeError: null, refreshStripeStatus: vi.fn(),
    ...overrides,
  } as unknown as ReturnType<typeof adminHome.useAdminHome>;
}

describe('BillingDashboard', () => {
  beforeEach(() => { vi.clearAllMocks(); vi.stubGlobal('open', vi.fn()); });

  it('presents payment setup guidance and routes every revenue-management quick flow', () => {
    const state = homeState();
    vi.mocked(adminHome.useAdminHome).mockReturnValue(state);
    render(<BillingDashboard />);
    expect(screen.getByText('Set up Stripe to accept payments')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('flow-stripe'));
    fireEvent.click(screen.getByTestId('flow-plans'));
    fireEvent.click(screen.getByTestId('flow-api-keys'));
    fireEvent.click(screen.getByTestId('flow-coupons'));
    fireEvent.click(screen.getByTestId('billing-guidance-setup'));
    expect(navigate).toHaveBeenNthCalledWith(1, '/admin/billing');
    expect(navigate).toHaveBeenNthCalledWith(2, '/admin/tiers');
    expect(navigate).toHaveBeenNthCalledWith(3, '/admin/api-keys');
    expect(navigate).toHaveBeenNthCalledWith(4, '/admin/coupons');
    expect(navigate).toHaveBeenLastCalledWith('/admin/billing');
    fireEvent.click(screen.getByRole('button', { name: 'Open Stripe dashboard' }));
    expect(window.open).toHaveBeenCalledWith('https://dashboard.stripe.com/apikeys', '_blank', 'noopener,noreferrer');
  });

  it('offers retry while Stripe status fetch fails and reports partial configuration', () => {
    const failed = homeState({ stripeError: 'Stripe status unavailable' });
    vi.mocked(adminHome.useAdminHome).mockReturnValue(failed);
    const { rerender } = render(<BillingDashboard />);
    expect(screen.getByText('Stripe status unavailable')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(failed.refreshStripeStatus).toHaveBeenCalledOnce();
    const partial = homeState({ stripeSettings: { publishable_key_set: true, secret_key_set: false, webhook_secret_set: false } });
    vi.mocked(adminHome.useAdminHome).mockReturnValue(partial);
    rerender(<BillingDashboard />);
    expect(screen.getByText('Stripe partially configured')).toBeInTheDocument();
    expect(screen.getByText('Configured')).toBeInTheDocument();
    expect(screen.getAllByText('Missing')).toHaveLength(2);
  });

  it('shows a ready state only when all Stripe credentials are configured and refreshes status', () => {
    const state = homeState({ stripeSettings: { publishable_key_set: true, secret_key_set: true, webhook_secret_set: true, source: 'database', updated_at: '2026-01-01T00:00:00Z' } });
    vi.mocked(adminHome.useAdminHome).mockReturnValue(state);
    render(<BillingDashboard />);
    expect(screen.getByText('Payments are ready')).toBeInTheDocument();
    expect(screen.queryByTestId('billing-guidance-setup')).not.toBeInTheDocument();
    fireEvent.click(screen.getByTestId('billing-stripe-refresh'));
    expect(state.refreshStripeStatus).toHaveBeenCalledOnce();
    expect(screen.getByText('Source: Admin override')).toBeInTheDocument();
  });
});
