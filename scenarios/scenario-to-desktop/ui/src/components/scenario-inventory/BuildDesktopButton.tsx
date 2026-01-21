import { useMemo, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "../ui/button";
import { Loader2, Hammer, CheckCircle, XCircle, AlertCircle, Check, HelpCircle } from "lucide-react";
import { PlatformChip } from "./PlatformChip";
import { WineInstallDialog } from "../wine";
import { PipelineErrorDisplay } from "../pipeline";
import type { PipelineConfig, BuildStageDetails, BuildPlatformResult } from "../../lib/api";
import { usePipelineMutation, usePipelineStatus, usePlatformSelection, useWineCheck } from "../../hooks";
import { getPlatformIcon, getPlatformName } from "../../domain/download";

interface BuildDesktopButtonProps {
  scenarioName: string;
}

const PLATFORM_OPTIONS = [
  { id: "win", label: "Windows (.msi installer)", helper: "Most laptops and desktops" },
  { id: "mac", label: "macOS (.pkg installer)", helper: "MacBook + iMac" },
  { id: "linux", label: "Linux (.AppImage/.deb)", helper: "Ubuntu, Fedora, etc." },
];

export function BuildDesktopButton({ scenarioName }: BuildDesktopButtonProps) {
  const queryClient = useQueryClient();

  // Platform selection with localStorage persistence
  const storageKey = useMemo(() => `desktop-platforms-${scenarioName}`, [scenarioName]);
  const { selectedPlatforms, togglePlatform } = usePlatformSelection({ storageKey });

  // Wine check for Windows builds on Linux
  const {
    wineCheck,
    showWineDialog,
    setShowWineDialog,
    pendingPlatforms,
    setPendingPlatforms,
    needsWineForPlatforms,
    handleWineInstallComplete: baseWineComplete,
  } = useWineCheck();

  // Pipeline mutation and status
  const {
    state: { buildId, error: mutationError },
    mutation,
    runPipelineWithConfig,
    reset,
  } = usePipelineMutation({
    invalidateOnSuccess: ["scenarios-desktop-status"],
  });

  const { pipelineStatus, isBuilding: statusIsBuilding } = usePipelineStatus({
    buildId,
    verbose: true,
    queryKeyPrefix: "build-status",
  });

  // Extract build details from pipeline status
  const buildDetails = pipelineStatus?.stages?.build?.details as BuildStageDetails | undefined;
  const buildStatus = pipelineStatus
    ? {
        status: pipelineStatus.status === "completed" ? "ready" : pipelineStatus.status === "failed" ? "failed" : "building",
        platform_results: buildDetails?.platform_results as Record<string, BuildPlatformResult> | undefined,
        requested_platforms: buildDetails?.requested_platforms,
      }
    : null;

  const isBuilding = mutation.isPending || statusIsBuilding;

  // Refresh scenarios list when build completes
  if ((buildStatus?.status === "ready" || buildStatus?.status === "partial") && buildId) {
    queryClient.invalidateQueries({ queryKey: ["scenarios-desktop-status"] });
  }

  // Handle build initiation
  const handleBuild = useCallback(
    (platforms?: string[]) => {
      const targets = platforms?.length ? platforms : selectedPlatforms;
      if (!targets.length) return;

      // Check if Wine is needed
      if (needsWineForPlatforms(targets)) {
        setPendingPlatforms(targets);
        setShowWineDialog(true);
        return;
      }

      const config: PipelineConfig = {
        scenario_name: scenarioName,
        platforms: targets,
      };
      runPipelineWithConfig(config);
    },
    [selectedPlatforms, needsWineForPlatforms, setPendingPlatforms, setShowWineDialog, scenarioName, runPipelineWithConfig]
  );

  // Handle Wine installation complete
  const handleWineInstallComplete = useCallback(() => {
    baseWineComplete();
    if (pendingPlatforms.length > 0) {
      const config: PipelineConfig = {
        scenario_name: scenarioName,
        platforms: pendingPlatforms,
      };
      runPipelineWithConfig(config);
      setPendingPlatforms([]);
    }
  }, [baseWineComplete, pendingPlatforms, scenarioName, runPipelineWithConfig, setPendingPlatforms]);

  // Show platform chips when build has results
  if (buildStatus?.platform_results) {
    const platforms = buildStatus.requested_platforms || ["win", "mac", "linux"];

    return (
      <div className="flex flex-col gap-3 w-full">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
          {platforms.map((platform) => (
            <PlatformChip
              key={platform}
              platform={platform}
              result={buildStatus.platform_results?.[platform]}
              scenarioName={scenarioName}
            />
          ))}
        </div>

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {buildStatus.status === "ready" && (
              <div className="flex items-center gap-1 text-green-400">
                <CheckCircle className="h-4 w-4" />
                <span className="text-sm">All platforms built successfully</span>
              </div>
            )}
            {buildStatus.status === "partial" && (
              <div className="flex items-center gap-1 text-yellow-400">
                <AlertCircle className="h-4 w-4" />
                <span className="text-sm">Some platforms built successfully</span>
              </div>
            )}
            {buildStatus.status === "failed" && (
              <div className="flex items-center gap-1 text-red-400">
                <XCircle className="h-4 w-4" />
                <span className="text-sm">Build failed</span>
              </div>
            )}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              reset();
              handleBuild(selectedPlatforms);
            }}
          >
            Rebuild Selected
          </Button>
        </div>
      </div>
    );
  }

  // Show mutation error
  if (mutationError) {
    return (
      <PipelineErrorDisplay
        title="Failed to start build"
        errorMessage={mutationError}
        onRetry={() => {
          reset();
          handleBuild();
        }}
      />
    );
  }

  if (isBuilding) {
    if (buildStatus?.platform_results) {
      const platforms = buildStatus.requested_platforms || ["win", "mac", "linux"];
      return (
        <div className="flex flex-col gap-3 w-full">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            {platforms.map((platform) => (
              <PlatformChip
                key={platform}
                platform={platform}
                result={buildStatus.platform_results?.[platform]}
                scenarioName={scenarioName}
              />
            ))}
          </div>
          <div className="flex items-center gap-2 text-blue-400">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span className="text-sm">Building platforms...</span>
          </div>
        </div>
      );
    }

    return (
      <div className="flex items-center gap-2 text-blue-400">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span className="text-sm">Starting build...</span>
      </div>
    );
  }

  const selectionSummary = (() => {
    if (selectedPlatforms.length === PLATFORM_OPTIONS.length)
      return "Building installers for every platform.";
    if (selectedPlatforms.length === 0) return "Select at least one platform to get started.";
    return `Building ${selectedPlatforms.map((p) => getPlatformName(p)).join(" + ")}.`;
  })();

  return (
    <>
      {showWineDialog && (
        <WineInstallDialog
          onClose={() => {
            setShowWineDialog(false);
            setPendingPlatforms([]);
          }}
          onInstallComplete={handleWineInstallComplete}
        />
      )}
      <div className="space-y-4">
        <div className="space-y-2">
          <p className="text-xs font-semibold uppercase tracking-wide text-slate-400">
            Which installers do you need?
          </p>
          <p className="text-xs text-slate-400">{selectionSummary}</p>
          <div className="grid gap-2 md:grid-cols-3">
            {PLATFORM_OPTIONS.map(({ id, label, helper }) => {
              const active = selectedPlatforms.includes(id);
              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => togglePlatform(id)}
                  className={`flex flex-col gap-1 rounded-xl border p-3 text-left transition focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                    active
                      ? "border-blue-400 bg-blue-950/40 shadow-inner"
                      : "border-slate-700 bg-slate-900/40"
                  }`}
                  aria-pressed={active}
                >
                  <div className="flex items-center justify-between text-sm font-semibold text-slate-100">
                    <div className="flex items-center gap-2">
                      <span>{getPlatformIcon(id)}</span>
                      {label}
                    </div>
                    {active && <Check className="h-4 w-4 text-green-400" />}
                  </div>
                  <p className="text-[11px] text-slate-400">{helper}</p>
                </button>
              );
            })}
          </div>
          <div className="flex items-center gap-1 text-[11px] text-slate-500">
            <HelpCircle className="h-3 w-3" />
            Need help choosing?{" "}
            <a
              href="https://github.com/vrooli/vrooli/blob/main/docs/deployment/tiers/tier-2-desktop.md"
              target="_blank"
              rel="noreferrer"
              className="text-blue-300 underline"
            >
              Read the desktop tier guide
            </a>
          </div>
        </div>
        {selectedPlatforms.length === 0 && (
          <p className="text-xs text-red-300">Select at least one platform to build.</p>
        )}
        <Button
          variant="default"
          size="sm"
          className="gap-2"
          onClick={() => handleBuild(selectedPlatforms)}
          disabled={selectedPlatforms.length === 0}
        >
          <Hammer className="h-4 w-4" />
          Build selected installers
        </Button>
      </div>
    </>
  );
}
