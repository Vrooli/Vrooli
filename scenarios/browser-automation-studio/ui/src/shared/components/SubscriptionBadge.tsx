import { CloudOff } from 'lucide-react';
import { useEntitlementStore, useIsEntitlementsEnabled, TIER_CONFIG } from '@stores/entitlementStore';
import { TierBadge, Tooltip, UsageMeter } from '@shared/ui';

interface SubscriptionBadgeProps {
  onClick?: () => void;
}

export function SubscriptionBadge({ onClick }: SubscriptionBadgeProps) {
  const { status, isOffline } = useEntitlementStore();
  const entitlementsEnabled = useIsEntitlementsEnabled();

  // Don't render if entitlements are disabled or no status
  if (!entitlementsEnabled || !status) {
    return null;
  }

  const tierConfig = TIER_CONFIG[status.tier];
  const isUnlimited = status.monthly_limit === -1;

  const tooltipContent = (
    <div className="space-y-2 min-w-[180px]">
      <div className="flex items-center justify-between">
        <span className="text-gray-400">Tier:</span>
        <span className={`font-medium ${tierConfig.color}`}>{tierConfig.label}</span>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-gray-400">Status:</span>
        <span className={`font-medium ${status.is_active ? 'text-green-400' : 'text-gray-400'}`}>
          {status.is_active ? 'Active' : 'Inactive'}
        </span>
      </div>
      <div className="pt-2 border-t border-gray-700">
        <div className="text-xs text-gray-400 mb-1">Monthly Usage</div>
        <UsageMeter
          used={status.monthly_used}
          limit={status.monthly_limit}
          size="sm"
          showText={false}
        />
        <div className="text-xs text-gray-400 mt-1">
          {isUnlimited
            ? `${status.monthly_used} used (unlimited)`
            : `${status.monthly_used} / ${status.monthly_limit}`
          }
        </div>
      </div>
      {isOffline && (
        <div className="flex items-center gap-1 text-xs text-amber-400 pt-1">
          <CloudOff size={12} />
          <span>Cached (offline)</span>
        </div>
      )}
      <div className="text-xs text-gray-500 pt-1">
        Click to manage subscription
      </div>
    </div>
  );

  return (
    <Tooltip content={tooltipContent} position="bottom">
      <div className="relative">
        <TierBadge
          tier={status.tier}
          size="sm"
          showLabel={true}
          showIcon={false}
          onClick={onClick}
          className="cursor-pointer"
        />
        {isOffline && (
          <div className="absolute -top-0.5 -right-0.5 w-2 h-2 bg-amber-500 rounded-full" />
        )}
      </div>
    </Tooltip>
  );
}

export default SubscriptionBadge;
