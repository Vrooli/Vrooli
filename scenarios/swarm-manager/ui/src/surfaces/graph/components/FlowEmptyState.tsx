/**
 * FlowEmptyState - Shown when the Flow (History) lens has nothing to display.
 *
 * Two cases:
 * 1. No focus node selected → prompt user to pick one via Topology or Operations.
 * 2. Focus node exists but returned no history → show hint or fallback message.
 */

import { useSearchParams } from "react-router-dom";
import { History } from "lucide-react";

interface FlowEmptyStateProps {
  hasFocusNode: boolean;
  hint?: string | null;
}

export function FlowEmptyState({ hasFocusNode, hint }: FlowEmptyStateProps) {
  const [, setSearchParams] = useSearchParams();

  const goToLens = (lens: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("lens", lens);
      next.delete("focus");
      next.delete("returnLens");
      return next;
    });
  };

  const message = hasFocusNode
    ? (hint || "No history found for this item. Try selecting a different node.")
    : "Select a node in Topology or Operations to view its execution history.";

  return (
    <div
      className="absolute left-1/2 top-1/2 z-10 -translate-x-1/2 -translate-y-1/2"
      data-testid="flow-empty-state"
    >
      <div className="rounded-xl border border-slate-700/70 bg-slate-950/90 px-8 py-6 text-center shadow-lg max-w-sm">
        <History className="mx-auto h-10 w-10 text-slate-500 mb-3" />
        <p className="text-sm font-medium text-slate-100">{message}</p>
        <div className="mt-4 flex items-center justify-center gap-2">
          <button
            type="button"
            onClick={() => goToLens("topology")}
            className="rounded-lg bg-cyan-600/20 px-4 py-2 text-sm font-medium text-cyan-300 transition-colors hover:bg-cyan-600/30"
            data-testid="flow-empty-go-topology"
          >
            Go to Topology
          </button>
          <button
            type="button"
            onClick={() => goToLens("operations")}
            className="rounded-lg bg-cyan-600/20 px-4 py-2 text-sm font-medium text-cyan-300 transition-colors hover:bg-cyan-600/30"
            data-testid="flow-empty-go-operations"
          >
            Go to Operations
          </button>
        </div>
      </div>
    </div>
  );
}
