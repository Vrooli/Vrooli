import { ConnectError } from '@connectrpc/connect';
import { create } from 'zustand';
import type {
  EntitlementStatus,
} from '@vrooli/proto-types/browser-automation-studio/v1/entitlement/entitlement_pb';

import { entitlementClient } from '../api/entitlement';
import { EntitlementStore } from '@vrooli/react-component-library/EntitlementStore/1.0.0';
import {
  toOperationLogPage,
  toUsagePeriod,
  type EntitlementStatusResponse,
  type OperationLogEntry,
  type SubscriptionStatus,
  type SubscriptionTier,
  type UsagePeriod,
} from './entitlementTypes';

export {
  TIER_CONFIG,
  type EntitlementStatusResponse,
  type FeatureAccessSummary,
  type OperationLogEntry,
  type OperationLogPage,
  type SubscriptionStatus,
  type SubscriptionTier,
  type UsagePeriod,
} from './entitlementTypes';

// The shared store owns the cross-scenario snapshot contract; Zustand below
// remains the BAS-specific reactive adapter for its historical UI selectors.
const sharedEntitlementStore = new EntitlementStore();

interface EntitlementState {
  userEmail: string;
  status: EntitlementStatusResponse | null;
  isLoading: boolean;
  error: string | null;
  lastFetched: Date | null;
  isOffline: boolean;
  pendingSyncCount: number | null;

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
  fetchPendingSyncCount: () => Promise<void>;
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

const toMaybeTier = (value: string | undefined): SubscriptionTier | undefined => {
  if (!value) return undefined;
  if (value === 'free' || value === 'solo' || value === 'pro' || value === 'studio' || value === 'business') {
    return value;
  }
  return undefined;
};

const toStatus = (proto: EntitlementStatus | undefined): EntitlementStatusResponse | null => {
  if (!proto) return null;
  const result = {
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
  sharedEntitlementStore.set({ identity: result.user_identity, tier: result.tier, status: result.status, features: result.features });
  return result;
};

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
  pendingSyncCount: null,

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

  fetchPendingSyncCount: async () => {
    try {
      const response = await fetch('/api/v1/monetization/outbox/pending');
      if (!response.ok) throw new Error(`pending usage request failed: ${response.status}`);
      const body = (await response.json()) as { pending?: unknown };
      const pending = typeof body.pending === 'number' && Number.isFinite(body.pending) ? body.pending : 0;
      set({ pendingSyncCount: Math.max(0, Math.floor(pending)) });
    } catch {
      set({ pendingSyncCount: null });
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
