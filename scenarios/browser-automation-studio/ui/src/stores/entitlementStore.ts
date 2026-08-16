import { ConnectError } from '@connectrpc/connect';
import { create } from 'zustand';
import type {
  EntitlementStatus,
  OperationLogEntry as ProtoOperationLogEntry,
  OperationLogPage as ProtoOperationLogPage,
  UsageSummary as ProtoUsageSummary,
} from '@vrooli/proto-types/browser-automation-studio/v1/entitlement/entitlement_pb';
import type { Timestamp } from '@bufbuild/protobuf/wkt';

import { entitlementClient } from '../api/entitlement';

// Subscription tier types
export type SubscriptionTier = 'free' | 'solo' | 'pro' | 'studio' | 'business';
export type SubscriptionStatus = 'active' | 'trialing' | 'past_due' | 'canceled' | 'inactive';

// API response type (kept in snake_case for downstream consumer compatibility).
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

  // AI Credits
  ai_credits_used: number;
  ai_credits_limit: number;
  ai_credits_remaining: number;
  ai_requests_count: number;
  ai_reset_date: string;
}

export interface FeatureAccessSummary {
  id: string;
  label: string;
  description: string;
  required_tier?: SubscriptionTier;
  has_access: boolean;
}

// Usage history types (snake_case for consumer compatibility).
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
  userEmail: string;
  status: EntitlementStatusResponse | null;
  isLoading: boolean;
  error: string | null;
  lastFetched: Date | null;
  isOffline: boolean;

  usageHistory: UsagePeriod[];
  historyLoading: boolean;
  selectedPeriod: string | null;
  operationLog: OperationLogEntry[];
  operationLogLoading: boolean;
  operationLogTotal: number;
  operationLogHasMore: boolean;

  fetchStatus: () => Promise<void>;
  setUserEmail: (email: string) => Promise<void>;
  clearUserEmail: () => Promise<void>;
  refreshEntitlement: () => Promise<void>;
  getUserEmail: () => Promise<string>;
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
  if (atIndex < 1) return false;
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

// ----------------------------------------------------------------------------
// proto → store-shape adapters
// ----------------------------------------------------------------------------

const tsToIso = (ts?: Timestamp): string => {
  if (!ts) return '';
  const seconds = typeof ts.seconds === 'bigint' ? Number(ts.seconds) : Number(ts.seconds ?? 0);
  const nanos = Number(ts.nanos ?? 0);
  return new Date(seconds * 1000 + Math.floor(nanos / 1_000_000)).toISOString();
};

const toMaybeTier = (value: string | undefined): SubscriptionTier | undefined => {
  if (!value) return undefined;
  if (value === 'free' || value === 'solo' || value === 'pro' || value === 'studio' || value === 'business') {
    return value;
  }
  return undefined;
};

const toStatus = (proto: EntitlementStatus | undefined): EntitlementStatusResponse | null => {
  if (!proto) return null;
  return {
    user_identity: proto.userIdentity,
    status: (proto.status || 'inactive') as SubscriptionStatus,
    tier: (toMaybeTier(proto.tier) ?? 'free') as SubscriptionTier,
    is_active: proto.isActive,
    features: proto.features ?? [],
    feature_access: (proto.featureAccess ?? []).map((fa) => ({
      id: fa.id,
      label: fa.label,
      description: fa.description,
      required_tier: toMaybeTier(fa.requiredTier),
      has_access: fa.hasAccess,
    })),
    monthly_limit: proto.monthlyLimit,
    monthly_used: proto.monthlyUsed,
    monthly_remaining: proto.monthlyRemaining,
    requires_watermark: proto.requiresWatermark,
    can_use_ai: proto.canUseAi,
    can_use_recording: proto.canUseRecording,
    entitlements_enabled: proto.entitlementsEnabled,
    ai_credits_used: proto.aiCreditsUsed,
    ai_credits_limit: proto.aiCreditsLimit,
    ai_credits_remaining: proto.aiCreditsRemaining,
    ai_requests_count: proto.aiRequestsCount,
    ai_reset_date: proto.aiResetDate,
  };
};

const toUsagePeriod = (u: ProtoUsageSummary): UsagePeriod => ({
  billing_month: u.billingMonth,
  total_credits_used: u.totalCreditsUsed,
  total_operations: u.totalOperations,
  by_operation: { ...u.byOperation },
  operation_counts: { ...u.operationCounts },
  credits_limit: u.creditsLimit,
  credits_remaining: u.creditsRemaining,
  period_start: tsToIso(u.periodStart),
  period_end: tsToIso(u.periodEnd),
  reset_date: tsToIso(u.resetDate),
});

const toOperationLogEntry = (e: ProtoOperationLogEntry): OperationLogEntry => ({
  id: e.id,
  operation_type: e.operationType,
  credits_charged: e.creditsCharged,
  success: e.success,
  created_at: tsToIso(e.createdAt),
  metadata: e.metadata ? (e.metadata.fields as unknown as Record<string, unknown>) : undefined,
  error_message: e.errorMessage || undefined,
});

const toOperationLogPage = (p: ProtoOperationLogPage): OperationLogPage => ({
  user_identity: p.userIdentity,
  billing_month: p.billingMonth,
  operations: (p.operations ?? []).map(toOperationLogEntry),
  total: p.total,
  limit: p.limit,
  offset: p.offset,
  has_more: p.hasMore,
});

const messageFromError = (err: unknown, fallback: string): string => {
  if (err instanceof ConnectError) return err.message;
  if (err instanceof Error) return err.message;
  return fallback;
};

const isNetworkLikeError = (err: unknown): boolean => {
  if (err instanceof TypeError && err.message.includes('fetch')) return true;
  if (err instanceof ConnectError) {
    // Connect maps unreachable / DNS / TLS failures to Unavailable.
    return err.code === 14 /* Unavailable */;
  }
  return false;
};

// ----------------------------------------------------------------------------
// store
// ----------------------------------------------------------------------------

export const useEntitlementStore = create<EntitlementState>((set, get) => ({
  userEmail: '',
  status: null,
  isLoading: false,
  error: null,
  lastFetched: null,
  isOffline: false,

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
      const resp = await entitlementClient.getStatus({});
      const data = toStatus(resp.status);
      if (!data) {
        set({ isLoading: false });
        return;
      }
      set({
        status: data,
        userEmail: data.user_identity || '',
        isLoading: false,
        lastFetched: new Date(),
        isOffline: false,
      });
    } catch (err) {
      set({
        error: messageFromError(err, 'Failed to fetch status'),
        isLoading: false,
        isOffline: isNetworkLikeError(err),
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
      const resp = await entitlementClient.setIdentity({ email: trimmedEmail });
      const data = toStatus(resp.status);
      if (!data) {
        set({ isLoading: false });
        return;
      }
      set({
        status: data,
        userEmail: data.user_identity || '',
        isLoading: false,
        lastFetched: new Date(),
        isOffline: false,
      });
    } catch (err) {
      set({ error: messageFromError(err, 'Failed to set email'), isLoading: false });
    }
  },

  clearUserEmail: async () => {
    set({ isLoading: true, error: null });
    try {
      await entitlementClient.clearIdentity({});
      set({
        userEmail: '',
        status: null,
        isLoading: false,
        lastFetched: new Date(),
      });
    } catch (err) {
      set({ error: messageFromError(err, 'Failed to clear email'), isLoading: false });
    }
  },

  refreshEntitlement: async () => {
    const currentUser = get().userEmail || get().status?.user_identity || '';
    if (!currentUser) {
      // Nothing to refresh without a user identity; just refetch status.
      await get().fetchStatus();
      return;
    }
    set({ isLoading: true, error: null });
    try {
      const resp = await entitlementClient.refreshStatus({ user: currentUser });
      const data = toStatus(resp.status);
      if (!data) {
        set({ isLoading: false });
        return;
      }
      set({
        status: data,
        userEmail: data.user_identity || '',
        isLoading: false,
        lastFetched: new Date(),
        isOffline: false,
      });
    } catch (err) {
      set({
        error: messageFromError(err, 'Failed to refresh'),
        isLoading: false,
        isOffline: isNetworkLikeError(err),
      });
    }
  },

  getUserEmail: async (): Promise<string> => {
    try {
      const resp = await entitlementClient.getIdentity({});
      const email = resp.email || '';
      set({ userEmail: email });
      return email;
    } catch {
      return '';
    }
  },

  fetchUsageHistory: async (months = 6, offset = 0) => {
    set({ historyLoading: true });
    try {
      const resp = await entitlementClient.getUsageHistory({ months, offset });
      set({
        usageHistory: (resp.periods ?? []).map(toUsagePeriod),
        historyLoading: false,
      });
    } catch (err) {
      console.error('Failed to fetch usage history:', err);
      set({ historyLoading: false });
    }
  },

  fetchOperationLog: async (month: string, category?: string, limit = 20, offset = 0) => {
    set({ operationLogLoading: true });
    try {
      const resp = await entitlementClient.getOperationLog({
        month,
        category: category ?? '',
        limit,
        offset,
      });
      const page = toOperationLogPage(resp);
      if (offset > 0) {
        set((state) => ({
          operationLog: [...state.operationLog, ...page.operations],
          operationLogTotal: page.total,
          operationLogHasMore: page.has_more,
          operationLogLoading: false,
        }));
      } else {
        set({
          operationLog: page.operations,
          operationLogTotal: page.total,
          operationLogHasMore: page.has_more,
          operationLogLoading: false,
        });
      }
    } catch (err) {
      console.error('Failed to fetch operation log:', err);
      set({ operationLogLoading: false });
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
  if (!status?.entitlements_enabled) return true;
  if (status.monthly_limit === -1) return true;
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
  const hasAccess = limit !== 0;

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
