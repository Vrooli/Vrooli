/**
 * WebSearchIndicator - Shows when web search is enabled for the current message.
 *
 * Displays a small pill/badge in the message input footer area.
 */
import { Globe, X } from "lucide-react";

interface WebSearchIndicatorProps {
  enabled: boolean;
  onDisable: () => void;
}

export function WebSearchIndicator({ enabled, onDisable }: WebSearchIndicatorProps) {
  if (!enabled) {
    return null;
  }

  return (
    <div
      className="inline-flex items-center gap-1.5 px-2 py-1 rounded-full bg-green-500/20 border border-green-500/30 text-xs text-green-400"
      data-testid="web-search-indicator"
    >
      <Globe className="h-3 w-3" />
      <span>Web search</span>
      <button
        onClick={onDisable}
        className="ml-0.5 hover:text-green-300 transition-colors"
        aria-label="Disable web search"
        data-testid="web-search-disable-button"
      >
        <X className="h-3 w-3" />
      </button>
    </div>
  );
}
