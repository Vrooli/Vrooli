import { useEffect, useMemo, useState } from "react";
import { AlertTriangle, Loader2, RefreshCw } from "lucide-react";
import { Button } from "../../ui/button";
import { Dialog } from "../../ui/dialog";
import { selectors } from "../../../consts/selectors";
import type {
  ActiveItemExecution,
  OperatingModeCatalogEntry,
} from "../../../types/operating-mode";
import type { InitiativeOperatingMode } from "../../../types";
import {
  type ActiveItemExecutionsConflict,
  parseActiveItemExecutionsConflict,
} from "../../../services/initiative-mode-service";
import { useAgentRunUrl } from "../../../services/external-links";
import { ModeComparePanel } from "./mode-compare-panel";
import { OperatingModeCard } from "./operating-mode-card";

export interface ModePickerDialogProps {
  isOpen: boolean;
  onClose: () => void;
  currentMode: InitiativeOperatingMode;
  catalog: OperatingModeCatalogEntry[];
  catalogLoading: boolean;
  catalogError?: unknown;
  catalogFetching?: boolean;
  onRetryCatalog?: () => void;
  isMutating: boolean;
  mutationError?: unknown;
  onConfirm: (mode: InitiativeOperatingMode, cancelActiveItemExecutions: boolean) => void;
}

export function ModePickerDialog({
  isOpen,
  onClose,
  currentMode,
  catalog,
  catalogLoading,
  catalogError,
  catalogFetching,
  onRetryCatalog,
  isMutating,
  mutationError,
  onConfirm,
}: ModePickerDialogProps) {
  const switchableModes = catalog.filter((entry) => entry.switchable);
  const [selectedModeKey, setSelectedModeKey] = useState<InitiativeOperatingMode>(currentMode);
  const [cancelAck, setCancelAck] = useState(false);
  // Conflict driven by the server's 409 response when the first switch attempt
  // discovers active item executions. Surfaces the actual list of affected
  // items so the operator confirms cancellation against real data, not a
  // client-side heuristic.
  const [switchConflict, setSwitchConflict] = useState<ActiveItemExecutionsConflict | null>(null);

  // Reset internal state on every open so leftover deltas from a prior session
  // never bleed across two switch flows. Doing this on `isOpen` rather than
  // unmount avoids losing state if the dialog is re-mounted by parents.
  useEffect(() => {
    if (isOpen) {
      setSelectedModeKey(currentMode);
      setCancelAck(false);
      setSwitchConflict(null);
    }
  }, [isOpen, currentMode]);

  // Reset the conflict + ack when the user picks a different target mode —
  // the previous conflict was scoped to the prior selection.
  useEffect(() => {
    setSwitchConflict(null);
    setCancelAck(false);
  }, [selectedModeKey]);

  // Promote a 409 mutationError into a typed conflict the dialog can render.
  // Plain (non-conflict) mutationError keeps flowing through the inline error
  // block below.
  useEffect(() => {
    if (!mutationError) return;
    const parsed = parseActiveItemExecutionsConflict(mutationError);
    if (parsed) setSwitchConflict(parsed);
  }, [mutationError]);

  const currentEntry = catalog.find((entry) => entry.mode === currentMode);
  const selectedEntry = catalog.find((entry) => entry.mode === selectedModeKey);
  const isSameMode = selectedModeKey === currentMode;

  const requiresCancelAck = Boolean(switchConflict);
  const canSubmit =
    !isSameMode && !isMutating && (!requiresCancelAck || cancelAck);

  const handleConfirm = () => {
    if (!canSubmit) return;
    // First attempt sends cancel=false so the server tells us if there's a
    // conflict. Once a conflict is rendered and the operator acknowledges,
    // the second click resubmits with cancel=true.
    onConfirm(selectedModeKey, requiresCancelAck);
  };

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title="Switch Operating Mode"
      maxWidth="max-w-3xl"
      isLoading={isMutating}
      testId={selectors.initiativeDetails.modePicker}
    >
      <div className="space-y-4">
        <p className="text-sm text-slate-400">
          Pick a methodology for how this initiative is executed. Differences in scope, run strategy, and capabilities are summarized below.
        </p>

        {catalogLoading && (
          <div
            className="grid gap-2 sm:grid-cols-2 md:grid-cols-3"
            aria-label="Loading operating modes"
            role="status"
          >
            {[0, 1, 2].map((idx) => (
              <div
                key={idx}
                className="h-32 animate-pulse rounded-lg border border-slate-800 bg-slate-900/40"
              />
            ))}
          </div>
        )}
        {Boolean(catalogError) && (
          <div className="flex flex-wrap items-start justify-between gap-2 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            <p className="min-w-0 flex-1">
              {catalogError instanceof Error ? catalogError.message : "Failed to load operating modes."}
            </p>
            {onRetryCatalog && (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={onRetryCatalog}
                disabled={catalogFetching}
                data-testid={selectors.initiativeDetails.modePickerRetry}
              >
                {catalogFetching ? (
                  <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
                ) : (
                  <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
                )}
                {catalogFetching ? "Retrying…" : "Retry"}
              </Button>
            )}
          </div>
        )}

        {!catalogLoading && switchableModes.length > 0 && (
          <div className="grid gap-2 sm:grid-cols-2 md:grid-cols-3">
            {switchableModes.map((entry) => (
              <OperatingModeCard
                key={entry.mode}
                mode={entry}
                selected={entry.mode === selectedModeKey}
                onClick={() => setSelectedModeKey(entry.mode)}
                data-testid={selectors.initiativeDetails.modePickerCard}
              />
            ))}
          </div>
        )}

        {currentEntry && selectedEntry && (
          <ModeComparePanel current={currentEntry} selected={selectedEntry} />
        )}

        {switchConflict && (
          <ActiveItemExecutionsPreview conflict={switchConflict} />
        )}

        {requiresCancelAck && (
          <label className="flex cursor-pointer items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-100">
            <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
            <span className="flex-1">
              These executions will be canceled if you continue.
              <span className="mt-2 flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={cancelAck}
                  onChange={(event) => setCancelAck(event.target.checked)}
                  className="h-3.5 w-3.5 accent-amber-400"
                  data-testid={selectors.initiativeDetails.modePickerOverrideAck}
                />
                I understand — cancel active item executions and switch.
              </span>
            </span>
          </label>
        )}

        {/* Plain mutation errors that aren't the 409 conflict still surface
            inline. The 409 is promoted into <ActiveItemExecutionsPreview> above,
            so suppress its raw message here to avoid double-rendering. */}
        {Boolean(mutationError) && !switchConflict && (
          <p className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {mutationError instanceof Error ? mutationError.message : "Failed to switch mode."}
          </p>
        )}

        <div className="flex items-center justify-end gap-2 pt-1">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onClose}
            disabled={isMutating}
            data-testid={selectors.initiativeDetails.modePickerCancel}
          >
            Cancel
          </Button>
          <Button
            type="button"
            size="sm"
            onClick={handleConfirm}
            disabled={!canSubmit}
            data-testid={selectors.initiativeDetails.modePickerConfirm}
          >
            {isMutating && <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />}
            {isMutating
              ? "Switching…"
              : requiresCancelAck
                ? "Cancel executions and switch"
                : "Switch Mode"}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}

function ActiveItemExecutionsPreview({ conflict }: { conflict: ActiveItemExecutionsConflict }) {
  const visible = useMemo(() => conflict.executions.slice(0, 5), [conflict.executions]);
  const overflow = conflict.executions.length - visible.length;

  return (
    <div
      className="rounded-md border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-sm text-amber-100"
      data-testid={selectors.initiativeDetails.cancellationItemList}
    >
      <p className="text-[13px] font-medium">
        {conflict.executions.length} item execution
        {conflict.executions.length === 1 ? "" : "s"} currently running
      </p>
      <p className="mt-0.5 text-xs text-amber-200/80">
        These will be canceled if you continue with the switch.
      </p>
      <ul className="mt-2 space-y-1">
        {visible.map((execution) => (
          <ExecutionRow key={execution.executionId ?? execution.itemRef} execution={execution} />
        ))}
      </ul>
      {overflow > 0 && (
        <p className="mt-1 text-xs text-amber-200/70">
          …and {overflow} more.
        </p>
      )}
    </div>
  );
}

function ExecutionRow({ execution }: { execution: ActiveItemExecution }) {
  const runUrl = useAgentRunUrl(execution.runId);

  return (
    <li className="flex flex-wrap items-center gap-2 text-[12px]">
      <code className="rounded bg-slate-900/60 px-1.5 py-0.5 font-mono text-amber-100">
        {execution.itemRef}
      </code>
      {execution.status && (
        <span className="rounded-full border border-amber-400/40 bg-amber-400/10 px-2 py-0.5 text-[11px] text-amber-100">
          {execution.status}
        </span>
      )}
      {execution.runId && (
        runUrl ? (
          <a
            href={runUrl}
            target="_blank"
            rel="noreferrer"
            className="font-mono text-[11px] text-cyan-300 hover:text-cyan-200"
          >
            run {execution.runId.slice(0, 8)}…
          </a>
        ) : (
          <span className="font-mono text-[11px] text-amber-200/70">run {execution.runId.slice(0, 8)}…</span>
        )
      )}
    </li>
  );
}
