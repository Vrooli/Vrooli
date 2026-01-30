import { z } from 'zod';
import {
  BillingIntervalSchema,
  HeaderBrandingModeSchema,
  HeaderCTAModeSchema,
  HeaderNavLinkTypeSchema,
  HeaderVisibilityConfigSchema,
  HeaderBehaviorConfigSchema,
  IntroPricingTypeSchema,
  MetadataSchema,
  PlanKindSchema,
  StripeCouponSchema,
} from './common.schema';

/**
 * Landing page Zod schemas for API response validation.
 */

// Plan display metadata
export const PlanDisplayMetadataSchema = z.object({
  subtitle: z.string().optional(),
  badge: z.string().optional(),
  cta_label: z.string().optional(),
  highlight: z.boolean().optional(),
  features: z.array(z.string()).optional(),
}).passthrough(); // Allow additional properties with [key: string]: unknown

// Plan option schema
export const PlanOptionSchema = z.object({
  plan_name: z.string(),
  plan_tier: z.string(),
  billing_interval: BillingIntervalSchema,
  amount_cents: z.number(),
  currency: z.string(),
  intro_enabled: z.boolean().default(false),
  intro_type: IntroPricingTypeSchema.optional(),
  intro_amount_cents: z.number().optional(),
  intro_periods: z.number().optional(),
  intro_price_lookup_key: z.string().optional(),
  stripe_price_id: z.string(),
  monthly_included_credits: z.number().default(0),
  one_time_bonus_credits: z.number().default(0),
  plan_rank: z.number().optional(),
  bonus_type: z.string().optional(),
  kind: PlanKindSchema.optional(),
  is_variable_amount: z.boolean().optional(),
  display_enabled: z.boolean().default(false),
  bundle_key: z.string().optional(),
  display_weight: z.number().default(0),
  metadata: PlanDisplayMetadataSchema.optional(),
});

// Bundle product schema
export const BundleProductSchema = z.object({
  id: z.number().optional(),
  bundle_key: z.string(),
  name: z.string(),
  stripe_product_id: z.string(),
  credits_per_usd: z.number(),
  display_credits_multiplier: z.number(),
  display_credits_label: z.string(),
  environment: z.string().optional(),
  metadata: MetadataSchema,
});

// Pricing overview schema
export const PricingOverviewSchema = z.object({
  bundle: BundleProductSchema,
  monthly: z.array(PlanOptionSchema),
  yearly: z.array(PlanOptionSchema),
  updated_at: z.string(),
});

// Landing section schema
export const LandingSectionSchema = z.object({
  id: z.number().optional(),
  section_type: z.string(),
  content: z.record(z.string(), z.unknown()),
  order: z.number(),
  enabled: z.boolean().optional(),
});

// Header nav link base type (for recursive reference)
interface HeaderNavLinkBase {
  id: string;
  type: 'section' | 'downloads' | 'custom' | 'menu';
  label: string;
  section_type?: string;
  section_id?: number;
  anchor?: string;
  href?: string;
  visible_on?: { desktop?: boolean; mobile?: boolean };
  children?: HeaderNavLinkBase[];
}

// Header nav link schema (recursive for children)
export const HeaderNavLinkSchema: z.ZodType<HeaderNavLinkBase> = z.lazy(() =>
  z.object({
    id: z.string(),
    type: HeaderNavLinkTypeSchema,
    label: z.string(),
    section_type: z.string().optional(),
    section_id: z.number().optional(),
    anchor: z.string().optional(),
    href: z.string().optional(),
    visible_on: HeaderVisibilityConfigSchema.optional(),
    children: z.array(HeaderNavLinkSchema).optional(),
  })
);

// Header branding config schema
export const HeaderBrandingConfigSchema = z.object({
  mode: HeaderBrandingModeSchema,
  label: z.string().optional(),
  subtitle: z.string().optional(),
  mobile_preference: z.enum(['auto', 'logo', 'name', 'stacked']).optional(),
  logo_url: z.string().nullable().optional(),
  logo_icon_url: z.string().nullable().optional(),
});

// Header nav config schema
export const HeaderNavConfigSchema = z.object({
  links: z.array(HeaderNavLinkSchema),
});

// Header CTA config schema
export const HeaderCTAConfigSchema = z.object({
  mode: HeaderCTAModeSchema,
  label: z.string().optional(),
  href: z.string().optional(),
  variant: z.enum(['solid', 'ghost']).optional(),
});

// Header CTA group schema
export const HeaderCTAGroupSchema = z.object({
  primary: HeaderCTAConfigSchema,
  secondary: HeaderCTAConfigSchema,
});

// Landing header config schema
export const LandingHeaderConfigSchema = z.object({
  branding: HeaderBrandingConfigSchema,
  nav: HeaderNavConfigSchema,
  ctas: HeaderCTAGroupSchema,
  behavior: HeaderBehaviorConfigSchema,
});

// Landing branding schema (public branding info)
export const LandingBrandingSchema = z.object({
  site_name: z.string(),
  tagline: z.string().nullable().optional(),
  logo_url: z.string().nullable().optional(),
  logo_icon_url: z.string().nullable().optional(),
  favicon_url: z.string().nullable().optional(),
  theme_primary_color: z.string().nullable().optional(),
  theme_background_color: z.string().nullable().optional(),
  support_chat_url: z.string().nullable().optional(),
  coming_soon_enabled: z.boolean().nullable().optional(),
  coming_soon_message: z.string().nullable().optional(),
});

// Download storefront schema
export const DownloadStorefrontSchema = z.object({
  store: z.string(),
  label: z.string(),
  url: z.string(),
  badge: z.string().optional(),
});

// Download asset schema
export const DownloadAssetSchema = z.object({
  id: z.number().optional(),
  bundle_key: z.string(),
  app_key: z.string(),
  platform: z.string(),
  artifact_url: z.string(),
  artifact_source: z.enum(['direct', 'managed']).optional(),
  artifact_id: z.number().optional(),
  release_version: z.string(),
  release_notes: z.string().optional(),
  checksum: z.string().optional(),
  requires_entitlement: z.boolean(),
  metadata: MetadataSchema,
  // Artifact info (populated when artifact_source is 'managed')
  artifact_filename: z.string().optional(),
  artifact_size_bytes: z.number().optional(),
  artifact_count: z.number().optional(),
});

// Download app schema
export const DownloadAppSchema = z.object({
  bundle_key: z.string(),
  app_key: z.string(),
  name: z.string(),
  tagline: z.string().optional(),
  description: z.string().optional(),
  icon_url: z.string().optional(),
  screenshot_url: z.string().optional(),
  install_overview: z.string().optional(),
  install_steps: z.array(z.string()).optional(),
  storefronts: z.array(DownloadStorefrontSchema).optional(),
  metadata: MetadataSchema,
  display_order: z.number().optional(),
  platforms: z.array(DownloadAssetSchema),
});

// Variant axes schema
export const VariantAxesSchema = z.record(z.string(), z.string());

// Variant schema (subset for landing config)
export const LandingVariantSchema = z.object({
  id: z.number().optional(),
  slug: z.string(),
  name: z.string(),
  description: z.string().optional(),
  axes: VariantAxesSchema.optional(),
});

// Landing config response schema
export const LandingConfigResponseSchema = z.object({
  variant: LandingVariantSchema,
  sections: z.array(LandingSectionSchema),
  pricing: PricingOverviewSchema.optional(),
  downloads: z.array(DownloadAppSchema),
  header: LandingHeaderConfigSchema,
  branding: LandingBrandingSchema.optional(),
  coupon_mappings: z.record(z.string(), z.string()).optional(),
  intro_offers: z.array(StripeCouponSchema).optional(),
  fallback: z.boolean(),
});

// Export inferred types
export type PlanDisplayMetadata = z.infer<typeof PlanDisplayMetadataSchema>;
export type PlanOption = z.infer<typeof PlanOptionSchema>;
export type BundleProduct = z.infer<typeof BundleProductSchema>;
export type PricingOverview = z.infer<typeof PricingOverviewSchema>;
export type LandingSection = z.infer<typeof LandingSectionSchema>;
export type LandingHeaderNavLink = z.infer<typeof HeaderNavLinkSchema>;
export type HeaderBrandingConfig = z.infer<typeof HeaderBrandingConfigSchema>;
export type HeaderNavConfig = z.infer<typeof HeaderNavConfigSchema>;
export type HeaderCTAConfig = z.infer<typeof HeaderCTAConfigSchema>;
export type HeaderCTAGroup = z.infer<typeof HeaderCTAGroupSchema>;
export type LandingHeaderConfig = z.infer<typeof LandingHeaderConfigSchema>;
export type LandingBranding = z.infer<typeof LandingBrandingSchema>;
export type DownloadStorefront = z.infer<typeof DownloadStorefrontSchema>;
export type DownloadAsset = z.infer<typeof DownloadAssetSchema>;
export type DownloadApp = z.infer<typeof DownloadAppSchema>;
export type VariantAxes = z.infer<typeof VariantAxesSchema>;
export type LandingVariant = z.infer<typeof LandingVariantSchema>;
export type LandingConfigResponse = z.infer<typeof LandingConfigResponseSchema>;
