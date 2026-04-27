import { Loader2, Play, Sparkles } from "lucide-react";
import { AutoAdvanceCountdown } from "./auto-advance-countdown";
import type { WorkshopAutoAdvance } from "../../services/backlog/types";
import type { BacklogKind } from "../../types";

interface WorkshopTransitionStatusProps {
  autoAdvance: WorkshopAutoAdvance;
  kind: BacklogKind;
  name: string;
  title?: string;
  onCancelled: () => void;
  onExpired: () => void;
  onRunNext?: () => void;
  onFinalize?: () => void;
}

export function WorkshopTransitionStatus({
  autoAdvance,
  kind,
  name,
  title,
  onCancelled,
  onExpired,
  onRunNext,
  onFinalize,
}: WorkshopTransitionStatusProps) {
  const nextMode = autoAdvance.nextMode ?? "workshop";

  if (autoAdvance.pending && autoAdvance.advanceAt) {
    return (
      <AutoAdvanceCountdown
        advanceAt={autoAdvance.advanceAt}
        delaySeconds={autoAdvance.delaySeconds ?? 10}
        nextMode={nextMode}
        kind={kind}
        name={name}
        onCancelled={onCancelled}
        onExpired={onExpired}
      />
    );
  }

  if (autoAdvance.triggered) {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-cyan-500/20 bg-cyan-500/[0.03] px-3 py-2.5 text-sm text-cyan-300">
        <Loader2 className="h-4 w-4 shrink-0 animate-spin" />
        {nextMode === "finalize" ? "Finalizing..." : "Generating next workshop round..."}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-slate-700 bg-slate-900/60 px-3 py-2.5 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <p className="text-sm font-medium text-slate-200">
          {title ?? (nextMode === "finalize" ? "Ready to finalize" : "Ready for another workshop round")}
        </p>
        <p className="text-xs text-slate-400">
          {nextMode === "finalize" ? "The backend says the next step is finalization." : "The backend says another workshop round is available."}
        </p>
      </div>
      {nextMode === "finalize" ? (
        <button
          type="button"
          onClick={onFinalize}
          className="inline-flex min-h-[36px] items-center justify-center gap-1.5 rounded border border-emerald-500/30 bg-emerald-500/10 px-3 py-1.5 text-sm text-emerald-200 hover:bg-emerald-500/20"
        >
          <Sparkles className="h-4 w-4" />
          Finalize
        </button>
      ) : (
        <button
          type="button"
          onClick={onRunNext}
          className="inline-flex min-h-[36px] items-center justify-center gap-1.5 rounded border border-cyan-500/30 bg-cyan-500/10 px-3 py-1.5 text-sm text-cyan-200 hover:bg-cyan-500/20"
        >
          <Play className="h-4 w-4" />
          Run next round
        </button>
      )}
    </div>
  );
}
