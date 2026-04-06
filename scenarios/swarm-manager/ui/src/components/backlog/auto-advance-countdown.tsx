/**
 * Inline countdown component for deferred auto-advance.
 * Shows remaining seconds before the next workshop round spawns,
 * with a cancel button to abort the pending advance.
 */
import { useState, useEffect, useCallback } from "react";
import { Loader2, X, Timer } from "lucide-react";
import { backlogService } from "../../services/backlog-service";
import type { BacklogKind } from "../../types";

interface AutoAdvanceCountdownProps {
  advanceAt: string;
  delaySeconds: number;
  nextMode: "workshop" | "finalize";
  kind: BacklogKind;
  name: string;
  onCancelled: () => void;
  onExpired: () => void;
}

export function AutoAdvanceCountdown({
  advanceAt,
  nextMode,
  kind,
  name,
  onCancelled,
  onExpired,
}: AutoAdvanceCountdownProps) {
  const [remaining, setRemaining] = useState(() => {
    const diff = Math.ceil((new Date(advanceAt).getTime() - Date.now()) / 1000);
    return Math.max(0, diff);
  });
  const [cancelling, setCancelling] = useState(false);
  const [expired, setExpired] = useState(false);

  useEffect(() => {
    if (remaining <= 0) {
      setExpired(true);
      onExpired();
      return;
    }
    const timer = setInterval(() => {
      const diff = Math.ceil((new Date(advanceAt).getTime() - Date.now()) / 1000);
      const next = Math.max(0, diff);
      setRemaining(next);
      if (next <= 0) {
        setExpired(true);
        onExpired();
      }
    }, 1000);
    return () => clearInterval(timer);
  }, [advanceAt, remaining, onExpired]);

  const handleCancel = useCallback(async () => {
    setCancelling(true);
    try {
      await backlogService.workshopCancelPendingAdvance(kind, name);
      onCancelled();
    } catch {
      setCancelling(false);
    }
  }, [kind, name, onCancelled]);

  const label = nextMode === "finalize" ? "Finalizing" : "Next workshop round";

  if (expired) {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-cyan-500/20 bg-cyan-500/[0.03] px-3 py-2.5 text-sm text-cyan-300">
        <Loader2 className="h-4 w-4 animate-spin shrink-0" />
        {nextMode === "finalize" ? "Finalizing..." : "Generating next workshop round..."}
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2 rounded-lg border border-amber-500/20 bg-amber-500/[0.03] px-3 py-2.5 text-sm text-amber-300">
      <Timer className="h-4 w-4 shrink-0" />
      <span className="flex-1">
        {label} in <span className="font-mono font-medium">{remaining}s</span>...
      </span>
      <button
        type="button"
        disabled={cancelling}
        onClick={handleCancel}
        className="flex items-center gap-1 rounded border border-amber-500/30 px-1.5 py-0.5 text-xs text-amber-400 transition-colors hover:bg-amber-500/10 disabled:opacity-50"
      >
        {cancelling ? (
          <Loader2 className="h-3 w-3 animate-spin" />
        ) : (
          <X className="h-3 w-3" />
        )}
        Cancel
      </button>
    </div>
  );
}
