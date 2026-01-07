import type { PhaseExecutionResult } from "../../lib/api";
import { cn } from "../../lib/utils";
import { formatDuration } from "../../lib/formatters";
import { selectors } from "../../consts/selectors";

interface PhaseResultCardSelectableProps {
  phase: PhaseExecutionResult;
  selectable?: boolean;
  selected?: boolean;
  disabled?: boolean;
  onToggle?: () => void;
  className?: string;
}

export function PhaseResultCardSelectable({
  phase,
  selectable = false,
  selected = false,
  disabled = false,
  onToggle,
  className
}: PhaseResultCardSelectableProps) {
  const isPassed = phase.status === "passed";
  const isClickable = selectable && !disabled;

  const handleClick = () => {
    if (isClickable && onToggle) {
      onToggle();
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (isClickable && onToggle && (e.key === "Enter" || e.key === " ")) {
      e.preventDefault();
      onToggle();
    }
  };

  return (
    <div
      className={cn(
        "rounded-xl border border-white/5 bg-black/20 p-3 text-sm relative",
        selectable && "transition-all duration-150",
        isClickable && "cursor-pointer hover:border-white/20 hover:bg-black/30",
        selected && "border-violet-500/50 bg-violet-500/10",
        disabled && "opacity-50 cursor-not-allowed",
        className
      )}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      tabIndex={isClickable ? 0 : undefined}
      role={selectable ? "checkbox" : undefined}
      aria-checked={selectable ? selected : undefined}
      aria-disabled={disabled}
      data-testid={selectors.runs.phaseCard}
    >
      {/* Checkbox overlay when in selection mode */}
      {selectable && (
        <div className="absolute top-2 right-2">
          <div
            className={cn(
              "w-5 h-5 rounded border-2 flex items-center justify-center transition-colors",
              disabled
                ? "border-slate-600 bg-slate-800"
                : selected
                  ? "border-violet-500 bg-violet-500"
                  : "border-slate-500 bg-transparent hover:border-violet-400"
            )}
          >
            {selected && (
              <svg
                className="w-3 h-3 text-white"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={3}
                  d="M5 13l4 4L19 7"
                />
              </svg>
            )}
          </div>
        </div>
      )}

      <div className="flex items-center justify-between pr-8">
        <span className="font-medium capitalize">{phase.name}</span>
        <span
          className={cn(
            "rounded-full px-2 py-0.5 text-xs uppercase",
            isPassed
              ? "bg-emerald-500/20 text-emerald-300"
              : "bg-red-500/20 text-red-300"
          )}
        >
          {phase.status}
        </span>
      </div>
      <p className="mt-1 text-xs text-slate-400">
        {formatDuration(phase.durationSeconds)} · {phase.observations?.length ?? 0} observations
      </p>
      {phase.error && (
        <p className="mt-1 text-xs text-red-300">
          {phase.error.slice(0, 120)}
          {phase.error.length > 120 ? "…" : ""}
        </p>
      )}
    </div>
  );
}
