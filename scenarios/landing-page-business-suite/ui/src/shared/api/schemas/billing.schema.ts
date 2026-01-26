import { z } from 'zod';
import { ConfigSourceSchema, FlexibleTimestampSchema, MetadataSchema } from './common.schema';
import { BundleProductSchema, PlanOptionSchema } from './landing.schema';

/**
 * Billing-related Zod schemas for API response validation.
 */

const NonEmptyStringSchema = z.string().min(1);
const CurrencyCodeSchema = z.string().regex(/^[a-z]{3}$/i, 'Invalid currency code');

// Stripe settings response schema
export const StripeSettingsResponseSchema = z.object({
  publishable_key_preview: z.string().optional(),
  publishable_key_set: z.boolean(),
  secret_key_set: z.boolean(),
  webhook_secret_set: z.boolean(),
  dashboard_url: z.string().url().optional(),
  updated_at: FlexibleTimestampSchema,
  source: z.union([ConfigSourceSchema, z.string()]),
});

// Checkout session schema
export const CheckoutSessionSchema = z.object({
  session_id: NonEmptyStringSchema,
  session_kind: z.string().optional(),
  status: z.string().optional(),
  url: z.string().url(),
  publishable_key: z.string().optional(),
  customer_email: z.string().optional(),
  stripe_price_id: z.string().optional(),
  amount_cents: z.number().int().nonnegative().optional(),
  currency: CurrencyCodeSchema.optional(),
  success_url: z.string().optional(),
  cancel_url: z.string().optional(),
});

// Billing portal response schema
export const BillingPortalResponseSchema = z.object({
  url: z.string().url(),
});

// Bundle catalog entry schema
export const BundleCatalogEntrySchema = z.object({
  bundle: BundleProductSchema,
  prices: z.array(PlanOptionSchema),
});

// Bundle catalog response schema
export const BundleCatalogResponseSchema = z.object({
  bundles: z.array(BundleCatalogEntrySchema),
});

// Subscription info schema
export const SubscriptionInfoSchema = z.object({
  status: z.string(),
  subscription_id: z.string().optional(),
  customer_email: z.string().optional(),
  plan_tier: z.string().optional(),
  price_id: z.string().optional(),
  bundle_key: z.string().optional(),
  updated_at: FlexibleTimestampSchema,
});

// Credit info schema
export const CreditInfoSchema = z.object({
  customer_email: z.string(),
  balance_credits: z.number(),
  bonus_credits: z.number(),
  display_credits_label: z.string(),
  display_credits_multiplier: z.number(),
});

// Entitlement payload schema
export const EntitlementPayloadSchema = z.object({
  status: z.string(),
  plan_tier: z.string().optional(),
  price_id: z.string().optional(),
  features: z.array(z.string()).optional(),
  credits: CreditInfoSchema.optional(),
  subscription: SubscriptionInfoSchema.optional(),
});

// Verify stripe price response schema
export const VerifyStripePriceResponseSchema = z.object({
  id: NonEmptyStringSchema,
  lookup_key: z.string().optional(),
  currency: CurrencyCodeSchema.optional(),
  amount_cents: z.number().int().nonnegative().optional(),
  interval: z.string().min(1).optional(),
  active: z.boolean().optional(),
  product: z.string().optional(),
});

// Stripe import schemas
export const StripePriceImportSchema = z.object({
  price_id: NonEmptyStringSchema,
  lookup_key: z.string().min(1).optional(),
  currency: CurrencyCodeSchema,
  amount_cents: z.number().int().nonnegative(),
  interval: z.string().min(1).optional(),
  product_id: NonEmptyStringSchema,
  product_name: NonEmptyStringSchema,
  active: z.boolean(),
  exists_locally: z.boolean(),
});

export const StripeProductWithPricesSchema = z.object({
  product_id: NonEmptyStringSchema,
  product_name: NonEmptyStringSchema,
  prices: z.array(StripePriceImportSchema),
});

export const StripeImportPreviewSchema = z.object({
  products: z.array(StripeProductWithPricesSchema),
  total_prices: z.number().int().nonnegative(),
  conflict_count: z.number().int().nonnegative(),
  new_count: z.number().int().nonnegative(),
});

export const StripeImportResultSchema = z.object({
  imported: z.number().int().nonnegative(),
  overwritten: z.number().int().nonnegative(),
  skipped: z.number().int().nonnegative(),
  errors: z.array(z.string()).optional(),
});

// Checkout session wrapper response
export const CheckoutSessionResponseSchema = z.object({
  session: CheckoutSessionSchema,
});

// Export inferred types
export type StripeSettingsResponse = z.infer<typeof StripeSettingsResponseSchema>;
export type CheckoutSession = z.infer<typeof CheckoutSessionSchema>;
export type BillingPortalResponse = z.infer<typeof BillingPortalResponseSchema>;
export type BundleCatalogEntry = z.infer<typeof BundleCatalogEntrySchema>;
export type BundleCatalogResponse = z.infer<typeof BundleCatalogResponseSchema>;
export type SubscriptionInfo = z.infer<typeof SubscriptionInfoSchema>;
export type CreditInfo = z.infer<typeof CreditInfoSchema>;
export type EntitlementPayload = z.infer<typeof EntitlementPayloadSchema>;
export type VerifyStripePriceResponse = z.infer<typeof VerifyStripePriceResponseSchema>;
export type StripePriceImport = z.infer<typeof StripePriceImportSchema>;
export type StripeProductWithPrices = z.infer<typeof StripeProductWithPricesSchema>;
export type StripeImportPreview = z.infer<typeof StripeImportPreviewSchema>;
export type StripeImportResult = z.infer<typeof StripeImportResultSchema>;
