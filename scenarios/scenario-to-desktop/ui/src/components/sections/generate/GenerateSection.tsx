/**
 * Generate section - displays generate stage status and results from the pipeline store.
 * Also contains the submit button for starting the generation pipeline.
 */

import { forwardRef, type FormEvent } from "react";
import { Wand2, FolderOpen } from "lucide-react";
import {
  SectionCard,
  getStatusDisplay,
  StageStatusOverview,
  StageAbout,
  StageDetailCard,
  StagePlaceholder,
  StageError,
} from "../shared";
import { usePipelineStore, selectStageStatus, selectErrorInfo } from "../../../store";
import { Button } from "../../ui/button";
import { ValidationErrors, type ValidationError } from "../../generator";

interface GenerateSectionProps {
  scenarioName: string;
  onRetry?: () => void;
  /** ID of the form to submit */
  formId?: string;
  /** Validation errors to display above the submit button */
  validationErrors?: ValidationError[];
  /** Callback when errors are dismissed */
  onDismissErrors?: () => void;
  /** Whether the generation is pending */
  isPending?: boolean;
  /** Whether there was an error with the mutation */
  isError?: boolean;
  /** Error message from the mutation */
  errorMessage?: string | null;
  /** Whether this is an update to an existing desktop app */
  isUpdateMode?: boolean;
  /** Direct submit handler when form is in a different section */
  onSubmit?: () => void;
}

export const GenerateSection = forwardRef<HTMLDivElement, GenerateSectionProps>(
  (
    {
      scenarioName,
      onRetry,
      formId,
      validationErrors = [],
      onDismissErrors,
      isPending = false,
      isError = false,
      errorMessage = null,
      isUpdateMode = false,
      onSubmit,
    },
    ref
  ) => {
    const generateResult = usePipelineStore((s) => s.generateResult);
    const stageStatus = usePipelineStore(selectStageStatus("generate"));
    const errorInfo = usePipelineStore(selectErrorInfo);
    const clearError = usePipelineStore((s) => s.clearError);
    const resetForRetry = usePipelineStore((s) => s.resetForRetry);

    const hasResult = Boolean(generateResult);
    const desktopPath = generateResult?.desktop_path;
    const buildId = generateResult?.build_id;
    const statusDisplay = getStatusDisplay(stageStatus, { completed: "Generated", running: "Generating" });

    // Handle form submission - either via form attribute or direct callback
    const handleSubmitClick = (e: FormEvent) => {
      e.preventDefault();
      if (onSubmit) {
        onSubmit();
      }
    };

    // Determine if the submit button should be shown
    const showSubmitButton = Boolean(formId) || Boolean(onSubmit);

    return (
      <SectionCard
        ref={ref}
        sectionId="generate"
        title="Generate"
        subtitle="Create desktop wrapper code"
        variant="pipeline"
        collapsible={true}
        contentClassName="space-y-4"
      >
        {/* Submit button at the top of the section */}
        {showSubmitButton && (
          <div className="space-y-3">
            {/* Validation errors - shown above submit button */}
            <ValidationErrors
              errors={validationErrors}
              onDismiss={onDismissErrors ?? (() => {})}
            />

            <Button
              type={formId ? "submit" : "button"}
              form={formId}
              onClick={onSubmit ? handleSubmitClick : undefined}
              className="w-full"
              disabled={isPending || validationErrors.length > 0}
            >
              {isPending
                ? "Generating..."
                : isUpdateMode
                  ? "Update Desktop Application"
                  : "Generate Desktop Application"}
            </Button>

            {isError && errorMessage && (
              <div className="rounded-lg bg-red-900/20 p-3 text-sm text-red-300">
                <strong>Error:</strong> {errorMessage}
              </div>
            )}
          </div>
        )}

        <StageAbout title="About generation">
          <p>
            The generate stage creates an Electron project scaffold stored in{" "}
            <code className="font-mono text-slate-200">platforms/electron</code>.
          </p>
        </StageAbout>

        <StageStatusOverview
          icon={Wand2}
          title="Generate Status"
          description={hasResult ? `Electron wrapper generated${buildId ? ` (Build: ${buildId.slice(0, 8)}...)` : ""}` : "Wrapper not yet generated"}
          statusDisplay={statusDisplay}
        />

        {hasResult && desktopPath && (
          <StageDetailCard icon={FolderOpen} label="Desktop Application Path">
            <code className="text-xs text-slate-300 font-mono break-all">{desktopPath}</code>
          </StageDetailCard>
        )}

        {!hasResult && stageStatus === "pending" && !showSubmitButton && (
          <StagePlaceholder
            scenarioName={scenarioName}
            withScenarioText="Complete configuration and click the Generate button above to start."
          />
        )}

        {stageStatus === "failed" && (
          <StageError
            stageName="Generate"
            errorInfo={errorInfo}
            onRetry={() => {
              resetForRetry();
              onRetry?.();
            }}
            onDismiss={clearError}
          />
        )}
      </SectionCard>
    );
  }
);

GenerateSection.displayName = "GenerateSection";
