/**
 * Bundle section - displays bundle stage status and results from the pipeline store.
 */

import { forwardRef } from "react";
import { Package, FileJson, FolderOpen, HardDrive } from "lucide-react";
import {
  SectionCard,
  getStatusDisplay,
  StageStatusOverview,
  StageDetailCard,
  StagePlaceholder,
  StageError,
  StageWarning,
} from "../shared";
import { usePipelineStore, selectStageStatus } from "../../../store";

interface BundleSectionProps {
  scenarioName: string;
}

export const BundleSection = forwardRef<HTMLDivElement, BundleSectionProps>(
  ({ scenarioName }, ref) => {
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

        {!hasResult && stageStatus === "pending" && (
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
