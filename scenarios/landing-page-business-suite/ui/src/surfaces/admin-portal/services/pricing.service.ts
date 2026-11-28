import type { BundleCatalogEntry, BundleProduct, PlanOption, PlanDisplayMetadata, PricingOverview } from '../../../shared/api';
import { injectDemoPlansForBundle, isDemoPlanOption } from '../../../shared/lib/pricingPlaceholders';
import { isFormDirty } from '../../../shared/lib/formUtils';

/**
 * Interval type slugs for pricing categorization
 */
export type IntervalSlug = 'month' | 'year' | 'one_time' | 'other';

/**
 * Form values for a single price entry
 */
export interface PriceFormValues {
  stripePriceId: string;
  planName: string;
  displayWeight: number;
  displayEnabled: boolean;
  subtitle: string;
  badge: string;
  ctaLabel: string;
  highlight: boolean;
  featuresText: string;
}

/**
 * State for a single price form including original values for dirty detection
 */
export interface PriceFormState {
  values: PriceFormValues;
  original: PriceFormValues;
  saving: boolean;
  error?: string;
  demo?: boolean;
}

/**
 * Data structure for pricing preview
 */
export interface PricingPreviewData {
  overview: PricingOverview;
  monthlyCount: number;
  placeholderCount: number;
}

/**
 * Explicit interval string mappings to avoid fragile substring matching.
 * Keys are normalized (lowercase) string representations.
 */
const INTERVAL_MAPPINGS: Record<string, IntervalSlug> = {
  month: 'month',
  monthly: 'month',
  year: 'year',
  yearly: 'year',
  annual: 'year',
  one_time: 'one_time',
  'one-time': 'one_time',
  onetime: 'one_time',
};

/**
 * Normalize billing interval to a consistent slug
 */
export function normalizeInterval(value: string | number | null | undefined): IntervalSlug {
  if (typeof value === 'number') {
    if (value === 1) return 'month';
    if (value === 2) return 'year';
    if (value === 3) return 'one_time';
  }
  const raw = String(value ?? '').toLowerCase();
  return INTERVAL_MAPPINGS[raw] ?? 'other';
}

/**
 * Get human-readable label for interval slug
 */
export function getIntervalLabel(slug: IntervalSlug): string {
  switch (slug) {
    case 'month':
      return 'Monthly';
    case 'year':
      return 'Yearly';
    case 'one_time':
      return 'One-time';
    default:
      return 'Other';
  }
}

/**
 * Filter prices by billing interval tab
 */
export function filterPricesByTab(
  prices: PlanOption[],
  tab: 'month' | 'year' | 'other',
  includeDemoPlaceholders: boolean
): PlanOption[] {
  return prices.filter((price) => {
    const interval = normalizeInterval(price.billing_interval);
    if (!includeDemoPlaceholders && isDemoPlanOption(price)) {
      return false;
    }
    if (tab === 'month') return interval === 'month';
    if (tab === 'year') return interval === 'year';
    return interval === 'one_time' || interval === 'other';
  });
}

/**
 * Create a synthetic demo bundle for previewing layout when no bundles exist
 */
export function createDemoBundleEntry(): BundleCatalogEntry {
  const demoBundle: BundleProduct = {
    bundle_key: 'demo_bundle',
    name: 'Demo Bundle',
    stripe_product_id: 'demo_product',
    credits_per_usd: 1000000,
    display_credits_multiplier: 1,
    display_credits_label: 'credits',
    environment: 'demo',
  };

  return {
    bundle: demoBundle,
    prices: [],
  };
}

/**
 * Ensure at least one bundle exists for demo display.
 * When bundles array is empty and demo placeholders are enabled,
 * returns a synthetic demo bundle that will be enriched with demo plans.
 */
export function ensureBundleForDemo(bundles: BundleCatalogEntry[], includeDemo: boolean): BundleCatalogEntry[] {
  if (bundles.length > 0 || !includeDemo) {
    return bundles;
  }
  return [createDemoBundleEntry()];
}

/**
 * Enrich bundles with demo placeholder plans
 */
export function enrichBundlesWithDemo(bundles: BundleCatalogEntry[], includeDemo: boolean): BundleCatalogEntry[] {
  if (!includeDemo) {
    return bundles;
  }
  return bundles.map((entry) => injectDemoPlansForBundle(entry));
}

/**
 * Build form values from price metadata
 */
export function buildPriceFormValues(
  metadata: PlanDisplayMetadata | undefined,
  defaults: { planName: string; displayWeight: number; displayEnabled: boolean; priceId: string }
): PriceFormValues {
  const features = Array.isArray(metadata?.features)
    ? metadata.features
    : [];

  return {
    stripePriceId: defaults.priceId,
    planName: defaults.planName,
    displayWeight: defaults.displayWeight,
    displayEnabled: defaults.displayEnabled,
    subtitle: (metadata?.subtitle as string) || '',
    badge: (metadata?.badge as string) || '',
    ctaLabel: (metadata?.cta_label as string) || '',
    highlight: Boolean(metadata?.highlight),
    featuresText: features.join('\n'),
  };
}

/**
 * Build price forms map from bundles
 */
export function buildPriceFormsFromBundles(bundles: BundleCatalogEntry[]): Record<string, PriceFormState> {
  const forms: Record<string, PriceFormState> = {};

  bundles.forEach((entry) => {
    entry.prices.forEach((price) => {
      const priceIdentifier = getPriceIdentifier(price);
      const key = `${entry.bundle.bundle_key}:${priceIdentifier}`;
      const values = buildPriceFormValues(price.metadata, {
        priceId: price.stripe_price_id,
        planName: price.plan_name,
        displayWeight: price.display_weight,
        displayEnabled: price.display_enabled,
      });
      forms[key] = {
        values,
        original: { ...values },
        saving: false,
        demo: isDemoPlanOption(price),
      };
    });
  });

  return forms;
}

/**
 * Get unique identifier for a price option
 */
export function getPriceIdentifier(price: PlanOption): string {
  const metadata = price.metadata;
  const primaryKey = metadata?.__price_pk;
  const metadataIdentifier =
    typeof primaryKey === 'string'
      ? primaryKey
      : typeof primaryKey === 'number'
        ? String(primaryKey)
        : undefined;
  return price.stripe_price_id || metadataIdentifier || price.plan_name;
}

/**
 * Check if a price form has unsaved changes
 */
export function isPriceFormDirty(state: PriceFormState): boolean {
  return isFormDirty(state.values, state.original);
}

/**
 * Parse features text into array
 */
export function parseFeaturesText(raw: string): string[] {
  return raw
    .split('\n')
    .map((entry) => entry.trim())
    .filter(Boolean);
}

/**
 * Sort plans by display weight and rank
 */
export function sortPlans(plans: PlanOption[]): PlanOption[] {
  return [...plans].sort((a, b) => {
    if (a.display_weight === b.display_weight) {
      const aRank = typeof a.plan_rank === 'number' ? a.plan_rank : Number.MAX_SAFE_INTEGER;
      const bRank = typeof b.plan_rank === 'number' ? b.plan_rank : Number.MAX_SAFE_INTEGER;
      return aRank - bRank;
    }
    return b.display_weight - a.display_weight;
  });
}

/**
 * Apply form overrides to a price option for preview
 */
export function applyFormOverrides(
  bundleKey: string,
  price: PlanOption,
  priceForms: Record<string, PriceFormState>
): PlanOption {
  const priceIdentifier = getPriceIdentifier(price);
  const key = `${bundleKey}:${priceIdentifier}`;
  const formState = priceForms[key];
  if (!formState) {
    return { ...price };
  }

  const nextMetadata: PlanDisplayMetadata = {
    ...(price.metadata ?? {}),
  };

  const setOrDelete = (field: keyof PlanDisplayMetadata, value: string) => {
    const trimmed = value.trim();
    if (trimmed.length > 0) {
      nextMetadata[field] = trimmed;
    } else {
      Reflect.deleteProperty(nextMetadata, field);
    }
  };

  setOrDelete('subtitle', formState.values.subtitle);
  setOrDelete('badge', formState.values.badge);
  setOrDelete('cta_label', formState.values.ctaLabel);
  nextMetadata.highlight = formState.values.highlight || undefined;
  const features = parseFeaturesText(formState.values.featuresText);
  if (features.length > 0) {
    nextMetadata.features = features;
  } else {
    Reflect.deleteProperty(nextMetadata, 'features');
  }

  const metadata = Object.keys(nextMetadata).length > 0 ? nextMetadata : undefined;
  const planName = formState.values.planName.trim().length > 0 ? formState.values.planName.trim() : price.plan_name;

  return {
    ...price,
    plan_name: planName,
    display_weight: formState.values.displayWeight,
    display_enabled: formState.values.displayEnabled,
    metadata,
  };
}

/**
 * Build pricing preview data from bundle and forms
 */
export function buildPricingPreviewData(
  entry: BundleCatalogEntry,
  priceForms: Record<string, PriceFormState>,
  includeDemo: boolean
): PricingPreviewData {
  const enhancedPlans = entry.prices
    .filter((plan) => includeDemo || !isDemoPlanOption(plan))
    .map((price) => applyFormOverrides(entry.bundle.bundle_key, price, priceForms));

  const monthlyPlans = sortPlans(
    enhancedPlans.filter((plan) => normalizeInterval(plan.billing_interval) === 'month' && plan.display_enabled)
  );
  const yearlyPlans = sortPlans(
    enhancedPlans.filter((plan) => normalizeInterval(plan.billing_interval) === 'year' && plan.display_enabled)
  );

  const placeholderCount = monthlyPlans.filter((plan) => isDemoPlanOption(plan)).length;
  const monthlyCount = monthlyPlans.length - placeholderCount;

  return {
    overview: {
      bundle: entry.bundle,
      monthly: monthlyPlans,
      yearly: yearlyPlans,
      updated_at: new Date().toISOString(),
    },
    monthlyCount,
    placeholderCount,
  };
}
