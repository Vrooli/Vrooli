import { apiGet, apiPost, apiDelete, apiPut, API_BASE } from './common';

// Types for the credit system

export interface APIKey {
  id: string;
  provider: string;
  key_hint: string;
  is_active: boolean;
  last_verified_at?: string;
  created_at: string;
  updated_at: string;
}

export interface APIKeyCreateRequest {
  provider: string;
  key: string;
}

export interface APIKeyTestResult {
  success: boolean;
  message: string;
  provider: string;
}

export interface TierLimit {
  id: string;
  tier_id: string;
  limit_type: 'cost_based' | 'app_specific';
  limit_key: string;
  limit_value: number;
  cost_multiplier: number;
  app_bundle_key?: string;
  reset_period: string;
  created_at: string;
  updated_at: string;
  display_dollars?: number;
}

export interface TierLimitUpdate {
  limit_value?: number;
  display_dollars?: number;
  is_unlimited?: boolean;
}

export interface UsageRecord {
  id: string;
  user_identity: string;
  billing_period: string;
  limit_key: string;
  usage_amount: number;
  app_bundle_key?: string;
  last_operation_at?: string;
  created_at: string;
  updated_at: string;
}

export interface UsageSummary {
  user_identity: string;
  billing_period: string;
  tier?: string;
  limits: Record<string, number>;
  usage: Record<string, number>;
  remaining: Record<string, number>;
  display_credits: Record<string, number>;
  reset_date: string;
  by_app?: Record<string, number>;
}

// API Key Management

export async function listAPIKeys(): Promise<{ keys: APIKey[] }> {
  return apiGet<{ keys: APIKey[] }>('/api/v1/admin/api-keys');
}

export async function createAPIKey(request: APIKeyCreateRequest): Promise<APIKey> {
  return apiPost<APIKey>('/api/v1/admin/api-keys', request);
}

export async function deleteAPIKey(provider: string): Promise<void> {
  const response = await fetch(`${API_BASE}/api/v1/admin/api-keys?provider=${encodeURIComponent(provider)}`, {
    method: 'DELETE',
    credentials: 'include',
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to delete API key' }));
    throw new Error(error.message || 'Failed to delete API key');
  }
}

export async function testAPIKey(provider: string): Promise<APIKeyTestResult> {
  return apiPost<APIKeyTestResult>(`/api/v1/admin/api-keys/test?provider=${encodeURIComponent(provider)}`, {});
}

export async function toggleAPIKey(provider: string, active: boolean): Promise<void> {
  return apiPost<void>('/api/v1/admin/api-keys/toggle', { provider, active });
}

// Tier Limits Management

export async function getAllTierLimits(): Promise<{ limits: Record<string, TierLimit[]> }> {
  return apiGet<{ limits: Record<string, TierLimit[]> }>('/api/v1/admin/tiers/limits');
}

export async function getTierLimits(tierID: string): Promise<{ tier_id: string; limits: TierLimit[] }> {
  return apiGet<{ tier_id: string; limits: TierLimit[] }>(`/api/v1/admin/tiers/${encodeURIComponent(tierID)}/limits`);
}

export async function updateTierLimit(
  tierID: string,
  limitKey: string,
  update: TierLimitUpdate,
  appBundleKey?: string
): Promise<TierLimit> {
  return apiPut<TierLimit>(`/api/v1/admin/tiers/${encodeURIComponent(tierID)}/limits`, {
    limit_key: limitKey,
    app_bundle_key: appBundleKey,
    update,
  });
}

export async function createTierLimit(limit: Partial<TierLimit>): Promise<TierLimit> {
  return apiPost<TierLimit>('/api/v1/admin/limits', limit);
}

export async function deleteTierLimit(tierID: string, limitKey: string, appBundleKey?: string): Promise<void> {
  return apiDelete<void>('/api/v1/admin/limits', {
    tier_id: tierID,
    limit_key: limitKey,
    app_bundle_key: appBundleKey,
  });
}

// App Limits

export async function getAppLimits(appBundleKey: string): Promise<{ app_bundle_key: string; limits: Record<string, TierLimit[]> }> {
  return apiGet<{ app_bundle_key: string; limits: Record<string, TierLimit[]> }>(`/api/v1/admin/apps/${encodeURIComponent(appBundleKey)}/limits`);
}

// Usage Management

export async function getUsageSummary(userIdentity?: string, tier?: string): Promise<UsageSummary> {
  const params = new URLSearchParams();
  if (userIdentity) params.set('user', userIdentity);
  if (tier) params.set('tier', tier);
  return apiGet<UsageSummary>(`/api/v1/usage/summary?${params.toString()}`);
}

export interface AdminUsageSummary {
  billing_period: string;
  records: UsageRecord[];
  user_totals: Record<string, number>;
  app_totals: Record<string, number>;
  total_users: number;
  total_records: number;
}

export async function getAdminUsageSummary(billingPeriod?: string): Promise<AdminUsageSummary> {
  const params = new URLSearchParams();
  if (billingPeriod) params.set('period', billingPeriod);
  return apiGet<AdminUsageSummary>(`/api/v1/admin/usage?${params.toString()}`);
}

// Helper functions

export function formatCredits(internalUnits: number): string {
  if (internalUnits < 0) return 'Unlimited';
  if (internalUnits === 0) return '0';

  // Convert internal units to display credits (divide by 100000)
  const displayCredits = internalUnits / 100000;

  if (displayCredits >= 1000) {
    return `${(displayCredits / 1000).toFixed(1)}k`;
  }
  return displayCredits.toFixed(0);
}

export function formatDollars(internalUnits: number, costMultiplier = 1000000): string {
  if (internalUnits < 0) return 'Unlimited';
  if (internalUnits === 0) return '$0';

  // Convert internal units to dollars
  // internal_units / cost_multiplier = cents, cents / 100 = dollars
  const dollars = internalUnits / costMultiplier / 100;

  if (dollars >= 1000) {
    return `$${(dollars / 1000).toFixed(1)}k`;
  }
  return `$${dollars.toFixed(2)}`;
}

export function dollarsToInternalUnits(dollars: number, costMultiplier = 1000000): number {
  // dollars * 100 = cents, cents * cost_multiplier = internal_units
  return Math.round(dollars * 100 * costMultiplier);
}

export const TIER_OPTIONS = [
  { value: 'free', label: 'Free' },
  { value: 'solo', label: 'Solo' },
  { value: 'pro', label: 'Pro' },
  { value: 'studio', label: 'Studio' },
  { value: 'business', label: 'Business' },
] as const;

export const PROVIDER_OPTIONS = [
  { value: 'openrouter', label: 'OpenRouter', description: 'Access to multiple AI models' },
  { value: 'openai', label: 'OpenAI', description: 'GPT-4 and other OpenAI models' },
  { value: 'anthropic', label: 'Anthropic', description: 'Claude models' },
] as const;
