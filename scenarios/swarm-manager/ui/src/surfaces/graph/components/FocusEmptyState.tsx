/**
 * FocusEmptyState - Shown when the Focus lens has no nodes needing attention.
 */

import { useSearchParams } from "react-router-dom";
import { CheckCircle } from "lucide-react";

export function FocusEmptyState() {
  const [, setSearchParams] = useSearchParams();

  const goToLens = (lens: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("lens", lens);
      return next;
    });
  };

  return (
    <div
      className="absolute left-1/2 top-1/2 z-10 -translate-x-1/2 -translate-y-1/2"
      data-testid="focus-empty-state"
    >
      <div className="rounded-xl border border-slate-700/70 bg-slate-950/90 px-8 py-6 text-center shadow-lg max-w-sm">
        <CheckCircle className="mx-auto h-10 w-10 text-emerald-400 mb-3" />
        <p className="text-sm font-medium text-slate-100">
          Nothing needs your attention!
        </p>
        <p className="mt-1 text-xs text-slate-400">
          You're all caught up.
        </p>
        <div className="mt-4 flex items-center justify-center gap-2">
          <button
            type="button"
            onClick={() => goToLens("topology")}
            className="rounded-lg bg-cyan-600/20 px-4 py-2 text-sm font-medium text-cyan-300 transition-colors hover:bg-cyan-600/30"
            data-testid="focus-empty-go-topology"
          >
            Go to Topology
          </button>
          <button
            type="button"
            onClick={() => goToLens("operations")}
            className="rounded-lg bg-cyan-600/20 px-4 py-2 text-sm font-medium text-cyan-300 transition-colors hover:bg-cyan-600/30"
            data-testid="focus-empty-go-operations"
          >
            Go to Operations
          </button>
        </div>
      </div>
    </div>
  );
}
