/**
 * ScenarioReviewResults
 *
 * Displays target scenario chips and post-run review status in the Output tab.
 * Extracted from BacklogScenariosPanel — the Info tab version now only shows
 * scenario chips without review results.
 */

import { CheckCircle2, FolderOpen, RefreshCw } from "lucide-react";
import { Card } from "../ui/card";
import { Button } from "../ui/button";
import { PostRunStatusBadge } from "../execution/post-run-status-badge";
import { executionService } from "../../services";
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
    if (latestExecution?.finalization) {
      return (
        <div className="border-t border-slate-800 pt-2">
          <PostRunStatusBadge
            execution={latestExecution}
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
    if (latestExecution?.status === "validating") {
      return (
        <div className="border-t border-slate-800 pt-2">
          <PostRunStatusBadge
            execution={{
              ...latestExecution,
              finalization: {
                eligible: true,
                status: "running",
                phase: "scope_detection",
                scopeSource: "none",
                warnings: [],
                affectedScenarios: [],
                aggregateClassification: "not_assessable",
                scenarios: [],
              },
            }}
          />
        </div>
      );
    }
    if (!latestExecution) {
      return (
        <div className="flex items-center gap-1.5 border-t border-slate-800 pt-2">
          <CheckCircle2 className="h-3.5 w-3.5 text-slate-500" />
          <span className="text-xs text-slate-400">Post-run checks will run after execution</span>
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
  };

  return (
    <Card padding="sm" className="rounded-lg border-slate-700/60 bg-slate-900/45">
      <div className="space-y-3">
        <div className="flex items-center gap-2 border-b border-slate-800 pb-2">
          <FolderOpen className="h-4 w-4 text-slate-400" />
          <h2 className="text-base font-semibold text-slate-100">Scenario Reviews</h2>
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
        {renderPostRunStatus()}
      </div>
    </Card>
  );
}
