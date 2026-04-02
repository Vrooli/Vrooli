/**
 * BacklogScenariosPanel
 *
 * Shows target scenarios for a backlog item as clickable chips.
 * Post-run review results are now displayed in the Output tab's
 * ScenarioReviewResults component.
 */

import { FolderOpen } from "lucide-react";
import { Card } from "../ui/card";

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
    <Card padding="sm" className="rounded-lg border-slate-700/60 bg-slate-900/45">
      <div className="space-y-3">
        <div className="flex items-center gap-2 border-b border-slate-800 pb-2">
          <FolderOpen className="h-4 w-4 text-slate-400" />
          <h2 className="text-base font-semibold text-slate-100">Target Scenarios</h2>
        </div>
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
      </div>
    </Card>
  );
}
