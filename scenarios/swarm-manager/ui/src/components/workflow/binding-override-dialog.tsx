/**
 * BindingOverrideDialog
 *
 * Picks which authored mode implements ONE operation for a target scope.
 * Mode choices come exclusively from the server's ListCompatibleModes
 * verdicts for this target + operation — incompatible modes are never
 * offered, and this component performs no client-side compatibility
 * judgement beyond displaying the server's verdicts.
 */

import { useEffect, useMemo, useState } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { selectors } from "../../consts/selectors";
import { compatibleModesForOperation, shortDigest } from "../../lib/agent-ops-utils";
import type { WorkflowCompatibleMode } from "../../types/agent-operations";

export interface BindingOverrideDialogProps {
  isOpen: boolean;
  onClose: () => void;
  /** Operation being overridden (contract id). */
  operation: string;
  /** Exact contract version the override pins (from the resolved-binding row). */
  operationVersion: string;
  /** Currently-winning mode id, preselected when compatible. */
  currentMode?: string;
  /** Server-computed catalog with per-operation verdicts for this target. */
  compatibleModes: WorkflowCompatibleMode[];
  modesLoading: boolean;
  modesError?: unknown;
  isMutating: boolean;
  mutationError?: unknown;
  onConfirm: (mode: WorkflowCompatibleMode) => void;
}

export function BindingOverrideDialog({
  isOpen,
  onClose,
  operation,
  operationVersion,
  currentMode,
  compatibleModes,
  modesLoading,
  modesError,
  isMutating,
  mutationError,
  onConfirm,
}: BindingOverrideDialogProps) {
  const offered = useMemo(
    () => compatibleModesForOperation(compatibleModes, operation, operationVersion),
    [compatibleModes, operation, operationVersion],
  );

  const [selectedMode, setSelectedMode] = useState<string>("");

  // Re-seed the selection each open so a prior override flow never bleeds in.
  useEffect(() => {
    if (!isOpen) return;
    const preselect = offered.find((mode) => mode.mode === currentMode);
    setSelectedMode(preselect?.mode ?? offered[0]?.mode ?? "");
  }, [isOpen, currentMode, offered]);

  const selectedEntry = offered.find((mode) => mode.mode === selectedMode);
  const canSubmit = Boolean(selectedEntry) && !isMutating;

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title={`Override binding — ${operation}`}
      maxWidth="max-w-lg"
      isLoading={isMutating}
      testId={selectors.workflowBindings.overrideDialog}
    >
      <div className="space-y-4">
        <p className="text-sm text-slate-400">
          Pick which mode implements{" "}
          <code className="rounded bg-slate-900/70 px-1 py-0.5 font-mono text-[12px] text-slate-300">
            {operation}
            {operationVersion ? `@${operationVersion}` : ""}
          </code>{" "}
          for this scope. Only modes the server verified as compatible are offered.
        </p>

        {modesLoading && (
          <p className="flex items-center gap-2 text-sm text-slate-400" role="status">
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            Loading compatible modes…
          </p>
        )}
        {Boolean(modesError) && (
          <p className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {modesError instanceof Error ? modesError.message : "Failed to load compatible modes."}
          </p>
        )}

        {!modesLoading && !modesError && offered.length === 0 && (
          <p
            className="rounded-md border border-amber-500/30 bg-amber-500/5 px-3 py-2 text-sm text-amber-200"
            data-testid={selectors.workflowBindings.overrideDialogEmpty}
          >
            No compatible modes for this operation — nothing can be offered.
          </p>
        )}

        {offered.length > 0 && (
          <ul className="space-y-1.5" role="radiogroup" aria-label="Compatible modes">
            {offered.map((mode) => {
              const selected = mode.mode === selectedMode;
              return (
                <li key={mode.mode}>
                  <button
                    type="button"
                    role="radio"
                    aria-checked={selected}
                    onClick={() => setSelectedMode(mode.mode)}
                    data-testid={selectors.workflowBindings.overrideDialogModeOption}
                    className={`flex w-full flex-wrap items-center justify-between gap-2 rounded-md border px-3 py-2 text-left text-sm transition-colors ${
                      selected
                        ? "border-cyan-400/60 bg-cyan-500/10 text-slate-100 ring-1 ring-cyan-400/40"
                        : "border-slate-800 bg-slate-900/40 text-slate-300 hover:border-slate-700"
                    }`}
                  >
                    <span className="font-medium">{mode.mode}</span>
                    <span className="flex items-center gap-2 font-mono text-[11px] text-slate-500">
                      <span>rev {mode.modeRevision || "—"}</span>
                      {mode.modeDigest && <span>{shortDigest(mode.modeDigest)}</span>}
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}

        <p className="text-xs text-slate-500">
          Bindings are snapshotted when an operation starts — this change applies to operations
          started after this change; running and completed operations keep their pinned binding.
        </p>

        {Boolean(mutationError) && (
          <p className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {mutationError instanceof Error ? mutationError.message : "Failed to save the override."}
          </p>
        )}

        <div className="flex items-center justify-end gap-2 pt-1">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onClose}
            disabled={isMutating}
            data-testid={selectors.workflowBindings.overrideDialogCancel}
          >
            Cancel
          </Button>
          <Button
            type="button"
            size="sm"
            onClick={() => selectedEntry && onConfirm(selectedEntry)}
            disabled={!canSubmit}
            data-testid={selectors.workflowBindings.overrideDialogConfirm}
          >
            {isMutating && <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />}
            {isMutating ? "Saving…" : "Save override"}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
