import { useEffect, useState } from "react";
import { AlertTriangle, Loader2 } from "lucide-react";
import { Button } from "../../ui/button";
import { Dialog } from "../../ui/dialog";
import { selectors } from "../../../consts/selectors";
import type {
  OperatingModeCatalogEntry,
} from "../../../types/operating-mode";
import type { InitiativeOperatingMode } from "../../../types";
import { ModeComparePanel } from "./mode-compare-panel";
import { OperatingModeCard } from "./operating-mode-card";

export interface ModePickerDialogProps {
  isOpen: boolean;
  onClose: () => void;
  currentMode: InitiativeOperatingMode;
  catalog: OperatingModeCatalogEntry[];
  catalogLoading: boolean;
  catalogError?: unknown;
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
  isMutating,
  mutationError,
  onConfirm,
}: ModePickerDialogProps) {
  const switchableModes = catalog.filter((entry) => entry.switchable);
  const [selectedModeKey, setSelectedModeKey] = useState<InitiativeOperatingMode>(currentMode);
  const [cancelAck, setCancelAck] = useState(false);

  // Reset internal state on every open so leftover deltas from a prior session
  // never bleed across two switch flows. Doing this on `isOpen` rather than
  // unmount avoids losing state if the dialog is re-mounted by parents.
  useEffect(() => {
    if (isOpen) {
      setSelectedModeKey(currentMode);
      setCancelAck(false);
    }
  }, [isOpen, currentMode]);

  const currentEntry = catalog.find((entry) => entry.mode === currentMode);
  const selectedEntry = catalog.find((entry) => entry.mode === selectedModeKey);
  const isSameMode = selectedModeKey === currentMode;

  const switchingFromItemExecution =
    Boolean(currentEntry?.capabilities.usesItemExecutionFlow) &&
    !selectedEntry?.capabilities.usesItemExecutionFlow;
  const requiresCancelAck = switchingFromItemExecution;
  const canSubmit =
    !isSameMode && !isMutating && (!requiresCancelAck || cancelAck);

  const handleConfirm = () => {
    if (!canSubmit) return;
    onConfirm(selectedModeKey, cancelAck);
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
          <p className="text-sm text-slate-500">Loading modes…</p>
        )}
        {Boolean(catalogError) && (
          <p className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {catalogError instanceof Error ? catalogError.message : "Failed to load operating modes."}
          </p>
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

        {requiresCancelAck && (
          <label className="flex cursor-pointer items-start gap-2 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-100">
            <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
            <span className="flex-1">
              Switching from item execution flow can cancel active member item executions.
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

        {Boolean(mutationError) && (
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
            {isMutating ? "Switching…" : "Switch Mode"}
          </Button>
        </div>
      </div>
    </Dialog>
  );
}
