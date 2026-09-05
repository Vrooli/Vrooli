import { useState } from "react";
import { Loader2, Trash2 } from "lucide-react";

interface ClearArchivedSectionProps {
  chatCount: number;
  onClearArchived: () => Promise<void>;
  isClearingArchived: boolean;
}

export function ClearArchivedSection({
  chatCount,
  onClearArchived,
  isClearingArchived,
}: ClearArchivedSectionProps) {
  const [showConfirm, setShowConfirm] = useState(false);

  return (
    <div className="px-3 py-2 border-b border-white/10 shrink-0">
      {!showConfirm ? (
        <button
          onClick={() => setShowConfirm(true)}
          className="w-full flex items-center justify-center gap-2 px-3 py-2 text-xs font-medium text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded-lg transition-colors"
          data-testid="clear-archived-button"
        >
          <Trash2 className="h-3.5 w-3.5" />
          Clear all archived
        </button>
      ) : (
        <div className="space-y-2">
          <p className="text-xs text-slate-400 text-center">
            Delete {chatCount} archived chat{chatCount !== 1 ? "s" : ""}?
          </p>
          <div className="flex gap-2">
            <button
              onClick={() => setShowConfirm(false)}
              className="flex-1 px-3 py-1.5 text-xs font-medium text-slate-400 hover:text-white bg-white/5 hover:bg-white/10 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={() => {
                void onClearArchived().then(() => { setShowConfirm(false); });
              }}
              disabled={isClearingArchived}
              className="flex-1 flex items-center justify-center gap-1 px-3 py-1.5 text-xs font-medium text-white bg-red-600 hover:bg-red-500 disabled:opacity-50 rounded-lg transition-colors"
              data-testid="confirm-clear-archived-button"
            >
              {isClearingArchived ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <Trash2 className="h-3 w-3" />
              )}
              Delete
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
