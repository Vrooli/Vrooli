import type { PlanOption, PricingOverview } from '../../../shared/api/types';
import type { Presentation } from './types';

/** Admin-curated plan narrative from the owner catalog's display metadata. */
export interface OwnerPlanDisplay {
  name: string;
  subtitle: string;
  badge: string;
  highlight: boolean;
  cta_label: string;
  features: string[];
}

export type OwnerPrice = Pick<PlanOption, 'stripe_price_id' | 'amount_cents' | 'currency' | 'billing_interval' | 'intro_enabled' | 'plan_tier' | 'kind'> & { display: OwnerPlanDisplay };
export type ResolvedPricing = Readonly<Record<string, { status: 'ready'; plan: OwnerPrice } | { status: 'unavailable'; reason: string }>>;

const displayString = (value: unknown, limit: number): string => typeof value === 'string' ? value.trim().slice(0, limit) : '';

/** Only the curated display fields survive; unknown metadata keys never reach the page. */
function planDisplay(plan: PlanOption): OwnerPlanDisplay {
  const metadata = plan.metadata ?? {};
  const features = Array.isArray(metadata.features)
    ? metadata.features.filter((feature): feature is string => typeof feature === 'string' && feature.trim() !== '').map(feature => feature.trim().slice(0, 120)).slice(0, 8)
    : [];
  return {
    name: displayString(plan.plan_name, 60),
    subtitle: displayString(metadata.subtitle, 120),
    badge: displayString(metadata.badge, 40),
    highlight: metadata.highlight === true,
    cta_label: displayString(metadata.cta_label, 60),
    features,
  };
}

/** Owner catalog only. Configured refs determine order; duplicate or hidden prices fail closed. */
export function resolvePricing(presentation: Presentation, pricing?: PricingOverview): ResolvedPricing {
  const plans = [...(pricing?.monthly ?? []), ...(pricing?.yearly ?? []), ...(pricing?.credit_topups ?? [])];
  return Object.fromEntries(presentation.page.blocks.flatMap(block => block.kind === 'pricing' ? block.content.plan_refs : []).map(ref => {
    const matches = plans.filter(plan => plan.stripe_price_id === ref);
    const plan = matches.length === 1 ? matches[0] : undefined;
    const valid = plan && plan.display_enabled && !plan.is_variable_amount &&
      Number.isSafeInteger(plan.amount_cents) && plan.amount_cents >= 0 && /^[a-z]{3}$/i.test(plan.currency) &&
      ['month', 'year', 'one_time'].includes(plan.billing_interval) && plan.bundle_key === pricing?.bundle.bundle_key;
    return [ref, valid ? { status: 'ready', plan: { stripe_price_id: plan.stripe_price_id, amount_cents: plan.amount_cents, currency: plan.currency, billing_interval: plan.billing_interval, intro_enabled: plan.intro_enabled, plan_tier: plan.plan_tier, kind: plan.kind, display: planDisplay(plan) } } : { status: 'unavailable', reason: presentation.page.display.shell.unavailable_reason }];
  }));
}

export function formatOwnerPrice(plan: Pick<OwnerPrice, 'amount_cents' | 'currency'>, locale: string): string {
  return new Intl.NumberFormat(locale, { style: 'currency', currency: plan.currency.toUpperCase() }).format(plan.amount_cents / 100);
}

/** Real pair savings only: a yearly price against twelve of the same tier's monthly price. */
export function yearlySavingsPercent(year: Pick<OwnerPrice, 'amount_cents'>, month: Pick<OwnerPrice, 'amount_cents'>): number {
  if (month.amount_cents <= 0 || year.amount_cents <= 0) return 0;
  const percent = Math.round((1 - year.amount_cents / (12 * month.amount_cents)) * 100);
  return percent > 0 && percent < 100 ? percent : 0;
}
