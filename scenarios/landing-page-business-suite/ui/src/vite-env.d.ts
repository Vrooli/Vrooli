/// <reference types="vite/client" />

declare global {
  interface Response {
    json(): Promise<unknown>;
  }
}

type ProtoSchema<_T> = unknown;

declare module '@vrooli/proto-types/landing-page-business-suite/shared/downloads_pb' {
  import type { Message } from '@bufbuild/protobuf';
  export interface DownloadStorefront { store: string; label: string; url: string; badge: string; }
  export interface DownloadAsset extends Message<'vrooli.landing_page_business_suite.v1.shared.DownloadAsset'> {
    id: bigint; bundleKey: string; appKey: string; platform: string; artifactUrl: string; artifactSource: string;
    artifactId?: bigint; releaseVersion: string; releaseNotes: string; checksum: string; requiresEntitlement: boolean;
    metadata?: Record<string, unknown>; variantKey: string; artifactFilename: string; artifactSizeBytes: bigint; artifactCount: number;
  }
  export interface DownloadApp extends Message<'vrooli.landing_page_business_suite.v1.shared.DownloadApp'> {
    id: bigint; bundleKey: string; appKey: string; name: string; tagline: string; description: string;
    iconUrl: string; screenshotUrl: string; installOverview: string; installSteps: string[]; storefronts: DownloadStorefront[];
    metadata?: Record<string, unknown>; displayOrder: number; platforms: DownloadAsset[]; updateApiKey: string;
    updatePolicy?: Record<string, unknown>;
  }
}

declare module '@vrooli/proto-types/landing-page-business-suite/download_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';
  import type { DownloadApp, DownloadAsset } from '@vrooli/proto-types/landing-page-business-suite/shared/downloads_pb';
  export interface AuthorizeDownloadRequest extends Message<'landing_page_business_suite.v1.AuthorizeDownloadRequest'> { app: string; platform: string; }
  export interface AuthorizeDownloadResponse extends Message<'landing_page_business_suite.v1.AuthorizeDownloadResponse'> { asset?: DownloadAsset; }
  export type ListDownloadAppsRequest = Message<'landing_page_business_suite.v1.ListDownloadAppsRequest'>;
  export interface ListDownloadAppsResponse extends Message<'landing_page_business_suite.v1.ListDownloadAppsResponse'> { apps: DownloadApp[]; }
  export interface CreateDownloadAppRequest extends Message<'landing_page_business_suite.v1.CreateDownloadAppRequest'> { app?: DownloadApp; }
  export interface SaveDownloadAppRequest extends Message<'landing_page_business_suite.v1.SaveDownloadAppRequest'> { appKey: string; app?: DownloadApp; }
  export interface DownloadAppResponse extends Message<'landing_page_business_suite.v1.DownloadAppResponse'> { app?: DownloadApp; }
  export interface DeleteDownloadAppRequest extends Message<'landing_page_business_suite.v1.DeleteDownloadAppRequest'> { appKey: string; }
  export type DeleteDownloadAppResponse = Message<'landing_page_business_suite.v1.DeleteDownloadAppResponse'>;
  export const AuthorizeDownloadRequestSchema: GenMessage<AuthorizeDownloadRequest>;
  export const AuthorizeDownloadResponseSchema: GenMessage<AuthorizeDownloadResponse>;
  export const ListDownloadAppsRequestSchema: GenMessage<ListDownloadAppsRequest>;
  export const ListDownloadAppsResponseSchema: GenMessage<ListDownloadAppsResponse>;
  export const CreateDownloadAppRequestSchema: GenMessage<CreateDownloadAppRequest>;
  export const SaveDownloadAppRequestSchema: GenMessage<SaveDownloadAppRequest>;
  export const DownloadAppResponseSchema: GenMessage<DownloadAppResponse>;
  export const DeleteDownloadAppRequestSchema: GenMessage<DeleteDownloadAppRequest>;
  export const DeleteDownloadAppResponseSchema: GenMessage<DeleteDownloadAppResponse>;
  export const DownloadService: GenService<{
    authorizeDownload: { methodKind: 'unary'; input: typeof AuthorizeDownloadRequestSchema; output: typeof AuthorizeDownloadResponseSchema };
    listDownloadApps: { methodKind: 'unary'; input: typeof ListDownloadAppsRequestSchema; output: typeof ListDownloadAppsResponseSchema };
    createDownloadApp: { methodKind: 'unary'; input: typeof CreateDownloadAppRequestSchema; output: typeof DownloadAppResponseSchema };
    saveDownloadApp: { methodKind: 'unary'; input: typeof SaveDownloadAppRequestSchema; output: typeof DownloadAppResponseSchema };
    deleteDownloadApp: { methodKind: 'unary'; input: typeof DeleteDownloadAppRequestSchema; output: typeof DeleteDownloadAppResponseSchema };
  }>;
}

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
  export type GetBrandingRequest = Message<'landing_page_business_suite.v1.GetBrandingRequest'>;
  export type GetPublicBrandingRequest = Message<'landing_page_business_suite.v1.GetPublicBrandingRequest'>;
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

declare module '@vrooli/proto-types/landing-page-business-suite/assets_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';

  export interface Asset extends Message<'landing_page_business_suite.v1.Asset'> {
    id: bigint;
    filename: string;
    originalFilename: string;
    mimeType: string;
    sizeBytes: bigint;
    storagePath: string;
    thumbnailPath?: string;
    altText?: string;
    category: string;
    uploadedBy?: string;
    createdAt?: { seconds: bigint; nanos: number };
    url: string;
    derivatives: Record<string, string>;
  }

  export interface ListAssetsRequest extends Message<'landing_page_business_suite.v1.ListAssetsRequest'> { category: string; }
  export interface ListAssetsResponse extends Message<'landing_page_business_suite.v1.ListAssetsResponse'> { assets: Asset[]; }
  export interface GetAssetRequest extends Message<'landing_page_business_suite.v1.GetAssetRequest'> { id: bigint; }
  export interface AssetResponse extends Message<'landing_page_business_suite.v1.AssetResponse'> { asset?: Asset; }
  export interface DeleteAssetRequest extends Message<'landing_page_business_suite.v1.DeleteAssetRequest'> { id: bigint; }
  export interface DeleteAssetResponse extends Message<'landing_page_business_suite.v1.DeleteAssetResponse'> { deleted: boolean; }

  export const ListAssetsRequestSchema: GenMessage<ListAssetsRequest>;
  export const ListAssetsResponseSchema: GenMessage<ListAssetsResponse>;
  export const GetAssetRequestSchema: GenMessage<GetAssetRequest>;
  export const AssetResponseSchema: GenMessage<AssetResponse>;
  export const DeleteAssetRequestSchema: GenMessage<DeleteAssetRequest>;
  export const DeleteAssetResponseSchema: GenMessage<DeleteAssetResponse>;

  export const AssetsService: GenService<{
    listAssets: { methodKind: 'unary'; input: typeof ListAssetsRequestSchema; output: typeof ListAssetsResponseSchema };
    getAsset: { methodKind: 'unary'; input: typeof GetAssetRequestSchema; output: typeof AssetResponseSchema };
    deleteAsset: { methodKind: 'unary'; input: typeof DeleteAssetRequestSchema; output: typeof DeleteAssetResponseSchema };
  }>;
}

declare module '@vrooli/proto-types/landing-page-business-suite/docs_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';

  export interface DocEntry extends Message<'landing_page_business_suite.v1.DocEntry'> {
    name: string;
    path: string;
    isDir: boolean;
    children: DocEntry[];
  }
  export interface GetDocsTreeResponse extends Message<'landing_page_business_suite.v1.GetDocsTreeResponse'> {
    entries: DocEntry[];
  }
  export type GetDocsTreeRequest = Message<'landing_page_business_suite.v1.GetDocsTreeRequest'>;
  export interface GetDocContentRequest extends Message<'landing_page_business_suite.v1.GetDocContentRequest'> { path: string; }
  export interface GetDocContentResponse extends Message<'landing_page_business_suite.v1.GetDocContentResponse'> {
    path: string;
    content: string;
    title: string;
  }
  export const GetDocsTreeRequestSchema: GenMessage<GetDocsTreeRequest>;
  export const GetDocsTreeResponseSchema: GenMessage<GetDocsTreeResponse>;
  export const GetDocContentRequestSchema: GenMessage<GetDocContentRequest>;
  export const GetDocContentResponseSchema: GenMessage<GetDocContentResponse>;
  export const DocsService: GenService<{
    getDocsTree: { methodKind: 'unary'; input: typeof GetDocsTreeRequestSchema; output: typeof GetDocsTreeResponseSchema };
    getDocContent: { methodKind: 'unary'; input: typeof GetDocContentRequestSchema; output: typeof GetDocContentResponseSchema };
  }>;
}

declare module '@vrooli/proto-types/landing-page-business-suite/bundles_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';
  import type { BillingInterval, IntroPricingType, PlanKind } from '@vrooli/proto-types/landing-page-business-suite/shared/commerce_pb';
  type ProtoValue = { toJson?: () => unknown };
  export interface Bundle { bundleKey?: string; name?: string; stripeProductId?: string; creditsPerUsd?: bigint | number; displayCreditsMultiplier?: number; displayCreditsLabel?: string; environment?: string; metadata?: Record<string, ProtoValue>; }
  export interface PlanOption { planName?: string; planTier?: string; billingInterval?: BillingInterval; amountCents?: bigint | number; currency?: string; introEnabled?: boolean; introType?: IntroPricingType; introAmountCents?: bigint | number; introPeriods?: number; introPriceLookupKey?: string; stripePriceId?: string; monthlyIncludedCredits?: bigint | number; oneTimeBonusCredits?: bigint | number; planRank?: number; bonusType?: string; kind?: PlanKind; isVariableAmount?: boolean; displayEnabled?: boolean; bundleKey?: string; displayWeight?: number; metadata?: Record<string, ProtoValue>; }
  export interface BundleCatalogEntry { bundle?: Bundle; prices?: PlanOption[]; }
  export type ListBundleCatalogRequest = Message<'landing_page_business_suite.v1.ListBundleCatalogRequest'>;
  export interface ListBundleCatalogResponse extends Message<'landing_page_business_suite.v1.ListBundleCatalogResponse'> { bundles?: BundleCatalogEntry[]; }
  export interface UpdateBundlePriceRequest extends Message<'landing_page_business_suite.v1.UpdateBundlePriceRequest'> { bundleKey: string; priceId: string; stripePriceId?: string; planName?: string; displayWeight?: number; displayEnabled?: boolean; subtitle?: string; badge?: string; ctaLabel?: string; highlight?: boolean; features?: string[]; featuresPresent?: boolean; }
  export interface UpdateBundlePriceResponse extends Message<'landing_page_business_suite.v1.UpdateBundlePriceResponse'> { price?: PlanOption; }
  export const ListBundleCatalogRequestSchema: GenMessage<ListBundleCatalogRequest>;
  export const ListBundleCatalogResponseSchema: GenMessage<ListBundleCatalogResponse>;
  export const UpdateBundlePriceRequestSchema: GenMessage<UpdateBundlePriceRequest>;
  export const UpdateBundlePriceResponseSchema: GenMessage<UpdateBundlePriceResponse>;
  export const BundleAdminService: GenService<{
    listBundleCatalog: { methodKind: 'unary'; input: typeof ListBundleCatalogRequestSchema; output: typeof ListBundleCatalogResponseSchema };
    updateBundlePrice: { methodKind: 'unary'; input: typeof UpdateBundlePriceRequestSchema; output: typeof UpdateBundlePriceResponseSchema };
  }>;
}

declare module '@vrooli/proto-types/landing-page-business-suite/coupons_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';
  export enum CouponDuration { UNSPECIFIED = 0, ONCE = 1, REPEATING = 2, FOREVER = 3 }
  export interface Coupon { id: string; name?: string; amountOff?: bigint; percentOff?: number; currency?: string; duration: CouponDuration; durationInMonths?: number; maxRedemptions?: number; redeemBy?: bigint; timesRedeemed: number; valid: boolean; created: bigint; isIntroCoupon: boolean; introTier?: string; }
  export interface CouponUsageStat { couponId: string; totalUses: bigint; lastUsedAt?: string; }
  export interface CouponImportPreviewItem { id: string; name?: string; amountOff?: bigint; percentOff?: number; currency?: string; duration: CouponDuration; durationInMonths?: number; timesRedeemed: number; valid: boolean; existsLocally: boolean; }
  export type ListCouponsRequest = Message<'landing_page_business_suite.v1.ListCouponsRequest'>;
  export interface ListCouponsResponse extends Message<'landing_page_business_suite.v1.ListCouponsResponse'> { coupons?: Coupon[]; introCouponMap?: Record<string, string>; }
  export interface CreateCouponRequest extends Message<'landing_page_business_suite.v1.CreateCouponRequest'> { id?: string; name?: string; amountOff?: bigint; percentOff?: number; currency?: string; duration: CouponDuration; durationInMonths?: number; maxRedemptions?: number; redeemBy?: bigint; }
  export interface CreateCouponResponse extends Message<'landing_page_business_suite.v1.CreateCouponResponse'> { coupon?: Coupon; }
  export interface GetCouponRequest extends Message<'landing_page_business_suite.v1.GetCouponRequest'> { couponId: string; }
  export interface GetCouponResponse extends Message<'landing_page_business_suite.v1.GetCouponResponse'> { coupon?: Coupon; }
  export interface UpdateCouponRequest extends Message<'landing_page_business_suite.v1.UpdateCouponRequest'> { couponId: string; name?: string; }
  export interface UpdateCouponResponse extends Message<'landing_page_business_suite.v1.UpdateCouponResponse'> { coupon?: Coupon; }
  export interface DeleteCouponRequest extends Message<'landing_page_business_suite.v1.DeleteCouponRequest'> { couponId: string; }
  export interface DeleteCouponResponse extends Message<'landing_page_business_suite.v1.DeleteCouponResponse'> { deleted: boolean; }
  export type ListCouponUsageRequest = Message<'landing_page_business_suite.v1.ListCouponUsageRequest'>;
  export interface ListCouponUsageResponse extends Message<'landing_page_business_suite.v1.ListCouponUsageResponse'> { usage?: CouponUsageStat[]; }
  export type GetCouponMappingsRequest = Message<'landing_page_business_suite.v1.GetCouponMappingsRequest'>;
  export interface GetCouponMappingsResponse extends Message<'landing_page_business_suite.v1.GetCouponMappingsResponse'> { mappings?: Record<string, string>; }
  export interface SetCouponForPlanRequest extends Message<'landing_page_business_suite.v1.SetCouponForPlanRequest'> { priceId: string; couponId: string; }
  export interface SetCouponForPlanResponse extends Message<'landing_page_business_suite.v1.SetCouponForPlanResponse'> { assigned: boolean; }
  export interface RemoveCouponFromPlanRequest extends Message<'landing_page_business_suite.v1.RemoveCouponFromPlanRequest'> { priceId: string; }
  export interface RemoveCouponFromPlanResponse extends Message<'landing_page_business_suite.v1.RemoveCouponFromPlanResponse'> { removed: boolean; }
  export type GetCouponImportPreviewRequest = Message<'landing_page_business_suite.v1.GetCouponImportPreviewRequest'>;
  export interface GetCouponImportPreviewResponse extends Message<'landing_page_business_suite.v1.GetCouponImportPreviewResponse'> { coupons?: CouponImportPreviewItem[]; totalCoupons: number; existingCount: number; newCount: number; }
  export const ListCouponsRequestSchema: GenMessage<ListCouponsRequest>; export const ListCouponsResponseSchema: GenMessage<ListCouponsResponse>; export const CreateCouponRequestSchema: GenMessage<CreateCouponRequest>; export const CreateCouponResponseSchema: GenMessage<CreateCouponResponse>; export const GetCouponRequestSchema: GenMessage<GetCouponRequest>; export const GetCouponResponseSchema: GenMessage<GetCouponResponse>; export const UpdateCouponRequestSchema: GenMessage<UpdateCouponRequest>; export const UpdateCouponResponseSchema: GenMessage<UpdateCouponResponse>; export const DeleteCouponRequestSchema: GenMessage<DeleteCouponRequest>; export const DeleteCouponResponseSchema: GenMessage<DeleteCouponResponse>; export const ListCouponUsageRequestSchema: GenMessage<ListCouponUsageRequest>; export const ListCouponUsageResponseSchema: GenMessage<ListCouponUsageResponse>; export const GetCouponMappingsRequestSchema: GenMessage<GetCouponMappingsRequest>; export const GetCouponMappingsResponseSchema: GenMessage<GetCouponMappingsResponse>; export const SetCouponForPlanRequestSchema: GenMessage<SetCouponForPlanRequest>; export const SetCouponForPlanResponseSchema: GenMessage<SetCouponForPlanResponse>; export const RemoveCouponFromPlanRequestSchema: GenMessage<RemoveCouponFromPlanRequest>; export const RemoveCouponFromPlanResponseSchema: GenMessage<RemoveCouponFromPlanResponse>; export const GetCouponImportPreviewRequestSchema: GenMessage<GetCouponImportPreviewRequest>; export const GetCouponImportPreviewResponseSchema: GenMessage<GetCouponImportPreviewResponse>;
  export const CouponAdminService: GenService<{ listCoupons: { methodKind: 'unary'; input: typeof ListCouponsRequestSchema; output: typeof ListCouponsResponseSchema }; createCoupon: { methodKind: 'unary'; input: typeof CreateCouponRequestSchema; output: typeof CreateCouponResponseSchema }; getCoupon: { methodKind: 'unary'; input: typeof GetCouponRequestSchema; output: typeof GetCouponResponseSchema }; updateCoupon: { methodKind: 'unary'; input: typeof UpdateCouponRequestSchema; output: typeof UpdateCouponResponseSchema }; deleteCoupon: { methodKind: 'unary'; input: typeof DeleteCouponRequestSchema; output: typeof DeleteCouponResponseSchema }; listCouponUsage: { methodKind: 'unary'; input: typeof ListCouponUsageRequestSchema; output: typeof ListCouponUsageResponseSchema }; getCouponMappings: { methodKind: 'unary'; input: typeof GetCouponMappingsRequestSchema; output: typeof GetCouponMappingsResponseSchema }; setCouponForPlan: { methodKind: 'unary'; input: typeof SetCouponForPlanRequestSchema; output: typeof SetCouponForPlanResponseSchema }; removeCouponFromPlan: { methodKind: 'unary'; input: typeof RemoveCouponFromPlanRequestSchema; output: typeof RemoveCouponFromPlanResponseSchema }; getCouponImportPreview: { methodKind: 'unary'; input: typeof GetCouponImportPreviewRequestSchema; output: typeof GetCouponImportPreviewResponseSchema } }>;
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
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';
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
    publishableKey?: string;
    secretKey?: string;
    webhookSecret?: string;
    dashboardUrl?: string;
    anomalyWebhookUrl?: string;
    anomalyWebhookEnabled?: boolean;
    anomalyRateLimits?: string;
    updatedAt?: { toJsonString?: () => string } | string | Date | { seconds?: number; nanos?: number };
  }

  export type GetStripeSettingsRequest = Message<'landing_page_business_suite.v1.GetStripeSettingsRequest'>;
  export interface UpdateStripeSettingsRequest extends Message<'landing_page_business_suite.v1.UpdateStripeSettingsRequest'> {
    publishableKey?: string; secretKey?: string; webhookSecret?: string; dashboardUrl?: string;
    anomalyWebhookUrl?: string; anomalyWebhookEnabled?: boolean; anomalyRateLimits?: string;
  }
  export interface RevealStripeSecretRequest extends Message<'landing_page_business_suite.v1.RevealStripeSecretRequest'> { field: string; }
  export interface RevealStripeSecretResponse extends Message<'landing_page_business_suite.v1.RevealStripeSecretResponse'> { field: string; value: string; }
  export interface GetStripeSettingsResponse extends Message<'landing_page_business_suite.v1.GetStripeSettingsResponse'> {
    snapshot?: StripeConfigSnapshot;
    settings?: StripeSettings;
  }

  export interface UpdateStripeSettingsResponse extends Message<'landing_page_business_suite.v1.UpdateStripeSettingsResponse'> {
    snapshot?: StripeConfigSnapshot;
    settings?: StripeSettings;
  }

  export const GetStripeSettingsRequestSchema: GenMessage<GetStripeSettingsRequest>;
  export const GetStripeSettingsResponseSchema: GenMessage<GetStripeSettingsResponse>;
  export const UpdateStripeSettingsRequestSchema: GenMessage<UpdateStripeSettingsRequest>;
  export const UpdateStripeSettingsResponseSchema: GenMessage<UpdateStripeSettingsResponse>;
  export const RevealStripeSecretRequestSchema: GenMessage<RevealStripeSecretRequest>;
  export const RevealStripeSecretResponseSchema: GenMessage<RevealStripeSecretResponse>;
  export const StripeSettingsService: GenService<{
    getStripeSettings: { methodKind: 'unary'; input: typeof GetStripeSettingsRequestSchema; output: typeof GetStripeSettingsResponseSchema };
    updateStripeSettings: { methodKind: 'unary'; input: typeof UpdateStripeSettingsRequestSchema; output: typeof UpdateStripeSettingsResponseSchema };
    revealStripeSecret: { methodKind: 'unary'; input: typeof RevealStripeSecretRequestSchema; output: typeof RevealStripeSecretResponseSchema };
  }>;
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

declare module '@vrooli/proto-types/landing-page-business-suite/variant_space_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';

  export type GetVariantSpaceRequest = Message<'landing_page_business_suite.v1.GetVariantSpaceRequest'>;
  export interface GetVariantSpaceResponse extends Message<'landing_page_business_suite.v1.GetVariantSpaceResponse'> {
    rawJson: Uint8Array;
  }
  export const GetVariantSpaceRequestSchema: GenMessage<GetVariantSpaceRequest>;
  export const GetVariantSpaceResponseSchema: GenMessage<GetVariantSpaceResponse>;
  export const VariantSpaceService: GenService<{
    getVariantSpace: {
      methodKind: 'unary';
      input: typeof GetVariantSpaceRequestSchema;
      output: typeof GetVariantSpaceResponseSchema;
    };
  }>;
}

declare module '@vrooli/proto-types/landing-page-business-suite/variant_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';

  export interface Variant extends Message<'landing_page_business_suite.v1.Variant'> {
    id: bigint; slug: string; name: string; description: string; weight: number; status: string;
    axes: Record<string, string>; headerConfig?: Record<string, unknown>; seoConfig?: Record<string, unknown>;
  }
  export interface ContentSection extends Message<'vrooli.landing_page_business_suite.v1.shared.ContentSection'> {
    id: bigint; variantId: bigint; sectionType: string; content?: { fields?: Record<string, unknown> }; order: number; enabled: boolean;
  }
  export interface VariantSnapshot extends Message<'landing_page_business_suite.v1.VariantSnapshot'> {
    slug: string; name: string; description: string; weight: number; status: string; axes: Record<string, string>;
    headerConfig?: Record<string, unknown>; seoConfig?: Record<string, unknown>; sections: ContentSection[];
  }
  export interface GetPublicVariantRequest extends Message<'landing_page_business_suite.v1.GetPublicVariantRequest'> { slug: string; }
  export interface GetVariantRequest extends Message<'landing_page_business_suite.v1.GetVariantRequest'> { slug: string; }
  export interface ListVariantsRequest extends Message<'landing_page_business_suite.v1.ListVariantsRequest'> { statusFilter: string; }
  export interface CreateVariantRequest extends Message<'landing_page_business_suite.v1.CreateVariantRequest'> { slug: string; name: string; description: string; weight: number; axes: Record<string, string>; }
  export interface AxesSelection extends Message<'landing_page_business_suite.v1.AxesSelection'> { values: Record<string, string>; }
  export interface UpdateVariantRequest extends Message<'landing_page_business_suite.v1.UpdateVariantRequest'> { slug: string; name?: string; description?: string; weight?: number; axes?: AxesSelection; headerConfig?: Record<string, unknown>; }
  export interface ArchiveVariantRequest extends Message<'landing_page_business_suite.v1.ArchiveVariantRequest'> { slug: string; }
  export interface DeleteVariantRequest extends Message<'landing_page_business_suite.v1.DeleteVariantRequest'> { slug: string; }
  export interface ExportVariantSnapshotRequest extends Message<'landing_page_business_suite.v1.ExportVariantSnapshotRequest'> { slug: string; }
  export interface ImportVariantSnapshotRequest extends Message<'landing_page_business_suite.v1.ImportVariantSnapshotRequest'> { slug: string; snapshot?: VariantSnapshot; }
  export interface VariantResponse extends Message<'landing_page_business_suite.v1.VariantResponse'> { variant?: Variant; }
  export interface ListVariantsResponse extends Message<'landing_page_business_suite.v1.ListVariantsResponse'> { variants: Variant[]; }
  export interface DeleteVariantResponse extends Message<'landing_page_business_suite.v1.DeleteVariantResponse'> { deleted: boolean; }
  export interface ExportVariantSnapshotResponse extends Message<'landing_page_business_suite.v1.ExportVariantSnapshotResponse'> { snapshot?: VariantSnapshot; }
  export interface ImportVariantSnapshotResponse extends Message<'landing_page_business_suite.v1.ImportVariantSnapshotResponse'> { snapshot?: VariantSnapshot; }
  export const VariantSchema: GenMessage<Variant>;
  export const VariantSnapshotSchema: GenMessage<VariantSnapshot>;
  export const VariantService: GenService<{
    selectVariant: { methodKind: 'unary'; input: GenMessage<SelectVariantRequest>; output: GenMessage<VariantResponse> };
    getPublicVariant: { methodKind: 'unary'; input: GenMessage<GetPublicVariantRequest>; output: GenMessage<VariantResponse> };
    getVariant: { methodKind: 'unary'; input: GenMessage<GetVariantRequest>; output: GenMessage<VariantResponse> };
    listVariants: { methodKind: 'unary'; input: GenMessage<ListVariantsRequest>; output: GenMessage<ListVariantsResponse> };
    createVariant: { methodKind: 'unary'; input: GenMessage<CreateVariantRequest>; output: GenMessage<VariantResponse> };
    updateVariant: { methodKind: 'unary'; input: GenMessage<UpdateVariantRequest>; output: GenMessage<VariantResponse> };
    archiveVariant: { methodKind: 'unary'; input: GenMessage<ArchiveVariantRequest>; output: GenMessage<VariantResponse> };
    deleteVariant: { methodKind: 'unary'; input: GenMessage<DeleteVariantRequest>; output: GenMessage<DeleteVariantResponse> };
    exportVariantSnapshot: { methodKind: 'unary'; input: GenMessage<ExportVariantSnapshotRequest>; output: GenMessage<ExportVariantSnapshotResponse> };
    importVariantSnapshot: { methodKind: 'unary'; input: GenMessage<ImportVariantSnapshotRequest>; output: GenMessage<ImportVariantSnapshotResponse> };
  }>;
}

declare module '@vrooli/proto-types/landing-page-business-suite/seo_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';
  export interface VariantSEOConfig { title?: string; description?: string; ogTitle?: string; ogDescription?: string; ogImageUrl?: string; twitterCard?: string; canonicalPath?: string; noindex?: boolean; structuredData?: Record<string, unknown>; }
  export interface GetVariantSEORequest extends Message<'landing_page_business_suite.v1.GetVariantSEORequest'> { slug: string; }
  export interface UpdateVariantSEORequest extends Message<'landing_page_business_suite.v1.UpdateVariantSEORequest'> { slug: string; config?: VariantSEOConfig; }
  export interface SEOResponse extends Message<'landing_page_business_suite.v1.SEOResponse'> {
    siteName: string; title: string; description: string; ogTitle: string; ogDescription: string;
    ogImageUrl: string; twitterCard: string; canonicalUrl: string; faviconUrl: string;
    appleTouchIconUrl: string; themePrimaryColor: string; noindex: boolean;
    structuredData?: Record<string, unknown>;
  }
  export interface UpdateVariantSEOResponse extends Message<'landing_page_business_suite.v1.UpdateVariantSEOResponse'> { success: boolean; updatedAt: string; }
  export const GetVariantSEORequestSchema: GenMessage<GetVariantSEORequest>;
  export const SEOResponseSchema: GenMessage<SEOResponse>;
  export const UpdateVariantSEORequestSchema: GenMessage<UpdateVariantSEORequest>;
  export const UpdateVariantSEOResponseSchema: GenMessage<UpdateVariantSEOResponse>;
  export const SeoService: GenService<{
    getVariantSEO: { methodKind: 'unary'; input: typeof GetVariantSEORequestSchema; output: typeof SEOResponseSchema };
    updateVariantSEO: { methodKind: 'unary'; input: typeof UpdateVariantSEORequestSchema; output: typeof UpdateVariantSEOResponseSchema };
  }>;
}

declare module '@vrooli/proto-types/landing-page-business-suite/config_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';
  export interface GetLandingConfigRequest extends Message<'landing_page_business_suite.v1.GetLandingConfigRequest'> { variantSlug: string; }
  export type LandingConfigResponse = Message<'landing_page_business_suite.v1.LandingConfigResponse'>;
  export const GetLandingConfigRequestSchema: GenMessage<GetLandingConfigRequest>;
  export const LandingConfigResponseSchema: GenMessage<LandingConfigResponse>;
  export const LandingConfigService: GenService<{
    getLandingConfig: { methodKind: 'unary'; input: typeof GetLandingConfigRequestSchema; output: typeof LandingConfigResponseSchema };
  }>;
}

declare module '@vrooli/proto-types/landing-page-business-suite/account_pb' {
  import type { Message } from '@bufbuild/protobuf';
  import type { GenMessage, GenService } from '@bufbuild/protobuf/codegenv2';

  export type GetMySubscriptionRequest = Message<'landing_page_business_suite.v1.GetMySubscriptionRequest'>;
  export type GetMyCreditsRequest = Message<'landing_page_business_suite.v1.GetMyCreditsRequest'>;
  export type GetEntitlementsRequest = Message<'landing_page_business_suite.v1.GetEntitlementsRequest'>;
  export interface SubscriptionStatus { state?: number; subscriptionId?: string; userIdentity?: string; planTier?: string; stripePriceId?: string; bundleKey?: string; cachedAt?: { toJsonString?: () => string }; }
  export interface CreditsBalance { customerEmail?: string; balanceCredits?: number; bundleKey?: string; }
  export interface GetMySubscriptionResponse extends Message<'vrooli.landing_page_business_suite.v1.shared.VerifySubscriptionResponse'> { status?: SubscriptionStatus; }
  export interface GetMyCreditsResponse extends Message<'landing_page_business_suite.v1.GetMyCreditsResponse'> { balance?: CreditsBalance; displayCreditsLabel?: string; displayCreditsMultiplier?: number; }
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
