import { fireEvent, screen } from '@testing-library/react';
import { renderWithProviders as render } from '../../../../test-utils/renderWithProviders';
import type React from 'react';
import { describe, expect, it, vi } from 'vitest';
import type { BundleCatalogEntry } from '../../../../shared/api';
import type { PriceFormState } from '../../services/pricing.service';
import { BundleCard } from './BundleCard';

vi.mock('./PriceFormCard', () => ({
  PriceFormCard: ({ priceIdentifier, isCollapsed, onToggleCollapse, draggable, onDragStart, onDragOver, onDragLeave, onDrop, onDragEnd }: {
    priceIdentifier: string; isCollapsed: boolean; onToggleCollapse: () => void; draggable?: boolean;
    onDragStart?: React.DragEventHandler<HTMLElement>; onDragOver?: React.DragEventHandler<HTMLElement>;
    onDragLeave?: React.DragEventHandler<HTMLElement>; onDrop?: React.DragEventHandler<HTMLElement>; onDragEnd?: React.DragEventHandler<HTMLElement>;
  }) => (
    <div draggable={draggable} onDragStart={onDragStart} onDragOver={onDragOver} onDragLeave={onDragLeave} onDrop={onDrop} onDragEnd={onDragEnd}>
      <button type="button" data-testid={`price-form-${priceIdentifier}`} onClick={onToggleCollapse}>{isCollapsed ? 'Collapsed' : 'Expanded'}</button>
    </div>
  ),
}));
vi.mock('./PriceReadOnlyCard', () => ({ PriceReadOnlyCard: ({ price }: { price: { plan_name: string } }) => <div>Read-only {price.plan_name}</div> }));
vi.mock('./PlanPreview', () => ({ PlanPreview: () => <div>Plan preview</div> }));
vi.mock('../../hooks/useResizableColumns', () => ({
  useResizableColumns: () => ({ isResizing: false, containerRef: { current: null }, handleResizeStart: vi.fn(), leftColumnStyle: {}, rightColumnStyle: {} }),
}));

const entry: BundleCatalogEntry = {
  bundle: { bundle_key: 'starter', name: 'Starter', stripe_product_id: 'prod_starter', credits_per_usd: 1, display_credits_multiplier: 1, display_credits_label: 'credits' },
  prices: [
    { plan_name: 'Starter monthly', plan_tier: 'starter', billing_interval: 'month', amount_cents: 1200, currency: 'usd', intro_enabled: false, stripe_price_id: 'price_month', monthly_included_credits: 10, one_time_bonus_credits: 0, display_enabled: true, display_weight: 10 },
    { plan_name: 'Starter yearly', plan_tier: 'starter', billing_interval: 'year', amount_cents: 12000, currency: 'usd', intro_enabled: false, stripe_price_id: 'price_year', monthly_included_credits: 10, one_time_bonus_credits: 0, display_enabled: true, display_weight: 5 },
  ],
};

const form = (priceId: string): PriceFormState => ({
  values: { stripePriceId: priceId, planName: priceId, displayWeight: 1, displayEnabled: true, subtitle: '', badge: '', ctaLabel: '', highlight: false, featuresText: '' },
  original: { stripePriceId: priceId, planName: priceId, displayWeight: 1, displayEnabled: true, subtitle: '', badge: '', ctaLabel: '', highlight: false, featuresText: '' },
  saving: false,
});

function props(overrides: Partial<React.ComponentProps<typeof BundleCard>> = {}) {
  return {
    mode: 'edit' as const,
    entry,
    priceForms: { 'starter:price_month': form('price_month'), 'starter:price_year': form('price_year') },
    pricingTab: 'month' as const,
    includeDemoPlaceholders: false,
    priceChecks: {},
    onPriceChange: vi.fn(),
    onSavePrice: vi.fn(),
    onVerifyPrice: vi.fn(),
    onRemoveDemoPlan: vi.fn(),
    onDeletePlan: vi.fn(),
    onReorderPlans: vi.fn(),
    onAddPlan: vi.fn(),
    ...overrides,
  };
}

describe('BundleCard', () => {
  it('filters by billing tab, expands a plan, and exposes add-plan actions', () => {
    const onAddPlan = vi.fn();
    render(<BundleCard {...props({ onAddPlan })} />);
    expect(screen.getAllByTestId('price-form-price_month')).not.toHaveLength(0);
    expect(screen.queryByTestId('price-form-price_year')).not.toBeInTheDocument();
    fireEvent.click(screen.getAllByTestId('price-form-price_month')[0]!);
    expect(screen.getAllByTestId('price-form-price_month')[0]).toHaveTextContent('Expanded');
    fireEvent.click(screen.getByRole('button', { name: 'Add Plan' }));
    expect(onAddPlan).toHaveBeenCalledWith('starter');
  });

  it('renders read-only plans in preview mode and communicates empty interval results', () => {
    const { rerender } = render(<BundleCard {...props({ mode: 'preview' })} />);
    expect(screen.getAllByText('Read-only Starter monthly')).not.toHaveLength(0);
    rerender(<BundleCard {...props({ pricingTab: 'other' })} />);
    expect(screen.getAllByText('No plans found for this interval.')).not.toHaveLength(0);
  });

  it('reorders editable plans through drag and drop', () => {
    const onReorderPlans = vi.fn();
    render(<BundleCard {...props({
      entry: { ...entry, prices: entry.prices.map((price) => ({ ...price, billing_interval: 'month' })) },
      pricingTab: 'month', onReorderPlans,
    })} />);
    const cards = Array.from(document.querySelectorAll('[draggable="true"]'));
    expect(cards).toHaveLength(4);
    const dataTransfer = {
      effectAllowed: '', dropEffect: '', setData: vi.fn(), getData: vi.fn(() => 'price_month'),
    };
    fireEvent.dragStart(cards[0]!, { dataTransfer });
    fireEvent.dragOver(cards[1]!, { dataTransfer });
    fireEvent.dragLeave(cards[1]!);
    fireEvent.drop(cards[1]!, { dataTransfer });
    fireEvent.dragEnd(cards[0]!);
    expect(onReorderPlans).toHaveBeenCalledWith('starter', ['price_year', 'price_month']);
  });
});
