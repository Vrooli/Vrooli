/**
 * ExecutionChangesTab — Shows changed files from the execution's sandbox,
 * grouped by affected scenario.
 */

import { ChevronDown, ChevronRight, FileCode2, FolderOpen } from "lucide-react";
import { useState } from "react";
import { DetailSection } from "../detail/DetailSection";
import { selectors } from "../../consts/selectors";
import type { Finalization } from "../../types";

export interface ExecutionChangesTabProps {
  finalization: Finalization | undefined;
  isActive: boolean;
}

export function ExecutionChangesTab({ finalization, isActive }: ExecutionChangesTabProps) {
  if (isActive) {
    return (
      <DetailSection title="Changes" hideDivider>
        <div className="py-6 text-center" data-testid={selectors.executionDetails.changesEmpty}>
          <p className="text-sm text-slate-400">Changes will be available after the execution completes.</p>
        </div>
      </DetailSection>
    );
  }

  const scenarios = finalization?.scenarios ?? [];
  const hasSandbox = finalization?.scopeSource?.includes("sandbox_diff") ?? false;
  const hasChangedPaths = scenarios.some((s) => s.changedPaths.length > 0);

  if (!finalization || (!hasSandbox && !hasChangedPaths)) {
    return (
      <DetailSection title="Changes" hideDivider>
        <div className="py-6 text-center" data-testid={selectors.executionDetails.changesEmpty}>
          <FolderOpen className="mx-auto mb-2 h-8 w-8 text-slate-600" />
          <p className="text-sm text-slate-400">
            No sandbox changes available for this execution.
          </p>
          <p className="mt-1 text-xs text-slate-500">
            Changes were applied directly without a sandbox environment.
          </p>
        </div>
      </DetailSection>
    );
  }

  return (
    <DetailSection title="Changes" hideDivider>
      <div className="space-y-3" data-testid={selectors.executionDetails.changesFileList}>
        {scenarios.map((scenario) => (
          <ScenarioChanges
            key={scenario.scenarioName}
            scenarioName={scenario.scenarioName}
            changedPaths={scenario.changedPaths}
          />
        ))}
      </div>
    </DetailSection>
  );
}

// --- Internal sub-component ---

interface ScenarioChangesProps {
  scenarioName: string;
  changedPaths: string[];
}

function ScenarioChanges({ scenarioName, changedPaths }: ScenarioChangesProps) {
  const [expanded, setExpanded] = useState(true);

  if (changedPaths.length === 0) return null;

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/40">
      <button
        type="button"
        className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-slate-800/30"
        onClick={() => setExpanded((v) => !v)}
      >
        {expanded ? (
          <ChevronDown className="h-3.5 w-3.5 text-slate-500" />
        ) : (
          <ChevronRight className="h-3.5 w-3.5 text-slate-500" />
        )}
        <span className="font-medium text-slate-200">{scenarioName}</span>
        <span className="ml-auto rounded bg-slate-700/60 px-1.5 py-0.5 text-xs text-slate-400">
          {changedPaths.length} {changedPaths.length === 1 ? "file" : "files"}
        </span>
      </button>
      {expanded && (
        <div className="border-t border-slate-800 px-3 py-2">
          <ul className="space-y-0.5">
            {changedPaths.map((path) => (
              <li key={path} className="flex items-center gap-1.5 text-xs text-slate-400 font-mono">
                <FileCode2 className="h-3 w-3 shrink-0 text-slate-600" />
                {path}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
