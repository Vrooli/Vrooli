import { useState } from "react";
import { ChevronDown, ChevronRight, FileText } from "lucide-react";
import type { AgentEvent } from "../../../lib/api";
import type { ViewMode } from "../../settings/Settings";
import { CodeBlock } from "../../markdown/components/CodeBlock";

interface AgentRawEventCardProps {
  event: AgentEvent;
  viewMode?: ViewMode;
}

/**
 * Generic collapsible card for displaying event types that don't have
 * a dedicated rendering component (artifact, message_deleted, etc.).
 * Default collapsed. Shows the event type badge, optional content, and
 * expandable raw JSON data.
 */
export function AgentRawEventCard({ event, viewMode = "bubble" }: AgentRawEventCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const isCompact = viewMode === "compact";

  // Try to pretty-print raw_data
  let formattedRawData: string | null = null;
  if (event.raw_data) {
    try {
      formattedRawData = JSON.stringify(JSON.parse(event.raw_data), null, 2);
    } catch {
      formattedRawData = event.raw_data;
    }
  }

  return (
    <div className={`rounded-lg border overflow-hidden ${
      isCompact ? "my-1 border-zinc-800/60" : "my-2 border-zinc-700/70"
    }`}>
      {/* Header */}
      <button
        onClick={() => formattedRawData && setIsExpanded(!isExpanded)}
        className={`
          w-full flex items-center gap-3 text-left transition-colors
          ${isCompact ? "px-3 py-2 bg-zinc-900/35 hover:bg-zinc-900/50" : "px-4 py-3 bg-zinc-800/45 hover:bg-zinc-800/65"}
          ${!formattedRawData ? "cursor-default" : ""}
        `}
      >
        <div className="p-1.5 rounded-md bg-zinc-600/20 flex-shrink-0">
          <FileText className="h-4 w-4 text-zinc-500" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="px-1.5 py-0.5 text-xs font-medium rounded bg-zinc-700/70 text-zinc-300">
              {event.type}
            </span>
            {event.content && (
              <span className="text-sm text-zinc-500 truncate">{event.content}</span>
            )}
          </div>
          <div className="text-xs text-zinc-600 mt-0.5">
            {new Date(event.timestamp).toLocaleTimeString()}
          </div>
        </div>
        {formattedRawData && (
          isExpanded ? (
            <ChevronDown className="h-4 w-4 text-zinc-500 flex-shrink-0" />
          ) : (
            <ChevronRight className="h-4 w-4 text-zinc-500 flex-shrink-0" />
          )
        )}
      </button>

      {/* Expanded content with bounded height */}
      {isExpanded && formattedRawData && (
        <div className={`border-t border-zinc-800/70 max-h-[32rem] overflow-y-auto ${isCompact ? "p-3" : "p-4"}`}>
          <div className="text-xs font-medium text-zinc-500 mb-1">Raw Data</div>
          <CodeBlock code={formattedRawData} language="json" />
        </div>
      )}
    </div>
  );
}

export default AgentRawEventCard;
