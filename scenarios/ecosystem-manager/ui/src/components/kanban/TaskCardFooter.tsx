/**
 * TaskCardFooter Component
 * Displays elapsed timer (for in-progress tasks), execution count, and action buttons
 */

import { Eye, Trash2, PlayCircle, CheckCircle2, Lock, Snowflake } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { ElapsedTimer } from './ElapsedTimer';
import type { Task } from '../../types/api';

interface TaskCardFooterProps {
  task: Task;
  onViewDetails?: () => void;
  onDelete?: () => void;
}

export function TaskCardFooter({ task, onViewDetails, onDelete }: TaskCardFooterProps) {
  const isInProgress = task.status === 'in-progress';
  const hasProcess = task.current_process;
  const executionCount = task.execution_count || 0;
  const completionCount = task.completion_count ?? 0;
  const showCooldown = (task.status === 'completed' || task.status === 'failed') && !!task.cooldown_until;
  const showAutoRequeueDisabled =
    task.status !== 'pending' &&
    task.status !== 'in-progress' &&
    task.auto_requeue === false &&
    !showCooldown;

  return (
    <div className="flex items-center justify-between gap-2 mt-3 pt-3 border-t border-border/60">
      {/* Left side: Timer and execution count */}
      <div className="flex items-center gap-3">
        {isInProgress && hasProcess && task.current_process?.start_time && (
          <ElapsedTimer startTime={task.current_process.start_time} />
        )}
        {showCooldown && <CooldownCountdown until={task.cooldown_until!} />}
        {showAutoRequeueDisabled && <AutoRequeueLocked />}
        <div
          className="flex items-center gap-1.5 text-xs text-muted-foreground"
          title="Completion count"
        >
          <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" />
          <span>{completionCount}</span>
        </div>
        {executionCount > 0 && (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground" title="Execution count">
            <PlayCircle className="h-3.5 w-3.5" />
            <span>{executionCount}</span>
          </div>
        )}
      </div>

      {/* Right side: Action buttons */}
      <div className="flex items-center gap-1">
        {onViewDetails && (
          <button
            onClick={(event) => {
              event.stopPropagation();
              onViewDetails();
            }}
            className="p-1.5 rounded hover:bg-foreground/5 text-muted-foreground hover:text-foreground transition-colors"
            title="View details"
            aria-label={`View details for task ${task.title || task.id}`}
          >
            <Eye className="h-4 w-4" />
          </button>
        )}
        {onDelete && (
          <button
            onClick={(event) => {
              event.stopPropagation();
              onDelete();
            }}
            className="p-1.5 rounded hover:bg-red-500/10 text-muted-foreground hover:text-red-500 transition-colors"
            title="Delete task"
            aria-label={`Delete task ${task.title || task.id}`}
          >
            <Trash2 className="h-4 w-4" />
          </button>
        )}
      </div>
    </div>
  );
}

function AutoRequeueLocked() {
  return (
    <div className="flex items-center gap-1.5 text-xs text-amber-600" title="Auto-enqueue disabled">
      <Lock className="h-3.5 w-3.5" />
      <span className="tabular-nums">Locked</span>
    </div>
  );
}

function CooldownCountdown({ until }: { until: string }) {
  const target = useMemo(() => {
    const parsed = new Date(until);
    return Number.isNaN(parsed.getTime()) ? null : parsed;
  }, [until]);

  const computeRemaining = () => {
    if (!target) return 0;
    return Math.max(0, Math.floor((target.getTime() - Date.now()) / 1000));
  };

  const [remaining, setRemaining] = useState<number>(() => computeRemaining());

  useEffect(() => {
    setRemaining(computeRemaining());
    const timer = setInterval(() => {
      setRemaining(computeRemaining());
    }, 1000);

    return () => clearInterval(timer);
  }, [until]);

  if (!target || remaining <= 0) return null;

  const format = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    if (mins === 0) return `${secs}s`;
    return `${mins}m${secs > 0 ? ` ${secs}s` : ''}`;
  };

  return (
    <div className="flex items-center gap-1.5 text-xs text-sky-500" title="Recycler cooldown">
      <Snowflake className="h-3.5 w-3.5" />
      <span className="tabular-nums">{format(remaining)}</span>
    </div>
  );
}
