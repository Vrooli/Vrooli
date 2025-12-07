import type { BundleCatalogEntry, BundleProduct, PlanOption, PlanDisplayMetadata } from '../api';

const DEMO_PLAN_FLAG = '__demo_placeholder';

interface TestPlanBlueprint {
  id: string;
  planName: string;
  planTier: string;
  amountCents: number;
  monthlyCredits: number;
  bonusCredits: number;
  displayWeight: number;
  subtitle: string;
  badge?: string;
  ctaLabel: string;
  highlight?: boolean;
  features: string[];
  introEnabled?: boolean;
  introAmountCents?: number;
  planRank: number;
}

const TEST_PLAN_BLUEPRINTS: TestPlanBlueprint[] = [
  {
    id: 'launch',
    planName: 'Launch',
    planTier: 'solo',
    amountCents: 3900,
    monthlyCredits: 5_000_000,
    bonusCredits: 0,
    displayWeight: 20,
    subtitle: 'Individuals validating a bundle idea',
    badge: 'Trial friendly',
    ctaLabel: 'Start for $1',
    highlight: false,
    features: ['5M included credits', 'Email support', 'Admin portal access'],
    introEnabled: true,
    introAmountCents: 100,
    planRank: 1,
  },
  {
    id: 'pro',
    planName: 'Pro',
    planTier: 'pro',
    amountCents: 12_900,
    monthlyCredits: 25_000_000,
    bonusCredits: 2_000_000,
    displayWeight: 40,
    subtitle: 'Teams upgrading the default experience',
    badge: 'Most popular',
    ctaLabel: 'Upgrade workspace',
    highlight: true,
    features: ['Priority agent queue', 'Desktop bundle downloads', '25M included credits'],
    introEnabled: true,
    introAmountCents: 100,
    planRank: 2,
  },
  {
    id: 'studio',
    planName: 'Studio',
    planTier: 'studio',
    amountCents: 32_900,
    monthlyCredits: 75_000_000,
    bonusCredits: 5_000_000,
    displayWeight: 30,
    subtitle: 'Enterprise-ready governance and rollouts',
    badge: 'White-glove',
    ctaLabel: 'Talk to sales',
    highlight: false,
    features: ['Unlimited automations', 'Dedicated architect', 'Compliance-ready exports'],
    introEnabled: false,
    introAmountCents: undefined,
    planRank: 3,
  },
];

function blueprintToPlan(bundle: BundleProduct, blueprint: TestPlanBlueprint): PlanOption {
  const metadata: PlanDisplayMetadata = {
    subtitle: blueprint.subtitle,
    badge: blueprint.badge,
    cta_label: blueprint.ctaLabel,
    highlight: blueprint.highlight ?? false,
    features: blueprint.features,
    [DEMO_PLAN_FLAG]: true,
  };

  return {
    plan_name: `${blueprint.planName} (Demo)`,
    plan_tier: blueprint.planTier,
    billing_interval: 'month',
    amount_cents: blueprint.amountCents,
    currency: 'usd',
    intro_enabled: Boolean(blueprint.introEnabled),
    intro_amount_cents: blueprint.introAmountCents,
    intro_periods: blueprint.introEnabled ? 1 : undefined,
    intro_price_lookup_key: undefined,
    stripe_price_id: `demo_${bundle.bundle_key}_${blueprint.id}`,
    monthly_included_credits: blueprint.monthlyCredits,
    one_time_bonus_credits: blueprint.bonusCredits,
    plan_rank: blueprint.planRank,
    bonus_type: 'none',
    kind: 'subscription',
    is_variable_amount: false,
    display_enabled: true,
    bundle_key: bundle.bundle_key,
    display_weight: blueprint.displayWeight,
    metadata,
  };
}

function generateDemoPlans(bundle: BundleProduct, needed: number, existingIds: Set<string>): PlanOption[] {
  const plans: PlanOption[] = [];
  for (const blueprint of TEST_PLAN_BLUEPRINTS) {
    const planId = `demo_${bundle.bundle_key}_${blueprint.id}`;
    if (existingIds.has(planId)) {
      continue;
    }
    plans.push(blueprintToPlan(bundle, blueprint));
    if (plans.length >= needed) {
      break;
    }
  }
  return plans;
}

export function isDemoPlanOption(option: { metadata?: PlanDisplayMetadata }): boolean {
  if (!option.metadata) {
    return false;
  }
  return Boolean((option.metadata as Record<string, unknown>)[DEMO_PLAN_FLAG]);
}

export function injectDemoPlansForBundle(entry: BundleCatalogEntry, minMonthlyCount = 3): BundleCatalogEntry {
  const monthlyRealCount = entry.prices.filter(
    (plan) => plan.billing_interval === 'month' && !isDemoPlanOption(plan)
  ).length;
  const needed = Math.max(0, minMonthlyCount - monthlyRealCount);
  if (needed === 0) {
    return entry;
  }

  const existingDemoIds = new Set(
    entry.prices.filter(isDemoPlanOption).map((plan) => plan.stripe_price_id)
  );
  const placeholders = generateDemoPlans(entry.bundle, needed, existingDemoIds);
  return {
    ...entry,
    prices: [...entry.prices, ...placeholders],
  };
}

export function ensureDemoPlansForDisplay(
  bundle: BundleProduct,
  plans: PlanOption[],
  minMonthlyCount = 3
): PlanOption[] {
  const monthlyRealCount = plans.filter(
    (plan) => plan.billing_interval === 'month' && !isDemoPlanOption(plan)
  ).length;
  const needed = Math.max(0, minMonthlyCount - monthlyRealCount);
  if (needed === 0) {
    return plans;
  }

  const existingDemoIds = new Set(plans.filter(isDemoPlanOption).map((plan) => plan.stripe_price_id));
  const placeholders = generateDemoPlans(bundle, needed, existingDemoIds);
  return [...plans, ...placeholders];
}
