/**
 * Bundle section - displays bundle stage status and results from the pipeline store.
 * Also includes the BundledRuntimeSection for configuring bundle manifests when in bundled mode.
 */

import { forwardRef, type Ref } from "react";
import { Package, FileJson, FolderOpen, HardDrive } from "lucide-react";
import {
  SectionCard,
  getStatusDisplay,
  StageAbout,
  StageStatusOverview,
  StageDetailCard,
  StagePlaceholder,
  StageError,
  StageWarning,
} from "../shared";
import { usePipelineStore, selectStageStatus } from "../../../store";
import { BundledRuntimeSection } from "../../runtime";
import type { DeploymentManagerBundleHelperHandle, BundleResult } from "../../runtime/DeploymentManagerBundleHelper";
import type { PipelineConfig } from "../../../lib/api";

interface BundleSectionProps {
  scenarioName: string;
  /** Whether the app is configured for bundled runtime mode */
  isBundled?: boolean;
  /** Current bundle manifest path */
  bundleManifestPath?: string;
  /** Callback when bundle manifest path changes */
  onBundleManifestChange?: (path: string) => void;
  /** Callback when bundle is exported (triggers preflight) */
  onBundleExported?: (manifestPath: string, config?: Partial<PipelineConfig>) => void;
  /** Callback when bundle export completes successfully */
  onBundleComplete?: (result: BundleResult) => void;
  /** Initial bundle result for restoration from server persistence */
  initialBundleResult?: BundleResult | null;
  /** Ref to bundle helper for imperative control */
  bundleHelperRef?: Ref<DeploymentManagerBundleHelperHandle>;
}

export const BundleSection = forwardRef<HTMLDivElement, BundleSectionProps>(
  (
    {
      scenarioName,
      isBundled = false,
      bundleManifestPath = "",
      onBundleManifestChange,
      onBundleExported,
      onBundleComplete,
      initialBundleResult,
      bundleHelperRef,
    },
    ref
  ) => {
    const bundleResult = usePipelineStore((s) => s.bundleResult);
    const stageStatus = usePipelineStore(selectStageStatus("bundle"));

    const hasResult = Boolean(bundleResult);
    const manifestPath = bundleResult?.manifest_path;
    const bundleDir = bundleResult?.bundle_dir;
    const totalSize = bundleResult?.total_size_human;
    const copiedArtifacts = bundleResult?.copied_artifacts ?? [];
    const runtimeBinaries = bundleResult?.runtime_binaries ?? {};
    const artifactCount = copiedArtifacts.length + Object.keys(runtimeBinaries).length;
    const statusDisplay = getStatusDisplay(stageStatus);

    return (
      <SectionCard
        ref={ref}
        sectionId="bundle"
        title="Bundle"
        subtitle="Package dependencies for distribution"
        variant="pipeline"
        collapsible={true}
        contentClassName="space-y-4"
      >
        <StageAbout title="About bundling">
          <p>The bundle stage collects scenario assets, runtime binaries, and dependencies into a portable package.</p>
        </StageAbout>

        {/* Bundle manifest configuration - only shown when in bundled mode */}
        {isBundled && onBundleManifestChange && (
          <BundledRuntimeSection
            bundleManifestPath={bundleManifestPath}
            onBundleManifestChange={onBundleManifestChange}
            scenarioName={scenarioName}
            bundleHelperRef={bundleHelperRef ?? null}
            onBundleExported={onBundleExported}
            onBundleComplete={onBundleComplete}
            initialBundleResult={initialBundleResult}
          />
        )}

        <StageStatusOverview
          icon={Package}
          title="Bundle Status"
          description={hasResult ? `${artifactCount} artifact${artifactCount !== 1 ? "s" : ""} bundled${totalSize ? ` (${totalSize})` : ""}` : "No bundle results yet"}
          statusDisplay={statusDisplay}
        />

        {hasResult && (
          <div className="space-y-3">
            {bundleDir && (
              <StageDetailCard icon={FolderOpen} label="Bundle Directory">
                <code className="text-xs text-slate-300 font-mono break-all">{bundleDir}</code>
              </StageDetailCard>
            )}
            {manifestPath && (
              <StageDetailCard icon={FileJson} label="Manifest Path">
                <code className="text-xs text-slate-300 font-mono break-all">{manifestPath}</code>
              </StageDetailCard>
            )}
            {totalSize && (
              <div className="flex items-center gap-2 text-xs text-slate-400">
                <HardDrive className="h-3.5 w-3.5" />
                <span>Total bundle size: {totalSize}</span>
              </div>
            )}
            {bundleResult?.size_warning && (
              <StageWarning>
                <strong>Warning:</strong> {bundleResult.size_warning.message}
              </StageWarning>
            )}
          </div>
        )}

        {!hasResult && stageStatus === "pending" && !isBundled && (
          <StagePlaceholder
            scenarioName={scenarioName}
            withScenarioText="Bundle stage will run when you generate the desktop application."
          />
        )}

        {stageStatus === "failed" && <StageError stageName="Bundle" />}
      </SectionCard>
    );
  }
);

BundleSection.displayName = "BundleSection";
