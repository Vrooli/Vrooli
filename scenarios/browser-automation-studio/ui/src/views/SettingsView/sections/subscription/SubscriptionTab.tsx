import { useEffect, useState, useCallback } from 'react';
import { useEntitlementStore } from '@stores/entitlementStore';
import { useAuthStore } from '@stores/authStore';
import { AuthSection } from './AuthSection';
import { SubscriptionStatusCard } from './SubscriptionStatusCard';
import { UnifiedUsageSection } from './UnifiedUsageSection';
import { UsageHistorySection } from './UsageHistorySection';
import { OperationLogModal } from './OperationLogModal';
import { FeatureAccessList } from './FeatureAccessList';
import { UpgradePromptSection } from './UpgradePromptSection';
import { LoadingSpinner } from '@shared/ui';
import { PendingSyncBadge } from '@vrooli/react-component-library/MonetizationAccount/2.0.0';

export function SubscriptionTab() {
  const {
    status,
    isLoading,
    fetchStatus,
    getUserEmail,
    fetchPendingSyncCount,
    pendingSyncCount,
  } = useEntitlementStore();

  // Operation log modal state
  const [operationLogMonth, setOperationLogMonth] = useState<string | null>(null);

  const handleViewOperations = useCallback((month: string) => {
    setOperationLogMonth(month);
  }, []);

  const handleCloseOperationLog = useCallback(() => {
    setOperationLogMonth(null);
  }, []);

  // Get auth state for syncing with entitlements
  const { isAuthenticated, user: authUser } = useAuthStore();
  const { setUserEmail } = useEntitlementStore();

  // Fetch entitlement status on mount
  useEffect(() => {
    const init = async () => {
      // Get the stored email
      await getUserEmail();
      // Then fetch the status
      await fetchStatus();
    };
    void init();
  }, [fetchStatus, getUserEmail]);

  useEffect(() => {
    void fetchPendingSyncCount();
  }, [fetchPendingSyncCount]);

  // Sync entitlement store when auth user changes
  useEffect(() => {
    if (isAuthenticated && authUser?.email) {
      // Set the email in entitlement store to fetch subscription status
      void setUserEmail(authUser.email);
    }
  }, [isAuthenticated, authUser?.email, setUserEmail]);

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
      {/* Authentication */}
      <AuthSection />

      {pendingSyncCount !== null && <PendingSyncBadge pending={pendingSyncCount} />}

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
