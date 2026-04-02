/**
 * BacklogScenariosPanel
 *
 * Shows target scenarios for a backlog item as clickable chips.
 * Post-run review results are now displayed in the Output tab's
 * ScenarioReviewResults component.
 */

import { FolderOpen } from "lucide-react";
import { DetailSection } from "../detail/DetailSection";

export interface BacklogScenariosPanelProps {
  targetScenarios: string[];
  onSelectScenario: (name: string) => void;
}

export function BacklogScenariosPanel({
  targetScenarios,
  onSelectScenario,
}: BacklogScenariosPanelProps) {
  if (targetScenarios.length === 0) return null;

  return (
    <DetailSection title="Target Scenarios" icon={FolderOpen}>
      <div className="flex flex-wrap gap-1.5">
        {targetScenarios.map((scenarioName) => (
          <button
            key={scenarioName}
            type="button"
            onClick={() => onSelectScenario(scenarioName)}
            className="inline-flex items-center rounded-full bg-violet-500/15 px-2.5 py-1 text-xs font-medium text-violet-400 hover:bg-violet-500/25 transition-colors"
          >
            {scenarioName}
          </button>
        ))}
      </div>
    </DetailSection>
  );
}
