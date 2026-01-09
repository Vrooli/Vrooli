import React from 'react';
import {
  Square,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  Clock,
  Loader2,
  Eye,
  Timer,
  RotateCcw,
  Folder,
} from 'lucide-react';
import { formatDistanceToNowStrict } from 'date-fns';

/** Safely check if a Date is valid */
const isValidDate = (date: Date): boolean => {
  return date instanceof Date && !isNaN(date.getTime());
};

/** Format duration in a compact way */
const formatDuration = (ms: number): string => {
  if (ms < 1000) return '< 1s';
  if (ms < 60000) return `${Math.round(ms / 1000)}s`;
  const minutes = Math.floor(ms / 60000);
  const seconds = Math.round((ms % 60000) / 1000);
  return seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
};

/** Format timestamp in a compact way (e.g., "3h ago" instead of "about 3 hours ago") */
const formatTimeAgo = (date: Date): string => {
  if (!isValidDate(date)) return 'Unknown';
  try {
    return formatDistanceToNowStrict(date, { addSuffix: true });
  } catch {
    return 'Unknown';
  }
};

export type ExecutionStatus = 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';

export interface ExecutionCardData {
  id: string;
  workflowId: string;
  workflowName: string;
  projectName?: string;
  status: ExecutionStatus;
  startedAt: Date;
  completedAt?: Date;
  error?: string;
  /** Optional progress percentage (0-100) for running executions */
  progress?: number;
  /** Optional current step description for running executions */
  currentStep?: string;
  /** Optional step count info (e.g., "5/5" or "3/5") */
  stepsCompleted?: number;
  stepsTotal?: number;
}

export interface ExecutionCardProps {
  execution: ExecutionCardData;
  /** Whether this execution is currently running (affects styling) */
  isRunning?: boolean;
  /** Whether this card is currently selected (for split view) */
  isSelected?: boolean;
  /** Callback when the card is clicked */
  onClick?: (executionId: string, workflowId: string) => void;
  /** Callback when the view button is clicked */
  onView?: (executionId: string, workflowId: string) => void;
  /** Callback when the stop button is clicked (only shown for running executions) */
  onStop?: (executionId: string) => void;
  /** Callback when the re-run button is clicked */
  onRerun?: (executionId: string, workflowId: string) => void;
  /** Whether to show the project name */
  showProjectName?: boolean;
  /** Whether to show the progress bar for running executions */
  showProgressBar?: boolean;
  /** Optional test ID for the card */
  testId?: string;
}

const statusConfig: Record<ExecutionStatus, {
  icon: typeof Clock;
  color: string;
  textColor: string;
  bgColor: string;
  borderColor: string;
  leftBorderColor: string;
  label: string;
  animate?: boolean;
}> = {
  pending: {
    icon: Clock,
    color: 'text-gray-400',
    textColor: 'text-gray-400',
    bgColor: 'bg-gray-500/10',
    borderColor: 'border-gray-500/30',
    leftBorderColor: 'border-l-gray-500',
    label: 'Pending',
  },
  running: {
    icon: Loader2,
    color: 'text-green-400',
    textColor: 'text-green-400',
    bgColor: 'bg-green-500/10',
    borderColor: 'border-green-500/30',
    leftBorderColor: 'border-l-green-500',
    label: 'Running',
    animate: true,
  },
  completed: {
    icon: CheckCircle2,
    color: 'text-emerald-400',
    textColor: 'text-emerald-400',
    bgColor: 'bg-emerald-500/10',
    borderColor: 'border-emerald-500/30',
    leftBorderColor: 'border-l-emerald-500',
    label: 'Completed',
  },
  failed: {
    icon: XCircle,
    color: 'text-red-400',
    textColor: 'text-red-400',
    bgColor: 'bg-red-500/10',
    borderColor: 'border-red-500/30',
    leftBorderColor: 'border-l-red-500',
    label: 'Failed',
  },
  cancelled: {
    icon: AlertTriangle,
    color: 'text-amber-400',
    textColor: 'text-amber-400',
    bgColor: 'bg-amber-500/10',
    borderColor: 'border-amber-500/30',
    leftBorderColor: 'border-l-amber-500',
    label: 'Cancelled',
  },
};

export const ExecutionCard: React.FC<ExecutionCardProps> = ({
  execution,
  isRunning = false,
  isSelected = false,
  onClick,
  onView,
  onStop,
  onRerun,
  showProjectName = true,
  showProgressBar = true,
  testId,
}) => {
  const config = statusConfig[execution.status];
  const StatusIcon = config.icon;

  const durationMs = execution.completedAt && isValidDate(execution.startedAt) && isValidDate(execution.completedAt)
    ? execution.completedAt.getTime() - execution.startedAt.getTime()
    : undefined;
  const durationLabel = durationMs !== undefined ? formatDuration(durationMs) : undefined;

  // For completed executions, don't show generic "Completed successfully"
  // Only show error message for failed, or current step for running
  const logSnippet = execution.error
    ? execution.error.split('\n')[0]
    : execution.currentStep
      ? execution.currentStep
      : undefined;

  const effectiveIsRunning = isRunning || execution.status === 'running' || execution.status === 'pending';

  // Step count display
  const stepCountLabel = execution.stepsTotal !== undefined && execution.stepsTotal > 0
    ? `${execution.stepsCompleted ?? 0}/${execution.stepsTotal} steps`
    : undefined;

  const handleCardClick = () => {
    if (onClick) {
      onClick(execution.id, execution.workflowId);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleCardClick();
    }
  };

  return (
    <div
      data-testid={testId}
      data-execution-id={execution.id}
      data-execution-status={execution.status}
      onClick={handleCardClick}
      onKeyDown={handleKeyDown}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      className={`
        p-4 rounded-lg border-l-4 border transition-all
        ${config.leftBorderColor}
        ${isSelected
          ? 'bg-gray-700/80 border-flow-accent ring-1 ring-flow-accent/50'
          : effectiveIsRunning
            ? 'bg-gray-800/80 border-gray-700/50 shadow-lg shadow-green-500/5'
            : 'bg-gray-800/50 border-gray-700/50'
        }
        ${onClick ? 'cursor-pointer hover:bg-gray-700/60 hover:border-gray-600' : ''}
      `}
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3 min-w-0 flex-1">
          <div className={`p-2 rounded-lg ${config.bgColor} flex-shrink-0`}>
            <StatusIcon
              size={18}
              className={`${config.color} ${config.animate ? 'animate-spin' : ''}`}
            />
          </div>
          <div className="min-w-0 flex-1">
            {/* Workflow name */}
            <div className="font-medium text-surface truncate">
              {execution.workflowName}
            </div>

            {/* Project name with folder icon */}
            {showProjectName && execution.projectName && (
              <div className="flex items-center gap-1 text-xs text-gray-500 truncate mt-0.5">
                <Folder size={10} className="flex-shrink-0" />
                <span className="truncate">{execution.projectName}</span>
              </div>
            )}

            {/* Metadata row: time, duration, steps */}
            <div className="flex items-center gap-2 mt-1.5 text-xs text-gray-400 flex-wrap">
              <span>
                {effectiveIsRunning
                  ? formatTimeAgo(execution.startedAt)
                  : execution.completedAt
                    ? formatTimeAgo(execution.completedAt)
                    : formatTimeAgo(execution.startedAt)}
              </span>
              {durationLabel && (
                <>
                  <span className="text-gray-600">·</span>
                  <span className="inline-flex items-center gap-1 text-gray-300">
                    <Timer size={11} />
                    {durationLabel}
                  </span>
                </>
              )}
              {stepCountLabel && (
                <>
                  <span className="text-gray-600">·</span>
                  <span className="text-gray-300">{stepCountLabel}</span>
                </>
              )}
            </div>

            {/* Error message or current step (not generic success message) */}
            {logSnippet && (
              <div className={`mt-2 text-xs line-clamp-1 ${
                execution.error ? 'text-red-400' : 'text-gray-400'
              }`}>
                {logSnippet}
              </div>
            )}

            {/* Progress bar for running executions */}
            {showProgressBar && effectiveIsRunning && execution.progress !== undefined && execution.progress > 0 && (
              <div className="mt-2 w-full bg-gray-700 rounded-full h-1.5">
                <div
                  className="bg-flow-accent h-1.5 rounded-full transition-all duration-300"
                  style={{ width: `${execution.progress}%` }}
                />
              </div>
            )}
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-1 flex-shrink-0">
          {effectiveIsRunning && onStop && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onStop(execution.id);
              }}
              className="p-1.5 text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded-md transition-colors"
              title="Stop execution"
            >
              <Square size={14} />
            </button>
          )}
          {!effectiveIsRunning && onRerun && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onRerun(execution.id, execution.workflowId);
              }}
              className="p-1.5 text-gray-400 hover:text-surface hover:bg-gray-700 rounded-md transition-colors"
              title="Re-run workflow"
            >
              <RotateCcw size={14} />
            </button>
          )}
          {onView && (
            <button
              onClick={(e) => {
                e.stopPropagation();
                onView(execution.id, execution.workflowId);
              }}
              className="p-1.5 text-gray-400 hover:text-surface hover:bg-gray-700 rounded-md transition-colors"
              title="View details"
            >
              <Eye size={14} />
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default ExecutionCard;
