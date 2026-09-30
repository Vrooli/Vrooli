import { describe, expect, it } from 'vitest';
import type { BundleCatalogEntry, BundleProduct, PlanOption } from '../api';
import {
  ensureDemoPlansForDisplay,
  injectDemoPlansForBundle,
  isDemoPlanOption,
} from './pricingPlaceholders';

const bundle: BundleProduct = {
  bundle_key: 'business-suite',
  name: 'Business Suite',
  stripe_product_id: 'prod_test',
  credits_per_usd: 1,
  display_credits_multiplier: 1,
  display_credits_label: 'credits',
};

const realPlan: PlanOption = {
  plan_name: 'Solo',
  plan_tier: 'solo',
  billing_interval: 'month',
  amount_cents: 3900,
  currency: 'usd',
  intro_enabled: false,
  stripe_price_id: 'price_solo',
  monthly_included_credits: 100,
  one_time_bonus_credits: 0,
  display_enabled: true,
  display_weight: 1,
};

const catalog = (prices: PlanOption[]): BundleCatalogEntry => ({ bundle, prices });

describe('pricing placeholder catalog helpers', () => {
  it('recognizes only explicitly marked demo plans', () => {
    expect(isDemoPlanOption({})).toBe(false);
    expect(isDemoPlanOption({ metadata: { __demo_placeholder: false } })).toBe(false);
    expect(isDemoPlanOption({ metadata: { __demo_placeholder: true } })).toBe(true);
  });

  it('keeps real monthly Stripe plans untouched', () => {
    const entry = catalog([realPlan]);
    expect(injectDemoPlansForBundle(entry)).toBe(entry);
    expect(ensureDemoPlansForDisplay(bundle, [realPlan], 1)).toEqual([realPlan]);
  });

  it('fills a catalog with demo plans up to the requested monthly count', () => {
    const result = injectDemoPlansForBundle(catalog([]), 2);

    expect(result.prices).toHaveLength(2);
    expect(result.prices.every(isDemoPlanOption)).toBe(true);
    expect(result.prices.map((plan) => plan.stripe_price_id)).toEqual([
      'demo_business-suite_launch',
      'demo_business-suite_pro',
    ]);
  });

  it('does not duplicate demo plans already present in the catalog', () => {
    const existing = ensureDemoPlansForDisplay(bundle, [], 1);
    const result = injectDemoPlansForBundle(catalog(existing), 3);

    expect(result.prices).toHaveLength(3);
    expect(new Set(result.prices.map((plan) => plan.stripe_price_id)).size).toBe(3);
  });

  it('adds only the missing plans when a display list has too few real plans', () => {
    const result = ensureDemoPlansForDisplay(bundle, [realPlan], 3);

    expect(result).toHaveLength(3);
    expect(result[0]).toBe(realPlan);
    expect(result.slice(1).every(isDemoPlanOption)).toBe(true);
  });
});
