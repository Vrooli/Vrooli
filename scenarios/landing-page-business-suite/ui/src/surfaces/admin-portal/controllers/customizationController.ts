import type { Variant, VariantStats, AnalyticsSummary } from '../../../shared/api';
import { listVariants, archiveVariant, deleteVariant, updateVariant } from '../../../shared/api';
import { buildDateRange, fetchAnalyticsSummary } from './analyticsController';
import { STALE_VARIANT_DAYS, DAY_MS, type WeightStatus } from '../services/adminHome.service';

export const SNAPSHOT_DAYS = 7;

export type TrafficShareMode = 'weighted' | 'even';

/**
 * Stale variant entry with days since update
 */
export interface StaleVariantEntry {
  variant: Variant;
  daysSinceUpdate: number;
}

/**
 * Underperforming variant info
 */
export interface UnderperformingInfo {
  stats: VariantStats;
  variant?: Variant;
}

/**
 * Load customization page data
 */
export async function loadCustomizationData(): Promise<{
  variants: Variant[];
  error: string | null;
}> {
  try {
    const data = await listVariants();
    return { variants: data.variants, error: null };
  } catch (err) {
    return {
      variants: [],
      error: err instanceof Error ? err.message : 'Failed to load variants',
    };
  }
}

/**
 * Load analytics snapshot for customization page
 */
export async function loadAnalyticsSnapshot(): Promise<{
  analytics: AnalyticsSummary | null;
  error: string | null;
}> {
  try {
    const range = buildDateRange(SNAPSHOT_DAYS);
    const data = await fetchAnalyticsSummary(range);
    return { analytics: data, error: null };
  } catch (err) {
    return {
      analytics: null,
      error: err instanceof Error ? err.message : 'Metrics not available',
    };
  }
}

/**
 * Archive a variant
 */
export async function handleArchiveVariant(slug: string): Promise<void> {
  await archiveVariant(slug);
}

/**
 * Delete a variant permanently
 */
export async function handleDeleteVariant(slug: string): Promise<void> {
  await deleteVariant(slug);
}

/**
 * Update variant weight
 */
export async function handleUpdateWeight(
  slug: string,
  weight: number
): Promise<void> {
  await updateVariant(slug, { weight });
}

/**
 * Filter active variants
 */
export function filterActiveVariants(variants: Variant[]): Variant[] {
  return variants.filter((v) => v.status === 'active');
}

/**
 * Filter archived variants
 */
export function filterArchivedVariants(variants: Variant[]): Variant[] {
  return variants.filter((v) => v.status === 'archived');
}

/**
 * Get weight for a variant from drafts or variant itself
 */
export function getVariantWeight(
  variant: Variant,
  weightDrafts: Record<string, number>
): number {
  return weightDrafts[variant.slug] ?? variant.weight ?? 0;
}

/**
 * Calculate total assigned weight
 */
export function calculateTotalWeight(
  variants: Variant[],
  weightDrafts: Record<string, number>
): number {
  return variants.reduce(
    (sum, variant) => sum + getVariantWeight(variant, weightDrafts),
    0
  );
}

/**
 * Determine weight status
 */
export function determineWeightStatus(
  variantCount: number,
  totalWeight: number
): WeightStatus {
  if (variantCount === 0) return 'empty';
  if (totalWeight === 100) return 'balanced';
  if (totalWeight > 100) return 'over';
  return 'under';
}

/**
 * Determine traffic share mode
 */
export function determineTrafficShareMode(totalWeight: number): TrafficShareMode {
  return totalWeight === 0 ? 'even' : 'weighted';
}

/**
 * Normalize traffic share percentage
 */
export function normalizeTrafficShare(
  weight: number,
  totalWeight: number,
  variantCount: number,
  mode: TrafficShareMode
): number {
  if (variantCount === 0) return 0;
  if (mode === 'even') {
    return (1 / variantCount) * 100;
  }
  if (totalWeight === 0) return 0;
  return (weight / totalWeight) * 100;
}

/**
 * Find stale variants (not updated in STALE_VARIANT_DAYS)
 */
export function findStaleVariants(
  variants: Variant[],
  limit = 3
): StaleVariantEntry[] {
  const now = Date.now();
  return variants
    .map((variant) => {
      if (!variant.updated_at) return null;
      const updatedAt = new Date(variant.updated_at);
      if (Number.isNaN(updatedAt.getTime())) return null;
      const daysSinceUpdate = Math.floor((now - updatedAt.getTime()) / DAY_MS);
      if (daysSinceUpdate < STALE_VARIANT_DAYS) return null;
      return { variant, daysSinceUpdate };
    })
    .filter((entry): entry is StaleVariantEntry => Boolean(entry))
    .sort((a, b) => b.daysSinceUpdate - a.daysSinceUpdate)
    .slice(0, limit);
}

/**
 * Find variants that have never been updated
 */
export function findNeverUpdatedVariants(variants: Variant[]): Variant[] {
  return variants.filter((variant) => !variant.updated_at);
}

/**
 * Find underperforming variant (lowest conversion rate)
 */
export function findUnderperformingVariant(
  variantStats: VariantStats[] | undefined,
  activeVariants: Variant[]
): UnderperformingInfo | null {
  if (!variantStats?.length || activeVariants.length === 0) {
    return null;
  }

  const activeSlugs = new Set(activeVariants.map((v) => v.slug));
  const relevant = variantStats.filter((stat) => activeSlugs.has(stat.variant_slug));

  if (relevant.length === 0) return null;

  const lowestStat = relevant.reduce<VariantStats | null>((worst, stat) => {
    if (!worst) return stat;
    return stat.conversion_rate < worst.conversion_rate ? stat : worst;
  }, null);

  if (!lowestStat) return null;

  const variant = activeVariants.find((v) => v.slug === lowestStat.variant_slug);
  return { stats: lowestStat, variant };
}

/**
 * Get attention candidate slugs (variants needing attention)
 */
export function getAttentionCandidateSlugs(
  staleVariants: StaleVariantEntry[],
  neverUpdatedVariants: Variant[],
  underperformingSlug?: string
): Set<string> {
  const slugs = new Set<string>();
  staleVariants.forEach(({ variant }) => slugs.add(variant.slug));
  neverUpdatedVariants.forEach((variant) => slugs.add(variant.slug));
  if (underperformingSlug) {
    slugs.add(underperformingSlug);
  }
  return slugs;
}

/**
 * Build attention reasons map for variants
 */
export function buildAttentionReasonsMap(
  staleVariants: StaleVariantEntry[],
  neverUpdatedVariants: Variant[],
  underperformingSlug?: string
): Map<string, string[]> {
  const map = new Map<string, string[]>();

  const pushReason = (slug: string, reason: string) => {
    map.set(slug, [...(map.get(slug) ?? []), reason]);
  };

  staleVariants.forEach(({ variant, daysSinceUpdate }) => {
    pushReason(variant.slug, `Stale · ${daysSinceUpdate}d`);
  });

  neverUpdatedVariants.forEach((variant) => {
    pushReason(variant.slug, 'Never customized');
  });

  if (underperformingSlug) {
    pushReason(underperformingSlug, 'Lowest conversion');
  }

  return map;
}

/**
 * Filter variants by search query and attention flag
 */
export function filterVariantsByQuery(
  variants: Variant[],
  query: string,
  attentionOnly: boolean,
  attentionCandidates: Set<string>
): Variant[] {
  const normalized = query.trim().toLowerCase();
  return variants.filter((variant) => {
    const matchesQuery =
      !normalized ||
      variant.name?.toLowerCase().includes(normalized) ||
      variant.slug.toLowerCase().includes(normalized);
    const matchesAttention = !attentionOnly || attentionCandidates.has(variant.slug);
    return matchesQuery && matchesAttention;
  });
}

/**
 * Build variant stats map by slug
 */
export function buildStatsMap(
  variantStats: VariantStats[] | undefined
): Map<string, VariantStats> {
  const map = new Map<string, VariantStats>();
  (variantStats ?? []).forEach((stat) => map.set(stat.variant_slug, stat));
  return map;
}

/**
 * Get trend glyph type for display
 */
export function getTrendType(
  trend?: VariantStats['trend']
): 'up' | 'down' | 'neutral' {
  if (trend === 'up') return 'up';
  if (trend === 'down') return 'down';
  return 'neutral';
}

// Re-export constants
export { STALE_VARIANT_DAYS, DAY_MS };
