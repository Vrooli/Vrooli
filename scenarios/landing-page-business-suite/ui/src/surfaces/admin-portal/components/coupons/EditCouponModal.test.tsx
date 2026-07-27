import { fireEvent, screen, waitFor } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { EditCouponModal } from './EditCouponModal';

const coupon = { id: 'launch-20', name: 'Launch discount', percent_off: 20, duration: 'repeating' as const, duration_in_months: 3, times_redeemed: 0, valid: true, created: 1_704_067_200, is_intro_coupon: false };

describe('EditCouponModal', () => {
  it('does not create a dialog without a selected coupon', () => {
    render(<EditCouponModal coupon={null} isOpen onClose={vi.fn()} onSave={vi.fn()} saving={false} />);
    expect(screen.queryByText('Edit Coupon')).not.toBeInTheDocument();
  });

  it('saves the friendly name only and closes after persistence succeeds', async () => {
    const onClose = vi.fn();
    const onSave = vi.fn().mockResolvedValue({ success: true });
    render(<EditCouponModal coupon={coupon} isOpen onClose={onClose} onSave={onSave} saving={false} />);

    expect(screen.getByText('20% off')).toBeInTheDocument();
    expect(screen.getByText('Duration: repeating (3 months)')).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText('Enter a display name (optional)'), { target: { value: 'Spring campaign' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => { expect(onSave).toHaveBeenCalledWith('launch-20', 'Spring campaign'); });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('keeps the dialog open and reports an update failure', async () => {
    const onClose = vi.fn();
    render(<EditCouponModal coupon={coupon} isOpen onClose={onClose} onSave={vi.fn().mockResolvedValue({ success: false, error: 'Coupon no longer exists' })} saving={false} />);
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    expect(await screen.findByText('Coupon no longer exists')).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
  });
});
