import type { APIKey } from '../../../shared/api';
import {
  listAPIKeys as apiListAPIKeys,
  createAPIKey as apiCreateAPIKey,
  deleteAPIKey as apiDeleteAPIKey,
  testAPIKey as apiTestAPIKey,
  toggleAPIKey as apiToggleAPIKey,
  PROVIDER_OPTIONS,
} from '../../../shared/api';

/**
 * Test result from API key validation
 */
export interface KeyTestResult {
  success: boolean;
  message: string;
}

/**
 * Form state for adding a new API key
 */
export interface NewKeyFormState {
  provider: string;
  keyValue: string;
}

/**
 * Default form state for new key
 */
export const DEFAULT_NEW_KEY_FORM: NewKeyFormState = {
  provider: '',
  keyValue: '',
};

/**
 * Validate new key form
 */
export function validateNewKeyForm(form: NewKeyFormState): string | null {
  if (!form.provider) {
    return 'Provider is required';
  }
  if (!form.keyValue) {
    return 'API key is required';
  }
  return null;
}

/**
 * Get human-readable label for a provider
 */
export function getProviderLabel(provider: string): string {
  const option = PROVIDER_OPTIONS.find((p) => p.value === provider);
  return option?.label || provider;
}

/**
 * Get description for a provider
 */
export function getProviderDescription(provider: string): string {
  const option = PROVIDER_OPTIONS.find((p) => p.value === provider);
  return option?.description || '';
}

/** Type for a provider option */
export type ProviderOption = (typeof PROVIDER_OPTIONS)[number];

/**
 * Get available providers (not yet configured)
 */
export function getAvailableProviders(configuredKeys: APIKey[]): ProviderOption[] {
  return PROVIDER_OPTIONS.filter(
    (p) => !configuredKeys.some((k) => k.provider === p.value)
  );
}

/**
 * Check if any providers are available for configuration
 */
export function hasAvailableProviders(configuredKeys: APIKey[]): boolean {
  return getAvailableProviders(configuredKeys).length > 0;
}

/**
 * Remove test result for a provider from results map
 */
export function removeTestResult(
  results: Record<string, KeyTestResult>,
  provider: string
): Record<string, KeyTestResult> {
  const next = { ...results };
  delete next[provider];
  return next;
}

/**
 * Add test result to results map
 */
export function addTestResult(
  results: Record<string, KeyTestResult>,
  provider: string,
  result: KeyTestResult
): Record<string, KeyTestResult> {
  return {
    ...results,
    [provider]: result,
  };
}

// API wrapper functions

/**
 * Fetch all API keys
 */
export async function fetchAPIKeys(): Promise<APIKey[]> {
  const response = await apiListAPIKeys();
  return response.keys || [];
}

/**
 * Create a new API key
 */
export async function createAPIKey(provider: string, key: string): Promise<void> {
  await apiCreateAPIKey({ provider, key });
}

/**
 * Delete an API key
 */
export async function deleteAPIKey(provider: string): Promise<void> {
  await apiDeleteAPIKey(provider);
}

/**
 * Test an API key
 */
export async function testAPIKey(provider: string): Promise<KeyTestResult> {
  const result = await apiTestAPIKey(provider);
  return {
    success: result.success,
    message: result.message,
  };
}

/**
 * Toggle API key active status
 */
export async function toggleAPIKey(
  provider: string,
  currentActive: boolean
): Promise<void> {
  await apiToggleAPIKey(provider, !currentActive);
}

// Re-export for convenience
export { PROVIDER_OPTIONS };
