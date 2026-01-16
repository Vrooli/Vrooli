import { useEffect, useMemo, useState, useCallback } from 'react';
import { useEntitlementStore, TIER_CONFIG, type SubscriptionTier, type ApiSource } from '@stores/entitlementStore';
import { EmailInputSection } from './EmailInputSection';
import { SubscriptionStatusCard } from './SubscriptionStatusCard';
import { UnifiedUsageSection } from './UnifiedUsageSection';
import { UsageHistorySection } from './UsageHistorySection';
import { OperationLogModal } from './OperationLogModal';
import { FeatureAccessList } from './FeatureAccessList';
import { UpgradePromptSection } from './UpgradePromptSection';
import { LoadingSpinner } from '@shared/ui';
import { isDesktopEnvironment } from '@/lib/desktop/tray';

const API_SOURCE_OPTIONS: { value: ApiSource; label: string; description: string }[] = [
  { value: 'production', label: 'Production (vrooli.com)', description: 'Use live subscription from vrooli.com' },
  { value: 'local', label: 'Local (localhost)', description: 'Use local landing-page-business-suite API' },
  { value: 'disabled', label: 'Disabled (override only)', description: 'Skip API calls, use tier override only' },
];

export function SubscriptionTab() {
  const {
    status,
    isLoading,
    fetchStatus,
    getUserEmail,
    overrideTier,
    setOverrideTier,
    apiSource,
    localApiPort,
    setApiSource,
    getApiSource,
  } = useEntitlementStore();
  const canOverrideTier = useMemo(() => !isDesktopEnvironment(), []);
  const activeOverride = overrideTier ?? null;

  // Local port input state
  const [localPortInput, setLocalPortInput] = useState<string>(String(localApiPort));

  // Operation log modal state
  const [operationLogMonth, setOperationLogMonth] = useState<string | null>(null);

  const handleViewOperations = useCallback((month: string) => {
    setOperationLogMonth(month);
  }, []);

  const handleCloseOperationLog = useCallback(() => {
    setOperationLogMonth(null);
  }, []);

  // Fetch entitlement status and API source on mount
  useEffect(() => {
    const init = async () => {
      // Fetch API source first (for dev mode)
      await getApiSource();
      // Get the stored email
      await getUserEmail();
      // Then fetch the status
      await fetchStatus();
    };
    void init();
  }, [fetchStatus, getUserEmail, getApiSource]);

  // Sync local port input when localApiPort changes
  useEffect(() => {
    setLocalPortInput(String(localApiPort));
  }, [localApiPort]);

  // Handle API source change
  const handleApiSourceChange = useCallback(
    async (newSource: ApiSource) => {
      const port = newSource === 'local' ? parseInt(localPortInput, 10) || 15000 : undefined;
      await setApiSource(newSource, port);
    },
    [setApiSource, localPortInput]
  );

  // Handle local port blur (save when user finishes editing)
  const handleLocalPortBlur = useCallback(async () => {
    const port = parseInt(localPortInput, 10);
    if (port > 0 && port !== localApiPort && apiSource === 'local') {
      await setApiSource('local', port);
    }
  }, [localPortInput, localApiPort, apiSource, setApiSource]);

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
        <div className="rounded-xl border border-amber-500/40 bg-amber-950/20 p-5 space-y-4">
          <div>
            <h3 className="text-sm font-semibold text-amber-200">Development Settings</h3>
            <p className="text-xs text-amber-200/70">
              Tier 1 settings for testing entitlements and subscription integration.
            </p>
          </div>

          {/* API Source Selector */}
          <div className="flex flex-col sm:flex-row sm:items-center gap-3">
            <label className="text-xs text-amber-200 min-w-[100px]">API Source:</label>
            <div className="flex items-center gap-2 flex-wrap">
              <select
                value={apiSource}
                onChange={(event) => void handleApiSourceChange(event.target.value as ApiSource)}
                className="rounded-lg border border-amber-500/40 bg-amber-950/40 px-3 py-2 text-xs text-amber-100 focus:outline-none focus:ring-2 focus:ring-amber-400/40"
              >
                {API_SOURCE_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
              {apiSource === 'local' && (
                <div className="flex items-center gap-1">
                  <span className="text-xs text-amber-200/70">Port:</span>
                  <input
                    type="number"
                    value={localPortInput}
                    onChange={(e) => setLocalPortInput(e.target.value)}
                    onBlur={() => void handleLocalPortBlur()}
                    className="w-20 rounded-lg border border-amber-500/40 bg-amber-950/40 px-2 py-2 text-xs text-amber-100 focus:outline-none focus:ring-2 focus:ring-amber-400/40"
                    placeholder="15000"
                    min="1"
                    max="65535"
                  />
                </div>
              )}
            </div>
          </div>

          {/* Tier Override Selector */}
          <div className="flex flex-col sm:flex-row sm:items-center gap-3">
            <label className="text-xs text-amber-200 min-w-[100px]">Tier Override:</label>
            <div className="flex items-center gap-3">
              <select
                value={activeOverride ?? ''}
                onChange={(event) => void setOverrideTier((event.target.value || null) as SubscriptionTier | null)}
                className="rounded-lg border border-amber-500/40 bg-amber-950/40 px-3 py-2 text-xs text-amber-100 focus:outline-none focus:ring-2 focus:ring-amber-400/40"
              >
                <option value="">Use API subscription</option>
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

          {/* Current config summary */}
          <div className="text-xs text-amber-200/60 pt-2 border-t border-amber-500/20">
            {apiSource === 'disabled' && activeOverride
              ? `Using tier override: ${TIER_CONFIG[activeOverride]?.label || activeOverride}`
              : apiSource === 'local'
                ? `Checking localhost:${localPortInput} for subscription status`
                : 'Checking vrooli.com for subscription status'}
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
