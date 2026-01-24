import { forwardRef, useEffect, useImperativeHandle, useMemo, useRef, useState } from "react";
import { usePipelineStore, selectIsRunning } from "../../store";
import type { BundleStageDetails } from "../../lib/api";
import { Button } from "../ui/button";
import { Label } from "../ui/label";
import { Progress } from "../ui/progress";
import { Select } from "../ui/select";
import { AlertTriangle, CheckCircle, Loader2 } from "lucide-react";

export type DeploymentManagerBundleHelperHandle = {
  exportBundle: () => void;
};

/** Bundle result data for stage persistence */
export interface BundleResult {
  bundleDetails: BundleStageDetails | null;
  manifestPath: string | null;
  checksum?: string;
  generatedAt?: string;
}

type DeploymentManagerBundleHelperProps = {
  scenarioName: string;
  onBundleManifestChange: (value: string) => void;
  onBundleExported?: (manifestPath: string) => void;
  /** Called when bundle export completes successfully. Use to persist stage results. */
  onBundleComplete?: (result: BundleResult) => void;
  /** Initial bundle details for restoration from server persistence. */
  initialBundleDetails?: BundleStageDetails | null;
  initialManifestPath?: string | null;
};

export const DeploymentManagerBundleHelper = forwardRef<DeploymentManagerBundleHelperHandle, DeploymentManagerBundleHelperProps>(
  ({
    scenarioName,
    onBundleManifestChange,
    onBundleExported,
    onBundleComplete,
    initialBundleDetails,
    initialManifestPath,
  }, ref) => {
    const [tier, setTier] = useState("tier-2-desktop");

    // Pipeline store
    const {
      setScenario,
      runBundleStage,
      runStatus,
      error: pipelineError,
      bundleResult,
      stageLogs,
      pipelineId,
      clearError,
    } = usePipelineStore();
    const isRunning = usePipelineStore(selectIsRunning);

    // Track previous scenario to detect genuine changes
    const prevScenarioRef = useRef<string>(scenarioName);
    const hasNotifiedRef = useRef<boolean>(false);

    // Local state for display
    const [localBundleDetails, setLocalBundleDetails] = useState<BundleStageDetails | null>(
      initialBundleDetails ?? null
    );
    const [localManifestPath, setLocalManifestPath] = useState<string | null>(
      initialManifestPath ?? null
    );

    // Set scenario in store when it changes
    useEffect(() => {
      if (scenarioName) {
        setScenario(scenarioName);
      }
    }, [scenarioName, setScenario]);

    // Sync from initial props
    useEffect(() => {
      if (initialBundleDetails) {
        setLocalBundleDetails(initialBundleDetails);
      }
      if (initialManifestPath) {
        setLocalManifestPath(initialManifestPath);
      }
    }, [initialBundleDetails, initialManifestPath]);

    // Reset state when scenario genuinely changes
    useEffect(() => {
      const prevScenario = prevScenarioRef.current;
      prevScenarioRef.current = scenarioName;

      if (prevScenario !== scenarioName && prevScenario !== "") {
        setLocalBundleDetails(null);
        setLocalManifestPath(null);
        hasNotifiedRef.current = false;
        clearError();
      }
    }, [scenarioName, clearError]);

    // Update local state when pipeline completes bundle stage
    useEffect(() => {
      if (bundleResult && !hasNotifiedRef.current) {
        setLocalBundleDetails(bundleResult);

        if (bundleResult.manifest_path) {
          setLocalManifestPath(bundleResult.manifest_path);
          onBundleManifestChange(bundleResult.manifest_path);
          onBundleExported?.(bundleResult.manifest_path);
        }

        // Notify parent of completion
        onBundleComplete?.({
          bundleDetails: bundleResult,
          manifestPath: bundleResult.manifest_path ?? null,
        });

        hasNotifiedRef.current = true;
      }
    }, [bundleResult, onBundleManifestChange, onBundleExported, onBundleComplete]);

    // Reset notification flag when starting a new bundle
    useEffect(() => {
      if (runStatus === "starting" || runStatus === "running") {
        hasNotifiedRef.current = false;
      }
    }, [runStatus]);

    useImperativeHandle(ref, () => ({
      exportBundle: handleExport,
    }));

    const handleExport = async () => {
      if (!scenarioName.trim()) {
        return;
      }

      setLocalBundleDetails(null);
      setLocalManifestPath(null);
      hasNotifiedRef.current = false;

      try {
        await runBundleStage({
          // tier is currently unused by pipeline but included for future support
        });
      } catch {
        // Error is handled by the store
      }
    };

    // Calculate progress based on run status
    const progress = useMemo(() => {
      switch (runStatus) {
        case "idle":
          return localBundleDetails ? 100 : 0;
        case "starting":
          return 5;
        case "running":
          return 50;
        case "completed":
          return 100;
        case "failed":
        case "cancelled":
          return 0;
        default:
          return 0;
      }
    }, [runStatus, localBundleDetails]);

    const bundleLogs = stageLogs.bundle ?? [];
    const isBusy = isRunning;

    const buttonLabel = isBusy ? "Building bundle..." : "Generate bundle";

    const getStatusTone = (status: string) => {
      switch (status) {
        case "completed":
          return "bg-emerald-500/10 text-emerald-200 border-emerald-500/40";
        case "running":
        case "starting":
          return "bg-blue-500/10 text-blue-200 border-blue-500/40";
        case "failed":
          return "bg-rose-500/10 text-rose-200 border-rose-500/40";
        case "cancelled":
          return "bg-amber-500/10 text-amber-200 border-amber-500/40";
        default:
          return "bg-slate-500/10 text-slate-200 border-slate-500/40";
      }
    };

    // Determine what to display
    const displayDetails = bundleResult ?? localBundleDetails;
    const displayManifestPath = bundleResult?.manifest_path ?? localManifestPath;
    const showStatus = isBusy || displayDetails;

    return (
      <div className="rounded border border-slate-800 bg-black/20 p-3 space-y-3 text-xs text-slate-200">
        <div className="flex items-center justify-between gap-2">
          <div>
            <p className="text-xs uppercase tracking-wide text-slate-400">Bundle helper</p>
            <p className="text-sm font-semibold text-slate-100">Generate bundle via pipeline</p>
          </div>
        </div>
        <p className="text-xs text-slate-400">
          Generate a bundle manifest and stage binaries/assets for offline use.
        </p>
        <div className="space-y-1">
          <Label className="text-[11px] uppercase tracking-wide text-slate-400">Tier</Label>
          <Select value={tier} onChange={(e) => setTier(e.target.value)} className="mt-0.5">
            <option value="tier-2-desktop">tier-2-desktop</option>
            <option value="tier-3-mobile" disabled>
              tier-3-mobile (desktop export only)
            </option>
          </Select>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" variant="outline" onClick={handleExport} disabled={isBusy}>
            {buttonLabel}
          </Button>
        </div>

        {isBusy && !displayDetails && (
          <p className="text-[11px] text-slate-400 flex items-center gap-2">
            <Loader2 className="h-3 w-3 animate-spin" />
            Starting bundle stage...
          </p>
        )}

        {showStatus && (
          <div className="rounded-lg border border-slate-800 bg-slate-950/50 p-3 space-y-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <p className="text-[11px] uppercase tracking-wide text-slate-400">Bundle stage</p>
                <p className="text-sm font-semibold text-slate-100">Package scenario for desktop</p>
              </div>
              <span className={`rounded-full border px-2 py-0.5 text-[11px] ${getStatusTone(runStatus)}`}>
                {runStatus}
              </span>
            </div>

            {pipelineId && (
              <div className="text-[11px] text-slate-400">
                Pipeline ID: <span className="font-mono text-slate-300">{pipelineId.slice(0, 8)}...</span>
              </div>
            )}

            <div className="space-y-1">
              <div className="flex justify-between text-[11px] text-slate-400">
                <span>Progress</span>
                <span className="text-slate-300">{progress}%</span>
              </div>
              <Progress value={progress} />
            </div>

            {displayDetails && (
              <div className="space-y-2">
                <p className="text-[11px] uppercase tracking-wide text-slate-500">Bundle details</p>
                <div className="rounded-md border border-slate-800/70 bg-slate-950/70 p-2 space-y-1">
                  {displayDetails.bundle_dir && (
                    <div className="flex items-center gap-2 text-[11px]">
                      <CheckCircle className="h-4 w-4 text-emerald-400" />
                      <span className="text-slate-300">Bundle directory:</span>
                      <span className="text-slate-500 truncate max-w-[200px]">{displayDetails.bundle_dir}</span>
                    </div>
                  )}
                  {displayDetails.manifest_path && (
                    <div className="flex items-center gap-2 text-[11px]">
                      <CheckCircle className="h-4 w-4 text-emerald-400" />
                      <span className="text-slate-300">Manifest:</span>
                      <span className="text-slate-500 truncate max-w-[200px]">{displayDetails.manifest_path}</span>
                    </div>
                  )}
                  {displayDetails.total_size_human && (
                    <div className="flex items-center gap-2 text-[11px]">
                      <span className="text-slate-300">Total size:</span>
                      <span className="text-slate-500">{displayDetails.total_size_human}</span>
                    </div>
                  )}
                  {displayDetails.copied_artifacts && displayDetails.copied_artifacts.length > 0 && (
                    <div className="flex items-center gap-2 text-[11px]">
                      <span className="text-slate-300">Artifacts:</span>
                      <span className="text-slate-500">{displayDetails.copied_artifacts.length} files</span>
                    </div>
                  )}
                </div>

                {displayDetails.size_warning && (
                  <div className="flex items-start gap-2 text-[11px] text-amber-300">
                    <AlertTriangle className="h-4 w-4 flex-shrink-0 mt-0.5" />
                    <span>{displayDetails.size_warning.message}</span>
                  </div>
                )}

                {displayDetails.runtime_binaries && Object.keys(displayDetails.runtime_binaries).length > 0 && (
                  <div className="space-y-1 mt-2">
                    <p className="text-[11px] uppercase tracking-wide text-slate-500">Platform Builds</p>
                    <div className="rounded-md border border-slate-800/70 bg-slate-950/70 p-2 space-y-1">
                      {Object.entries(displayDetails.runtime_binaries).map(([platform, binaryPath]) => (
                        <div key={platform} className="flex items-center gap-2 text-[11px]">
                          <CheckCircle className="h-3 w-3 text-emerald-400 flex-shrink-0" />
                          <span className="font-medium text-slate-300 capitalize">{platform.replace("-", " ")}</span>
                          <span className="text-slate-500 truncate ml-auto max-w-[150px]" title={binaryPath}>
                            {binaryPath.split("/").pop()}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {bundleLogs.length > 0 && (
              <div>
                <p className="text-[11px] uppercase tracking-wide text-slate-500 mb-1">Build log</p>
                <div className="max-h-40 overflow-auto rounded-md border border-slate-800/70 bg-slate-950/80 p-2 font-mono text-[11px] text-slate-300">
                  {bundleLogs.map((line, idx) => (
                    <div key={idx}>{line}</div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {pipelineError && (
          <p className="rounded border border-amber-500/40 bg-amber-500/10 px-2 py-1 text-amber-100">
            {pipelineError}
          </p>
        )}

        {displayManifestPath && !isBusy && (
          <p className="text-[11px] text-slate-300">
            Saved to {displayManifestPath}
          </p>
        )}

        {!pipelineError && !displayDetails && !isBusy && (
          <p className="text-[11px] text-slate-400">
            Click Generate to run the pipeline bundle stage. This will package your scenario's binaries and assets
            for offline desktop use.
          </p>
        )}
      </div>
    );
  }
);
