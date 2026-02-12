import { useState } from "react";
import { ChevronDown, ChevronRight, FileText } from "lucide-react";
import type { AgentEvent } from "../../../lib/api";

interface AgentRawEventCardProps {
  event: AgentEvent;
}

/**
 * Generic collapsible card for displaying event types that don't have
 * a dedicated rendering component (metric, artifact, message_deleted, etc.).
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
        onClick={() => setIsExpanded(!isExpanded)}
        className="
          w-full flex items-center gap-3 px-4 py-3
          bg-zinc-800/50 hover:bg-zinc-800
          transition-colors text-left
        "
      >
        <div className="p-1.5 rounded-md bg-zinc-600/20">
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
          <div className="text-xs text-zinc-500">
            {new Date(event.timestamp).toLocaleTimeString()}
          </div>
        </div>
        {formattedRawData && (
          isExpanded ? (
            <ChevronDown className="h-4 w-4 text-zinc-400" />
          ) : (
            <ChevronRight className="h-4 w-4 text-zinc-400" />
          )
        )}
      </button>

      {/* Expanded content */}
      {isExpanded && formattedRawData && (
        <div className="border-t border-zinc-700 p-4">
          <div className="text-xs font-medium text-zinc-400 mb-2">Raw Data</div>
          <pre className="text-sm text-zinc-300 bg-zinc-900 rounded-md p-3 overflow-x-auto max-h-64">
            {formattedRawData}
          </pre>
        </div>
      )}
    </div>
  );
}

export default AgentRawEventCard;
