/**
 * Smoke Test section - displays smoke test stage status and results from the pipeline store.
 */

import { forwardRef, useMemo } from "react";
import { TestTube, CheckCircle2, XCircle, Monitor, FileText } from "lucide-react";
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

interface SmokeTestSectionProps {
  scenarioName: string;
}

export const SmokeTestSection = forwardRef<HTMLDivElement, SmokeTestSectionProps>(
  ({ scenarioName }, ref) => {
    const smokeTestResult = usePipelineStore((s) => s.smokeTestResult);
    const buildResult = usePipelineStore((s) => s.buildResult);
    const stageStatus = usePipelineStore(selectStageStatus("smoketest"));

    const hasResult = Boolean(smokeTestResult);
    const hasBuildArtifacts = Object.keys(buildResult?.artifacts ?? {}).length > 0;
    const { status: testStatus, platform, artifact_path: artifactPath, error, logs = [], telemetry_uploaded: telemetryUploaded } = smokeTestResult ?? {};
    const testPassed = testStatus === "completed" || testStatus === "passed";

    const statusDisplay = useMemo(() => {
      if (stageStatus === "completed") return { ...STATUS_CONFIG.completed, label: testPassed ? "Passed" : "Completed" };
      if (stageStatus === "running") return { ...STATUS_CONFIG.running, label: "Testing" };
      return STATUS_CONFIG[stageStatus as keyof typeof STATUS_CONFIG] ?? STATUS_CONFIG.pending;
    }, [stageStatus, testPassed]);

    const getDescription = () => {
      if (hasResult) return `Tested on ${platform ?? "unknown platform"}`;
      return hasBuildArtifacts ? "Ready to test" : "Waiting for build artifacts";
    };

    return (
      <SectionCard
        ref={ref}
        sectionId="smoketest"
        title="Smoke Test"
        subtitle="Validate built artifacts"
        variant="pipeline"
        collapsible={true}
        contentClassName="space-y-4"
      >
        <StageAbout title="About smoke testing">
          <p>The smoke test stage verifies that built installers launch correctly on each platform.</p>
        </StageAbout>

        <StageStatusOverview icon={TestTube} title="Smoke Test Status" description={getDescription()} statusDisplay={statusDisplay} />

        {hasResult && (
          <div className="space-y-3">
            <div className="grid gap-2 sm:grid-cols-2">
              {platform && (
                <div className="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-950/50 p-3">
                  <Monitor className="h-4 w-4 text-slate-400" />
                  <div>
                    <p className="text-xs text-slate-500">Platform</p>
                    <p className="text-sm text-slate-300">{platform}</p>
                  </div>
                </div>
              )}
              {telemetryUploaded !== undefined && (
                <div className="flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-950/50 p-3">
                  {telemetryUploaded ? <CheckCircle2 className="h-4 w-4 text-green-400" /> : <XCircle className="h-4 w-4 text-slate-400" />}
                  <div>
                    <p className="text-xs text-slate-500">Telemetry</p>
                    <p className="text-sm text-slate-300">{telemetryUploaded ? "Uploaded" : "Not uploaded"}</p>
                  </div>
                </div>
              )}
            </div>

            {artifactPath && (
              <StageDetailCard icon={FileText} label="Tested Artifact">
                <code className="text-xs text-slate-300 font-mono break-all">{artifactPath}</code>
              </StageDetailCard>
            )}

            {logs.length > 0 && (
              <StageDetailCard icon={FileText} label={`Test Logs (${logs.length} lines)`}>
                <div className="max-h-32 overflow-y-auto">
                  <pre className="text-xs text-slate-400 font-mono whitespace-pre-wrap">{logs.slice(-10).join("\n")}</pre>
                </div>
              </StageDetailCard>
            )}

            {error && <StageError stageName="Smoke test"><strong>Error:</strong> {error}</StageError>}
          </div>
        )}

        {!hasResult && stageStatus === "pending" && (
          <StagePlaceholder
            scenarioName={scenarioName}
            withScenarioText={hasBuildArtifacts ? "Smoke tests will run automatically after build completes." : "Smoke testing will be available after building installers."}
            withoutScenarioText="Select a scenario to enable smoke testing."
          />
        )}

        {stageStatus === "failed" && !hasResult && <StageError stageName="Smoke test" />}
      </SectionCard>
    );
  }
);

SmokeTestSection.displayName = "SmokeTestSection";
