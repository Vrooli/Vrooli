import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import { describe, expect, it, vi } from 'vitest';
import { PriceFormCard } from './PriceFormCard';

const onChange = vi.fn(() => vi.fn());
const baseProps = (demo = false) => ({
  bundleKey: 'business', priceIdentifier: 'price_pro',
  price: { plan_name: 'Pro', stripe_price_id: 'price_pro', currency: 'usd', billing_interval: 'month', amount_cents: 7900, display_enabled: false } as never,
  formState: {
    values: { planName: 'Pro Plus', displayWeight: '20', stripePriceId: 'price_pro', displayEnabled: true, subtitle: '', badge: '', ctaLabel: '', highlight: false, featuresText: '' },
    original: { planName: 'Pro', displayWeight: '10', stripePriceId: 'price_pro', displayEnabled: false, subtitle: '', badge: '', ctaLabel: '', highlight: false, featuresText: '' },
    saving: false, demo, error: 'A saved error',
  } as never,
  onPriceChange: onChange, onSavePrice: vi.fn().mockResolvedValue(undefined), onVerifyPrice: vi.fn().mockResolvedValue(undefined),
  onRemoveDemoPlan: vi.fn(), onDeletePlan: vi.fn(), isCollapsed: false, onToggleCollapse: vi.fn(), planIndex: 1, draggable: true,
  onDragStart: vi.fn(), onDragEnd: vi.fn(), onDragOver: vi.fn(), onDragLeave: vi.fn(), onDrop: vi.fn(),
});

describe('PriceFormCard', () => {
  it('edits, verifies, saves, deletes, reorders, and assigns coupons for a real plan', () => {
    const props = baseProps();
    const assign = vi.fn().mockResolvedValue({ success: true });
    render(<PriceFormCard {...props} priceCheck={{ status: 'ok', message: 'Stripe verified' }} availableCoupons={[{ id: 'coupon_1', name: 'Launch', percent_off: 20 } as never]} onAssignCoupon={assign} onUnassignCoupon={vi.fn()} />);

    fireEvent.change(screen.getByDisplayValue('Pro Plus'), { target: { value: 'Pro Team' } });
    fireEvent.click(screen.getByLabelText('Visible'));
    fireEvent.click(screen.getByLabelText('Highlight tier (apply hero styling)'));
    fireEvent.click(screen.getByRole('button', { name: 'Verify' }));
    fireEvent.click(screen.getByRole('button', { name: 'Save changes' }));
    fireEvent.click(screen.getByTitle('Delete plan'));
    fireEvent.keyDown(screen.getByRole('button', { name: /Pro Plus/ }), { key: 'Enter' });
    fireEvent.click(screen.getByLabelText('Drag to reorder plan'));
    fireEvent.change(screen.getByRole('combobox'), { target: { value: 'coupon_1' } });

    expect(onChange).toHaveBeenCalledWith('business', 'price_pro', 'planName');
    expect(onChange).toHaveBeenCalledWith('business', 'price_pro', 'displayEnabled');
    expect(props.onVerifyPrice).toHaveBeenCalledWith('business', 'price_pro');
    expect(props.onSavePrice).toHaveBeenCalledWith('business', 'price_pro');
    expect(props.onDeletePlan).toHaveBeenCalledWith('business', 'price_pro');
    expect(props.onToggleCollapse).toHaveBeenCalled();
    expect(assign).toHaveBeenCalledWith('price_pro', 'coupon_1');
    expect(screen.getByText('Stripe verified')).toBeInTheDocument();
    expect(screen.getByText('Hidden from landing page visitors')).toBeInTheDocument();
  });

  it('renders demo plan safeguards and removes a demo placeholder', () => {
    const props = baseProps(true);
    render(<PriceFormCard {...props} />);

    fireEvent.click(screen.getByRole('button', { name: 'Remove demo placeholder' }));
    expect(props.onRemoveDemoPlan).toHaveBeenCalledWith('business', 'price_pro');
    expect(screen.getByRole('button', { name: 'Demo plan' })).toBeDisabled();
    expect(screen.getByText('Connect Stripe & reload to edit this slot.')).toBeInTheDocument();
  });

  it('shows an in-progress verification state and a coupon that can be removed', () => {
    const props = baseProps();
    const unassign = vi.fn().mockResolvedValue({ success: true });
    render(
      <PriceFormCard
        {...props}
        priceCheck={{ status: 'checking' }}
        assignedCoupon={{ id: 'coupon_launch', name: 'Launch special', amount_off: 1500, currency: 'usd', duration: 'once', valid: true } as never}
        availableCoupons={[]}
        onAssignCoupon={vi.fn()}
        onUnassignCoupon={unassign}
      />,
    );

    expect(screen.getAllByText('Checking...')).toHaveLength(2);
    expect(screen.getByRole('button', { name: 'Checking...' })).toBeDisabled();
    expect(screen.getByText('Launch special')).toBeInTheDocument();
    expect(screen.getByText(/\$64 \(save \$15\)/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Unassign intro coupon' }));
    expect(unassign).toHaveBeenCalledWith('price_pro');
  });

  it('keeps a collapsed, non-draggable price card concise while retaining its plan identity', () => {
    const props = baseProps();
    render(<PriceFormCard {...props} isCollapsed onToggleCollapse={undefined} draggable={false} onDeletePlan={undefined} priceCheck={{ status: 'error' }} />);

    expect(screen.getByText('Pro Plus')).toBeInTheDocument();
    expect(screen.queryByLabelText('Plan Name')).not.toBeInTheDocument();
    expect(screen.getByLabelText('Drag to reorder plan')).not.toHaveAttribute('title');
    expect(screen.queryByTitle('Delete plan')).not.toBeInTheDocument();
  });
});
