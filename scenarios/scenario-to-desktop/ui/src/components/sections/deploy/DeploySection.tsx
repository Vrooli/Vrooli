/**
 * Deploy section - displays LPBS deployment stage status and results from the pipeline store.
 */

import { forwardRef, useMemo } from "react";
import { Cloud, Package, Link, CheckCircle2, ShieldAlert, Clock, ShieldCheck } from "lucide-react";
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
    const currentState = usePipelineStore((s) => s.pipelineStatus?.current_state);
    const currentStage = usePipelineStore((s) => s.pipelineStatus?.current_stage);
    const deployConfig = usePipelineStore((s) => s.pipelineStatus?.config?.deploy);

    type GateDisplayState = "none" | "waiting" | "checking" | "passed" | "failed";
    const gateState: GateDisplayState = (() => {
      if (!deployConfig?.deployment_manager_profile_id) return "none";
      if (currentState === "gate_blocked" && currentStage === "deploy") return "waiting";
      if (stageStatus === "running" && currentStage === "deploy" && currentState === "executing_stage") return "checking";
      if (stageStatus === "completed") return "passed";
      if (stageStatus === "failed") return "failed";
      return "none";
    })();

    const hasResult = Boolean(deployResult);
    const hasBuildArtifacts = Object.keys(buildResult?.artifacts ?? {}).length > 0;
    const artifacts = deployResult?.artifacts ?? [];
    const updateUrl = deployResult?.update_url;

    const statusDisplay = useMemo(() => {
      if (gateState === "waiting") return { ...STATUS_CONFIG.running, label: "Awaiting Approval" };
      if (stageStatus === "completed") return { ...STATUS_CONFIG.completed, label: "Deployed" };
      if (stageStatus === "running") return { ...STATUS_CONFIG.running, label: "Deploying" };
      if (stageStatus === "skipped") return { ...STATUS_CONFIG.skipped, label: "Skipped" };
      return STATUS_CONFIG[stageStatus as keyof typeof STATUS_CONFIG] ?? STATUS_CONFIG.pending;
    }, [stageStatus, gateState]);

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

        {gateState !== "none" && (
          <StageDetailCard
            icon={gateState === "waiting" ? ShieldAlert : gateState === "checking" ? Clock : gateState === "passed" ? ShieldCheck : ShieldAlert}
            label="Approval Gate"
          >
            {gateState === "waiting" && (
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-amber-400 animate-pulse" />
                <span className="text-sm text-amber-300">Waiting for approval in deployment-manager</span>
              </div>
            )}
            {gateState === "checking" && (
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-blue-400 animate-pulse" />
                <span className="text-sm text-blue-300">Checking gate status...</span>
              </div>
            )}
            {gateState === "passed" && (
              <div className="flex items-center gap-2">
                <CheckCircle2 className="h-4 w-4 text-green-400" />
                <span className="text-sm text-green-300">Approval gate passed</span>
              </div>
            )}
            {gateState === "failed" && (
              <div className="flex items-center gap-2">
                <ShieldAlert className="h-4 w-4 text-red-400" />
                <span className="text-sm text-red-300">Approval gate failed or timed out</span>
              </div>
            )}
          </StageDetailCard>
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
