import { useState } from "react";
import { ChevronDown, ChevronRight, Package, Sparkles } from "lucide-react";
import type { AgentEvent } from "../../../lib/api";
import { getCompactionReduction } from "../../../lib/api";
import type { ViewMode } from "../../settings/Settings";
import { cn } from "../../../lib/utils";

interface AgentCompactionCardProps {
  event: AgentEvent;
  viewMode?: ViewMode;
}

export function AgentCompactionCard({ event, viewMode = "bubble" }: AgentCompactionCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const isCompact = viewMode === "compact";
  const reduction = getCompactionReduction(event);
  const isManual = event.compaction_trigger === "manual";

  return (
    <div
      className={cn(
        "relative",
        isCompact ? "my-2" : "my-4"
      )}
    >
      {/* Divider line */}
      <div className="absolute inset-x-0 top-1/2 -translate-y-1/2 h-px bg-gradient-to-r from-transparent via-amber-500/50 to-transparent" />

      {/* Card */}
      <div
        className={cn(
          "relative mx-auto max-w-2xl rounded-lg border",
          "border-amber-500/30 bg-amber-500/5",
          isCompact ? "px-3 py-2" : "px-4 py-3"
        )}
      >
        {/* Header */}
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="flex w-full items-center justify-between gap-3 text-left"
        >
          <div className="flex items-center gap-2">
            <Package className="h-4 w-4 text-amber-400" />
            <span className={cn(
              "font-medium text-amber-200",
              isCompact ? "text-xs" : "text-sm"
            )}>
              Conversation Compacted
            </span>
            {isManual && (
              <span className="rounded bg-amber-500/20 px-1.5 py-0.5 text-[10px] font-medium text-amber-300">
                Manual
              </span>
            )}
          </div>

          <div className="flex items-center gap-3">
            {/* Stats badges */}
            {event.compaction_messages_compacted != null && event.compaction_messages_compacted > 0 && (
              <span className="text-xs text-zinc-400">
                {event.compaction_messages_compacted} messages
              </span>
            )}
            {reduction !== null && (
              <span className="flex items-center gap-1 text-xs text-emerald-400">
                <Sparkles className="h-3 w-3" />
                {reduction}% smaller
              </span>
            )}

            {/* Expand toggle */}
            {isExpanded ? (
              <ChevronDown className="h-4 w-4 text-zinc-400" />
            ) : (
              <ChevronRight className="h-4 w-4 text-zinc-400" />
            )}
          </div>
        </button>

        {/* Original command (if manual) */}
        {isManual && event.compaction_original_command && (
          <div className={cn(
            "mt-2 rounded bg-zinc-800/50 font-mono text-amber-300/80",
            isCompact ? "px-2 py-1 text-xs" : "px-3 py-1.5 text-sm"
          )}>
            {event.compaction_original_command}
          </div>
        )}

        {/* Focus indicator */}
        {event.compaction_focus && (
          <div className={cn(
            "mt-2 text-zinc-400",
            isCompact ? "text-xs" : "text-sm"
          )}>
            Focus: <span className="text-zinc-300">{event.compaction_focus}</span>
          </div>
        )}

        {/* Expandable summary */}
        {isExpanded && (
          <div className={cn(
            "mt-3 border-t border-amber-500/20 pt-3",
            isCompact ? "text-xs" : "text-sm"
          )}>
            <div className="mb-2 text-xs font-medium uppercase tracking-wider text-zinc-500">
              Summary
            </div>
            <div className="whitespace-pre-wrap text-zinc-300">
              {event.content}
            </div>

            {/* Token stats (if available) */}
            {event.compaction_tokens_before != null && event.compaction_tokens_before > 0 && (
              <div className="mt-3 flex gap-4 text-xs text-zinc-500">
                <span>
                  Before: <span className="text-zinc-400">{event.compaction_tokens_before.toLocaleString()} tokens</span>
                </span>
                <span>
                  After: <span className="text-zinc-400">{(event.compaction_tokens_after ?? 0).toLocaleString()} tokens</span>
                </span>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export default AgentCompactionCard;
