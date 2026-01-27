import { z } from 'zod';
import {
  AxisVariantStatusSchema,
  FlexibleTimestampSchema,
  TrendSchema,
  VariantStatusSchema,
} from './common.schema';
import {
  LandingHeaderConfigSchema,
  LandingSectionSchema,
  VariantAxesSchema,
} from './landing.schema';

/**
 * Variant-related Zod schemas for API response validation.
 */

// Variant space axis variant schema
export const VariantSpaceAxisVariantSchema = z.object({
  id: z.string(),
  label: z.string(),
  description: z.string().optional(),
  examples: z.record(z.string(), z.string()).optional(),
  defaultWeight: z.number().optional(),
  tags: z.array(z.string()).optional(),
  status: AxisVariantStatusSchema,
  agentHints: z.array(z.string()).optional(),
});

// Variant space axis schema
export const VariantSpaceAxisSchema = z.object({
  _note: z.string().optional(),
  variants: z.array(VariantSpaceAxisVariantSchema),
});

// Variant space constraints schema
export const VariantSpaceConstraintsSchema = z.object({
  _note: z.string().optional(),
  disallowedCombinations: z.array(z.record(z.string(), z.string())).optional(),
});

// Variant space schema
export const VariantSpaceSchema = z.object({
  _name: z.string(),
  _schemaVersion: z.number(),
  _note: z.string().optional(),
  _agentGuidelines: z.array(z.string()).optional(),
  axes: z.record(z.string(), VariantSpaceAxisSchema),
  constraints: VariantSpaceConstraintsSchema.optional(),
});

// Full variant schema
export const VariantSchema = z.object({
  id: z.number().optional(),
  slug: z.string(),
  name: z.string(),
  description: z.string().optional(),
  weight: z.number().optional(),
  status: VariantStatusSchema,
  created_at: FlexibleTimestampSchema,
  updated_at: FlexibleTimestampSchema,
  archived_at: FlexibleTimestampSchema,
  axes: VariantAxesSchema.optional(),
  header_config: LandingHeaderConfigSchema.optional(),
});

// Variant list response schema
export const VariantListResponseSchema = z.object({
  variants: z.array(VariantSchema),
});

// Variant snapshot meta schema
export const VariantSnapshotMetaSchema = z.object({
  slug: z.string(),
  name: z.string(),
  description: z.string().optional(),
  axes: VariantAxesSchema,
  header_config: LandingHeaderConfigSchema.optional(),
  seo_config: z.record(z.string(), z.unknown()).optional(),
});

// Variant snapshot metadata schema
export const VariantSnapshotMetadataSchema = z.object({
  mode: z.enum(['content-only', 'full']).optional(),
  updated_at: FlexibleTimestampSchema,
});

// Variant snapshot schema
export const VariantSnapshotSchema = z.object({
  variant: VariantSnapshotMetaSchema,
  sections: z.array(LandingSectionSchema),
  _metadata: VariantSnapshotMetadataSchema.optional(),
});

// Variant stats schema
export const VariantStatsSchema = z.object({
  variant_id: z.number(),
  variant_slug: z.string(),
  variant_name: z.string(),
  views: z.number(),
  cta_clicks: z.number(),
  conversions: z.number(),
  downloads: z.number(),
  conversion_rate: z.number(),
  avg_scroll_depth: z.number().optional(),
  trend: TrendSchema,
});

// Analytics summary schema
export const AnalyticsSummarySchema = z.object({
  total_visitors: z.number(),
  total_downloads: z.number().optional(),
  variant_stats: z.array(VariantStatsSchema),
  top_cta: z.string().optional(),
  top_cta_ctr: z.number().optional(),
});

// Variant SEO config schema (input)
export const VariantSEOConfigSchema = z.object({
  title: z.string().optional(),
  description: z.string().optional(),
  og_title: z.string().optional(),
  og_description: z.string().optional(),
  og_image_url: z.string().optional(),
  twitter_card: z.enum(['summary', 'summary_large_image']).optional(),
  canonical_path: z.string().optional(),
  noindex: z.boolean().optional(),
  structured_data: z.record(z.string(), z.unknown()).optional(),
});

// Variant SEO response schema
export const VariantSEOResponseSchema = z.object({
  site_name: z.string(),
  title: z.string(),
  description: z.string(),
  og_title: z.string(),
  og_description: z.string(),
  og_image_url: z.string().optional(),
  twitter_card: z.union([z.enum(['summary', 'summary_large_image']), z.string()]).optional(),
  canonical_url: z.string().optional(),
  favicon_url: z.string().optional(),
  apple_touch_icon_url: z.string().optional(),
  theme_primary_color: z.string().optional(),
  noindex: z.boolean(),
  structured_data: z.record(z.string(), z.unknown()).optional(),
});

// Export inferred types
export type VariantSpaceAxisVariant = z.infer<typeof VariantSpaceAxisVariantSchema>;
export type VariantSpaceAxis = z.infer<typeof VariantSpaceAxisSchema>;
export type VariantSpaceConstraints = z.infer<typeof VariantSpaceConstraintsSchema>;
export type VariantSpace = z.infer<typeof VariantSpaceSchema>;
export type Variant = z.infer<typeof VariantSchema>;
export type VariantListResponse = z.infer<typeof VariantListResponseSchema>;
export type VariantSnapshotMeta = z.infer<typeof VariantSnapshotMetaSchema>;
export type VariantSnapshotMetadata = z.infer<typeof VariantSnapshotMetadataSchema>;
export type VariantSnapshot = z.infer<typeof VariantSnapshotSchema>;
export type VariantStats = z.infer<typeof VariantStatsSchema>;
export type AnalyticsSummary = z.infer<typeof AnalyticsSummarySchema>;
export type VariantSEOConfig = z.infer<typeof VariantSEOConfigSchema>;
export type VariantSEOResponse = z.infer<typeof VariantSEOResponseSchema>;
