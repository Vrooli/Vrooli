import { useMemo } from "react";
import { cn } from "../lib/utils";
import type { ViewMode } from "../lib/api";
import { getFileTypeInfo } from "../lib/fileTypes";

interface ViewModeSelectorProps {
  mode: ViewMode;
  onChange: (mode: ViewMode) => void;
  disabled?: boolean;
  className?: string;
  compact?: boolean;
  /** File path to determine if preview is available */
  filePath?: string;
  /** Whether the file has git changes (shows Diff/Full+Diff buttons) */
  hasDiff?: boolean;
}

interface ModeOption {
  value: ViewMode;
  label: string;
  shortLabel: string;
  description: string;
}

const baseModes: ModeOption[] = [
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

const previewMode: ModeOption = {
  value: "preview",
  label: "Preview",
  shortLabel: "View",
  description: "Preview rendered content"
};

export function ViewModeSelector({
  mode,
  onChange,
  disabled = false,
  className,
  compact = false,
  filePath,
  hasDiff = true
}: ViewModeSelectorProps) {
  const modes = useMemo(() => {
    // Start with base modes, filtering out diff modes if no changes
    let availableModes = hasDiff
      ? baseModes
      : baseModes.filter((m) => m.value !== "diff" && m.value !== "full_diff");

    // Add preview mode if file type supports it
    if (filePath) {
      const fileType = getFileTypeInfo(filePath);
      if (fileType.canPreview) {
        availableModes = [...availableModes, previewMode];
      }
    }

    return availableModes;
  }, [filePath, hasDiff]);

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
