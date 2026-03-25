/**
 * Detailed readiness panel for backlog detail views.
 *
 * Shows 5 labeled readiness dimensions with score badges, round count,
 * next nudge message, and a "ready for execution" banner.
 */
import { Play } from "lucide-react";
import { cn } from "../../lib";
import { Button } from "../ui/button";
import { READINESS_DIMENSIONS, DIMENSION_LABELS, SCORE_COLORS } from "../../lib/maturity";
import type { ReadinessIndicatorData } from "../../lib/maturity";

interface ReadinessDetailsPanelProps {
  data: ReadinessIndicatorData;
  /** When provided and data.ready is true, renders a "Run" CTA button. */
  onRun?: () => void;
}

const SCORE_BG_CLASSES: Record<string, string> = {
  slate: "bg-slate-700 text-slate-300",
  rose: "bg-rose-500/20 text-rose-400",
  amber: "bg-amber-500/20 text-amber-400",
  emerald: "bg-emerald-500/20 text-emerald-400",
};

export function ReadinessDetailsPanel({ data, onRun }: ReadinessDetailsPanelProps) {
  if (data.roundsCompleted === 0) return null;

  return (
    <div className="space-y-3 rounded-lg border border-slate-700 bg-slate-800/50 p-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-slate-200">Workshop Readiness</h3>
        <span className="rounded bg-slate-700 px-2 py-0.5 text-xs text-slate-300">
          Round {data.roundsCompleted}
        </span>
      </div>

      <div className="space-y-2">
        {READINESS_DIMENSIONS.map((dim) => {
          const score = data.effectiveScores[dim] ?? 0;
          const color = SCORE_COLORS[score] ?? "slate";
          return (
            <div key={dim} className="flex items-center justify-between">
              <span className="text-xs text-slate-400">{DIMENSION_LABELS[dim]}</span>
              <span
                className={cn(
                  "rounded px-2 py-0.5 text-xs font-medium",
                  SCORE_BG_CLASSES[color],
                )}
              >
                {score}/3
              </span>
            </div>
          );
        })}
      </div>

      {data.pendingItems > 0 && (
        <div className="text-xs text-amber-400">
          {data.pendingItems} pending item{data.pendingItems === 1 ? "" : "s"} to respond to
        </div>
      )}

      {data.ready && (
        <div className="space-y-2">
          <div className="rounded-md border border-emerald-500/20 bg-emerald-500/5 px-3 py-2 text-sm text-emerald-400">
            Ready for execution
          </div>
          {onRun && (
            <Button
              variant="default"
              size="sm"
              className="w-full"
              onClick={onRun}
            >
              <Play className="mr-2 h-4 w-4" />
              Run
            </Button>
          )}
        </div>
      )}

      {data.nextNudge && !data.ready && (
        <div className="text-xs text-slate-400 italic">{data.nextNudge}</div>
      )}
    </div>
  );
}
