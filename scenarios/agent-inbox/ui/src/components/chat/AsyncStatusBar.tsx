/**
 * AsyncStatusBar - Minimal floating status bar for async operations.
 *
 * Displays a compact single-line bar showing active async operations.
 * Replaces the bulky AsyncOperationsPanel with a more professional design.
 */

import { useState } from "react";
import {
  Zap,
  RefreshCw,
  History,
  ChevronDown,
  ChevronUp,
  Loader2,
  CheckCircle2,
  XCircle,
  AlertCircle,
} from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";

interface AsyncStatusBarProps {
  operations: AsyncStatusUpdate[];
  completedCount: number;
  onRefresh: (toolCallId: string) => Promise<void>;
  onCancel: (toolCallId: string) => Promise<void>;
  onOpenDrawer: (operation?: AsyncStatusUpdate) => void;
  isCollapsed: boolean;
  onToggleCollapse: () => void;
}

/** Format tool name for display */
function formatToolName(name: string): string {
  return name
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

/** Get compact status indicator */
function getStatusIndicator(op: AsyncStatusUpdate) {
  if (op.is_terminal) {
    if (op.status === "completed" || op.status === "success" || op.status === "needs_review") {
      return <CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" />;
    }
    if (op.status === "failed" || op.status === "error" || op.status === "timeout") {
      return <AlertCircle className="h-3.5 w-3.5 text-red-400" />;
    }
    if (op.status === "cancelled" || op.status === "stopped") {
      return <XCircle className="h-3.5 w-3.5 text-slate-400" />;
    }
  }
  return <Loader2 className="h-3.5 w-3.5 text-yellow-400 animate-spin" />;
}

/** Progress bar component */
function MiniProgress({ progress }: { progress: number }) {
  return (
    <div className="w-16 h-1.5 bg-slate-700 rounded-full overflow-hidden">
      <div
        className="h-full bg-yellow-400 transition-all duration-500"
        style={{ width: `${progress}%` }}
      />
    </div>
  );
}

export function AsyncStatusBar({
  operations,
  completedCount,
  onRefresh,
  onCancel: _onCancel,
  onOpenDrawer,
  isCollapsed,
  onToggleCollapse,
}: AsyncStatusBarProps) {
  const [refreshingIds, setRefreshingIds] = useState<Set<string>>(new Set());

  const activeOps = operations.filter((op) => !op.is_terminal);
  const recentCompleted = operations.filter((op) => op.is_terminal);

  // Show nothing if no operations at all
  if (activeOps.length === 0 && recentCompleted.length === 0 && completedCount === 0) {
    return null;
  }

  const handleRefreshAll = async () => {
    const ids = new Set(activeOps.map((op) => op.tool_call_id));
    setRefreshingIds(ids);
    try {
      await Promise.all(activeOps.map((op) => onRefresh(op.tool_call_id)));
    } finally {
      setRefreshingIds(new Set());
    }
  };

  if (isCollapsed) {
    // Collapsed view - just show count and expand button
    return (
      <div className="h-9 px-4 flex items-center gap-2 bg-slate-900/80 backdrop-blur-sm border-b border-slate-800/50 text-sm">
        <Zap className="h-3.5 w-3.5 text-yellow-400" />
        <span className="text-slate-300">
          {activeOps.length > 0 ? `${activeOps.length} active` : "Operations"}
        </span>
        <Button
          variant="ghost"
          size="sm"
          onClick={onToggleCollapse}
          className="h-6 px-2 text-slate-400 hover:text-slate-200"
        >
          <ChevronDown className="h-3.5 w-3.5" />
        </Button>
      </div>
    );
  }

  return (
    <div className="h-9 px-4 flex items-center gap-4 bg-slate-900/80 backdrop-blur-sm border-b border-slate-800/50 text-sm overflow-x-auto">
      {/* Active count indicator */}
      <div className="flex items-center gap-1.5 text-slate-300 flex-shrink-0">
        <Zap className="h-3.5 w-3.5 text-yellow-400" />
        <span>
          {activeOps.length > 0
            ? `${activeOps.length} active`
            : recentCompleted.length > 0
            ? "Recent"
            : ""}
        </span>
      </div>

      {/* Operation indicators */}
      <div className="flex items-center gap-3 flex-1 min-w-0 overflow-x-auto">
        {/* Show first few active operations */}
        {activeOps.slice(0, 3).map((op) => (
          <button
            key={op.tool_call_id}
            onClick={() => onOpenDrawer(op)}
            className="flex items-center gap-2 px-2 py-1 rounded bg-slate-800/50 hover:bg-slate-700/50 transition-colors flex-shrink-0"
          >
            {getStatusIndicator(op)}
            <span className="text-slate-300 text-xs truncate max-w-[120px]">
              {formatToolName(op.tool_name)}
            </span>
            {typeof op.progress === "number" && op.progress >= 0 && op.progress <= 100 && (
              <MiniProgress progress={op.progress} />
            )}
            {op.progress !== undefined && (
              <span className="text-slate-500 text-xs">{op.progress}%</span>
            )}
          </button>
        ))}

        {/* Show recent completed */}
        {recentCompleted.slice(0, 2).map((op) => (
          <button
            key={op.tool_call_id}
            onClick={() => onOpenDrawer(op)}
            className="flex items-center gap-2 px-2 py-1 rounded bg-slate-800/30 hover:bg-slate-700/50 transition-colors flex-shrink-0 opacity-70"
          >
            {getStatusIndicator(op)}
            <span className="text-slate-400 text-xs truncate max-w-[100px]">
              {formatToolName(op.tool_name)}
            </span>
          </button>
        ))}

        {/* Overflow indicator */}
        {activeOps.length > 3 && (
          <span className="text-slate-500 text-xs flex-shrink-0">
            +{activeOps.length - 3} more
          </span>
        )}
      </div>

      {/* Action buttons */}
      <div className="flex items-center gap-1 flex-shrink-0">
        {/* Refresh all active */}
        {activeOps.length > 0 && (
          <Tooltip content="Refresh all">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => { void handleRefreshAll(); }}
              disabled={refreshingIds.size > 0}
              className="h-6 w-6 p-0 text-slate-400 hover:text-slate-200"
            >
              <RefreshCw
                className={`h-3.5 w-3.5 ${refreshingIds.size > 0 ? "animate-spin" : ""}`}
              />
            </Button>
          </Tooltip>
        )}

        {/* History button */}
        <Tooltip content="View history">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onOpenDrawer()}
            className="h-6 w-6 p-0 text-slate-400 hover:text-slate-200"
          >
            <History className="h-3.5 w-3.5" />
          </Button>
        </Tooltip>

        {/* Collapse button */}
        <Tooltip content="Collapse">
          <Button
            variant="ghost"
            size="sm"
            onClick={onToggleCollapse}
            className="h-6 w-6 p-0 text-slate-400 hover:text-slate-200"
          >
            <ChevronUp className="h-3.5 w-3.5" />
          </Button>
        </Tooltip>
      </div>
    </div>
  );
}
