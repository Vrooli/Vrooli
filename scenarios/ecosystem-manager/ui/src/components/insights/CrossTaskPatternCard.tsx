/**
 * CrossTaskPatternCard
 * Displays a pattern that affects multiple tasks with task breakdown
 */

import { AlertTriangle, FileText } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import type { CrossTaskPattern } from '@/types/api';

interface CrossTaskPatternCardProps {
  pattern: CrossTaskPattern;
  className?: string;
}

export function CrossTaskPatternCard({ pattern, className }: CrossTaskPatternCardProps) {
  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'bg-red-500/20 text-red-100 border-red-400/60';
      case 'high':
        return 'bg-orange-500/20 text-orange-100 border-orange-400/60';
      case 'medium':
        return 'bg-yellow-500/20 text-yellow-100 border-yellow-400/60';
      case 'low':
        return 'bg-blue-500/20 text-blue-100 border-blue-400/60';
      default:
        return 'bg-slate-500/20 text-slate-200 border-slate-500/40';
    }
  };

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'timeout':
        return '⏱️';
      case 'failure_mode':
        return '❌';
      case 'rate_limit':
        return '🚦';
      case 'stuck_state':
        return '🔄';
      default:
        return '⚠️';
    }
  };

  return (
    <Card className={`bg-slate-800/50 border-white/10 ${className}`}>
      <div className="p-4 space-y-3">
        {/* Header */}
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-start gap-2 flex-1">
            <span className="text-lg mt-0.5">{getTypeIcon(pattern.type)}</span>
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-1">
                <Badge variant="outline" className={getSeverityColor(pattern.severity)}>
                  {pattern.severity}
                </Badge>
                <Badge variant="outline" className="text-xs">
                  {pattern.type.replace('_', ' ')}
                </Badge>
              </div>
              <p className="text-sm font-medium text-white">{pattern.description}</p>
            </div>
          </div>
          <div className="flex items-center gap-1 text-xs text-slate-400">
            <AlertTriangle className="h-3 w-3" />
            <span>{pattern.frequency}x</span>
          </div>
        </div>

        {/* Affected Tasks */}
        <div className="space-y-1">
          <div className="text-xs font-medium text-slate-400 flex items-center gap-1">
            <FileText className="h-3 w-3" />
            Affects {pattern.affected_tasks.length} task{pattern.affected_tasks.length !== 1 ? 's' : ''}
          </div>
          <div className="flex flex-wrap gap-1.5">
            {pattern.affected_tasks.slice(0, 5).map((taskId) => (
              <span
                key={taskId}
                className="px-2 py-1 bg-slate-900/50 border border-white/5 rounded text-xs text-slate-300 font-mono"
              >
                {taskId.length > 30 ? `${taskId.slice(0, 30)}...` : taskId}
              </span>
            ))}
            {pattern.affected_tasks.length > 5 && (
              <span className="px-2 py-1 text-xs text-slate-500">
                +{pattern.affected_tasks.length - 5} more
              </span>
            )}
          </div>
        </div>

        {/* Task Types */}
        {pattern.task_types && pattern.task_types.length > 0 && (
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <span>Task types:</span>
            {pattern.task_types.map((type) => (
              <Badge key={type} variant="outline" className="text-xs">
                {type}
              </Badge>
            ))}
          </div>
        )}

        {/* Evidence */}
        {pattern.evidence && pattern.evidence.length > 0 && (
          <details className="group">
            <summary className="text-xs text-slate-400 cursor-pointer hover:text-slate-300 transition-colors">
              View evidence ({pattern.evidence.length} items)
            </summary>
            <div className="mt-2 space-y-1.5 pl-3 border-l-2 border-slate-700">
              {pattern.evidence.slice(0, 3).map((evidence, idx) => (
                <div
                  key={idx}
                  className="text-xs text-slate-300 bg-slate-900/30 p-2 rounded font-mono"
                >
                  {evidence}
                </div>
              ))}
              {pattern.evidence.length > 3 && (
                <p className="text-xs text-slate-500">+{pattern.evidence.length - 3} more...</p>
              )}
            </div>
          </details>
        )}
      </div>
    </Card>
  );
}
