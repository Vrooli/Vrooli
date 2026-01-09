import { cn } from '@/lib/utils';
import type { ExecutionStatistics } from '@/types/api';

interface ExecutionStatsProps {
  stats: ExecutionStatistics;
  className?: string;
}

export function ExecutionStats({ stats, className }: ExecutionStatsProps) {
  const successRate = Math.round(stats.success_rate * 100);
  const successRateColor =
    successRate >= 75
      ? 'text-emerald-400'
      : successRate >= 50
        ? 'text-amber-400'
        : 'text-red-400';

  return (
    <div className={cn('grid grid-cols-2 md:grid-cols-4 gap-4', className)}>
      <div className="bg-slate-800/50 rounded-lg p-4 border border-slate-700/50">
        <div className="text-xs text-slate-400 mb-1">Total Executions</div>
        <div className="text-2xl font-semibold text-slate-100">{stats.total_executions}</div>
      </div>

      <div className="bg-slate-800/50 rounded-lg p-4 border border-slate-700/50">
        <div className="text-xs text-slate-400 mb-1">Success Rate</div>
        <div className={cn('text-2xl font-semibold', successRateColor)}>{successRate}%</div>
      </div>

      <div className="bg-slate-800/50 rounded-lg p-4 border border-slate-700/50">
        <div className="text-xs text-slate-400 mb-1">Avg Duration</div>
        <div className="text-2xl font-semibold text-slate-100">{stats.avg_duration}</div>
      </div>

      <div className="bg-slate-800/50 rounded-lg p-4 border border-slate-700/50">
        <div className="text-xs text-slate-400 mb-1">Most Common Exit</div>
        <div className="text-sm font-medium text-slate-100 truncate">
          {stats.most_common_exit_reason || 'N/A'}
        </div>
      </div>

      <div className="bg-slate-800/50 rounded-lg p-3 border border-slate-700/50">
        <div className="flex items-center justify-between">
          <span className="text-xs text-slate-400">Completed</span>
          <span className="text-lg font-semibold text-emerald-400">{stats.success_count}</span>
        </div>
      </div>

      <div className="bg-slate-800/50 rounded-lg p-3 border border-slate-700/50">
        <div className="flex items-center justify-between">
          <span className="text-xs text-slate-400">Failed</span>
          <span className="text-lg font-semibold text-red-400">{stats.failure_count}</span>
        </div>
      </div>

      <div className="bg-slate-800/50 rounded-lg p-3 border border-slate-700/50">
        <div className="flex items-center justify-between">
          <span className="text-xs text-slate-400">Timeout</span>
          <span className="text-lg font-semibold text-orange-400">{stats.timeout_count}</span>
        </div>
      </div>

      <div className="bg-slate-800/50 rounded-lg p-3 border border-slate-700/50">
        <div className="flex items-center justify-between">
          <span className="text-xs text-slate-400">Rate Limited</span>
          <span className="text-lg font-semibold text-amber-400">{stats.rate_limit_count}</span>
        </div>
      </div>
    </div>
  );
}
