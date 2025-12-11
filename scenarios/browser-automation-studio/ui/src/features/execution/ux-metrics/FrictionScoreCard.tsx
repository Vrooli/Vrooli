import clsx from 'clsx';
import { TrendingDown, TrendingUp, Minus } from 'lucide-react';
import { getFrictionScoreColor } from '@stores/uxMetricsStore';

interface FrictionScoreCardProps {
  score: number;
  trend?: 'improving' | 'degrading' | 'stable';
  className?: string;
}

/**
 * Displays the overall friction score with visual indicators.
 * Score ranges from 0 (no friction) to 100 (maximum friction).
 */
export function FrictionScoreCard({ score, trend, className }: FrictionScoreCardProps) {
  const scoreColorClass = getFrictionScoreColor(score);

  const getTrendIcon = () => {
    switch (trend) {
      case 'improving':
        return <TrendingDown className="w-4 h-4 text-green-400" />;
      case 'degrading':
        return <TrendingUp className="w-4 h-4 text-red-400" />;
      case 'stable':
      default:
        return <Minus className="w-4 h-4 text-slate-400" />;
    }
  };

  const getScoreLabel = () => {
    if (score >= 70) return 'High Friction';
    if (score >= 40) return 'Moderate Friction';
    if (score > 0) return 'Low Friction';
    return 'No Friction Detected';
  };

  return (
    <div
      className={clsx(
        'bg-slate-900/50 rounded-xl p-4 border border-white/5',
        className
      )}
    >
      <div className="flex items-center justify-between mb-1">
        <span className="text-xs uppercase tracking-wider text-slate-400">
          Friction Score
        </span>
        {trend && getTrendIcon()}
      </div>
      <div className={clsx('text-3xl font-bold', scoreColorClass)}>
        {score.toFixed(0)}
        <span className="text-lg text-slate-500">/100</span>
      </div>
      <div className="text-xs text-slate-500 mt-1">{getScoreLabel()}</div>
      {/* Progress bar visualization */}
      <div className="mt-3 h-2 bg-slate-800 rounded-full overflow-hidden">
        <div
          className={clsx(
            'h-full rounded-full transition-all duration-500',
            score >= 70 ? 'bg-red-500' : score >= 40 ? 'bg-yellow-500' : 'bg-green-500'
          )}
          style={{ width: `${Math.min(score, 100)}%` }}
        />
      </div>
    </div>
  );
}
