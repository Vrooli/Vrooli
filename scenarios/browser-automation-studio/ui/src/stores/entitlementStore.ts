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

// API response type from backend
export interface EntitlementStatusResponse {
  user_identity: string;
  status: SubscriptionStatus;
  tier: SubscriptionTier;
  is_active: boolean;
  features: string[];
  monthly_limit: number; // -1 for unlimited
  monthly_used: number;
  monthly_remaining: number; // -1 for unlimited
  requires_watermark: boolean;
  can_use_ai: boolean;
  can_use_recording: boolean;
  entitlements_enabled: boolean;
}

interface EntitlementState {
  // State
  userEmail: string;
  status: EntitlementStatusResponse | null;
  isLoading: boolean;
  error: string | null;
  lastFetched: Date | null;
  isOffline: boolean;

  // Actions
  fetchStatus: () => Promise<void>;
  setUserEmail: (email: string) => Promise<void>;
  clearUserEmail: () => Promise<void>;
  refreshEntitlement: () => Promise<void>;
  getUserEmail: () => Promise<string>;
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
  isLoading: false,
  error: null,
  lastFetched: null,
  isOffline: false,

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

export default useEntitlementStore;
