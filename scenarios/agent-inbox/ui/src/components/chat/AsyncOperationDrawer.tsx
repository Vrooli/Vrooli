/**
 * AsyncOperationDrawer - Slide-out drawer for async operation details and history.
 *
 * Shows full operation details including progress, skills, result JSON, and actions.
 * Can also display history of completed operations with pagination.
 */

import { useState, useEffect, useCallback, type ComponentType, type SVGProps } from "react";
import {
  X,
  RefreshCw,
  XCircle,
  Copy,
  Check,
  ChevronRight,
  Loader2,
  CheckCircle2,
  AlertCircle,
  History,
  MessageSquarePlus,
  BookOpen,
} from "lucide-react";
import * as LucideIcons from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import type { AsyncStatusUpdate } from "../../hooks/useAsyncStatus";
import { parseToolInput, type SkillAttachment } from "../../lib/tool-utils";

type IconComponent = ComponentType<SVGProps<SVGSVGElement> & { className?: string }>;

function getIconComponent(name: string): IconComponent {
  const Icon = (LucideIcons as unknown as Record<string, IconComponent>)[name];
  return Icon || BookOpen;
}

interface AsyncOperationDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  /** Specific operation to show, or null for history view */
  operation: AsyncStatusUpdate | null;
  /** Tool arguments JSON - used to extract and display skills */
  toolArguments?: string;
  completedOperations: AsyncStatusUpdate[];
  /** Map of tool_call_id to arguments JSON for history items */
  argumentsMap?: Map<string, string>;
  onRefresh: (toolCallId: string) => Promise<void>;
  onCancel: (toolCallId: string) => Promise<void>;
  onInsertReference: (operation: AsyncStatusUpdate) => void;
  onLoadMoreHistory: () => Promise<void>;
  hasMoreHistory: boolean;
}

/** Skill chip for operation detail view */
function OperationSkillChip({ skill }: { skill: SkillAttachment }) {
  const iconName = skill.tags?.[0] || "BookOpen";
  const IconComponent = getIconComponent(
    iconName.charAt(0).toUpperCase() + iconName.slice(1).replace(/-/g, "")
  );

  return (
    <Tooltip content={skill.content.slice(0, 200) + (skill.content.length > 200 ? "..." : "")}>
      <span
        className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium
          bg-indigo-500/20 text-indigo-300 border border-indigo-500/30"
      >
        <IconComponent className="h-3 w-3" />
        <span>{skill.label}</span>
      </span>
    </Tooltip>
  );
}

/** Format tool name for display */
function formatToolName(name: string): string {
  return name
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

/** Get status display info */
function getStatusDisplay(status: string, isTerminal: boolean) {
  if (isTerminal) {
    if (status === "completed" || status === "success" || status === "needs_review") {
      return {
        icon: CheckCircle2,
        color: "text-emerald-400",
        bgColor: "bg-emerald-500/10",
        borderColor: "border-emerald-500/20",
        label: status === "needs_review" ? "Needs Review" : "Completed",
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
  return {
    icon: Loader2,
    color: "text-yellow-400",
    bgColor: "bg-yellow-500/10",
    borderColor: "border-yellow-500/20",
    label: "Running",
  };
}

/** JSON syntax highlighter component */
function JsonDisplay({ data }: { data: unknown }) {
  const [copied, setCopied] = useState(false);

  const jsonString = JSON.stringify(data, null, 2);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(jsonString);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative">
      <Button
        variant="ghost"
        size="sm"
        onClick={handleCopy}
        className="absolute top-2 right-2 h-7 w-7 p-0 text-slate-500 hover:text-slate-300"
      >
        {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
      </Button>
      <pre className="bg-slate-800/50 rounded-lg p-4 overflow-auto max-h-64 text-xs text-slate-300 font-mono">
        {jsonString}
      </pre>
    </div>
  );
}

/** Operation detail view */
function OperationDetail({
  operation,
  toolArguments,
  onRefresh,
  onCancel,
  onInsertReference,
}: {
  operation: AsyncStatusUpdate;
  toolArguments?: string;
  onRefresh: (toolCallId: string) => Promise<void>;
  onCancel: (toolCallId: string) => Promise<void>;
  onInsertReference: (operation: AsyncStatusUpdate) => void;
}) {
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [isCancelling, setIsCancelling] = useState(false);

  const statusDisplay = getStatusDisplay(operation.status, operation.is_terminal);
  const StatusIcon = statusDisplay.icon;

  // Parse tool arguments to extract skills
  const parsedInput = parseToolInput(toolArguments);
  const skills = parsedInput.skills;
  const hasSkills = skills.length > 0;
  const hasArguments = Object.keys(parsedInput.arguments).length > 0;

  const handleRefresh = async () => {
    setIsRefreshing(true);
    try {
      await onRefresh(operation.tool_call_id);
    } finally {
      setIsRefreshing(false);
    }
  };

  const handleCancel = async () => {
    setIsCancelling(true);
    try {
      await onCancel(operation.tool_call_id);
    } finally {
      setIsCancelling(false);
    }
  };

  const progressValue = typeof operation.progress === "number" ? operation.progress : undefined;

  return (
    <div className="flex-1 overflow-y-auto p-4 space-y-4">
      {/* Header */}
      <div className="flex items-start gap-3">
        <div
          className={`w-10 h-10 rounded-full ${statusDisplay.bgColor} flex items-center justify-center flex-shrink-0`}
        >
          <StatusIcon
            className={`h-5 w-5 ${statusDisplay.color} ${
              !operation.is_terminal ? "animate-spin" : ""
            }`}
          />
        </div>
        <div className="flex-1 min-w-0">
          <h3 className="text-lg font-medium text-white">
            {formatToolName(operation.tool_name)}
          </h3>
          <div className="flex items-center gap-2 mt-1">
            <span
              className={`text-xs px-2 py-0.5 rounded-full ${statusDisplay.bgColor} ${statusDisplay.color}`}
            >
              {statusDisplay.label}
            </span>
            <span className="text-xs text-slate-500">
              {new Date(operation.updated_at).toLocaleTimeString()}
            </span>
          </div>
        </div>
      </div>

      {/* Progress */}
      {progressValue !== undefined && !operation.is_terminal && (
        <div>
          <div className="flex justify-between text-xs text-slate-400 mb-1">
            <span>Progress</span>
            <span>{progressValue}%</span>
          </div>
          <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
            <div
              className="h-full bg-yellow-400 transition-all duration-500"
              style={{ width: `${progressValue}%` }}
            />
          </div>
        </div>
      )}

      {/* Skills - shown if arguments contain skills */}
      {hasSkills && (
        <div>
          <label className="text-xs text-slate-500 uppercase tracking-wide">Skills</label>
          <div className="flex flex-wrap gap-2 mt-2">
            {skills.map((skill) => (
              <OperationSkillChip key={skill.key} skill={skill} />
            ))}
          </div>
        </div>
      )}

      {/* Arguments - shown if there are non-skill arguments */}
      {hasArguments && (
        <div>
          <label className="text-xs text-slate-500 uppercase tracking-wide">Arguments</label>
          <div className="mt-2">
            <JsonDisplay data={parsedInput.arguments} />
          </div>
        </div>
      )}

      {/* Phase */}
      {operation.phase && (
        <div>
          <label className="text-xs text-slate-500 uppercase tracking-wide">Phase</label>
          <p className="text-sm text-slate-300 mt-1">{operation.phase}</p>
        </div>
      )}

      {/* Message */}
      {operation.message && (
        <div>
          <label className="text-xs text-slate-500 uppercase tracking-wide">Status</label>
          <p className="text-sm text-slate-300 mt-1">{operation.message}</p>
        </div>
      )}

      {/* Error */}
      {operation.error && (
        <div className={`p-3 rounded-lg ${operation.is_terminal ? "bg-red-500/10 border border-red-500/20" : "bg-amber-500/10 border border-amber-500/20"}`}>
          <label className="text-xs text-slate-500 uppercase tracking-wide">
            {operation.is_terminal ? "Error" : "Warning"}
          </label>
          <p className={`text-sm mt-1 ${operation.is_terminal ? "text-red-300" : "text-amber-300"}`}>
            {operation.error}
          </p>
        </div>
      )}

      {/* Result */}
      {operation.result != null && (
        <div>
          <label className="text-xs text-slate-500 uppercase tracking-wide">Result</label>
          <div className="mt-2">
            <JsonDisplay data={operation.result} />
          </div>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center gap-2 pt-4 border-t border-slate-700/50">
        {!operation.is_terminal && (
          <>
            <Button
              variant="outline"
              size="sm"
              onClick={handleRefresh}
              disabled={isRefreshing}
              className="text-slate-300"
            >
              {isRefreshing ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <RefreshCw className="h-4 w-4 mr-2" />
              )}
              Refresh
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleCancel}
              disabled={isCancelling}
              className="text-red-400 hover:text-red-300"
            >
              {isCancelling ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : (
                <XCircle className="h-4 w-4 mr-2" />
              )}
              Cancel
            </Button>
          </>
        )}
        {operation.is_terminal && operation.result != null && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => onInsertReference(operation)}
            className="text-indigo-400 hover:text-indigo-300"
          >
            <MessageSquarePlus className="h-4 w-4 mr-2" />
            Ask About This
          </Button>
        )}
      </div>
    </div>
  );
}

/** History list view */
function HistoryList({
  operations,
  onSelectOperation,
  onLoadMore,
  hasMore,
  onInsertReference,
}: {
  operations: AsyncStatusUpdate[];
  onSelectOperation: (op: AsyncStatusUpdate) => void;
  onLoadMore: () => Promise<void>;
  hasMore: boolean;
  onInsertReference: (operation: AsyncStatusUpdate) => void;
}) {
  const [isLoadingMore, setIsLoadingMore] = useState(false);

  const handleLoadMore = async () => {
    setIsLoadingMore(true);
    try {
      await onLoadMore();
    } finally {
      setIsLoadingMore(false);
    }
  };

  if (operations.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center text-slate-500">
        <div className="text-center">
          <History className="h-12 w-12 mx-auto mb-3 opacity-50" />
          <p>No completed operations</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="p-2 space-y-1">
        {operations.map((op) => {
          const statusDisplay = getStatusDisplay(op.status, op.is_terminal);
          const StatusIcon = statusDisplay.icon;

          return (
            <button
              key={op.tool_call_id}
              onClick={() => onSelectOperation(op)}
              className="w-full flex items-center gap-3 p-3 rounded-lg hover:bg-slate-800/50 transition-colors text-left"
            >
              <div
                className={`w-8 h-8 rounded-full ${statusDisplay.bgColor} flex items-center justify-center flex-shrink-0`}
              >
                <StatusIcon className={`h-4 w-4 ${statusDisplay.color}`} />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm text-white truncate">
                  {formatToolName(op.tool_name)}
                </p>
                <p className="text-xs text-slate-500">
                  {new Date(op.updated_at).toLocaleString()}
                </p>
              </div>
              <div className="flex items-center gap-2">
                {op.result != null && (
                  <Tooltip content="Ask about this">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        onInsertReference(op);
                      }}
                      className="h-7 w-7 p-0 text-slate-500 hover:text-indigo-400"
                    >
                      <MessageSquarePlus className="h-3.5 w-3.5" />
                    </Button>
                  </Tooltip>
                )}
                <ChevronRight className="h-4 w-4 text-slate-600" />
              </div>
            </button>
          );
        })}
      </div>

      {hasMore && (
        <div className="p-4 text-center">
          <Button
            variant="outline"
            size="sm"
            onClick={handleLoadMore}
            disabled={isLoadingMore}
          >
            {isLoadingMore ? (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            ) : (
              <History className="h-4 w-4 mr-2" />
            )}
            Load More
          </Button>
        </div>
      )}
    </div>
  );
}

export function AsyncOperationDrawer({
  isOpen,
  onClose,
  operation,
  toolArguments,
  completedOperations,
  argumentsMap,
  onRefresh,
  onCancel,
  onInsertReference,
  onLoadMoreHistory,
  hasMoreHistory,
}: AsyncOperationDrawerProps) {
  const [selectedOp, setSelectedOp] = useState<AsyncStatusUpdate | null>(operation);

  // Sync selectedOp with operation prop
  useEffect(() => {
    setSelectedOp(operation);
  }, [operation]);

  const handleBack = useCallback(() => {
    setSelectedOp(null);
  }, []);

  // Get arguments for the selected operation
  const getSelectedOpArguments = (): string | undefined => {
    if (!selectedOp) return undefined;
    // If this is the primary operation, use toolArguments prop
    if (operation && selectedOp.tool_call_id === operation.tool_call_id) {
      return toolArguments;
    }
    // Otherwise look up in the arguments map
    return argumentsMap?.get(selectedOp.tool_call_id);
  };

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 z-40"
        onClick={onClose}
      />

      {/* Drawer */}
      <div className="fixed right-0 top-0 bottom-0 w-96 max-w-full bg-slate-900 border-l border-slate-800 z-50 flex flex-col shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-slate-800">
          <div className="flex items-center gap-2">
            {selectedOp && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleBack}
                className="h-7 px-2 text-slate-400 hover:text-slate-200"
              >
                <ChevronRight className="h-4 w-4 rotate-180" />
              </Button>
            )}
            <h2 className="text-sm font-medium text-white">
              {selectedOp ? formatToolName(selectedOp.tool_name) : "Operation History"}
            </h2>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={onClose}
            className="h-7 w-7 p-0 text-slate-400 hover:text-slate-200"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Content */}
        {selectedOp ? (
          <OperationDetail
            operation={selectedOp}
            toolArguments={getSelectedOpArguments()}
            onRefresh={onRefresh}
            onCancel={onCancel}
            onInsertReference={onInsertReference}
          />
        ) : (
          <HistoryList
            operations={completedOperations}
            onSelectOperation={setSelectedOp}
            onLoadMore={onLoadMoreHistory}
            hasMore={hasMoreHistory}
            onInsertReference={onInsertReference}
          />
        )}
      </div>
    </>
  );
}
