/**
 * Header section for GeneratorForm.
 * Shows pipeline status summary, timestamps, saving indicator, and reset button.
 */

import { Button } from "../ui/button";
import { PipelineStatusSummary } from "../state/PipelineStatusOverview";
import type { ValidationStatus } from "../../lib/api";

export interface GeneratorFormHeaderProps {
  scenarioName: string;
  validationStatus: ValidationStatus | null;
  createdLabel: string | null;
  updatedLabel: string | null;
  isSaving: boolean;
  onReset: () => void;
}

export function GeneratorFormHeader({
  scenarioName,
  validationStatus,
  createdLabel,
  updatedLabel,
  isSaving,
  onReset,
}: GeneratorFormHeaderProps) {
  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex flex-wrap items-center gap-3">
        {scenarioName && (
          <PipelineStatusSummary validationStatus={validationStatus} />
        )}
        {(createdLabel || updatedLabel) && (
          <div className="flex flex-wrap items-center gap-2 text-xs text-slate-400">
            {createdLabel && (
              <span>Started {createdLabel}</span>
            )}
            {updatedLabel && (
              <span>Saved {updatedLabel}</span>
            )}
            {isSaving && (
              <span className="text-blue-400">Saving...</span>
            )}
          </div>
        )}
      </div>
      <Button
        type="button"
        size="sm"
        variant="outline"
        className="sm:ml-auto"
        disabled={!scenarioName}
        onClick={onReset}
      >
        Reset progress
      </Button>
    </div>
  );
}
