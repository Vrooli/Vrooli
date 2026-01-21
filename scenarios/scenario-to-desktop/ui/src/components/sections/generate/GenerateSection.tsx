/**
 * Generate section - displays generate stage status and results from the pipeline store.
 */

import { forwardRef } from "react";
import { Wand2, FolderOpen } from "lucide-react";
import {
  SectionCard,
  getStatusDisplay,
  StageStatusOverview,
  StageAbout,
  StageDetailCard,
  StagePlaceholder,
  StageError,
} from "../shared";
import { usePipelineStore, selectStageStatus } from "../../../store";

interface GenerateSectionProps {
  scenarioName: string;
}

export const GenerateSection = forwardRef<HTMLDivElement, GenerateSectionProps>(
  ({ scenarioName }, ref) => {
    const generateResult = usePipelineStore((s) => s.generateResult);
    const stageStatus = usePipelineStore(selectStageStatus("generate"));

    const hasResult = Boolean(generateResult);
    const desktopPath = generateResult?.desktop_path;
    const buildId = generateResult?.build_id;
    const statusDisplay = getStatusDisplay(stageStatus, { completed: "Generated", running: "Generating" });

    return (
      <SectionCard
        ref={ref}
        sectionId="generate"
        title="Generate"
        subtitle="Create desktop wrapper code"
        variant="pipeline"
        collapsible={true}
        contentClassName="space-y-4"
      >
        <StageAbout title="About generation">
          <p>
            The generate stage creates an Electron project scaffold stored in{" "}
            <code className="font-mono text-slate-200">platforms/electron</code>.
          </p>
        </StageAbout>

        <StageStatusOverview
          icon={Wand2}
          title="Generate Status"
          description={hasResult ? `Electron wrapper generated${buildId ? ` (Build: ${buildId.slice(0, 8)}...)` : ""}` : "Wrapper not yet generated"}
          statusDisplay={statusDisplay}
        />

        {hasResult && desktopPath && (
          <StageDetailCard icon={FolderOpen} label="Desktop Application Path">
            <code className="text-xs text-slate-300 font-mono break-all">{desktopPath}</code>
          </StageDetailCard>
        )}

        {!hasResult && stageStatus === "pending" && (
          <StagePlaceholder
            scenarioName={scenarioName}
            withScenarioText='Use the "Generate Desktop Application" button in the Configuration section above.'
          />
        )}

        {stageStatus === "failed" && <StageError stageName="Generate" />}
      </SectionCard>
    );
  }
);

GenerateSection.displayName = "GenerateSection";
