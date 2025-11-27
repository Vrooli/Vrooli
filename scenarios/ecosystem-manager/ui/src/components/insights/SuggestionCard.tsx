import { cn } from '@/lib/utils';
import type { Suggestion, SuggestionPriority, SuggestionType } from '@/types/api';
import { CheckCircle2, Clock, Code, FileText, Settings, XCircle } from 'lucide-react';
import { useState } from 'react';

interface SuggestionCardProps {
  suggestion: Suggestion;
  onApply?: (suggestionId: string) => void;
  onReject?: (suggestionId: string) => void;
  isApplying?: boolean;
  className?: string;
}

const SUGGESTION_TYPE_INFO: Record<
  SuggestionType,
  { icon: typeof FileText; label: string; color: string }
> = {
  prompt: {
    icon: FileText,
    label: 'Prompt',
    color: 'text-blue-400',
  },
  timeout: {
    icon: Clock,
    label: 'Timeout',
    color: 'text-orange-400',
  },
  code: {
    icon: Code,
    label: 'Code',
    color: 'text-purple-400',
  },
  autosteer_profile: {
    icon: Settings,
    label: 'Auto Steer',
    color: 'text-emerald-400',
  },
};

const PRIORITY_STYLES: Record<SuggestionPriority, string> = {
  critical: 'bg-red-500/15 border-red-400/50',
  high: 'bg-orange-500/15 border-orange-400/50',
  medium: 'bg-amber-500/15 border-amber-400/50',
  low: 'bg-slate-700/30 border-slate-600/50',
};

const PRIORITY_BADGE_STYLES: Record<SuggestionPriority, string> = {
  critical: 'bg-red-500/20 text-red-100 border-red-400/30',
  high: 'bg-orange-500/20 text-orange-100 border-orange-400/30',
  medium: 'bg-amber-500/20 text-amber-100 border-amber-400/30',
  low: 'bg-slate-600/20 text-slate-100 border-slate-500/30',
};

export function SuggestionCard({
  suggestion,
  onApply,
  onReject,
  isApplying,
  className,
}: SuggestionCardProps) {
  const [showDetails, setShowDetails] = useState(false);
  const typeInfo = SUGGESTION_TYPE_INFO[suggestion.type];
  const Icon = typeInfo.icon;

  const isApplied = suggestion.status === 'applied';
  const isRejected = suggestion.status === 'rejected';
  const isPending = suggestion.status === 'pending';

  return (
    <div
      className={cn(
        'rounded-lg p-4 border transition-all',
        PRIORITY_STYLES[suggestion.priority],
        (isApplied || isRejected) && 'opacity-60',
        className
      )}
    >
      <div className="flex items-start gap-3 mb-3">
        <Icon className={cn('w-5 h-5 mt-0.5 flex-shrink-0', typeInfo.color)} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1 flex-wrap">
            <span className="text-xs font-medium text-slate-300">{typeInfo.label}</span>
            <span
              className={cn(
                'text-xs px-2 py-0.5 rounded border uppercase',
                PRIORITY_BADGE_STYLES[suggestion.priority]
              )}
            >
              {suggestion.priority}
            </span>
            {isApplied && (
              <span className="text-xs px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-100 border border-emerald-400/30 flex items-center gap-1">
                <CheckCircle2 className="w-3 h-3" />
                Applied
              </span>
            )}
            {isRejected && (
              <span className="text-xs px-2 py-0.5 rounded bg-slate-600/20 text-slate-300 border border-slate-500/30 flex items-center gap-1">
                <XCircle className="w-3 h-3" />
                Rejected
              </span>
            )}
          </div>
          <h4 className="text-sm font-semibold text-slate-100 mb-2">{suggestion.title}</h4>
          <p className="text-sm text-slate-300 leading-relaxed">{suggestion.description}</p>
        </div>
      </div>

      <div className="mt-3 pt-3 border-t border-slate-700/50">
        <div className="flex items-center justify-between mb-2">
          <div className="text-xs text-slate-400">
            <span className="font-medium">Impact:</span> {suggestion.impact.success_rate_improvement}
            {suggestion.impact.time_reduction && ` • ${suggestion.impact.time_reduction}`}
            <span className="ml-1 text-slate-500">
              ({suggestion.impact.confidence} confidence)
            </span>
          </div>
          <button
            onClick={() => setShowDetails(!showDetails)}
            className="text-xs text-blue-400 hover:text-blue-300 transition-colors"
          >
            {showDetails ? 'Hide' : 'Show'} Changes ({suggestion.changes.length})
          </button>
        </div>

        {showDetails && (
          <div className="mt-3 space-y-2">
            <div className="text-xs text-slate-400 mb-1">Proposed Changes:</div>
            {suggestion.changes.map((change, idx) => (
              <div
                key={idx}
                className="bg-slate-900/50 rounded p-2 border border-slate-700/30 text-xs"
              >
                <div className="flex items-center gap-2 mb-1">
                  <span className="font-mono text-blue-400">{change.file}</span>
                  <span className="text-slate-500">({change.type})</span>
                </div>
                <div className="text-slate-300">{change.description}</div>
                {change.config_path && (
                  <div className="mt-1 text-slate-400">
                    Path: <span className="font-mono">{change.config_path}</span> ={' '}
                    <span className="text-emerald-400">{JSON.stringify(change.config_value)}</span>
                  </div>
                )}
              </div>
            ))}
            <div className="mt-2 pt-2 border-t border-slate-700/30">
              <div className="text-xs text-slate-400 mb-1">Rationale:</div>
              <p className="text-xs text-slate-300 italic">{suggestion.impact.rationale}</p>
            </div>
          </div>
        )}
      </div>

      {isPending && (onApply || onReject) && (
        <div className="mt-3 pt-3 border-t border-slate-700/50 flex items-center gap-2">
          {onApply && (
            <button
              onClick={() => onApply(suggestion.id)}
              disabled={isApplying}
              className={cn(
                'flex-1 px-3 py-1.5 rounded text-xs font-medium transition-all',
                'bg-emerald-600 hover:bg-emerald-500 text-white',
                'disabled:opacity-50 disabled:cursor-not-allowed'
              )}
            >
              {isApplying ? 'Applying...' : 'Apply Changes'}
            </button>
          )}
          {onReject && (
            <button
              onClick={() => onReject(suggestion.id)}
              disabled={isApplying}
              className={cn(
                'px-3 py-1.5 rounded text-xs font-medium transition-all',
                'bg-slate-700 hover:bg-slate-600 text-slate-200',
                'disabled:opacity-50 disabled:cursor-not-allowed'
              )}
            >
              Dismiss
            </button>
          )}
        </div>
      )}

      {isApplied && suggestion.applied_at && (
        <div className="mt-2 text-xs text-slate-500 italic">
          Applied on {new Date(suggestion.applied_at).toLocaleString()}
        </div>
      )}
    </div>
  );
}
