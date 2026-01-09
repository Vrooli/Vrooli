/**
 * AsyncOperationCard Component
 *
 * Displays the status of a long-running async tool operation.
 * Shows progress, current phase, and allows cancellation.
 *
 * Used by ChatView to display active operations from useAsyncStatus hook.
 */

import { useState } from "react";
import {
  Loader2,
  XCircle,
  CheckCircle2,
  AlertCircle,
  Clock,
  Zap,
} from "lucide-react";
import { Button } from "../ui/button";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";

interface AsyncOperationCardProps {
  operation: AsyncStatusUpdate;
  onCancel?: (toolCallId: string) => Promise<void>;
}

/** Get status color and icon based on operation status */
function getStatusDisplay(status: string, isTerminal: boolean) {
  if (isTerminal) {
    if (
      status === "completed" ||
      status === "success" ||
      status === "needs_review"
    ) {
      return {
        icon: CheckCircle2,
        color: "text-green-400",
        bgColor: "bg-green-500/20",
        borderColor: "border-green-500/30",
        label: status === "needs_review" ? "Needs Review" : "Completed",
      };
    }
    if (status === "failed" || status === "error" || status === "timeout") {
      return {
        icon: AlertCircle,
        color: "text-red-400",
        bgColor: "bg-red-500/20",
        borderColor: "border-red-500/30",
        label: status === "timeout" ? "Timed Out" : "Failed",
      };
    }
    if (status === "cancelled" || status === "stopped") {
      return {
        icon: XCircle,
        color: "text-slate-400",
        bgColor: "bg-slate-500/20",
        borderColor: "border-slate-500/30",
        label: "Cancelled",
      };
    }
  }

  // Running/pending states
  if (status === "pending" || status === "starting") {
    return {
      icon: Clock,
      color: "text-blue-400",
      bgColor: "bg-blue-500/20",
      borderColor: "border-blue-500/30",
      label: "Starting...",
    };
  }

  // Default: running
  return {
    icon: Zap,
    color: "text-yellow-400",
    bgColor: "bg-yellow-500/20",
    borderColor: "border-yellow-500/30",
    label: "Running",
  };
}

/** Format tool name for display */
function formatToolName(name: string): string {
  return name
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

export function AsyncOperationCard({
  operation,
  onCancel,
}: AsyncOperationCardProps) {
  const [isCancelling, setIsCancelling] = useState(false);

  const statusDisplay = getStatusDisplay(operation.status, operation.is_terminal);
  const StatusIcon = statusDisplay.icon;

  const handleCancel = async () => {
    if (!onCancel || isCancelling) return;

    setIsCancelling(true);
    try {
      await onCancel(operation.tool_call_id);
    } catch (err) {
      console.error("Failed to cancel operation:", err);
    } finally {
      setIsCancelling(false);
    }
  };

  // Calculate if we should show progress bar
  // Defensive check: ensure progress is actually a number, not an object
  const progressValue = typeof operation.progress === 'number' ? operation.progress : undefined;
  const showProgress =
    progressValue !== undefined &&
    progressValue >= 0 &&
    progressValue <= 100;

  return (
    <div
      className={`border ${statusDisplay.borderColor} ${statusDisplay.bgColor} rounded-lg p-4 my-2 transition-all duration-300`}
      data-testid={`async-operation-${operation.tool_call_id}`}
    >
      {/* Header Row */}
      <div className="flex items-center gap-3 mb-2">
        {/* Status Icon */}
        <div
          className={`w-8 h-8 rounded-full ${statusDisplay.bgColor} flex items-center justify-center flex-shrink-0`}
        >
          {operation.is_terminal ? (
            <StatusIcon className={`h-4 w-4 ${statusDisplay.color}`} />
          ) : (
            <Loader2
              className={`h-4 w-4 ${statusDisplay.color} animate-spin`}
            />
          )}
        </div>

        {/* Tool Name & Status */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium text-white truncate">
              {formatToolName(operation.tool_name)}
            </span>
            <span
              className={`text-xs px-2 py-0.5 rounded-full ${statusDisplay.bgColor} ${statusDisplay.color}`}
            >
              {statusDisplay.label}
            </span>
          </div>

          {/* Phase */}
          {operation.phase && (
            <p className="text-xs text-slate-400 truncate">
              Phase: {typeof operation.phase === "string" ? operation.phase : JSON.stringify(operation.phase)}
            </p>
          )}
        </div>

        {/* Cancel Button */}
        {!operation.is_terminal && onCancel && (
          <Button
            variant="ghost"
            size="sm"
            onClick={handleCancel}
            disabled={isCancelling}
            className="text-slate-400 hover:text-red-400 hover:bg-red-500/10 flex-shrink-0"
            data-testid={`cancel-${operation.tool_call_id}`}
          >
            {isCancelling ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <XCircle className="h-4 w-4" />
            )}
          </Button>
        )}
      </div>

      {/* Progress Bar */}
      {showProgress && !operation.is_terminal && (
        <div className="mt-3 mb-2">
          <div className="flex justify-between text-xs text-slate-400 mb-1">
            <span>Progress</span>
            <span>{progressValue}%</span>
          </div>
          <div className="h-1.5 bg-slate-700 rounded-full overflow-hidden">
            <div
              className={`h-full ${
                statusDisplay.color.replace("text-", "bg-")
              } transition-all duration-500 ease-out`}
              style={{ width: `${progressValue}%` }}
            />
          </div>
        </div>
      )}

      {/* Message */}
      {operation.message && (
        <p className="text-sm text-slate-300 mt-2 leading-relaxed">
          {typeof operation.message === "string" ? operation.message : JSON.stringify(operation.message)}
        </p>
      )}

      {/* Error Message */}
      {operation.error && (
        <div className="mt-2 p-2 bg-red-500/10 border border-red-500/20 rounded text-sm text-red-300">
          {typeof operation.error === "string" ? operation.error : JSON.stringify(operation.error)}
        </div>
      )}

      {/* Result Summary (for completed operations) */}
      {operation.is_terminal && operation.result != null && (
        <div className="mt-2 p-2 bg-slate-800/50 rounded text-xs text-slate-400">
          <span className="text-slate-500">Result available</span>
        </div>
      )}
    </div>
  );
}

/**
 * AsyncOperationsPanel - Container for multiple async operation cards.
 * Shows a header when operations are present.
 */
export function AsyncOperationsPanel({
  operations,
  onCancel,
}: {
  operations: AsyncStatusUpdate[];
  onCancel?: (toolCallId: string) => Promise<void>;
}) {
  if (operations.length === 0) {
    return null;
  }

  const activeCount = operations.filter((op) => !op.is_terminal).length;

  return (
    <div className="px-4 py-2 border-b border-slate-700/50 bg-slate-900/50">
      <div className="flex items-center gap-2 mb-2">
        <Zap className="h-4 w-4 text-yellow-400" />
        <span className="text-sm font-medium text-slate-300">
          {activeCount > 0
            ? `${activeCount} Active Operation${activeCount > 1 ? "s" : ""}`
            : "Recent Operations"}
        </span>
      </div>
      <div className="space-y-2">
        {operations.map((op) => (
          <AsyncOperationCard
            key={op.tool_call_id}
            operation={op}
            onCancel={onCancel}
          />
        ))}
      </div>
    </div>
  );
}
