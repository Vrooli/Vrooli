import { useState } from "react";
import { Loader2, Play, CheckCircle2, XCircle, AlertTriangle, ChevronDown, ChevronRight } from "lucide-react";
import { Button } from "./ui/button";
import { useTestExecutions, useTriggerTestExecution } from "../lib/hooks";
import type { TestExecutionResult, TestPhaseResult, AgentContextItem } from "../lib/api";
import { AttachToAgentButton } from "./AgentTab";
import { testFailureContextItems } from "../lib/agentContext";
import { MutationErrorBanner, ServiceUnavailableBanner, formatDuration } from "./ScenarioReviewPanelShared";

export function TestsTab({
  scenarioSlug,
  repoId,
  testGenieAvailable,
  agentManagerAvailable,
  onAttachToAgent,
}: {
  scenarioSlug: string;
  repoId?: string | null;
  testGenieAvailable: boolean;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
}) {
  const testExecutions = useTestExecutions(scenarioSlug, testGenieAvailable, repoId);
  const triggerTest = useTriggerTestExecution(repoId);
  const [expandedPhase, setExpandedPhase] = useState<string | null>(null);

  if (!testGenieAvailable) {
    return <ServiceUnavailableBanner name="Test Genie" message="Start the test-genie scenario to run automated tests" />;
  }

  const isRunning = triggerTest.isPending;
  const executions = testExecutions.data?.items ?? [];
  const latest = executions[0] as TestExecutionResult | undefined;
  const history = executions.slice(1, 6);

  return (
    <div className="space-y-4">
      <MutationErrorBanner error={triggerTest.error} onDismiss={() => triggerTest.reset()} />
      {/* Run Tests button */}
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-medium text-slate-400">Test Execution</h3>
        <Button
          variant="outline"
          size="sm"
          onClick={() => triggerTest.mutate({ scenarioName: scenarioSlug })}
          disabled={isRunning}
          className="h-7 text-xs gap-1"
        >
          {isRunning ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <Play className="h-3 w-3" />
          )}
          Run Tests
        </Button>
      </div>

      {/* In-progress banner */}
      {isRunning && (
        <div className="flex items-center gap-2 px-3 py-2 bg-blue-950/50 border border-blue-900/50 rounded-lg text-blue-300 text-xs">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Running tests...
        </div>
      )}

      {/* Loading state */}
      {testExecutions.isLoading ? (
        <div className="space-y-3">
          <div className="h-24 animate-pulse bg-slate-800 rounded" />
          <div className="h-16 animate-pulse bg-slate-800 rounded" />
        </div>
      ) : !latest ? (
        <div className="flex flex-col items-center justify-center py-8 text-slate-500">
          <p className="text-sm">No test results yet</p>
          <p className="text-xs mt-1">Run tests to see results here</p>
        </div>
      ) : (
        <>
          {/* Latest execution card */}
          <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4 space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                {latest.success ? (
                  <CheckCircle2 className="h-4 w-4 text-emerald-400" />
                ) : (
                  <XCircle className="h-4 w-4 text-red-400" />
                )}
                <span className={`text-sm font-medium ${latest.success ? "text-emerald-300" : "text-red-300"}`}>
                  {latest.success ? "All Passed" : "Failed"}
                </span>
              </div>
              <span className="text-[11px] text-slate-500">
                {latest.completedAt ? new Date(latest.completedAt).toLocaleString() : latest.startedAt}
              </span>
            </div>
            <div className="flex gap-4 text-xs text-slate-400">
              <span>{latest.phaseSummary.total} total</span>
              <span className="text-emerald-400">{latest.phaseSummary.passed} passed</span>
              {latest.phaseSummary.failed > 0 && (
                <span className="text-red-400">{latest.phaseSummary.failed} failed</span>
              )}
              <span>{formatDuration(latest.phaseSummary.durationSeconds)}</span>
            </div>
            {latest.preset && (
              <div className="text-[11px] text-slate-500">Preset: {latest.preset}</div>
            )}
            {latest.warnings && latest.warnings.length > 0 && (
              <div className="space-y-1">
                {latest.warnings.map((w, i) => (
                  <div key={i} className="flex items-start gap-1.5 text-[11px] text-amber-400">
                    <AlertTriangle className="h-3 w-3 mt-0.5 shrink-0" />
                    {w}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Phase list */}
          <div className="space-y-1">
            <h4 className="text-xs font-medium text-slate-400 mb-2">Phases</h4>
            {latest.phases.map((phase) => (
              <div key={phase.name} className="flex items-start gap-1">
                <div className="flex-1 min-w-0">
                  <PhaseRow
                    phase={phase}
                    expanded={expandedPhase === phase.name}
                    onToggle={() => setExpandedPhase(expandedPhase === phase.name ? null : phase.name)}
                  />
                </div>
                {agentManagerAvailable && phase.status === "failed" && onAttachToAgent && (
                  <div className="mt-2 shrink-0">
                    <AttachToAgentButton onClick={() => { const items = testFailureContextItems([phase], scenarioSlug); if (items[0]) onAttachToAgent(items[0]); }} />
                  </div>
                )}
              </div>
            ))}
          </div>

          {/* History */}
          {history.length > 0 && (
            <div>
              <h4 className="text-xs font-medium text-slate-400 mb-2">Recent History</h4>
              <div className="space-y-1">
                {history.map((exec) => (
                  <div
                    key={exec.executionId}
                    className="flex items-center justify-between px-3 py-2 rounded bg-slate-900/50 border border-slate-800/50"
                  >
                    <div className="flex items-center gap-2">
                      {exec.success ? (
                        <div className="h-2 w-2 rounded-full bg-emerald-500" />
                      ) : (
                        <div className="h-2 w-2 rounded-full bg-red-500" />
                      )}
                      <span className="text-[11px] text-slate-400">
                        {exec.completedAt ? new Date(exec.completedAt).toLocaleString() : exec.startedAt}
                      </span>
                    </div>
                    <div className="flex gap-3 text-[11px] text-slate-500">
                      {exec.preset && <span>{exec.preset}</span>}
                      <span>{formatDuration(exec.phaseSummary.durationSeconds)}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}

// ============================================================================
// Phase Row
// ============================================================================

function PhaseRow({
  phase,
  expanded,
  onToggle,
}: {
  phase: TestPhaseResult;
  expanded: boolean;
  onToggle: () => void;
}) {
  const hasDetails = Boolean(phase.error || phase.remediation || (phase.observations && phase.observations.length > 0));

  return (
    <div className="rounded border border-slate-800/50 bg-slate-900/30">
      <button
        type="button"
        onClick={hasDetails ? onToggle : undefined}
        className={`w-full flex items-center justify-between px-3 py-2 text-xs ${hasDetails ? "cursor-pointer hover:bg-slate-800/30" : "cursor-default"}`}
      >
        <div className="flex items-center gap-2">
          {hasDetails ? (
            expanded ? <ChevronDown className="h-3 w-3 text-slate-500" /> : <ChevronRight className="h-3 w-3 text-slate-500" />
          ) : (
            <div className="w-3" />
          )}
          <div className={`h-2 w-2 rounded-full ${phase.status === "passed" ? "bg-emerald-500" : "bg-red-500"}`} />
          <span className="text-slate-200">{phase.name}</span>
        </div>
        <span className="text-slate-500">{formatDuration(phase.durationSeconds)}</span>
      </button>

      {expanded && hasDetails && (
        <div className="px-3 pb-3 pt-1 border-t border-slate-800/30 space-y-2">
          {phase.error && (
            <div className="text-[11px] text-red-400 bg-red-950/30 rounded px-2 py-1.5">
              {phase.error}
            </div>
          )}
          {phase.classification && (
            <div className="text-[11px] text-slate-400">
              <span className="text-slate-500">Classification:</span> {phase.classification}
            </div>
          )}
          {phase.remediation && (
            <div className="text-[11px] text-amber-300 bg-amber-950/20 rounded px-2 py-1.5">
              {phase.remediation}
            </div>
          )}
          {phase.observations && phase.observations.length > 0 && (
            <div className="space-y-1">
              {phase.observations.map((obs, i) => (
                <div key={i} className="text-[11px] text-slate-400 flex gap-1.5">
                  {obs.icon && <span>{obs.icon}</span>}
                  {obs.prefix && (
                    <span className={`font-medium ${
                      obs.prefix === "ERROR" ? "text-red-400" :
                      obs.prefix === "WARNING" ? "text-amber-400" :
                      obs.prefix === "SUCCESS" ? "text-emerald-400" : "text-slate-400"
                    }`}>
                      {obs.prefix}:
                    </span>
                  )}
                  <span>{obs.text}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

