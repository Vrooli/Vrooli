/**
 * Sticky toolbar that appears when items are selected, offering batch actions.
 */
import { CheckCircle2, AlertTriangle, X } from "lucide-react";
import { cn } from "../../lib";

interface BulkActionToolbarProps {
  selectedCount: number;
  onApproveSelected: () => void;
  onFlagSelected: () => void;
  onClearSelection: () => void;
  disabled?: boolean;
}

export function BulkActionToolbar({
  selectedCount,
  onApproveSelected,
  onFlagSelected,
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

        {/* Review actions remain explicit: Plan Workshop owns agent-assisted review. */}
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
      </div>
    </div>
  );
}
