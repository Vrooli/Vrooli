// provider-free-exception: tests pure owner-fact projection and prop-fed pricing rendering, with no commerce/API/provider effects.
import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import { decodeProductPresentation } from './decode';
import { publicConfig } from './publicTestFixtures';
import { ownerPlan, ownerPricing } from './commerceFixtures';
import { resolvePricing } from './commerce';
import { PresentationPage } from './PresentationPage';
import { actionKey, type Block } from './types';
import { resolvePublicActions } from './publicIntegration';
afterEach(cleanup);
const block: Block<'pricing'> = { id: 'plans', kind: 'pricing', variant: 'compact', version: 1, content: { heading: 'Configured plans', description: 'Configured terms', plan_refs: ['price-year', 'price-month'], actions: [{ kind: 'purchase', label: 'Choose monthly', accessible_label: 'Choose monthly', plan_ref: 'price-month' }] } };
function fixture() { const wire = publicConfig().presentation; if (!wire) throw new Error('fixture'); const p = decodeProductPresentation(wire); p.page.blocks = [structuredClone(block)]; return p; }
describe('owner-fed pricing', () => {
  it('takes narrative only from configured presentation actions, never owner marketing metadata or plan names', () => {
    const p = fixture(); const pricing = structuredClone(ownerPricing);
    pricing.monthly = [{ ...ownerPlan, plan_name: 'PRIVATE-OWNER-NAME', metadata: { subtitle: 'PRIVATE-SUBTITLE', features: ['PRIVATE-FEATURE'] } }];
    const resolved = resolvePricing(p, pricing);
    expect(resolved['price-month']).toEqual({ status: 'ready', plan: { stripe_price_id: 'price-month', amount_cents: 1234, currency: 'usd', billing_interval: 'month', intro_enabled: false } });
    render(<PresentationPage presentation={p} resolvedPricing={resolved} />);
    expect(screen.getByRole('heading', { name: 'Choose monthly' })).toBeInTheDocument();
    expect(document.body).not.toHaveTextContent(/PRIVATE-OWNER-NAME|PRIVATE-SUBTITLE|PRIVATE-FEATURE|Configured yearly plan/);
  });
  it('routes repeated configured download CTAs through the unique owner observation', () => {
    const wire = publicConfig().presentation; if (!wire) throw new Error('fixture');
    const p = decodeProductPresentation(wire);
    const action = { kind: 'download' as const, app_key: 'example-app', label: 'Configured download', accessible_label: 'Configured download' };
    p.page.display.shell.header_action = action;
    p.page.blocks = [{ id: 'closing', kind: 'closing-action', variant: 'plain', version: 1, content: { heading: 'Configured closing', description: '', actions: [action, action] } }];
    wire.actions = [{ $typeName: 'vrooli.landing_page_business_suite.v1.shared.ResolvedPresentationAction', key: actionKey(action), status: 'ready', href: '/apps/example/download', reason: '', appKey: 'example-app', planRef: '' }];
    render(<PresentationPage presentation={p} resolvedActions={resolvePublicActions(wire, p, '/proxy')} />);
    const links = screen.getAllByRole('link', { name: 'Configured download' }); expect(links).toHaveLength(3);
    for (const link of links) expect(link).toHaveAttribute('href', '/proxy/apps/example/download');
  });
  it('shows exact source amounts and intervals in configured order with owner-only actions', () => {
    const p = fixture(); const { container } = render(<PresentationPage presentation={p} resolvedPricing={resolvePricing(p, ownerPricing)} resolvedActions={{ [actionKey(block.content.actions[0]!)]: { status: 'ready', href: '/owner-checkout' } }} />);
    expect([...container.querySelectorAll('[data-price-ref]')].map(node => node.getAttribute('data-price-ref'))).toEqual(['price-year', 'price-month']);
    expect(screen.getByText('$123.45')).toBeInTheDocument(); expect(screen.getByText('$12.34')).toBeInTheDocument();
    expect(screen.getByText('Billed yearly')).toBeInTheDocument(); expect(screen.getByText('Billed monthly')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Choose monthly' })).toHaveAttribute('href', '/owner-checkout');
  });
  it.each(['missing', 'hidden', 'duplicate', 'invalid', 'wrong-bundle', 'variable'])('keeps %s owner prices unavailable without inventing plans', kind => {
    const p = fixture(); const pricing = structuredClone(ownerPricing);
    pricing.monthly = kind === 'missing' ? [] : kind === 'duplicate' ? [ownerPlan, ownerPlan] : [{ ...ownerPlan, display_enabled: kind !== 'hidden', amount_cents: kind === 'invalid' ? -1 : ownerPlan.amount_cents, bundle_key: kind === 'wrong-bundle' ? 'private' : 'example', is_variable_amount: kind === 'variable' }];
    const resolved = resolvePricing(p, pricing); expect(resolved['price-month']?.status).toBe('unavailable');
    render(<PresentationPage presentation={p} resolvedPricing={resolved} />);
    expect(screen.queryByText('$12.34')).not.toBeInTheDocument(); expect(screen.queryByRole('link', { name: 'Choose monthly' })).not.toBeInTheDocument();
  });
  it('renders unavailable preview without providers, fetches or product fallbacks', () => {
    const p = fixture(); render(<PresentationPage presentation={p} />);
    expect(screen.getAllByText('Owner unavailable')).toHaveLength(2);
    expect(screen.queryByText(/\$/)).not.toBeInTheDocument(); expect(document.body).not.toHaveTextContent(/Aquila|Browser Automation/);
  });
  it('formats source currency and zero without a synthetic free tier or monthly annual conversion', () => {
    const p = fixture(); p.page.locale = 'de'; const pricing = structuredClone(ownerPricing);
    pricing.monthly = [{ ...ownerPlan, amount_cents: 0, currency: 'eur' }];
    render(<PresentationPage presentation={p} resolvedPricing={resolvePricing(p, pricing)} />);
    expect(screen.getByText('0,00 €')).toBeInTheDocument(); expect(screen.getByText('123,45 $')).toBeInTheDocument();
  });
});
