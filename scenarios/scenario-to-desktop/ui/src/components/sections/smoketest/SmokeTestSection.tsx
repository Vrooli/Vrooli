/**
 * Smoke Test section - displays smoke test stage status and results from the pipeline store.
 * Self-manages the smoke test action via the pipeline store's runStage("smoketest").
 *
 * Status detection distinguishes between:
 * - Pipeline completed at a checkpoint (stopped_after_stage set) → show "Run Smoke Test" button
 * - Pipeline genuinely failed/cancelled before smoketest → show warning + "New Pipeline"
 */

import { forwardRef, useMemo, useCallback, useRef, useState } from "react";
import {
  TestTube,
  CheckCircle2,
  XCircle,
  Monitor,
  FileText,
  Video,
  Loader2,
  Square,
  AlertCircle,
  RotateCcw,
  ScreenShare,
} from "lucide-react";
import {
  SectionCard,
  STATUS_CONFIG,
  StageAbout,
  StageStatusOverview,
  StageDetailCard,
  StagePlaceholder,
  StageError,
} from "../shared";
import { buildUrl, type SmokeTestStageDetails } from "../../../lib/api";
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
import { selectors } from "../../../consts/selectors";
import { useLiveDesktopStore } from "../../../store/liveDesktopStore";
import {
  Platform,
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import { SmokeTestStatus } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/operation_results_pb";

function platformLabel(platform: Platform): string {
  switch (platform) {
    case Platform.WIN:
      return "Windows";
    case Platform.MAC:
      return "macOS";
    case Platform.LINUX:
      return "Linux";
    default:
      return "unknown platform";
  }
}

/**
 * Diagnose screen recording errors and provide actionable troubleshooting.
 * Parses the raw error string from the backend to identify the likely cause
 * and suggest specific fixes.
 */
function ScreenRecordingError({ error }: { error: string }) {
  const lowerErr = error.toLowerCase();

  // Identify the likely cause from error patterns
  const diagnostics: { cause: string; suggestions: string[] } = (() => {
    if (lowerErr.includes("xvfb") && lowerErr.includes("not become ready")) {
      return {
        cause: "Virtual display (Xvfb) failed to start",
        suggestions: [
          "Install Xvfb: sudo apt-get install xvfb",
          "Check if display :99 is already in use: ls /tmp/.X99-lock",
          "Install xdpyinfo: sudo apt-get install x11-utils",
        ],
      };
    }
    if (
      lowerErr.includes("xvfb") ||
      lowerErr.includes("failed to start xvfb")
    ) {
      return {
        cause: "Xvfb (X Virtual Framebuffer) is not installed or cannot start",
        suggestions: [
          "Install Xvfb: sudo apt-get install xvfb",
          "Check system resources (memory, /tmp space)",
        ],
      };
    }
    if (
      lowerErr.includes("resource-ffmpeg") &&
      (lowerErr.includes("not found") || lowerErr.includes("no such file"))
    ) {
      return {
        cause: "The resource-ffmpeg CLI is not installed or not in PATH",
        suggestions: [
          "Install the FFmpeg resource: vrooli resource install ffmpeg",
          "Verify it's in PATH: which resource-ffmpeg",
        ],
      };
    }
    if (lowerErr.includes("ffmpeg") && lowerErr.includes("not found")) {
      return {
        cause: "FFmpeg is not installed",
        suggestions: [
          "Install FFmpeg: sudo apt-get install ffmpeg",
          "Install the FFmpeg resource: vrooli resource install ffmpeg",
        ],
      };
    }
    if (
      lowerErr.includes("display") &&
      (lowerErr.includes("cannot open") || lowerErr.includes("could not"))
    ) {
      return {
        cause: "No X display available for screen capture",
        suggestions: [
          "Ensure Xvfb is installed: sudo apt-get install xvfb",
          "Check DISPLAY environment variable",
        ],
      };
    }
    if (lowerErr.includes("permission denied")) {
      return {
        cause: "Insufficient permissions for screen capture",
        suggestions: [
          "Check file permissions on the output directory",
          "Ensure the process has access to the X display",
        ],
      };
    }
    if (lowerErr.includes("timeout") || lowerErr.includes("timed out")) {
      return {
        cause: "Screen capture timed out",
        suggestions: [
          "The capture process took too long to start",
          "Check system resources (CPU, memory)",
        ],
      };
    }
    // Generic fallback — show the raw error with general guidance
    return {
      cause: "Screen capture failed to start",
      suggestions: [
        "Ensure Xvfb is installed: sudo apt-get install xvfb",
        "Ensure the FFmpeg resource is available: which resource-ffmpeg",
        "Check the API logs for more details",
      ],
    };
  })();

  return (
    <div className="rounded-lg border border-amber-900/50 bg-amber-950/20 p-3 space-y-2">
      <div className="flex items-start gap-2">
        <Video className="h-4 w-4 mt-0.5 shrink-0 text-amber-400" />
        <div className="space-y-1.5 min-w-0">
          <p className="text-sm font-medium text-amber-300">
            Screen Recording Failed
          </p>
          <p className="text-xs text-amber-400/80">{diagnostics.cause}</p>
          <div className="rounded bg-slate-950/60 p-2">
            <pre className="text-xs text-slate-400 font-mono whitespace-pre-wrap break-all">
              {error}
            </pre>
          </div>
          <div>
            <p className="text-xs text-slate-400 mb-1">Troubleshooting:</p>
            <ul className="text-xs text-slate-400 space-y-0.5">
              {diagnostics.suggestions.map((s, i) => (
                <li key={i} className="flex items-start gap-1.5">
                  <span className="text-slate-500 shrink-0">-</span>
                  <span className="font-mono">{s}</span>
                </li>
              ))}
            </ul>
          </div>
          <p className="text-xs text-slate-500 italic">
            Note: Screen recording is optional — the smoke test result is not
            affected.
          </p>
        </div>
      </div>
    </div>
  );
}

type EvidenceReviewData = NonNullable<SmokeTestStageDetails["evidenceReview"]>;

function EvidenceReviewPanel({
  review,
  videoRef,
}: {
  review: EvidenceReviewData;
  videoRef: { current: HTMLVideoElement | null };
}) {
  const [selectedChapter, setSelectedChapter] = useState<string | null>(null);
  const selected = review.chapters.find((chapter) => chapter.id === selectedChapter);
  const isPass = review.disposition === "pass";
  const statusLabel = review.disposition.replace(/_/g, " ");

  const seekToChapter = (chapter: (typeof review.chapters)[number]) => {
    setSelectedChapter(chapter.id);
    const offset = chapter.videoStartOffsetMs;
    if (offset != null && videoRef.current) {
      videoRef.current.currentTime = Number(offset) / 1000;
      void videoRef.current.play().catch(() => undefined);
    }
  };

  return (
    <StageDetailCard icon={FileText} label="Evidence Review">
      <div className="space-y-3" data-testid="evidence-review">
        <div className="flex flex-wrap items-center gap-2 text-xs">
          <span className={isPass ? "text-emerald-300" : "text-amber-300"}>
            Verdict: {statusLabel}
          </span>
          <span className="text-slate-500">Capability: {review.capability}</span>
          {review.profile && <span className="text-slate-500">Profile: {review.profile}</span>}
          {review.providerTier && <span className="text-slate-500">Provider: {review.providerTier}</span>}
          {review.safeRouteClass && <span className="text-slate-500">Route: {review.safeRouteClass}</span>}
        </div>
        {!isPass && (
          <div className="rounded border border-amber-900/60 bg-amber-950/20 p-2 text-xs text-amber-200">
            <p>{review.reason || "The selected claim was not proven."}</p>
            <p className="mt-1 text-amber-300/80">Next action: inspect the failed chapter and raw journey capture.</p>
          </div>
        )}
        <ol className="space-y-1" aria-label="Evidence chapters">
          {review.chapters.map((chapter, index) => (
            <li key={chapter.id}>
              <button
                type="button"
                className={`w-full rounded border p-2 text-left text-xs transition-colors ${selectedChapter === chapter.id ? "border-blue-500 bg-blue-950/30" : "border-slate-800 hover:border-slate-700"}`}
                onClick={() => { seekToChapter(chapter); }}
                aria-label={`Review chapter ${String(index + 1)}: ${chapter.purpose}`}
              >
                <span className="flex items-center justify-between gap-2">
                  <span className="text-slate-200">{String(index + 1)}. {chapter.purpose}</span>
                  <span className={chapter.disposition === "passed" ? "text-emerald-300" : "text-amber-300"}>{chapter.disposition}</span>
                </span>
                <span className="mt-1 block text-slate-500">
                  {chapter.expected ? `Expected: ${chapter.expected}` : "Action"}
                  {chapter.observed ? ` · Observed: ${chapter.observed}` : ""}
                  {chapter.videoStartOffsetMs == null ? " · No video offset recorded" : ` · Video ${((Number(chapter.videoStartOffsetMs) || 0) / 1000).toFixed(1)}s`}
                </span>
              </button>
            </li>
          ))}
        </ol>
        {selected?.error && <pre className="max-h-24 overflow-auto whitespace-pre-wrap break-words rounded bg-slate-950 p-2 text-xs text-red-300">{selected.error}</pre>}
        <p className="text-[11px] text-slate-600">Timeline events: {String(review.eventCount)}. Raw captures remain available from the captures drawer.</p>
      </div>
    </StageDetailCard>
  );
}

interface SmokeTestSectionProps {
  scenarioName: string;
}

export const SmokeTestSection = forwardRef<
  HTMLDivElement,
  SmokeTestSectionProps
>(({ scenarioName }, ref) => {
  const smokeTestResult = usePipelineStore((s) => s.smokeTestResult);
  const buildResult = usePipelineStore((s) => s.buildResult);
  const stageStatus = usePipelineStore(selectStageStatus(StageName.SMOKE_TEST));
  const pipelineOverallStatus = usePipelineStore(
    (s) => s.pipelineStatus?.status,
  );
  const stoppedAfterStage = usePipelineStore(
    (s) => s.pipelineStatus?.stoppedAfterStage,
  );
  const stageLogs = usePipelineStore((s) => s.stageLogs["smoketest"]);
  const errorInfo = usePipelineStore(selectErrorInfo);
  const clearError = usePipelineStore((s) => s.clearError);
  const resetForRetry = usePipelineStore((s) => s.resetForRetry);
  const runStage = usePipelineStore((s) => s.runStage);
  const cancelPipeline = usePipelineStore((s) => s.cancelPipeline);
  const createNewPipeline = usePipelineStore(
    (s) => s.createNewPipelineForScenario,
  );
  const isRunning = usePipelineStore(selectIsRunning);
  const currentStage = usePipelineStore(selectCurrentStage);
  const progress = usePipelineStore(selectProgress);
  const isBusy = usePipelineStore(selectIsBusy);
  const isSubmitting = usePipelineStore(selectIsSubmitting);

  // Local error state for runStage("smoketest") call failures
  const [mutationError, setMutationError] = useState<string | null>(null);
  const [isCreatingPipeline, setIsCreatingPipeline] = useState(false);
  const evidenceVideoRef = useRef<HTMLVideoElement>(null);

  const hasResult = Boolean(smokeTestResult);
  const hasBuildArtifacts =
    Object.keys(buildResult?.artifacts ?? {}).length > 0;
  const testStatus = smokeTestResult?.status;
  const platform = smokeTestResult?.platform;
  const artifactPath = smokeTestResult?.artifactPath;
  const error = smokeTestResult?.error;
  const logs = smokeTestResult?.logs ?? [];
  const telemetryUploaded = smokeTestResult?.telemetryUploaded;
  const testPassed = testStatus === SmokeTestStatus.PASSED;
  const progressPercent = Math.round(progress * 100);

  // Distinguish between normal checkpoint stop and actual failure/cancellation.
  // A pipeline that completed at a checkpoint (e.g. stopped_after_stage: "build") is
  // the normal stage-by-stage flow — the user just needs to click "Run Smoke Test".
  // A pipeline that failed or was cancelled is an actual problem.
  const pipelineCompletedAtCheckpoint =
    pipelineOverallStatus === StageStatus.COMPLETED &&
    Boolean(stoppedAfterStage) &&
    !hasResult &&
    stageStatus === StageStatus.PENDING;
  const pipelineFailedBeforeSmoketest =
    (pipelineOverallStatus === StageStatus.FAILED ||
      pipelineOverallStatus === StageStatus.CANCELLED) &&
    !hasResult &&
    stageStatus === StageStatus.PENDING &&
    hasBuildArtifacts;

  // Can test when build artifacts exist (allow re-running after completion)
  const canTest = hasBuildArtifacts;
  // Show the action area (button, progress, or starting state)
  const showTestAction =
    canTest &&
    stageStatus !== StageStatus.FAILED &&
    !pipelineFailedBeforeSmoketest;

  const statusConfig = (status: StageStatus) => {
    const config = STATUS_CONFIG[status] ?? STATUS_CONFIG[StageStatus.PENDING];
    if (!config) throw new Error("Missing pending stage status configuration");
    return config;
  };

  const statusDisplay = useMemo(() => {
    if (stageStatus === StageStatus.COMPLETED)
      return {
        ...statusConfig(StageStatus.COMPLETED),
        label: testPassed ? "Passed" : "Completed",
      };
    if (isRunning)
      return { ...statusConfig(StageStatus.RUNNING), label: "Testing" };
    if (pipelineFailedBeforeSmoketest)
      return { ...statusConfig(StageStatus.FAILED), label: "Skipped" };
    return statusConfig(stageStatus);
  }, [stageStatus, testPassed, isRunning, pipelineFailedBeforeSmoketest]);

  const getDescription = () => {
    if (isRunning) return "Verifying built artifacts launch correctly...";
    if (hasResult)
      return `Tested on ${platformLabel(platform ?? Platform.UNSPECIFIED)}`;
    if (pipelineFailedBeforeSmoketest)
      return "Pipeline ended before smoke testing could run";
    return hasBuildArtifacts ? "Ready to test" : "Waiting for build artifacts";
  };

  const handleRunSmokeTest = useCallback(async () => {
    setMutationError(null);
    try {
      await runStage(StageName.SMOKE_TEST);
    } catch (err) {
      setMutationError(
        err instanceof Error ? err.message : "Failed to start smoke test",
      );
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
      data-testid={selectors.generator.smokeTestSection}
    >
      <StageAbout title="About smoke testing">
        <p>
          The smoke test stage verifies that built installers launch correctly
          on each platform.
        </p>
      </StageAbout>

      <StageStatusOverview
        icon={TestTube}
        title="Smoke Test Status"
        description={getDescription()}
        statusDisplay={statusDisplay}
      />

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
                    {currentStage
                      ? `Running ${formatStageName(currentStage)} stage...`
                      : "Starting pipeline..."}
                  </span>
                  <span className="text-slate-400">{progressPercent}%</span>
                </div>
                <div className="h-2 w-full rounded-full bg-slate-800 overflow-hidden">
                  <div
                    className="h-full bg-blue-500 transition-all duration-500 ease-out rounded-full"
                    style={{
                      width: `${String(Math.max(progressPercent, 2))}%`,
                    }}
                  />
                </div>
              </div>

              {/* Live logs during running */}
              {displayLogs.length > 0 && (
                <StageDetailCard
                  icon={FileText}
                  label={`Live Logs (${String(displayLogs.length)} lines)`}
                >
                  <div className="max-h-32 overflow-y-auto">
                    <pre className="text-xs text-slate-400 font-mono whitespace-pre-wrap">
                      {displayLogs.slice(-10).join("\n")}
                    </pre>
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
              onClick={() => {
                void handleRunSmokeTest();
              }}
              className="w-full"
              disabled={isBusy}
              data-testid={selectors.generator.smokeTestRun}
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Starting...
                </>
              ) : hasResult ? (
                "Re-run Smoke Test"
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
              <p className="text-sm font-medium text-amber-300">
                Build artifacts available
              </p>
              <p className="text-xs text-amber-400/70">
                The previous pipeline{" "}
                {pipelineOverallStatus === StageStatus.CANCELLED
                  ? "was cancelled"
                  : "failed"}{" "}
                before smoke testing could run. Start a new pipeline to test the
                built artifacts.
              </p>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                void handleNewPipeline();
              }}
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
            {platform !== undefined && platform !== Platform.UNSPECIFIED && (
              <div className="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-950/50 p-3">
                <Monitor className="h-4 w-4 text-slate-400" />
                <div>
                  <p className="text-xs text-slate-500">Platform</p>
                  <p className="text-sm text-slate-300">
                    {platformLabel(platform)}
                  </p>
                </div>
              </div>
            )}
            {telemetryUploaded !== undefined && (
              <div className="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-950/50 p-3">
                {telemetryUploaded ? (
                  <CheckCircle2 className="h-4 w-4 text-green-400" />
                ) : (
                  <XCircle className="h-4 w-4 text-slate-400" />
                )}
                <div>
                  <p className="text-xs text-slate-500">Telemetry</p>
                  <p className="text-sm text-slate-300">
                    {telemetryUploaded ? "Uploaded" : "Not uploaded"}
                  </p>
                </div>
              </div>
            )}
          </div>

          {artifactPath && (
            <StageDetailCard icon={FileText} label="Tested Artifact">
              <code className="text-xs text-slate-300 font-mono break-all">
                {artifactPath}
              </code>
            </StageDetailCard>
          )}

          {logs.length > 0 && (
            <StageDetailCard
              icon={FileText}
              label={`Test Logs (${String(logs.length)} lines)`}
            >
              <div className="max-h-32 overflow-y-auto">
                <pre className="text-xs text-slate-400 font-mono whitespace-pre-wrap">
                  {logs.slice(-10).join("\n")}
                </pre>
              </div>
            </StageDetailCard>
          )}

          {smokeTestResult?.screenRecording?.recorded &&
            smokeTestResult.screenRecording.captureId && (
              <StageDetailCard icon={Video} label="Screen Recording">
                <div className="space-y-2">
                  <video
                    ref={evidenceVideoRef}
                    controls
                    className="w-full rounded border border-slate-700"
                    src={buildUrl(
                      `/captures/${encodeURIComponent(scenarioName)}/${encodeURIComponent(smokeTestResult.screenRecording.captureId)}/file`,
                    )}
                  >
                    Your browser does not support the video tag.
                  </video>
                  <div className="flex gap-4 text-xs text-slate-500">
                    {smokeTestResult.screenRecording.durationMs != null && (
                      <span>
                        Duration:{" "}
                        {(
                          Number(smokeTestResult.screenRecording.durationMs) /
                          1000
                        ).toFixed(1)}
                        s
                      </span>
                    )}
                    {smokeTestResult.screenRecording.fileSizeBytes != null && (
                      <span>
                        Size:{" "}
                        {(
                          Number(
                            smokeTestResult.screenRecording.fileSizeBytes,
                          ) /
                          1024 /
                          1024
                        ).toFixed(1)}{" "}
                        MB
                      </span>
                    )}
                  </div>
                </div>
              </StageDetailCard>
            )}

          {smokeTestResult?.screenRecording?.error && (
            <ScreenRecordingError
              error={smokeTestResult.screenRecording.error}
            />
          )}

          {smokeTestResult?.evidenceReview && (
            <EvidenceReviewPanel
              review={smokeTestResult.evidenceReview}
              videoRef={evidenceVideoRef}
            />
          )}

          {error && (
            <StageError stageName="Smoke test">
              <strong>Error:</strong> {error}
            </StageError>
          )}
        </div>
      )}

      {/* Launch Interactive Desktop — available when build artifacts exist */}
      {hasBuildArtifacts && !isRunning && (
        <Button
          type="button"
          variant="outline"
          className="w-full border-blue-800/60 text-blue-300 hover:bg-blue-950/30 hover:text-blue-200"
          onClick={() => {
            useLiveDesktopStore.getState().open(scenarioName, artifactPath);
          }}
          data-testid={selectors.generator.liveDesktopLaunch}
        >
          <ScreenShare className="mr-2 h-4 w-4" />
          Launch Interactive Desktop
        </Button>
      )}

      {/* Placeholder when not ready */}
      {!hasResult &&
        !pipelineFailedBeforeSmoketest &&
        !pipelineCompletedAtCheckpoint &&
        !hasBuildArtifacts &&
        stageStatus === StageStatus.PENDING && (
          <StagePlaceholder
            scenarioName={scenarioName}
            withScenarioText="Smoke testing will be available after building installers."
            withoutScenarioText="Select a scenario to enable smoke testing."
          />
        )}

      {/* Stage-level failure (from pipeline execution) */}
      {stageStatus === StageStatus.FAILED && (
        <StageError
          stageName="Smoke test"
          errorInfo={errorInfo}
          onRetry={handleRetry}
          onDismiss={clearError}
        />
      )}
    </SectionCard>
  );
});

SmokeTestSection.displayName = "SmokeTestSection";
