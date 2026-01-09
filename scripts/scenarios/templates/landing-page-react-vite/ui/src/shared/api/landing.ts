import { fromJson } from '@bufbuild/protobuf';
import {
  BillingInterval,
  GetPricingResponseSchema,
  IntroPricingType,
  PlanKind,
} from '@proto-lprv/pricing_pb';
import { apiCall } from './common';
import type { LandingConfigResponse, PlanOption, PricingOverview } from './types';

export function getLandingConfig(variantSlug?: string) {
  const params = new URLSearchParams();
  if (variantSlug) {
    params.set('variant', variantSlug);
  }
  const query = params.toString() ? `?${params.toString()}` : '';
  return apiCall<LandingConfigResponse>(`/landing-config${query}`);
}

export function getPlans() {
  return apiCall('/plans').then((resp) => {
    const message = fromJson(GetPricingResponseSchema, resp, {
      ignoreUnknownFields: true,
      protoFieldName: true,
    });
    const toObjectMap = (input?: Record<string, { toJson: () => unknown }>) => {
      if (!input) return undefined;
      return Object.fromEntries(
        Object.entries(input).map(([key, value]) => [key, value?.toJson?.() ?? null])
      );
    };
    const planKind = (kind?: PlanKind): PlanOption['kind'] => {
      switch (kind) {
        case PlanKind.PLAN_KIND_CREDITS_TOPUP:
          return 'credits_topup';
        case PlanKind.PLAN_KIND_SUPPORTER_CONTRIBUTION:
          return 'supporter_contribution';
        default:
          return 'subscription';
      }
    };

    const billingInterval = (interval?: BillingInterval): PlanOption['billing_interval'] => {
      switch (interval) {
        case BillingInterval.BILLING_INTERVAL_YEAR:
          return 'year';
        case BillingInterval.BILLING_INTERVAL_ONE_TIME:
          return 'one_time';
        default:
          return 'month';
      }
    };

    const introType = (type?: IntroPricingType): PlanOption['intro_type'] => {
      switch (type) {
        case IntroPricingType.INTRO_PRICING_TYPE_PERCENTAGE:
          return 'percentage';
        case IntroPricingType.INTRO_PRICING_TYPE_FLAT_AMOUNT:
          return 'flat_amount';
        default:
          return undefined;
      }
    };

    const normalizePlan = (plan: any): PlanOption => ({
      plan_name: plan.planName,
      plan_tier: plan.planTier,
      billing_interval: billingInterval(plan.billingInterval),
      amount_cents: Number(plan.amountCents ?? 0),
      currency: plan.currency,
      intro_enabled: Boolean(plan.introEnabled),
      intro_type: introType(plan.introType),
      intro_amount_cents: plan.introAmountCents != null ? Number(plan.introAmountCents) : undefined,
      intro_periods: plan.introPeriods != null ? Number(plan.introPeriods) : undefined,
      intro_price_lookup_key: plan.introPriceLookupKey,
      stripe_price_id: plan.stripePriceId,
      monthly_included_credits: Number(plan.monthlyIncludedCredits ?? 0),
      one_time_bonus_credits: Number(plan.oneTimeBonusCredits ?? 0),
      plan_rank: plan.planRank != null ? Number(plan.planRank) : undefined,
      bonus_type: plan.bonusType,
      kind: planKind(plan.kind),
      is_variable_amount: Boolean(plan.isVariableAmount),
      display_enabled: Boolean(plan.displayEnabled),
      bundle_key: plan.bundleKey,
      display_weight: Number(plan.displayWeight ?? 0),
      metadata: toObjectMap(plan.metadata),
    });

    const pricing = message.pricing;
    const overview: PricingOverview = {
      bundle: {
        bundle_key: pricing?.bundle?.bundleKey ?? '',
        name: pricing?.bundle?.name ?? '',
        stripe_product_id: pricing?.bundle?.stripeProductId ?? '',
        credits_per_usd: Number(pricing?.bundle?.creditsPerUsd ?? 0),
        display_credits_multiplier: Number(pricing?.bundle?.displayCreditsMultiplier ?? 0),
        display_credits_label: pricing?.bundle?.displayCreditsLabel ?? 'credits',
        environment: pricing?.bundle?.environment ?? 'production',
        metadata: toObjectMap(pricing?.bundle?.metadata),
      },
      monthly: (pricing?.monthly ?? []).map(normalizePlan),
      yearly: (pricing?.yearly ?? []).map(normalizePlan),
      updated_at: pricing?.updatedAt?.toJsonString() ?? new Date().toISOString(),
    };
    return overview;
  });
}
