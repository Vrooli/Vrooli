/**
 * SearchModeToggle - Segmented control for switching the sidebar search
 * between plain substring filtering and AI semantic search. Disabled with
 * a tooltip when AI search is unavailable (Ollama or Qdrant unreachable).
 */

import { cn } from "../../../../lib/utils";

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
  unavailableReason,
}: SearchModeToggleProps) {
  return (
    <div
      role="group"
      aria-label="Search mode"
      className="inline-flex overflow-hidden rounded border border-slate-700/50 text-xs"
      data-testid="search-mode-toggle"
    >
      <ToggleButton
        label="Plain"
        active={mode === "plain"}
        onClick={() => onChange("plain")}
        testId="search-mode-plain"
      />
      <ToggleButton
        label="AI"
        active={mode === "ai"}
        onClick={() => onChange("ai")}
        disabled={!aiAvailable}
        title={aiAvailable ? "Semantic search (embeddings)" : unavailableReason ?? "AI search unavailable"}
        testId="search-mode-ai"
      />
    </div>
  );
}

interface ToggleButtonProps {
  label: string;
  active: boolean;
  onClick: () => void;
  disabled?: boolean;
  title?: string;
  testId: string;
}

function ToggleButton({ label, active, onClick, disabled, title, testId }: ToggleButtonProps) {
  return (
    <button
      type="button"
      onClick={disabled ? undefined : onClick}
      disabled={disabled}
      title={title}
      data-testid={testId}
      aria-pressed={active}
      className={cn(
        "px-2 py-1 transition-colors",
        active ? "bg-sky-500/20 text-sky-300" : "bg-transparent text-slate-400 hover:text-slate-200",
        disabled && "cursor-not-allowed opacity-50 hover:text-slate-400",
      )}
    >
      {label}
    </button>
  );
}
