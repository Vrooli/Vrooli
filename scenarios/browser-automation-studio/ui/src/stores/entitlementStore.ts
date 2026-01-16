import { create } from 'zustand';
import { API_BASE } from '../config';

const joinApi = (base: string, path: string): string => {
  const normalizedBase = base.replace(/\/+$/, '');
  const normalizedPath = path.replace(/^\/+/, '');
  return `${normalizedBase}/${normalizedPath}`;
};

// Subscription tier types
export type SubscriptionTier = 'free' | 'solo' | 'pro' | 'studio' | 'business';
export type SubscriptionStatus = 'active' | 'trialing' | 'past_due' | 'canceled' | 'inactive';

// API source types for dev mode switching
export type ApiSource = 'production' | 'local' | 'disabled';

export interface ApiSourceConfig {
  source: ApiSource;
  localPort: number;
}

// API response type from backend
export interface EntitlementStatusResponse {
  user_identity: string;
  status: SubscriptionStatus;
  tier: SubscriptionTier;
  is_active: boolean;
  features: string[];
  feature_access?: FeatureAccessSummary[];
  monthly_limit: number; // -1 for unlimited
  monthly_used: number;
  monthly_remaining: number; // -1 for unlimited
  requires_watermark: boolean;
  can_use_ai: boolean;
  can_use_recording: boolean;
  entitlements_enabled: boolean;
  override_tier?: SubscriptionTier;

  // AI Credits
  ai_credits_used: number;
  ai_credits_limit: number; // -1 for unlimited
  ai_credits_remaining: number; // -1 for unlimited
  ai_requests_count: number;
  ai_reset_date: string; // ISO date
}

export interface FeatureAccessSummary {
  id: string;
  label: string;
  description: string;
  required_tier?: SubscriptionTier;
  has_access: boolean;
}

// Usage history types
export interface UsagePeriod {
  billing_month: string;
  total_credits_used: number;
  total_operations: number;
  by_operation: Record<string, number>;
  operation_counts: Record<string, number>;
  credits_limit: number;
  credits_remaining: number;
  period_start: string;
  period_end: string;
  reset_date: string;
}

export interface OperationLogEntry {
  id: string;
  operation_type: string;
  credits_charged: number;
  success: boolean;
  created_at: string;
  metadata?: Record<string, unknown>;
  error_message?: string;
}

export interface OperationLogPage {
  user_identity: string;
  billing_month: string;
  operations: OperationLogEntry[];
  total: number;
  limit: number;
  offset: number;
  has_more: boolean;
}

interface EntitlementState {
  // State
  userEmail: string;
  status: EntitlementStatusResponse | null;
  overrideTier: SubscriptionTier | null;
  isLoading: boolean;
  error: string | null;
  lastFetched: Date | null;
  isOffline: boolean;

  // API source state (for dev mode)
  apiSource: ApiSource;
  localApiPort: number;

  // Usage history state
  usageHistory: UsagePeriod[];
  historyLoading: boolean;
  selectedPeriod: string | null; // YYYY-MM format
  operationLog: OperationLogEntry[];
  operationLogLoading: boolean;
  operationLogTotal: number;
  operationLogHasMore: boolean;

  // Actions
  fetchStatus: () => Promise<void>;
  setUserEmail: (email: string) => Promise<void>;
  clearUserEmail: () => Promise<void>;
  refreshEntitlement: () => Promise<void>;
  getUserEmail: () => Promise<string>;
  setOverrideTier: (tier: SubscriptionTier | null) => Promise<void>;

  // API source actions (for dev mode)
  getApiSource: () => Promise<void>;
  setApiSource: (source: ApiSource, localPort?: number) => Promise<void>;

  // Usage history actions
  fetchUsageHistory: (months?: number, offset?: number) => Promise<void>;
  fetchOperationLog: (month: string, category?: string, limit?: number, offset?: number) => Promise<void>;
  setSelectedPeriod: (month: string | null) => void;
  clearOperationLog: () => void;
}

// Helper to check if email is valid (basic client-side validation)
export const isValidEmail = (email: string): boolean => {
  const trimmed = email.trim();
  if (!trimmed) return false;
  const atIndex = trimmed.indexOf('@');
  if (atIndex < 1) return false; // @ must not be first character
  const domain = trimmed.slice(atIndex + 1);
  return domain.length > 0 && domain.includes('.') && !domain.endsWith('.');
};

// Tier display configuration
export const TIER_CONFIG: Record<SubscriptionTier, { label: string; color: string; bgColor: string; borderColor: string }> = {
  free: {
    label: 'Free',
    color: 'text-gray-400',
    bgColor: 'bg-gray-700/50',
    borderColor: 'border-gray-600',
  },
  solo: {
    label: 'Solo',
    color: 'text-blue-400',
    bgColor: 'bg-blue-900/30',
    borderColor: 'border-blue-600',
  },
  pro: {
    label: 'Pro',
    color: 'text-purple-400',
    bgColor: 'bg-purple-900/30',
    borderColor: 'border-purple-600',
  },
  studio: {
    label: 'Studio',
    color: 'text-amber-400',
    bgColor: 'bg-amber-900/30',
    borderColor: 'border-amber-600',
  },
  business: {
    label: 'Business',
    color: 'text-emerald-400',
    bgColor: 'bg-gradient-to-r from-emerald-900/30 to-teal-900/30',
    borderColor: 'border-emerald-600',
  },
};

// Status display configuration
export const STATUS_CONFIG: Record<SubscriptionStatus, { label: string; color: string; icon: 'check' | 'clock' | 'alert' | 'x' }> = {
  active: { label: 'Active', color: 'text-green-400', icon: 'check' },
  trialing: { label: 'Trial', color: 'text-blue-400', icon: 'clock' },
  past_due: { label: 'Past Due', color: 'text-amber-400', icon: 'alert' },
  canceled: { label: 'Canceled', color: 'text-red-400', icon: 'x' },
  inactive: { label: 'Inactive', color: 'text-gray-400', icon: 'x' },
};

export const useEntitlementStore = create<EntitlementState>((set, get) => ({
  userEmail: '',
  status: null,
  overrideTier: null,
  isLoading: false,
  error: null,
  lastFetched: null,
  isOffline: false,

  // API source state (for dev mode)
  apiSource: 'production' as ApiSource,
  localApiPort: 15000, // Default LPBS API port range start

  // Usage history state
  usageHistory: [],
  historyLoading: false,
  selectedPeriod: null,
  operationLog: [],
  operationLogLoading: false,
  operationLogTotal: 0,
  operationLogHasMore: false,

  fetchStatus: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(joinApi(API_BASE, 'entitlement/status'), {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to fetch entitlement status: ${response.status}`);
      }

      const data: EntitlementStatusResponse = await response.json();
      set({
        status: data,
        userEmail: data.user_identity || '',
        overrideTier: data.override_tier ?? null,
        isLoading: false,
        lastFetched: new Date(),
        isOffline: false,
      });
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to fetch status';
      // Check if this is a network error (offline)
      const isNetworkError = err instanceof TypeError && err.message.includes('fetch');
      set({
        error: errorMessage,
        isLoading: false,
        isOffline: isNetworkError,
      });
    }
  },

  setUserEmail: async (email: string) => {
    const trimmedEmail = email.trim().toLowerCase();
    if (!trimmedEmail) {
      set({ error: 'Email is required' });
      return;
    }
    if (!isValidEmail(trimmedEmail)) {
      set({ error: 'Please enter a valid email address' });
      return;
    }

    set({ isLoading: true, error: null });
    try {
      const response = await fetch(joinApi(API_BASE, 'entitlement/identity'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({ email: trimmedEmail }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to set email: ${response.status}`);
      }

      // After setting email, fetch the updated status
      await get().fetchStatus();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to set email';
      set({
        error: errorMessage,
        isLoading: false,
      });
    }
  },

  clearUserEmail: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(joinApi(API_BASE, 'entitlement/identity'), {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to clear email: ${response.status}`);
      }

      set({
        userEmail: '',
        status: null,
        overrideTier: null,
        isLoading: false,
        lastFetched: new Date(),
      });
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to clear email';
      set({
        error: errorMessage,
        isLoading: false,
      });
    }
  },

  refreshEntitlement: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(joinApi(API_BASE, 'entitlement/refresh'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to refresh entitlement: ${response.status}`);
      }

      const data: EntitlementStatusResponse = await response.json();
      set({
        status: data,
        userEmail: data.user_identity || '',
        overrideTier: data.override_tier ?? null,
        isLoading: false,
        lastFetched: new Date(),
        isOffline: false,
      });
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to refresh';
      const isNetworkError = err instanceof TypeError && err.message.includes('fetch');
      set({
        error: errorMessage,
        isLoading: false,
        isOffline: isNetworkError,
      });
    }
  },

  getUserEmail: async (): Promise<string> => {
    try {
      const response = await fetch(joinApi(API_BASE, 'entitlement/identity'), {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });

      if (!response.ok) {
        return '';
      }

      const data = await response.json();
      const email = data.email || '';
      set({ userEmail: email });
      return email;
    } catch {
      return '';
    }
  },

  setOverrideTier: async (tier: SubscriptionTier | null) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(joinApi(API_BASE, 'entitlement/override'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({ tier: tier ?? '' }),
      });

      if (!response.ok && response.status !== 204) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to set override tier: ${response.status}`);
      }

      await get().fetchStatus();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to set override tier';
      set({
        error: errorMessage,
        isLoading: false,
      });
    }
  },

  // API source actions (for dev mode)
  getApiSource: async () => {
    try {
      const response = await fetch(joinApi(API_BASE, 'entitlement/api-source'), {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });

      if (!response.ok) {
        // Default to production if endpoint doesn't exist or fails
        return;
      }

      const data = await response.json();
      set({
        apiSource: (data.source || 'production') as ApiSource,
        localApiPort: data.local_port || 15000,
      });
    } catch {
      // Silently fail - defaults are fine
    }
  },

  setApiSource: async (source: ApiSource, localPort?: number) => {
    set({ isLoading: true, error: null });
    try {
      const body: Record<string, unknown> = { source };
      if (localPort !== undefined) {
        body.local_port = localPort;
      }

      const response = await fetch(joinApi(API_BASE, 'entitlement/api-source'), {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify(body),
      });

      if (!response.ok && response.status !== 204) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || `Failed to set API source: ${response.status}`);
      }

      set({
        apiSource: source,
        localApiPort: localPort ?? get().localApiPort,
        isLoading: false,
      });

      // Refresh entitlement status with new API source
      await get().fetchStatus();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to set API source';
      set({
        error: errorMessage,
        isLoading: false,
      });
    }
  },

  // Usage history actions
  fetchUsageHistory: async (months = 6, offset = 0) => {
    set({ historyLoading: true });
    try {
      const params = new URLSearchParams();
      params.set('months', months.toString());
      if (offset > 0) params.set('offset', offset.toString());

      const response = await fetch(joinApi(API_BASE, `entitlement/usage/history?${params.toString()}`), {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch usage history: ${response.status}`);
      }

      const data = await response.json();
      set({
        usageHistory: data.periods || [],
        historyLoading: false,
      });
    } catch (err) {
      console.error('Failed to fetch usage history:', err);
      set({
        historyLoading: false,
      });
    }
  },

  fetchOperationLog: async (month: string, category?: string, limit = 20, offset = 0) => {
    set({ operationLogLoading: true });
    try {
      const params = new URLSearchParams();
      params.set('month', month);
      if (category) params.set('category', category);
      params.set('limit', limit.toString());
      if (offset > 0) params.set('offset', offset.toString());

      const response = await fetch(joinApi(API_BASE, `entitlement/usage/operations?${params.toString()}`), {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch operation log: ${response.status}`);
      }

      const data: OperationLogPage = await response.json();

      // If offset > 0, append to existing operations
      if (offset > 0) {
        set((state) => ({
          operationLog: [...state.operationLog, ...data.operations],
          operationLogTotal: data.total,
          operationLogHasMore: data.has_more,
          operationLogLoading: false,
        }));
      } else {
        set({
          operationLog: data.operations,
          operationLogTotal: data.total,
          operationLogHasMore: data.has_more,
          operationLogLoading: false,
        });
      }
    } catch (err) {
      console.error('Failed to fetch operation log:', err);
      set({
        operationLogLoading: false,
      });
    }
  },

  setSelectedPeriod: (month: string | null) => {
    set({ selectedPeriod: month });
  },

  clearOperationLog: () => {
    set({
      operationLog: [],
      operationLogTotal: 0,
      operationLogHasMore: false,
    });
  },
}));

// Convenience hooks for common checks
export const useIsEntitlementsEnabled = (): boolean => {
  const status = useEntitlementStore((state) => state.status);
  return status?.entitlements_enabled ?? false;
};

export const useCanExecuteWorkflow = (): boolean => {
  const status = useEntitlementStore((state) => state.status);
  if (!status?.entitlements_enabled) return true; // No restrictions when disabled
  if (status.monthly_limit === -1) return true; // Unlimited
  return status.monthly_remaining > 0;
};

export const useCanUseAI = (): boolean => {
  const status = useEntitlementStore((state) => state.status);
  if (!status?.entitlements_enabled) return true;
  return status.can_use_ai;
};

export const useCanUseRecording = (): boolean => {
  const status = useEntitlementStore((state) => state.status);
  if (!status?.entitlements_enabled) return true;
  return status.can_use_recording;
};

export const useRequiresWatermark = (): boolean => {
  const status = useEntitlementStore((state) => state.status);
  if (!status?.entitlements_enabled) return false;
  return status.requires_watermark;
};

export const useCurrentTier = (): SubscriptionTier => {
  const status = useEntitlementStore((state) => state.status);
  return status?.tier ?? 'free';
};

// AI Credits hooks
export interface AICreditsInfo {
  used: number;
  limit: number;
  remaining: number;
  requestsCount: number;
  resetDate: string;
  isUnlimited: boolean;
  hasAccess: boolean;
  percentUsed: number;
}

export const useAICredits = (): AICreditsInfo => {
  const status = useEntitlementStore((state) => state.status);

  if (!status?.entitlements_enabled) {
    return {
      used: 0,
      limit: -1,
      remaining: -1,
      requestsCount: 0,
      resetDate: '',
      isUnlimited: true,
      hasAccess: true,
      percentUsed: 0,
    };
  }

  const used = status.ai_credits_used ?? 0;
  const limit = status.ai_credits_limit ?? 0;
  const remaining = status.ai_credits_remaining ?? 0;
  const isUnlimited = limit < 0;
  const hasAccess = limit !== 0; // 0 means no access, -1 means unlimited, positive means limited

  return {
    used,
    limit,
    remaining,
    requestsCount: status.ai_requests_count ?? 0,
    resetDate: status.ai_reset_date ?? '',
    isUnlimited,
    hasAccess,
    percentUsed: isUnlimited || limit <= 0 ? 0 : Math.min(100, Math.round((used / limit) * 100)),
  };
};

export default useEntitlementStore;
