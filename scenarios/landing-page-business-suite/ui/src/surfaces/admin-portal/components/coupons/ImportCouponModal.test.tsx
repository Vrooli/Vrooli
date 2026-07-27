import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { ImportCouponModal } from './ImportCouponModal';
import type { UseCouponImportReturn } from '../../hooks/useCouponImport';
import type { CouponImportPreview } from '../../../../shared/api/billing';

const preview: CouponImportPreview = {
  coupons: [
    { id: 'active-20', name: 'Active twenty', percent_off: 20, duration: 'once' as const, times_redeemed: 2, valid: true, exists_locally: true },
    { id: 'expired-10', amount_off: 1000, currency: 'usd', duration: 'forever' as const, times_redeemed: 0, valid: false, exists_locally: false },
  ], total_coupons: 2, existing_count: 1, new_count: 1,
};

function couponImport(overrides: Partial<UseCouponImportReturn> = {}): UseCouponImportReturn {
  return { isModalOpen: true, openModal: vi.fn(), closeModal: vi.fn(), preview, loading: false, error: null, refreshPreview: vi.fn(), ...overrides };
}

describe('ImportCouponModal', () => {
  it('presents Stripe coupon state, assignment data, discounts, and safe external management link', () => {
    render(<ImportCouponModal couponImport={couponImport()} />);
    expect(screen.getByText('2 coupons found')).toBeInTheDocument();
    expect(screen.getByText('1 assigned to plans')).toBeInTheDocument();
    expect(screen.getByText('20% off')).toBeInTheDocument();
    expect(screen.getByText('$10.00 USD off')).toBeInTheDocument();
    expect(screen.getByText('Expired/Invalid')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Manage in Stripe/ })).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('filters available coupons and requests a fresh Stripe preview', () => {
    const state = couponImport();
    render(<ImportCouponModal couponImport={state} />);
    fireEvent.click(screen.getByRole('button', { name: 'Active' }));
    expect(screen.getByText('Active twenty')).toBeInTheDocument();
    expect(screen.queryByText('$10.00 USD off')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
    expect(state.refreshPreview).toHaveBeenCalledOnce();
  });

  it('keeps close and failure feedback available when Stripe preview cannot load', () => {
    const state = couponImport({ preview: null, error: 'Stripe credentials are missing' });
    render(<ImportCouponModal couponImport={state} />);
    expect(screen.getByText('Stripe credentials are missing')).toBeInTheDocument();
    fireEvent.click(screen.getAllByRole('button', { name: 'Close' })[0]!);
    expect(state.closeModal).toHaveBeenCalledOnce();
  });

  it('shows an isolated loading state instead of stale coupon data', () => {
    render(<ImportCouponModal couponImport={couponImport({ loading: true, preview: null })} />);
    expect(screen.getByText('Loading coupons from Stripe...')).toBeInTheDocument();
    expect(screen.queryByText('Active twenty')).not.toBeInTheDocument();
  });
});
