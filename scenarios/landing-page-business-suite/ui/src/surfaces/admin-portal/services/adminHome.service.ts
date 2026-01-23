import type { AnalyticsSummary, DownloadApp, SiteBranding, Variant, VariantStats } from '../../../shared/api';
import {
  calculateDaysSince,
  formatRelativeTime,
  DAY_MS,
} from '../../../shared/lib/dateFormatters';

/**
 * Shared constants for variant staleness calculations
 */
export const STALE_VARIANT_DAYS = 10;

// Re-export for backward compatibility
export { DAY_MS };

/**
 * Number of days used for the health snapshot analytics window
 */
export const HEALTH_SNAPSHOT_DAYS = 7;

/**
 * Weight allocation status
 */
export type WeightStatus = 'empty' | 'balanced' | 'under' | 'over';

/**
 * Represents a variant that needs attention
 */
export interface VariantAttention {
  slug: string;
  name: string;
  reasons: string[];
  conversionRate?: number;
  daysSinceUpdate?: number | null;
  updatedLabel: string;
  sectionId?: number;
  sectionType?: string;
}

/**
 * Health snapshot for admin dashboard
 */
export interface HealthSnapshot {
  activeCount: number;
  archivedCount: number;
  attentionCount: number;
  totalWeight: number;
  weightStatus: WeightStatus;
  highlightedAttention?: VariantAttention;
}

/**
 * Branding health status
 */
export interface BrandingHealthStatus {
  hasIdentity: boolean;
  hasFavicon: boolean;
  hasSeo: boolean;
  hasOgImage: boolean;
  configuredCount: number;
  totalChecks: number;
}

/**
 * Downloads health status
 */
export interface DownloadsHealthStatus {
  appCount: number;
  platformsConfigured: number;
  storefrontsConfigured: number;
  hasApps: boolean;
}


/**
 * Format variant updated label for display.
 * Uses formatRelativeTime but handles invalid dates specially.
 */
function formatUpdatedLabel(updatedAt?: string | null): string {
  if (!updatedAt) {
    return 'Never customized';
  }
  const parsed = new Date(updatedAt);
  if (Number.isNaN(parsed.getTime())) {
    return 'Last updated date unavailable';
  }
  return formatRelativeTime(updatedAt, {
    nullLabel: 'Never customized',
    prefix: 'Updated ',
  });
}

/**
 * Get the attention priority for a variant
 *
 * Higher priority = more urgent attention needed
 * - 3: Lowest conversion (most urgent)
 * - 2: Stale variant
 * - 1: Never customized
 * - 0: Other
 *
 * @param entry - Variant attention entry
 * @returns Priority score (0-3)
 */
export function getAttentionPriority(entry: VariantAttention): number {
  if (entry.reasons.some((reason) => reason.toLowerCase().includes('lowest conversion'))) {
    return 3;
  }
  if (entry.reasons.some((reason) => reason.startsWith('Stale'))) {
    return 2;
  }
  if (entry.reasons.some((reason) => reason.startsWith('Never'))) {
    return 1;
  }
  return 0;
}

/**
 * Describe the weight allocation status in human-readable terms
 *
 * @param status - Weight status
 * @param totalWeight - Total weight percentage assigned
 * @param activeCount - Number of active variants
 * @returns Description of the weight status
 */
export function describeWeightStatus(
  status: WeightStatus,
  totalWeight: number,
  activeCount: number
): string {
  if (activeCount === 0) {
    return 'No active variants are routing traffic. Create one to render the public landing.';
  }
  if (status === 'balanced') {
    return 'Traffic is fully allocated across variants.';
  }
  if (status === 'under') {
    return `${Math.max(0, 100 - Math.round(totalWeight))}% of visitors are idle because weights total less than 100%.`;
  }
  if (status === 'over') {
    return `Weights exceed 100% by ${Math.round(totalWeight - 100)}%. Adjust them to match your intent.`;
  }
  return 'Assign weights to control where visitors land.';
}

/**
 * Build a health snapshot from variants and analytics data
 *
 * @param variants - List of all variants
 * @param analytics - Analytics summary (may be null if unavailable)
 * @returns Health snapshot for the admin dashboard
 */
export function buildHealthSnapshot(
  variants: Variant[],
  analytics: AnalyticsSummary | null
): HealthSnapshot {
  const activeVariants = variants.filter((variant) => variant.status !== 'archived');
  const archivedVariants = variants.filter((variant) => variant.status === 'archived');
  const totalWeight = activeVariants.reduce((sum, variant) => sum + (variant.weight ?? 0), 0);

  let weightStatus: WeightStatus = 'empty';
  if (activeVariants.length > 0) {
    if (Math.round(totalWeight) === 100) {
      weightStatus = 'balanced';
    } else if (totalWeight > 100) {
      weightStatus = 'over';
    } else {
      weightStatus = 'under';
    }
  }

  const statsBySlug = new Map<string, VariantStats>();
  analytics?.variant_stats?.forEach((stat) => statsBySlug.set(stat.variant_slug, stat));

  const attentionEntries = new Map<string, VariantAttention>();

  const registerAttention = (
    variant: Variant,
    reason: string,
    extras?: Partial<VariantAttention>
  ) => {
    const existing = attentionEntries.get(variant.slug);
    if (existing) {
      const next: VariantAttention = {
        ...existing,
        ...extras,
        reasons: existing.reasons.includes(reason)
          ? existing.reasons
          : [...existing.reasons, reason],
      };
      attentionEntries.set(variant.slug, next);
      return;
    }

    attentionEntries.set(variant.slug, {
      slug: variant.slug,
      name: variant.name ?? variant.slug,
      reasons: [reason],
      conversionRate: extras?.conversionRate,
      daysSinceUpdate: calculateDaysSince(variant.updated_at),
      updatedLabel: formatUpdatedLabel(variant.updated_at),
      sectionId: extras?.sectionId,
      sectionType: extras?.sectionType ?? 'hero',
    });
  };

  // Check for stale/never-customized variants
  activeVariants.forEach((variant) => {
    const daysSinceUpdate = calculateDaysSince(variant.updated_at);
    if (daysSinceUpdate === null) {
      registerAttention(variant, 'Never customized');
    } else if (daysSinceUpdate >= STALE_VARIANT_DAYS) {
      registerAttention(variant, `Stale · ${daysSinceUpdate}d`, { daysSinceUpdate });
    }
  });

  // Check for underperforming variants
  if (analytics?.variant_stats?.length) {
    const activeSlugs = new Set(activeVariants.map((variant) => variant.slug));
    const relevantStats = analytics.variant_stats.filter((stat) =>
      activeSlugs.has(stat.variant_slug)
    );
    const underperforming = relevantStats.reduce<VariantStats | null>((worst, stat) => {
      if (!worst) {
        return stat;
      }
      return stat.conversion_rate < worst.conversion_rate ? stat : worst;
    }, null);

    if (underperforming) {
      const matchingVariant = activeVariants.find(
        (variant) => variant.slug === underperforming.variant_slug
      );
      if (matchingVariant) {
        registerAttention(matchingVariant, 'Lowest conversion', {
          conversionRate: underperforming.conversion_rate,
        });
      }
    }
  }

  // Find the highest priority attention item
  const attentionList = Array.from(attentionEntries.values());
  let highlightedAttention: VariantAttention | undefined;
  let highestPriority = -1;

  attentionList.forEach((entry) => {
    const priority = getAttentionPriority(entry);
    if (
      !highlightedAttention ||
      priority > highestPriority ||
      (priority === highestPriority &&
        (entry.daysSinceUpdate ?? -1) > (highlightedAttention.daysSinceUpdate ?? -1))
    ) {
      highestPriority = priority;
      highlightedAttention = entry;
    }
  });

  return {
    activeCount: activeVariants.length,
    archivedCount: archivedVariants.length,
    attentionCount: attentionList.length,
    totalWeight,
    weightStatus,
    highlightedAttention,
  };
}

/**
 * Compute branding health status from site branding
 *
 * @param branding - Site branding configuration
 * @returns Branding health status
 */
export function computeBrandingHealth(branding: SiteBranding): BrandingHealthStatus {
  const hasIdentity = Boolean(branding.site_name && branding.logo_url);
  const hasFavicon = Boolean(branding.favicon_url);
  const hasSeo = Boolean(branding.default_title && branding.default_description);
  const hasOgImage = Boolean(branding.default_og_image_url);
  const checks = [hasIdentity, hasFavicon, hasSeo, hasOgImage];

  return {
    hasIdentity,
    hasFavicon,
    hasSeo,
    hasOgImage,
    configuredCount: checks.filter(Boolean).length,
    totalChecks: checks.length,
  };
}

/**
 * Compute downloads health status from download apps
 *
 * @param apps - Array of download apps
 * @returns Downloads health status
 */
export function computeDownloadsHealth(apps: DownloadApp[]): DownloadsHealthStatus {
  let platformsConfigured = 0;
  let storefrontsConfigured = 0;

  apps.forEach((app) => {
    if (app.platforms) {
      platformsConfigured += app.platforms.filter(
        (p) => p.artifact_url && p.release_version
      ).length;
    }
    if (app.storefronts) {
      storefrontsConfigured += app.storefronts.filter((s) => s.url).length;
    }
  });

  return {
    appCount: apps.length,
    platformsConfigured,
    storefrontsConfigured,
    hasApps: apps.length > 0,
  };
}
