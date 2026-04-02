/**
 * ScenarioReviewResults
 *
 * Displays target scenario chips and post-run review status in the Output tab.
 * Extracted from BacklogScenariosPanel — the Info tab version now only shows
 * scenario chips without review results.
 */

import { CheckCircle2, FolderOpen, RefreshCw } from "lucide-react";
import { Button } from "../ui/button";
import { PostRunStatusBadge } from "../execution/post-run-status-badge";
import { DetailSection } from "../detail/DetailSection";
import { executionService } from "../../services";
import { resolvePostRunExecution } from "../../lib";
import type { ExecutionRecord } from "../../types";

export interface ScenarioReviewResultsProps {
  /** Latest execution record with potential finalization data. */
  latestExecution: ExecutionRecord | undefined;
  /** Target scenario names. */
  targetScenarios: string[];
  /** Navigate to a scenario. */
  onSelectScenario: (name: string) => void;
}

export function ScenarioReviewResults({
  latestExecution,
  targetScenarios,
  onSelectScenario,
}: ScenarioReviewResultsProps) {
  if (targetScenarios.length === 0) return null;

  const renderPostRunStatus = () => {
    if (latestExecution) {
      const resolved = resolvePostRunExecution(latestExecution);
      if (resolved) {
        return (
          <div className="border-t border-slate-800 pt-2">
            <PostRunStatusBadge
              execution={resolved}
              onRunChecks={async () => {
                try {
                  await executionService.triggerReview(latestExecution.executionId);
                } catch {
                  // Will be visible on next query refetch
                }
              }}
            />
          </div>
        );
      }
      // Completed/failed execution with no finalization data — offer to run it.
      return (
        <div className="space-y-2 border-t border-slate-800 pt-2">
          <div className="flex items-center gap-1.5">
            <CheckCircle2 className="h-3.5 w-3.5 text-slate-500" />
            <span className="text-xs text-slate-400">No post-run checks yet</span>
          </div>
          <Button
            size="sm"
            variant="outline"
            className="w-full"
            onClick={async () => {
              try {
                await executionService.triggerReview(latestExecution.executionId);
              } catch {
                // Error handled by query refetch showing updated state
              }
            }}
          >
            <RefreshCw className="mr-1.5 h-3 w-3" />
            Run Post-Run Checks
          </Button>
        </div>
      );
    }
    return (
      <div className="flex items-center gap-1.5 border-t border-slate-800 pt-2">
        <CheckCircle2 className="h-3.5 w-3.5 text-slate-500" />
        <span className="text-xs text-slate-400">Post-run checks will run after execution</span>
      </div>
    );
  };

  return (
    <DetailSection title="Scenario Reviews" icon={FolderOpen}>
      <div className="space-y-3">
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
        {renderPostRunStatus()}
      </div>
    </DetailSection>
  );
}
