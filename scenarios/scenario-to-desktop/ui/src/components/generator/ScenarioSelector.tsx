/**
 * Scenario selector component for the generator form.
 * Displays either a locked scenario badge or a selectable scenario picker.
 */

import { Button } from "../ui/button";
import { Label } from "../ui/label";
import type { ScenarioDesktopStatus } from "../scenario-inventory/types";

export interface ScenarioSelectorProps {
  scenarioName: string;
  loadingScenarios: boolean;
  selectedScenario?: ScenarioDesktopStatus;
  onOpenScenarioModal: () => void;
  onLoadSaved?: () => void;
  locked?: boolean;
  onUnlock?: () => void;
}

export function ScenarioSelector({
  scenarioName,
  loadingScenarios,
  selectedScenario,
  onOpenScenarioModal,
  onLoadSaved,
  locked = false,
  onUnlock
}: ScenarioSelectorProps) {
  const displayName = selectedScenario?.display_name || scenarioName;

  if (locked && scenarioName) {
    return (
      <div className="rounded-lg border border-slate-800 bg-slate-950/60 p-3 flex items-center justify-between gap-3">
        <div>
          <p className="text-xs uppercase tracking-wide text-slate-400">Scenario</p>
          <p className="text-sm font-semibold text-slate-100">{scenarioName}</p>
        </div>
        {onUnlock && (
          <button
            type="button"
            onClick={onUnlock}
            className="text-xs text-blue-300 underline hover:text-blue-200"
          >
            Change scenario
          </button>
        )}
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-1.5 gap-3">
        <Label htmlFor="scenarioName">Scenario Name</Label>
        <div className="flex items-center gap-2">
          {selectedScenario?.connection_config && onLoadSaved && (
            <button
              type="button"
              onClick={onLoadSaved}
              className="text-xs text-blue-300 hover:text-blue-200"
            >
              Load saved URLs
            </button>
          )}
          <button
            type="button"
            onClick={onOpenScenarioModal}
            className="text-xs text-blue-400 hover:text-blue-300"
          >
            Browse scenarios
          </button>
        </div>
      </div>

      <div className="mt-1.5 rounded-md border border-slate-800 bg-slate-950/60 p-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p className="text-sm font-semibold text-slate-100">
              {displayName || (loadingScenarios ? "Loading scenarios..." : "Select a scenario")}
            </p>
            <p className="text-xs text-slate-400">
              {scenarioName ? `Slug: ${scenarioName}` : "Browse scenarios to choose a desktop target."}
            </p>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={onOpenScenarioModal}>
            Browse scenarios
          </Button>
        </div>
      </div>
      <p className="mt-1.5 text-xs text-slate-400">
        Select from available scenarios or enter a slug from the modal.
      </p>
    </div>
  );
}
