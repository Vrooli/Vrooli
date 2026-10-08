import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { afterEach, describe, expect, it, vi } from 'vitest';
import { CouponCard } from './CouponCard';

const coupon = {
  id: 'intro-pro', name: 'Pro launch offer', percent_off: 25, duration: 'repeating' as const,
  duration_in_months: 3, times_redeemed: 4, max_redemptions: 20, created: 1_704_067_200,
  redeem_by: 1_735_689_600, valid: true, is_intro_coupon: true, intro_tier: 'pro',
};

afterEach(() => { vi.restoreAllMocks(); });

describe('CouponCard', () => {
  it('renders subscription discount, intro-pricing, duration, and local usage information', () => {
    render(<CouponCard coupon={coupon} usageStats={{ coupon_id: 'intro-pro', total_uses: 2 }} onDelete={vi.fn()} isDeleting={false} onEdit={vi.fn()} />);

    expect(screen.getByText('intro-pro')).toBeInTheDocument();
    expect(screen.getByText('25% off')).toBeInTheDocument();
    expect(screen.getByText('Intro: pro')).toBeInTheDocument();
    expect(screen.getByText('3 months')).toBeInTheDocument();
    expect(screen.getByText('Used: 4 / 20')).toBeInTheDocument();
    expect(screen.getByText('2 local intro uses')).toBeInTheDocument();
    expect(screen.getByTitle('Edit coupon')).toBeEnabled();
  });

  it('requires confirmation before deleting an intro coupon and preserves it when declined', () => {
    const onDelete = vi.fn().mockResolvedValue({ success: true });
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false);
    render(<CouponCard coupon={coupon} onDelete={onDelete} isDeleting={false} />);

    fireEvent.click(screen.getByTitle('Delete coupon'));
    expect(confirm).toHaveBeenCalledWith(expect.stringContaining('break intro pricing'));
    expect(onDelete).not.toHaveBeenCalled();
  });

  it('calls edit and delete handlers for a confirmed regular fixed-amount coupon', () => {
    const onDelete = vi.fn().mockResolvedValue({ success: true });
    const onEdit = vi.fn();
    vi.spyOn(window, 'confirm').mockReturnValue(true);
    render(<CouponCard coupon={{ ...coupon, is_intro_coupon: false, amount_off: 1299, percent_off: null, currency: 'usd', duration: 'forever' }} onDelete={onDelete} onEdit={onEdit} isDeleting={false} />);

    fireEvent.click(screen.getByTitle('Edit coupon'));
    fireEvent.click(screen.getByTitle('Delete coupon'));
    expect(onEdit).toHaveBeenCalledWith(expect.objectContaining({ id: 'intro-pro' }));
    expect(onDelete).toHaveBeenCalledWith('intro-pro');
    expect(screen.getByText('$12.99 USD off')).toBeInTheDocument();
    expect(screen.getByText('Forever')).toBeInTheDocument();
  });

  it('marks invalid coupons as expired and blocks duplicate deletion requests', () => {
    render(<CouponCard coupon={{ ...coupon, valid: false }} onDelete={vi.fn()} isDeleting />);
    expect(screen.getByText('Expired')).toBeInTheDocument();
    expect(screen.getByTitle('Delete coupon')).toBeDisabled();
  });
});
