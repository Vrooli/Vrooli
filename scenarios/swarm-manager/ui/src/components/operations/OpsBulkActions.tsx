/**
 * OpsBulkActions — sticky bottom bar for bulk-stop affordances.
 *
 * Visibility gate: the bar is hidden entirely unless the operator has
 * explicitly entered selection mode (via the "Select" toggle next to the
 * view tabs). Inside selection mode it shows whenever at least one row is
 * selected (Stop selected) or any active row exists (Stop all running).
 * Both stop actions are gated by `<ConfirmDialog>`. "Stop all running"
 * requires the operator to type `STOP ALL` because it cancels every
 * active run regardless of selection — a wider blast radius warrants a
 * wider confirmation surface.
 *
 * After a stop call resolves, the component surfaces a transient outcome
 * panel summarizing how many runs stopped vs. failed. The panel sits
 * above the action buttons inside the same sticky container so it does
 * not displace page content; operators dismiss it with the inline X
 * button or by issuing another stop.
 *
 * Wiring rules:
 *   - The component never reads `OperationsView` directly. It pulls
 *     selection / in-flight / counts from the operations-store so the
 *     shape mirrors the activity-row interactions.
 *   - The container always renders (so the layout is stable), but its
 *     sticky shell only shows when there is something to act on; this
 *     avoids the bar bouncing into existence on every refresh tick.
 */

import { useState } from "react";
import { Loader2, X } from "lucide-react";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { selectors } from "../../consts/selectors";
import { cn } from "../../lib/utils";
import {
  selectActiveCount,
  useOperationsStore,
} from "../../stores/operations-store";

const STOP_ALL_CONFIRMATION = "STOP ALL";

export interface OpsBulkActionsProps {
  /** Optional class on the outer container (positioning hooks for tests / pages). */
  className?: string;
}

export function OpsBulkActions({ className }: OpsBulkActionsProps) {
  const selectionMode = useOperationsStore((s) => s.selectionMode);
  const selectionSize = useOperationsStore((s) => s.selection.size);
  const activeCount = useOperationsStore(selectActiveCount);
  const isBulkStopping = useOperationsStore((s) => s.isBulkStopping);
  const lastResult = useOperationsStore((s) => s.lastBulkStopResult);
  const clearSelection = useOperationsStore((s) => s.clearSelection);
  const bulkStopSelected = useOperationsStore((s) => s.bulkStopSelected);
  const bulkStopAll = useOperationsStore((s) => s.bulkStopAll);

  const [stopSelectedOpen, setStopSelectedOpen] = useState(false);
  const [stopAllOpen, setStopAllOpen] = useState(false);
  const [outcomeDismissed, setOutcomeDismissed] = useState(false);

  // Outcome surfaces even after selection mode is turned off so the
  // operator can read the result of their stop without having to flip the
  // toggle back on. It self-dismisses on the inline X.
  const showOutcome = lastResult !== null && !outcomeDismissed;
  const visible = selectionMode && (selectionSize > 0 || activeCount > 0);

  if (!visible && !showOutcome) {
    return null;
  }

  const onConfirmStopSelected = async () => {
    setStopSelectedOpen(false);
    setOutcomeDismissed(false);
    try {
      await bulkStopSelected();
    } catch {
      // Error state is captured on the store; avoid throwing past the click.
    }
  };

  const onConfirmStopAll = async () => {
    setStopAllOpen(false);
    setOutcomeDismissed(false);
    try {
      await bulkStopAll();
    } catch {
      // Error state is captured on the store; avoid throwing past the click.
    }
  };

  return (
    <div
      className={cn(
        "sticky bottom-0 left-0 right-0 z-20 border-t border-white/10 bg-slate-900/95 px-4 py-3 backdrop-blur-sm",
        className,
      )}
      data-testid={selectors.operationsCenter.bulkActionBar}
      data-visible={visible ? "true" : "false"}
    >
      {showOutcome && (
        <BulkStopOutcomePanel
          stopped={lastResult?.stopped ?? 0}
          failed={lastResult?.failed ?? 0}
          total={lastResult?.total ?? 0}
          firstError={
            lastResult?.outcomes.find((o) => !o.success)?.error ?? null
          }
          onDismiss={() => setOutcomeDismissed(true)}
        />
      )}

      {visible && (
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-xs text-slate-400">
          {selectionSize > 0 ? (
            <>
              <span className="font-medium text-slate-200">
                {selectionSize} selected
              </span>
              {" — "}choose an action below.
            </>
          ) : (
            <>
              <span className="font-medium text-slate-200">
                {activeCount} active
              </span>
              {" — "}select rows above or stop everything at once.
            </>
          )}
        </p>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="destructive"
            size="sm"
            disabled={selectionSize === 0 || isBulkStopping}
            onClick={() => setStopSelectedOpen(true)}
            data-testid={selectors.operationsCenter.bulkStopSelected}
          >
            {isBulkStopping ? (
              <Loader2 className="mr-2 h-4 w-4 animate-spin" aria-hidden />
            ) : null}
            {selectionSize > 0
              ? `Stop selected (${selectionSize})`
              : "Stop selected"}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            disabled={activeCount === 0 || isBulkStopping}
            onClick={() => setStopAllOpen(true)}
            data-testid={selectors.operationsCenter.bulkStopAll}
          >
            Stop all running
          </Button>
          {selectionSize > 0 && (
            <Button
              variant="ghost"
              size="sm"
              onClick={clearSelection}
              data-testid={selectors.operationsCenter.bulkClearSelection}
            >
              Clear selection
            </Button>
          )}
        </div>
      </div>
      )}

      <ConfirmDialog
        isOpen={stopSelectedOpen}
        onClose={() => setStopSelectedOpen(false)}
        onConfirm={onConfirmStopSelected}
        title={`Stop ${selectionSize} selected ${selectionSize === 1 ? "run" : "runs"}?`}
        description="Stopped runs cannot be resumed. The agents will exit the next time they yield control; their work-in-progress branches survive."
        confirmLabel={`Stop ${selectionSize}`}
        isLoading={isBulkStopping}
        testIds={{ dialog: selectors.operationsCenter.bulkStopConfirmDialog }}
      />

      <ConfirmDialog
        isOpen={stopAllOpen}
        onClose={() => setStopAllOpen(false)}
        onConfirm={onConfirmStopAll}
        title={`Stop all ${activeCount} running ${activeCount === 1 ? "run" : "runs"}?`}
        description="Every active agent run is cancelled. Type STOP ALL exactly to confirm — this affects every initiative, capture, and standalone item currently in flight."
        confirmationText={STOP_ALL_CONFIRMATION}
        confirmLabel={`Stop ${activeCount}`}
        isLoading={isBulkStopping}
        testIds={{ dialog: selectors.operationsCenter.bulkStopAllConfirmDialog }}
      />
    </div>
  );
}

interface BulkStopOutcomePanelProps {
  stopped: number;
  failed: number;
  total: number;
  firstError: string | null;
  onDismiss: () => void;
}

function BulkStopOutcomePanel({
  stopped,
  failed,
  total,
  firstError,
  onDismiss,
}: BulkStopOutcomePanelProps) {
  const tone =
    failed === 0 ? "border-emerald-400/40 bg-emerald-500/10 text-emerald-200"
    : stopped === 0 ? "border-rose-400/40 bg-rose-500/10 text-rose-200"
    : "border-amber-400/40 bg-amber-500/10 text-amber-200";

  return (
    <div
      className={cn(
        "mb-2 flex items-start gap-3 rounded-md border px-3 py-2 text-xs",
        tone,
      )}
      data-testid={selectors.operationsCenter.bulkStopOutcomeToast}
      role="status"
      aria-live="polite"
    >
      <div className="min-w-0 flex-1">
        <p className="font-medium">
          {failed === 0
            ? `Stopped ${stopped} ${stopped === 1 ? "run" : "runs"}.`
            : stopped === 0
              ? `Failed to stop ${failed} of ${total}.`
              : `Stopped ${stopped} of ${total}; ${failed} failed.`}
        </p>
        {firstError && (
          <p className="mt-0.5 truncate text-[11px] opacity-80">
            First error: {firstError}
          </p>
        )}
      </div>
      <button
        type="button"
        className="shrink-0 rounded p-0.5 transition-colors hover:bg-white/10"
        onClick={onDismiss}
        aria-label="Dismiss outcome"
      >
        <X className="h-3.5 w-3.5" aria-hidden />
      </button>
    </div>
  );
}
