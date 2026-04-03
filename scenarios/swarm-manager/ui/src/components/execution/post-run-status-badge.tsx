import { useState } from "react";
import { AlertTriangle, Check, ChevronDown, ChevronUp, Loader2 } from "lucide-react";
import { cn } from "../../lib";
import type { ExecutionRecord, Finalization, FinalizationPhase, ScenarioFinalization } from "../../types";

const CLASSIFICATION_LABELS: Record<string, string> = {
  ready: "Post-run checks passed",
  ready_with_notes: "Passed with notes",
  needs_work: "Needs fixup",
  not_assessable: "Checks inconclusive",
  skipped: "Checks skipped",
};

function phaseLabel(phase?: FinalizationPhase | string): string {
  switch (phase) {
    case "scope_detection":
      return "Resolving affected scenarios";
    case "restarting":
      return "Restarting scenarios";
    case "health_check":
      return "Waiting for health checks";
    case "reviewing":
      return "Running scenario reviews";
    case "evidence_gathering":
      return "Gathering evidence";
    default:
      return "Running post-run checks";
  }
}

function classificationTone(finalization?: Finalization): string {
  if (!finalization) {
    return "border-slate-600 bg-slate-800/50 text-slate-300";
  }
  switch (finalization.aggregateClassification) {
    case "ready":
      return "border-emerald-500/30 bg-emerald-500/10 text-emerald-300";
    case "ready_with_notes":
      return "border-amber-500/30 bg-amber-500/10 text-amber-300";
    case "needs_work":
      return "border-red-500/30 bg-red-500/10 text-red-300";
    case "skipped":
      return "border-slate-600 bg-slate-800/50 text-slate-400";
    default:
      return "border-slate-600 bg-slate-800/50 text-slate-300";
  }
}

function ScenarioSummary({ scenario }: { scenario: ScenarioFinalization }) {
  return (
    <div className="rounded-md border border-white/5 bg-slate-900/50 px-2.5 py-2">
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs font-medium text-slate-200">{scenario.scenarioName}</span>
        <span className="text-[11px] text-slate-500">
          restart {scenario.restart.status || "pending"} · health {scenario.health.status || "pending"} · review {scenario.review.status || "pending"}
        </span>
      </div>
      {scenario.changedPaths.length > 0 && (
        <p className="mt-1 text-[11px] text-slate-500">
          {scenario.changedPaths.length} changed path{scenario.changedPaths.length === 1 ? "" : "s"}
        </p>
      )}
      {scenario.health.details && scenario.health.status !== "completed" && (
        <p className="mt-1 text-[11px] text-red-300">{scenario.health.details}</p>
      )}
      {scenario.review.skipReason && (
        <p className="mt-1 text-[11px] text-amber-300">{scenario.review.skipReason}</p>
      )}
      {scenario.review.result?.summary && (
        <p className="mt-1 text-[11px] text-slate-400">{scenario.review.result.summary}</p>
      )}
    </div>
  );
}

export interface PostRunStatusBadgeProps {
  execution: ExecutionRecord;
}

export function PostRunStatusBadge({ execution }: PostRunStatusBadgeProps) {
  const [expanded, setExpanded] = useState(false);
  const finalization = execution.finalization;
  const hasDetails = Boolean(finalization?.aggregateSummary || finalization?.warnings.length || finalization?.scenarios.length);

  if (!finalization) {
    return null;
  }

  if (execution.status === "validating" || finalization.status === "running" || finalization.status === "pending") {
    return (
      <div className="flex items-center gap-2 rounded-md border border-indigo-500/30 bg-indigo-500/10 px-2 py-1.5 text-xs text-indigo-300" data-testid="post-run-validating-indicator">
        <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin" />
        <span>{phaseLabel(finalization.phase)}</span>
      </div>
    );
  }

  return (
    <div className="space-y-1.5" data-testid="post-run-status-badge">
      <button
        type="button"
        onClick={() => {
          if (hasDetails) {
            setExpanded((prev) => !prev);
          }
        }}
        className={cn(
          "flex w-full items-center gap-2 rounded-md border px-2 py-1.5 text-xs transition-colors",
          classificationTone(finalization),
          hasDetails && "cursor-pointer hover:border-white/20",
        )}
      >
        {finalization.aggregateClassification === "ready" ? <Check className="h-3.5 w-3.5 shrink-0" /> : <AlertTriangle className="h-3.5 w-3.5 shrink-0" />}
        <span className="flex-1 text-left">
          {CLASSIFICATION_LABELS[finalization.aggregateClassification] ?? CLASSIFICATION_LABELS.not_assessable}
        </span>
        {hasDetails && (expanded ? <ChevronUp className="h-3 w-3 shrink-0 text-slate-500" /> : <ChevronDown className="h-3 w-3 shrink-0 text-slate-500" />)}
      </button>

      {expanded && (
        <div className="space-y-2 rounded-md bg-slate-800/50 px-2.5 py-2">
          {finalization.aggregateSummary && (
            <p className="text-[11px] leading-relaxed text-slate-300">{finalization.aggregateSummary}</p>
          )}
          {finalization.warnings.map((warning) => (
            <p key={`${warning.code}-${warning.createdAt}-${warning.scenarioName ?? ""}`} className="text-[11px] text-amber-300">
              {warning.scenarioName ? `${warning.scenarioName}: ` : ""}{warning.message}
            </p>
          ))}
          {finalization.scenarios.map((scenario) => (
            <ScenarioSummary key={scenario.scenarioName} scenario={scenario} />
          ))}
        </div>
      )}
    </div>
  );
}
