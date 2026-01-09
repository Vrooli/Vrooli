import { CheckCircle, Clock, AlertTriangle, XCircle, CloudOff, RefreshCw, Loader2, type LucideIcon } from 'lucide-react';
import { useEntitlementStore, STATUS_CONFIG, TIER_CONFIG } from '@stores/entitlementStore';
import type { SubscriptionStatus } from '@stores/entitlementStore';
import { TierBadge } from '@shared/ui';
import { formatDistanceToNow } from 'date-fns';

const STATUS_ICONS: Record<'check' | 'clock' | 'alert' | 'x', LucideIcon> = {
  check: CheckCircle,
  clock: Clock,
  alert: AlertTriangle,
  x: XCircle,
};

export function SubscriptionStatusCard() {
  const { status, isLoading, isOffline, lastFetched, refreshEntitlement } = useEntitlementStore();

  if (!status) {
    return (
      <div className="rounded-xl border border-gray-700 bg-gray-800/50 p-6">
        <div className="text-center text-gray-400">
          <p>No subscription status available.</p>
          <p className="text-sm mt-1">Enter your email to verify your subscription.</p>
        </div>
      </div>
    );
  }

  const statusConfig = STATUS_CONFIG[status.status as SubscriptionStatus] || STATUS_CONFIG.inactive;
  const StatusIcon = STATUS_ICONS[statusConfig.icon];
  const tierConfig = TIER_CONFIG[status.tier];

  return (
    <div className={`rounded-xl border ${tierConfig.borderColor} ${tierConfig.bgColor} p-6`}>
      <div className="flex items-start justify-between">
        <div className="space-y-3">
          {/* Tier Badge */}
          <div className="flex items-center gap-3">
            <TierBadge tier={status.tier} size="lg" />
            {isOffline && (
              <span className="flex items-center gap-1 text-xs text-amber-400 bg-amber-900/30 px-2 py-1 rounded-full">
                <CloudOff size={12} />
                Cached
              </span>
            )}
          </div>

          {/* Status */}
          <div className="flex items-center gap-2">
            <StatusIcon size={16} className={statusConfig.color} />
            <span className={`text-sm font-medium ${statusConfig.color}`}>
              {statusConfig.label}
            </span>
          </div>

          {/* User Identity */}
          {status.user_identity && (
            <p className="text-sm text-gray-400">
              {status.user_identity}
            </p>
          )}
        </div>

        {/* Refresh Button */}
        <button
          onClick={() => void refreshEntitlement()}
          disabled={isLoading}
          className="p-2 rounded-lg hover:bg-gray-700/50 transition-colors disabled:opacity-50"
          title="Refresh subscription status"
        >
          {isLoading ? (
            <Loader2 size={18} className="animate-spin text-gray-400" />
          ) : (
            <RefreshCw size={18} className="text-gray-400" />
          )}
        </button>
      </div>

      {/* Last Updated */}
      {lastFetched && (
        <div className="mt-4 pt-4 border-t border-gray-700/50">
          <p className="text-xs text-gray-500">
            Last updated {formatDistanceToNow(lastFetched, { addSuffix: true })}
          </p>
        </div>
      )}
    </div>
  );
}

export default SubscriptionStatusCard;
