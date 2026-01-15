import { useEffect, useMemo, useState, useCallback } from 'react';
import { useEntitlementStore, TIER_CONFIG, type SubscriptionTier } from '@stores/entitlementStore';
import { EmailInputSection } from './EmailInputSection';
import { SubscriptionStatusCard } from './SubscriptionStatusCard';
import { UnifiedUsageSection } from './UnifiedUsageSection';
import { UsageHistorySection } from './UsageHistorySection';
import { OperationLogModal } from './OperationLogModal';
import { FeatureAccessList } from './FeatureAccessList';
import { UpgradePromptSection } from './UpgradePromptSection';
import { LoadingSpinner } from '@shared/ui';
import { isDesktopEnvironment } from '@/lib/desktop/tray';

export function SubscriptionTab() {
  const { status, isLoading, fetchStatus, getUserEmail, overrideTier, setOverrideTier } = useEntitlementStore();
  const canOverrideTier = useMemo(() => !isDesktopEnvironment(), []);
  const activeOverride = overrideTier ?? null;

  // Operation log modal state
  const [operationLogMonth, setOperationLogMonth] = useState<string | null>(null);

  const handleViewOperations = useCallback((month: string) => {
    setOperationLogMonth(month);
  }, []);

  const handleCloseOperationLog = useCallback(() => {
    setOperationLogMonth(null);
  }, []);

  // Fetch entitlement status on mount
  useEffect(() => {
    const init = async () => {
      // First get the stored email
      await getUserEmail();
      // Then fetch the status
      await fetchStatus();
    };
    void init();
  }, [fetchStatus, getUserEmail]);

  // Show loading state on initial load
  if (isLoading && !status) {
    return (
      <div className="flex items-center justify-center py-12">
        <LoadingSpinner size={32} />
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {canOverrideTier && (
        <div className="rounded-xl border border-amber-500/40 bg-amber-950/20 p-5">
          <div className="flex items-center justify-between gap-4">
            <div>
              <h3 className="text-sm font-semibold text-amber-200">Local Subscription Override</h3>
              <p className="text-xs text-amber-200/70">
                Tier 1 override for testing entitlements without the billing service.
              </p>
            </div>
            <div className="flex items-center gap-3">
              <select
                value={activeOverride ?? ''}
                onChange={(event) => void setOverrideTier((event.target.value || null) as SubscriptionTier | null)}
                className="rounded-lg border border-amber-500/40 bg-amber-950/40 px-3 py-2 text-xs text-amber-100 focus:outline-none focus:ring-2 focus:ring-amber-400/40"
              >
                <option value="">Use live subscription</option>
                {Object.entries(TIER_CONFIG).map(([tier, config]) => (
                  <option key={tier} value={tier}>
                    {config.label}
                  </option>
                ))}
              </select>
              {activeOverride && (
                <button
                  type="button"
                  onClick={() => void setOverrideTier(null)}
                  className="rounded-lg border border-amber-500/40 px-3 py-2 text-xs text-amber-100 hover:bg-amber-950/60"
                >
                  Clear
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Email Input */}
      <EmailInputSection />

      {/* Status Card - only show if we have status */}
      {status && <SubscriptionStatusCard />}

      {/* Unified Usage - shows both AI credits and executions */}
      {status && <UnifiedUsageSection />}

      {/* Usage History - banking-style period navigation */}
      {status && <UsageHistorySection onViewOperations={handleViewOperations} />}

      {/* Feature Access List - only show if we have status */}
      {status && <FeatureAccessList />}

      {/* Upgrade Prompt - component handles its own visibility */}
      <UpgradePromptSection />

      {/* Operation Log Modal */}
      <OperationLogModal
        isOpen={operationLogMonth !== null}
        onClose={handleCloseOperationLog}
        month={operationLogMonth ?? ''}
      />
    </div>
  );
}

export default SubscriptionTab;
