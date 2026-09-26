import * as MonetizationAccount from '@components/MonetizationAccount';
import { useEntitlementStore } from '@stores/entitlementStore';

/** BAS adapter for the RCL subscription primitive and the shared lease state. */
export function SubscriptionStatusCard() {
  const { status, refreshEntitlement } = useEntitlementStore();
  if (!status) return null;

  return (
    <div className="space-y-2">
      <MonetizationAccount.SubscriptionStatusCard
        plan={status.tier}
        status={status.status}
        credits={status.monthly_remaining}
        label="credits remaining"
      />
      <button type="button" onClick={() => void refreshEntitlement()} className="text-sm text-flow-accent">
        Refresh subscription status
      </button>
    </div>
  );
}

export default SubscriptionStatusCard;
