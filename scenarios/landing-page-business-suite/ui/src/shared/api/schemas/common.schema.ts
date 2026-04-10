import { z } from 'zod';

/**
 * Common Zod schemas for shared types used across API responses.
 * These provide runtime validation for API data.
 */

// Timestamp schema - accepts ISO 8601 strings
export const TimestampSchema = z.string().datetime({ offset: true }).optional();

// Flexible timestamp that accepts any string (for API responses that may not be ISO 8601)
export const FlexibleTimestampSchema = z.string().optional();

// Metadata schema - Record<string, unknown> with specific handling
export const MetadataSchema = z.record(z.string(), z.unknown()).optional();

// Status enum for variants
export const VariantStatusSchema = z.enum(['active', 'archived', 'deleted', 'fallback']).optional();

// Status enum for variant space axis variants
export const AxisVariantStatusSchema = z.enum(['active', 'experimental', 'deprecated']).optional();

// Billing interval enum
export const BillingIntervalSchema = z.enum(['month', 'year', 'one_time']);

// Twitter card type enum
export const TwitterCardSchema = z.enum(['summary', 'summary_large_image']).optional();

// Trend enum for analytics
export const TrendSchema = z.enum(['up', 'down', 'stable']).optional();

// Asset category enum
export const AssetCategorySchema = z.enum(['logo', 'favicon', 'og_image', 'general']);

// Asset schema
export const AssetSchema = z.object({
  id: z.number(),
  filename: z.string(),
  original_filename: z.string(),
  mime_type: z.string(),
  size_bytes: z.number(),
  storage_path: z.string(),
  thumbnail_path: z.string().nullable().optional(),
  alt_text: z.string().nullable().optional(),
  category: z.string(),
  uploaded_by: z.string().nullable().optional(),
  created_at: z.string(),
  url: z.string(),
  derivatives: z.record(z.string()).optional(),
});

export const AssetListSchema = z.object({
  assets: z.array(AssetSchema).optional(),
});

// Header branding mode enum
export const HeaderBrandingModeSchema = z.enum(['none', 'logo', 'name', 'logo_and_name']);

// Header nav link type enum
export const HeaderNavLinkTypeSchema = z.enum(['section', 'downloads', 'custom', 'menu']);

// Header CTA mode enum
export const HeaderCTAModeSchema = z.enum(['inherit_hero', 'downloads', 'custom', 'hidden']);

// Section type enum
export const SectionTypeSchema = z.enum([
  'hero',
  'features',
  'pricing',
  'cta',
  'testimonials',
  'faq',
  'footer',
  'video',
  'downloads',
]);

// Intro pricing type enum
export const IntroPricingTypeSchema = z.enum(['flat_amount', 'percentage']).optional();

// Plan kind enum
export const PlanKindSchema = z.enum(['subscription', 'credits_topup', 'supporter_contribution']).optional();

// Artifact source enum
export const ArtifactSourceSchema = z.enum(['direct', 'managed']).optional();

// Storage provider enum
export const StorageProviderSchema = z.literal('s3');

// Config source enum
export const ConfigSourceSchema = z.enum(['env', 'database']);

// Header visibility config
export const HeaderVisibilityConfigSchema = z.object({
  desktop: z.boolean().optional(),
  mobile: z.boolean().optional(),
});

// Header behavior config
export const HeaderBehaviorConfigSchema = z.object({
  sticky: z.boolean(),
  hide_on_scroll: z.boolean(),
});

// Generic success response
export const SuccessResponseSchema = z.object({
  success: z.boolean().optional(),
  updated_at: FlexibleTimestampSchema,
});

export type Timestamp = z.infer<typeof TimestampSchema>;
export type Metadata = z.infer<typeof MetadataSchema>;
export type VariantStatus = z.infer<typeof VariantStatusSchema>;
export type BillingInterval = z.infer<typeof BillingIntervalSchema>;
export type HeaderBrandingMode = z.infer<typeof HeaderBrandingModeSchema>;
export type HeaderNavLinkType = z.infer<typeof HeaderNavLinkTypeSchema>;
export type HeaderCTAMode = z.infer<typeof HeaderCTAModeSchema>;
export type SectionType = z.infer<typeof SectionTypeSchema>;
export type HeaderVisibilityConfig = z.infer<typeof HeaderVisibilityConfigSchema>;
export type HeaderBehaviorConfig = z.infer<typeof HeaderBehaviorConfigSchema>;

// Stripe coupon schema - placed here to avoid circular dependency between billing.schema and landing.schema
const NonEmptyStringSchema = z.string().min(1);
export const StripeCouponSchema = z.object({
  id: NonEmptyStringSchema,
  name: z.string().optional(),
  amount_off: z.number().int().nonnegative().nullable().optional(),
  percent_off: z.number().min(0).max(100).nullable().optional(),
  currency: z.string().optional(),
  duration: z.enum(['once', 'repeating', 'forever']),
  duration_in_months: z.number().int().positive().nullable().optional(),
  max_redemptions: z.number().int().positive().nullable().optional(),
  redeem_by: z.number().int().positive().nullable().optional(),
  times_redeemed: z.number().int().nonnegative(),
  valid: z.boolean(),
  created: z.number().int().positive(),
  is_intro_coupon: z.boolean(),
  intro_tier: z.string().optional(),
});

export type StripeCoupon = z.infer<typeof StripeCouponSchema>;
