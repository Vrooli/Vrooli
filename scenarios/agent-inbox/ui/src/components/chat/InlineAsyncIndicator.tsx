/**
 * InlineAsyncIndicator - Compact indicator shown in assistant message bubbles.
 *
 * Displays async operation status inline within the conversation where the tool was called.
 * Provides quick access to details and allows referencing results in follow-up messages.
 */

import {
  Zap,
  Loader2,
  CheckCircle2,
  AlertCircle,
  XCircle,
  ExternalLink,
  MessageSquarePlus,
} from "lucide-react";
import { Button } from "../ui/button";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";

interface InlineAsyncIndicatorProps {
  operation: AsyncStatusUpdate;
  onOpenDetails: () => void;
  onInsertReference: () => void;
}

/** Format tool name for display */
function formatToolName(name: string): string {
  return name
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

/** Get status icon and color */
function getStatusDisplay(status: string, isTerminal: boolean) {
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
        label: "Failed",
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
  return {
    icon: Zap,
    color: "text-yellow-400",
    bgColor: "bg-slate-800/50",
    borderColor: "border-slate-700/50",
    label: "Running",
  };
}

/** Get a brief summary of the result */
function getResultSummary(result: unknown): string | null {
  if (result === null || result === undefined) return null;

  // If it's a string, truncate it
  if (typeof result === "string") {
    return result.length > 50 ? result.slice(0, 47) + "..." : result;
  }

  // If it has a message or summary field, use that
  if (typeof result === "object" && result !== null) {
    const obj = result as Record<string, unknown>;
    if (typeof obj.message === "string") {
      return obj.message.length > 50 ? obj.message.slice(0, 47) + "..." : obj.message;
    }
    if (typeof obj.summary === "string") {
      return obj.summary.length > 50 ? obj.summary.slice(0, 47) + "..." : obj.summary;
    }
    // Count files if present
    if (Array.isArray(obj.files)) {
      return `Created ${obj.files.length} file${obj.files.length !== 1 ? "s" : ""}`;
    }
    if (typeof obj.files_created === "number") {
      return `Created ${obj.files_created} file${obj.files_created !== 1 ? "s" : ""}`;
    }
  }

  // Default: just indicate result available
  return "Result available";
}

export function InlineAsyncIndicator({
  operation,
  onOpenDetails,
  onInsertReference,
}: InlineAsyncIndicatorProps) {
  const statusDisplay = getStatusDisplay(operation.status, operation.is_terminal);
  const StatusIcon = statusDisplay.icon;
  const progressValue = typeof operation.progress === "number" ? operation.progress : undefined;
  const resultSummary = operation.is_terminal ? getResultSummary(operation.result) : null;

  return (
    <div
      className={`border ${statusDisplay.borderColor} ${statusDisplay.bgColor} rounded px-3 py-1.5 my-2`}
    >
      <div className="flex items-center gap-2">
        {/* Status icon */}
        {operation.is_terminal ? (
          <StatusIcon className={`h-4 w-4 ${statusDisplay.color}`} />
        ) : (
          <Loader2 className={`h-4 w-4 ${statusDisplay.color} animate-spin`} />
        )}

        {/* Tool name */}
        <span className="text-sm text-white font-medium">
          {formatToolName(operation.tool_name)}
        </span>

        {/* Status badge */}
        <span
          className={`text-xs px-1.5 py-0.5 rounded ${statusDisplay.bgColor} ${statusDisplay.color}`}
        >
          {statusDisplay.label}
        </span>

        {/* Progress for running operations */}
        {!operation.is_terminal && progressValue !== undefined && progressValue >= 0 && (
          <div className="flex items-center gap-1.5 ml-auto">
            <div className="w-16 h-1.5 bg-slate-700 rounded-full overflow-hidden">
              <div
                className="h-full bg-yellow-400 transition-all duration-500"
                style={{ width: `${progressValue}%` }}
              />
            </div>
            <span className="text-xs text-slate-400">{progressValue}%</span>
          </div>
        )}

        {/* Actions for completed operations */}
        {operation.is_terminal && (
          <div className="flex items-center gap-1 ml-auto">
            <Button
              variant="ghost"
              size="sm"
              onClick={onOpenDetails}
              className="h-6 px-2 text-xs text-slate-400 hover:text-slate-200"
            >
              <ExternalLink className="h-3 w-3 mr-1" />
              Details
            </Button>
            {operation.result != null && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onInsertReference}
                className="h-6 px-2 text-xs text-indigo-400 hover:text-indigo-300"
              >
                <MessageSquarePlus className="h-3 w-3 mr-1" />
                Ask About
              </Button>
            )}
          </div>
        )}

        {/* Details button for running operations */}
        {!operation.is_terminal && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onOpenDetails}
            className="h-6 px-2 text-xs text-slate-400 hover:text-slate-200 ml-auto"
          >
            <ExternalLink className="h-3 w-3 mr-1" />
            Details
          </Button>
        )}
      </div>

      {/* Message/Summary line */}
      {(operation.message || resultSummary) && (
        <p className="text-xs text-slate-400 mt-1 truncate pl-6">
          {operation.is_terminal ? resultSummary : operation.message}
        </p>
      )}
    </div>
  );
}
