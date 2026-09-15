import type { PlanOption, PricingOverview } from '../../../shared/api/types';
import type { Presentation } from './types';

export type OwnerPrice = Pick<PlanOption, 'stripe_price_id' | 'amount_cents' | 'currency' | 'billing_interval' | 'intro_enabled'>;
export type ResolvedPricing = Readonly<Record<string, { status: 'ready'; plan: OwnerPrice } | { status: 'unavailable'; reason: string }>>;

/** Owner catalog only. Configured refs determine order; duplicate or hidden prices fail closed. */
export function resolvePricing(presentation: Presentation, pricing?: PricingOverview): ResolvedPricing {
  const plans = [...(pricing?.monthly ?? []), ...(pricing?.yearly ?? []), ...(pricing?.credit_topups ?? [])];
  return Object.fromEntries(presentation.page.blocks.flatMap(block => block.kind === 'pricing' ? block.content.plan_refs : []).map(ref => {
    const matches = plans.filter(plan => plan.stripe_price_id === ref);
    const plan = matches.length === 1 ? matches[0] : undefined;
    const valid = plan && plan.display_enabled && !plan.is_variable_amount &&
      Number.isSafeInteger(plan.amount_cents) && plan.amount_cents >= 0 && /^[a-z]{3}$/i.test(plan.currency) &&
      ['month', 'year', 'one_time'].includes(plan.billing_interval) && plan.bundle_key === pricing?.bundle.bundle_key;
    return [ref, valid ? { status: 'ready', plan: { stripe_price_id: plan.stripe_price_id, amount_cents: plan.amount_cents, currency: plan.currency, billing_interval: plan.billing_interval, intro_enabled: plan.intro_enabled } } : { status: 'unavailable', reason: presentation.page.display.shell.unavailable_reason }];
  }));
}

export function formatOwnerPrice(plan: Pick<OwnerPrice, 'amount_cents' | 'currency'>, locale: string): string {
  return new Intl.NumberFormat(locale, { style: 'currency', currency: plan.currency.toUpperCase() }).format(plan.amount_cents / 100);
}
