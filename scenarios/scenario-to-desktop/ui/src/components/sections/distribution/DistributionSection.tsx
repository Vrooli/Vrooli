/**
 * Distribution section - displays distribution stage status and results from the pipeline store.
 */

import { forwardRef, useMemo } from "react";
import { Cloud, CheckCircle2, XCircle, Clock, ExternalLink, Server } from "lucide-react";
import {
  SectionCard,
  STATUS_CONFIG,
  StageStatusOverview,
  StageAbout,
  StagePlaceholder,
  StageError,
} from "../shared";
import { usePipelineStore, selectStageStatus } from "../../../store";
import { Badge } from "../../ui/badge";
import { DistributionUploadSection } from "../../distribution";
import type { DistributionPlatformUpload, DistributionTargetStatus } from "../../../lib/api";

interface DistributionSectionProps {
  scenarioName: string;
}

function flattenUploads(targets?: Record<string, DistributionTargetStatus>): Array<{ targetName: string; platform: string; upload: DistributionPlatformUpload }> {
  if (!targets) return [];
  const flattened: Array<{ targetName: string; platform: string; upload: DistributionPlatformUpload }> = [];
  for (const [targetName, target] of Object.entries(targets)) {
    if (target.uploads) {
      for (const [platform, upload] of Object.entries(target.uploads)) {
        flattened.push({ targetName, platform, upload });
      }
    }
  }
  return flattened;
}

export const DistributionSection = forwardRef<HTMLDivElement, DistributionSectionProps>(
  ({ scenarioName }, ref) => {
    const distributionResult = usePipelineStore((s) => s.distributionResult);
    const buildResult = usePipelineStore((s) => s.buildResult);
    const stageStatus = usePipelineStore(selectStageStatus("distribution"));

    const hasResult = Boolean(distributionResult);
    const hasBuildArtifacts = Object.keys(buildResult?.artifacts ?? {}).length > 0;
    const targets = distributionResult?.targets ?? {};
    const targetEntries = Object.entries(targets);
    const uploads = useMemo(() => flattenUploads(distributionResult?.targets), [distributionResult?.targets]);
    const successfulUploads = uploads.filter((u) => u.upload.status === "completed").length;
    const { version, error } = distributionResult ?? {};
    const artifactsMap = buildResult?.artifacts ?? {};

    const statusDisplay = useMemo(() => {
      if (stageStatus === "completed") return { ...STATUS_CONFIG.completed, label: successfulUploads > 0 ? "Uploaded" : "Completed" };
      if (stageStatus === "running") return { ...STATUS_CONFIG.running, label: "Uploading" };
      return STATUS_CONFIG[stageStatus as keyof typeof STATUS_CONFIG] ?? STATUS_CONFIG.pending;
    }, [stageStatus, successfulUploads]);

    const getDescription = () => {
      if (hasResult) return `${successfulUploads}/${uploads.length} artifacts uploaded${version ? ` (v${version})` : ""}`;
      return hasBuildArtifacts ? "Ready to upload" : "Waiting for build artifacts";
    };

    return (
      <SectionCard
        ref={ref}
        sectionId="distribution"
        title="Distribution"
        subtitle="Upload artifacts to cloud storage"
        variant="pipeline"
        collapsible={true}
        contentClassName="space-y-4"
      >
        <StageAbout title="About distribution">
          <p>Upload built installers to cloud storage for public distribution.</p>
        </StageAbout>

        <StageStatusOverview icon={Cloud} title="Distribution Status" description={getDescription()} statusDisplay={statusDisplay} />

        {hasResult && targetEntries.length > 0 && (
          <div className="space-y-3">
            {targetEntries.map(([targetName, target]) => (
              <div key={targetName} className="rounded-lg border border-slate-800 bg-slate-950/50 p-3">
                <div className="flex items-center gap-2 text-xs text-slate-400 mb-2">
                  <Server className="h-3.5 w-3.5" />
                  <span className="font-medium text-slate-300">{targetName}</span>
                  <Badge variant="outline" className="text-xs">{target.status}</Badge>
                </div>
                {target.uploads && Object.keys(target.uploads).length > 0 && (
                  <div className="space-y-2">
                    {Object.entries(target.uploads).map(([platform, upload]) => (
                      <div key={platform} className="flex items-center justify-between rounded-md border border-slate-700 bg-slate-900/50 p-2">
                        <div className="flex items-center gap-2">
                          {upload.status === "completed" ? <CheckCircle2 className="h-4 w-4 text-green-400" /> : upload.status === "failed" ? <XCircle className="h-4 w-4 text-red-400" /> : <Clock className="h-4 w-4 text-slate-400" />}
                          <span className="text-sm text-slate-300">{platform}</span>
                        </div>
                        {upload.url && (
                          <a href={upload.url} target="_blank" rel="noopener noreferrer" className="flex items-center gap-1 text-xs text-blue-400 hover:text-blue-300">
                            <ExternalLink className="h-3 w-3" />View
                          </a>
                        )}
                      </div>
                    ))}
                  </div>
                )}
                {target.error && <p className="mt-2 text-xs text-red-300">{target.error}</p>}
              </div>
            ))}
          </div>
        )}

        {error && <StageError stageName="Distribution"><strong>Error:</strong> {error}</StageError>}

        {hasBuildArtifacts && !hasResult && scenarioName && (
          <DistributionUploadSection scenarioName={scenarioName} artifacts={artifactsMap} />
        )}

        {!hasBuildArtifacts && stageStatus === "pending" && (
          <StagePlaceholder
            scenarioName={scenarioName}
            withScenarioText="Distribution will be available after building installers in the Build section above."
            withoutScenarioText="Select a scenario to enable distribution."
          />
        )}

        {stageStatus === "failed" && !hasResult && <StageError stageName="Distribution" />}
      </SectionCard>
    );
  }
);

DistributionSection.displayName = "DistributionSection";
