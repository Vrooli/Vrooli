/**
 * Sticky toolbar that appears when items are selected, offering batch actions.
 */
import { CheckCircle2, AlertTriangle, Sparkles, X } from "lucide-react";
import { cn } from "../../lib";

interface BulkActionToolbarProps {
  selectedCount: number;
  onApproveSelected: () => void;
  onFlagSelected: () => void;
  onSendToAgent: () => void;
  onClearSelection: () => void;
  disabled?: boolean;
}

export function BulkActionToolbar({
  selectedCount,
  onApproveSelected,
  onFlagSelected,
  onSendToAgent,
  onClearSelection,
  disabled,
}: BulkActionToolbarProps) {
  if (selectedCount === 0) return null;

  const btnBase = "flex items-center justify-center gap-1.5 whitespace-nowrap rounded-md px-3 py-2 text-xs font-medium transition-colors";

  return (
    <div className="sticky bottom-0 z-10 mt-3 overflow-hidden rounded-lg border border-slate-700/80 bg-slate-900/97 shadow-xl shadow-black/30 backdrop-blur-md">
      <div className="flex items-center gap-2 px-3 py-2.5 sm:px-4">
        {/* Count + clear */}
        <div className="flex items-center gap-1.5 mr-auto">
          <span className="inline-flex h-5 min-w-[20px] items-center justify-center rounded-full bg-cyan-500/20 px-1.5 text-[11px] font-semibold tabular-nums text-cyan-400">
            {selectedCount}
          </span>
          <button
            type="button"
            onClick={onClearSelection}
            className="rounded-full p-0.5 text-slate-500 transition-colors hover:bg-slate-800 hover:text-slate-300"
            title="Clear selection"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>

        {/* Action buttons
         *
         * Approve/Flag always show labels — their checkmark/triangle icons are
         * generic enough to be ambiguous without text (approve what? flag how?).
         *
         * Agent uses only the sparkle icon on mobile — the ✨ sparkle is now
         * universally associated with AI actions, so the label adds no clarity
         * but would cause wrapping on narrow screens (the original "Send to
         * Agent" text broke across 3 lines on mobile, which prompted this). */}
        <button
          type="button"
          disabled={disabled}
          onClick={onApproveSelected}
          className={cn(
            btnBase,
            "border border-emerald-500/30 bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20",
            disabled && "opacity-50 cursor-not-allowed",
          )}
        >
          <CheckCircle2 className="h-3.5 w-3.5" />
          Approve
        </button>
        <button
          type="button"
          disabled={disabled}
          onClick={onFlagSelected}
          className={cn(
            btnBase,
            "border border-amber-500/30 bg-amber-500/10 text-amber-400 hover:bg-amber-500/20",
            disabled && "opacity-50 cursor-not-allowed",
          )}
        >
          <AlertTriangle className="h-3.5 w-3.5" />
          Flag
        </button>
        <button
          type="button"
          disabled={disabled}
          onClick={onSendToAgent}
          className={cn(
            btnBase,
            "border border-cyan-500/30 bg-cyan-500/10 text-cyan-400 hover:bg-cyan-500/20",
            disabled && "opacity-50 cursor-not-allowed",
          )}
          title="Send to Agent"
        >
          <Sparkles className="h-3.5 w-3.5" />
          <span className="hidden sm:inline">Agent</span>
        </button>
      </div>
    </div>
  );
}
