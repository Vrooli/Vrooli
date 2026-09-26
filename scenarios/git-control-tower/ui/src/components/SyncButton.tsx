import { PushSafetyNotice } from "./PushSafetyIndicators";
import { useState, useRef, useEffect } from "react";
import { ArrowUpDown, ArrowUp, ArrowDown, Loader2 } from "lucide-react";
import { Button } from "./ui/button";

interface SyncButtonProps {
  ahead: number;
  behind: number;
  canPush: boolean;
  canPull: boolean;
  onPush: () => void;
  onPull: () => void;
  isPushing: boolean;
  isPulling: boolean;
  /** Phase and elapsed time of the running operation, e.g. "Pushing 1m 12s". */
  progressLabel?: string;
  warning?: string;
}

export function SyncButton({
  ahead,
  behind,
  canPush,
  canPull,
  onPush,
  onPull,
  isPushing,
  isPulling,
  progressLabel,
  warning
}: SyncButtonProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const isActive = isPushing || isPulling;

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    window.addEventListener("mousedown", handler);
    return () => window.removeEventListener("mousedown", handler);
  }, [open]);

  // Keep the menu open while an operation runs: a push can take minutes, and the popover
  // is where its phase and elapsed time are legible. Close it again once it finishes so
  // the result is read from the toast rather than a stale menu.
  const wasActive = useRef(false);
  useEffect(() => {
    if (isActive) {
      setOpen(true);
    } else if (wasActive.current) {
      setOpen(false);
    }
    wasActive.current = isActive;
  }, [isActive]);

  if (ahead === 0 && behind === 0 && !isActive) return null;

  const pushLabel = isPushing
    ? progressLabel ?? "Pushing…"
    : `Push ${ahead} commit${ahead !== 1 ? "s" : ""}`;
  const pullLabel = isPulling
    ? progressLabel ?? "Pulling…"
    : `Pull ${behind} commit${behind !== 1 ? "s" : ""}`;

  return (
    <div className="relative" ref={ref} data-testid="sync-button">
      <PushSafetyNotice />
      <button
        className="flex items-center gap-1.5 px-2 py-1 rounded-md hover:bg-slate-800 transition-colors text-sm"
        onClick={() => setOpen((prev) => !prev)}
        title={isActive ? progressLabel ?? "Syncing with remote" : warning || "Sync with remote"}
        data-testid="sync-button-trigger"
      >
        {isActive ? (
          <Loader2 className="h-4 w-4 text-slate-400 animate-spin" />
        ) : (
          <ArrowUpDown className="h-4 w-4 text-slate-400" />
        )}
        {ahead > 0 && (
          <span className="flex items-center gap-0.5 text-xs text-emerald-400">
            <ArrowUp className="h-3 w-3" />
            {ahead}
          </span>
        )}
        {behind > 0 && (
          <span className="flex items-center gap-0.5 text-xs text-amber-400">
            <ArrowDown className="h-3 w-3" />
            {behind}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute left-0 mt-2 w-56 rounded-lg border border-slate-800 bg-slate-950/95 p-2 shadow-xl z-50">
          <div className="space-y-1">
            {(ahead > 0 || isPushing) && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => { onPush(); setOpen(false); }}
                disabled={isActive || !canPush}
                className="w-full justify-start gap-2 h-8 text-xs"
                data-testid="sync-push-button"
              >
                {isPushing ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <ArrowUp className="h-3.5 w-3.5 text-emerald-400" />
                )}
                {pushLabel}
              </Button>
            )}
            {(behind > 0 || isPulling) && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => { onPull(); setOpen(false); }}
                disabled={isActive || !canPull}
                className="w-full justify-start gap-2 h-8 text-xs"
                data-testid="sync-pull-button"
              >
                {isPulling ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <ArrowDown className="h-3.5 w-3.5 text-amber-400" />
                )}
                {pullLabel}
              </Button>
            )}
            {isActive && (
              <p className="text-[11px] text-slate-400 px-2 py-1" data-testid="sync-progress-note">
                Large transfers can take several minutes. Leaving this panel does not cancel it.
              </p>
            )}
            {!isActive && !canPush && ahead > 0 && (
              <p className="text-[11px] text-amber-400 px-2 py-1">Pull required before push</p>
            )}
            {!isActive && warning && (
              <p className="text-[11px] text-amber-400 px-2 py-1">{warning}</p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
