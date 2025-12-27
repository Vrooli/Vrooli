/**
 * VersionPicker - ChatGPT-style version navigation for branched messages.
 *
 * Shows the current version number (e.g., "2/3") with navigation arrows
 * to switch between alternative responses. Only shown when a message
 * has siblings (alternative versions).
 */
import { ChevronLeft, ChevronRight } from "lucide-react";
import { Tooltip } from "../ui/tooltip";

interface VersionPickerProps {
  /** Current version (1-based index) */
  current: number;
  /** Total number of versions */
  total: number;
  /** Called when user navigates to previous version */
  onPrevious: () => void;
  /** Called when user navigates to next version */
  onNext: () => void;
  /** Optional className for styling */
  className?: string;
  /** Whether navigation is disabled (e.g., during loading) */
  disabled?: boolean;
}

export function VersionPicker({
  current,
  total,
  onPrevious,
  onNext,
  className = "",
  disabled = false,
}: VersionPickerProps) {
  // Don't show if there's only one version
  if (total <= 1) return null;

  const hasPrevious = current > 1;
  const hasNext = current < total;

  return (
    <div
      className={`inline-flex items-center gap-0.5 text-xs ${className}`}
      data-testid="version-picker"
    >
      <Tooltip content="Previous version" side="top">
        <button
          onClick={onPrevious}
          disabled={disabled || !hasPrevious}
          className={`p-0.5 rounded transition-colors ${
            hasPrevious && !disabled
              ? "hover:bg-white/10 text-slate-400 hover:text-slate-200 cursor-pointer"
              : "text-slate-600 cursor-not-allowed"
          }`}
          aria-label="Previous version"
        >
          <ChevronLeft className="h-3.5 w-3.5" />
        </button>
      </Tooltip>

      <span
        className="min-w-[2.5rem] text-center text-slate-400 select-none"
        title={`Version ${current} of ${total}`}
      >
        {current}/{total}
      </span>

      <Tooltip content="Next version" side="top">
        <button
          onClick={onNext}
          disabled={disabled || !hasNext}
          className={`p-0.5 rounded transition-colors ${
            hasNext && !disabled
              ? "hover:bg-white/10 text-slate-400 hover:text-slate-200 cursor-pointer"
              : "text-slate-600 cursor-not-allowed"
          }`}
          aria-label="Next version"
        >
          <ChevronRight className="h-3.5 w-3.5" />
        </button>
      </Tooltip>
    </div>
  );
}
