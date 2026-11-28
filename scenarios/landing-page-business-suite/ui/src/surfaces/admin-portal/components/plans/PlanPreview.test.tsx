import { cleanup, screen } from '@testing-library/react';
import { renderWithProviders as render } from '@vrooli/api-base/testing';
import { afterEach, describe, expect, it } from 'vitest';
import { PlanPreview } from './PlanPreview';
import { ownerPricing, ownerPlan } from '../../../public-landing/presentation/commerceFixtures';
import { buildPricingPreviewData, buildPriceFormsFromBundles } from '../../services/pricing.service';
afterEach(cleanup);
describe('private configured pricing preview', () => {
  it('shows owner amounts and billing without transaction controls or product defaults', () => {
    render(<PlanPreview data={{ overview: ownerPricing, monthlyCount: 1, placeholderCount: 0 }} />);
    expect(screen.getByText('$12.34')).toBeInTheDocument();
    expect(screen.getByText('$123.45')).toBeInTheDocument();
    expect(screen.getByText('Monthly')).toBeInTheDocument();
    expect(screen.getByText('Yearly')).toBeInTheDocument();
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
    expect(screen.queryByRole('link')).not.toBeInTheDocument();
    expect(document.body).not.toHaveTextContent(/Aquila|Browser Automation|Start for \$1/);
  });
  it('previews unsaved owner form values without saving or inventing a three-plan layout', () => {
    const entry = { bundle: ownerPricing.bundle, prices: [ownerPlan] };
    const forms = buildPriceFormsFromBundles([entry]);
    const state = forms[entry.bundle.bundle_key + ':' + ownerPlan.stripe_price_id];
    if (!state) throw new Error('fixture');
    state.values.planName = 'Unsaved owner name'; state.values.subtitle = 'Unsaved subtitle'; state.values.featuresText = 'Configured feature'; state.values.ctaLabel = 'Configured CTA';
    render(<PlanPreview data={buildPricingPreviewData(entry, forms, false)} />);
    expect(screen.getByRole('heading', { name: 'Unsaved owner name' })).toBeInTheDocument();
    expect(screen.getByText('Unsaved subtitle')).toBeInTheDocument();
    expect(screen.getByText('Configured feature')).toBeInTheDocument();
    expect(screen.getByText('Configured label: Configured CTA')).toBeInTheDocument();
    expect(screen.getAllByRole('article')).toHaveLength(1);
    expect(ownerPlan.plan_name).not.toBe('Unsaved owner name');
  });
  it('shows an honest empty state when only hidden or demo plans exist', () => {
    render(<PlanPreview data={{ overview: { ...ownerPricing, monthly: [{ ...ownerPlan, display_enabled: false }, { ...ownerPlan, metadata: { __demo_placeholder: true } }], yearly: [] }, monthlyCount: 0, placeholderCount: 1 }} />);
    expect(screen.getByRole('status')).toHaveTextContent('No enabled configured plans to preview');
    expect(screen.queryByRole('article')).not.toBeInTheDocument();
  });
});
