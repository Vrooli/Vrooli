import { useCallback, useEffect } from 'react';
import {
  useEntitlementStore,
  useCanExecuteWorkflow,
  useCanUseAI,
  useCanUseRecording,
  useRequiresWatermark,
  useIsEntitlementsEnabled,
  useCurrentTier,
} from '@stores/entitlementStore';

/**
 * Hook to initialize entitlement state on app mount.
 * Call this once in App.tsx to ensure entitlements are fetched on startup.
 */
export function useEntitlementInit() {
  const { fetchStatus, getUserEmail } = useEntitlementStore();

  useEffect(() => {
    const init = async () => {
      // First try to get the stored email
      await getUserEmail();
      // Then fetch the entitlement status
      await fetchStatus();
    };
    void init();
  }, [fetchStatus, getUserEmail]);
}

/**
 * Hook to check if user can execute a workflow before starting.
 * Returns { canExecute, showLimitModal, closeLimitModal, refreshAfterExecution }
 */
export function useExecutionEntitlement() {
  const { status, fetchStatus } = useEntitlementStore();
  const canExecute = useCanExecuteWorkflow();
  const entitlementsEnabled = useIsEntitlementsEnabled();

  // Refresh entitlement after execution to update usage count
  const refreshAfterExecution = useCallback(async () => {
    if (entitlementsEnabled) {
      await fetchStatus();
    }
  }, [entitlementsEnabled, fetchStatus]);

  return {
    canExecute,
    monthlyUsed: status?.monthly_used ?? 0,
    monthlyLimit: status?.monthly_limit ?? -1,
    monthlyRemaining: status?.monthly_remaining ?? -1,
    refreshAfterExecution,
  };
}

/**
 * Hook to check if user can use AI features.
 */
export function useAIEntitlement() {
  const canUseAI = useCanUseAI();
  const entitlementsEnabled = useIsEntitlementsEnabled();

  return {
    canUseAI,
    // If entitlements disabled, always allow
    isGated: entitlementsEnabled && !canUseAI,
  };
}

/**
 * Hook to check if user can use recording features.
 */
export function useRecordingEntitlement() {
  const canUseRecording = useCanUseRecording();
  const entitlementsEnabled = useIsEntitlementsEnabled();

  return {
    canUseRecording,
    isGated: entitlementsEnabled && !canUseRecording,
  };
}

/**
 * Hook to check if exports require watermark.
 */
export function useWatermarkEntitlement() {
  const requiresWatermark = useRequiresWatermark();
  const entitlementsEnabled = useIsEntitlementsEnabled();

  return {
    requiresWatermark,
    // If entitlements disabled, no watermark
    showWatermark: entitlementsEnabled && requiresWatermark,
  };
}

// Re-export store hooks for convenience
export {
  useCanExecuteWorkflow,
  useCanUseAI,
  useCanUseRecording,
  useRequiresWatermark,
  useIsEntitlementsEnabled,
  useCurrentTier,
};
