import type {
  LandingConfigResponse,
  LandingSection,
  DownloadApp,
  PricingOverview,
  VariantAxes,
  LandingHeaderConfig,
  PlanOption,
} from '../api';
import rawFallback from '../../../../.vrooli/fallback/fallback.json';
import { normalizeHeaderConfig } from './headerConfig';

interface FallbackPayload {
  variant: LandingConfigResponse['variant'];
  sections?: LandingSection[] | null;
  pricing?: Partial<PricingOverview>;
  downloads?: DownloadApp[];
  axes?: VariantAxes;
  header?: LandingHeaderConfig;
}

function normalizeSections(sectionsInput?: LandingSection[] | null): LandingSection[] {
  const sections = Array.isArray(sectionsInput) ? sectionsInput : [];

  return sections.map((section, index) => ({
    ...section,
    order: typeof section.order === 'number' && Number.isFinite(section.order) ? section.order : index + 1,
    enabled: section.enabled ?? true,
  }));
}

function normalizePlan(plan: Partial<PlanOption>): PlanOption {
  return {
    plan_name: plan.plan_name ?? '',
    plan_tier: plan.plan_tier ?? '',
    billing_interval: plan.billing_interval ?? 'month',
    amount_cents: plan.amount_cents ?? 0,
    currency: plan.currency ?? 'usd',
    intro_enabled: plan.intro_enabled ?? false,
    intro_type: plan.intro_type,
    intro_amount_cents: plan.intro_amount_cents,
    intro_periods: plan.intro_periods,
    intro_price_lookup_key: plan.intro_price_lookup_key,
    stripe_price_id: plan.stripe_price_id ?? '',
    monthly_included_credits: plan.monthly_included_credits ?? 0,
    one_time_bonus_credits: plan.one_time_bonus_credits ?? 0,
    plan_rank: plan.plan_rank,
    bonus_type: plan.bonus_type,
    kind: plan.kind,
    is_variable_amount: plan.is_variable_amount,
    display_enabled: plan.display_enabled ?? true,
    bundle_key: plan.bundle_key,
    display_weight: plan.display_weight ?? 0,
    metadata: plan.metadata,
  };
}

function normalizePricing(pricing?: Partial<PricingOverview>): PricingOverview | undefined {
  if (!pricing || !pricing.bundle) return undefined;
  const monthly = Array.isArray(pricing.monthly) ? pricing.monthly.map(normalizePlan) : [];
  const yearly = Array.isArray(pricing.yearly) ? pricing.yearly.map(normalizePlan) : [];
  return {
    bundle: pricing.bundle,
    monthly,
    yearly,
    updated_at: pricing.updated_at ?? new Date().toISOString(),
  };
}

const FALLBACK_CONFIG: LandingConfigResponse = (() => {
  const payload = rawFallback as unknown as FallbackPayload;
  const variant = {
    ...payload.variant,
    axes: payload.variant.axes ?? payload.axes,
  };

  return {
    variant,
    sections: normalizeSections(payload.sections),
    pricing: normalizePricing(payload.pricing),
    downloads: payload.downloads ?? [],
    header: normalizeHeaderConfig(payload.header, variant.name),
    fallback: true,
  };
})();

export function getFallbackLandingConfig(): LandingConfigResponse {
  return JSON.parse(JSON.stringify(FALLBACK_CONFIG));
}
