/**
 * Smoke Test section - displays smoke test stage status and results from the pipeline store.
 * Self-manages the smoke test action via the pipeline store's runStage("smoketest").
 *
 * Status detection distinguishes between:
 * - Pipeline completed at a checkpoint (stopped_after_stage set) → show "Run Smoke Test" button
 * - Pipeline genuinely failed/cancelled before smoketest → show warning + "New Pipeline"
 */

import { forwardRef, useMemo, useCallback, useState } from "react";
import { TestTube, CheckCircle2, XCircle, Monitor, FileText, Video, Loader2, Square, AlertCircle, RotateCcw } from "lucide-react";
import {
  SectionCard,
  STATUS_CONFIG,
  StageAbout,
  StageStatusOverview,
  StageDetailCard,
  StagePlaceholder,
  StageError,
} from "../shared";
import { buildUrl } from "../../../lib/api";
import {
  usePipelineStore,
  selectStageStatus,
  selectErrorInfo,
  selectIsRunning,
  selectCurrentStage,
  selectProgress,
  selectIsBusy,
  selectIsSubmitting,
} from "../../../store";
import { Button } from "../../ui/button";
import { formatStageName } from "../../../lib/status-display";

interface SmokeTestSectionProps {
  scenarioName: string;
}

export const SmokeTestSection = forwardRef<HTMLDivElement, SmokeTestSectionProps>(
  ({ scenarioName }, ref) => {
    const smokeTestResult = usePipelineStore((s) => s.smokeTestResult);
    const buildResult = usePipelineStore((s) => s.buildResult);
    const stageStatus = usePipelineStore(selectStageStatus("smoketest"));
    const pipelineOverallStatus = usePipelineStore((s) => s.pipelineStatus?.status);
    const stoppedAfterStage = usePipelineStore((s) => s.pipelineStatus?.stopped_after_stage);
    const stageLogs = usePipelineStore((s) => s.stageLogs["smoketest"]);
    const errorInfo = usePipelineStore(selectErrorInfo);
    const clearError = usePipelineStore((s) => s.clearError);
    const resetForRetry = usePipelineStore((s) => s.resetForRetry);
    const runStage = usePipelineStore((s) => s.runStage);
    const cancelPipeline = usePipelineStore((s) => s.cancelPipeline);
    const createNewPipeline = usePipelineStore((s) => s.createNewPipelineForScenario);
    const isRunning = usePipelineStore(selectIsRunning);
    const currentStage = usePipelineStore(selectCurrentStage);
    const progress = usePipelineStore(selectProgress);
    const isBusy = usePipelineStore(selectIsBusy);
    const isSubmitting = usePipelineStore(selectIsSubmitting);

    // Local error state for runStage("smoketest") call failures
    const [mutationError, setMutationError] = useState<string | null>(null);
    const [isCreatingPipeline, setIsCreatingPipeline] = useState(false);

    const hasResult = Boolean(smokeTestResult);
    const hasBuildArtifacts = Object.keys(buildResult?.artifacts ?? {}).length > 0;
    const { status: testStatus, platform, artifact_path: artifactPath, error, logs = [], telemetry_uploaded: telemetryUploaded } = smokeTestResult ?? {};
    const testPassed = testStatus === "completed" || testStatus === "passed";
    const progressPercent = Math.round(progress * 100);

    // Distinguish between normal checkpoint stop and actual failure/cancellation.
    // A pipeline that completed at a checkpoint (e.g. stopped_after_stage: "build") is
    // the normal stage-by-stage flow — the user just needs to click "Run Smoke Test".
    // A pipeline that failed or was cancelled is an actual problem.
    const pipelineCompletedAtCheckpoint = pipelineOverallStatus === "completed"
      && Boolean(stoppedAfterStage)
      && !hasResult
      && stageStatus === "pending";
    const pipelineFailedBeforeSmoketest = (pipelineOverallStatus === "failed" || pipelineOverallStatus === "cancelled")
      && !hasResult
      && stageStatus === "pending"
      && hasBuildArtifacts;

    // Can test when build artifacts exist, no result yet, and stage hasn't completed or failed
    const canTest = hasBuildArtifacts && !hasResult && stageStatus !== "completed";
    // Show the action area (button, progress, or starting state)
    const showTestAction = canTest && stageStatus !== "failed" && !pipelineFailedBeforeSmoketest;

    const statusDisplay = useMemo(() => {
      if (stageStatus === "completed") return { ...STATUS_CONFIG.completed, label: testPassed ? "Passed" : "Completed" };
      if (isRunning) return { ...STATUS_CONFIG.running, label: "Testing" };
      if (pipelineFailedBeforeSmoketest) return { ...STATUS_CONFIG.failed, label: "Skipped" };
      return STATUS_CONFIG[stageStatus as keyof typeof STATUS_CONFIG] ?? STATUS_CONFIG.pending;
    }, [stageStatus, testPassed, isRunning, pipelineFailedBeforeSmoketest]);

    const getDescription = () => {
      if (isRunning) return "Verifying built artifacts launch correctly...";
      if (hasResult) return `Tested on ${platform ?? "unknown platform"}`;
      if (pipelineFailedBeforeSmoketest) return "Pipeline ended before smoke testing could run";
      return hasBuildArtifacts ? "Ready to test" : "Waiting for build artifacts";
    };

    const handleRunSmokeTest = useCallback(async () => {
      setMutationError(null);
      try {
        await runStage("smoketest");
      } catch (err) {
        setMutationError(err instanceof Error ? err.message : "Failed to start smoke test");
      }
    }, [runStage]);

    const handleCancel = useCallback(() => {
      void cancelPipeline();
    }, [cancelPipeline]);

    const handleRetry = useCallback(() => {
      setMutationError(null);
      resetForRetry();
    }, [resetForRetry]);

    const handleNewPipeline = useCallback(async () => {
      setIsCreatingPipeline(true);
      try {
        await createNewPipeline();
      } finally {
        setIsCreatingPipeline(false);
      }
    }, [createNewPipeline]);

    // Merge stage-level logs with result-level logs for display during running state
    const displayLogs = isRunning && !hasResult ? (stageLogs ?? []) : logs;

    return (
      <SectionCard
        ref={ref}
        sectionId="smoketest"
        title="Smoke Test"
        subtitle="Validate built artifacts"
        variant="pipeline"
        collapsible={true}
        contentClassName="space-y-4"
      >
        <StageAbout title="About smoke testing">
          <p>The smoke test stage verifies that built installers launch correctly on each platform.</p>
        </StageAbout>

        <StageStatusOverview icon={TestTube} title="Smoke Test Status" description={getDescription()} statusDisplay={statusDisplay} />

        {/* Smoke test action area: button, progress, or starting state */}
        {showTestAction && (
          <div className="space-y-3">
            {isRunning ? (
              <>
                {/* Progress bar when pipeline is running */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-blue-400 flex items-center gap-1.5">
                      <Loader2 className="h-3 w-3 animate-spin" />
                      {currentStage ? `Running ${formatStageName(currentStage)} stage...` : "Starting pipeline..."}
                    </span>
                    <span className="text-slate-400">{progressPercent}%</span>
                  </div>
                  <div className="h-2 w-full rounded-full bg-slate-800 overflow-hidden">
                    <div
                      className="h-full bg-blue-500 transition-all duration-500 ease-out rounded-full"
                      style={{ width: `${Math.max(progressPercent, 2)}%` }}
                    />
                  </div>
                </div>

                {/* Live logs during running */}
                {displayLogs.length > 0 && (
                  <StageDetailCard icon={FileText} label={`Live Logs (${displayLogs.length} lines)`}>
                    <div className="max-h-32 overflow-y-auto">
                      <pre className="text-xs text-slate-400 font-mono whitespace-pre-wrap">{displayLogs.slice(-10).join("\n")}</pre>
                    </div>
                  </StageDetailCard>
                )}

                {/* Cancel button */}
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleCancel}
                  className="w-full border-red-800/60 text-red-300 hover:bg-red-950/30 hover:text-red-200"
                >
                  <Square className="mr-2 h-3.5 w-3.5" />
                  Cancel Smoke Test
                </Button>
              </>
            ) : (
              <Button
                onClick={handleRunSmokeTest}
                className="w-full"
                disabled={isBusy}
              >
                {isSubmitting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Starting...
                  </>
                ) : (
                  "Run Smoke Test"
                )}
              </Button>
            )}

            {/* Inline mutation error */}
            {mutationError && !isRunning && (
              <div className="flex items-start gap-2 rounded-lg border border-red-800/60 bg-red-950/30 p-3 text-sm text-red-300">
                <AlertCircle className="h-4 w-4 mt-0.5 shrink-0 text-red-400" />
                <div>
                  <strong>Error:</strong> {mutationError}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Pipeline failed/cancelled before smoketest — show warning and New Pipeline button */}
        {pipelineFailedBeforeSmoketest && (
          <div className="space-y-3">
            <div className="flex items-center justify-between gap-3 rounded-lg border border-amber-900/50 bg-amber-950/20 p-4">
              <div>
                <p className="text-sm font-medium text-amber-300">Build artifacts available</p>
                <p className="text-xs text-amber-400/70">
                  The previous pipeline {pipelineOverallStatus === "cancelled" ? "was cancelled" : "failed"} before
                  smoke testing could run. Start a new pipeline to test the built artifacts.
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={handleNewPipeline}
                disabled={isCreatingPipeline}
                className="shrink-0 border-emerald-800/50 text-emerald-400 hover:bg-emerald-900/40 hover:text-emerald-300"
              >
                <RotateCcw className="mr-1.5 h-3.5 w-3.5" />
                {isCreatingPipeline ? "Creating..." : "New Pipeline"}
              </Button>
            </div>
          </div>
        )}

        {/* Completed results */}
        {hasResult && (
          <div className="space-y-3">
            <div className="grid gap-2 sm:grid-cols-2">
              {platform && (
                <div className="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-950/50 p-3">
                  <Monitor className="h-4 w-4 text-slate-400" />
                  <div>
                    <p className="text-xs text-slate-500">Platform</p>
                    <p className="text-sm text-slate-300">{platform}</p>
                  </div>
                </div>
              )}
              {telemetryUploaded !== undefined && (
                <div className="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-950/50 p-3">
                  {telemetryUploaded ? <CheckCircle2 className="h-4 w-4 text-green-400" /> : <XCircle className="h-4 w-4 text-slate-400" />}
                  <div>
                    <p className="text-xs text-slate-500">Telemetry</p>
                    <p className="text-sm text-slate-300">{telemetryUploaded ? "Uploaded" : "Not uploaded"}</p>
                  </div>
                </div>
              )}
            </div>

            {artifactPath && (
              <StageDetailCard icon={FileText} label="Tested Artifact">
                <code className="text-xs text-slate-300 font-mono break-all">{artifactPath}</code>
              </StageDetailCard>
            )}

            {logs.length > 0 && (
              <StageDetailCard icon={FileText} label={`Test Logs (${logs.length} lines)`}>
                <div className="max-h-32 overflow-y-auto">
                  <pre className="text-xs text-slate-400 font-mono whitespace-pre-wrap">{logs.slice(-10).join("\n")}</pre>
                </div>
              </StageDetailCard>
            )}

            {smokeTestResult?.screen_recording?.recorded && smokeTestResult?.smoke_test_id && (
              <StageDetailCard icon={Video} label="Screen Recording">
                <div className="space-y-2">
                  <video
                    controls
                    className="w-full rounded border border-slate-700"
                    src={buildUrl(`/smoketest/${encodeURIComponent(smokeTestResult.smoke_test_id)}/video`)}
                  >
                    Your browser does not support the video tag.
                  </video>
                  <div className="flex gap-4 text-xs text-slate-500">
                    {smokeTestResult.screen_recording.duration_ms != null && (
                      <span>Duration: {(smokeTestResult.screen_recording.duration_ms / 1000).toFixed(1)}s</span>
                    )}
                    {smokeTestResult.screen_recording.file_size_bytes != null && (
                      <span>Size: {(smokeTestResult.screen_recording.file_size_bytes / 1024 / 1024).toFixed(1)} MB</span>
                    )}
                  </div>
                </div>
              </StageDetailCard>
            )}

            {smokeTestResult?.screen_recording?.error && (
              <StageDetailCard icon={Video} label="Screen Recording">
                <p className="text-xs text-amber-400">Recording failed: {smokeTestResult.screen_recording.error}</p>
              </StageDetailCard>
            )}

            {error && <StageError stageName="Smoke test"><strong>Error:</strong> {error}</StageError>}
          </div>
        )}

        {/* Placeholder when not ready */}
        {!hasResult && !pipelineFailedBeforeSmoketest && !pipelineCompletedAtCheckpoint && !hasBuildArtifacts && stageStatus === "pending" && (
          <StagePlaceholder
            scenarioName={scenarioName}
            withScenarioText="Smoke testing will be available after building installers."
            withoutScenarioText="Select a scenario to enable smoke testing."
          />
        )}

        {/* Stage-level failure (from pipeline execution) */}
        {stageStatus === "failed" && (
          <StageError
            stageName="Smoke test"
            errorInfo={errorInfo}
            onRetry={handleRetry}
            onDismiss={clearError}
          />
        )}
      </SectionCard>
    );
  }
);

SmokeTestSection.displayName = "SmokeTestSection";
