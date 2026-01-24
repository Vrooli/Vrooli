import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { BundleCatalogEntry, BundleProduct, PlanOption, PlanDisplayMetadata } from '../../../shared/api';
import {
  normalizeInterval,
  getIntervalLabel,
  filterPricesByTab,
  enrichBundlesWithDemo,
  buildPriceFormValues,
  buildPriceFormsFromBundles,
  getPriceIdentifier,
  isPriceFormDirty,
  parseFeaturesText,
  sortPlans,
  applyFormOverrides,
  buildPricingPreviewData,
  type IntervalSlug,
  type PriceFormState,
} from './pricing.service';

const mockBundle: BundleProduct = {
  id: 1,
  bundle_key: 'test_bundle',
  name: 'Test Bundle',
  stripe_product_id: 'prod_test',
  credits_per_usd: 1000000,
  display_credits_multiplier: 0.001,
  display_credits_label: 'credits',
};

const createMockPlan = (overrides: Partial<PlanOption> = {}): PlanOption => ({
  plan_name: 'Test Plan',
  plan_tier: 'pro',
  billing_interval: 'month',
  amount_cents: 9900,
  currency: 'usd',
  intro_enabled: false,
  stripe_price_id: 'price_123',
  monthly_included_credits: 10000000,
  one_time_bonus_credits: 0,
  plan_rank: 1,
  display_enabled: true,
  display_weight: 50,
  ...overrides,
});

describe('pricing.service', () => {
  describe('normalizeInterval', () => {
    it('normalizes numeric interval 1 to month', () => {
      expect(normalizeInterval(1)).toBe('month');
    });

    it('normalizes numeric interval 2 to year', () => {
      expect(normalizeInterval(2)).toBe('year');
    });

    it('normalizes numeric interval 3 to one_time', () => {
      expect(normalizeInterval(3)).toBe('one_time');
    });

    it('normalizes string "month" to month', () => {
      expect(normalizeInterval('month')).toBe('month');
    });

    it('normalizes string "monthly" to month', () => {
      expect(normalizeInterval('monthly')).toBe('month');
    });

    it('normalizes string "YEAR" to year (case insensitive)', () => {
      expect(normalizeInterval('YEAR')).toBe('year');
    });

    it('normalizes string "yearly" to year', () => {
      expect(normalizeInterval('yearly')).toBe('year');
    });

    it('normalizes string "one_time" to one_time', () => {
      expect(normalizeInterval('one_time')).toBe('one_time');
    });

    it('normalizes string "one-time" to one_time', () => {
      expect(normalizeInterval('one-time')).toBe('one_time');
    });

    it('normalizes string "onetime" to one_time', () => {
      expect(normalizeInterval('onetime')).toBe('one_time');
    });

    it('returns other for unknown values', () => {
      expect(normalizeInterval('weekly')).toBe('other');
      expect(normalizeInterval(99)).toBe('other');
    });

    it('handles null and undefined', () => {
      expect(normalizeInterval(null)).toBe('other');
      expect(normalizeInterval(undefined)).toBe('other');
    });

    it('normalizes "annual" to year', () => {
      expect(normalizeInterval('annual')).toBe('year');
    });

    it('normalizes mixed case variants', () => {
      expect(normalizeInterval('Month')).toBe('month');
      expect(normalizeInterval('MONTHLY')).toBe('month');
      expect(normalizeInterval('Yearly')).toBe('year');
      expect(normalizeInterval('ANNUAL')).toBe('year');
      expect(normalizeInterval('One_Time')).toBe('one_time');
    });

    it('does NOT match partial strings (regression test)', () => {
      expect(normalizeInterval('bimonthly')).toBe('other');
      expect(normalizeInterval('5months')).toBe('other');
      expect(normalizeInterval('year_end')).toBe('other');
      expect(normalizeInterval('monthly_premium')).toBe('other');
    });

    it('returns other for empty string', () => {
      expect(normalizeInterval('')).toBe('other');
    });

    it('returns other for whitespace-only string', () => {
      expect(normalizeInterval('   ')).toBe('other');
    });
  });

  describe('getIntervalLabel', () => {
    it('returns Monthly for month slug', () => {
      expect(getIntervalLabel('month')).toBe('Monthly');
    });

    it('returns Yearly for year slug', () => {
      expect(getIntervalLabel('year')).toBe('Yearly');
    });

    it('returns One-time for one_time slug', () => {
      expect(getIntervalLabel('one_time')).toBe('One-time');
    });

    it('returns Other for other slug', () => {
      expect(getIntervalLabel('other')).toBe('Other');
    });
  });

  describe('filterPricesByTab', () => {
    const monthlyPlan = createMockPlan({ billing_interval: 'month', stripe_price_id: 'price_monthly' });
    const yearlyPlan = createMockPlan({ billing_interval: 'year', stripe_price_id: 'price_yearly' });
    const oneTimePlan = createMockPlan({ billing_interval: 'one_time', stripe_price_id: 'price_onetime' });
    const demoPlan = createMockPlan({
      billing_interval: 'month',
      stripe_price_id: 'demo_test',
      metadata: { __demo_placeholder: true } as PlanDisplayMetadata,
    });

    it('filters to monthly plans when tab is month', () => {
      const prices = [monthlyPlan, yearlyPlan, oneTimePlan];
      const result = filterPricesByTab(prices, 'month', true);
      expect(result).toHaveLength(1);
      expect(result[0]?.stripe_price_id).toBe('price_monthly');
    });

    it('filters to yearly plans when tab is year', () => {
      const prices = [monthlyPlan, yearlyPlan, oneTimePlan];
      const result = filterPricesByTab(prices, 'year', true);
      expect(result).toHaveLength(1);
      expect(result[0]?.stripe_price_id).toBe('price_yearly');
    });

    it('filters to one_time and other plans when tab is other', () => {
      const prices = [monthlyPlan, yearlyPlan, oneTimePlan];
      const result = filterPricesByTab(prices, 'other', true);
      expect(result).toHaveLength(1);
      expect(result[0]?.stripe_price_id).toBe('price_onetime');
    });

    it('excludes demo plans when includeDemoPlaceholders is false', () => {
      const prices = [monthlyPlan, demoPlan];
      const result = filterPricesByTab(prices, 'month', false);
      expect(result).toHaveLength(1);
      expect(result[0]?.stripe_price_id).toBe('price_monthly');
    });

    it('includes demo plans when includeDemoPlaceholders is true', () => {
      const prices = [monthlyPlan, demoPlan];
      const result = filterPricesByTab(prices, 'month', true);
      expect(result).toHaveLength(2);
    });
  });

  describe('enrichBundlesWithDemo', () => {
    it('returns bundles unchanged when includeDemo is false', () => {
      const entry: BundleCatalogEntry = {
        bundle: mockBundle,
        prices: [],
      };
      const result = enrichBundlesWithDemo([entry], false);
      expect(result).toEqual([entry]);
    });

    it('enriches bundles with demo plans when includeDemo is true', () => {
      const entry: BundleCatalogEntry = {
        bundle: mockBundle,
        prices: [],
      };
      const result = enrichBundlesWithDemo([entry], true);
      expect(result[0]?.prices.length).toBeGreaterThan(0);
    });
  });

  describe('buildPriceFormValues', () => {
    it('extracts features array from metadata', () => {
      const metadata: PlanDisplayMetadata = {
        features: ['Feature 1', 'Feature 2', 'Feature 3'],
        subtitle: 'Test subtitle',
        badge: 'Popular',
        cta_label: 'Get Started',
        highlight: true,
      };
      const defaults = {
        planName: 'Pro Plan',
        displayWeight: 50,
        displayEnabled: true,
        priceId: 'price_123',
      };

      const result = buildPriceFormValues(metadata, defaults);

      expect(result.stripePriceId).toBe('price_123');
      expect(result.planName).toBe('Pro Plan');
      expect(result.displayWeight).toBe(50);
      expect(result.displayEnabled).toBe(true);
      expect(result.subtitle).toBe('Test subtitle');
      expect(result.badge).toBe('Popular');
      expect(result.ctaLabel).toBe('Get Started');
      expect(result.highlight).toBe(true);
      expect(result.featuresText).toBe('Feature 1\nFeature 2\nFeature 3');
    });

    it('handles undefined metadata', () => {
      const defaults = {
        planName: 'Basic',
        displayWeight: 10,
        displayEnabled: false,
        priceId: 'price_456',
      };

      const result = buildPriceFormValues(undefined, defaults);

      expect(result.planName).toBe('Basic');
      expect(result.subtitle).toBe('');
      expect(result.badge).toBe('');
      expect(result.featuresText).toBe('');
    });

    it('handles empty features array', () => {
      const metadata: PlanDisplayMetadata = {
        features: [],
      };
      const defaults = {
        planName: 'Plan',
        displayWeight: 0,
        displayEnabled: true,
        priceId: 'price_789',
      };

      const result = buildPriceFormValues(metadata, defaults);
      expect(result.featuresText).toBe('');
    });
  });

  describe('buildPriceFormsFromBundles', () => {
    it('builds forms map from bundles', () => {
      const entry: BundleCatalogEntry = {
        bundle: mockBundle,
        prices: [
          createMockPlan({ stripe_price_id: 'price_a', plan_name: 'Plan A' }),
          createMockPlan({ stripe_price_id: 'price_b', plan_name: 'Plan B' }),
        ],
      };

      const result = buildPriceFormsFromBundles([entry]);

      expect(Object.keys(result)).toHaveLength(2);
      expect(result['test_bundle:price_a']).toBeDefined();
      expect(result['test_bundle:price_b']).toBeDefined();
      expect(result['test_bundle:price_a']?.values.planName).toBe('Plan A');
    });

    it('marks demo plans in form state', () => {
      const demoPlan = createMockPlan({
        stripe_price_id: 'demo_plan',
        metadata: { __demo_placeholder: true } as PlanDisplayMetadata,
      });
      const entry: BundleCatalogEntry = {
        bundle: mockBundle,
        prices: [demoPlan],
      };

      const result = buildPriceFormsFromBundles([entry]);
      expect(result['test_bundle:demo_plan']?.demo).toBe(true);
    });
  });

  describe('getPriceIdentifier', () => {
    it('returns stripe_price_id when available', () => {
      const price = createMockPlan({ stripe_price_id: 'price_abc123' });
      expect(getPriceIdentifier(price)).toBe('price_abc123');
    });

    it('falls back to __price_pk from metadata', () => {
      const price = createMockPlan({
        stripe_price_id: '',
        metadata: { __price_pk: 42 } as PlanDisplayMetadata,
      });
      expect(getPriceIdentifier(price)).toBe('42');
    });

    it('falls back to plan_name as last resort', () => {
      const price = createMockPlan({
        stripe_price_id: '',
        plan_name: 'Fallback Plan',
      });
      expect(getPriceIdentifier(price)).toBe('Fallback Plan');
    });
  });

  describe('isPriceFormDirty', () => {
    it('returns false when values match original', () => {
      const values = {
        stripePriceId: 'price_123',
        planName: 'Test',
        displayWeight: 50,
        displayEnabled: true,
        subtitle: '',
        badge: '',
        ctaLabel: '',
        highlight: false,
        featuresText: '',
      };
      const state: PriceFormState = {
        values: { ...values },
        original: { ...values },
        saving: false,
      };

      expect(isPriceFormDirty(state)).toBe(false);
    });

    it('returns true when values differ from original', () => {
      const original = {
        stripePriceId: 'price_123',
        planName: 'Test',
        displayWeight: 50,
        displayEnabled: true,
        subtitle: '',
        badge: '',
        ctaLabel: '',
        highlight: false,
        featuresText: '',
      };
      const state: PriceFormState = {
        values: { ...original, planName: 'Modified' },
        original,
        saving: false,
      };

      expect(isPriceFormDirty(state)).toBe(true);
    });
  });

  describe('parseFeaturesText', () => {
    it('parses newline-separated features', () => {
      const result = parseFeaturesText('Feature 1\nFeature 2\nFeature 3');
      expect(result).toEqual(['Feature 1', 'Feature 2', 'Feature 3']);
    });

    it('trims whitespace from features', () => {
      const result = parseFeaturesText('  Feature 1  \n  Feature 2  ');
      expect(result).toEqual(['Feature 1', 'Feature 2']);
    });

    it('filters out empty lines', () => {
      const result = parseFeaturesText('Feature 1\n\n\nFeature 2\n');
      expect(result).toEqual(['Feature 1', 'Feature 2']);
    });

    it('returns empty array for empty string', () => {
      const result = parseFeaturesText('');
      expect(result).toEqual([]);
    });
  });

  describe('sortPlans', () => {
    it('sorts by display_weight descending', () => {
      const plans = [
        createMockPlan({ display_weight: 10, plan_name: 'Low' }),
        createMockPlan({ display_weight: 50, plan_name: 'Mid' }),
        createMockPlan({ display_weight: 90, plan_name: 'High' }),
      ];

      const result = sortPlans(plans);
      expect(result[0]?.plan_name).toBe('High');
      expect(result[1]?.plan_name).toBe('Mid');
      expect(result[2]?.plan_name).toBe('Low');
    });

    it('sorts by plan_rank when display_weight is equal', () => {
      const plans = [
        createMockPlan({ display_weight: 50, plan_rank: 3, plan_name: 'Third' }),
        createMockPlan({ display_weight: 50, plan_rank: 1, plan_name: 'First' }),
        createMockPlan({ display_weight: 50, plan_rank: 2, plan_name: 'Second' }),
      ];

      const result = sortPlans(plans);
      expect(result[0]?.plan_name).toBe('First');
      expect(result[1]?.plan_name).toBe('Second');
      expect(result[2]?.plan_name).toBe('Third');
    });

    it('handles undefined plan_rank by treating as MAX_SAFE_INTEGER', () => {
      const plans = [
        createMockPlan({ display_weight: 50, plan_rank: undefined, plan_name: 'NoRank' }),
        createMockPlan({ display_weight: 50, plan_rank: 1, plan_name: 'Ranked' }),
      ];

      const result = sortPlans(plans);
      expect(result[0]?.plan_name).toBe('Ranked');
      expect(result[1]?.plan_name).toBe('NoRank');
    });

    it('does not mutate the original array', () => {
      const plans = [
        createMockPlan({ display_weight: 10, plan_name: 'A' }),
        createMockPlan({ display_weight: 20, plan_name: 'B' }),
      ];
      const originalFirst = plans[0];

      sortPlans(plans);
      expect(plans[0]).toBe(originalFirst);
    });
  });

  describe('applyFormOverrides', () => {
    it('returns price unchanged when no form state exists', () => {
      const price = createMockPlan();
      const result = applyFormOverrides('test_bundle', price, {});

      expect(result.plan_name).toBe(price.plan_name);
    });

    it('applies form values to price metadata', () => {
      const price = createMockPlan({ stripe_price_id: 'price_123' });
      const formState: PriceFormState = {
        values: {
          stripePriceId: 'price_123',
          planName: 'Updated Plan',
          displayWeight: 75,
          displayEnabled: true,
          subtitle: 'New subtitle',
          badge: 'Best Value',
          ctaLabel: 'Buy Now',
          highlight: true,
          featuresText: 'Feature A\nFeature B',
        },
        original: {
          stripePriceId: 'price_123',
          planName: 'Test Plan',
          displayWeight: 50,
          displayEnabled: true,
          subtitle: '',
          badge: '',
          ctaLabel: '',
          highlight: false,
          featuresText: '',
        },
        saving: false,
      };

      const result = applyFormOverrides('test_bundle', price, { 'test_bundle:price_123': formState });

      expect(result.plan_name).toBe('Updated Plan');
      expect(result.display_weight).toBe(75);
      expect(result.metadata?.subtitle).toBe('New subtitle');
      expect(result.metadata?.badge).toBe('Best Value');
      expect(result.metadata?.cta_label).toBe('Buy Now');
      expect(result.metadata?.highlight).toBe(true);
      expect(result.metadata?.features).toEqual(['Feature A', 'Feature B']);
    });

    it('removes empty metadata fields', () => {
      const price = createMockPlan({ stripe_price_id: 'price_123' });
      const formState: PriceFormState = {
        values: {
          stripePriceId: 'price_123',
          planName: 'Plan',
          displayWeight: 50,
          displayEnabled: true,
          subtitle: '',
          badge: '',
          ctaLabel: '',
          highlight: false,
          featuresText: '',
        },
        original: {
          stripePriceId: 'price_123',
          planName: 'Plan',
          displayWeight: 50,
          displayEnabled: true,
          subtitle: '',
          badge: '',
          ctaLabel: '',
          highlight: false,
          featuresText: '',
        },
        saving: false,
      };

      const result = applyFormOverrides('test_bundle', price, { 'test_bundle:price_123': formState });

      expect(result.metadata?.subtitle).toBeUndefined();
      expect(result.metadata?.badge).toBeUndefined();
    });
  });

  describe('buildPricingPreviewData', () => {
    it('builds preview data with monthly and yearly plans', () => {
      const entry: BundleCatalogEntry = {
        bundle: mockBundle,
        prices: [
          createMockPlan({ billing_interval: 'month', stripe_price_id: 'price_m', display_enabled: true }),
          createMockPlan({ billing_interval: 'year', stripe_price_id: 'price_y', display_enabled: true }),
        ],
      };

      const result = buildPricingPreviewData(entry, {}, false);

      expect(result.overview.bundle).toBe(mockBundle);
      expect(result.overview.monthly).toHaveLength(1);
      expect(result.overview.yearly).toHaveLength(1);
      expect(result.monthlyCount).toBe(1);
      expect(result.placeholderCount).toBe(0);
    });

    it('excludes disabled plans from preview', () => {
      const entry: BundleCatalogEntry = {
        bundle: mockBundle,
        prices: [
          createMockPlan({ billing_interval: 'month', display_enabled: true, stripe_price_id: 'enabled' }),
          createMockPlan({ billing_interval: 'month', display_enabled: false, stripe_price_id: 'disabled' }),
        ],
      };

      const result = buildPricingPreviewData(entry, {}, false);

      expect(result.overview.monthly).toHaveLength(1);
      expect(result.overview.monthly[0]?.stripe_price_id).toBe('enabled');
    });

    it('counts placeholder plans separately', () => {
      const demoPlan = createMockPlan({
        billing_interval: 'month',
        stripe_price_id: 'demo_plan',
        metadata: { __demo_placeholder: true } as PlanDisplayMetadata,
        display_enabled: true,
      });
      const realPlan = createMockPlan({
        billing_interval: 'month',
        stripe_price_id: 'real_plan',
        display_enabled: true,
      });
      const entry: BundleCatalogEntry = {
        bundle: mockBundle,
        prices: [demoPlan, realPlan],
      };

      const result = buildPricingPreviewData(entry, {}, true);

      expect(result.monthlyCount).toBe(1);
      expect(result.placeholderCount).toBe(1);
    });

    it('excludes demo plans when includeDemo is false', () => {
      const demoPlan = createMockPlan({
        billing_interval: 'month',
        stripe_price_id: 'demo_plan',
        metadata: { __demo_placeholder: true } as PlanDisplayMetadata,
        display_enabled: true,
      });
      const entry: BundleCatalogEntry = {
        bundle: mockBundle,
        prices: [demoPlan],
      };

      const result = buildPricingPreviewData(entry, {}, false);

      expect(result.overview.monthly).toHaveLength(0);
    });

    it('sorts plans by display weight in overview', () => {
      const entry: BundleCatalogEntry = {
        bundle: mockBundle,
        prices: [
          createMockPlan({ billing_interval: 'month', display_weight: 10, plan_name: 'Low', display_enabled: true }),
          createMockPlan({ billing_interval: 'month', display_weight: 90, plan_name: 'High', display_enabled: true }),
        ],
      };

      const result = buildPricingPreviewData(entry, {}, false);

      expect(result.overview.monthly[0]?.plan_name).toBe('High');
      expect(result.overview.monthly[1]?.plan_name).toBe('Low');
    });
  });
});
