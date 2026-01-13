/// <reference types="vite/client" />

type ProtoSchema<T> = unknown;

declare module '@proto-lprv/billing_pb' {
  export enum SubscriptionState {
    SUBSCRIPTION_STATE_UNSPECIFIED = 0,
    SUBSCRIPTION_STATE_ACTIVE = 1,
    SUBSCRIPTION_STATE_TRIALING = 2,
    SUBSCRIPTION_STATE_PAST_DUE = 3,
    SUBSCRIPTION_STATE_CANCELED = 4,
  }

  export interface SubscriptionStatus {
    state?: SubscriptionState;
    subscriptionId?: string;
    userIdentity?: string;
    planTier?: string;
    stripePriceId?: string;
    bundleKey?: string;
    cachedAt?: { toJsonString?: () => string };
  }

  export interface VerifySubscriptionResponse {
    status?: SubscriptionStatus;
  }

  export const VerifySubscriptionResponseSchema: ProtoSchema<VerifySubscriptionResponse>;
}

declare module '@proto-lprv/settings_pb' {
  export enum ConfigSource {
    CONFIG_SOURCE_UNSPECIFIED = 0,
    CONFIG_SOURCE_ENV = 1,
    CONFIG_SOURCE_DATABASE = 2,
  }

  export interface StripeConfigSnapshot {
    publishableKeyPreview?: string;
    publishableKeySet?: boolean;
    secretKeySet?: boolean;
    webhookSecretSet?: boolean;
    source?: ConfigSource | string | number;
  }

  export interface StripeSettings {
    dashboardUrl?: string;
    updatedAt?: { toJsonString?: () => string } | string | Date | { seconds?: number; nanos?: number };
  }

  export interface GetStripeSettingsResponse {
    snapshot?: StripeConfigSnapshot;
    settings?: StripeSettings;
  }

  export interface UpdateStripeSettingsResponse {
    snapshot?: StripeConfigSnapshot;
    settings?: StripeSettings;
  }

  export const GetStripeSettingsResponseSchema: ProtoSchema<GetStripeSettingsResponse>;
  export const UpdateStripeSettingsResponseSchema: ProtoSchema<UpdateStripeSettingsResponse>;
}

declare module '@proto-lprv/pricing_pb' {
  export enum BillingInterval {
    BILLING_INTERVAL_UNSPECIFIED = 0,
    BILLING_INTERVAL_MONTH = 1,
    BILLING_INTERVAL_YEAR = 2,
    BILLING_INTERVAL_ONE_TIME = 3,
  }

  export enum IntroPricingType {
    INTRO_PRICING_TYPE_UNSPECIFIED = 0,
    INTRO_PRICING_TYPE_FLAT_AMOUNT = 1,
    INTRO_PRICING_TYPE_PERCENTAGE = 2,
  }

  export enum PlanKind {
    PLAN_KIND_UNSPECIFIED = 0,
    PLAN_KIND_SUBSCRIPTION = 1,
    PLAN_KIND_CREDITS_TOPUP = 2,
    PLAN_KIND_SUPPORTER_CONTRIBUTION = 3,
  }

  export interface PricingBundle {
    bundleKey?: string;
    name?: string;
    stripeProductId?: string;
    creditsPerUsd?: number | string;
    displayCreditsMultiplier?: number | string;
    displayCreditsLabel?: string;
    environment?: string;
    metadata?: Record<string, { toJson?: () => unknown }>;
  }

  export interface PricingPlan {
    planName?: string;
    planTier?: string;
    billingInterval?: BillingInterval;
    amountCents?: number | string;
    currency?: string;
    introEnabled?: boolean;
    introType?: IntroPricingType;
    introAmountCents?: number | string;
    introPeriods?: number | string;
    introPriceLookupKey?: string;
    stripePriceId?: string;
    monthlyIncludedCredits?: number | string;
    oneTimeBonusCredits?: number | string;
    planRank?: number | string;
    bonusType?: string;
    kind?: PlanKind;
    isVariableAmount?: boolean;
    displayEnabled?: boolean;
    bundleKey?: string;
    displayWeight?: number | string;
    metadata?: Record<string, { toJson?: () => unknown }>;
  }

  export interface PricingPayload {
    bundle?: PricingBundle;
    monthly?: PricingPlan[];
    yearly?: PricingPlan[];
    updatedAt?: { toJsonString?: () => string } | string | { seconds?: number; nanos?: number };
  }

  export interface GetPricingResponse {
    pricing?: PricingPayload;
  }

  export const GetPricingResponseSchema: ProtoSchema<GetPricingResponse>;
}
