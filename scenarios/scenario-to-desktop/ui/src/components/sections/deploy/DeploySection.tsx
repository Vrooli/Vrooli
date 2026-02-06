/**
 * Deploy section - displays LPBS deployment stage status and results from the pipeline store.
 */

import { forwardRef, useMemo } from "react";
import { Cloud, Package, Link, CheckCircle2 } from "lucide-react";
import {
  SectionCard,
  STATUS_CONFIG,
  StageAbout,
  StageStatusOverview,
  StageDetailCard,
  StagePlaceholder,
  StageError,
} from "../shared";
import { usePipelineStore, selectStageStatus } from "../../../store";

interface DeploySectionProps {
  scenarioName: string;
}

export const DeploySection = forwardRef<HTMLDivElement, DeploySectionProps>(
  ({ scenarioName }, ref) => {
    const deployResult = usePipelineStore((s) => s.deployResult);
    const buildResult = usePipelineStore((s) => s.buildResult);
    const stageStatus = usePipelineStore(selectStageStatus("deploy"));

    const hasResult = Boolean(deployResult);
    const hasBuildArtifacts = Object.keys(buildResult?.artifacts ?? {}).length > 0;
    const artifacts = deployResult?.artifacts ?? [];
    const updateUrl = deployResult?.update_url;

    const statusDisplay = useMemo(() => {
      if (stageStatus === "completed") return { ...STATUS_CONFIG.completed, label: "Deployed" };
      if (stageStatus === "running") return { ...STATUS_CONFIG.running, label: "Deploying" };
      if (stageStatus === "skipped") return { ...STATUS_CONFIG.skipped, label: "Skipped" };
      return STATUS_CONFIG[stageStatus as keyof typeof STATUS_CONFIG] ?? STATUS_CONFIG.pending;
    }, [stageStatus]);

    const getDescription = () => {
      if (hasResult && artifacts.length > 0)
        return `${artifacts.length} artifact(s) deployed to LPBS`;
      if (stageStatus === "skipped") return "Deploy stage was skipped";
      return hasBuildArtifacts ? "Ready to deploy" : "Waiting for build artifacts";
    };

    return (
      <SectionCard
        ref={ref}
        sectionId="deploy"
        title="Deploy"
        subtitle="Upload to LPBS"
        variant="pipeline"
        collapsible={true}
        contentClassName="space-y-4"
      >
        <StageAbout title="About deployment">
          <p>
            The deploy stage uploads built artifacts to a remote LPBS instance, registers them as
            download artifacts, and derives an auto-update endpoint URL.
          </p>
        </StageAbout>

        <StageStatusOverview icon={Cloud} title="Deploy Status" description={getDescription()} statusDisplay={statusDisplay} />

        {hasResult && (
          <div className="space-y-3">
            {artifacts.length > 0 && (
              <StageDetailCard icon={Package} label={`Uploaded Artifacts (${artifacts.length})`}>
                <div className="space-y-2">
                  {artifacts.map((a, i) => (
                    <div key={i} className="flex items-center gap-2 rounded border border-slate-800 bg-slate-950/50 p-2">
                      <CheckCircle2 className="h-4 w-4 text-green-400 shrink-0" />
                      <div className="min-w-0">
                        <p className="text-sm text-slate-300">{a.platform ?? "unknown"}</p>
                        {a.artifact_id && (
                          <p className="text-xs text-slate-500">artifact #{a.artifact_id}</p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </StageDetailCard>
            )}

            {updateUrl && (
              <StageDetailCard icon={Link} label="Update URL">
                <code className="text-xs text-slate-300 font-mono break-all">{updateUrl}</code>
              </StageDetailCard>
            )}
          </div>
        )}

        {!hasResult && stageStatus === "pending" && (
          <StagePlaceholder
            scenarioName={scenarioName}
            withScenarioText="Configure a deploy target with --deploy-target to enable deployment."
            withoutScenarioText="Select a scenario to enable deployment."
          />
        )}

        {stageStatus === "failed" && !hasResult && <StageError stageName="Deploy" />}
      </SectionCard>
    );
  }
);

DeploySection.displayName = "DeploySection";
