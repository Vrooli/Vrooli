import { Zap, TrendingUp, Calendar } from 'lucide-react';
import { useEntitlementStore } from '@stores/entitlementStore';
import { UsageMeter } from '@shared/ui';

export function UsageMeterSection() {
  const { status } = useEntitlementStore();

  if (!status) {
    return null;
  }

  const isUnlimited = status.monthly_limit === -1;
  const usagePercentage = isUnlimited ? 0 : Math.round((status.monthly_used / status.monthly_limit) * 100);

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-medium text-surface flex items-center gap-2">
          <TrendingUp size={20} className="text-flow-accent" />
          Monthly Usage
        </h3>
        <p className="text-sm text-gray-400 mt-1">
          Track your workflow execution usage for the current billing period.
        </p>
      </div>

      <div className="rounded-xl border border-gray-700 bg-gray-800/50 p-6">
        <div className="space-y-6">
          {/* Usage Meter */}
          <UsageMeter
            used={status.monthly_used}
            limit={status.monthly_limit}
            label="Workflow Executions"
            size="lg"
          />

          {/* Usage Stats */}
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
            <div className="text-center p-3 rounded-lg bg-gray-900/50">
              <div className="flex items-center justify-center gap-1 text-gray-400 text-xs mb-1">
                <Zap size={12} />
                Used
              </div>
              <div className="text-2xl font-bold text-surface">
                {status.monthly_used}
              </div>
            </div>

            <div className="text-center p-3 rounded-lg bg-gray-900/50">
              <div className="flex items-center justify-center gap-1 text-gray-400 text-xs mb-1">
                <Calendar size={12} />
                Limit
              </div>
              <div className="text-2xl font-bold text-surface">
                {isUnlimited ? '∞' : status.monthly_limit}
              </div>
            </div>

            <div className="text-center p-3 rounded-lg bg-gray-900/50 col-span-2 sm:col-span-1">
              <div className="flex items-center justify-center gap-1 text-gray-400 text-xs mb-1">
                <TrendingUp size={12} />
                Remaining
              </div>
              <div className={`text-2xl font-bold ${
                isUnlimited ? 'text-green-400' :
                status.monthly_remaining === 0 ? 'text-red-400' :
                status.monthly_remaining <= 10 ? 'text-amber-400' :
                'text-surface'
              }`}>
                {isUnlimited ? '∞' : status.monthly_remaining}
              </div>
            </div>
          </div>

          {/* Warning Messages */}
          {!isUnlimited && status.monthly_remaining === 0 && (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-red-900/20 border border-red-800/50 text-red-400 text-sm">
              <Zap size={16} />
              You&apos;ve reached your monthly limit. Upgrade to continue executing workflows.
            </div>
          )}

          {!isUnlimited && status.monthly_remaining > 0 && usagePercentage >= 80 && (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-amber-900/20 border border-amber-800/50 text-amber-400 text-sm">
              <Zap size={16} />
              You&apos;re approaching your monthly limit ({usagePercentage}% used).
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default UsageMeterSection;
