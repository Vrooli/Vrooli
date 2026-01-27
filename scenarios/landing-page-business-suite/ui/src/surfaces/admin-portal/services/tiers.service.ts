import type { TierLimit, TierLimitUpdate } from '../../../shared/api';
import {
  getAllTierLimits as apiGetAllTierLimits,
  updateTierLimit as apiUpdateTierLimit,
  formatDollars,
} from '../../../shared/api';
import { TIER_OPTIONS } from '../../../shared/lib/tierUtils';

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
