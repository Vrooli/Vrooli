import { Sparkles, Zap, Calendar, TrendingUp } from 'lucide-react';
import { useEntitlementStore, useAICredits } from '@stores/entitlementStore';
import { UsageMeter } from '@shared/ui';
import { formatDistanceToNow } from 'date-fns';

export function UnifiedUsageSection() {
  const { status } = useEntitlementStore();
  const aiCredits = useAICredits();

  if (!status) {
    return null;
  }

  const executionIsUnlimited = status.monthly_limit === -1;
  const aiIsUnlimited = aiCredits.isUnlimited;

  // Calculate days until reset
  const resetDate = aiCredits.resetDate ? new Date(aiCredits.resetDate) : null;
  const daysUntilReset = resetDate ? formatDistanceToNow(resetDate, { addSuffix: false }) : null;

  // Calculate usage percentages for warnings
  const executionPercentage = executionIsUnlimited ? 0 : Math.round((status.monthly_used / status.monthly_limit) * 100);
  const aiPercentage = aiCredits.percentUsed;

  const showExecutionWarning = !executionIsUnlimited && executionPercentage >= 80;
  const showAIWarning = aiCredits.hasAccess && !aiIsUnlimited && aiPercentage >= 80;
  const executionLimitReached = !executionIsUnlimited && status.monthly_remaining === 0;
  const aiLimitReached = aiCredits.hasAccess && !aiIsUnlimited && aiCredits.remaining === 0;

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-lg font-medium text-surface flex items-center gap-2">
          <TrendingUp size={20} className="text-flow-accent" />
          Current Period Usage
        </h3>
        <p className="text-sm text-gray-400 mt-1">
          Track your credits and workflow executions for the current billing period.
        </p>
      </div>

      <div className="rounded-xl border border-gray-700 bg-gray-800/50 p-6">
        <div className="space-y-6">
          {/* Usage Meters - Side by Side */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* AI Credits Meter */}
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <Sparkles size={16} className="text-purple-400" />
                <span className="text-sm font-medium text-gray-300">AI Credits</span>
              </div>
              {aiCredits.hasAccess ? (
                <>
                  <UsageMeter
                    used={aiCredits.used}
                    limit={aiCredits.limit}
                    size="md"
                  />
                  <div className="flex justify-between text-xs text-gray-400">
                    <span>{aiCredits.used} used</span>
                    <span>{aiIsUnlimited ? 'Unlimited' : `${aiCredits.remaining} remaining`}</span>
                  </div>
                </>
              ) : (
                <div className="text-sm text-gray-500 italic">
                  AI features not available on your plan
                </div>
              )}
            </div>

            {/* Workflow Executions Meter */}
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <Zap size={16} className="text-amber-400" />
                <span className="text-sm font-medium text-gray-300">Workflow Executions</span>
              </div>
              <UsageMeter
                used={status.monthly_used}
                limit={status.monthly_limit}
                size="md"
              />
              <div className="flex justify-between text-xs text-gray-400">
                <span>{status.monthly_used} used</span>
                <span>{executionIsUnlimited ? 'Unlimited' : `${status.monthly_remaining} remaining`}</span>
              </div>
            </div>
          </div>

          {/* Reset Date */}
          {resetDate && (
            <div className="flex items-center justify-center gap-2 text-sm text-gray-400 pt-2 border-t border-gray-700">
              <Calendar size={14} />
              <span>
                Resets {daysUntilReset} ({resetDate.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })})
              </span>
            </div>
          )}

          {/* Warning Messages */}
          {executionLimitReached && (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-red-900/20 border border-red-800/50 text-red-400 text-sm">
              <Zap size={16} />
              You&apos;ve reached your execution limit. Upgrade to continue executing workflows.
            </div>
          )}

          {aiLimitReached && (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-red-900/20 border border-red-800/50 text-red-400 text-sm">
              <Sparkles size={16} />
              You&apos;ve used all your AI credits. Upgrade for more AI features.
            </div>
          )}

          {showExecutionWarning && !executionLimitReached && (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-amber-900/20 border border-amber-800/50 text-amber-400 text-sm">
              <Zap size={16} />
              You&apos;re approaching your execution limit ({executionPercentage}% used).
            </div>
          )}

          {showAIWarning && !aiLimitReached && (
            <div className="flex items-center gap-2 p-3 rounded-lg bg-amber-900/20 border border-amber-800/50 text-amber-400 text-sm">
              <Sparkles size={16} />
              You&apos;re approaching your AI credit limit ({aiPercentage}% used).
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default UnifiedUsageSection;
