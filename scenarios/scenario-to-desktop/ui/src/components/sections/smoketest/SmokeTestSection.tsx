/**
 * Smoke Test section - displays smoke test stage status and results from the pipeline store.
 *
 * State machine: The smoke test status depends on both its own stage status AND the overall
 * pipeline status. When the pipeline is in a terminal state (failed/cancelled) but the build
 * completed, this section must show actionable messaging rather than "waiting" messages.
 */

import { forwardRef, useMemo, useCallback, useState } from "react";
import { TestTube, CheckCircle2, XCircle, Monitor, FileText, Video, Loader2, StopCircle, RotateCcw } from "lucide-react";
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
import { usePipelineStore, selectStageStatus } from "../../../store";
import { isTerminalState } from "../../../services/pipeline.service";

interface SmokeTestSectionProps {
  scenarioName: string;
}

export const SmokeTestSection = forwardRef<HTMLDivElement, SmokeTestSectionProps>(
  ({ scenarioName }, ref) => {
    const smokeTestResult = usePipelineStore((s) => s.smokeTestResult);
    const buildResult = usePipelineStore((s) => s.buildResult);
    const stageStatus = usePipelineStore(selectStageStatus("smoketest"));
    const pipelineOverallStatus = usePipelineStore((s) => s.pipelineStatus?.status);
    const stageLogs = usePipelineStore((s) => s.stageLogs["smoketest"]);
    const cancelPipeline = usePipelineStore((s) => s.cancelPipeline);
    const createNewPipeline = usePipelineStore((s) => s.createNewPipelineForScenario);
    const [isCancelling, setIsCancelling] = useState(false);
    const [isCreatingPipeline, setIsCreatingPipeline] = useState(false);

    const handleCancel = useCallback(async () => {
      setIsCancelling(true);
      try {
        await cancelPipeline();
      } finally {
        setIsCancelling(false);
      }
    }, [cancelPipeline]);

    const handleNewPipeline = useCallback(async () => {
      setIsCreatingPipeline(true);
      try {
        await createNewPipeline();
      } finally {
        setIsCreatingPipeline(false);
      }
    }, [createNewPipeline]);

    const hasResult = Boolean(smokeTestResult);
    const hasBuildArtifacts = Object.keys(buildResult?.artifacts ?? {}).length > 0;
    const { status: testStatus, platform, artifact_path: artifactPath, error, logs = [], telemetry_uploaded: telemetryUploaded } = smokeTestResult ?? {};
    const testPassed = testStatus === "completed" || testStatus === "passed";
    const isRunning = stageStatus === "running";

    // Detect when the pipeline ended before smoketest could run.
    // This happens when a pipeline is interrupted (e.g., server restart) or fails at a later
    // stage while build artifacts still exist from a completed build stage.
    const pipelineTerminal = isTerminalState(pipelineOverallStatus);
    const pipelineStoppedBeforeSmoketest = pipelineTerminal && !hasResult && stageStatus === "pending" && hasBuildArtifacts;

    const statusDisplay = useMemo(() => {
      if (stageStatus === "completed") return { ...STATUS_CONFIG.completed, label: testPassed ? "Passed" : "Completed" };
      if (isRunning) return { ...STATUS_CONFIG.running, label: "Testing" };
      if (pipelineStoppedBeforeSmoketest) return { ...STATUS_CONFIG.failed, label: "Skipped" };
      return STATUS_CONFIG[stageStatus as keyof typeof STATUS_CONFIG] ?? STATUS_CONFIG.pending;
    }, [stageStatus, testPassed, isRunning, pipelineStoppedBeforeSmoketest]);

    const getDescription = () => {
      if (isRunning) return "Verifying built artifacts launch correctly...";
      if (hasResult) return `Tested on ${platform ?? "unknown platform"}`;
      if (pipelineStoppedBeforeSmoketest) return "Pipeline ended before smoke testing could run";
      return hasBuildArtifacts ? "Ready to test" : "Waiting for build artifacts";
    };

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

        {/* Pipeline stopped before smoketest - show explanation and retry button */}
        {pipelineStoppedBeforeSmoketest && (
          <div className="space-y-3">
            <div className="flex items-center justify-between gap-3 rounded-lg border border-amber-900/50 bg-amber-950/20 p-4">
              <div>
                <p className="text-sm font-medium text-amber-300">Build artifacts available</p>
                <p className="text-xs text-amber-400/70">
                  The previous pipeline {pipelineOverallStatus === "cancelled" ? "was cancelled" : "failed"} before
                  smoke testing could run. Start a new pipeline to test the built artifacts.
                </p>
              </div>
              <button
                type="button"
                onClick={handleNewPipeline}
                disabled={isCreatingPipeline}
                className="flex items-center gap-1.5 shrink-0 rounded-md border border-emerald-800/50 bg-emerald-950/30 px-3 py-1.5 text-xs font-medium text-emerald-400 hover:bg-emerald-900/40 hover:text-emerald-300 disabled:opacity-50 transition-colors"
                title="Create a new pipeline to run smoke tests"
              >
                <RotateCcw className="h-3.5 w-3.5" />
                {isCreatingPipeline ? "Creating..." : "New Pipeline"}
              </button>
            </div>
          </div>
        )}

        {/* Running state - show progress indicator, cancel button, and live logs */}
        {isRunning && !hasResult && (
          <div className="space-y-3">
            <div className="flex items-center justify-between gap-3 rounded-lg border border-blue-900/50 bg-blue-950/20 p-4">
              <div className="flex items-center gap-3">
                <Loader2 className="h-5 w-5 animate-spin text-blue-400 shrink-0" />
                <div>
                  <p className="text-sm font-medium text-blue-300">Smoke test in progress</p>
                  <p className="text-xs text-blue-400/70">
                    Launching the built installer and verifying it starts correctly.
                    This may take a few minutes.
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={handleCancel}
                disabled={isCancelling}
                className="flex items-center gap-1.5 shrink-0 rounded-md border border-red-800/50 bg-red-950/30 px-3 py-1.5 text-xs font-medium text-red-400 hover:bg-red-900/40 hover:text-red-300 disabled:opacity-50 transition-colors"
                title="Cancel the running pipeline"
              >
                <StopCircle className="h-3.5 w-3.5" />
                {isCancelling ? "Cancelling..." : "Cancel"}
              </button>
            </div>

            {displayLogs.length > 0 && (
              <StageDetailCard icon={FileText} label={`Live Logs (${displayLogs.length} lines)`}>
                <div className="max-h-32 overflow-y-auto">
                  <pre className="text-xs text-slate-400 font-mono whitespace-pre-wrap">{displayLogs.slice(-10).join("\n")}</pre>
                </div>
              </StageDetailCard>
            )}
          </div>
        )}

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

        {!hasResult && !pipelineStoppedBeforeSmoketest && !hasBuildArtifacts && stageStatus === "pending" && (
          <StagePlaceholder
            scenarioName={scenarioName}
            withScenarioText="Smoke testing will be available after building installers."
            withoutScenarioText="Select a scenario to enable smoke testing."
          />
        )}

        {stageStatus === "failed" && !hasResult && <StageError stageName="Smoke test" />}
      </SectionCard>
    );
  }
);

SmokeTestSection.displayName = "SmokeTestSection";
