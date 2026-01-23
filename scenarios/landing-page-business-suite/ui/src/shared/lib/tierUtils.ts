/**
 * Shared tier utility functions for tier limit management.
 *
 * These functions are used by both appLimits.service.ts and tiers.service.ts
 * to provide consistent tier handling across the application.
 */

import type { TierLimit, TierLimitUpdate } from '../api';
import { TIER_OPTIONS } from '../api';

/**
 * Generate a unique key for identifying a limit edit.
 *
 * @param tierID - The tier identifier
 * @param limitKey - The limit key name
 * @returns A combined key in the format "tierID:limitKey"
 *
 * @example
 * getEditKey('pro', 'ai_credits') // 'pro:ai_credits'
 */
export function getEditKey(tierID: string, limitKey: string): string {
  return `${tierID}:${limitKey}`;
}

/**
 * Get human-readable label for a tier.
 *
 * @param tierID - The tier identifier
 * @returns The tier's display label, or the tierID itself if not found
 *
 * @example
 * getTierLabel('pro') // 'Pro'
 * getTierLabel('unknown') // 'unknown'
 */
export function getTierLabel(tierID: string): string {
  const option = TIER_OPTIONS.find((t) => t.value === tierID);
  return option?.label || tierID;
}

/**
 * Get Tailwind color class for tier badge styling.
 *
 * @param tierID - The tier identifier
 * @returns Tailwind CSS class for the tier's color
 *
 * @example
 * getTierColor('pro') // 'text-purple-400'
 * getTierColor('free') // 'text-slate-400'
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
 * Check if a limit value represents unlimited.
 *
 * Negative values indicate unlimited in the tier limit system.
 *
 * @param limitValue - The numeric limit value
 * @returns true if the limit is unlimited (negative)
 *
 * @example
 * isUnlimitedValue(-1) // true
 * isUnlimitedValue(100) // false
 */
export function isUnlimitedValue(limitValue: number): boolean {
  return limitValue < 0;
}

/**
 * Parse and validate a limit value input.
 *
 * Accepts 'unlimited', '-1', or positive dollar amounts.
 *
 * @param value - The string value to parse
 * @returns Parsed result, or null if invalid
 *
 * @example
 * parseEditedValue('unlimited') // { isUnlimited: true }
 * parseEditedValue('-1') // { isUnlimited: true }
 * parseEditedValue('5.00') // { displayDollars: 5 }
 * parseEditedValue('invalid') // null
 * parseEditedValue('-5') // null (negative non-unlimited)
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
 * Build TierLimitUpdate from parsed value.
 *
 * @param parsedValue - The result from parseEditedValue (excluding null)
 * @returns API update payload
 *
 * @example
 * buildTierLimitUpdate({ isUnlimited: true }) // { is_unlimited: true }
 * buildTierLimitUpdate({ displayDollars: 5 }) // { display_dollars: 5 }
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
 * Get display value for a limit.
 *
 * @param limit - The tier limit object
 * @returns Display string ('unlimited' or formatted dollar value)
 *
 * @example
 * getDisplayValue({ limit_value: -1, display_dollars: null }) // 'unlimited'
 * getDisplayValue({ limit_value: 500, display_dollars: 5 }) // '5.00'
 * getDisplayValue({ limit_value: 0, display_dollars: null }) // '0'
 */
export function getDisplayValue(limit: TierLimit): string {
  const isUnlimited = limit.limit_value < 0;
  if (isUnlimited) {
    return 'unlimited';
  }
  return limit.display_dollars?.toFixed(2) ?? '0';
}

// Re-export TIER_OPTIONS for convenience
export { TIER_OPTIONS };
