/**
 * FlowEmptyState - Shown when the Flow lens is active but no focus node is set.
 */

import { History } from "lucide-react";

interface FlowEmptyStateProps {
  onGoToTopology: () => void;
  hint?: string | null;
}

export function FlowEmptyState({ onGoToTopology, hint }: FlowEmptyStateProps) {
  return (
    <div
      className="flex h-full w-full items-center justify-center"
      data-testid="flow-empty-state"
    >
      <div className="rounded-xl border border-slate-700/70 bg-slate-950/90 px-8 py-6 text-center shadow-lg max-w-sm">
        <History className="mx-auto h-10 w-10 text-slate-500 mb-3" />
        <p className="text-sm font-medium text-slate-100">
          {hint || "Select a node in Topology to view its execution history"}
        </p>
        <button
          type="button"
          onClick={onGoToTopology}
          className="mt-4 rounded-lg bg-cyan-600/20 px-4 py-2 text-sm font-medium text-cyan-300 transition-colors hover:bg-cyan-600/30"
          data-testid="flow-empty-go-topology"
        >
          Go to Topology
        </button>
      </div>
    </div>
  );
}
