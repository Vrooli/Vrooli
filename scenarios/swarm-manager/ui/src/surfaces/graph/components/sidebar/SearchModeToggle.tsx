/**
 * SearchModeToggle - Inline icon toggle for switching the sidebar search
 * between plain substring filtering and AI semantic search.
 */

import { Sparkles } from "lucide-react";
import { cn } from "../../../../lib/utils";
import { Tooltip } from "../../../../components/ui/tooltip";

export type SearchMode = "plain" | "ai";

interface SearchModeToggleProps {
  mode: SearchMode;
  onChange: (mode: SearchMode) => void;
  aiAvailable: boolean;
  unavailableReason?: string;
}

export function SearchModeToggle({
  mode,
  onChange,
  aiAvailable,
}: SearchModeToggleProps) {
  if (!aiAvailable) return null;

  const aiActive = mode === "ai";

  const label = aiActive ? "Use plain search" : "Use AI search";

  return (
    <Tooltip content={aiActive ? "AI search active" : "Switch to AI search"}>
      <button
        type="button"
        onClick={() => onChange(aiActive ? "plain" : "ai")}
        title={label}
        data-testid="search-mode-ai"
        aria-label={label}
        aria-pressed={aiActive}
        className={cn(
          "inline-flex h-6 w-6 items-center justify-center rounded text-slate-400 transition-colors hover:bg-slate-700/70 hover:text-slate-100",
          aiActive && "bg-sky-500/20 text-sky-300 hover:bg-sky-500/25 hover:text-sky-200",
        )}
      >
        <Sparkles className="h-3.5 w-3.5" aria-hidden />
      </button>
    </Tooltip>
  );
}
