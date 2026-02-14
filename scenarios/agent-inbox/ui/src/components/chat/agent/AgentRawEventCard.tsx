import { useState } from "react";
import { ChevronDown, ChevronRight, FileText } from "lucide-react";
import type { AgentEvent } from "../../../lib/api";
import { CodeBlock } from "../../markdown/components/CodeBlock";

interface AgentRawEventCardProps {
  event: AgentEvent;
}

/**
 * Generic collapsible card for displaying event types that don't have
 * a dedicated rendering component (artifact, message_deleted, etc.).
 * Default collapsed. Shows the event type badge, optional content, and
 * expandable raw JSON data.
 */
export function AgentRawEventCard({ event }: AgentRawEventCardProps) {
  const [isExpanded, setIsExpanded] = useState(false);

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
    <div className="my-2 rounded-lg border border-zinc-700 overflow-hidden">
      {/* Header */}
      <button
        onClick={() => formattedRawData && setIsExpanded(!isExpanded)}
        className={`
          w-full flex items-center gap-3 px-4 py-3
          bg-zinc-800/50 hover:bg-zinc-800
          transition-colors text-left
          ${!formattedRawData ? "cursor-default" : ""}
        `}
      >
        <div className="p-1.5 rounded-md bg-zinc-600/20 flex-shrink-0">
          <FileText className="h-4 w-4 text-zinc-400" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="px-1.5 py-0.5 text-xs font-medium rounded bg-zinc-700 text-zinc-300">
              {event.type}
            </span>
            {event.content && (
              <span className="text-sm text-zinc-400 truncate">{event.content}</span>
            )}
          </div>
          <div className="text-xs text-zinc-500 mt-0.5">
            {new Date(event.timestamp).toLocaleTimeString()}
          </div>
        </div>
        {formattedRawData && (
          isExpanded ? (
            <ChevronDown className="h-4 w-4 text-zinc-400 flex-shrink-0" />
          ) : (
            <ChevronRight className="h-4 w-4 text-zinc-400 flex-shrink-0" />
          )
        )}
      </button>

      {/* Expanded content with bounded height */}
      {isExpanded && formattedRawData && (
        <div className="border-t border-zinc-700 p-4 max-h-[32rem] overflow-y-auto">
          <div className="text-xs font-medium text-zinc-400 mb-1">Raw Data</div>
          <CodeBlock code={formattedRawData} language="json" />
        </div>
      )}
    </div>
  );
}

export default AgentRawEventCard;
