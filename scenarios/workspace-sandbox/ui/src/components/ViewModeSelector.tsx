import { cn } from "../lib/utils";
import type { ViewMode } from "../lib/api";

interface ViewModeSelectorProps {
  mode: ViewMode;
  onChange: (mode: ViewMode) => void;
  disabled?: boolean;
  className?: string;
  compact?: boolean;
}

interface ModeOption {
  value: ViewMode;
  label: string;
  shortLabel: string;
  description: string;
}

const modes: ModeOption[] = [
  {
    value: "diff",
    label: "Diff",
    shortLabel: "Diff",
    description: "Show only changed lines"
  },
  {
    value: "full_diff",
    label: "Full + Diff",
    shortLabel: "Full",
    description: "Show full file with changes highlighted"
  },
  {
    value: "source",
    label: "Source",
    shortLabel: "Src",
    description: "Show file content only"
  }
];

export function ViewModeSelector({
  mode,
  onChange,
  disabled = false,
  className,
  compact = false
}: ViewModeSelectorProps) {
  return (
    <div
      className={cn(
        "inline-flex rounded-lg border border-slate-700 bg-slate-900/50 p-0.5",
        disabled && "opacity-50 pointer-events-none",
        className
      )}
      role="radiogroup"
      aria-label="View mode"
    >
      {modes.map((option) => (
        <button
          key={option.value}
          type="button"
          role="radio"
          aria-checked={mode === option.value}
          aria-label={option.description}
          title={option.description}
          onClick={() => onChange(option.value)}
          disabled={disabled}
          className={cn(
            "relative px-2.5 py-1 text-xs font-medium rounded-md transition-all",
            "focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50",
            mode === option.value
              ? "bg-slate-700 text-white shadow-sm"
              : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/50"
          )}
        >
          {compact ? option.shortLabel : option.label}
        </button>
      ))}
    </div>
  );
}
