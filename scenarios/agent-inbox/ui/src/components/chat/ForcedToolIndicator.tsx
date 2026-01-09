/**
 * ForcedToolIndicator - Shows when a tool is forced for the current message.
 *
 * Displays a small pill/badge in the message input footer area.
 * Uses purple/violet color scheme to distinguish from green web search.
 */
import { Wrench, X } from "lucide-react";

interface ForcedToolIndicatorProps {
  scenario: string;
  toolName: string;
  onClear: () => void;
}

export function ForcedToolIndicator({ scenario, toolName, onClear }: ForcedToolIndicatorProps) {
  return (
    <div
      className="inline-flex items-center gap-1.5 px-2 py-1 rounded-full bg-violet-500/20 border border-violet-500/30 text-xs text-violet-400"
      data-testid="forced-tool-indicator"
    >
      <Wrench className="h-3 w-3" />
      <span className="max-w-[150px] truncate" title={`${scenario}: ${toolName}`}>
        {toolName}
      </span>
      <button
        onClick={onClear}
        className="ml-0.5 hover:text-violet-300 transition-colors"
        aria-label="Clear forced tool"
        data-testid="forced-tool-clear-button"
      >
        <X className="h-3 w-3" />
      </button>
    </div>
  );
}
