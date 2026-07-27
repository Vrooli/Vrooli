/// <reference types="vite/client" />

declare global {
  interface Response {
    json(): Promise<unknown>;
  }
}

type ProtoSchema<_T> = unknown;

declare module '@proto-lpbs/billing_pb' {
  export enum SubscriptionState {
    UNSPECIFIED = 0,
    ACTIVE = 1,
    TRIALING = 2,
    PAST_DUE = 3,
    CANCELED = 4,
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

declare module '@proto-lpbs/shared/commerce_pb' {
  export enum SubscriptionState {
    UNSPECIFIED = 0,
    ACTIVE = 1,
    TRIALING = 2,
    PAST_DUE = 3,
    CANCELED = 4,
    INACTIVE = 5,
  }

  export enum BillingInterval {
    UNSPECIFIED = 0,
    MONTH = 1,
    YEAR = 2,
    ONE_TIME = 3,
  }

  export enum IntroPricingType {
    UNSPECIFIED = 0,
    FLAT_AMOUNT = 1,
    PERCENTAGE = 2,
  }

  export enum PlanKind {
    UNSPECIFIED = 0,
    SUBSCRIPTION = 1,
    CREDITS_TOPUP = 2,
    SUPPORTER_CONTRIBUTION = 3,
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

declare module '@proto-lpbs/settings_pb' {
  export enum ConfigSource {
    UNSPECIFIED = 0,
    ENV = 1,
    DATABASE = 2,
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

declare module '@proto-lpbs/pricing_pb' {
  export enum BillingInterval {
    UNSPECIFIED = 0,
    MONTH = 1,
    YEAR = 2,
    ONE_TIME = 3,
  }

  export enum IntroPricingType {
    UNSPECIFIED = 0,
    FLAT_AMOUNT = 1,
    PERCENTAGE = 2,
  }

  export enum PlanKind {
    UNSPECIFIED = 0,
    SUBSCRIPTION = 1,
    CREDITS_TOPUP = 2,
    SUPPORTER_CONTRIBUTION = 3,
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
