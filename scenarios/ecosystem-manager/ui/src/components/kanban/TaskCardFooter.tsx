/**
 * TaskCardFooter Component
 * Displays elapsed timer (for in-progress tasks), execution count, and action buttons
 */

import { Eye, Trash2, PlayCircle } from 'lucide-react';
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

  return (
    <div className="flex items-center justify-between gap-2 mt-3 pt-3 border-t border-border/60">
      {/* Left side: Timer and execution count */}
      <div className="flex items-center gap-3">
        {isInProgress && hasProcess && task.current_process?.start_time && (
          <ElapsedTimer startTime={task.current_process.start_time} />
        )}
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
