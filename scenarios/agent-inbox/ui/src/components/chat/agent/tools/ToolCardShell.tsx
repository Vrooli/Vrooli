import { useState, type ReactNode } from "react";
import { ChevronDown, ChevronRight, Check, X } from "lucide-react";
import type { AgentEvent } from "../../../../lib/api";

interface ToolCardShellProps {
  /** Icon rendered in the colored badge */
  icon: ReactNode;
  /** Badge background color class (e.g. "bg-orange-500/20") */
  iconBg: string;
  /** Tool display name */
  toolName: string;
  /** Prominent summary shown in collapsed view (e.g. description, URL, file path) */
  summary?: string;
  /** Timestamp from the event */
  timestamp: string;
  /** Whether the tool result was successful */
  result?: AgentEvent;
  /** Expandable content (rendered only when expanded) */
  children?: ReactNode;
  /** Whether the card starts expanded */
  defaultExpanded?: boolean;
  /** Message rendering style */
  compact?: boolean;
}

/**
 * Shared collapsible card shell for all tool-specific event components.
 * Handles expand/collapse, success/failure badge, and overflow containment.
 */
export function ToolCardShell({
  icon,
  iconBg,
  toolName,
  summary,
  timestamp,
  result,
  children,
  defaultExpanded = false,
  compact = false,
}: ToolCardShellProps) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);
  const hasExpandableContent = !!children;

  return (
    <div className={`rounded-lg border overflow-hidden ${compact ? "my-1 border-zinc-800/60" : "my-2 border-zinc-700/70"}`}>
      {/* Header */}
      <button
        onClick={() => hasExpandableContent && setIsExpanded(!isExpanded)}
        className={`
          w-full flex items-center gap-3 text-left transition-colors
          ${compact ? "px-3 py-2 bg-zinc-900/35 hover:bg-zinc-900/50" : "px-4 py-3 bg-zinc-800/45 hover:bg-zinc-800/65"}
          ${!hasExpandableContent ? "cursor-default" : ""}
        `}
      >
        <div className={`p-1.5 rounded-md flex-shrink-0 ${iconBg}`}>
          {icon}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium text-zinc-200 text-sm">{toolName}</span>
            {result && (
              result.tool_success ? (
                <span className="flex items-center gap-1 text-xs text-green-400">
                  <Check className="h-3 w-3" />
                  Success
                </span>
              ) : (
                <span className="flex items-center gap-1 text-xs text-red-400">
                  <X className="h-3 w-3" />
                  Failed
                </span>
              )
            )}
          </div>
          {summary && (
            <div className="text-sm text-zinc-300 truncate mt-0.5">{summary}</div>
          )}
          <div className="text-xs text-zinc-600 mt-0.5">
            {new Date(timestamp).toLocaleTimeString()}
          </div>
        </div>
        {hasExpandableContent && (
          isExpanded ? (
            <ChevronDown className="h-4 w-4 text-zinc-500 flex-shrink-0" />
          ) : (
            <ChevronRight className="h-4 w-4 text-zinc-500 flex-shrink-0" />
          )
        )}
      </button>

      {/* Expanded content with bounded height */}
      {isExpanded && children && (
        <div className={`border-t border-zinc-800/70 max-h-[32rem] overflow-y-auto ${compact ? "p-3" : ""}`}>
          {children}
        </div>
      )}
    </div>
  );
}
