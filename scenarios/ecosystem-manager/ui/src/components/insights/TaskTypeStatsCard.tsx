/**
 * TaskTypeStatsCard
 * Displays aggregated statistics for a task type or operation
 */

import { TrendingUp, Clock, CheckCircle2, XCircle } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import type { TaskTypeStats } from '@/types/api';

interface TaskTypeStatsCardProps {
  label: string;
  stats: TaskTypeStats;
  className?: string;
}

export function TaskTypeStatsCard({ label, stats, className }: TaskTypeStatsCardProps) {
  const getSuccessRateColor = (rate: number) => {
    if (rate >= 80) return 'text-green-400';
    if (rate >= 60) return 'text-yellow-400';
    if (rate >= 40) return 'text-orange-400';
    return 'text-red-400';
  };

  const successRate = stats.success_rate ?? 0;
  const failureRate = 100 - successRate;

  return (
    <Card className={`bg-slate-800/50 border-white/10 ${className}`}>
      <div className="p-4 space-y-3">
        {/* Header */}
        <div className="flex items-center justify-between">
          <h4 className="text-sm font-semibold text-white">{label}</h4>
          <Badge variant="outline" className="text-xs">
            {stats.count} task{stats.count !== 1 ? 's' : ''}
          </Badge>
        </div>

        {/* Success Rate */}
        <div className="space-y-2">
          <div className="flex items-center justify-between text-xs">
            <span className="text-slate-400">Success Rate</span>
            <span className={`font-semibold ${getSuccessRateColor(successRate)}`}>
              {successRate.toFixed(1)}%
            </span>
          </div>
          <div className="w-full h-2 bg-slate-900/50 rounded-full overflow-hidden">
            <div
              className={`h-full transition-all ${
                successRate >= 80
                  ? 'bg-green-500'
                  : successRate >= 60
                    ? 'bg-yellow-500'
                    : successRate >= 40
                      ? 'bg-orange-500'
                      : 'bg-red-500'
              }`}
              style={{ width: `${successRate}%` }}
            />
          </div>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <div className="flex items-center gap-1 text-xs text-slate-400">
              <CheckCircle2 className="h-3 w-3" />
              <span>Success</span>
            </div>
            <div className="text-lg font-semibold text-green-400">
              {successRate.toFixed(0)}%
            </div>
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-1 text-xs text-slate-400">
              <XCircle className="h-3 w-3" />
              <span>Failure</span>
            </div>
            <div className="text-lg font-semibold text-red-400">
              {failureRate.toFixed(0)}%
            </div>
          </div>
        </div>

        {/* Average Duration */}
        <div className="pt-2 border-t border-white/5">
          <div className="flex items-center justify-between text-xs">
            <div className="flex items-center gap-1 text-slate-400">
              <Clock className="h-3 w-3" />
              <span>Avg Duration</span>
            </div>
            <span className="font-mono text-slate-200">{stats.avg_duration || '--'}</span>
          </div>
        </div>

        {/* Top Pattern */}
        {stats.top_pattern && (
          <div className="pt-2 border-t border-white/5">
            <div className="text-xs text-slate-400 mb-1 flex items-center gap-1">
              <TrendingUp className="h-3 w-3" />
              <span>Most Common Issue</span>
            </div>
            <p className="text-xs text-slate-300 bg-slate-900/30 p-2 rounded">
              {stats.top_pattern}
            </p>
          </div>
        )}
      </div>
    </Card>
  );
}
