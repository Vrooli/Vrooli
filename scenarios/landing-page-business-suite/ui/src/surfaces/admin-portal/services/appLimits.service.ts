import type { TierLimit, TierLimitUpdate } from '../../../shared/api';
import {
  getAppLimits,
  updateTierLimit as apiUpdateTierLimit,
  createTierLimit as apiCreateTierLimit,
  deleteTierLimit as apiDeleteTierLimit,
} from '../../../shared/api';
import { TIER_OPTIONS } from '../../../shared/lib/tierUtils';

/**
 * App options for app-specific limits configuration
 */
export const APP_OPTIONS = [
  { value: 'browser-automation-studio', label: 'Browser Automation Studio' },
  // Future apps can be added here
] as const;

export type AppOption = (typeof APP_OPTIONS)[number];

/**
 * Form state for creating a new limit
 */
export interface NewLimitFormState {
  tier_id: string;
  limit_key: string;
  display_dollars: string;
}

/**
 * Default values for new limit form
 */
export const DEFAULT_NEW_LIMIT: NewLimitFormState = {
  tier_id: 'solo',
  limit_key: '',
  display_dollars: '',
};


/**
 * Collect all unique limit keys across tiers
 */
export function collectLimitKeys(limits: Record<string, TierLimit[]>): Set<string> {
  const limitKeys = new Set<string>();
  Object.values(limits).forEach((tierLimits) => {
    tierLimits.forEach((limit) => {
      limitKeys.add(limit.limit_key);
    });
  });
  return limitKeys;
}


function costLimitValue(input: string): number | null {
  const value = input.trim().toLowerCase();
  if (value === 'unlimited' || value === '-1') return -1;
  const dollars = Number(value);
  const units = Math.round(dollars * 100 * 1000000);
  return Number.isFinite(dollars) && dollars >= 0 && Number.isSafeInteger(units) ? units : null;
}

/**
 * Validate new limit form data
 */
export function validateNewLimitForm(form: NewLimitFormState): string | null {
  if (!form.limit_key.trim()) {
    return 'Please enter a limit key';
  }
  if (costLimitValue(form.display_dollars) === null) return 'Enter a non-negative dollar amount or unlimited';
  return null;
}

/**
 * Build request payload for creating a new limit
 */
export function buildCreateLimitPayload(
  form: NewLimitFormState,
  selectedApp: string
): Partial<TierLimit> {
  const error = validateNewLimitForm(form);
  const value = costLimitValue(form.display_dollars);
  if (error || value === null || !selectedApp.trim()) throw new Error(error ?? 'Select an exact application');

  return {
    tier_id: form.tier_id,
    limit_type: 'cost_based',
    limit_key: form.limit_key.trim(),
    limit_value: value,
    cost_multiplier: 1000000,
    app_bundle_key: selectedApp,
    reset_period: 'monthly',
  };
}


/**
 * Get the selected app label
 */
export function getSelectedAppLabel(selectedApp: string): string {
  const app = APP_OPTIONS.find((a) => a.value === selectedApp);
  return app?.label || selectedApp;
}

// API wrapper functions for cleaner abstraction

/**
 * Fetch app limits from API
 */
export async function fetchAppLimits(
  appBundleKey: string
): Promise<Record<string, TierLimit[]>> {
  const response = await getAppLimits(appBundleKey);
  return response.limits;
}

/**
 * Update a tier limit via API
 */
export async function saveTierLimit(
  tierID: string,
  limitKey: string,
  update: TierLimitUpdate,
  selectedApp: string
): Promise<TierLimit> {
  return apiUpdateTierLimit(tierID, limitKey, update, selectedApp);
}

/**
 * Create a new tier limit via API
 */
export async function createNewTierLimit(
  limit: Partial<TierLimit>
): Promise<TierLimit> {
  return apiCreateTierLimit(limit);
}

/**
 * Delete a tier limit via API
 */
export async function removeTierLimit(
  tierID: string,
  limitKey: string,
  appBundleKey?: string
): Promise<void> {
  return apiDeleteTierLimit(tierID, limitKey, appBundleKey);
}

// Re-export TIER_OPTIONS for convenience
export { TIER_OPTIONS };

/** Presentation follows the declared server meter kind; app scope alone never
 * establishes currency units. The server owns the legacy AI-cap mapping. */
export function getAppMeterKind(limit: TierLimit): 'cost_based' | 'count_based' | null {
  return limit.limit_type === 'cost_based' || limit.limit_type === 'count_based'
    ? limit.limit_type
    : null;
}
