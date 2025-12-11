import { useEffect } from 'react';
import { useEntitlementStore } from '@stores/entitlementStore';
import { EmailInputSection } from './EmailInputSection';
import { SubscriptionStatusCard } from './SubscriptionStatusCard';
import { UsageMeterSection } from './UsageMeterSection';
import { FeatureAccessList } from './FeatureAccessList';
import { UpgradePromptSection } from './UpgradePromptSection';
import { LoadingSpinner } from '@shared/ui';

export function SubscriptionTab() {
  const { status, isLoading, fetchStatus, getUserEmail } = useEntitlementStore();

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

  // Check if entitlements are disabled
  if (status && !status.entitlements_enabled) {
    return (
      <div className="rounded-xl border border-gray-700 bg-gray-800/50 p-6 text-center">
        <p className="text-gray-400">
          Subscription management is not enabled for this installation.
        </p>
        <p className="text-sm text-gray-500 mt-2">
          All features are available without restrictions.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      {/* Email Input */}
      <EmailInputSection />

      {/* Status Card - only show if we have status */}
      {status && <SubscriptionStatusCard />}

      {/* Usage Meter - only show if we have status */}
      {status && <UsageMeterSection />}

      {/* Feature Access List - only show if we have status */}
      {status && <FeatureAccessList />}

      {/* Upgrade Prompt - component handles its own visibility */}
      <UpgradePromptSection />
    </div>
  );
}

export default SubscriptionTab;
