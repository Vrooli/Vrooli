import { useState, useCallback } from "react";
import { RefreshCw, Loader2, CheckCircle2, XCircle, AlertTriangle, ChevronDown, ChevronRight } from "lucide-react";
import { Button } from "./ui/button";
import { useStartAuditorCheck, useAuditorJobStatus } from "../lib/hooks";
import type { AgentContextItem } from "../lib/api";
import { AttachToAgentButton } from "./AgentTab";
import { ruleViolationContextItems, rulesSummaryContextItem } from "../lib/agentContext";
import { MutationErrorBanner, ServiceUnavailableBanner } from "./ScenarioReviewPanelShared";
import { SurfaceComparePanel } from "../features/baselines/SurfaceComparePanel";
import { SurfaceCaptureEmptyState } from "../features/baselines/SurfaceCaptureEmptyState";
import { useSurfaceBaselineModal } from "../features/baselines/useSurfaceBaselineModal";

export function RulesTab({
  scenarioSlug,
  repoId,
  auditorAvailable,
  agentManagerAvailable,
  onAttachToAgent,
  initialJobId,
  onJobIdChange,
  onOpenBaselines,
}: {
  scenarioSlug: string;
  repoId?: string | null;
  auditorAvailable: boolean;
  agentManagerAvailable?: boolean;
  onAttachToAgent?: (item: AgentContextItem) => void;
  initialJobId?: string | null;
  onJobIdChange?: (id: string | null) => void;
  onOpenBaselines?: () => void;
}) {
  const startCheck = useStartAuditorCheck(repoId);
  const [jobId, setJobIdInternal] = useState<string | null>(initialJobId ?? null);
  const setJobId = useCallback((id: string | null) => {
    setJobIdInternal(id);
    onJobIdChange?.(id);
  }, [onJobIdChange]);
  const jobStatus = useAuditorJobStatus(jobId, repoId);
  const [expandedViolation, setExpandedViolation] = useState<string | null>(null);
  const { openCaptureBaseline, baselineModal } = useSurfaceBaselineModal(scenarioSlug, repoId);
  const openBaselines = onOpenBaselines ?? (() => {});

  if (!auditorAvailable) {
    return <ServiceUnavailableBanner name="Scenario Auditor" message="Start the scenario-auditor scenario to view standards compliance and rule violations" />;
  }

  const handleRunCheck = () => {
    startCheck.mutate({ scenarioName: scenarioSlug }, {
      onSuccess: (data) => setJobId(data.job_id),
    });
  };

  const status = jobStatus.data?.status;
  const isRunning = status === "running" || status === "pending" || startCheck.isPending;
  const isCompleted = status === "completed";
  const isFailed = status === "failed";
  const result = jobStatus.data?.result;
  const violations = result?.violations ?? [];
  const summary = result?.summary;

  // No job started yet
  if (!jobId && !startCheck.data) {
    return (
      <div className="space-y-4">
        {baselineModal}
        <SurfaceCaptureEmptyState
          surface="rules"
          hasService={auditorAvailable}
          onCaptureLoose={handleRunCheck}
          onCaptureBaseline={openCaptureBaseline}
          captureLabel="Run check"
          isCapturing={startCheck.isPending}
        />
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <MutationErrorBanner error={startCheck.error} onDismiss={() => startCheck.reset()} />
      {baselineModal}
      <SurfaceComparePanel
        scenario={scenarioSlug}
        surface="rules"
        repoId={repoId}
        onOpenBaselines={openBaselines}
        onCaptureBaseline={openCaptureBaseline}
        viewingLabel={isCompleted ? `${violations.length} violation${violations.length !== 1 ? "s" : ""}` : "latest check"}
      />

      {/* Progress / summary bar */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3 text-xs">
          {isRunning && (
            <span className="flex items-center gap-1 text-blue-400">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              {jobStatus.data?.message || "Running..."}
            </span>
          )}
          {isCompleted && (
            <>
              {violations.length === 0 ? (
                <span className="flex items-center gap-1 text-emerald-400">
                  <CheckCircle2 className="h-3.5 w-3.5" /> No violations
                </span>
              ) : (
                <span className="flex items-center gap-1 text-red-400">
                  <XCircle className="h-3.5 w-3.5" /> {violations.length} violation{violations.length !== 1 ? "s" : ""}
                </span>
              )}
              {result?.files_scanned != null && (
                <span className="text-slate-500">{result.files_scanned} files scanned</span>
              )}
            </>
          )}
          {isFailed && (
            <span className="flex items-center gap-1 text-red-400">
              <AlertTriangle className="h-3.5 w-3.5" /> {jobStatus.data?.error || "Check failed"}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          {agentManagerAvailable && onAttachToAgent && violations.length > 0 && (
            <AttachToAgentButton onClick={() => onAttachToAgent(rulesSummaryContextItem(violations, summary, scenarioSlug))} />
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={handleRunCheck}
            disabled={isRunning}
            className="h-7 text-xs gap-1"
          >
            {isRunning ? (
              <Loader2 className="h-3 w-3 animate-spin" />
            ) : (
              <RefreshCw className="h-3 w-3" />
            )}
            {isRunning ? "Running..." : "Re-run"}
          </Button>
        </div>
      </div>

      {/* Progress bar while running */}
      {isRunning && jobStatus.data && jobStatus.data.total_files > 0 && (
        <div className="space-y-1">
          <div className="h-1.5 w-full bg-slate-800 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 rounded-full transition-all duration-300"
              style={{ width: `${Math.round((jobStatus.data.processed_files / jobStatus.data.total_files) * 100)}%` }}
            />
          </div>
          <p className="text-[11px] text-slate-500">
            {jobStatus.data.processed_files}/{jobStatus.data.total_files} files
            {jobStatus.data.current_file && ` · ${jobStatus.data.current_file}`}
          </p>
        </div>
      )}

      {/* Violations list */}
      {isCompleted && violations.length > 0 && (
        <div className="space-y-1">
          {violations.map((v, i) => {
            const key = v.id || `${v.type}-${i}`;
            const isExpanded = expandedViolation === key;
            return (
              <div key={key} className={`rounded border ${
                v.severity === "high" ? "border-red-900/30 bg-red-950/20" :
                v.severity === "medium" ? "border-amber-900/30 bg-amber-950/20" :
                "border-slate-800/30 bg-slate-900/20"
              }`}>
                <div className="flex items-center">
                  <button
                    type="button"
                    onClick={() => setExpandedViolation(isExpanded ? null : key)}
                    className="flex-1 flex items-center gap-2 px-3 py-2 text-xs cursor-pointer hover:bg-slate-800/20"
                  >
                    {isExpanded ? (
                      <ChevronDown className="h-3 w-3 text-slate-500" />
                    ) : (
                      <ChevronRight className="h-3 w-3 text-slate-500" />
                    )}
                    <div className={`h-1.5 w-1.5 rounded-full shrink-0 ${
                      v.severity === "high" ? "bg-red-500" :
                      v.severity === "medium" ? "bg-amber-500" :
                      v.severity === "low" ? "bg-yellow-500" : "bg-blue-500"
                    }`} />
                    <span className="text-slate-200 font-medium truncate">{v.title}</span>
                    <span className="text-slate-600 shrink-0">{v.type}</span>
                    {v.source && (
                      <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-slate-800 text-slate-400 shrink-0">{v.source}</span>
                    )}
                  </button>
                  {agentManagerAvailable && onAttachToAgent && (
                    <div className="pr-2">
                      <AttachToAgentButton onClick={() => {
                        const items = ruleViolationContextItems([v], scenarioSlug);
                        if (items[0]) onAttachToAgent(items[0]);
                      }} />
                    </div>
                  )}
                </div>
                {isExpanded && (
                  <div className="px-3 pb-3 pt-1 border-t border-slate-800/30 space-y-2 text-[11px]">
                    {v.description && <p className="text-slate-300">{v.description}</p>}
                    {v.file_path && (
                      <div className="text-slate-400">
                        <span className="text-slate-500">File:</span>{" "}
                        <code className="text-slate-300">{v.file_path}{v.line_number ? `:${v.line_number}` : ""}</code>
                      </div>
                    )}
                    {v.code_snippet && (
                      <pre className="p-2 rounded bg-slate-900 border border-slate-800 text-slate-300 overflow-x-auto text-[10px]">{v.code_snippet}</pre>
                    )}
                    {v.recommendation && (
                      <div className="text-slate-400">
                        <span className="text-slate-500">Recommendation:</span> {v.recommendation}
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Completed with no violations */}
      {isCompleted && violations.length === 0 && (
        <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-emerald-400" />
            <span className="text-xs text-emerald-300 font-medium">All rules passed — no violations found</span>
          </div>
        </div>
      )}
    </div>
  );
}
