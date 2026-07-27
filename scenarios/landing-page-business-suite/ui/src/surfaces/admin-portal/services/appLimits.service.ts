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


/**
 * Validate new limit form data
 */
export function validateNewLimitForm(form: NewLimitFormState): string | null {
  if (!form.limit_key.trim()) {
    return 'Please enter a limit key';
  }
  return null;
}

/**
 * Build request payload for creating a new limit
 */
export function buildCreateLimitPayload(
  form: NewLimitFormState,
  selectedApp: string
): Partial<TierLimit> {
  const dollars = parseFloat(form.display_dollars) || 0;

  return {
    tier_id: form.tier_id,
    limit_type: 'app_specific',
    limit_key: form.limit_key.trim(),
    limit_value: Math.round(dollars * 100 * 1000000), // Convert to internal units
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
