import { useState, useEffect } from "react";
import { AlertTriangle, Check, ChevronDown, ChevronUp, Info, Loader2 } from "lucide-react";
import { cn } from "../../lib";
import type { ExecutionRecord, Finalization, FinalizationPhase, FinalizationWarning } from "../../types";

// ---------------------------------------------------------------------------
// Finalization phase stepper constants
// ---------------------------------------------------------------------------

const FINALIZATION_PHASE_ORDER = [
  "scope_detection", "restarting", "reviewing", "evidence_gathering",
] as const;

const PHASE_SHORT_LABELS: Record<string, string> = {
  scope_detection: "Scope",
  restarting: "Restart & Health",
  reviewing: "Review",
  evidence_gathering: "Evidence",
};

function useElapsedTime(startedAt: string | undefined, isActive: boolean) {
  const [elapsed, setElapsed] = useState(0);
  useEffect(() => {
    if (!startedAt || !isActive) {
      setElapsed(0);
      return;
    }
    const start = new Date(startedAt).getTime();
    const tick = () => setElapsed(Math.floor((Date.now() - start) / 1000));
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [startedAt, isActive]);
  return elapsed;
}

function formatElapsed(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}m ${s}s`;
}

/** Warning codes that indicate the evidence phase was skipped — shown inline, not behind expand. */
const EVIDENCE_SKIP_CODES = new Set([
  "evidence_skipped_disabled",
  "evidence_skipped_policy_error",
]);

function isEvidenceSkipWarning(w: FinalizationWarning): boolean {
  return EVIDENCE_SKIP_CODES.has(w.code);
}

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
    case "health_check":
      return "Restarting & checking scenarios";
    case "reviewing":
      return "Running scenario reviews";
    case "evidence_gathering":
      return "Gathering evidence";
    default:
      return "Running post-run checks";
  }
}

/** Map backend phase to stepper index (health_check maps to restarting). */
function normalizePhaseForStepper(phase: string): (typeof FINALIZATION_PHASE_ORDER)[number] {
  if (phase === "health_check") return "restarting";
  return phase as (typeof FINALIZATION_PHASE_ORDER)[number];
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

function FinalizationProgress({ finalization }: { finalization: Finalization }) {
  const isActive = finalization.status === "running" || finalization.status === "pending";
  const elapsed = useElapsedTime(finalization.startedAt, isActive);
  const currentPhaseIdx = FINALIZATION_PHASE_ORDER.indexOf(
    normalizePhaseForStepper(finalization.phase ?? ""),
  );

  return (
    <div
      className="space-y-2 rounded-md border border-indigo-500/30 bg-indigo-500/10 px-3 py-2"
      data-testid="post-run-validating-indicator"
    >
      {/* Phase label + timer */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-xs text-indigo-300">
          <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin" />
          <span>{phaseLabel(finalization.phase)}</span>
        </div>
        {isActive && finalization.startedAt && (
          <span className="text-[11px] tabular-nums text-indigo-400/70">
            {formatElapsed(elapsed)}
          </span>
        )}
      </div>

      {/* Phase stepper */}
      <div className="flex items-center gap-0.5" data-testid="post-run-phase-stepper">
        {FINALIZATION_PHASE_ORDER.map((phase, i) => {
          const isPast = currentPhaseIdx > i;
          const isCurrent = currentPhaseIdx === i;
          return (
            <div key={phase} className="flex items-center gap-0.5">
              {i > 0 && (
                <div className={cn("h-px w-2.5", isPast ? "bg-indigo-400" : "bg-slate-600")} />
              )}
              <div className="flex flex-col items-center gap-0.5">
                <div
                  className={cn(
                    "h-1.5 w-1.5 rounded-full transition-colors",
                    isPast && "bg-indigo-400",
                    isCurrent && "bg-indigo-400 animate-pulse",
                    !isPast && !isCurrent && "bg-slate-600",
                  )}
                />
                <span
                  className={cn(
                    "text-[9px] leading-none",
                    isPast ? "text-indigo-400/70" : isCurrent ? "text-indigo-300" : "text-slate-600",
                  )}
                >
                  {PHASE_SHORT_LABELS[phase]}
                </span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export interface PostRunStatusBadgeProps {
  execution: ExecutionRecord;
}

export function PostRunStatusBadge({ execution }: PostRunStatusBadgeProps) {
  const [expanded, setExpanded] = useState(false);
  const finalization = execution.finalization;

  if (!finalization) {
    return null;
  }

  if (execution.status === "validating" || finalization.status === "running" || finalization.status === "pending") {
    return <FinalizationProgress finalization={finalization} />;
  }

  const evidenceSkipWarnings = finalization.warnings.filter(isEvidenceSkipWarning);
  const otherWarnings = finalization.warnings.filter((w) => !isEvidenceSkipWarning(w));
  const hasDetails = Boolean(finalization.aggregateSummary || otherWarnings.length);

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

      {/* Evidence-skip warnings are always visible — they explain a missing expected feature */}
      {evidenceSkipWarnings.map((warning) => (
        <div
          key={`${warning.code}-${warning.createdAt}`}
          className="flex items-start gap-2 rounded-md border border-slate-600/50 bg-slate-800/30 px-2.5 py-1.5"
          data-testid="evidence-skip-warning"
        >
          <Info className="mt-0.5 h-3 w-3 shrink-0 text-slate-400" />
          <p className="text-[11px] leading-relaxed text-slate-400">{warning.message}</p>
        </div>
      ))}

      {expanded && (
        <div className="space-y-2 rounded-md bg-slate-800/50 px-2.5 py-2">
          {finalization.aggregateSummary && (
            <p className="text-[11px] leading-relaxed text-slate-300">{finalization.aggregateSummary}</p>
          )}
          {otherWarnings.map((warning) => (
            <p key={`${warning.code}-${warning.createdAt}-${warning.scenarioName ?? ""}`} className="text-[11px] text-amber-300">
              {warning.scenarioName ? `${warning.scenarioName}: ` : ""}{warning.message}
            </p>
          ))}
        </div>
      )}
    </div>
  );
}
