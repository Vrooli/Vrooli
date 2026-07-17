/**
 * LatestExecutionSummary
 *
 * Persistent section at the top of the Output tab showing the active
 * run state or the empty state. Completed/failed execution status and
 * actions are now handled by ReviewStatusHeader in ReviewFlow.
 *
 * When the canonical workflow projection covers this item, the active run
 * carries an operation-provenance affordance and a canonical operation
 * history (ListExecutionHistory) renders below — coexisting with the legacy
 * poller-owned records until the Phase-8 migration completes. Sources are
 * labeled honestly: canonical entries carry the provenance badge; the
 * legacy-derived active row is marked "legacy record" when uncovered.
 */

import { ExternalLink, ShieldAlert, ShieldCheck, Square } from "lucide-react";
import { Button } from "../ui/button";
import { formatRelativeTime, provenanceFromExecutionSummary } from "../../lib";
import type { OperationProvenanceData } from "../../lib";
import { selectors } from "../../consts/selectors";
import { buildAgentRunUrl } from "../../services/external-links";
import { OperationProvenancePopover } from "../workflow/operation-provenance-popover";
import type { ExecutionRecord } from "../../types";
import type { WorkflowExecutionSummary } from "../../types/agent-operations";
import type { AgentActivityRecord } from "../../stores/agent-activities-store";

export interface LatestExecutionSummaryProps {
  /** Most recent execution, or undefined for empty state. */
  latestExecution: ExecutionRecord | undefined;
  /** Whether an agent run is actively executing work. */
  agentRunIsBusy: boolean;
  /** The latest agent activity record (for live status). */
  latestAgentActivity: AgentActivityRecord | null;
  /** Agent manager UI URL for external run links. */
  agentManagerUiUrl: string | null;
  /** Stop a running agent. */
  onStopRun: (runId: string) => void;
  /** Canonical operation matched to the active run (workflow projection). */
  runProvenance?: OperationProvenanceData | null;
  /** Canonical execution provenance history (newest first), when the workflow exists. */
  canonicalHistory?: WorkflowExecutionSummary[];
}

function CanonicalHistoryList({ history }: { history: WorkflowExecutionSummary[] }) {
  return (
    <div className="mt-3 space-y-1.5" data-testid="canonical-operation-history">
      <p className="text-[11px] font-medium uppercase tracking-wide text-slate-500">
        Operation history
      </p>
      {history.map((entry) => (
        <div
          key={entry.executionId}
          className="flex flex-wrap items-center gap-2 rounded-md border border-slate-800 bg-slate-900/50 px-2 py-1.5"
          data-testid="canonical-operation-history-entry"
        >
          <span className="font-mono text-xs text-slate-300">
            {entry.operation}
            {entry.operationVersion ? `@${entry.operationVersion}` : ""}
          </span>
          {entry.outcome && (
            <span className="rounded bg-slate-800 px-1.5 py-0.5 text-[10px] text-slate-400">
              {entry.outcome}
            </span>
          )}
          {entry.legacyImport ? (
            <span
              className="inline-flex items-center gap-1 text-[10px] text-slate-400"
              title="Imported pre-cutover execution record: no mode/prompt provenance existed for this run, so reproducibility does not apply"
            >
              legacy import
            </span>
          ) : (
            <span
              className={
                entry.reproducible
                  ? "inline-flex items-center gap-1 text-[10px] text-emerald-400"
                  : "inline-flex items-center gap-1 text-[10px] text-amber-300"
              }
              title={
                entry.reproducible
                  ? "Verified evidence: snapshot digests still reproduce the pinned provenance"
                  : "Snapshot digests no longer match the pinned provenance"
              }
            >
              {entry.reproducible ? (
                <ShieldCheck className="h-3 w-3" aria-hidden />
              ) : (
                <ShieldAlert className="h-3 w-3" aria-hidden />
              )}
              {entry.reproducible ? "verified" : "drift"}
            </span>
          )}
          <div className="ml-auto flex items-center gap-2">
            {entry.recordedAt && (
              <span className="text-[10px] text-slate-500">
                {formatRelativeTime(entry.recordedAt)}
              </span>
            )}
            <OperationProvenancePopover data={provenanceFromExecutionSummary(entry)} />
          </div>
        </div>
      ))}
    </div>
  );
}

export function LatestExecutionSummary({
  latestExecution,
  agentRunIsBusy,
  latestAgentActivity,
  agentManagerUiUrl,
  onStopRun,
  runProvenance,
  canonicalHistory,
}: LatestExecutionSummaryProps) {
  const testId = `${selectors.backlogDetails.outputTab}-latest-exec`;
  const runId = latestAgentActivity?.runId ?? latestExecution?.runId;
  const runUrl = buildAgentRunUrl(agentManagerUiUrl, runId);
  const historyBlock =
    canonicalHistory && canonicalHistory.length > 0 ? (
      <CanonicalHistoryList history={canonicalHistory} />
    ) : null;

  // Active run state — agent is currently running
  if (agentRunIsBusy && latestAgentActivity) {
    return (
      <div data-testid={testId}>
      <div className="rounded-lg border border-cyan-500/30 bg-cyan-500/10 p-3">
        <div className="flex items-center gap-2">
          <span className="relative flex h-2 w-2 shrink-0">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-cyan-400 opacity-75" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-cyan-500" />
          </span>
          <span className="text-xs font-medium capitalize text-cyan-200">
            {latestAgentActivity.status.replace("_", " ")}
          </span>
          {latestAgentActivity.purpose && (
            <span className="rounded bg-cyan-500/20 px-1.5 py-0.5 text-[11px] font-medium text-cyan-300">
              {latestAgentActivity.purpose.replace("_", " ")}
            </span>
          )}
          <span className="text-xs text-slate-400">
            {formatRelativeTime(latestAgentActivity.requestedAt)}
          </span>
          {runProvenance ? (
            <OperationProvenancePopover data={runProvenance} />
          ) : (
            <span
              className="rounded border border-slate-700/70 px-1.5 py-0.5 text-[10px] text-slate-500"
              title="This run is tracked by the legacy execution pipeline (no canonical operation record)."
              data-testid={`${testId}-legacy-badge`}
            >
              legacy record
            </span>
          )}
          <div className="ml-auto flex items-center gap-1.5">
            {runUrl && (
              <a
                href={runUrl}
                target="_blank"
                rel="noopener noreferrer"
                data-testid={`${testId}-run-link`}
              >
                <Button variant="outline" size="sm" className="h-7 px-2 text-xs">
                  <ExternalLink className="mr-1 h-3 w-3" />
                  View Run
                </Button>
              </a>
            )}
            <Button
              variant="outline"
              size="sm"
              className="h-7 px-2 text-xs"
              onClick={() => onStopRun(latestAgentActivity.runId ?? "")}
              disabled={latestAgentActivity.isStopping}
              data-testid={`${testId}-stop`}
            >
              <Square className="mr-1 h-3 w-3" />
              {latestAgentActivity.isStopping ? "Stopping..." : "Stop"}
            </Button>
          </div>
        </div>
      </div>
      {historyBlock}
      </div>
    );
  }

  // Empty state — no executions yet
  if (!latestExecution) {
    return (
      <div className="py-3" data-testid={testId}>
        {historyBlock ?? (
          <p className="text-sm text-slate-400">
            No executions yet. Queue or run the agent to see results here.
          </p>
        )}
      </div>
    );
  }

  // Completed/failed — status is shown by ReviewStatusHeader; the canonical
  // operation history (when present) still renders for inspectability.
  return historyBlock ? <div data-testid={testId}>{historyBlock}</div> : null;
}
