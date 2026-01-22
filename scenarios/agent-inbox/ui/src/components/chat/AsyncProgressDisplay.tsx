/**
 * AsyncProgressDisplay - Shared component for displaying async operation progress.
 *
 * Used by:
 * - ToolCallDetailModal (modal variant)
 * - InlineAsyncIndicator (inline variant - optional refactor)
 * - AsyncOperationDrawer (card variant)
 *
 * Provides consistent styling for progress bars, status icons, and phase indicators.
 */

import {
  Loader2,
  CheckCircle2,
  AlertCircle,
  XCircle,
  Zap,
  Clock,
} from "lucide-react";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";

export type AsyncProgressVariant = "inline" | "card" | "modal";

interface AsyncProgressDisplayProps {
  status: string;
  isTerminal: boolean;
  progress?: number;
  message?: string;
  phase?: string;
  variant: AsyncProgressVariant;
  className?: string;
}

/** Get status display info (icon, colors, label) */
export function getAsyncStatusDisplay(status: string, isTerminal: boolean) {
  if (isTerminal) {
    if (status === "completed" || status === "success" || status === "needs_review") {
      return {
        icon: CheckCircle2,
        color: "text-emerald-400",
        bgColor: "bg-emerald-500/10",
        borderColor: "border-emerald-500/20",
        label: "Completed",
      };
    }
    if (status === "failed" || status === "error" || status === "timeout") {
      return {
        icon: AlertCircle,
        color: "text-red-400",
        bgColor: "bg-red-500/10",
        borderColor: "border-red-500/20",
        label: status === "timeout" ? "Timed Out" : "Failed",
      };
    }
    if (status === "cancelled" || status === "stopped") {
      return {
        icon: XCircle,
        color: "text-slate-400",
        bgColor: "bg-slate-500/10",
        borderColor: "border-slate-500/20",
        label: "Cancelled",
      };
    }
  }

  // Non-terminal statuses
  if (status === "running" || status === "in_progress") {
    return {
      icon: Loader2,
      color: "text-yellow-400",
      bgColor: "bg-yellow-500/10",
      borderColor: "border-yellow-500/20",
      label: "Running",
      animate: true,
    };
  }

  if (status === "pending" || status === "queued") {
    return {
      icon: Clock,
      color: "text-slate-400",
      bgColor: "bg-slate-500/10",
      borderColor: "border-slate-500/20",
      label: "Pending",
    };
  }

  // Default fallback
  return {
    icon: Zap,
    color: "text-yellow-400",
    bgColor: "bg-yellow-500/10",
    borderColor: "border-yellow-500/20",
    label: "In Progress",
  };
}

/** Check if a status represents a failed state */
export function isAsyncFailedStatus(status: string): boolean {
  return ["failed", "error", "timeout", "cancelled", "stopped"].includes(status);
}

/** Check if a status represents a successful completion */
export function isAsyncSuccessStatus(status: string): boolean {
  return ["completed", "success", "needs_review"].includes(status);
}

/**
 * Shared progress display component that can be used in different contexts.
 */
export function AsyncProgressDisplay({
  status,
  isTerminal,
  progress,
  message,
  phase,
  variant,
  className = "",
}: AsyncProgressDisplayProps) {
  const statusDisplay = getAsyncStatusDisplay(status, isTerminal);
  const StatusIcon = statusDisplay.icon;
  const hasProgress = typeof progress === "number" && progress >= 0;

  // Size classes based on variant
  const sizeClasses = {
    inline: {
      container: "py-1",
      icon: "h-3.5 w-3.5",
      text: "text-xs",
      progress: "h-1 w-12",
    },
    card: {
      container: "py-2",
      icon: "h-4 w-4",
      text: "text-sm",
      progress: "h-1.5 w-20",
    },
    modal: {
      container: "py-3",
      icon: "h-5 w-5",
      text: "text-sm",
      progress: "h-2 w-32",
    },
  };

  const sizes = sizeClasses[variant];

  return (
    <div className={`${sizes.container} ${className}`}>
      {/* Status and Progress Row */}
      <div className="flex items-center gap-2">
        <StatusIcon
          className={`${sizes.icon} ${statusDisplay.color} ${
            (statusDisplay as { animate?: boolean }).animate ? "animate-spin" : ""
          }`}
        />

        {/* Progress bar for running operations */}
        {!isTerminal && hasProgress && (
          <div className="flex items-center gap-2">
            <div
              className={`${sizes.progress} bg-slate-700 rounded-full overflow-hidden`}
            >
              <div
                className="h-full bg-yellow-400 transition-all duration-500"
                style={{ width: `${progress}%` }}
              />
            </div>
            <span className={`${sizes.text} text-slate-400`}>{progress}%</span>
          </div>
        )}

        {/* Phase indicator */}
        {phase && (
          <span
            className={`${sizes.text} px-2 py-0.5 rounded-full ${statusDisplay.bgColor} ${statusDisplay.color}`}
          >
            {phase}
          </span>
        )}
      </div>

      {/* Status message */}
      {message && (
        <p className={`${sizes.text} text-slate-400 mt-1.5 ${variant === "inline" ? "truncate" : ""}`}>
          {message}
        </p>
      )}
    </div>
  );
}

/**
 * Convenience component that takes an AsyncStatusUpdate directly.
 */
interface AsyncProgressFromOperationProps {
  operation: AsyncStatusUpdate;
  variant: AsyncProgressVariant;
  className?: string;
}

export function AsyncProgressFromOperation({
  operation,
  variant,
  className,
}: AsyncProgressFromOperationProps) {
  return (
    <AsyncProgressDisplay
      status={operation.status}
      isTerminal={operation.is_terminal}
      progress={operation.progress}
      message={operation.message}
      phase={operation.phase}
      variant={variant}
      className={className}
    />
  );
}
