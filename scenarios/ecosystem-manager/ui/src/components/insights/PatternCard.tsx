import { cn } from '@/lib/utils';
import type { Pattern, PatternSeverity, PatternType } from '@/types/api';
import { AlertCircle, AlertTriangle, Clock, Info, Zap } from 'lucide-react';

interface PatternCardProps {
  pattern: Pattern;
  className?: string;
}

const PATTERN_TYPE_INFO: Record<
  PatternType,
  { icon: typeof AlertCircle; label: string; color: string }
> = {
  failure_mode: {
    icon: AlertCircle,
    label: 'Failure Mode',
    color: 'text-red-400',
  },
  timeout: {
    icon: Clock,
    label: 'Timeout',
    color: 'text-orange-400',
  },
  rate_limit: {
    icon: Zap,
    label: 'Rate Limit',
    color: 'text-amber-400',
  },
  stuck_state: {
    icon: AlertTriangle,
    label: 'Stuck State',
    color: 'text-purple-400',
  },
};

const SEVERITY_STYLES: Record<PatternSeverity, string> = {
  critical: 'bg-red-500/15 border-red-400/50',
  high: 'bg-orange-500/15 border-orange-400/50',
  medium: 'bg-amber-500/15 border-amber-400/50',
  low: 'bg-blue-500/15 border-blue-400/50',
};

const SEVERITY_BADGE_STYLES: Record<PatternSeverity, string> = {
  critical: 'bg-red-500/20 text-red-100 border-red-400/30',
  high: 'bg-orange-500/20 text-orange-100 border-orange-400/30',
  medium: 'bg-amber-500/20 text-amber-100 border-amber-400/30',
  low: 'bg-blue-500/20 text-blue-100 border-blue-400/30',
};

export function PatternCard({ pattern, className }: PatternCardProps) {
  const typeInfo = PATTERN_TYPE_INFO[pattern.type] || {
    icon: Info,
    label: pattern.type,
    color: 'text-slate-400',
  };
  const Icon = typeInfo.icon;

  return (
    <div
      className={cn(
        'rounded-lg p-4 border transition-all',
        SEVERITY_STYLES[pattern.severity],
        className
      )}
    >
      <div className="flex items-start gap-3 mb-3">
        <Icon className={cn('w-5 h-5 mt-0.5 flex-shrink-0', typeInfo.color)} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-xs font-medium text-slate-300">{typeInfo.label}</span>
            <span
              className={cn(
                'text-xs px-2 py-0.5 rounded border uppercase',
                SEVERITY_BADGE_STYLES[pattern.severity]
              )}
            >
              {pattern.severity}
            </span>
            <span className="text-xs text-slate-400">
              {pattern.frequency} occurrence{pattern.frequency !== 1 ? 's' : ''}
            </span>
          </div>
          <p className="text-sm text-slate-100 leading-relaxed">{pattern.description}</p>
        </div>
      </div>

      {pattern.evidence && pattern.evidence.length > 0 && (
        <div className="mt-3 pt-3 border-t border-slate-700/50">
          <div className="text-xs text-slate-400 mb-2">Evidence:</div>
          <div className="space-y-1">
            {pattern.evidence.slice(0, 3).map((evidence, idx) => (
              <div
                key={idx}
                className="text-xs font-mono bg-slate-900/50 px-2 py-1 rounded border border-slate-700/30 text-slate-300 truncate"
              >
                {evidence}
              </div>
            ))}
            {pattern.evidence.length > 3 && (
              <div className="text-xs text-slate-500 italic">
                +{pattern.evidence.length - 3} more...
              </div>
            )}
          </div>
        </div>
      )}

      {pattern.examples && pattern.examples.length > 0 && (
        <div className="mt-2 pt-2 border-t border-slate-700/30">
          <div className="text-xs text-slate-400">
            Examples: {pattern.examples.slice(0, 3).join(', ')}
            {pattern.examples.length > 3 && ` +${pattern.examples.length - 3} more`}
          </div>
        </div>
      )}
    </div>
  );
}
