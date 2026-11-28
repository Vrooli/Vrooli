import { create } from '@bufbuild/protobuf';
import { PlanOptionSchema } from '@vrooli/proto-types/landing-page-react-vite/v1/pricing_pb';
import {
  BillingInterval,
  PlanKind,
  jsonMapToRecord,
  recordToJsonMap,
  type BundleCatalogEntry,
  type Bundle,
  type PlanOption,
} from '../api';

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

function blueprintToPlan(bundle: Bundle, blueprint: TestPlanBlueprint): PlanOption {
  const metadata = recordToJsonMap({
    subtitle: blueprint.subtitle,
    badge: blueprint.badge,
    cta_label: blueprint.ctaLabel,
    highlight: blueprint.highlight ?? false,
    features: blueprint.features,
    [DEMO_PLAN_FLAG]: true,
  });

  return create(PlanOptionSchema, {
    planName: `${blueprint.planName} (Demo)`,
    planTier: blueprint.planTier,
    billingInterval: BillingInterval.MONTH,
    amountCents: BigInt(blueprint.amountCents),
    currency: 'usd',
    introEnabled: Boolean(blueprint.introEnabled),
    introAmountCents: blueprint.introAmountCents != null ? BigInt(blueprint.introAmountCents) : undefined,
    introPeriods: blueprint.introEnabled ? 1 : 0,
    stripePriceId: `demo_${bundle.bundleKey}_${blueprint.id}`,
    monthlyIncludedCredits: BigInt(blueprint.monthlyCredits),
    oneTimeBonusCredits: BigInt(blueprint.bonusCredits),
    planRank: blueprint.planRank,
    bonusType: 'none',
    kind: PlanKind.SUBSCRIPTION,
    isVariableAmount: false,
    displayEnabled: true,
    bundleKey: bundle.bundleKey,
    displayWeight: blueprint.displayWeight,
    metadata,
  });
}

function generateDemoPlans(bundle: Bundle, needed: number, existingIds: Set<string>): PlanOption[] {
  const plans: PlanOption[] = [];
  for (const blueprint of TEST_PLAN_BLUEPRINTS) {
    const planId = `demo_${bundle.bundleKey}_${blueprint.id}`;
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

export function isDemoPlanOption(option: Pick<PlanOption, 'metadata'>): boolean {
  const meta = jsonMapToRecord(option.metadata);
  return Boolean(meta[DEMO_PLAN_FLAG]);
}

export function injectDemoPlansForBundle(entry: BundleCatalogEntry, minMonthlyCount = 3): BundleCatalogEntry {
  const monthlyRealCount = entry.prices.filter(
    (plan) => plan.billingInterval === BillingInterval.MONTH && !isDemoPlanOption(plan)
  ).length;
  const needed = Math.max(0, minMonthlyCount - monthlyRealCount);
  if (needed === 0 || !entry.bundle) {
    return entry;
  }

  const existingDemoIds = new Set(
    entry.prices.filter(isDemoPlanOption).map((plan) => plan.stripePriceId)
  );
  const placeholders = generateDemoPlans(entry.bundle, needed, existingDemoIds);
  return {
    ...entry,
    prices: [...entry.prices, ...placeholders],
  };
}

export function ensureDemoPlansForDisplay(
  bundle: Bundle,
  plans: PlanOption[],
  minMonthlyCount = 3
): PlanOption[] {
  const monthlyRealCount = plans.filter(
    (plan) => plan.billingInterval === BillingInterval.MONTH && !isDemoPlanOption(plan)
  ).length;
  const needed = Math.max(0, minMonthlyCount - monthlyRealCount);
  if (needed === 0) {
    return plans;
  }

  const existingDemoIds = new Set(plans.filter(isDemoPlanOption).map((plan) => plan.stripePriceId));
  const placeholders = generateDemoPlans(bundle, needed, existingDemoIds);
  return [...plans, ...placeholders];
}
