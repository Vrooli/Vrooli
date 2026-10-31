/**
 * Release handoff section. Packaging happens here; deployment-manager owns promotion.
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
import {
  Platform,
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

interface DeploySectionProps {
  scenarioName: string;
}

function statusConfig(status: StageStatus) {
  const config = STATUS_CONFIG[status] ?? STATUS_CONFIG[StageStatus.PENDING];
  if (!config) throw new Error("Missing pending stage status configuration");
  return config;
}

function platformLabel(platform: Platform): string {
  switch (platform) {
    case Platform.WIN:
      return "Windows";
    case Platform.MAC:
      return "macOS";
    case Platform.LINUX:
      return "Linux";
    default:
      return "Unknown platform";
  }
}

export const DeploySection = forwardRef<HTMLDivElement, DeploySectionProps>(
  ({ scenarioName }, ref) => {
    const deployResult = usePipelineStore((s) => s.deployResult);
    const buildResult = usePipelineStore((s) => s.buildResult);
    const stageStatus = usePipelineStore(selectStageStatus(StageName.DEPLOY));
    const hasResult = Boolean(deployResult);
    const hasBuildArtifacts =
      Object.keys(buildResult?.artifacts ?? {}).length > 0;
    const artifacts = deployResult?.artifacts ?? [];
    const updateUrl = deployResult?.updateUrl;

    const statusDisplay = useMemo(() => {
      if (stageStatus === StageStatus.COMPLETED)
        return {
          ...statusConfig(StageStatus.COMPLETED),
          label: "Ready for Promotion",
        };
      if (stageStatus === StageStatus.RUNNING)
        return {
          ...statusConfig(StageStatus.RUNNING),
          label: "Preparing Handoff",
        };
      if (stageStatus === StageStatus.SKIPPED)
        return { ...statusConfig(StageStatus.SKIPPED), label: "Skipped" };
      return statusConfig(stageStatus);
    }, [stageStatus]);

    const getDescription = () => {
      if (hasResult && artifacts.length > 0)
        return `${String(artifacts.length)} artifact(s) ready for deployment-manager`;
      if (stageStatus === StageStatus.SKIPPED)
        return "Promotion is managed outside this scenario";
      return hasBuildArtifacts
        ? "Ready to deploy"
        : "Waiting for build artifacts";
    };

    return (
      <SectionCard
        ref={ref}
        sectionId="deploy"
        title="Release Handoff"
        subtitle="Artifacts for deployment-manager"
        variant="pipeline"
        collapsible={true}
        contentClassName="space-y-4"
      >
        <StageAbout title="About release handoff">
          <p>
            scenario-to-desktop builds and evidences desktop artifacts.
            deployment-manager owns promotion, approvals, and distribution.
          </p>
        </StageAbout>

        <StageStatusOverview
          icon={Cloud}
          title="Handoff Status"
          description={getDescription()}
          statusDisplay={statusDisplay}
        />

        {hasResult && (
          <div className="space-y-3">
            {artifacts.length > 0 && (
              <StageDetailCard
                icon={Package}
                label={`Prepared Artifacts (${String(artifacts.length)})`}
              >
                <div className="space-y-2">
                  {artifacts.map((a, i) => (
                    <div
                      key={i}
                      className="flex items-center gap-2 rounded border border-slate-800 bg-slate-950/50 p-2"
                    >
                      <CheckCircle2 className="h-4 w-4 text-green-400 shrink-0" />
                      <div className="min-w-0">
                        <p className="text-sm text-slate-300">
                          {platformLabel(a.platform)}
                        </p>
                        {a.artifactId !== 0n && (
                          <p className="text-xs text-slate-500">
                            artifact #{String(a.artifactId)}
                          </p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </StageDetailCard>
            )}

            {updateUrl && (
              <StageDetailCard icon={Link} label="Update URL">
                <code className="text-xs text-slate-300 font-mono break-all">
                  {updateUrl}
                </code>
              </StageDetailCard>
            )}
          </div>
        )}

        {!hasResult && stageStatus === StageStatus.PENDING && (
          <StagePlaceholder
            scenarioName={scenarioName}
            withScenarioText="Build artifacts are handed to deployment-manager for promotion."
            withoutScenarioText="Select a scenario to prepare deployment artifacts."
          />
        )}

        {stageStatus === StageStatus.FAILED && !hasResult && (
          <StageError stageName="Release handoff" />
        )}
      </SectionCard>
    );
  },
);

DeploySection.displayName = "DeploySection";
