import clsx from 'clsx';
import { Lightbulb, AlertTriangle, Clock } from 'lucide-react';
import type { MetricsSummary } from '@stores/uxMetricsStore';
import { getFrictionTypeLabel, type FrictionType } from '@stores/uxMetricsStore';

interface MetricsSummaryCardProps {
  summary: MetricsSummary;
  className?: string;
}

/**
 * Displays a summary of UX metrics with actionable recommendations.
 */
export function MetricsSummaryCard({ summary, className }: MetricsSummaryCardProps) {
  const hasHighFrictionSteps = summary.highFrictionSteps.length > 0;
  const hasRecommendations = summary.recommendedActions.length > 0;
  const hasSlowSteps = summary.slowestSteps.length > 0;
  const hasTopFrictionTypes = summary.topFrictionTypes.length > 0;

  if (!hasHighFrictionSteps && !hasRecommendations && !hasTopFrictionTypes) {
    return (
      <div className={clsx('bg-green-500/10 rounded-xl p-4 border border-green-500/20', className)}>
        <div className="flex items-center gap-2 text-green-400">
          <Lightbulb className="w-5 h-5" />
          <span className="font-medium">No issues detected</span>
        </div>
        <p className="text-sm text-green-300/80 mt-1">
          This execution shows no significant UX friction. Great job!
        </p>
      </div>
    );
  }

  return (
    <div className={clsx('bg-slate-900/50 rounded-xl p-4 border border-white/5', className)}>
      <h3 className="text-sm font-medium text-white flex items-center gap-2 mb-3">
        <Lightbulb className="w-4 h-4 text-flow-accent" />
        UX Insights
      </h3>

      <div className="space-y-4">
        {/* High Friction Steps */}
        {hasHighFrictionSteps && (
          <div>
            <div className="flex items-center gap-2 text-sm text-red-400 mb-1">
              <AlertTriangle className="w-4 h-4" />
              <span className="font-medium">High Friction Steps</span>
            </div>
            <p className="text-sm text-slate-400">
              Steps {summary.highFrictionSteps.map((s) => s + 1).join(', ')} show significant
              friction that may impact user experience.
            </p>
          </div>
        )}

        {/* Slowest Steps */}
        {hasSlowSteps && (
          <div>
            <div className="flex items-center gap-2 text-sm text-yellow-400 mb-1">
              <Clock className="w-4 h-4" />
              <span className="font-medium">Slowest Steps</span>
            </div>
            <p className="text-sm text-slate-400">
              Steps {summary.slowestSteps.map((s) => s + 1).join(', ')} took the longest to
              complete and may benefit from optimization.
            </p>
          </div>
        )}

        {/* Top Friction Types */}
        {hasTopFrictionTypes && (
          <div>
            <div className="text-xs uppercase tracking-wider text-slate-500 mb-2">
              Common Friction Types
            </div>
            <div className="flex flex-wrap gap-2">
              {summary.topFrictionTypes.map((type) => (
                <span
                  key={type}
                  className="px-2 py-1 text-xs bg-slate-800 rounded text-slate-300"
                >
                  {getFrictionTypeLabel(type as FrictionType)}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Recommendations */}
        {hasRecommendations && (
          <div className="border-t border-slate-800 pt-3 mt-3">
            <div className="text-xs uppercase tracking-wider text-slate-500 mb-2">
              Recommendations
            </div>
            <ul className="space-y-2">
              {summary.recommendedActions.map((action, idx) => (
                <li key={idx} className="flex items-start gap-2 text-sm text-slate-300">
                  <span className="text-flow-accent">-</span>
                  <span>{action}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}
