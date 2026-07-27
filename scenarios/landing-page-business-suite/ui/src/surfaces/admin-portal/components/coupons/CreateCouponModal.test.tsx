import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { CreateCouponModal } from './CreateCouponModal';

describe('CreateCouponModal', () => {
  it('does not render or submit while closed', () => {
    render(<CreateCouponModal isOpen={false} onClose={vi.fn()} onCreate={vi.fn()} creating={false} error={null} />);
    expect(screen.queryByText('Create Coupon')).not.toBeInTheDocument();
  });

  it('normalizes a percentage coupon payload and clears its form after a successful creation', async () => {
    const onCreate = vi.fn().mockResolvedValue({ success: true });
    render(<CreateCouponModal isOpen onClose={vi.fn()} onCreate={onCreate} creating={false} error={null} />);

    fireEvent.change(screen.getByPlaceholderText('e.g., SUMMER_SALE_20'), { target: { value: 'summer sale!20' } });
    fireEvent.change(screen.getByPlaceholderText('e.g., Summer Sale 20% Off'), { target: { value: ' Summer launch ' } });
    fireEvent.change(screen.getByPlaceholderText('e.g., 20'), { target: { value: '25' } });
    fireEvent.change(screen.getByPlaceholderText('Unlimited if blank'), { target: { value: '50' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create Coupon' }));

    await waitFor(() => { expect(onCreate).toHaveBeenCalledWith({ id: 'SUMMERSALE20', name: 'Summer launch', percent_off: 25, duration: 'once', max_redemptions: 50 }); });
    expect(screen.getByPlaceholderText('e.g., SUMMER_SALE_20')).toHaveValue('');
  });

  it('rejects invalid discount input before it can reach Stripe', () => {
    const onCreate = vi.fn();
    render(<CreateCouponModal isOpen onClose={vi.fn()} onCreate={onCreate} creating={false} error="Create failed" />);

    fireEvent.change(screen.getByPlaceholderText('e.g., 20'), { target: { value: '101' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create Coupon' }));
    expect(onCreate).not.toHaveBeenCalled();
    expect(screen.getByText('Create failed')).toBeInTheDocument();
  });

  it('resets and closes safely through the cancellation action', () => {
    const onClose = vi.fn();
    render(<CreateCouponModal isOpen onClose={onClose} onCreate={vi.fn()} creating={false} error={null} />);
    fireEvent.change(screen.getByPlaceholderText('e.g., 20'), { target: { value: '10' } });
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onClose).toHaveBeenCalledOnce();
    expect(screen.getByPlaceholderText('e.g., 20')).toHaveValue(null);
  });

  it('creates a repeating fixed-amount coupon with currency and expiration', async () => {
    const onCreate = vi.fn().mockResolvedValue({ success: true });
    render(<CreateCouponModal isOpen onClose={vi.fn()} onCreate={onCreate} creating={false} error={null} />);

    fireEvent.click(screen.getByRole('button', { name: 'Amount Off' }));
    fireEvent.change(screen.getByPlaceholderText('e.g., 10.00'), { target: { value: '12.34' } });
    const [currency, duration] = screen.getAllByRole('combobox');
    fireEvent.click(currency!);
    fireEvent.click(await screen.findByRole('option', { name: 'EUR' }));
    fireEvent.click(duration!);
    fireEvent.click(await screen.findByRole('option', { name: /Repeating/ }));
    fireEvent.change(screen.getByPlaceholderText('e.g., 3'), { target: { value: '3' } });
    const expiration = document.querySelector('input[type="date"]');
    if (!(expiration instanceof HTMLInputElement)) throw new Error('expiration field was not rendered');
    fireEvent.change(expiration, { target: { value: '2030-01-02' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create Coupon' }));

    await waitFor(() => expect(onCreate).toHaveBeenCalledWith(expect.objectContaining({
      amount_off: 1234, currency: 'eur', duration: 'repeating', duration_in_months: 3,
      redeem_by: Math.floor(new Date('2030-01-02').getTime() / 1000),
    })));
  });
});
