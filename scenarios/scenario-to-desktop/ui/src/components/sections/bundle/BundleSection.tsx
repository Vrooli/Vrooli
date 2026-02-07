/**
 * Bundle section - orchestrates bundle stage with linear numbered steps.
 * Follows the same UX pattern as PreflightSection for consistency.
 *
 * Architecture: Pure store-driven rendering
 * - Reads state from pipelineStore via selectors
 * - Triggers actions via store methods on user interaction (button clicks)
 * - NO effects that trigger API mutations or pipeline creation
 * - NO local state that duplicates store state
 *
 * Steps:
 * 1. Bundle context - Shows scenario and manifest path configuration
 * 2. Generate bundle - Button to run bundle stage + progress
 * 3. Bundle results - Shows artifacts, sizes, and warnings
 */

import { forwardRef, useCallback, useEffect, useMemo, useRef, type Ref } from "react";
import { Braces, Copy, Download, LayoutList, Loader2, Package } from "lucide-react";
import { useState } from "react";
import {
  SectionCard,
  StagePlaceholder,
} from "../shared";
import { usePipelineStore, selectStageStatus, selectIsRunning } from "../../../store";
import { Button } from "../../ui/button";
import { Select } from "../../ui/select";
import { Progress } from "../../ui/progress";
import { BundleStepHeader } from "./BundleStepHeader";
import { BundleManifestInput } from "./BundleManifestInput";
import { BundleResultsCard } from "./BundleResultsCard";
import { writeToClipboard, triggerBlobDownload } from "../../../lib/browser";
import type { PreflightStepStatus } from "../../../lib/preflight-constants";
import type { BundleStageDetails } from "../../../lib/api";

/** Bundle result data for stage persistence */
export interface BundleResult {
  bundleDetails: BundleStageDetails | null;
  manifestPath: string | null;
  checksum?: string;
  generatedAt?: string;
}

/**
 * Handle for imperative bundle control.
 * Compatible with legacy DeploymentManagerBundleHelperHandle.
 */
export type BundleSectionHandle = {
  exportBundle: () => void;
};

/**
 * @deprecated Use BundleSectionHandle instead.
 * Kept for backwards compatibility.
 */
export type DeploymentManagerBundleHelperHandle = BundleSectionHandle;

interface BundleSectionProps {
  scenarioName: string;
  /** Whether the app is configured for bundled runtime mode */
  isBundled?: boolean;
  /** Current bundle manifest path */
  bundleManifestPath?: string;
  /** Callback when bundle manifest path changes */
  onBundleManifestChange?: (path: string) => void;
  /** Callback when bundle export completes successfully - used for server persistence */
  onBundleComplete?: (result: BundleResult) => void;
  /** Ref to bundle helper for imperative control */
  bundleHelperRef?: Ref<BundleSectionHandle>;
}

export const BundleSection = forwardRef<HTMLDivElement, BundleSectionProps>(
  (
    {
      scenarioName,
      isBundled = false,
      bundleManifestPath = "",
      onBundleManifestChange,
      onBundleComplete,
      bundleHelperRef,
    },
    ref
  ) => {
    // UI-only local state (following PreflightSection pattern)
    const [viewMode, setViewMode] = useState<"summary" | "json">("summary");
    const [copyStatus, setCopyStatus] = useState<"idle" | "copied" | "error">("idle");
    const [tier, setTier] = useState("tier-2-desktop");

    // Pipeline store state - single source of truth
    const bundleResult = usePipelineStore((s) => s.bundleResult);
    const stageStatus = usePipelineStore(selectStageStatus("bundle"));
    const pipelineStatus = usePipelineStore((s) => s.pipelineStatus);
    const runStatus = usePipelineStore((s) => s.runStatus);
    const errorInfo = usePipelineStore((s) => s.errorInfo);
    const stageLogs = usePipelineStore((s) => s.stageLogs);
    const pipelineId = usePipelineStore((s) => s.pipelineId);
    const isRunning = usePipelineStore(selectIsRunning);

    // Pipeline store actions
    const runBundleStage = usePipelineStore((s) => s.runBundleStage);
    const setScenario = usePipelineStore((s) => s.setScenario);
    const clearError = usePipelineStore((s) => s.clearError);

    // Track previous scenario for change detection (minimal ref usage)
    const prevScenarioRef = useRef<string>(scenarioName);
    // Track if we've notified for the current bundle result to prevent duplicates
    const lastNotifiedResultRef = useRef<BundleStageDetails | null>(null);

    // Set scenario in store when it changes
    useEffect(() => {
      if (scenarioName) {
        setScenario(scenarioName);
      }
    }, [scenarioName, setScenario]);

    // Clear error when scenario changes
    useEffect(() => {
      const prevScenario = prevScenarioRef.current;
      prevScenarioRef.current = scenarioName;

      if (prevScenario !== scenarioName && prevScenario !== "") {
        clearError();
        lastNotifiedResultRef.current = null;
      }
    }, [scenarioName, clearError]);

    // Notify parent of bundle completion for server persistence
    // Only fires when bundleResult changes AND bundle stage is complete
    useEffect(() => {
      const bundleStage = pipelineStatus?.stages?.bundle;
      const isComplete = bundleStage?.status === "completed";

      // Only notify if:
      // 1. We have a bundle result
      // 2. The stage is marked complete
      // 3. We haven't already notified for this exact result
      if (bundleResult && isComplete && bundleResult !== lastNotifiedResultRef.current) {
        lastNotifiedResultRef.current = bundleResult;
        onBundleComplete?.({
          bundleDetails: bundleResult,
          manifestPath: bundleResult.manifest_path ?? null,
        });

        // Also update manifest path in parent if changed
        if (bundleResult.manifest_path) {
          onBundleManifestChange?.(bundleResult.manifest_path);
        }
      }
    }, [bundleResult, pipelineStatus?.stages?.bundle, onBundleComplete, onBundleManifestChange]);

    // Expose imperative handle
    useEffect(() => {
      if (bundleHelperRef && typeof bundleHelperRef === "object" && bundleHelperRef !== null) {
        (bundleHelperRef as React.MutableRefObject<BundleSectionHandle | null>).current = {
          exportBundle: handleExport,
        };
      }
    });

    const handleExport = useCallback(async () => {
      if (!scenarioName.trim()) {
        return;
      }

      try {
        await runBundleStage({});
      } catch {
        // Error is handled by the store
      }
    }, [scenarioName, runBundleStage]);

    // Derived state - use store directly, no local cache
    const displayDetails = bundleResult;
    const displayManifestPath = bundleResult?.manifest_path ?? null;
    const pipelineError = errorInfo?.message ?? null;
    const bundleLogs = stageLogs.bundle ?? [];
    const isBusy = isRunning;
    const hasResult = Boolean(bundleResult);
    const hasRun = Boolean(bundleResult || pipelineError || pipelineStatus);

    // Calculate progress based on run status
    const progress = useMemo(() => {
      switch (runStatus) {
        case "idle":
          return bundleResult ? 100 : 0;
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
    }, [runStatus, bundleResult]);

    // Step status calculations
    const getContextStatus = (): PreflightStepStatus => {
      if (!scenarioName) {
        return { state: "pending", label: "Pending" };
      }
      return { state: "pass", label: "Ready" };
    };

    const getGenerateStatus = (): PreflightStepStatus => {
      if (isBusy) {
        return { state: "testing", label: "Running" };
      }
      if (stageStatus === "failed" || pipelineError) {
        return { state: "fail", label: "Failed" };
      }
      if (hasResult) {
        return { state: "pass", label: "Complete" };
      }
      return { state: "pending", label: "Pending" };
    };

    const getResultsStatus = (): PreflightStepStatus => {
      if (!hasRun) {
        return { state: "pending", label: "Pending" };
      }
      if (stageStatus === "failed" || pipelineError) {
        return { state: "fail", label: "Failed" };
      }
      if (hasResult) {
        return { state: "pass", label: "Available" };
      }
      return { state: "pending", label: "Pending" };
    };

    // JSON export payload
    const bundlePayload = useMemo(
      () => ({
        scenario_name: scenarioName,
        bundle_manifest_path: displayManifestPath,
        result: displayDetails,
        error: pipelineError || undefined,
      }),
      [scenarioName, displayManifestPath, displayDetails, pipelineError]
    );

    const copyJson = async () => {
      if (!displayDetails && !pipelineError) {
        return;
      }
      const result = await writeToClipboard(JSON.stringify(bundlePayload, null, 2));
      if (result.success) {
        setCopyStatus("copied");
        setTimeout(() => setCopyStatus("idle"), 1500);
      } else {
        console.warn("Failed to copy bundle JSON", result.error);
        setCopyStatus("error");
        setTimeout(() => setCopyStatus("idle"), 2000);
      }
    };

    const downloadJson = () => {
      const blob = new Blob([JSON.stringify(bundlePayload, null, 2)], { type: "application/json" });
      triggerBlobDownload(blob, "bundle.json");
    };

    // Non-bundled mode: Show placeholder
    if (!isBundled) {
      return (
        <SectionCard
          ref={ref}
          sectionId="bundle"
          title="Bundle"
          subtitle="Package dependencies for distribution"
          variant="pipeline"
          collapsible={true}
        >
          <StagePlaceholder
            scenarioName={scenarioName}
            withScenarioText="Bundle stage will run when you generate the desktop application."
          />
        </SectionCard>
      );
    }

    return (
      <SectionCard
        ref={ref}
        sectionId="bundle"
        title="Bundle"
        subtitle="Package dependencies for distribution"
        variant="pipeline"
        collapsible={true}
        contentClassName="space-y-3"
      >
        {/* Header */}
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p className="text-sm font-semibold text-slate-100">Bundle generation</p>
            <p className="text-xs text-slate-400">
              Packages scenario assets, runtime binaries, and dependencies for offline desktop use.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <div className="flex items-center gap-1 rounded-full border border-slate-800/70 bg-slate-950/60 p-1 text-[11px]">
              <Button
                type="button"
                size="sm"
                variant={viewMode === "summary" ? "default" : "ghost"}
                onClick={() => setViewMode("summary")}
                aria-label="Show bundle summary"
                className="h-10 w-10 p-0"
              >
                <LayoutList className="h-4 w-4" />
              </Button>
              <Button
                type="button"
                size="sm"
                variant={viewMode === "json" ? "default" : "ghost"}
                onClick={() => setViewMode("json")}
                aria-label="Show bundle JSON"
                className="h-10 w-10 p-0"
              >
                <Braces className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>

        {/* Error display */}
        {pipelineError && (
          <div className="rounded-md border border-red-800 bg-red-950/40 p-3 text-sm text-red-200">
            {pipelineError}
          </div>
        )}

        {/* Summary View */}
        {viewMode === "summary" && (
          <div className="space-y-4">
            {/* Step 1: Bundle Context */}
            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <BundleStepHeader
                index={1}
                title="Bundle context"
                subtitle="Scenario and manifest configuration"
                status={getContextStatus()}
              />
              <div className="space-y-2 text-[11px] text-slate-300">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="text-slate-400">Scenario</span>
                  <span className="text-slate-200">{scenarioName || "Not selected"}</span>
                </div>
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="text-slate-400">Tier</span>
                  <Select
                    value={tier}
                    onChange={(e) => setTier(e.target.value)}
                    className="h-7 text-[11px] w-auto min-w-[140px]"
                  >
                    <option value="tier-2-desktop">tier-2-desktop</option>
                    <option value="tier-3-mobile" disabled>
                      tier-3-mobile (coming soon)
                    </option>
                  </Select>
                </div>
              </div>
              {onBundleManifestChange && (
                <BundleManifestInput
                  value={bundleManifestPath}
                  onChange={onBundleManifestChange}
                  disabled={isBusy}
                />
              )}
            </div>

            {/* Step 2: Generate Bundle */}
            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <BundleStepHeader
                index={2}
                title="Generate bundle"
                subtitle="Run the bundle pipeline stage"
                status={getGenerateStatus()}
              />
              <p className="text-[11px] text-slate-400">
                Runs the pipeline bundle stage to collect assets and generate the manifest.
              </p>
              <div className="flex flex-wrap items-center gap-2">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={handleExport}
                  disabled={isBusy || !scenarioName}
                  className="gap-2"
                >
                  {isBusy ? (
                    <>
                      <Loader2 className="h-3 w-3 animate-spin" />
                      Building bundle...
                    </>
                  ) : (
                    <>
                      <Package className="h-3 w-3" />
                      Generate bundle
                    </>
                  )}
                </Button>
              </div>

              {/* Progress indicator */}
              {(isBusy || (hasResult && runStatus === "completed")) && (
                <div className="space-y-2">
                  <div className="flex justify-between text-[11px] text-slate-400">
                    <span>Progress</span>
                    <span className="text-slate-300">{progress}%</span>
                  </div>
                  <Progress value={progress} />
                  {pipelineId && (
                    <div className="text-[11px] text-slate-400">
                      Pipeline ID: <span className="font-mono text-slate-300">{pipelineId.slice(0, 8)}...</span>
                    </div>
                  )}
                </div>
              )}

              {/* Build logs */}
              {bundleLogs.length > 0 && (
                <div>
                  <p className="text-[11px] uppercase tracking-wide text-slate-500 mb-1">Build log</p>
                  <div className="max-h-32 overflow-auto rounded-md border border-slate-800/70 bg-slate-950/80 p-2 font-mono text-[11px] text-slate-300">
                    {bundleLogs.map((line, idx) => (
                      <div key={idx}>{line}</div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Step 3: Bundle Results */}
            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <BundleStepHeader
                index={3}
                title="Bundle results"
                subtitle="Generated artifacts and diagnostics"
                status={getResultsStatus()}
              />
              {!hasRun && (
                <p className="text-[11px] text-slate-400">
                  Run the bundle stage to generate results.
                </p>
              )}
              {displayDetails && <BundleResultsCard result={displayDetails} />}
              {displayManifestPath && !isBusy && (
                <p className="text-[11px] text-emerald-300">
                  Bundle saved to {displayManifestPath}
                </p>
              )}
              {hasRun && !displayDetails && !pipelineError && (
                <p className="text-[11px] text-slate-400">
                  No bundle results yet. The bundle stage may still be running.
                </p>
              )}
            </div>
          </div>
        )}

        {/* JSON View */}
        {viewMode === "json" && (
          <div className="rounded-md border border-slate-800 bg-slate-950/60 p-3 space-y-2">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-xs font-semibold text-slate-200">Bundle JSON</p>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={copyJson}
                  disabled={!displayDetails && !pipelineError}
                  aria-label="Copy bundle JSON"
                  className="h-10 w-10 p-0"
                >
                  <Copy className="h-4 w-4" />
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={downloadJson}
                  disabled={!displayDetails && !pipelineError}
                  aria-label="Download bundle JSON"
                  className="h-10 w-10 p-0"
                >
                  <Download className="h-4 w-4" />
                </Button>
              </div>
            </div>
            <pre className="max-h-96 overflow-auto whitespace-pre-wrap rounded-md border border-slate-800/70 bg-slate-950/80 p-3 text-[11px] text-slate-200">
              {JSON.stringify(bundlePayload, null, 2)}
            </pre>
            {copyStatus !== "idle" && (
              <p className="text-[11px] text-slate-400">
                {copyStatus === "copied" ? "Copied to clipboard." : "Copy failed."}
              </p>
            )}
            <p className="text-[11px] text-slate-400">
              Use this view to share the full bundle snapshot with an agent or teammate.
            </p>
          </div>
        )}
      </SectionCard>
    );
  }
);

BundleSection.displayName = "BundleSection";
