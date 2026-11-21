/**
 * Build section - displays build stage status and results from the pipeline store.
 * Self-manages the build action via the pipeline store's runStage("build").
 */

import { forwardRef, useState, useCallback } from "react";
import {
  Hammer,
  Monitor,
  Apple,
  Terminal,
  FolderOpen,
  Loader2,
  Square,
  AlertCircle,
} from "lucide-react";
import {
  SectionCard,
  getStatusDisplay,
  StageAbout,
  StageStatusOverview,
  StageDetailCard,
  StagePlaceholder,
  StageError,
} from "../shared";
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
import {
  Platform,
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

function platformLabel(platform: Platform): string {
  switch (platform) {
    case Platform.WIN:
      return "win";
    case Platform.MAC:
      return "mac";
    case Platform.LINUX:
      return "linux";
    default:
      return "unknown";
  }
}

interface BuildSectionProps {
  scenarioName: string;
}

/** Platform icon mapping */
const PLATFORM_ICONS: Record<string, typeof Monitor> = {
  win: Monitor,
  windows: Monitor,
  mac: Apple,
  darwin: Apple,
  macos: Apple,
  linux: Terminal,
};

export const BuildSection = forwardRef<HTMLDivElement, BuildSectionProps>(
  ({ scenarioName }, ref) => {
    const buildResult = usePipelineStore((s) => s.buildResult);
    const generateResult = usePipelineStore((s) => s.generateResult);
    const stageStatus = usePipelineStore(selectStageStatus(StageName.BUILD));
    const errorInfo = usePipelineStore(selectErrorInfo);
    const clearError = usePipelineStore((s) => s.clearError);
    const resetForRetry = usePipelineStore((s) => s.resetForRetry);
    const runStage = usePipelineStore((s) => s.runStage);
    const cancelPipeline = usePipelineStore((s) => s.cancelPipeline);
    const isRunning = usePipelineStore(selectIsRunning);
    const currentStage = usePipelineStore(selectCurrentStage);
    const progress = usePipelineStore(selectProgress);
    const isBusy = usePipelineStore(selectIsBusy);
    const isSubmitting = usePipelineStore(selectIsSubmitting);

    // Local error state for runStage("build") call failures
    const [mutationError, setMutationError] = useState<string | null>(null);

    const hasResult = Boolean(buildResult);
    const artifacts = buildResult?.artifacts ?? {};
    const artifactEntries = Object.entries(artifacts);
    const outputPath = buildResult?.outputPath;
    const platforms = buildResult?.requestedPlatforms.map(platformLabel) ?? [];
    const canBuild = Boolean(generateResult);
    const progressPercent = Math.round(progress * 100);

    const statusDisplay = getStatusDisplay(stageStatus, {
      [StageStatus.COMPLETED]: "Built",
      [StageStatus.RUNNING]: "Building",
    });

    const handleBuild = useCallback(async () => {
      setMutationError(null);
      try {
        await runStage(StageName.BUILD);
      } catch (err) {
        setMutationError(
          err instanceof Error ? err.message : "Failed to start build",
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

    const getDescription = () => {
      if (hasResult) {
        return `${String(artifactEntries.length)} artifact${artifactEntries.length !== 1 ? "s" : ""} built for ${platforms.join(", ")}`;
      }
      if (isRunning && currentStage === StageName.BUILD) {
        return "Building installers...";
      }
      return canBuild ? "Ready to build" : "Waiting for generate stage";
    };

    // Show the build action area when generate is done (allow re-running after completion)
    const showBuildAction = canBuild && stageStatus !== StageStatus.FAILED;

    return (
      <SectionCard
        ref={ref}
        sectionId="build"
        title="Build"
        subtitle="Compile installers for target platforms"
        variant="pipeline"
        collapsible={true}
        contentClassName="space-y-4"
        data-testid={selectors.generator.buildSection}
      >
        <StageAbout title="About building">
          <p>
            The build stage packages the Electron wrapper into distributable
            installers for Windows, macOS, and Linux.
          </p>
        </StageAbout>

        <StageStatusOverview
          icon={Hammer}
          title="Build Status"
          description={getDescription()}
          statusDisplay={statusDisplay}
        />

        {/* Build action area: button, progress, or starting state */}
        {showBuildAction && (
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
                    <span className="text-slate-400">
                      {String(progressPercent)}%
                    </span>
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

                {/* Cancel button */}
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleCancel}
                  className="w-full border-red-800/60 text-red-300 hover:bg-red-950/30 hover:text-red-200"
                >
                  <Square className="mr-2 h-3.5 w-3.5" />
                  Cancel Build
                </Button>
              </>
            ) : (
              <Button
                onClick={() => {
                  void handleBuild();
                }}
                className="w-full"
                disabled={isBusy}
                data-testid={selectors.generator.buildStart}
              >
                {isSubmitting ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Starting...
                  </>
                ) : hasResult ? (
                  "Rebuild Installers"
                ) : (
                  "Build Installers"
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

        {/* Build details when available */}
        {hasResult && (
          <div className="space-y-3">
            {outputPath && (
              <StageDetailCard icon={FolderOpen} label="Output Path">
                <code className="text-xs text-slate-300 font-mono break-all">
                  {outputPath}
                </code>
              </StageDetailCard>
            )}

            {/* Build artifacts */}
            {artifactEntries.length > 0 && (
              <div className="rounded-lg border border-slate-800 bg-slate-950/50 p-3">
                <p className="text-xs text-slate-400 mb-2">Built Artifacts</p>
                <div className="space-y-2">
                  {artifactEntries.map(([platform, path]) => {
                    const PlatformIcon =
                      PLATFORM_ICONS[platform.toLowerCase()] ?? Monitor;
                    return (
                      <div
                        key={platform}
                        className="flex items-center justify-between rounded-md border border-slate-700 bg-slate-900/50 p-2"
                      >
                        <div className="flex items-center gap-2">
                          <PlatformIcon className="h-4 w-4 text-slate-400" />
                          <span className="text-sm text-slate-300">
                            {platform}
                          </span>
                        </div>
                        <code className="text-xs text-slate-400 font-mono truncate max-w-[200px]">
                          {path.split("/").pop()}
                        </code>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Placeholder when not ready */}
        {!canBuild && stageStatus === StageStatus.PENDING && (
          <StagePlaceholder
            scenarioName={scenarioName}
            withScenarioText="Generate the wrapper first using the Configuration section."
            withoutScenarioText="Select a scenario to unlock installer builds."
          />
        )}

        {/* Error state */}
        {stageStatus === StageStatus.FAILED && (
          <StageError
            stageName="Build"
            errorInfo={errorInfo}
            onRetry={handleRetry}
            onDismiss={clearError}
          />
        )}
      </SectionCard>
    );
  },
);

BuildSection.displayName = "BuildSection";
