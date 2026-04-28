/**
 * ScenarioResultCards — Per-scenario finalization results.
 *
 * Displays a card for each affected scenario showing restart status,
 * health check status, and GCT review dimensions. Only renders when
 * finalization data is available and not actively running.
 */

import { Check, AlertTriangle, X, Minus, ChevronDown, ChevronRight } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { renderMarkdown } from "../../lib/render-markdown";
import { cn } from "../../lib";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord, ScenarioFinalization, FinalizationStatus, ReviewDimension } from "../../types";
import { scenarioDetailPath } from "../../app/routes/route-paths";

export interface ScenarioResultCardsProps {
  execution: ExecutionRecord;
}

const STATUS_ICONS: Record<string, typeof Check> = {
  completed: Check,
  failed: X,
  skipped: Minus,
};

const STATUS_COLORS: Record<string, string> = {
  completed: "text-emerald-400",
  failed: "text-red-400",
  skipped: "text-slate-500",
  pending: "text-slate-500",
  running: "text-indigo-400",
};

const DIMENSION_COLORS: Record<string, string> = {
  green: "bg-emerald-500",
  yellow: "bg-amber-500",
  red: "bg-red-500",
  skipped: "bg-slate-600",
};

const CLASSIFICATION_STYLES: Record<string, string> = {
  ready: "bg-emerald-500/15 text-emerald-400",
  ready_with_notes: "bg-amber-500/15 text-amber-400",
  needs_work: "bg-red-500/15 text-red-400",
  not_assessable: "bg-slate-500/15 text-slate-400",
};

const CLASSIFICATION_LABELS: Record<string, string> = {
  ready: "Ready",
  ready_with_notes: "Notes",
  needs_work: "Needs work",
  not_assessable: "Inconclusive",
};

function StatusIndicator({ status, label }: { status: FinalizationStatus; label: string }) {
  const Icon = STATUS_ICONS[status];
  const color = STATUS_COLORS[status] ?? STATUS_COLORS.pending;
  return (
    <span className={cn("flex items-center gap-1 text-[11px]", color)}>
      {Icon ? <Icon className="h-3 w-3" /> : <span className="h-3 w-3" />}
      {label}
    </span>
  );
}

function DimensionBar({ dimension }: { dimension: ReviewDimension }) {
  const color = DIMENSION_COLORS[dimension.status] ?? DIMENSION_COLORS.skipped;
  return (
    <div className="flex items-center gap-1.5" title={dimension.details ?? dimension.name}>
      <div className={cn("h-1.5 w-1.5 shrink-0 rounded-full", color)} />
      <span className="text-[10px] text-slate-400">{dimension.name.replace(/_/g, " ")}</span>
    </div>
  );
}

function ScenarioCard({
  scenario,
}: {
  scenario: ScenarioFinalization;
}) {
  const navigate = useNavigate();
  const review = scenario.review;
  const classification = review.result?.classification;
  const dimensions = review.result?.dimensions ?? [];
  const summary = review.result?.summary;
  const hasExpandableDetails = Boolean(summary) || Boolean(scenario.health.details && scenario.health.status !== "completed") || Boolean(review.skipReason);
  const [expanded, setExpanded] = useState(false);

  return (
    <div
      className="rounded-lg border border-white/5 bg-slate-900/50"
      data-testid={selectors.review.scenarioResultCard}
    >
      {/* Header: scenario name + status indicators */}
      <div className="flex items-center gap-2 px-3 py-2">
        {hasExpandableDetails && (
          <button
            type="button"
            onClick={() => setExpanded((p) => !p)}
            className="shrink-0 text-slate-500 hover:text-slate-300 transition-colors"
          >
            {expanded ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
          </button>
        )}

        <button
          type="button"
          onClick={() => navigate(scenarioDetailPath(scenario.scenarioName))}
          className="text-xs font-medium text-slate-200 hover:text-violet-400 transition-colors truncate"
        >
          {scenario.scenarioName}
        </button>

        <div className="ml-auto flex items-center gap-3">
          <StatusIndicator status={scenario.restart.status} label="restart" />
          <StatusIndicator status={scenario.health.status} label="health" />
          {review.status === "skipped" ? (
            <StatusIndicator status="skipped" label="review" />
          ) : classification ? (
            <span className={cn(
              "rounded-full px-2 py-0.5 text-[10px] font-medium",
              CLASSIFICATION_STYLES[classification] ?? CLASSIFICATION_STYLES.not_assessable,
            )}>
              {CLASSIFICATION_LABELS[classification] ?? classification}
            </span>
          ) : (
            <StatusIndicator status={review.status} label="review" />
          )}
        </div>
      </div>

      {/* GCT dimensions — always visible when available */}
      {dimensions.length > 0 && (
        <div className="border-t border-white/5 px-3 py-1.5">
          <div className="flex flex-wrap gap-x-3 gap-y-1">
            {dimensions.map((d) => (
              <DimensionBar key={d.name} dimension={d} />
            ))}
          </div>
        </div>
      )}

      {/* Expandable details: summary, health errors, skip reasons */}
      {expanded && hasExpandableDetails && (
        <div className="border-t border-white/5 px-3 py-2 space-y-2">
          {scenario.health.details && scenario.health.status !== "completed" && (
            <div className="flex items-start gap-1.5 text-[11px] text-red-300">
              <AlertTriangle className="mt-0.5 h-3 w-3 shrink-0" />
              <span className="prose-sm-slate" dangerouslySetInnerHTML={{ __html: renderMarkdown(scenario.health.details) }} />
            </div>
          )}
          {review.skipReason && (
            <div className="prose-sm-slate text-[11px] text-amber-300" dangerouslySetInnerHTML={{ __html: renderMarkdown(review.skipReason) }} />
          )}
          {summary && (
            <div className="prose-sm-slate text-[11px] leading-relaxed text-slate-400" dangerouslySetInnerHTML={{ __html: renderMarkdown(summary) }} />
          )}
        </div>
      )}
    </div>
  );
}

export function ScenarioResultCards({ execution }: ScenarioResultCardsProps) {
  const finalization = execution.finalization;
  if (!finalization || finalization.scenarios.length === 0) return null;

  // Don't show cards while finalization is actively running — the stepper handles that
  if (finalization.status === "running" || finalization.status === "pending") return null;

  return (
    <div className="space-y-1.5" data-testid={selectors.review.scenarioResultCards}>
      <p className="text-[11px] font-medium uppercase tracking-wider text-slate-500">
        Scenario Results
      </p>
      {finalization.scenarios.map((scenario) => (
        <ScenarioCard
          key={scenario.scenarioName}
          scenario={scenario}
        />
      ))}
    </div>
  );
}
