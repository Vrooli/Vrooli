/**
 * BacklogScenariosPanel
 *
 * Shows target scenarios for a backlog item as clickable chips.
 * Post-run review results are now displayed in the Output tab's
 * ScenarioReviewResults component.
 */

import { FolderOpen } from "lucide-react";
import { EntityLink } from "../ui/entity-link";
import { DetailSection } from "../detail/DetailSection";

export interface BacklogScenariosPanelProps {
  targetScenarios: string[];
}

export function BacklogScenariosPanel({
  targetScenarios,
}: BacklogScenariosPanelProps) {
  if (targetScenarios.length === 0) return null;

  return (
    <DetailSection title="Target Scenarios" icon={FolderOpen}>
      <div className="flex flex-wrap gap-1.5">
        {targetScenarios.map((scenarioName) => (
          <EntityLink
            key={scenarioName}
            entityType="scenario"
            name={scenarioName}
            label={scenarioName}
          />
        ))}
      </div>
    </DetailSection>
  );
}
