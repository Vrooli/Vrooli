/**
 * BacklogScenariosPanel
 *
 * Shows target scenarios for a backlog item with post-run status badges
 * and the ability to trigger post-run checks.
 *
 * Extracted from BacklogDetailsPage to reduce file size.
 */

import { CheckCircle2, FolderOpen, RefreshCw } from "lucide-react";
import { Card } from "../ui/card";
import { Button } from "../ui/button";
import { PostRunStatusBadge } from "../execution/post-run-status-badge";
import { executionService } from "../../services";
import type { ExecutionRecord } from "../../types";

export interface BacklogScenariosPanelProps {
  targetScenarios: string[];
  executionHistory: ExecutionRecord[] | undefined;
  onSelectScenario: (name: string) => void;
}

export function BacklogScenariosPanel({
  targetScenarios,
  executionHistory,
  onSelectScenario,
}: BacklogScenariosPanelProps) {
  if (targetScenarios.length === 0) return null;

  const latestExec = executionHistory?.[0];

  const renderPostRunStatus = () => {
    if (latestExec?.finalization) {
      return (
        <div className="border-t border-slate-800 pt-2">
          <PostRunStatusBadge
            execution={latestExec}
            onRunChecks={async () => {
              try {
                await executionService.triggerReview(latestExec.executionId);
              } catch {
                // Will be visible on next query refetch
              }
            }}
          />
        </div>
      );
    }
    if (latestExec?.status === "validating") {
      return (
        <div className="border-t border-slate-800 pt-2">
          <PostRunStatusBadge
            execution={{
              ...latestExec,
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
    if (!latestExec) {
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
              await executionService.triggerReview(latestExec.executionId);
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
        {renderPostRunStatus()}
      </div>
    </Card>
  );
}
