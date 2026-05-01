import { Save, Workflow } from "lucide-react";
import { Button } from "../../ui/button";
import { Select } from "../../ui/select";
import { selectors } from "../../../consts/selectors";
import type { InitiativeOperatingMode } from "../../../types";
import type { OperatingModeCatalogEntry } from "../../../types/operating-mode";
import { modeLabel } from "./utils";

export function ModeSwitchControl({
  currentMode,
  selectedMode,
  confirmItemCancellation,
  switchingAwayFromItemLevel,
  isPending,
  error,
  catalogModes,
  catalogLoading,
  catalogError,
  onSelectedModeChange,
  onConfirmItemCancellationChange,
  onSave,
}: {
  currentMode: InitiativeOperatingMode;
  selectedMode: InitiativeOperatingMode;
  confirmItemCancellation: boolean;
  switchingAwayFromItemLevel: boolean;
  isPending: boolean;
  error: unknown;
  catalogModes: OperatingModeCatalogEntry[];
  catalogLoading: boolean;
  catalogError: unknown;
  onSelectedModeChange: (mode: InitiativeOperatingMode) => void;
  onConfirmItemCancellationChange: (value: boolean) => void;
  onSave: () => void;
}) {
  const currentLabel = modeLabel(
    currentMode,
    catalogModes.find((mode) => mode.mode === currentMode)?.label,
  );
  const switchableModes = catalogModes.filter((mode) => mode.switchable);
  const switchDisabled = catalogLoading || Boolean(catalogError) || switchableModes.length === 0;

  return (
    <div className="rounded-lg border border-slate-800/80 bg-slate-900/55 p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
            <Workflow className="h-3.5 w-3.5" />
            Operating Mode
          </div>
          <p className="mt-2 text-sm text-slate-400">
            Current mode: <span className="font-medium text-slate-100">{currentLabel}</span>
          </p>
        </div>
        <div className="flex min-w-0 flex-col gap-2 sm:w-72">
          <Select
            value={selectedMode}
            onChange={(event) => {
              onSelectedModeChange(event.target.value);
              onConfirmItemCancellationChange(false);
            }}
            withChevron
            disabled={switchDisabled}
            data-testid={selectors.initiativeDetails.modeSelect}
          >
            {catalogLoading && <option value={selectedMode}>Loading modes...</option>}
            {!catalogLoading && switchableModes.map((mode) => (
              <option key={mode.mode} value={mode.mode}>{modeLabel(mode.mode, mode.label)}</option>
            ))}
          </Select>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              if (switchingAwayFromItemLevel && !confirmItemCancellation) {
                onConfirmItemCancellationChange(true);
                return;
              }
              onSave();
            }}
            disabled={selectedMode === currentMode || isPending || switchDisabled}
            data-testid={selectors.initiativeDetails.modeSave}
          >
            <Save className="mr-1.5 h-4 w-4" />
            {isPending ? "Saving..." : confirmItemCancellation ? "Confirm Switch" : "Save Mode"}
          </Button>
        </div>
      </div>
      {confirmItemCancellation && (
        <p className="mt-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-100">
          Switching from the item execution flow can cancel active member item executions. Click Confirm Switch to cancel any active item executions and change modes.
        </p>
      )}
      {Boolean(catalogError) && (
        <p className="mt-3 text-sm text-red-300">{catalogError instanceof Error ? catalogError.message : "Failed to load operating modes."}</p>
      )}
      {Boolean(error) && (
        <p className="mt-3 text-sm text-red-300">{error instanceof Error ? error.message : "Failed to save mode."}</p>
      )}
    </div>
  );
}
