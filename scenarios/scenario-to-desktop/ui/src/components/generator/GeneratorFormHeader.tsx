/**
 * Header section for GeneratorForm.
 * Shows pipeline status summary, timestamps, saving indicator, and reset button.
 */

import { RotateCcw } from "lucide-react";
import { Button } from "../ui/button";
import { PipelineStatusSummary } from "../state/PipelineStatusOverview";
import { useIsMobile } from "../../hooks/useMediaQuery";
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
  const isMobile = useIsMobile();

  return (
    <div className="flex items-center justify-between gap-2">
      <div className="flex flex-wrap items-center gap-2 md:gap-3 min-w-0">
        {scenarioName && (
          <PipelineStatusSummary validationStatus={validationStatus} />
        )}
        {(createdLabel || updatedLabel) && (
          <div className="flex flex-wrap items-center gap-x-2 text-xs text-slate-400">
            {createdLabel && <span>Started {createdLabel}</span>}
            {updatedLabel && (
              <>
                {createdLabel && <span className="text-slate-600">&middot;</span>}
                <span>Saved {updatedLabel}</span>
              </>
            )}
            {isSaving && <span className="text-blue-400">Saving...</span>}
          </div>
        )}
      </div>
      <Button
        type="button"
        size="sm"
        variant="outline"
        className="shrink-0"
        disabled={!scenarioName}
        onClick={onReset}
      >
        {isMobile ? (
          <>
            <RotateCcw className="h-3.5 w-3.5 mr-1" />
            Reset
          </>
        ) : (
          "Reset progress"
        )}
      </Button>
    </div>
  );
}
