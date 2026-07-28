/// <reference types="vite/client" />

declare global {
  interface Response {
    json(): Promise<unknown>;
  }
}

type ProtoSchema<_T> = unknown;

declare module '@vrooli/proto-types/landing-page-business-suite/billing_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';
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

  export enum SessionKind {
    UNSPECIFIED = 0,
    SUBSCRIPTION = 1,
    CREDITS_TOPUP = 2,
    SUPPORTER_CONTRIBUTION = 3,
  }
  export interface CheckoutSession { sessionId: string; url: string; sessionKind: SessionKind; status: number; publishableKey: string; customerEmail: string; stripePriceId: string; amountCents: bigint; currency: string; successUrl: string; cancelUrl: string; }
  export interface CreateCheckoutSessionRequest extends Message<'landing_page_business_suite.v1.CreateCheckoutSessionRequest'> { priceId: string; customerEmail: string; successUrl: string; cancelUrl: string; sessionKind: SessionKind; }
  export interface CreateCheckoutSessionResponse extends Message<'landing_page_business_suite.v1.CreateCheckoutSessionResponse'> { session?: CheckoutSession; }
  export interface GetBillingPortalRequest extends Message<'landing_page_business_suite.v1.GetBillingPortalRequest'> { returnUrl: string; }
  export interface BillingPortalResponse extends Message<'landing_page_business_suite.v1.BillingPortalResponse'> { url: string; }
  export const CreateCheckoutSessionRequestSchema: GenMessage<CreateCheckoutSessionRequest>;
  export const CreateCheckoutSessionResponseSchema: GenMessage<CreateCheckoutSessionResponse>;
  export const GetBillingPortalRequestSchema: GenMessage<GetBillingPortalRequest>;
  export const BillingPortalResponseSchema: GenMessage<BillingPortalResponse>;
  export const LandingPagePaymentsService: GenService<{
    createCheckoutSession: { methodKind: 'unary'; input: typeof CreateCheckoutSessionRequestSchema; output: typeof CreateCheckoutSessionResponseSchema };
    getBillingPortal: { methodKind: 'unary'; input: typeof GetBillingPortalRequestSchema; output: typeof BillingPortalResponseSchema };
  }>;
}

declare module '@vrooli/proto-types/landing-page-business-suite/branding_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';
  export interface GetBrandingRequest extends Message<'landing_page_business_suite.v1.GetBrandingRequest'> {}
  export interface GetPublicBrandingRequest extends Message<'landing_page_business_suite.v1.GetPublicBrandingRequest'> {}
  export interface ClearBrandingFieldRequest extends Message<'landing_page_business_suite.v1.ClearBrandingFieldRequest'> { field: string; }
  export interface UpdateBrandingRequest extends Message<'landing_page_business_suite.v1.UpdateBrandingRequest'> { [key: string]: unknown; }
  export interface BrandingResponse extends Message<'landing_page_business_suite.v1.BrandingResponse'> { branding?: { toJson(): unknown; }; }
  export interface PublicBrandingResponse extends Message<'landing_page_business_suite.v1.PublicBrandingResponse'> { branding?: { toJson(): unknown; }; }
  export const GetBrandingRequestSchema: GenMessage<GetBrandingRequest>;
  export const GetPublicBrandingRequestSchema: GenMessage<GetPublicBrandingRequest>;
  export const ClearBrandingFieldRequestSchema: GenMessage<ClearBrandingFieldRequest>;
  export const UpdateBrandingRequestSchema: GenMessage<UpdateBrandingRequest>;
  export const BrandingResponseSchema: GenMessage<BrandingResponse>;
  export const PublicBrandingResponseSchema: GenMessage<PublicBrandingResponse>;
  export const BrandingService: GenService<{
    getBranding: { methodKind: 'unary'; input: typeof GetBrandingRequestSchema; output: typeof BrandingResponseSchema };
    updateBranding: { methodKind: 'unary'; input: typeof UpdateBrandingRequestSchema; output: typeof BrandingResponseSchema };
    clearBrandingField: { methodKind: 'unary'; input: typeof ClearBrandingFieldRequestSchema; output: typeof BrandingResponseSchema };
    getPublicBranding: { methodKind: 'unary'; input: typeof GetPublicBrandingRequestSchema; output: typeof PublicBrandingResponseSchema };
  }>;
}

declare module '@vrooli/proto-types/landing-page-business-suite/shared/commerce_pb' {
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

declare module '@vrooli/proto-types/landing-page-business-suite/settings_pb' {
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

declare module '@vrooli/proto-types/landing-page-business-suite/pricing_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';

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

  export interface GetPricingRequest extends Message<'landing_page_business_suite.v1.GetPricingRequest'> {
    bundleKey: string;
    includeHidden: boolean;
  }

  export const GetPricingRequestSchema: GenMessage<GetPricingRequest>;

  export interface GetPricingResponse extends Message<'landing_page_business_suite.v1.GetPricingResponse'> {
    pricing?: PricingPayload;
  }

  export const GetPricingResponseSchema: GenMessage<GetPricingResponse>;
  export const PricingService: GenService<{
    getPricing: {
      methodKind: 'unary';
      input: typeof GetPricingRequestSchema;
      output: typeof GetPricingResponseSchema;
    };
  }>;
}

declare module '@vrooli/proto-types/landing-page-business-suite/account_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';

  export interface GetMySubscriptionRequest extends Message<'landing_page_business_suite.v1.GetMySubscriptionRequest'> {}
  export interface GetMyCreditsRequest extends Message<'landing_page_business_suite.v1.GetMyCreditsRequest'> {}
  export interface GetEntitlementsRequest extends Message<'landing_page_business_suite.v1.GetEntitlementsRequest'> {}
  export interface SubscriptionStatus { state?: number; subscriptionId?: string; userIdentity?: string; planTier?: string; stripePriceId?: string; bundleKey?: string; cachedAt?: { toJsonString?: () => string }; }
  export interface CreditsBalance { customerEmail?: string; balanceCredits?: number; bundleKey?: string; }
  export interface GetMySubscriptionResponse extends Message<'vrooli.landing_page_business_suite.v1.shared.VerifySubscriptionResponse'> { status?: SubscriptionStatus; }
  export interface GetMyCreditsResponse extends Message<'landing_page_business_suite.v1.GetMyCreditsResponse'> { balance?: CreditsBalance; displayCreditsLabel: string; displayCreditsMultiplier: number; }
  export interface GetEntitlementsResponse extends Message<'landing_page_business_suite.v1.GetEntitlementsResponse'> { status: string; planTier: string; priceId: string; features: string[]; credits?: CreditsBalance; subscription?: SubscriptionStatus; billingCycleStart: number; }
  export const GetMySubscriptionRequestSchema: GenMessage<GetMySubscriptionRequest>;
  export const GetMyCreditsRequestSchema: GenMessage<GetMyCreditsRequest>;
  export const GetEntitlementsRequestSchema: GenMessage<GetEntitlementsRequest>;
  export const GetMySubscriptionResponseSchema: GenMessage<GetMySubscriptionResponse>;
  export const GetMyCreditsResponseSchema: GenMessage<GetMyCreditsResponse>;
  export const GetEntitlementsResponseSchema: GenMessage<GetEntitlementsResponse>;
  export const AccountService: GenService<{
    getMySubscription: { methodKind: 'unary'; input: typeof GetMySubscriptionRequestSchema; output: typeof GetMySubscriptionResponseSchema };
    getMyCredits: { methodKind: 'unary'; input: typeof GetMyCreditsRequestSchema; output: typeof GetMyCreditsResponseSchema };
    getEntitlements: { methodKind: 'unary'; input: typeof GetEntitlementsRequestSchema; output: typeof GetEntitlementsResponseSchema };
  }>;
}
