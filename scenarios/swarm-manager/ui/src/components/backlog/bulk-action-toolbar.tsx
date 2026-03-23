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

  return (
    <div className="sticky bottom-0 z-10 mt-2 flex items-center gap-2 rounded-lg border border-slate-700 bg-slate-900/95 px-3 py-2 shadow-lg backdrop-blur-sm">
      <span className="text-xs font-medium text-slate-300">
        {selectedCount} selected
      </span>
      <div className="flex-1" />
      <button
        type="button"
        disabled={disabled}
        onClick={onApproveSelected}
        className={cn(
          "flex items-center gap-1 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-1.5 text-xs font-medium text-emerald-400 transition-colors hover:bg-emerald-500/20",
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
          "flex items-center gap-1 rounded-md border border-amber-500/30 bg-amber-500/10 px-2.5 py-1.5 text-xs font-medium text-amber-400 transition-colors hover:bg-amber-500/20",
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
          "flex items-center gap-1 rounded-md border border-cyan-500/30 bg-cyan-500/10 px-2.5 py-1.5 text-xs font-medium text-cyan-400 transition-colors hover:bg-cyan-500/20",
          disabled && "opacity-50 cursor-not-allowed",
        )}
      >
        <Sparkles className="h-3.5 w-3.5" />
        Send to Agent
      </button>
      <button
        type="button"
        onClick={onClearSelection}
        className="rounded p-1.5 text-slate-500 transition-colors hover:text-slate-300"
        title="Clear selection"
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
