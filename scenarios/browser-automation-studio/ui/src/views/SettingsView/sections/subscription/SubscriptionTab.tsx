import { useEffect, useRef, useState, useCallback } from 'react';
import { useEntitlementStore } from '@stores/entitlementStore';
import { useAuthStore } from '@stores/authStore';
import { UnifiedUsageSection } from './UnifiedUsageSection';
import { UsageHistorySection } from './UsageHistorySection';
import { OperationLogModal } from './OperationLogModal';
import { FeatureAccessList } from './FeatureAccessList';
import { UpgradePromptSection } from './UpgradePromptSection';
import { LoadingSpinner } from '@shared/ui';
import { MonetizationAccountPanel } from '@vrooli/react-component-library/MonetizationAccount/2.0.2';
import { resolveUpgradeDestination } from '@shared/upgradeDestination';

function statusLabel(status: string | undefined, configured: boolean): string {
  if (status === 'active') return 'Active';
  if (status === 'trialing') return 'Trial';
  if (status === 'past_due') return 'Payment needs attention';
  if (status === 'canceled') return 'Canceled';
  if (status === 'inactive') return 'Inactive';
  return configured ? 'Connected' : 'Not signed in';
}

function openHostedAccount(path: string): void {
  window.open(resolveUpgradeDestination(path), '_blank', 'noopener,noreferrer');
}

export function SubscriptionTab() {
  const {
    status,
    isLoading,
    error: entitlementError,
    fetchStatus,
    refreshEntitlement,
  } = useEntitlementStore();

  // Operation log modal state
  const [operationLogMonth, setOperationLogMonth] = useState<string | null>(null);

  const handleViewOperations = useCallback((month: string) => {
    setOperationLogMonth(month);
  }, []);

  const handleCloseOperationLog = useCallback(() => {
    setOperationLogMonth(null);
  }, []);

  const auth = useAuthStore();
  const { isAuthenticated, signIn, signOut, error: authError } = auth;
  const previousAuthState = useRef(isAuthenticated);

  // Fetch entitlement status on mount
  useEffect(() => {
    const init = async () => {
      await fetchStatus();
    };
    void init();
  }, [fetchStatus]);

  // Re-fetch when the shared session changes. The server resolves account
  // identity from the LPBS authority instead of accepting a UI-supplied email.
  useEffect(() => {
    if (previousAuthState.current !== isAuthenticated) void fetchStatus();
    previousAuthState.current = isAuthenticated;
  }, [isAuthenticated, fetchStatus]);

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
      <MonetizationAccountPanel
        configured={isAuthenticated}
        plan={status?.tier ?? 'free'}
        status={status?.status ?? 'inactive'}
        statusLabel={statusLabel(status?.status, isAuthenticated)}
        creditsLabel={status ? (status.ai_credits_limit < 0 ? 'Unlimited' : `${status.ai_credits_remaining} remaining`) : 'Unavailable'}
        busyAction={auth.isLoading ? (auth.isAuthenticated ? 'sign-out' : 'sign-in') : isLoading ? 'refresh' : null}
        error={authError || entitlementError || undefined}
        onSignIn={() => void signIn()}
        onSignOut={() => void signOut()}
        onRefresh={() => void refreshEntitlement()}
        onManageAccount={() => openHostedAccount('/account')}
        onComparePlans={() => openHostedAccount('/pricing')}
      />

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
