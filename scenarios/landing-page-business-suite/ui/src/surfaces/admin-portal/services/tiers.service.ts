import type { TierLimit, TierLimitUpdate } from '../../../shared/api';
import {
  getAllTierLimits as apiGetAllTierLimits,
  updateTierLimit as apiUpdateTierLimit,
  formatDollars,
  TIER_OPTIONS,
} from '../../../shared/api';

/**
 * Default values for tier limits
 */
export const DEFAULT_TIER_VALUES: Record<string, string> = {
  'free:ai_credits': '0',
  'solo:ai_credits': '5',
  'pro:ai_credits': '20',
  'studio:ai_credits': '100',
  'business:ai_credits': 'unlimited',
};

/**
 * Get edit key for a tier limit
 */
export function getEditKey(tierID: string, limitKey: string): string {
  return `${tierID}:${limitKey}`;
}

/**
 * Get human-readable label for a tier
 */
export function getTierLabel(tierID: string): string {
  const option = TIER_OPTIONS.find((t) => t.value === tierID);
  return option?.label || tierID;
}

/**
 * Get color class for tier
 */
export function getTierColor(tierID: string): string {
  switch (tierID) {
    case 'free':
      return 'text-slate-400';
    case 'solo':
      return 'text-blue-400';
    case 'pro':
      return 'text-purple-400';
    case 'studio':
      return 'text-amber-400';
    case 'business':
      return 'text-emerald-400';
    default:
      return 'text-slate-400';
  }
}

/**
 * Check if a limit value represents unlimited
 */
export function isUnlimitedValue(limitValue: number): boolean {
  return limitValue < 0;
}

/**
 * Parse and validate an edited value
 */
export function parseEditedValue(
  value: string
): { isUnlimited: true } | { displayDollars: number } | null {
  const normalizedValue = value.toLowerCase().trim();

  if (normalizedValue === 'unlimited' || normalizedValue === '-1') {
    return { isUnlimited: true };
  }

  const dollars = parseFloat(normalizedValue);
  if (isNaN(dollars) || dollars < 0) {
    return null;
  }

  return { displayDollars: dollars };
}

/**
 * Build tier limit update from parsed value
 */
export function buildTierLimitUpdate(
  parsedValue: { isUnlimited: true } | { displayDollars: number }
): TierLimitUpdate {
  if ('isUnlimited' in parsedValue) {
    return { is_unlimited: true };
  }
  return { display_dollars: parsedValue.displayDollars };
}

/**
 * Get display value for a limit
 */
export function getDisplayValue(limit: TierLimit): string {
  const isUnlimited = limit.limit_value < 0;
  if (isUnlimited) {
    return 'unlimited';
  }
  return limit.display_dollars?.toFixed(2) ?? '0';
}

/**
 * Collect cost-based limit keys from limits
 */
export function collectCostBasedLimitKeys(
  limits: Record<string, TierLimit[]>
): Set<string> {
  const limitKeys = new Set<string>();
  Object.values(limits).forEach((tierLimits) => {
    tierLimits.forEach((limit) => {
      if (limit.limit_type === 'cost_based') {
        limitKeys.add(limit.limit_key);
      }
    });
  });
  return limitKeys;
}

/**
 * Find AI credits limit for a tier
 */
export function findAICreditsLimit(
  tierLimits: TierLimit[] | undefined
): TierLimit | undefined {
  if (!tierLimits) return undefined;
  return tierLimits.find(
    (l) => l.limit_key === 'ai_credits' && l.limit_type === 'cost_based'
  );
}

/**
 * Calculate doubled limits for all tiers
 */
export function calculateDoubledLimits(
  limits: Record<string, TierLimit[]>
): Record<string, string> {
  const doubled: Record<string, string> = {};
  TIER_OPTIONS.forEach((tier) => {
    const limit = findAICreditsLimit(limits[tier.value]);
    if (limit && limit.display_dollars && limit.limit_value >= 0) {
      doubled[`${tier.value}:ai_credits`] = (limit.display_dollars * 2).toString();
    }
  });
  return doubled;
}

// API wrapper functions

/**
 * Fetch all tier limits
 */
export async function fetchAllTierLimits(): Promise<Record<string, TierLimit[]>> {
  const response = await apiGetAllTierLimits();
  return response.limits || {};
}

/**
 * Update a tier limit
 */
export async function saveTierLimit(
  tierID: string,
  limitKey: string,
  update: TierLimitUpdate
): Promise<void> {
  await apiUpdateTierLimit(tierID, limitKey, update);
}

// Re-export for convenience
export { formatDollars, TIER_OPTIONS };
