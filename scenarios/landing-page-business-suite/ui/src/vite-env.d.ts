/// <reference types="vite/client" />

declare module '@proto-lprv/billing_pb' {
  export enum SubscriptionState {
    SUBSCRIPTION_STATE_UNSPECIFIED = 0,
    SUBSCRIPTION_STATE_ACTIVE = 1,
    SUBSCRIPTION_STATE_TRIALING = 2,
    SUBSCRIPTION_STATE_PAST_DUE = 3,
    SUBSCRIPTION_STATE_CANCELED = 4,
  }

  export const VerifySubscriptionResponseSchema: unknown;
}

declare module '@proto-lprv/settings_pb' {
  export enum ConfigSource {
    CONFIG_SOURCE_UNSPECIFIED = 0,
    CONFIG_SOURCE_ENV = 1,
    CONFIG_SOURCE_DATABASE = 2,
  }

  export type StripeConfigSnapshot = Record<string, unknown>;
  export type StripeSettings = Record<string, unknown>;

  export const GetStripeSettingsResponseSchema: unknown;
  export const UpdateStripeSettingsResponseSchema: unknown;
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

  export const GetPricingResponseSchema: unknown;
}
